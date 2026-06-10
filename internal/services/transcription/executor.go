//go:build completion

package transcription

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"YyslsPlayer/internal/storage"
	"YyslsPlayer/internal/services/transcription/analyzer"
	"YyslsPlayer/internal/services/transcription/audio"
	"YyslsPlayer/internal/services/transcription/engine"
	"YyslsPlayer/internal/services/transcription/shared"
)

const (
	// maxConcurrentTasks 并发转录任务上限。
	maxConcurrentTasks = 1
	// taskPollInterval 任务队列轮询间隔。
	taskPollInterval = 2 * time.Second
	// maxAudioDurationMs 音频最大时长（默认 10 分钟）。
	maxAudioDurationMs = 10 * 60 * 1000
	// maxAudioFileSize 音频最大体积（默认 200 MB）。
	maxAudioFileSize = 200 * 1024 * 1024
)

// Executor 是转录任务的后台执行器。
//
// 负责从队列中取任务、按阶段执行、推送进度事件、处理取消和错误。
// 同一时间最多运行 maxConcurrentTasks 个任务。
type Executor struct {
	holder  *storage.Holder
	log     *slog.Logger
	emitter EventEmitter
	dataDir string // 应用数据根目录，任务工作目录位于 <dataDir>/transcriptions/<id>/

	mu       sync.Mutex
	activeCtx    context.Context
	activeCancel context.CancelFunc
	running  bool

	// 可替换的 adapter，便于测试注入
	audioProber  AudioProber
	audioDecoder AudioDecoder
	engine       Engine
	postproc     Postprocessor
	midiWriter   MidiWriter
	midiImporter MidiImporter
}

// EventEmitter 是向前端推送事件的接口。
type EventEmitter interface {
	Emit(name string, payload any)
}

// AudioProber 音频格式探测接口。
type AudioProber interface {
	Probe(path string) (*audio.ProbeResult, error)
	Available() bool
}

// AudioDecoder 音频解码接口。
type AudioDecoder interface {
	Decode(ctx context.Context, path string, workDir string) (pcmPath string, sampleRate int, channels int, err error)
	Available() bool
}

// Engine 转录引擎接口。
type Engine interface {
	Name() string
	Version() string
	Available() bool
	Transcribe(ctx context.Context, pcmPath string, workDir string, config engine.Config) (*engine.Result, error)
}

// RawNote 引擎原始音符（alias 便于 executor 内部使用）。
type RawNote = engine.RawNote

// Postprocessor 后处理接口。
type Postprocessor interface {
	Process(raw []RawNote, config TranscriptionConfigDTO, durationMs int64) (*PostprocessResult, error)
}

// PostprocessResult 后处理结果。
type PostprocessResult struct {
	Notes              []RawNote              `json:"notes"`
	QualityReport      shared.QualityReport   `json:"qualityReport"`
	MelodySummary      shared.MelodySummary   `json:"melodySummary"`
	DroppedCount       int                    `json:"droppedCount"`
	LowConfidenceCount int                    `json:"lowConfidenceCount"`
	OutOfRangeCount    int                    `json:"outOfRangeCount"`
}

// MidiWriter MIDI 文件写出接口。
type MidiWriter interface {
	Write(notes []RawNote, bpm float64, targetPath string) error
}

// MidiImporter MIDI 导入现有链路接口。
type MidiImporter interface {
	Import(midiPath string, displayName string) (projectID uint, noteCount int, durationMs int64, fileHash string, err error)
}

// NewExecutor 创建转录任务执行器。
func NewExecutor(holder *storage.Holder, emitter EventEmitter, dataDir string, opts ...ExecutorOption) *Executor {
	e := &Executor{
		holder:   holder,
		log:      slog.Default().With("component", "transcription.executor"),
		emitter:  emitter,
		dataDir:  dataDir,
		running:  true,
	}
	for _, o := range opts {
		o(e)
	}
	return e
}

// ExecutorOption 执行器可选配置。
type ExecutorOption func(*Executor)

// WithAudioProber 注入自定义音频探测器。
func WithAudioProber(p AudioProber) ExecutorOption {
	return func(e *Executor) { e.audioProber = p }
}

// WithAudioDecoder 注入自定义音频解码器。
func WithAudioDecoder(d AudioDecoder) ExecutorOption {
	return func(e *Executor) { e.audioDecoder = d }
}

// WithEngine 注入自定义转录引擎。
func WithEngine(eng Engine) ExecutorOption {
	return func(e *Executor) { e.engine = eng }
}

// WithPostprocessor 注入自定义后处理器。
func WithPostprocessor(p Postprocessor) ExecutorOption {
	return func(e *Executor) { e.postproc = p }
}

// WithMidiWriter 注入自定义 MIDI 写出器。
func WithMidiWriter(w MidiWriter) ExecutorOption {
	return func(e *Executor) { e.midiWriter = w }
}

// WithMidiImporter 注入自定义 MIDI 导入器。
func WithMidiImporter(i MidiImporter) ExecutorOption {
	return func(e *Executor) { e.midiImporter = i }
}

// store 返回当前活跃存储。
func (e *Executor) store() *storage.Store {
	return e.holder.Current().Store
}

// Start 启动任务调度循环。应在应用启动后调用。
func (e *Executor) Start() {
	go e.loop()
}

// Shutdown 停止调度循环并取消活跃任务。应在应用退出时调用。
func (e *Executor) Shutdown() {
	e.mu.Lock()
	e.running = false
	if e.activeCancel != nil {
		e.activeCancel()
	}
	e.mu.Unlock()
}

// CancelActive 取消当前活跃任务（不停止调度循环）。
// 用于 CancelTask：取消 running 任务后，executor 继续从队列取下一个任务。
func (e *Executor) CancelActive() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.activeCancel != nil {
		e.activeCancel()
	}
}

// Recover 恢复上次遗留的 running/cancelling 任务。
func (e *Executor) Recover() {
	if err := e.store().RecoverTranscriptionTasks(); err != nil {
		e.log.Warn("recover transcription tasks failed", "error", err)
	}
}

// loop 是后台调度循环。
func (e *Executor) loop() {
	e.log.Info("transcription executor started")
	defer e.log.Info("transcription executor stopped")

	for {
		e.mu.Lock()
		active := e.running
		e.mu.Unlock()
		if !active {
			return
		}
		e.processNext()
		time.Sleep(taskPollInterval)
	}
}

// processNext 尝试取下一个 queued 任务并执行。
func (e *Executor) processNext() {
	// 检查是否已有活跃任务
	e.mu.Lock()
	if e.activeCtx != nil {
		e.mu.Unlock()
		return
	}
	e.mu.Unlock()

	// 取最早的 queued 任务
	tasks := e.store().ListTranscriptionTasks(1, 0)
	if len(tasks) == 0 || tasks[0].Status != string(StatusQueued) {
		return
	}
	task := tasks[0]

	// 执行任务
	ctx, cancel := context.WithCancel(context.Background())
	e.mu.Lock()
	e.activeCtx = ctx
	e.activeCancel = cancel
	e.mu.Unlock()

	defer func() {
		e.mu.Lock()
		e.activeCtx = nil
		e.activeCancel = nil
		e.mu.Unlock()
		cancel()
	}()

	e.runTask(ctx, &task)
}

// runTask 执行单个转录任务的完整流水线。
func (e *Executor) runTask(ctx context.Context, task *storage.TranscriptionTask) {
	taskID := task.ID
	log := e.log.With("taskId", taskID)
	pd := &shared.PipelineDebug{} // 诊断记录

	stageStart := func(stage string, input string) string {
		pd.Stages = append(pd.Stages, shared.StageDebug{
			Stage: stage, Status: "running",
			StartTime: time.Now().UnixMilli(), Input: input,
		})
		return stage
	}
	stageEnd := func(stage string, output, diagnostics string) {
		now := time.Now().UnixMilli()
		for i := range pd.Stages {
			if pd.Stages[i].Stage == stage && pd.Stages[i].Status == "running" {
				pd.Stages[i].Status = "ok"
				pd.Stages[i].EndTime = now
				pd.Stages[i].DurationMs = now - pd.Stages[i].StartTime
				pd.Stages[i].Output = output
				pd.Stages[i].Diagnostics = diagnostics
				return
			}
		}
	}
	stageFail := func(stage string, errMsg string) {
		now := time.Now().UnixMilli()
		for i := range pd.Stages {
			if pd.Stages[i].Stage == stage && pd.Stages[i].Status == "running" {
				pd.Stages[i].Status = "failed"
				pd.Stages[i].EndTime = now
				pd.Stages[i].DurationMs = now - pd.Stages[i].StartTime
				pd.Stages[i].Error = errMsg
				return
			}
		}
	}

	// 标记为 running
	now := time.Now().UnixMilli()
	task.Status = string(StatusRunning)
	task.Stage = string(StageProbe)
	task.StartedAt = &now
	task.UpdatedAt = now
	if err := e.store().UpdateTranscriptionTask(task); err != nil {
		log.Error("update task to running", "error", err)
		return
	}
	e.emitProgress(taskID, 0, "starting")

	// 建立任务工作目录
	workDir, err := ensureTaskWorkDir(e.dataDir, taskID)
	if err != nil {
		e.failTask(task, pd, "transcription.storage_failed", err.Error())
		return
	}

	// 加载配置
	var config TranscriptionConfigDTO
	if task.ConfigJSON != "" {
		if err := json.Unmarshal([]byte(task.ConfigJSON), &config); err != nil {
			e.failTask(task, pd, "transcription.invalid_config", "failed to parse task config")
			return
		}
	} else {
		cfg, _ := e.store().GetTranscriptionConfig()
		config = configFromStorage(cfg)
	}

	// Stage 1: Probe
	e.updateStage(task, StageProbe)
	e.emitProgress(taskID, 0.05, "probing audio format")
	probeSt := stageStart("probe", fmt.Sprintf("path=%s size=%d", task.SourcePath, func() int64 {
		if fi, err := os.Stat(task.SourcePath); err == nil { return fi.Size() }
		return -1
	}()))
	probe, err := e.audioProber.Probe(task.SourcePath)
	if err != nil {
		stageFail("probe", err.Error())
		e.failTask(task, pd, "audio.probe_failed", err.Error())
		return
	}
	stageEnd(probeSt,
		fmt.Sprintf("format=%s duration=%dms sr=%d ch=%d codec=%s bitrate=%d",
			probe.Format, probe.DurationMs, probe.SampleRate, probe.Channels, probe.Codec, probe.Bitrate),
		fmt.Sprintf("container=%s file_size=%d", probe.Container, probe.FileSizeBytes))
	task.DurationMs = probe.DurationMs
	task.SampleRate = probe.SampleRate
	task.Channels = probe.Channels
	task.Engine = e.engine.Name()
	task.EngineVersion = e.engine.Version()
	e.store().UpdateTranscriptionTask(task)

	// 校验时长和文件大小
	if probe.DurationMs > maxAudioDurationMs {
		e.failTask(task, pd, "audio.too_long", "audio duration exceeds limit")
		return
	}
	if probe.FileSizeBytes > maxAudioFileSize {
		e.failTask(task, pd, "audio.too_large", "audio file size exceeds limit")
		return
	}

	// Stage 2: Decode
	e.updateStage(task, StageDecode)
	e.emitProgress(taskID, 0.10, "decoding audio")
	decodeSt := stageStart("decode", fmt.Sprintf("workdir=%s source=%s sr=%d ch=%d dur=%dms",
		workDir, task.SourcePath, probe.SampleRate, probe.Channels, probe.DurationMs))
	pcmPath, pcmSr, pcmCh, err := e.audioDecoder.Decode(ctx, task.SourcePath, workDir)
	if err != nil {
		stageFail("decode", err.Error())
		if ctx.Err() != nil {
			e.cancelTask(task)
			return
		}
		e.failTask(task, pd, "audio.decode_failed", err.Error())
		return
	}
	stageEnd(decodeSt,
		fmt.Sprintf("pcm=%s sample_rate=%d channels=%d", pcmPath, pcmSr, pcmCh),
		fmt.Sprintf("target_sr=%d target_ch=%d", pcmSr, pcmCh))

	// Stage 2.5: Analyze (音频质量)
	e.updateStage(task, StageAnalyze)
	e.emitProgress(taskID, 0.18, "analyzing audio quality")
	analyzeSt := stageStart("analyze", fmt.Sprintf("pcm=%s", pcmPath))
	audioQuality := analyzeAudioQualityFromFile(pcmPath)
	e.saveAnalysis(taskID, "audioQuality", audioQuality)
	stageEnd(analyzeSt,
		fmt.Sprintf("score=%.0f clipping=%v silence=%.2f",
			audioQuality.Score, audioQuality.ClippingDetected, audioQuality.SilenceRatio),
		fmt.Sprintf("audio_quality_analysis_complete"))

	// Stage 3: Transcribe (带超时)
	e.updateStage(task, StageTranscribe)
	e.emitProgress(taskID, 0.25, "transcribing with model engine")
	select {
	case <-ctx.Done():
		stageFail("transcribe", "cancelled")
		e.cancelTask(task)
		return
	default:
	}
	// worker 超时：音频时长 * 3，最少 120s，加 60s buffer
	engineTimeout := max(task.DurationMs*3, 120000) + 60000
	if engineTimeout > 30*60*1000 {
		engineTimeout = 30 * 60 * 1000 // 上限 30 分钟
	}
	engineCtx, engineCancel := context.WithTimeout(ctx, time.Duration(engineTimeout)*time.Millisecond)
	defer engineCancel()

	transcribeSt := stageStart("transcribe",
		fmt.Sprintf("pcm=%s engine=%s min_confidence=%.2f max_polyphony=%d timeout=%ds",
			pcmPath, e.engine.Name(), config.MinConfidence, config.MaxPolyphony, engineTimeout/1000))
	engineResult, err := e.engine.Transcribe(engineCtx, pcmPath, workDir, engine.Config{
		MinConfidence: config.MinConfidence,
		MinDurationMs: config.MinDurationMs,
		MaxPolyphony:  config.MaxPolyphony,
	})
	if err != nil {
		stageFail("transcribe", err.Error())
		if ctx.Err() != nil {
			e.cancelTask(task)
			return
		}
		if engineCtx.Err() != nil {
			e.failTask(task, pd, "engine.timeout", "transcription engine timed out")
			return
		}
		e.failTask(task, pd, "engine.failed", err.Error())
		return
	}
	if len(engineResult.Notes) == 0 {
		stageEnd(transcribeSt,
			"raw_notes=0", "no notes detected by engine")
		e.failTask(task, pd, "postprocess.empty_result", "no notes detected")
		return
	}
	e.emitProgress(taskID, 0.60, "transcription complete")
	stageEnd(transcribeSt,
		fmt.Sprintf("raw_notes=%d bpm_est=%.1f duration_ms=%d",
			len(engineResult.Notes), engineResult.BPMEstimate, engineResult.DurationMs),
		engineResult.Diagnostics)

	// Stage 3.5: BPM + Key/Scale 分析
	e.emitProgress(taskID, 0.65, "analyzing BPM and key")
	bpmEstim := analyzer.EstimateBPM(engineResult.Notes)
	pitchHist := analyzer.PitchClassHistogram(engineResult.Notes)
	keyCandidates := analyzer.EstimateKey(pitchHist)
	e.saveAnalysis(taskID, "bpm", bpmEstim)
	e.saveAnalysis(taskID, "key", keyCandidates)
	// 追加 BPM 诊断到 transcribe 阶段的 output（替换旧值）
	for i := range pd.Stages {
		if pd.Stages[i].Stage == "transcribe" {
			pd.Stages[i].Output += fmt.Sprintf(" bpm_conf=%.2f key=%s",
				bpmEstim.Confidence,
				func() string {
					if len(keyCandidates) > 0 {
						return fmt.Sprintf("%s_%s(%.2f)", keyCandidates[0].Tonic, keyCandidates[0].Mode, keyCandidates[0].Confidence)
					}
					return "unknown"
				}())
			break
		}
	}

	// Stage 4: Postprocess
	e.updateStage(task, StagePostprocess)
	e.emitProgress(taskID, 0.75, "post-processing notes")
	select {
	case <-ctx.Done():
		stageFail("postprocess", "cancelled")
		e.cancelTask(task)
		return
	default:
	}
	postprocSt := stageStart("postprocess",
		fmt.Sprintf("raw_notes=%d mode=%s min_conf=%.2f quantize=%s max_poly=%d target_base=%d lanes=%d policy=%s",
			len(engineResult.Notes), config.Mode, config.MinConfidence, config.Quantize,
			config.MaxPolyphony, config.TargetBaseNote, config.TargetLaneCount, config.OutOfRangePolicy))
	result, err := e.postproc.Process(engineResult.Notes, config, probe.DurationMs)
	if err != nil {
		stageFail("postprocess", err.Error())
		e.failTask(task, pd, "postprocess.failed", err.Error())
		return
	}
	stageEnd(postprocSt,
		fmt.Sprintf("filtered_notes=%d dropped=%d low_conf=%d out_range=%d range=[%d-%d]",
			len(result.Notes), result.DroppedCount, result.LowConfidenceCount,
			result.OutOfRangeCount,
			result.MelodySummary.MinNote, result.MelodySummary.MaxNote),
		fmt.Sprintf("avg_velocity=%.1f polyphony_rate=%.2f playability=%.0f",
			result.MelodySummary.AverageVelocity, result.MelodySummary.PolyphonyRate,
			result.MelodySummary.PlayabilityScore))

	// 用分析阶段真实值覆盖质量报告中的默认值
	result.QualityReport.EstimatedBPM = bpmEstim.BPM
	result.QualityReport.BPMConfidence = bpmEstim.Confidence
	result.QualityReport.AudioQualityScore = audioQuality.Score
	if len(keyCandidates) > 0 {
		result.QualityReport.KeyEstimate = shared.KeyEstimate{
			Tonic:      keyCandidates[0].Tonic,
			Mode:       keyCandidates[0].Mode,
			Confidence: keyCandidates[0].Confidence,
			Method:     "pitch_class",
			Candidates: convertKeyCandidates(keyCandidates),
		}
	}
	result.MelodySummary.EstimatedBPM = bpmEstim.BPM

	// 批量写入处理后的 notes
	notes := make([]storage.TranscriptionNote, len(result.Notes))
	for i, n := range result.Notes {
		notes[i] = storage.TranscriptionNote{
			TaskID:     taskID,
			Note:       n.Note,
			Velocity:   clampVelocity(int(n.Velocity)),
			StartMs:    n.StartMs,
			DurationMs: n.DurationMs,
			Confidence: n.Confidence,
			Source:     "postprocess",
		}
	}
	if err := e.store().BatchCreateNotes(notes); err != nil {
		e.failTask(task, pd, "transcription.storage_failed", err.Error())
		return
	}

	// 写入分析结果
	e.saveAnalysis(taskID, "playability", result.QualityReport)
	e.saveAnalysis(taskID, "melody", result.MelodySummary)

	// Stage 5: MIDI write
	e.updateStage(task, StageMidi)
	e.emitProgress(taskID, 0.90, "writing MIDI file")
	select {
	case <-ctx.Done():
		stageFail("midi", "cancelled")
		e.cancelTask(task)
		return
	default:
	}
	midiPath := filepath.Join(workDir, "result.mid")
	midiSt := stageStart("midi",
		fmt.Sprintf("notes=%d bpm=%.1f dest=%s", len(result.Notes), result.MelodySummary.EstimatedBPM, midiPath))
	if err := e.midiWriter.Write(result.Notes, result.MelodySummary.EstimatedBPM, midiPath); err != nil {
		stageFail("midi", err.Error())
		e.failTask(task, pd, "midi.write_failed", err.Error())
		return
	}
	stageEnd(midiSt,
		fmt.Sprintf("midi_path=%s", midiPath),
		"MIDI file written successfully")

	// 完成
	reportJSON, _ := json.Marshal(result.QualityReport)
	summaryJSON, _ := json.Marshal(result.MelodySummary)
	now2 := time.Now().UnixMilli()
	task.Status = string(StatusCompleted)
	task.Stage = string(StageCompleted)
	task.Progress = 1.0
	task.ResultMidiPath = midiPath
	task.SummaryJSON = string(summaryJSON)
	task.ReportJSON = string(reportJSON)
	task.FinishedAt = &now2
	task.UpdatedAt = now2
	e.store().UpdateTranscriptionTask(task)

	// 保存管道诊断记录
	e.saveAnalysis(taskID, "pipelineDebug", pd)

	e.emitProgress(taskID, 1.0, "completed")
	e.emitCompleted(taskID, result)
	log.Info("task completed", "notes", len(result.Notes))
}

// failTask 将任务标记为失败，并保存管道诊断记录。
func (e *Executor) failTask(task *storage.TranscriptionTask, pd *shared.PipelineDebug, code, msg string) {
	// 先保存管道诊断记录（含各阶段输入/输出/错误详情）
	if pd != nil && len(pd.Stages) > 0 {
		e.saveAnalysis(task.ID, "pipelineDebug", pd)
	}

	now := time.Now().UnixMilli()
	task.Status = string(StatusFailed)
	task.ErrorCode = &code
	task.ErrorMessage = &msg
	task.FinishedAt = &now
	task.UpdatedAt = now
	if err := e.store().UpdateTranscriptionTask(task); err != nil {
		e.log.Error("fail task update", "taskId", task.ID, "error", err)
	}
	e.emitFailed(task.ID, code, msg)
	// 清理解码后的临时 PCM（result.mid 可能未生成，失败任务保留 worker 日志诊断用）
	_ = cleanupTaskWorkDir(e.dataDir, task.ID)
}

// cancelTask 将任务标记为已取消。
func (e *Executor) cancelTask(task *storage.TranscriptionTask) {
	now := time.Now().UnixMilli()
	task.Status = string(StatusCancelled)
	task.FinishedAt = &now
	task.UpdatedAt = now
	_ = e.store().UpdateTranscriptionTask(task)
	e.emitCancelled(task.ID)
	// 取消任务清理所有临时文件
	_ = cleanupTaskWorkDir(e.dataDir, task.ID)
}

// updateStage 更新任务阶段并持久化。
func (e *Executor) updateStage(task *storage.TranscriptionTask, stage TaskStage) {
	task.Stage = string(stage)
	task.UpdatedAt = time.Now().UnixMilli()
	_ = e.store().UpdateTranscriptionTask(task)
}

// saveAnalysis 写入一条分析记录。
func (e *Executor) saveAnalysis(taskID uint, kind string, payload any) {
	payloadJSON, _ := json.Marshal(payload)
	_ = e.store().SaveAnalysis(&storage.TranscriptionAnalysis{
		TaskID:      taskID,
		Kind:        kind,
		PayloadJSON: string(payloadJSON),
	})
}

// Helper functions
func (e *Executor) emitProgress(taskID uint, progress float64, message string) {
	e.emitter.Emit(EventTaskProgress, TranscriptionProgress{
		TaskID:   formatTaskID(taskID),
		Status:   string(StatusRunning),
		Progress: progress,
		Message:  message,
	})
}

func (e *Executor) emitCompleted(taskID uint, result *PostprocessResult) {
	totalNotes := len(result.Notes)
	inRange := totalNotes - result.OutOfRangeCount
	coverage := 0.0
	if totalNotes > 0 {
		coverage = float64(inRange) / float64(totalNotes) * 100
	}
	reportJSON, _ := json.Marshal(result.QualityReport)
	e.emitter.Emit(EventTaskCompleted, TranscriptionResult{
		TaskID:               formatTaskID(taskID),
		TotalNotes:           totalNotes,
		InRangeNotes:         inRange,
		OutRangeNotes:        result.OutOfRangeCount,
		EstimatedBPM:         result.MelodySummary.EstimatedBPM,
		SuggestedOctaveShift: result.QualityReport.SuggestedOctaveShift,
		CoveragePercent:      coverage,
		QualityReport:        string(reportJSON),
	})
}

func (e *Executor) emitFailed(taskID uint, code, msg string) {
	e.emitter.Emit(EventTaskFailed, TranscriptionError{
		TaskID:       formatTaskID(taskID),
		ErrorCode:    code,
		ErrorMessage: msg,
	})
}

func (e *Executor) emitCancelled(taskID uint) {
	e.emitter.Emit(EventTaskCancelled, TranscriptionProgress{
		TaskID:   formatTaskID(taskID),
		Status:   string(StatusCancelled),
		Progress: 0,
	})
}

func formatTaskID(id uint) string {
	return fmt.Sprintf("%d", id)
}

func configFromStorage(c storage.TranscriptionConfig) TranscriptionConfigDTO {
	return TranscriptionConfigDTO{
		Mode:                 c.Mode,
		MinConfidence:        c.MinConfidence,
		MinDurationMs:        c.MinDurationMs,
		MergeGapMs:           c.MergeGapMs,
		Quantize:             c.Quantize,
		MaxPolyphony:         c.MaxPolyphony,
		TargetBaseNote:       c.TargetBaseNote,
		TargetLaneCount:      c.TargetLaneCount,
		OutOfRangePolicy:     c.OutOfRangePolicy,
		PreferMelodyRegister: c.PreferMelodyRegister,
	}
}

func clampVelocity(v int) int {
	if v < 1 {
		return 1
	}
	if v > 127 {
		return 127
	}
	return v
}

func max(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

// analyzeAudioQualityFromFile 对解码后的 WAV 文件做质量分析。
// 简化实现：读取 WAV 的 PCM 数据（跳过 44 字节头），分析削波和静音。
func analyzeAudioQualityFromFile(wavPath string) analyzer.AudioQuality {
	data, err := os.ReadFile(wavPath)
	if err != nil || len(data) <= 44 {
		return analyzer.AudioQuality{Score: 80} // fallback
	}
	pcmData := data[44:]
	sampleCount := len(pcmData) / 4 // float32 = 4 bytes
	if sampleCount > 100000 {
		sampleCount = 100000 // 取前 100k 样本分析
	}
	samples := make([]float32, sampleCount)
	for i := 0; i < sampleCount; i++ {
		offset := i * 4
		if offset+3 >= len(pcmData) {
			break
		}
		// little-endian float32
		bits := uint32(pcmData[offset]) | uint32(pcmData[offset+1])<<8 | uint32(pcmData[offset+2])<<16 | uint32(pcmData[offset+3])<<24
		samples[i] = float32frombits(bits)
	}
	return analyzer.AnalyzeAudioQuality(samples, 22050)
}

func float32frombits(b uint32) float32 {
	// 简单的手动实现替代 math.Float32frombits
	n := int32(b)
	if n == 0 {
		return 0
	}
	sign := float32(1)
	if n < 0 {
		sign = -1
		n = -n
	}
	// 近似：用位运算提取指数和尾数
	exp := int((uint32(n) >> 23) & 0xFF)
	mantissa := (uint32(n) & 0x7FFFFF) | 0x800000
	result := float32(mantissa) / float32(1<<23)
	for e := exp - 127; e > 0; e-- {
		result *= 2
	}
	for e := exp - 127; e < 0; e++ {
		result /= 2
	}
	return sign * result
}

func convertKeyCandidates(candidates []analyzer.KeyCandidate) []shared.KeyCandidate {
	result := make([]shared.KeyCandidate, len(candidates))
	for i, c := range candidates {
		result[i] = shared.KeyCandidate{
			Tonic:      c.Tonic,
			Mode:       c.Mode,
			Confidence: c.Confidence,
		}
	}
	return result
}
//go:build completion

// Package transcription 提供音频转 MIDI（扒谱）能力。
// 仅在 completion 版本中编译；lite 版本不包含此包。
//
// 职责：
//   - 接收音频文件并解码为 PCM
//   - 通过自动音乐转录模型识别音符候选
//   - 后处理生成面向 36 键模式的可演奏 MIDI
//   - 管理异步转录任务（创建、进度、取消、结果落库）
package transcription

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"YyslsPlayer/internal/services/midi"
	"YyslsPlayer/internal/services/transcription/audio"
	"YyslsPlayer/internal/services/transcription/engine"
	"YyslsPlayer/internal/services/transcription/midiout"
	"YyslsPlayer/internal/services/transcription/postprocess"
	"YyslsPlayer/internal/services/transcription/shared"
	"YyslsPlayer/internal/storage"
	"YyslsPlayer/internal/utils/filex"
	"YyslsPlayer/internal/utils/httpx"
	"YyslsPlayer/internal/utils/logx"
	"YyslsPlayer/internal/utils/procx"

	"github.com/wailsapp/wails/v3/pkg/application"
	wailsevents "github.com/wailsapp/wails/v3/pkg/events"
)

// Service 是音频转录业务的外观。
// 所有转录 API 都通过本 Service 暴露给 Wails bindings 和前端。
type Service struct {
	holder        *storage.Holder
	executor      *Executor
	app           *application.App // 应用实例，用于创建子窗口
	downloadProxy string           // FFmpeg 下载代理地址（如 http://127.0.0.1:7890）
}

// New 创建 transcription Service 并返回。
// emitter 用于向前端推送进度事件，可传 nil 后续通过 AttachEmitter 注入。
func New(holder *storage.Holder, emitter EventEmitter) *Service {
	dataDir := filepath.Dir(holder.Current().Path)
	svc := &Service{
		holder:   holder,
		executor: NewExecutor(holder, emitter, dataDir),
	}
	return svc
}

// dataDir 从 holder 提取应用数据根目录。
func (s *Service) dataDir() string {
	return filepath.Dir(s.holder.Current().Path)
}

// AttachEmitter 注入事件发射器（用于推迟到 app 创建后绑定 Wails 事件总线）。
//wails:ignore
func (s *Service) AttachEmitter(fn func(name string, payload any)) {
	s.executor.emitter = eventFuncEmitter(fn)
}

// eventFuncEmitter 将 func(name, payload) 适配为 EventEmitter。
type eventFuncEmitter func(name string, payload any)

func (f eventFuncEmitter) Emit(name string, payload any) { f(name, payload) }

// postprocAdapter 将 postprocess.Processor 适配为 Postprocessor 接口。
type postprocAdapter struct{}

func (a *postprocAdapter) Process(raw []RawNote, cfg TranscriptionConfigDTO, durationMs int64) (*PostprocessResult, error) {
	sharedCfg := shared.Config{
		Mode:                 cfg.Mode,
		MinConfidence:        cfg.MinConfidence,
		MinDurationMs:        cfg.MinDurationMs,
		MergeGapMs:           cfg.MergeGapMs,
		Quantize:             cfg.Quantize,
		MaxPolyphony:         cfg.MaxPolyphony,
		TargetBaseNote:       cfg.TargetBaseNote,
		TargetLaneCount:      cfg.TargetLaneCount,
		OutOfRangePolicy:     cfg.OutOfRangePolicy,
		PreferMelodyRegister: cfg.PreferMelodyRegister,
	}

	p := postprocess.New(sharedCfg)
	result, err := p.Process(raw, durationMs)
	if err != nil {
		return nil, err
	}

	return &PostprocessResult{
		Notes:              result.Notes,
		QualityReport:      result.QualityReport,
		MelodySummary:      result.MelodySummary,
		DroppedCount:       result.DroppedCount,
		LowConfidenceCount: result.LowConfidenceCount,
		OutOfRangeCount:    result.OutOfRangeCount,
	}, nil
}


// SetExecutorOptions 注入自定义 adapter（测试用）。
//wails:ignore
func (s *Service) SetExecutorOptions(opts ...ExecutorOption) {
	for _, o := range opts {
		o(s.executor)
	}
}

// Start 初始化组件并启动后台任务执行器。
//wails:ignore
func (s *Service) Start() {
	// 引擎选择：优先 Basic Pitch（真实模型），不可用时降级为 mock
	var selectedEngine engine.Engine
	bpEngine := engine.NewBasicPitchEngine()
	if bpEngine.Available() {
		selectedEngine = bpEngine
		logx.For("transcription").Info("using basic pitch engine")
	} else {
		selectedEngine = engine.NewMockEngine()
		logx.For("transcription").Info("basic pitch not available, using mock engine")
	}

	s.executor.audioProber = audio.NewFFmpegProber()
	s.executor.audioDecoder = audio.NewFFmpegDecoder()
	s.executor.engine = selectedEngine
	s.executor.postproc = &postprocAdapter{}
	s.executor.midiWriter = midiout.NewSMFWriter()
	s.executor.midiImporter = midiout.NewMidiImporter(s.holder)

	s.executor.Recover()
	s.executor.Start()
}

// Shutdown 停止后台任务执行器。
//wails:ignore
func (s *Service) Shutdown() {
	s.executor.Shutdown()
}

// SetApp 注入应用实例（用于创建子窗口）。
//wails:ignore
func (s *Service) SetApp(app *application.App) {
	s.app = app
}

// store 返回当前活跃存储。
func (s *Service) store() *storage.Store {
	return s.holder.Current().Store
}

// ===== Wails 导出方法 =====

// GetCapability 返回当前构建的能力检测结果。
func (s *Service) GetCapability(_ context.Context) (TranscriptionCapabilityDTO, error) {
	missing := []string{}
	// 检查 FFmpeg 可用性
	if prober, ok := s.executor.audioProber.(interface{ Available() bool }); ok && !prober.Available() {
		missing = append(missing, "ffmpeg")
	}
	// 检查引擎可用性
	if eng, ok := s.executor.engine.(interface{ Available() bool }); ok && !eng.Available() {
		missing = append(missing, "model")
	}

	return TranscriptionCapabilityDTO{
		TranscriptionEnabled: true,
		BuildFlavor:          "completion",
		MissingComponents:    missing,
	}, nil
}

// SetDownloadProxy 设置 FFmpeg 下载时使用的代理地址。
// proxyAddr 格式如 "http://127.0.0.1:7890"，传空字符串清除代理。
func (s *Service) SetDownloadProxy(_ context.Context, proxyAddr string) {
	s.downloadProxy = strings.TrimSpace(proxyAddr)
	logx.For("transcription").Info("download proxy updated", "proxy", s.downloadProxy)
}

// FFMPEG_RELEASE_URL ffmpeg Windows 构建下载地址（BtbN/FFmpeg-Builds，包含 ffmpeg.exe + ffprobe.exe）。
const FFMPEG_RELEASE_URL = "https://github.com/BtbN/FFmpeg-Builds/releases/download/latest/ffmpeg-master-latest-win64-gpl.zip"

// InstallFfmpeg 弹出进度窗口自动下载安装 FFmpeg。
// 前端不阻塞等待；下载完成后通过 EventFfmpegInstalled 通知。
func (s *Service) InstallFfmpeg(_ context.Context) (string, error) {
	targetDir := filepath.Join("assets", "transcription", "ffmpeg")

	// 检查是否已安装
	if fi, _ := os.Stat(filepath.Join(targetDir, "ffmpeg.exe")); fi != nil {
		return "FFmpeg 已安装", nil
	}

	if s.app == nil {
		return "", fmt.Errorf("application not available")
	}

	// 创建进度子窗口
	w := s.app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:          "ffmpeg-installer",
		Title:         "安装 FFmpeg",
		Width:         420,
		Height:        340,
		MinWidth:      420,
		MinHeight:     340,
		MaxWidth:      420,
		MaxHeight:     340,
		DisableResize: true,
		Frameless:     false,
		HTML:          installFfmpegHTML,
	})

	// 等待 webview runtime 就绪后再启动下载，避免 ExecJS 在 JS 上下文未就绪时执行
	// 同时设置 5s 安全超时：若 runtime 事件异常未触发，超时后仍启动下载
	var once sync.Once
	startDownload := func() { go s.downloadFfmpegWithProgress(w, targetDir) }
	readyTimer := time.AfterFunc(5*time.Second, func() {
		logx.For("transcription").Warn("ffmpeg installer window runtime ready timeout, starting download anyway")
		once.Do(startDownload)
	})

	w.OnWindowEvent(wailsevents.Common.WindowRuntimeReady, func(e *application.WindowEvent) {
		readyTimer.Stop()
		once.Do(startDownload)
	})

	w.Center()
	w.Show()

	return "正在下载安装 FFmpeg，请查看进度窗口", nil
}

// EventFfmpegInstalled FFmpeg 安装完成事件名。
const EventFfmpegInstalled = "transcription.ffmpeg.installed"

// EventFfmpegInstallFailed FFmpeg 安装失败事件名。
const EventFfmpegInstallFailed = "transcription.ffmpeg.failed"

func (s *Service) downloadFfmpegWithProgress(w application.Window, targetDir string) {
	execJS := func(js string) {
		if w != nil {
			w.ExecJS(js)
		}
	}
	updateStatus := func(text string) {
		execJS(fmt.Sprintf(`setStatus(%q)`, text))
	}
	updateProgress := func(pct int) {
		execJS(fmt.Sprintf(`setProgress(%d)`, pct))
	}
	updateDetail := func(speedBps, downloaded, total int64, etaSec int) {
		execJS(fmt.Sprintf(`setDetail(%d,%d,%d,%d)`, speedBps, downloaded, total, etaSec))
	}

	fail := func(err error) {
		logx.For("transcription").Error("ffmpeg install failed", "error", err)
		updateStatus("安装失败: " + err.Error())
		time.Sleep(3 * time.Second)
		if w != nil {
			w.Close()
		}
		if s.executor.emitter != nil {
			s.executor.emitter.Emit(EventFfmpegInstallFailed, map[string]string{
				"error": err.Error(),
			})
		}
	}

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		fail(fmt.Errorf("创建目录失败: %w", err))
		return
	}

	updateStatus("正在下载 FFmpeg...")
	updateProgress(5)

	// 应用下载代理（如有）
	if s.downloadProxy != "" {
		logx.For("transcription").Info("applying download proxy", "proxy", s.downloadProxy)
		if err := httpx.SetProxy(s.downloadProxy); err != nil {
			fail(fmt.Errorf("代理设置失败: %w", err))
			return
		}
		defer func() {
			httpx.ClearProxy()
			logx.For("transcription").Info("download proxy cleared")
		}()
		updateStatus("正在通过代理连接 GitHub...")
	}

	// 为下载设置 ResponseHeaderTimeout，防止代理/服务器无响应导致永久挂起。
	// 恢复默认值（0 = 无限制）避免影响后续普通请求。
	if cl := httpx.Client(); cl != nil {
		if tr, ok := cl.Transport.(*http.Transport); ok {
			prev := tr.ResponseHeaderTimeout
			tr.ResponseHeaderTimeout = 30 * time.Second
			defer func() { tr.ResponseHeaderTimeout = prev }()
		}
	}

	tmpPath := filepath.Join(os.TempDir(), "yyslsplayer-ffmpeg.zip")
	firstReport := true
	if err := downloadToFileWithProgress(FFMPEG_RELEASE_URL, tmpPath, func(dp DownloadProgress) {
		if firstReport {
			firstReport = false
			updateStatus("已连接，正在下载...")
		}
		updateProgress(5 + dp.Pct/2) // 下载占 50%
		updateDetail(dp.SpeedBps, dp.Downloaded, dp.Total, dp.EtaSec)
	}); err != nil {
		fail(fmt.Errorf("下载失败: %w", err))
		return
	}
	defer os.Remove(tmpPath)
	updateProgress(55)

	updateStatus("正在解压 FFmpeg...")
	extracted, err := extractZipFiles(tmpPath, targetDir, map[string]string{
		"*/bin/ffmpeg.exe":  "ffmpeg.exe",
		"*/bin/ffprobe.exe": "ffprobe.exe",
	})
	if err != nil {
		fail(fmt.Errorf("解压失败: %w", err))
		return
	}
	updateProgress(90)

	if extracted < 2 {
		fail(fmt.Errorf("解压后文件不完整 (extracted=%d)", extracted))
		return
	}

	updateStatus("安装完成！")
	updateProgress(100)
	logx.For("transcription").Info("ffmpeg installed via child window")

	time.Sleep(800 * time.Millisecond)
	if w != nil {
		w.Close()
	}

	// 通知主窗口刷新
	if s.executor.emitter != nil {
		s.executor.emitter.Emit(EventFfmpegInstalled, map[string]string{
			"status": "ok",
		})
	}
}

// DownloadProgress 下载进度详情，用于子窗口展示。
type DownloadProgress struct {
	Pct        int   // 百分比 0-100
	Downloaded int64 // 已下载字节
	Total      int64 // 总字节（-1 未知）
	SpeedBps   int64 // 瞬时速度 bytes/s
	EtaSec     int   // 预估剩余秒数（-1 未知）
}

// downloadToFileWithProgress 带详细进度回调的下载。
// 注意：下载大文件（~120MB）可能耗时较长，使用 30 分钟超时而非 httpx.DefaultTimeout。
func downloadToFileWithProgress(url, dest string, onProgress func(DownloadProgress)) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	req, cancelReq, err := httpx.NewRequest(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	defer cancelReq()

	resp, err := httpx.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	totalSize := resp.ContentLength
	// 立即上报初始状态，让子窗口知道连接已建立、总大小等信息
	onProgress(DownloadProgress{Pct: 0, Downloaded: 0, Total: totalSize, SpeedBps: 0, EtaSec: -1})

	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()

	buf := make([]byte, 32*1024)
	var downloaded int64
	lastPct := 0
	startTime := time.Now()
	lastSampleTime := startTime
	var lastSampleBytes int64

	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			f.Write(buf[:n])
			downloaded += int64(n)

			now := time.Now()
			elapsed := now.Sub(startTime)
			sampleElapsed := now.Sub(lastSampleTime)

			// 每 0.5s 更新一次速度采样，避免抖动
			var speedBps int64
			var etaSec int
			if sampleElapsed >= 500*time.Millisecond {
				speedBps = int64(float64(downloaded-lastSampleBytes) / sampleElapsed.Seconds())
				lastSampleTime = now
				lastSampleBytes = downloaded
			}

			if totalSize > 0 && speedBps > 0 {
				etaSec = int(float64(totalSize-downloaded) / float64(speedBps))
			}

			pct := 0
			if totalSize > 0 {
				pct = int(downloaded * 100 / totalSize)
			} else if elapsed > 0 {
				// 无 Content-Length 时用时间估算
				pct = int(float64(elapsed.Milliseconds()) / 300_000 * 100)
				if pct > 95 {
					pct = 95
				}
			}

			if pct > lastPct || sampleElapsed >= 500*time.Millisecond {
				lastPct = pct
				onProgress(DownloadProgress{
					Pct:        pct,
					Downloaded: downloaded,
					Total:      totalSize,
					SpeedBps:   speedBps,
					EtaSec:     etaSec,
				})
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return readErr
		}
	}
	onProgress(DownloadProgress{Pct: 100, Downloaded: downloaded, Total: totalSize})
	return nil
}

// InstallModel 尝试安装转录模型依赖（pip install basic-pitch）。
// 前提条件：系统已安装 Python 3 + pip。
func (s *Service) InstallModel(_ context.Context) (string, error) {
	// 检查 pip 是否可用
	if err := checkPipAvailable(); err != nil {
		return "", fmt.Errorf("Python/pip 环境未配置: %w\n请先安装 Python 3，然后运行 pip install basic-pitch", err)
	}

	logx.For("transcription").Info("installing basic-pitch via pip")

	// 使用 procx 运行 pip install
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	err := procx.Run(ctx, procx.Spec{
		Name: "pip",
		Args: []string{"install", "basic-pitch"},
		OnStderr: func(line string) {
			logx.For("transcription").Debug("pip stderr", "line", line)
		},
		KillGracePeriod: 10 * time.Second,
	})
	if err != nil {
		return "", fmt.Errorf("pip install basic-pitch 失败: %w", err)
	}

	return "Model 依赖安装完成，重启应用后生效", nil
}

// checkPipAvailable 检查 pip 是否可用。
func checkPipAvailable() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return procx.Run(ctx, procx.Spec{Name: "pip", Args: []string{"--version"}})
}

// ProbeAudio 探测音频文件格式信息。
func (s *Service) ProbeAudio(_ context.Context, path string) (AudioProbeResult, error) {
	if s.executor.audioProber == nil {
		return AudioProbeResult{}, fmt.Errorf("audio prober not configured")
	}
	result, err := s.executor.audioProber.Probe(path)
	if err != nil {
		return AudioProbeResult{}, err
	}
	return AudioProbeResult{
		Format:        result.Format,
		DurationMs:    result.DurationMs,
		SampleRate:    result.SampleRate,
		Channels:      result.Channels,
		Bitrate:       result.Bitrate,
		Codec:         result.Codec,
		Container:     result.Container,
		FileSizeBytes: result.FileSizeBytes,
	}, nil
}

// CreateTask 创建转录任务并入队。
func (s *Service) CreateTask(_ context.Context, req CreateTaskRequest) (TranscriptionTaskDTO, error) {
	fileHash, err := computeFileHash(req.SourcePath)
	if err != nil {
		return TranscriptionTaskDTO{}, fmt.Errorf("audio.file_not_found: %w", err)
	}
	cfg := req.Config
	if cfg.Mode == "" {
		storedCfg, _ := s.store().GetTranscriptionConfig()
		cfg = configFromStorage(storedCfg)
	}
	configJSON, _ := json.Marshal(cfg)
	task := &storage.TranscriptionTask{
		SourcePath:     req.SourcePath,
		SourceFileName: filepath.Base(req.SourcePath),
		SourceHash:     fileHash,
		Status:         string(StatusQueued),
		Stage:          string(StageQueued),
		ConfigJSON:     string(configJSON),
	}
	if err := s.store().CreateTranscriptionTask(task); err != nil {
		return TranscriptionTaskDTO{}, fmt.Errorf("transcription.storage_failed: %w", err)
	}
	logx.For("transcription").Info("task created", "id", task.ID, "source", req.SourcePath)
	return taskToDTO(task), nil
}

// ListTasks 分页列出转录任务。
func (s *Service) ListTasks(_ context.Context, limit, offset int) ([]TranscriptionTaskDTO, error) {
	tasks := s.store().ListTranscriptionTasks(limit, offset)
	dtos := make([]TranscriptionTaskDTO, len(tasks))
	for i, t := range tasks {
		dtos[i] = taskToDTO(&t)
	}
	return dtos, nil
}

// GetTask 获取转录任务详情（含音符和分析）。
func (s *Service) GetTask(_ context.Context, id uint) (TranscriptionTaskDetailDTO, error) {
	task, ok := s.store().GetTranscriptionTask(id)
	if !ok {
		return TranscriptionTaskDetailDTO{}, fmt.Errorf("transcription.task_not_found")
	}
	notes := s.store().ListNotesByTask(id)
	analysis := s.store().ListAnalysisByTask(id)
	noteDTOs := make([]TranscriptionNoteDTO, len(notes))
	for i, n := range notes { noteDTOs[i] = noteToDTO(&n) }
	analysisDTOs := make([]TranscriptionAnalysisDTO, len(analysis))
	for i, a := range analysis { analysisDTOs[i] = analysisToDTO(&a) }
	return TranscriptionTaskDetailDTO{
		Task:             taskToDTO(&task),
		ConfigJSON:       task.ConfigJSON,
		Engine:           task.Engine,
		EngineVersion:    task.EngineVersion,
		SampleRate:       task.SampleRate,
		Channels:         task.Channels,
		SourceHash:       task.SourceHash,
		ResultMidiPath:   task.ResultMidiPath,
		ImportedProjectID: task.ImportedProjectID,
		SummaryJSON:      task.SummaryJSON,
		ReportJSON:       task.ReportJSON,
		Notes:            noteDTOs,
		Analysis:         analysisDTOs,
	}, nil
}

// CancelTask 取消一个排队中或运行中的任务。
func (s *Service) CancelTask(_ context.Context, id uint) error {
	task, ok := s.store().GetTranscriptionTask(id)
	if !ok { return fmt.Errorf("transcription.task_not_found") }
	currentStatus := TaskStatus(task.Status)
	if currentStatus == StatusQueued {
		task.Status = string(StatusCancelled)
		_ = s.store().UpdateTranscriptionTask(&task)
		return nil
	}
	if currentStatus == StatusRunning {
		task.Status = string(StatusCancelling)
		_ = s.store().UpdateTranscriptionTask(&task)
		s.executor.CancelActive()
		return nil
	}
	return fmt.Errorf("transcription.invalid_state")
}

// RetryTask 使用新参数重试一个失败的任务（创建新任务）。
func (s *Service) RetryTask(_ context.Context, id uint, config TranscriptionConfigDTO) (TranscriptionTaskDTO, error) {
	task, ok := s.store().GetTranscriptionTask(id)
	if !ok { return TranscriptionTaskDTO{}, fmt.Errorf("transcription.task_not_found") }
	return s.CreateTask(context.Background(), CreateTaskRequest{SourcePath: task.SourcePath, Config: config})
}

// DeleteTask 删除任务及其关联数据和文件。
func (s *Service) DeleteTask(_ context.Context, id uint) error {
	if err := s.store().DeleteTranscriptionTask(id); err != nil {
		return fmt.Errorf("transcription.task_not_found: %w", err)
	}
	_ = cleanupTaskWorkDir(s.dataDir(), id)
	return nil
}

// ImportResultAsMidiProject 将转录结果导入为 MidiProject。
func (s *Service) ImportResultAsMidiProject(_ context.Context, id uint) (MidiProjectImportResult, error) {
	task, ok := s.store().GetTranscriptionTask(id)
	if !ok { return MidiProjectImportResult{}, fmt.Errorf("transcription.task_not_found") }
	if task.Status != string(StatusCompleted) { return MidiProjectImportResult{}, fmt.Errorf("transcription.invalid_state") }
	if task.ResultMidiPath == "" { return MidiProjectImportResult{}, fmt.Errorf("midi.import_failed: no result MIDI") }
	if task.ImportedProjectID != nil { return MidiProjectImportResult{ProjectID: *task.ImportedProjectID, DisplayName: task.SourceFileName + " - 转录"}, nil }
	displayName := task.SourceFileName
	if displayName == "" { displayName = "转录结果" }
	displayName += " - 转录"
	projectID, noteCount, durationMs, fileHash, err := s.executor.midiImporter.Import(task.ResultMidiPath, displayName)
	if err != nil { return MidiProjectImportResult{}, fmt.Errorf("midi.import_failed: %w", err) }
	task.ImportedProjectID = &projectID
	_ = s.store().UpdateTranscriptionTask(&task)
	profile := s.createTranscriptionProfile(projectID, task.ConfigJSON)
	if saved, err := s.store().SaveProfile(profile); err == nil { _ = s.store().UpdateProjectDefaultProfile(projectID, saved.ID) }
	if smokeErr := midi.BuildPlayPlanForProject(s.store(), projectID); smokeErr != nil {
		return MidiProjectImportResult{ProjectID: projectID, DisplayName: displayName, NoteCount: noteCount, DurationMs: durationMs, FileHash: fileHash}, fmt.Errorf("midi.playplan_failed: %w", smokeErr)
	}
	return MidiProjectImportResult{ProjectID: projectID, DisplayName: displayName, NoteCount: noteCount, DurationMs: durationMs, FileHash: fileHash}, nil
}

func (s *Service) createTranscriptionProfile(projectID uint, configJSON string) storage.MidiProfile {
	now := time.Now().UnixMilli()
	p := storage.MidiProfile{ProjectID: &projectID, Name: "转录默认", BaseNote: 48, OutOfRangePolicy: "drop", Speed: 1.0, MinPressMs: 35, ReleaseGapMs: 15, CreatedAt: now, UpdatedAt: now}
	if configJSON != "" {
		var cfg TranscriptionConfigDTO
		if err := json.Unmarshal([]byte(configJSON), &cfg); err == nil {
			p.BaseNote = cfg.TargetBaseNote
			p.OutOfRangePolicy = cfg.OutOfRangePolicy
		}
	}
	return p
}

// ExportResultMidi 导出转录结果 MIDI 到用户指定路径。
func (s *Service) ExportResultMidi(_ context.Context, id uint, targetPath string) error {
	task, ok := s.store().GetTranscriptionTask(id)
	if !ok { return fmt.Errorf("transcription.task_not_found") }
	if task.ResultMidiPath == "" { return fmt.Errorf("midi.write_failed: no result MIDI") }
	const maxMidiSize int64 = 50 << 20
	src, err := filex.ReadLimit(task.ResultMidiPath, maxMidiSize)
	if err != nil { return fmt.Errorf("midi.write_failed: read source: %w", err) }
	return filex.WriteAtomic(targetPath, src, 0644)
}

// GetConfig 获取默认转录配置。
func (s *Service) GetConfig(_ context.Context) (TranscriptionConfigDTO, error) {
	cfg, err := s.store().GetTranscriptionConfig()
	if err != nil { return TranscriptionConfigDTO{}, err }
	return configFromStorage(cfg), nil
}

// UpdateConfig 更新默认转录配置。
func (s *Service) UpdateConfig(_ context.Context, dto TranscriptionConfigDTO) error {
	return s.store().UpdateTranscriptionConfig(func(c *storage.TranscriptionConfig) {
		c.Mode = dto.Mode
		c.MinConfidence = dto.MinConfidence
		c.MinDurationMs = dto.MinDurationMs
		c.MergeGapMs = dto.MergeGapMs
		c.Quantize = dto.Quantize
		c.MaxPolyphony = dto.MaxPolyphony
		c.TargetBaseNote = dto.TargetBaseNote
		c.TargetLaneCount = dto.TargetLaneCount
		c.OutOfRangePolicy = dto.OutOfRangePolicy
		c.PreferMelodyRegister = dto.PreferMelodyRegister
	})
}

// ===== 内部辅助 =====

// installFfmpegHTML 进度窗口 HTML 模板。
const installFfmpegHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<style>
* { margin:0; padding:0; box-sizing:border-box; }
body { font-family: "Microsoft YaHei", "PingFang SC", sans-serif;
  background: #1a1a2e; color: #e0e0e0; display:flex;
  align-items:center; justify-content:center; height:100vh; }
#container { width: 360px; text-align: center; }
#title { font-size: 16px; font-weight: 600; margin-bottom: 20px; }
#status { font-size: 13px; opacity: 0.7; margin-bottom: 14px; min-height: 18px; }
#bar-outer { background: #2a2a3e; border-radius: 6px; height: 8px; overflow: hidden; margin-bottom: 10px; }
#bar-inner { background: linear-gradient(90deg, #6366f1, #8b5cf6); height: 100%; width: 0%;
  border-radius: 6px; transition: width 0.3s ease; }
#pct { font-size: 12px; opacity: 0.5; margin-bottom: 16px; }
#detail { display: grid; grid-template-columns: 1fr 1fr; gap: 8px 16px;
  font-size: 12px; opacity: 0.55; text-align: left; }
#detail .label { opacity: 0.6; }
#detail .value { font-variant-numeric: tabular-nums; }
#detail .right { text-align: right; }
</style>
</head>
<body>
<div id="container">
  <div id="title">正在安装 FFmpeg</div>
  <div id="status">准备中...</div>
  <div id="bar-outer"><div id="bar-inner"></div></div>
  <div id="pct">0%</div>
  <div id="detail">
    <div><span class="label">下载速度</span> <span class="value" id="speed">--</span></div>
    <div class="right"><span class="label">已下载</span> <span class="value" id="downloaded">--</span></div>
    <div><span class="label">剩余时间</span> <span class="value" id="eta">--</span></div>
    <div class="right"><span class="label">总大小</span> <span class="value" id="total">--</span></div>
  </div>
</div>
<script>
function setStatus(msg) { document.getElementById('status').innerText = msg; }
function setProgress(pct) {
  document.getElementById('bar-inner').style.width = pct + '%';
  document.getElementById('pct').innerText = pct + '%';
}
function fmtSize(bytes) {
  if (bytes <= 0) return '--';
  if (bytes < 1024) return bytes + ' B';
  if (bytes < 1048576) return (bytes / 1024).toFixed(1) + ' KB';
  return (bytes / 1048576).toFixed(1) + ' MB';
}
function fmtSpeed(bps) {
  if (bps <= 0) return '--';
  if (bps < 1024) return bps + ' B/s';
  if (bps < 1048576) return (bps / 1024).toFixed(1) + ' KB/s';
  return (bps / 1048576).toFixed(1) + ' MB/s';
}
function fmtEta(sec) {
  if (sec < 0) return '--';
  if (sec < 60) return sec + ' 秒';
  if (sec < 3600) return Math.floor(sec/60) + ' 分 ' + (sec%60) + ' 秒';
  return Math.floor(sec/3600) + ' 时 ' + Math.floor((sec%3600)/60) + ' 分';
}
function setDetail(speedBps, downloaded, total, etaSec) {
  document.getElementById('speed').innerText = fmtSpeed(speedBps);
  document.getElementById('downloaded').innerText = fmtSize(downloaded);
  document.getElementById('total').innerText = total > 0 ? fmtSize(total) : '未知';
  document.getElementById('eta').innerText = fmtEta(etaSec);
}
</script>
</body>
</html>`

// extractZipFiles 从 zip 中按 glob 匹配提取文件。
// targets: glob pattern → 输出文件名
// 返回成功提取的文件数。
func extractZipFiles(zipPath, destDir string, targets map[string]string) (int, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return 0, err
	}
	defer r.Close()

	extracted := 0
	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		for pattern, outName := range targets {
			matched, _ := filepath.Match(pattern, filepath.ToSlash(f.Name))
			if matched {
				outPath := filepath.Join(destDir, outName)
				if err := extractZipFile(f, outPath); err != nil {
					return extracted, fmt.Errorf("extract %s: %w", f.Name, err)
				}
				extracted++
			}
		}
	}
	return extracted, nil
}

func extractZipFile(f *zip.File, dest string) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, rc)
	return err
}

// ===== 辅助函数 =====

func taskToDTO(t *storage.TranscriptionTask) TranscriptionTaskDTO {
	return TranscriptionTaskDTO{
		ID:             t.ID,
		SourceFileName: t.SourceFileName,
		Status:         TaskStatus(t.Status),
		Stage:          TaskStage(t.Stage),
		Progress:       t.Progress,
		DurationMs:     t.DurationMs,
		ErrorCode:      t.ErrorCode,
		ErrorMessage:   t.ErrorMessage,
		CreatedAt:      t.CreatedAt,
		UpdatedAt:      t.UpdatedAt,
	}
}

func noteToDTO(n *storage.TranscriptionNote) TranscriptionNoteDTO {
	return TranscriptionNoteDTO{
		ID:         n.ID,
		TaskID:     n.TaskID,
		Note:       n.Note,
		Velocity:   n.Velocity,
		StartMs:    n.StartMs,
		DurationMs: n.DurationMs,
		Confidence: n.Confidence,
		Source:     n.Source,
		FlagsJSON:  n.FlagsJSON,
	}
}

func analysisToDTO(a *storage.TranscriptionAnalysis) TranscriptionAnalysisDTO {
	return TranscriptionAnalysisDTO{
		ID:          a.ID,
		TaskID:      a.TaskID,
		Kind:        a.Kind,
		PayloadJSON: a.PayloadJSON,
		CreatedAt:   a.CreatedAt,
	}
}

func computeFileHash(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return fmt.Sprintf("%x", sum), nil
}

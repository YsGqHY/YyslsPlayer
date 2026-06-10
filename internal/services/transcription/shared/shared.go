//go:build completion

// Package shared 提供转录服务的共享数据类型。
// 不依赖 transcription 包的任何其他部分，避免循环导入。
package shared

// ===== 后处理相关类型 =====

// Config 转录参数配置（被 postprocess / transcription / engine 共享）。
type Config struct {
	Mode                 string  `json:"mode"`
	MinConfidence        float64 `json:"minConfidence"`
	MinDurationMs        int     `json:"minDurationMs"`
	MergeGapMs           int     `json:"mergeGapMs"`
	Quantize             string  `json:"quantize"`
	MaxPolyphony         int     `json:"maxPolyphony"`
	TargetBaseNote       int     `json:"targetBaseNote"`
	TargetLaneCount      int     `json:"targetLaneCount"`
	OutOfRangePolicy     string  `json:"outOfRangePolicy"`
	PreferMelodyRegister bool    `json:"preferMelodyRegister"`
	// 分析阶段产出（可选，由 executor 从 analyzer 填充）
	EstimatedBPM      float64      `json:"-"`
	BPMConfidence     float64      `json:"-"`
	KeyEstimate       KeyEstimate  `json:"-"`
	AudioQualityScore float64      `json:"-"`
}

// QualityReport 转录质量报告。
type QualityReport struct {
	OverallScore            float64       `json:"overallScore"`
	TranscriptionConfidence float64       `json:"transcriptionConfidence"`
	PlayabilityScore        float64       `json:"playabilityScore"`
	AudioQualityScore       float64       `json:"audioQualityScore"`
	EstimatedBPM            float64       `json:"estimatedBpm"`
	BPMConfidence           float64       `json:"bpmConfidence"`
	KeyEstimate             KeyEstimate   `json:"keyEstimate"`
	ScaleProfile            ScaleProfile  `json:"scaleProfile"`
	NoteCount               int           `json:"noteCount"`
	DroppedCandidateCount   int           `json:"droppedCandidateCount"`
	LowConfidenceCount      int           `json:"lowConfidenceCount"`
	OutOfRangeCount         int           `json:"outOfRangeCount"`
	ShortNoteCount          int           `json:"shortNoteCount"`
	MaxPolyphony            int           `json:"maxPolyphony"`
	DenseSegments           []DenseSegment `json:"denseSegments"`
	SuggestedTranspose      int           `json:"suggestedTranspose"`
	SuggestedOctaveShift    int           `json:"suggestedOctaveShift"`
	Warnings                []string      `json:"warnings"`
}

// KeyEstimate 调性估计。
type KeyEstimate struct {
	Tonic      string          `json:"tonic"`
	Mode       string          `json:"mode"`
	Scale      string          `json:"scale"`
	Confidence float64         `json:"confidence"`
	Method     string          `json:"method"`
	Candidates []KeyCandidate  `json:"candidates"`
}

// KeyCandidate 候选调性。
type KeyCandidate struct {
	Tonic      string  `json:"tonic"`
	Mode       string  `json:"mode"`
	Confidence float64 `json:"confidence"`
}

// ScaleProfile 音阶特征。
type ScaleProfile struct {
	PitchClassHistogram []float64 `json:"pitchClassHistogram"`
	DetectedScaleNotes  []int     `json:"detectedScaleNotes"`
	OutOfScaleRate      float64   `json:"outOfScaleRate"`
	SuggestedTranspose  int       `json:"suggestedTranspose"`
}

// DenseSegment 过密片段描述。
type DenseSegment struct {
	StartMs     int64 `json:"startMs"`
	EndMs       int64 `json:"endMs"`
	PolyphonyAt int   `json:"polyphonyAt"`
	NoteCount   int   `json:"noteCount"`
}

// MelodySummary 旋律摘要。
type MelodySummary struct {
	NoteCount        int     `json:"noteCount"`
	DurationMs       int64   `json:"durationMs"`
	MinNote          int     `json:"minNote"`
	MaxNote          int     `json:"maxNote"`
	AverageVelocity  float64 `json:"averageVelocity"`
	EstimatedBPM     float64 `json:"estimatedBpm"`
	DominantRegister string  `json:"dominantRegister"`
	PolyphonyRate    float64 `json:"polyphonyRate"`
	PlayabilityScore float64 `json:"playabilityScore"`
}

// ===== 管道诊断相关类型 =====

// PipelineDebug 转录管道完整诊断记录，每个任务产生一份。
// 作为 analysis (kind="pipelineDebug") 持久化，前端解析呈现。
type PipelineDebug struct {
	Stages []StageDebug `json:"stages"`
}

// StageDebug 单个处理阶段的诊断记录。
type StageDebug struct {
	Stage       string  `json:"stage"`       // probe / decode / analyze / transcribe / postprocess / midi
	Status      string  `json:"status"`      // ok / failed / skipped
	StartTime   int64   `json:"startTime"`   // unix millis
	EndTime     int64   `json:"endTime"`     // unix millis
	DurationMs  int64   `json:"durationMs"`  // 耗时毫秒
	Input       string  `json:"input"`       // 输入摘要
	Output      string  `json:"output"`      // 输出摘要
	Diagnostics string  `json:"diagnostics"` // 诊断详情
	Error       string  `json:"error,omitempty"` // 错误信息（仅 failed 时）
}

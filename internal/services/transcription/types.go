//go:build completion

package transcription

// TaskStatus 表示转录任务的当前生命周期状态（对齐文档 §8.6）。
type TaskStatus string

const (
	StatusQueued    TaskStatus = "queued"
	StatusRunning   TaskStatus = "running"
	StatusCancelling TaskStatus = "cancelling"
	StatusCompleted TaskStatus = "completed"
	StatusFailed    TaskStatus = "failed"
	StatusCancelled TaskStatus = "cancelled"
)

// TaskStage 表示转录任务的执行阶段（对齐文档 §8.4）。
type TaskStage string

const (
	StageQueued      TaskStage = "queued"
	StageProbe       TaskStage = "probe"
	StageDecode      TaskStage = "decode"
	StageAnalyze     TaskStage = "analyze"
	StageTranscribe  TaskStage = "transcribe"
	StagePostprocess TaskStage = "postprocess"
	StageMidi        TaskStage = "midi"
	StageCompleted   TaskStage = "completed"
)

// ===== DTO ====

// CreateTaskRequest 创建转录任务请求。
type CreateTaskRequest struct {
	SourcePath string              `json:"sourcePath"`
	Config     TranscriptionConfigDTO `json:"config"`
}

// TranscriptionTaskDTO 转录任务摘要（列表用）。
type TranscriptionTaskDTO struct {
	ID             uint       `json:"id"`
	SourceFileName string     `json:"sourceFileName"`
	Status         TaskStatus `json:"status"`
	Stage          TaskStage  `json:"stage"`
	Progress       float64    `json:"progress"`
	DurationMs     int64      `json:"durationMs"`
	ErrorCode      *string    `json:"errorCode,omitempty"`
	ErrorMessage   *string    `json:"errorMessage,omitempty"`
	CreatedAt      int64      `json:"createdAt"`
	UpdatedAt      int64      `json:"updatedAt"`
}

// TranscriptionTaskDetailDTO 转录任务详情（含音符和分析）。
type TranscriptionTaskDetailDTO struct {
	Task             TranscriptionTaskDTO        `json:"task"`
	ConfigJSON       string                      `json:"configJson"`
	Engine           string                      `json:"engine"`
	EngineVersion    string                      `json:"engineVersion"`
	SampleRate       int                         `json:"sampleRate"`
	Channels         int                         `json:"channels"`
	SourceHash       string                      `json:"sourceHash"`
	ResultMidiPath   string                      `json:"resultMidiPath,omitempty"`
	ImportedProjectID *uint                       `json:"importedProjectId,omitempty"`
	SummaryJSON      string                      `json:"summaryJson,omitempty"`
	ReportJSON       string                      `json:"reportJson,omitempty"`
	Notes            []TranscriptionNoteDTO      `json:"notes"`
	Analysis         []TranscriptionAnalysisDTO  `json:"analysis"`
}

// TranscriptionNoteDTO 转录音符候选。
type TranscriptionNoteDTO struct {
	ID         uint    `json:"id"`
	TaskID     uint    `json:"taskId"`
	Note       int     `json:"note"`
	Velocity   int     `json:"velocity"`
	StartMs    int64   `json:"startMs"`
	DurationMs int64   `json:"durationMs"`
	Confidence float64 `json:"confidence"`
	Source     string  `json:"source"`
	FlagsJSON  string  `json:"flagsJson,omitempty"`
}

// TranscriptionAnalysisDTO 分析结果。
type TranscriptionAnalysisDTO struct {
	ID          uint   `json:"id"`
	TaskID      uint   `json:"taskId"`
	Kind        string `json:"kind"`
	PayloadJSON string `json:"payloadJson"`
	CreatedAt   int64  `json:"createdAt"`
}

// AudioProbeResult 音频文件探测结果。
type AudioProbeResult struct {
	Format       string  `json:"format"`
	DurationMs   int64   `json:"durationMs"`
	SampleRate   int     `json:"sampleRate"`
	Channels     int     `json:"channels"`
	Bitrate      int64   `json:"bitrate"`
	Codec        string  `json:"codec"`
	Container    string  `json:"container"`
	FileSizeBytes int64  `json:"fileSizeBytes"`
}

// TranscriptionConfigDTO 转录参数配置。
type TranscriptionConfigDTO struct {
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
}

// TranscriptionQualityReport 转录质量报告（对齐文档 §11）。
type TranscriptionQualityReport struct {
	OverallScore            float64          `json:"overallScore"`
	TranscriptionConfidence float64          `json:"transcriptionConfidence"`
	PlayabilityScore        float64          `json:"playabilityScore"`
	AudioQualityScore       float64          `json:"audioQualityScore"`
	EstimatedBPM            float64          `json:"estimatedBpm"`
	BPMConfidence           float64          `json:"bpmConfidence"`
	KeyEstimate             KeyEstimateDTO   `json:"keyEstimate"`
	ScaleProfile            ScaleProfileDTO  `json:"scaleProfile"`
	NoteCount               int              `json:"noteCount"`
	DroppedCandidateCount   int              `json:"droppedCandidateCount"`
	LowConfidenceCount      int              `json:"lowConfidenceCount"`
	OutOfRangeCount         int              `json:"outOfRangeCount"`
	ShortNoteCount          int              `json:"shortNoteCount"`
	MaxPolyphony            int              `json:"maxPolyphony"`
	DenseSegments           []DenseSegment   `json:"denseSegments"`
	SuggestedTranspose      int              `json:"suggestedTranspose"`
	SuggestedOctaveShift    int              `json:"suggestedOctaveShift"`
	Warnings                []string         `json:"warnings"`
}

// KeyEstimateDTO 调性估计（对齐文档 §7.1）。
type KeyEstimateDTO struct {
	Tonic      string           `json:"tonic"`
	Mode       string           `json:"mode"`
	Scale      string           `json:"scale"`
	Confidence float64          `json:"confidence"`
	Method     string           `json:"method"`
	Candidates []KeyCandidateDTO `json:"candidates"`
}

// KeyCandidateDTO 候选调性。
type KeyCandidateDTO struct {
	Tonic      string  `json:"tonic"`
	Mode       string  `json:"mode"`
	Confidence float64 `json:"confidence"`
}

// ScaleProfileDTO 音阶特征（对齐文档 §7.2）。
type ScaleProfileDTO struct {
	PitchClassHistogram []float64 `json:"pitchClassHistogram"`
	DetectedScaleNotes  []int     `json:"detectedScaleNotes"`
	OutOfScaleRate      float64   `json:"outOfScaleRate"`
	SuggestedTranspose  int       `json:"suggestedTranspose"`
}

// DenseSegment 过密片段描述。
type DenseSegment struct {
	StartMs      int64 `json:"startMs"`
	EndMs        int64 `json:"endMs"`
	PolyphonyAt  int   `json:"polyphonyAt"`
	NoteCount    int   `json:"noteCount"`
}

// TranscriptionCapabilityDTO 能力检测结果（对齐文档 §15.4）。
type TranscriptionCapabilityDTO struct {
	TranscriptionEnabled bool     `json:"transcriptionEnabled"`
	BuildFlavor          string   `json:"buildFlavor"`
	MissingComponents    []string `json:"missingComponents"`
}

// MidiProjectImportResult 导入 MidiProject 的结果。
type MidiProjectImportResult struct {
	ProjectID   uint   `json:"projectId"`
	DisplayName string `json:"displayName"`
	NoteCount   int    `json:"noteCount"`
	DurationMs  int64  `json:"durationMs"`
	FileHash    string `json:"fileHash"`
}

// ===== 事件类型 =====

// TranscriptionProgress 向前端推送的进度快照。
type TranscriptionProgress struct {
	TaskID   string `json:"taskId"`
	Status   string `json:"status"`
	Progress float64 `json:"progress"`
	Message  string `json:"message,omitempty"`
}

// TranscriptionResult 转录完成后的汇总信息。
type TranscriptionResult struct {
	TaskID               string  `json:"taskId"`
	MidiProjectID        string  `json:"midiProjectId,omitempty"`
	TotalNotes           int     `json:"totalNotes"`
	InRangeNotes         int     `json:"inRangeNotes"`
	OutRangeNotes        int     `json:"outRangeNotes"`
	EstimatedBPM         float64 `json:"estimatedBpm"`
	SuggestedOctaveShift int     `json:"suggestedOctaveShift"`
	CoveragePercent      float64 `json:"coveragePercent"`
	QualityReport        string  `json:"qualityReport,omitempty"`
}

// TranscriptionError 转录失败事件负载。
type TranscriptionError struct {
	TaskID       string `json:"taskId"`
	ErrorCode    string `json:"errorCode"`
	ErrorMessage string `json:"errorMessage"`
}

// 事件名称常量。前端通过 Wails events 订阅。
const (
	EventTaskProgress  = "transcription:task:progress"
	EventTaskCompleted = "transcription:task:completed"
	EventTaskFailed    = "transcription:task:failed"
	EventTaskCancelled = "transcription:task:cancelled"
)

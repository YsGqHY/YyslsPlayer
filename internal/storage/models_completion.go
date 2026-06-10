//go:build completion

package storage

const (
	TranscriptionTasksTable    = "transcription_tasks"
	TranscriptionNotesTable    = "transcription_notes"
	TranscriptionAnalysisTable = "transcription_analysis"
	TranscriptionConfigTable   = "transcription_config"
)

// appendCompletionModels 将 completion 版本专属的持久化模型追加到 AllModels。
// lite 版本编译时此函数为空。
func appendCompletionModels() {
	AllModels = append(AllModels,
		ModelDescriptor{
			Model:     &TranscriptionTask{},
			TableName: TranscriptionTasksTable,
			LabelKey:  "transcriptionTasks",
			Clearable: true,
		},
		ModelDescriptor{
			Model:     &TranscriptionNote{},
			TableName: TranscriptionNotesTable,
			LabelKey:  "transcriptionNotes",
			Clearable: true,
		},
		ModelDescriptor{
			Model:     &TranscriptionAnalysis{},
			TableName: TranscriptionAnalysisTable,
			LabelKey:  "transcriptionAnalysis",
			Clearable: true,
		},
		ModelDescriptor{
			Model:     &TranscriptionConfig{},
			TableName: TranscriptionConfigTable,
			LabelKey:  "transcriptionConfig",
			Clearable: false,
		},
	)
}

// TranscriptionTask 转录任务持久化模型（对齐 docs/开发文档-1.1.0.md §9.1）。
//
// CreatedAt / UpdatedAt 为毫秒时间戳，由 Store 方法手动维护。
type TranscriptionTask struct {
	ID               uint    `gorm:"primaryKey;autoIncrement" json:"id"`
	SourcePath       string  `gorm:"size:1024;not null" json:"sourcePath"`
	SourceFileName   string  `gorm:"size:255" json:"sourceFileName"`
	SourceHash       string  `gorm:"size:128" json:"sourceHash"`
	Status           string  `gorm:"size:32;default:queued" json:"status"`
	Stage            string  `gorm:"size:32" json:"stage"`
	Progress         float64 `gorm:"default:0" json:"progress"`
	ConfigJSON       string  `gorm:"type:text" json:"configJson"`
	Engine           string  `gorm:"size:64" json:"engine"`
	EngineVersion    string  `gorm:"size:64" json:"engineVersion"`
	DurationMs       int64   `json:"durationMs"`
	SampleRate       int     `json:"sampleRate"`
	Channels         int     `json:"channels"`
	ResultMidiPath   string  `gorm:"size:1024" json:"resultMidiPath"`
	ImportedProjectID *uint  `json:"importedProjectId"`
	SummaryJSON      string  `gorm:"type:text" json:"summaryJson"`
	ReportJSON       string  `gorm:"type:text" json:"reportJson"`
	ErrorCode        *string `gorm:"size:64" json:"errorCode"`
	ErrorMessage     *string `gorm:"type:text" json:"errorMessage"`
	CreatedAt        int64   `gorm:"autoCreateTime:false" json:"createdAt"`
	UpdatedAt        int64   `gorm:"autoUpdateTime:false" json:"updatedAt"`
	StartedAt        *int64  `json:"startedAt"`
	FinishedAt       *int64  `json:"finishedAt"`
}

func (TranscriptionTask) TableName() string { return TranscriptionTasksTable }

// TranscriptionNote 转录音符候选持久化模型（对齐 docs/开发文档-1.1.0.md §9.2）。
//
// 每个任务可能包含数千条 note；通过 taskID 索引关联。
type TranscriptionNote struct {
	ID         uint    `gorm:"primaryKey;autoIncrement" json:"id"`
	TaskID     uint    `gorm:"index" json:"taskId"`
	Note       int     `json:"note"`
	Velocity   int     `json:"velocity"`
	StartMs    int64   `json:"startMs"`
	DurationMs int64   `json:"durationMs"`
	Confidence float64 `json:"confidence"`
	Source     string  `gorm:"size:32" json:"source"`    // model / postprocess / manual
	FlagsJSON  string  `gorm:"type:text" json:"flagsJson"` // lowConfidence / merged / quantized / droppedCandidate
}

func (TranscriptionNote) TableName() string { return TranscriptionNotesTable }

// TranscriptionAnalysis 转录分析结果持久化模型（对齐 docs/开发文档-1.1.0.md §9.3）。
//
// 每个任务可有多条分析记录，按 kind 区分类型。payloadJSON 为分析结果 JSON。
type TranscriptionAnalysis struct {
	ID          uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	TaskID      uint   `gorm:"index" json:"taskId"`
	Kind        string `gorm:"size:32" json:"kind"`   // tempo / key / scale / audioQuality / playability
	PayloadJSON string `gorm:"type:text" json:"payloadJson"`
	CreatedAt   int64  `gorm:"autoCreateTime:false" json:"createdAt"`
}

func (TranscriptionAnalysis) TableName() string { return TranscriptionAnalysisTable }

// TranscriptionConfig 转录参数默认配置。
//
// 始终只有 ID=1 一行，通过 Upsert 维护。
type TranscriptionConfig struct {
	ID                 uint    `gorm:"primaryKey" json:"id"`
	Mode               string  `gorm:"size:32;default:melody" json:"mode"`        // melody / polyphonic
	MinConfidence      float64 `gorm:"default:0.55" json:"minConfidence"`
	MinDurationMs      int     `gorm:"default:60" json:"minDurationMs"`
	MergeGapMs         int     `gorm:"default:40" json:"mergeGapMs"`
	Quantize           string  `gorm:"size:16;default:light" json:"quantize"` // off / light / strong
	MaxPolyphony       int     `gorm:"default:2" json:"maxPolyphony"`
	TargetBaseNote     int     `gorm:"default:48" json:"targetBaseNote"`
	TargetLaneCount    int     `gorm:"default:36" json:"targetLaneCount"`
	OutOfRangePolicy   string  `gorm:"size:16;default:drop" json:"outOfRangePolicy"`
	PreferMelodyRegister bool  `gorm:"default:true" json:"preferMelodyRegister"`
	UpdatedAt          int64   `gorm:"autoUpdateTime:false" json:"updatedAt"`
}

func (TranscriptionConfig) TableName() string { return TranscriptionConfigTable }

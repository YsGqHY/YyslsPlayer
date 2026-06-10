package storage

const (
	MidiProjectsTable = "midi_projects"
	MidiEventsTable   = "midi_events"
	MidiProfilesTable = "midi_profiles"
	Keymap36Table     = "keymap_36"
	PlayHistoryTable  = "play_history"
)

// MidiProject stores imported MIDI metadata and the active default profile.
//
// 说明：CreatedAt / UpdatedAt 均为毫秒时间戳，由 Store 方法手动维护。
// GORM 默认会把名为 CreatedAt/UpdatedAt 的 int64 字段当成"自动秒级时间戳"接管，
// 这会覆盖我们写入的毫秒值，因此统一用 autoCreateTime:false / autoUpdateTime:false 关闭。
type MidiProject struct {
	ID               uint    `gorm:"primaryKey;autoIncrement" json:"id"`
	DisplayName      string  `gorm:"size:255" json:"displayName"`
	FileName         string  `gorm:"size:255" json:"fileName"`
	SourcePath       *string `gorm:"size:1024" json:"sourcePath"`
	FileHash         string  `gorm:"size:128;index" json:"fileHash"`
	PPQ              int     `json:"ppq"`
	TrackCount       int     `json:"trackCount"`
	ChannelCount     int     `json:"channelCount"`
	DurationMs       int64   `json:"durationMs"`
	NoteCount        int     `json:"noteCount"`
	FileSizeBytes    int64   `json:"fileSizeBytes"`
	DefaultProfileID *uint   `json:"defaultProfileId"`
	CreatedAt        int64   `gorm:"autoCreateTime:false" json:"createdAt"`
	UpdatedAt        int64   `gorm:"index;autoUpdateTime:false" json:"updatedAt"`
}

func (MidiProject) TableName() string { return MidiProjectsTable }

// MidiEvent stores normalized note events on an absolute millisecond timeline.
type MidiEvent struct {
	ID         uint  `gorm:"primaryKey;autoIncrement" json:"id"`
	ProjectID  uint  `gorm:"index" json:"projectId"`
	Track      int   `json:"track"`
	Channel    int   `json:"channel"`
	Note       int   `json:"note"`
	Velocity   int   `json:"velocity"`
	StartMs    int64 `json:"startMs"`
	DurationMs int64 `json:"durationMs"`
}

func (MidiEvent) TableName() string { return MidiEventsTable }

// MidiProfile stores playable range, timing and mapping parameters.
type MidiProfile struct {
	ID                uint    `gorm:"primaryKey;autoIncrement" json:"id"`
	ProjectID         *uint   `gorm:"index" json:"projectId"`
	Name              string  `gorm:"size:255" json:"name"`
	BaseNote          int     `json:"baseNote"`
	Transpose         int     `json:"transpose"`
	OctaveShift       int     `json:"octaveShift"`
	Speed             float64 `json:"speed"`
	OutOfRangePolicy  string  `gorm:"size:32" json:"outOfRangePolicy"`
	MinPressMs        int     `json:"minPressMs"`
	ReleaseGapMs      int     `json:"releaseGapMs"`
	KeymapProfileID   uint    `json:"keymapProfileId"`
	EnabledTracksJSON string  `gorm:"type:text" json:"enabledTracksJson"`
	CreatedAt         int64   `gorm:"autoCreateTime:false" json:"createdAt"`
	UpdatedAt         int64   `gorm:"autoUpdateTime:false" json:"updatedAt"`
}

func (MidiProfile) TableName() string { return MidiProfilesTable }

// Keymap36 stores one physical key mapping per 36-key lane.
type Keymap36 struct {
	ID               uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	ProfileID        uint   `gorm:"index:idx_keymap_profile_lane" json:"profileId"`
	ProfileName      string `gorm:"size:255" json:"profileName"`
	Lane             int    `gorm:"index:idx_keymap_profile_lane" json:"lane"`
	Label            string `gorm:"size:32" json:"label"`
	PitchClass       int    `json:"pitchClass"`
	IsBlackKey       bool   `json:"isBlackKey"`
	VirtualKey       int    `json:"virtualKey"`
	ScanCode         int    `json:"scanCode"`
	ModifierKeysJSON string `gorm:"type:text" json:"modifierKeysJson"`
	DisplayOrder     int    `json:"displayOrder"`
	CreatedAt        int64  `gorm:"autoCreateTime:false" json:"createdAt"`
	UpdatedAt        int64  `gorm:"autoUpdateTime:false" json:"updatedAt"`
}

func (Keymap36) TableName() string { return Keymap36Table }

// PlayHistory records player sessions for diagnostics and recent activity.
type PlayHistory struct {
	ID         uint    `gorm:"primaryKey;autoIncrement" json:"id"`
	ProjectID  uint    `gorm:"index" json:"projectId"`
	ProfileID  uint    `json:"profileId"`
	StartedAt  int64   `json:"startedAt"`
	EndedAt    *int64  `json:"endedAt"`
	DurationMs int64   `json:"durationMs"`
	Completed  bool    `json:"completed"`
	ErrorCode  *string `gorm:"size:64" json:"errorCode"`
	DryRun     bool    `json:"dryRun"`
}

func (PlayHistory) TableName() string { return PlayHistoryTable }

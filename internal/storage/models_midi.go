package storage

const (
	MidiProjectsTable = "midi_projects"
	MidiEventsTable   = "midi_events"
	MidiProfilesTable = "midi_profiles"
	Keymap36Table     = "keymap_36"
	PlayHistoryTable  = "play_history"
)

// MidiProject stores imported MIDI metadata and the active default profile.
type MidiProject struct {
	ID               uint    `json:"id"`
	DisplayName      string  `json:"displayName"`
	FileName         string  `json:"fileName"`
	SourcePath       *string `json:"sourcePath"`
	FileHash         string  `json:"fileHash"`
	PPQ              int     `json:"ppq"`
	TrackCount       int     `json:"trackCount"`
	ChannelCount     int     `json:"channelCount"`
	DurationMs       int64   `json:"durationMs"`
	NoteCount        int     `json:"noteCount"`
	FileSizeBytes    int64   `json:"fileSizeBytes"`
	DefaultProfileID *uint   `json:"defaultProfileId"`
	CreatedAt        int64   `json:"createdAt"`
	UpdatedAt        int64   `json:"updatedAt"`
}

func (MidiProject) TableName() string { return MidiProjectsTable }

// MidiEvent stores normalized note events on an absolute millisecond timeline.
type MidiEvent struct {
	ID         uint  `json:"id"`
	ProjectID  uint  `json:"projectId"`
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
	ID                uint    `json:"id"`
	ProjectID         *uint   `json:"projectId"`
	Name              string  `json:"name"`
	BaseNote          int     `json:"baseNote"`
	Transpose         int     `json:"transpose"`
	OctaveShift       int     `json:"octaveShift"`
	Speed             float64 `json:"speed"`
	OutOfRangePolicy  string  `json:"outOfRangePolicy"`
	MinPressMs        int     `json:"minPressMs"`
	ReleaseGapMs      int     `json:"releaseGapMs"`
	KeymapProfileID   uint    `json:"keymapProfileId"`
	EnabledTracksJSON string  `json:"enabledTracksJson"`
	CreatedAt         int64   `json:"createdAt"`
	UpdatedAt         int64   `json:"updatedAt"`
}

func (MidiProfile) TableName() string { return MidiProfilesTable }

// Keymap36 stores one physical key mapping per 36-key lane.
type Keymap36 struct {
	ID               uint   `json:"id"`
	ProfileID        uint   `json:"profileId"`
	ProfileName      string `json:"profileName"`
	Lane             int    `json:"lane"`
	Label            string `json:"label"`
	PitchClass       int    `json:"pitchClass"`
	IsBlackKey       bool   `json:"isBlackKey"`
	VirtualKey       int    `json:"virtualKey"`
	ScanCode         int    `json:"scanCode"`
	ModifierKeysJSON string `json:"modifierKeysJson"`
	DisplayOrder     int    `json:"displayOrder"`
	CreatedAt        int64  `json:"createdAt"`
	UpdatedAt        int64  `json:"updatedAt"`
}

func (Keymap36) TableName() string { return Keymap36Table }

// PlayHistory records player sessions for diagnostics and recent activity.
type PlayHistory struct {
	ID         uint    `json:"id"`
	ProjectID  uint    `json:"projectId"`
	ProfileID  uint    `json:"profileId"`
	StartedAt  int64   `json:"startedAt"`
	EndedAt    *int64  `json:"endedAt"`
	DurationMs int64   `json:"durationMs"`
	Completed  bool    `json:"completed"`
	ErrorCode  *string `json:"errorCode"`
	DryRun     bool    `json:"dryRun"`
}

func (PlayHistory) TableName() string { return PlayHistoryTable }

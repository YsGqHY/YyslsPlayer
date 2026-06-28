//go:build completion

package storage

const (
	MacroProfilesTable = "macro_profiles"
	MacroStepsTable    = "macro_steps"
)

// MacroProfile stores user-created keyboard macro metadata and trigger binding.
type MacroProfile struct {
	ID                 uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	Name               string `gorm:"size:255;not null" json:"name"`
	Description        string `gorm:"type:text" json:"description"`
	TriggerAccelerator string `gorm:"size:128;index" json:"triggerAccelerator"`
	Enabled            bool   `gorm:"index" json:"enabled"`
	RepeatMode         string `gorm:"size:32" json:"repeatMode"`
	RepeatCount        int    `json:"repeatCount"`
	RepeatIntervalMs   int64  `json:"repeatIntervalMs"`
	InterruptPolicy    string `gorm:"size:32" json:"interruptPolicy"`
	CreatedAt          int64  `gorm:"autoCreateTime:false" json:"createdAt"`
	UpdatedAt          int64  `gorm:"index;autoUpdateTime:false" json:"updatedAt"`
}

func (MacroProfile) TableName() string { return MacroProfilesTable }

// MacroStep stores one linear block in a keyboard macro timeline.
type MacroStep struct {
	ID               uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	MacroID          uint   `gorm:"index:idx_macro_steps_macro_order" json:"macroId"`
	OrderIndex       int    `gorm:"index:idx_macro_steps_macro_order" json:"orderIndex"`
	Kind             string `gorm:"size:32;not null" json:"kind"`
	KeyLabel         string `gorm:"size:64" json:"keyLabel"`
	VirtualKey       int    `json:"virtualKey"`
	ScanCode         int    `json:"scanCode"`
	DeviceKind       string `gorm:"size:16" json:"deviceKind"`
	ModifierKeysJSON string `gorm:"type:text" json:"modifierKeysJson"`
	DurationMs       int64  `json:"durationMs"`
	WaitMs           int64  `json:"waitMs"`
	PayloadJSON      string `gorm:"type:text" json:"payloadJson"`
	CreatedAt        int64  `gorm:"autoCreateTime:false" json:"createdAt"`
	UpdatedAt        int64  `gorm:"autoUpdateTime:false" json:"updatedAt"`
}

func (MacroStep) TableName() string { return MacroStepsTable }

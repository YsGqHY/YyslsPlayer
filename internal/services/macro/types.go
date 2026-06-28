//go:build completion

package macro

import "errors"

const (
	SourceHotkey = "macro"

	RepeatModeOnce   = "once"
	RepeatModeCount  = "count"
	RepeatModeHold   = "hold"
	RepeatModeToggle = "toggle"

	InterruptIgnore  = "ignore"
	InterruptRestart = "stop-current-and-run"

	StepDelay     = "delay"
	StepKeyTap    = "keyTap"
	StepKeyDown   = "keyDown"
	StepKeyUp     = "keyUp"
	StepChordTap  = "chordTap"
	StepMouseTap    = "mouseTap"
	StepMouseDown   = "mouseDown"
	StepMouseUp     = "mouseUp"
	StepMouseScroll = "mouseScroll"
	// StepMouseMove moves the cursor by a relative (dx, dy) pixel offset carried
	// in the step's PayloadJSON. Mirrors Logitech G HUB's mouse-movement action;
	// zero DB migration since it reuses the payload column.
	StepMouseMove = "mouseMove"
	// StepText injects a Unicode string (chat phrase, etc.) carried in the step's
	// PayloadJSON. Mirrors Logitech G HUB's text-block action; zero DB migration.
	StepText = "text"

	DeviceKeyboard = "keyboard"
	DeviceMouse    = "mouse"

	StateIdle      = "idle"
	StateRunning   = "running"
	StateStopping  = "stopping"
	StateCompleted = "completed"
	StateStopped   = "stopped"
	StateError     = "error"

	RecordStateIdle      = "idle"
	RecordStateRecording = "recording"
	RecordStateStopped   = "stopped"
	RecordStateError     = "error"
)

var (
	ErrMacroNotFound       = errors.New("MACRO_NOT_FOUND")
	ErrMacroInvalid        = errors.New("MACRO_INVALID")
	ErrMacroBusy           = errors.New("MACRO_BUSY")
	ErrMacroPlayerActive   = errors.New("MACRO_PLAYER_ACTIVE")
	ErrMacroRecording      = errors.New("MACRO_RECORDING_ACTIVE")
	ErrMacroNoSteps        = errors.New("MACRO_NO_STEPS")
	ErrMacroTriggerInvalid = errors.New("MACRO_TRIGGER_INVALID")
	ErrMacroRecordUnsup    = errors.New("MACRO_RECORD_UNSUPPORTED")
)

type MacroSummaryDTO struct {
	ID                 uint   `json:"id"`
	Name               string `json:"name"`
	Description        string `json:"description"`
	TriggerAccelerator string `json:"triggerAccelerator"`
	Enabled            bool   `json:"enabled"`
	RepeatMode         string `json:"repeatMode"`
	RepeatCount        int    `json:"repeatCount"`
	RepeatIntervalMs   int64  `json:"repeatIntervalMs"`
	InterruptPolicy    string `json:"interruptPolicy"`
	StepCount          int    `json:"stepCount"`
	Registered         bool   `json:"registered"`
	ErrorCode          string `json:"errorCode"`
	CreatedAt          int64  `json:"createdAt"`
	UpdatedAt          int64  `json:"updatedAt"`
}

type MacroDetailDTO struct {
	Profile MacroSummaryDTO `json:"profile"`
	Steps   []MacroStepDTO  `json:"steps"`
}

type MacroStepDTO struct {
	ID               uint   `json:"id"`
	MacroID          uint   `json:"macroId"`
	OrderIndex       int    `json:"orderIndex"`
	Kind             string `json:"kind"`
	KeyLabel         string `json:"keyLabel"`
	VirtualKey       int    `json:"virtualKey"`
	ScanCode         int    `json:"scanCode"`
	DeviceKind       string `json:"deviceKind"`
	ModifierKeysJSON string `json:"modifierKeysJson"`
	DurationMs       int64  `json:"durationMs"`
	WaitMs           int64  `json:"waitMs"`
	PayloadJSON      string `json:"payloadJson"`
}

type SaveMacroRequest struct {
	ID                 uint           `json:"id"`
	Name               string         `json:"name"`
	Description        string         `json:"description"`
	TriggerAccelerator string         `json:"triggerAccelerator"`
	Enabled            bool           `json:"enabled"`
	RepeatMode         string         `json:"repeatMode"`
	RepeatCount        int            `json:"repeatCount"`
	RepeatIntervalMs   int64          `json:"repeatIntervalMs"`
	InterruptPolicy    string         `json:"interruptPolicy"`
	Steps              []MacroStepDTO `json:"steps"`
}

type MacroStateDTO struct {
	State     string `json:"state"`
	MacroID   uint   `json:"macroId"`
	MacroName string `json:"macroName"`
	StepIndex int    `json:"stepIndex"`
	StepCount int    `json:"stepCount"`
	StartedAt int64  `json:"startedAt"`
	UpdatedAt int64  `json:"updatedAt"`
	ErrorCode string `json:"errorCode"`
	Message   string `json:"message"`
}

type MacroStepEventDTO struct {
	MacroID   uint         `json:"macroId"`
	StepIndex int          `json:"stepIndex"`
	Step      MacroStepDTO `json:"step"`
	At        int64        `json:"at"`
}

type MacroErrorDTO struct {
	MacroID   uint   `json:"macroId"`
	ErrorCode string `json:"errorCode"`
	Message   string `json:"message"`
	At        int64  `json:"at"`
}

type AssignableKeyDTO struct {
	Label      string `json:"label"`
	VirtualKey int    `json:"virtualKey"`
	ScanCode   int    `json:"scanCode"`
	Modifier   bool   `json:"modifier"`
	DeviceKind string `json:"deviceKind"`
}

// RecordStateDTO describes the live recording session status.
type RecordStateDTO struct {
	State     string `json:"state"`
	StepCount int    `json:"stepCount"`
	StartedAt int64  `json:"startedAt"`
	UpdatedAt int64  `json:"updatedAt"`
	ErrorCode string `json:"errorCode"`
	Message   string `json:"message"`
}

// RecordStepEventDTO is emitted as each key/mouse event is captured.
type RecordStepEventDTO struct {
	StepIndex int          `json:"stepIndex"`
	Step      MacroStepDTO `json:"step"`
	At        int64        `json:"at"`
}

// RecordResultDTO is returned by StopRecording with the full captured timeline.
type RecordResultDTO struct {
	Steps     []MacroStepDTO `json:"steps"`
	DurationMs int64         `json:"durationMs"`
}

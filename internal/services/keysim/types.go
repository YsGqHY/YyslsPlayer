// Package keysim centralizes physical keyboard simulation for playback.
package keysim

import (
	"context"
	"errors"
)

const (
	ActionPress   ActionKind = "press"
	ActionRelease ActionKind = "release"
	// ActionText injects a Unicode string carried in KeyAction.Text rather than a
	// single key. Mirrors Logitech G HUB's "text block" macro action.
	ActionText ActionKind = "text"
	// ActionMouseMove moves the cursor by a relative (Dx, Dy) pixel offset rather
	// than pressing a key. Mirrors Logitech G HUB's mouse-movement macro action.
	ActionMouseMove ActionKind = "mouseMove"

	PhysicalDown PhysicalKind = "down"
	PhysicalUp   PhysicalKind = "up"

	DefaultDryRunLogLimit      = 2000
	DefaultModifierHoldDelayMs = 8
)

var (
	ErrUnsupportedPlatform    = errors.New("KEYSIM_UNSUPPORTED_PLATFORM")
	ErrHookLaunderUnavailable = errors.New("KEYSIM_HOOK_LAUNDER_UNAVAILABLE")
	ErrSendFailed             = errors.New("KEYSIM_SEND_FAILED")
	ErrReleaseFailed          = errors.New("KEYSIM_RELEASE_FAILED")
	ErrInvalidAction          = errors.New("KEYSIM_INVALID_ACTION")
	ErrInvalidKey             = errors.New("KEYSIM_INVALID_KEY")
	ErrTextUnsupported        = errors.New("KEYSIM_TEXT_UNSUPPORTED")
	ErrMouseMoveUnsupported   = errors.New("KEYSIM_MOUSE_MOVE_UNSUPPORTED")
)

type ActionKind string
type PhysicalKind string

type Key struct {
	Label      string `json:"label"`
	VirtualKey int    `json:"virtualKey"`
	ScanCode   int    `json:"scanCode"`
	// Kind distinguishes keyboard keys (empty / KeyKindKeyboard) from mouse
	// buttons (KeyKindMouse). Empty keeps backward compatibility with existing
	// MIDI/player keyframes and persisted macro steps.
	Kind string `json:"kind,omitempty"`
}

type KeyAction struct {
	TimeMs         int64      `json:"timeMs"`
	Action         ActionKind `json:"action"`
	Lane           int        `json:"lane"`
	SourceNote     int        `json:"sourceNote"`
	NormalizedNote int        `json:"normalizedNote"`
	Velocity       int        `json:"velocity"`
	Key            Key        `json:"key"`
	Modifiers      []Key      `json:"modifiers"`
	// Text is the Unicode payload sent when Action == ActionText. Ignored otherwise.
	Text string `json:"text,omitempty"`
	// TextDelayMs optionally throttles per-character text injection (ActionText).
	TextDelayMs int64 `json:"textDelayMs,omitempty"`
	// Dx / Dy carry the relative cursor offset in pixels when Action ==
	// ActionMouseMove. Ignored otherwise.
	Dx int `json:"dx,omitempty"`
	Dy int `json:"dy,omitempty"`
}

type KeyEvent struct {
	TimeMs         int64        `json:"timeMs"`
	Kind           PhysicalKind `json:"kind"`
	Key            Key          `json:"key"`
	Lane           int          `json:"lane"`
	SourceNote     int          `json:"sourceNote"`
	NormalizedNote int          `json:"normalizedNote"`
	Velocity       int          `json:"velocity"`
	Modifier       bool         `json:"modifier"`
}

type PressedKey struct {
	Key      Key  `json:"key"`
	Count    int  `json:"count"`
	Modifier bool `json:"modifier"`
}

type RunOptions struct {
	DryRun              bool `json:"dryRun"`
	DryRunLogLimit      int  `json:"dryRunLogLimit"`
	InterKeyDelayMs     int  `json:"interKeyDelayMs"`
	ModifierHoldDelayMs int  `json:"modifierHoldDelayMs"`
}

type RunResult struct {
	DryRun               bool        `json:"dryRun"`
	Keyframes            []KeyAction `json:"keyframes"`
	TotalKeyframes       int         `json:"totalKeyframes"`
	KeyframesTruncated   bool        `json:"keyframesTruncated"`
	Events               []KeyEvent  `json:"events"`
	TotalEvents          int         `json:"totalEvents"`
	Truncated            bool        `json:"truncated"`
	ReleasedKeys         int         `json:"releasedKeys"`
	RecoveryReleasedKeys int         `json:"recoveryReleasedKeys"`
}

type StateSnapshot struct {
	Pressed []PressedKey `json:"pressed"`
}

type Driver interface {
	Send(ctx context.Context, event KeyEvent, opts RunOptions) error
}

// TextSender is implemented by drivers that can inject a Unicode string directly
// (via KEYEVENTF_UNICODE on Windows). Drivers without text support are handled by
// the keysim service returning ErrTextUnsupported.
type TextSender interface {
	SendText(ctx context.Context, text string, opts RunOptions) error
}

// MouseMover is implemented by drivers that can move the cursor by a relative
// (dx, dy) pixel offset (via MOUSEEVENTF_MOVE on Windows). Drivers without move
// support are handled by the keysim service returning ErrMouseMoveUnsupported.
type MouseMover interface {
	MoveMouse(ctx context.Context, dx, dy int, opts RunOptions) error
}

type ChainHeadRefresher interface {
	RefreshChainHead(ctx context.Context) error
}

// Package keysim centralizes physical keyboard simulation for playback.
package keysim

import (
	"context"
	"errors"
)

const (
	ActionPress   ActionKind = "press"
	ActionRelease ActionKind = "release"

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
)

type ActionKind string
type PhysicalKind string

type Key struct {
	Label      string `json:"label"`
	VirtualKey int    `json:"virtualKey"`
	ScanCode   int    `json:"scanCode"`
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

type ChainHeadRefresher interface {
	RefreshChainHead(ctx context.Context) error
}

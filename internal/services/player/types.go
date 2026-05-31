package player

import (
	"errors"

	"YyslsPlayer/internal/services/midi"
)

type PlayerState string

const (
	StateIdle      PlayerState = "idle"
	StateReady     PlayerState = "ready"
	StatePlaying   PlayerState = "playing"
	StatePaused    PlayerState = "paused"
	StateCompleted PlayerState = "completed"
	StateStopped   PlayerState = "stopped"
	StateError     PlayerState = "error"

	EventState    = "player:state"
	EventPosition = "player:position"
	EventError    = "player:error"

	DefaultLookaheadMs = 20
	MinLookaheadMs     = 5
	MaxLookaheadMs     = 50
	DefaultProgressHz  = 10
)

var (
	ErrPlayerBusy        = errors.New("PLAYER_BUSY")
	ErrPlayerNotFound    = errors.New("PLAYER_NOT_FOUND")
	ErrInvalidTransition = errors.New("PLAYER_INVALID_STATE")
	ErrInvalidLookahead  = errors.New("PLAYER_INVALID_LOOKAHEAD")
	ErrInvalidSeek       = errors.New("PLAYER_INVALID_SEEK")
	ErrInvalidKeyFrame   = errors.New("PLAYER_INVALID_KEYFRAME")
	ErrPlayPlanEmpty     = errors.New("PLAYPLAN_EMPTY")
)

type StartRequest struct {
	Plan            midi.PlayPlanDTO `json:"plan"`
	DryRun          bool             `json:"dryRun"`
	LookaheadMs     int              `json:"lookaheadMs"`
	StartPositionMs int64            `json:"startPositionMs"`
}

type PlayerSessionDTO struct {
	SessionID   string      `json:"sessionId"`
	State       PlayerState `json:"state"`
	PositionMs  int64       `json:"positionMs"`
	DurationMs  int64       `json:"durationMs"`
	DryRun      bool        `json:"dryRun"`
	LookaheadMs int         `json:"lookaheadMs"`
	ErrorCode   string      `json:"errorCode"`
	Message     string      `json:"message"`
	ProjectID   uint        `json:"projectId"`
	ProfileID   uint        `json:"profileId"`
	FrameCount  int         `json:"frameCount"`
	StartedAt   int64       `json:"startedAt"`
	UpdatedAt   int64       `json:"updatedAt"`
}

type PlayerStateDTO struct {
	SessionID   string      `json:"sessionId"`
	State       PlayerState `json:"state"`
	PositionMs  int64       `json:"positionMs"`
	DurationMs  int64       `json:"durationMs"`
	DryRun      bool        `json:"dryRun"`
	LookaheadMs int         `json:"lookaheadMs"`
	ErrorCode   string      `json:"errorCode"`
	Message     string      `json:"message"`
}

type PlayerPositionDTO struct {
	SessionID  string      `json:"sessionId"`
	State      PlayerState `json:"state"`
	PositionMs int64       `json:"positionMs"`
	DurationMs int64       `json:"durationMs"`
	Progress   float64     `json:"progress"`
	UpdatedAt  int64       `json:"updatedAt"`
}

type PlayerErrorDTO struct {
	SessionID   string      `json:"sessionId"`
	State       PlayerState `json:"state"`
	PositionMs  int64       `json:"positionMs"`
	DurationMs  int64       `json:"durationMs"`
	DryRun      bool        `json:"dryRun"`
	LookaheadMs int         `json:"lookaheadMs"`
	ErrorCode   string      `json:"errorCode"`
	Message     string      `json:"message"`
	UpdatedAt   int64       `json:"updatedAt"`
}

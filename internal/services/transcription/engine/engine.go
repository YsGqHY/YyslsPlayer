//go:build completion

// Package engine 提供转录引擎 adapter（Basic Pitch、Mock 等）。
package engine

import (
	"context"
)

// Config 引擎运行配置（引擎层不依赖 transcription DTO）。
type Config struct {
	MinConfidence float64 `json:"minConfidence"`
	MinDurationMs int     `json:"minDurationMs"`
	MaxPolyphony  int     `json:"maxPolyphony"`
}

// Engine 转录引擎接口。
type Engine interface {
	Name() string
	Version() string
	Available() bool
	Transcribe(ctx context.Context, pcmPath string, workDir string, config Config) (*Result, error)
}

// Result 转录引擎原始输出。
type Result struct {
	Notes       []RawNote `json:"notes"`
	BPMEstimate float64   `json:"bpmEstimate"`
	DurationMs  int64     `json:"durationMs"`
	Diagnostics string    `json:"diagnostics"`
}

// RawNote 引擎原始音符。
type RawNote struct {
	Note       int     `json:"note"`
	Velocity   float64 `json:"velocity"`
	StartMs    int64   `json:"startMs"`
	DurationMs int64   `json:"durationMs"`
	Confidence float64 `json:"confidence"`
}

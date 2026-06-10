//go:build completion

package engine

import (
	"context"
	"math/rand"
)

// MockEngine 是用于测试的虚拟转录引擎。
// 生成可预测的旋律音符（C Major 音阶循环）。
type MockEngine struct{}

// NewMockEngine 创建 Mock 引擎。
func NewMockEngine() *MockEngine {
	return &MockEngine{}
}

func (m *MockEngine) Name() string    { return "mock" }
func (m *MockEngine) Version() string { return "1.0.0" }
func (m *MockEngine) Available() bool { return true }

// Transcribe 生成一系列模拟音符（C Major 上下行循环）。
func (m *MockEngine) Transcribe(ctx context.Context, pcmPath string, workDir string, config Config) (*Result, error) {
	// C Major 音阶：C4 (60) 到 C5 (72)
	cMajor := []int{60, 62, 64, 65, 67, 69, 71, 72, 71, 69, 67, 65, 64, 62, 60}

	rng := rand.New(rand.NewSource(42)) // 固定种子保证可重复
	notes := make([]RawNote, 0, len(cMajor)*3)

	startMs := int64(0)
	for i := 0; i < 3; i++ { // 3 遍循环
		for _, note := range cMajor {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
			}

			dur := int64(400 + rng.Intn(200)) // 400-600ms
			notes = append(notes, RawNote{
				Note:       note + rng.Intn(3) - 1, // 轻微随机偏移
				Velocity:   float64(80 + rng.Intn(20)),
				StartMs:    startMs,
				DurationMs: dur,
				Confidence: 0.85 + rng.Float64()*0.15,
			})
			startMs += dur
		}
	}

	return &Result{
		Notes:       notes,
		BPMEstimate: 120.0,
		DurationMs:  startMs,
		Diagnostics: "mock engine generated notes",
	}, nil
}

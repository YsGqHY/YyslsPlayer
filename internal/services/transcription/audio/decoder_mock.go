//go:build completion

package audio

import (
	"context"
	"fmt"
	"os"
)

// MockProber 是用于测试的模拟音频探测器。
type MockProber struct {
	Result *ProbeResult
	Err    error
}

// Probe 返回预设的探测结果。
func (m *MockProber) Probe(path string) (*ProbeResult, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	if m.Result != nil {
		return m.Result, nil
	}
	// 默认返回有效结果
	return &ProbeResult{
		Format:        "wav (pcm_s16le)",
		DurationMs:    30000,
		SampleRate:    44100,
		Channels:      2,
		Bitrate:       1411200,
		Codec:         "pcm_s16le",
		Container:     "wav",
		FileSizeBytes: 5284000,
	}, nil
}

// Available 始终返回 true。
func (m *MockProber) Available() bool { return true }

// MockDecoder 是用于测试的模拟音频解码器。
type MockDecoder struct {
	Err error
}

// Decode 返回一个假解码结果（创建空文件）。
func (m *MockDecoder) Decode(ctx context.Context, path string, workDir string) (string, int, int, error) {
	if m.Err != nil {
		return "", 0, 0, m.Err
	}
	if workDir == "" {
		workDir = os.TempDir()
	}
	pcmPath := workDir + "/decoded_mock.wav"
	// 创建一个最小 WAV 文件头（44 字节），方便后续引擎读取
	header := make([]byte, 44)
	if err := os.WriteFile(pcmPath, header, 0644); err != nil {
		return "", 0, 0, fmt.Errorf("mock decode: %w", err)
	}
	return pcmPath, 22050, 1, nil
}

// Available 始终返回 true。
func (m *MockDecoder) Available() bool { return true }

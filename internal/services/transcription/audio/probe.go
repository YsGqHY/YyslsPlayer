//go:build completion

// Package audio 提供音频格式探测、FFmpeg 解码和临时文件管理。
package audio

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"YyslsPlayer/internal/utils/logx"
	"YyslsPlayer/internal/utils/procx"
)

// ProbeResult 音频探测结果。
type ProbeResult struct {
	Format       string `json:"format"`
	DurationMs   int64  `json:"durationMs"`
	SampleRate   int     `json:"sampleRate"`
	Channels     int     `json:"channels"`
	Bitrate      int64  `json:"bitrate"`
	Codec        string `json:"codec"`
	Container    string `json:"container"`
	FileSizeBytes int64 `json:"fileSizeBytes"`
}

// Prober 音频格式探测接口。
type Prober interface {
	Probe(path string) (*ProbeResult, error)
	Available() bool
}

// FFmpegProber 使用 FFmpeg 进行音频探测。
type FFmpegProber struct {
	ffmpegPath string
	available  bool
}

// NewFFmpegProber 创建 FFmpeg 探测器。
func NewFFmpegProber() *FFmpegProber {
	p := &FFmpegProber{}
	p.ffmpegPath, p.available = p.findFFmpeg()
	return p
}

// Available 返回 FFmpeg 是否可用（每次调用重新检查文件系统，支持运行时安装后即时生效）。
func (p *FFmpegProber) Available() bool {
	p.ffmpegPath, p.available = p.findFFmpeg()
	return p.available
}

// Probe 探测音频文件格式信息。
func (p *FFmpegProber) Probe(path string) (*ProbeResult, error) {
	if !p.available {
		return nil, fmt.Errorf("asset.ffmpeg_missing: FFmpeg not found")
	}

	fi, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("audio.file_not_found: %w", err)
	}

	output, err := procx.RunCapture(context.Background(), procx.Spec{
		Name: p.ffmpegPath,
		Args: []string{"-v", "quiet", "-print_format", "json", "-show_format", "-show_streams", path},
	})
	if err != nil {
		// 保留 captured output 作为诊断信息，避免只展示无意义的退出码
		stderr := strings.TrimSpace(string(output))
		if stderr == "" {
			stderr = "(no output)"
		}
		logx.For("audio").Error("ffprobe probe failed",
			"binary", p.ffmpegPath,
			"path", path,
			"error", err,
			"output", stderr)
		return nil, fmt.Errorf("audio.probe_failed: %w (output: %s)", err, stderr)
	}

	var probeData ffprobeOutput
	if err := json.Unmarshal(output, &probeData); err != nil {
		return nil, fmt.Errorf("audio.probe_failed: invalid probe output: %w", err)
	}

	result := &ProbeResult{
		FileSizeBytes: fi.Size(),
		Container:     probeData.Format.FormatName,
		Bitrate:       parseBitrate(probeData.Format.BitRate),
		DurationMs:    int64(parseFloat(probeData.Format.Duration) * 1000),
	}

	for _, stream := range probeData.Streams {
		if stream.CodecType == "audio" {
			result.Codec = stream.CodecName
			result.SampleRate = stream.SampleRate
			result.Channels = stream.Channels
			result.Format = fmt.Sprintf("%s (%s)", probeData.Format.FormatName, stream.CodecName)
			break
		}
	}

	return result, nil
}

func (p *FFmpegProber) findFFmpeg() (string, bool) {
	// Probe 使用 ffprobe 命令（-print_format/-show_format/-show_streams 均为 ffprobe 专有参数），
	// 因此优先搜索 ffprobe；fallback 到 ffmpeg 仅供紧急情况（ffmpeg 无法执行 ffprobe 命令）。
	builtin := []string{
		"assets/transcription/ffmpeg/ffprobe.exe",
		"assets/transcription/ffmpeg/ffmpeg.exe",
	}
	for _, path := range builtin {
		if _, err := os.Stat(path); err == nil {
			logx.For("audio").Info("using builtin ffprobe", "path", path)
			return path, true
		}
	}
	if path, err := exec.LookPath("ffprobe"); err == nil {
		return path, true
	}
	if path, err := exec.LookPath("ffmpeg"); err == nil {
		logx.For("audio").Warn("ffprobe not found, falling back to ffmpeg (ffprobe flags may not work)", "path", path)
		return path, true
	}

	logx.For("audio").Warn("FFmpeg/FFprobe not found")
	return "", false
}

type ffprobeOutput struct {
	Streams []ffprobeStream `json:"streams"`
	Format  ffprobeFormat   `json:"format"`
}

type ffprobeStream struct {
	CodecName  string `json:"codec_name"`
	CodecType  string `json:"codec_type"`
	SampleRate int    `json:"sample_rate,string"`
	Channels   int    `json:"channels"`
}

type ffprobeFormat struct {
	FormatName string `json:"format_name"`
	Duration   string `json:"duration"`
	BitRate    string `json:"bit_rate"`
}

func parseBitrate(s string) int64 {
	var result float64
	fmt.Sscanf(strings.TrimSpace(s), "%f", &result)
	return int64(result)
}

func parseFloat(s string) float64 {
	var result float64
	fmt.Sscanf(strings.TrimSpace(s), "%f", &result)
	return result
}

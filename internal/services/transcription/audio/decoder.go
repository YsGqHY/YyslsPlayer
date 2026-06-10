//go:build completion

package audio

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"YyslsPlayer/internal/utils/logx"
	"YyslsPlayer/internal/utils/procx"
)

// Decoder 音频解码接口。
type Decoder interface {
	Decode(ctx context.Context, path string, workDir string) (pcmPath string, sampleRate int, channels int, err error)
	Available() bool
}

// FFmpegDecoder 使用 FFmpeg 将音频解码为标准 WAV/PCM。
type FFmpegDecoder struct {
	ffmpegPath string
	available  bool
}

// NewFFmpegDecoder 创建 FFmpeg 解码器。
func NewFFmpegDecoder() *FFmpegDecoder {
	d := &FFmpegDecoder{}
	d.ffmpegPath, d.available = d.findFFmpeg()
	return d
}

// Available 返回 FFmpeg 是否可用（每次调用重新检查文件系统，支持运行时安装后即时生效）。
func (d *FFmpegDecoder) Available() bool {
	d.ffmpegPath, d.available = d.findFFmpeg()
	return d.available
}

// Decode 解码音频到临时 WAV 文件。
// 解码目标：22050Hz, mono, float32。
func (d *FFmpegDecoder) Decode(ctx context.Context, path string, workDir string) (string, int, int, error) {
	if !d.available {
		return "", 0, 0, fmt.Errorf("asset.ffmpeg_missing")
	}

	if workDir == "" {
		workDir = filepath.Join(os.TempDir(), "yyslsplayer", "decode")
	}
	if err := os.MkdirAll(workDir, 0755); err != nil {
		return "", 0, 0, fmt.Errorf("audio.decode_failed: %w", err)
	}

	outPath := filepath.Join(workDir, "decoded.wav")

	var stderrBuf string
	err := procx.Run(ctx, procx.Spec{
		Name: d.ffmpegPath,
		Args: []string{
			"-y",
			"-i", path,
			"-ar", "22050",
			"-ac", "1",
			"-sample_fmt", "flt",
			"-f", "wav",
			outPath,
		},
		OnStderr: func(line string) {
			if len(stderrBuf) < 4096 {
				stderrBuf += line + "\n"
			}
		},
	})
	if err != nil {
		diag := strings.TrimSpace(stderrBuf)
		if diag == "" {
			diag = "(no output)"
		}
		logx.For("audio").Error("ffmpeg decode failed", "binary", d.ffmpegPath, "error", err, "stderr", diag)
		return "", 0, 0, fmt.Errorf("audio.decode_failed: %w (output: %s)", err, diag)
	}

	return outPath, 22050, 1, nil
}

func (d *FFmpegDecoder) findFFmpeg() (string, bool) {
	paths := []string{
		"assets/transcription/ffmpeg/ffmpeg.exe",
	}
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path, true
		}
	}
	if path, err := exec.LookPath("ffmpeg"); err == nil {
		return path, true
	}
	return "", false
}

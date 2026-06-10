//go:build completion

package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"YyslsPlayer/internal/utils/logx"
	"YyslsPlayer/internal/utils/procx"
)

// PythonWorkerRequest 是发给 Python worker 的 JSON 请求。
type PythonWorkerRequest struct {
	PCMFilePath string                 `json:"pcmFilePath"`
	Config      map[string]interface{} `json:"config"`
	OutputDir   string                 `json:"outputDir"`
}

// PythonWorkerResponse 是 Python worker 返回的 JSON 响应。
type PythonWorkerResponse struct {
	Success    bool      `json:"success"`
	Notes      []RawNote `json:"notes"`
	BPM        float64   `json:"bpm"`
	DurationMs int64     `json:"durationMs"`
	Error      string    `json:"error,omitempty"`
}

// BasicPitchEngine 通过 Python 子进程调用 Basic Pitch 模型进行转录。
//
// 通信协议：输入/输出通过 JSON 文件交换，避免 stdout 解析不确定性问题。
type BasicPitchEngine struct {
	pythonPath string
	workerPath string
	available  bool
}

// NewBasicPitchEngine 创建 Basic Pitch 引擎。
func NewBasicPitchEngine() *BasicPitchEngine {
	e := &BasicPitchEngine{}
	e.pythonPath, e.workerPath, e.available = e.findWorker()
	return e
}

func (e *BasicPitchEngine) Name() string    { return "basicPitch" }
func (e *BasicPitchEngine) Version() string { return "spotify/basic-pitch" }
func (e *BasicPitchEngine) Available() bool { return e.available }

// Transcribe 调用 Python worker 进行转录。
func (e *BasicPitchEngine) Transcribe(ctx context.Context, pcmPath string, workDir string, config Config) (*Result, error) {
	if !e.available {
		return nil, fmt.Errorf("asset.worker_missing: Python worker not found")
	}

	if workDir == "" {
		return nil, fmt.Errorf("engine.failed: workDir required for basic pitch")
	}

	req := PythonWorkerRequest{
		PCMFilePath: pcmPath,
		Config: map[string]interface{}{
			"minConfidence": config.MinConfidence,
			"minDurationMs": config.MinDurationMs,
			"maxPolyphony":  config.MaxPolyphony,
		},
		OutputDir: workDir,
	}

	reqPath := filepath.Join(workDir, "worker_request.json")
	reqJSON, err := json.MarshalIndent(req, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("engine.failed: %w", err)
	}
	if err := os.WriteFile(reqPath, reqJSON, 0644); err != nil {
		return nil, fmt.Errorf("engine.failed: %w", err)
	}

	respPath := filepath.Join(workDir, "worker_response.json")

	var stderrBuf string
	err = procx.Run(ctx, procx.Spec{
		Name: e.pythonPath,
		Args: []string{e.workerPath, reqPath, respPath},
		Dir:  workDir,
		OnStderr: func(line string) {
			if len(stderrBuf) < 8192 {
				stderrBuf += line + "\n"
			}
		},
	})
	if err != nil {
		logx.For("engine").Error("worker failed", "error", err, "stderr", stderrBuf)
		return nil, fmt.Errorf("engine.failed: worker process error: %w", err)
	}

	respJSON, err := os.ReadFile(respPath)
	if err != nil {
		return nil, fmt.Errorf("engine.failed: cannot read response: %w", err)
	}

	var resp PythonWorkerResponse
	if err := json.Unmarshal(respJSON, &resp); err != nil {
		return nil, fmt.Errorf("engine.failed: invalid response: %w", err)
	}

	if !resp.Success {
		return nil, fmt.Errorf("engine.failed: %s", resp.Error)
	}

	return &Result{
		Notes:       resp.Notes,
		BPMEstimate: resp.BPM,
		DurationMs:  resp.DurationMs,
		Diagnostics: "basic pitch worker completed",
	}, nil
}

func (e *BasicPitchEngine) findWorker() (python, worker string, ok bool) {
	workerDirs := []string{
		"assets/transcription/worker",
	}
	for _, dir := range workerDirs {
		w := filepath.Join(dir, "worker.py")
		if _, err := os.Stat(w); err == nil {
			for _, py := range []string{"python", "python3"} {
				if p, err := exec.LookPath(py); err == nil {
					return p, w, true
				}
			}
		}
	}

	if p, err := exec.LookPath("python"); err == nil {
		w := "assets/transcription/worker/worker.py"
		if _, err := os.Stat(w); err == nil {
			return p, w, true
		}
	}

	return "", "", false
}

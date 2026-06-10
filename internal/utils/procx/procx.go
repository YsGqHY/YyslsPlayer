// Package procx 提供受控子进程管理，替代业务代码直接使用 exec.Cmd。
//
// 铁律：业务代码不得直接 os/exec；子进程调用统一走 procx。
//
// 跨平台保障：
//   - Windows：JobObject + JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE，父进程退出自动清理
//   - Unix：setpgid + kill(-pgid)，信号广播整个进程组
//
// 当前 P0 提供 Run（阻塞运行 + context 取消），后续可按需扩展 StartCtx/Process。
package procx

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"
)

// Spec 定义子进程启动参数。
type Spec struct {
	// Name 子进程可执行文件路径。
	Name string
	// Args 命令行参数（不包含程序名）。
	Args []string
	// Dir 工作目录，空则继承父进程。
	Dir string
	// Env 环境变量，nil 则继承父进程。
	Env []string
	// OnStdout 每行 stdout 回调（可选）。
	OnStdout func(line string)
	// OnStderr 每行 stderr 回调（可选）。
	OnStderr func(line string)
	// KillGracePeriod 取消后等待优雅退出的最长时间，默认 5s。
	KillGracePeriod time.Duration
}

// RunCapture 启动子进程并等待完成，返回捕获的合并 stdout+stderr 输出。
//
// 与 Run 的区别：同时设置 OnStdout/OnStderr 时，回调仍会被调用，
// 但捕获的输出是原始字节流而非行分割。
func RunCapture(ctx context.Context, spec Spec) ([]byte, error) {
	if spec.Name == "" {
		return nil, fmt.Errorf("procx: name is required")
	}
	grace := spec.KillGracePeriod
	if grace <= 0 {
		grace = 5 * time.Second
	}

	cmd := exec.CommandContext(ctx, spec.Name, spec.Args...)
	if spec.Dir != "" {
		cmd.Dir = spec.Dir
	}
	if spec.Env != nil {
		cmd.Env = spec.Env
	}

	setProcessGroup(cmd)

	var outBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &outBuf

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("procx: start %s: %w", spec.Name, err)
	}
	if err := attachProcess(cmd); err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return nil, fmt.Errorf("procx: attach %s: %w", spec.Name, err)
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err := <-done:
		return outBuf.Bytes(), err
	case <-ctx.Done():
		killProcessGroup(cmd)
		select {
		case <-done:
			return outBuf.Bytes(), ctx.Err()
		case <-time.After(grace):
			_ = cmd.Process.Kill()
			<-done
			return outBuf.Bytes(), ctx.Err()
		}
	}
}

// Run 启动子进程并等待其完成。
//
// ctx 取消时先在 KillGracePeriod 内给进程组发 kill 信号，
// 超时后强制终止。返回的 error 包含 ctx.Err() 或子进程退出码。
func Run(ctx context.Context, spec Spec) error {
	if spec.Name == "" {
		return fmt.Errorf("procx: name is required")
	}
	grace := spec.KillGracePeriod
	if grace <= 0 {
		grace = 5 * time.Second
	}

	cmd := exec.CommandContext(ctx, spec.Name, spec.Args...)
	if spec.Dir != "" {
		cmd.Dir = spec.Dir
	}
	if spec.Env != nil {
		cmd.Env = spec.Env
	}

	// 平台特定：确保子进程在独立进程组中运行（Windows JobObject / Unix setpgid）
	setProcessGroup(cmd)

	if spec.OnStdout != nil {
		setupOutput(cmd, spec.OnStdout, spec.OnStderr)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("procx: start %s: %w", spec.Name, err)
	}

	// 平台特定：将子进程加入管理（Windows JobObject / Unix pgid 已在 Start 前设置）
	if err := attachProcess(cmd); err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return fmt.Errorf("procx: attach %s: %w", spec.Name, err)
	}

	// 等待完成或 ctx 取消
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		// 先给进程组发 kill 信号（平台特定）
		killProcessGroup(cmd)
		// 等待 grace 时间
		select {
		case <-done:
			return ctx.Err()
		case <-time.After(grace):
			// 强制 kill
			_ = cmd.Process.Kill()
			<-done
			return ctx.Err()
		}
	}
}

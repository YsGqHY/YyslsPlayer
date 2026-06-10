//go:build !windows

package procx

import (
	"os/exec"
	"syscall"
)

func setProcessGroup(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
}

func attachProcess(cmd *exec.Cmd) error {
	// setpgid 已在 Start 前设置，无需额外操作
	return nil
}

func killProcessGroup(cmd *exec.Cmd) {
	if cmd.Process != nil {
		// 向整个进程组发送 SIGKILL
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
}

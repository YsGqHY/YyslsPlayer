package procx

import (
	"bufio"
	"io"
	"os/exec"
)

// setupOutput 为 cmd 设置 stdout/stderr 逐行回调。
// 若 OnStdout 为 nil 则不收集 stdout；OnStderr 同理。
func setupOutput(cmd *exec.Cmd, onStdout func(string), onStderr func(string)) {
	if onStdout != nil {
		stdout, err := cmd.StdoutPipe()
		if err == nil {
			go func() {
				scanner := bufio.NewScanner(stdout)
				for scanner.Scan() {
					onStdout(scanner.Text())
				}
			}()
		}
	}
	if onStderr != nil {
		stderr, err := cmd.StderrPipe()
		if err == nil {
			go func() {
				scanner := bufio.NewScanner(stderr)
				for scanner.Scan() {
					onStderr(scanner.Text())
				}
			}()
		}
	} else {
		// 无回调时丢弃 stderr
		cmd.Stderr = io.Discard
	}
}

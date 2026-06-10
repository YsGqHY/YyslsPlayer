//go:build windows

package procx

import (
	"os/exec"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	kernel32          = syscall.NewLazyDLL("kernel32.dll")
	procCreateJobObjectW     = kernel32.NewProc("CreateJobObjectW")
	procSetInformationJobObject = kernel32.NewProc("SetInformationJobObject")
	procAssignProcessToJobObject = kernel32.NewProc("AssignProcessToJobObject")
)

const (
	jobObjectLimitKillOnJobClose = 0x2000
	jobObjectExtendedLimitInformation = 9
)

type jobObjectExtendedLimitInfo struct {
	BasicLimitInformation jobObjectBasicLimitInfo
	IoInfo                [16]byte
	ProcessMemoryLimit    uintptr
	JobMemoryLimit        uintptr
	PeakProcessMemoryUsed uintptr
	PeakJobMemoryUsed     uintptr
}

type jobObjectBasicLimitInfo struct {
	PerProcessUserTimeLimit uint64
	PerJobUserTimeLimit     uint64
	LimitFlags              uint32
	MinimumWorkingSetSize   uintptr
	MaximumWorkingSetSize   uintptr
	ActiveProcessLimit      uint32
	Affinity                uintptr
	PriorityClass           uint32
	SchedulingClass         uint32
}

var jobObject syscall.Handle

func init() {
	h, _, _ := procCreateJobObjectW.Call(0, 0)
	if h != 0 {
		jobObject = syscall.Handle(h)
		var info jobObjectExtendedLimitInfo
		info.BasicLimitInformation.LimitFlags = jobObjectLimitKillOnJobClose
		procSetInformationJobObject.Call(
			uintptr(jobObject),
			jobObjectExtendedLimitInformation,
			uintptr(unsafe.Pointer(&info)),
			uintptr(unsafe.Sizeof(info)),
		)
	}
}

func setProcessGroup(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.CreationFlags |= syscall.CREATE_NEW_PROCESS_GROUP
}

func attachProcess(cmd *exec.Cmd) error {
	if jobObject != 0 && cmd.Process != nil {
		ret, _, _ := procAssignProcessToJobObject.Call(
			uintptr(jobObject),
			uintptr(cmd.Process.Pid),
		)
		if ret == 0 {
			// 继续运行，不因 JobObject 分配失败而终止
		}
	}
	return nil
}

func killProcessGroup(cmd *exec.Cmd) {
	if cmd.Process != nil {
		// 向进程组发 Ctrl+Break 事件
		_ = windows.GenerateConsoleCtrlEvent(windows.CTRL_BREAK_EVENT, uint32(cmd.Process.Pid))
	}
}

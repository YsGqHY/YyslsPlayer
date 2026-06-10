//go:build windows

package app

import (
	"errors"
	"fmt"
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	shell32Admin         = windows.NewLazySystemDLL("shell32.dll")
	procShellExecuteW    = shell32Admin.NewProc("ShellExecuteW")
	advapi32Admin        = windows.NewLazySystemDLL("advapi32.dll")
	procOpenProcessToken = advapi32Admin.NewProc("OpenProcessToken")
	procGetTokenInfo     = advapi32Admin.NewProc("GetTokenInformation")
)

const (
	tokenQuery     = 0x0008
	tokenElevation = 20
)

// EnsureAdminRelaunch makes keyboard simulation run from a high-integrity process.
// Low/medium integrity processes cannot reliably hook or inject into elevated games.
func EnsureAdminRelaunch() error {
	elevated, err := isProcessElevated()
	if err != nil {
		return err
	}
	if elevated {
		stripRelaunchFlag()
		return nil
	}
	if hasArgIn(os.Args, relaunchFlag) {
		return fmt.Errorf("application was relaunched for elevation but is still not elevated")
	}
	return relaunchAsAdmin()
}

func isProcessElevated() (bool, error) {
	var token windows.Handle
	ret, _, callErr := procOpenProcessToken.Call(uintptr(windows.CurrentProcess()), uintptr(tokenQuery), uintptr(unsafe.Pointer(&token)))
	if ret == 0 {
		if callErr != nil && callErr != windows.ERROR_SUCCESS {
			return false, callErr
		}
		return false, errors.New("OpenProcessToken returned 0")
	}
	defer windows.CloseHandle(token)

	var elevation uint32
	var returned uint32
	ret, _, callErr = procGetTokenInfo.Call(uintptr(token), uintptr(tokenElevation), uintptr(unsafe.Pointer(&elevation)), unsafe.Sizeof(elevation), uintptr(unsafe.Pointer(&returned)))
	if ret == 0 {
		if callErr != nil && callErr != windows.ERROR_SUCCESS {
			return false, callErr
		}
		return false, errors.New("GetTokenInformation(TokenElevation) returned 0")
	}
	return elevation != 0, nil
}

func relaunchAsAdmin() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	params := elevatedParams()
	verb, _ := windows.UTF16PtrFromString("runas")
	file, _ := windows.UTF16PtrFromString(exe)
	args, _ := windows.UTF16PtrFromString(params)
	cwd, _ := windows.UTF16PtrFromString(currentWorkingDirectory())

	ret, _, callErr := procShellExecuteW.Call(
		0,
		uintptr(unsafe.Pointer(verb)),
		uintptr(unsafe.Pointer(file)),
		uintptr(unsafe.Pointer(args)),
		uintptr(unsafe.Pointer(cwd)),
		1, // SW_SHOWNORMAL
	)
	if ret <= 32 {
		if callErr != nil && callErr != windows.ERROR_SUCCESS {
			return callErr
		}
		return fmt.Errorf("ShellExecuteW(runas) failed: code=%d", ret)
	}
	os.Exit(0)
	return nil
}

func elevatedParams() string {
	return elevatedParamsFromArgs(os.Args)
}

func stripRelaunchFlag() {
	os.Args = stripArg(os.Args, relaunchFlag)
}

func currentWorkingDirectory() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	return cwd
}

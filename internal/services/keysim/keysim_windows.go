//go:build windows

package keysim

import (
	"context"
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	inputKeyboard       uint32 = 1
	keyEventFKeyUp      uint32 = 0x0002
	keyEventFScancode   uint32 = 0x0008
	keyEventFExtended   uint32 = 0x0001
	extendedKeyPrefix          = 0xE000
)

var (
	user32          = windows.NewLazySystemDLL("user32.dll")
	procSendInput   = user32.NewProc("SendInput")
)

type windowsDriver struct{}

type keyboardInput struct {
	VirtualKey uint16
	ScanCode   uint16
	Flags      uint32
	Time       uint32
	ExtraInfo  uintptr
}

type input struct {
	Type uint32
	Ki   keyboardInput
	_    [8]byte
}

func NewDefaultDriver() Driver {
	return NewWindowsDriver()
}

func NewWindowsDriver() Driver {
	return windowsDriver{}
}

func (windowsDriver) Send(_ context.Context, event KeyEvent, opts RunOptions) error {
	if opts.DryRun {
		return nil
	}
	ki, err := keyboardInputFromEvent(event)
	if err != nil {
		return err
	}
	in := input{Type: inputKeyboard, Ki: ki}
	ret, _, callErr := procSendInput.Call(
		uintptr(1),
		uintptr(unsafe.Pointer(&in)),
		unsafe.Sizeof(in),
	)
	if ret != 1 {
		if callErr != windows.ERROR_SUCCESS {
			return fmt.Errorf("%w: %v", ErrSendFailed, callErr)
		}
		return fmt.Errorf("%w: sent=%d", ErrSendFailed, ret)
	}
	return nil
}

func keyboardInputFromEvent(event KeyEvent) (keyboardInput, error) {
	if err := validateKey(event.Key); err != nil {
		return keyboardInput{}, err
	}
	flags := uint32(0)
	if event.Kind == PhysicalUp {
		flags |= keyEventFKeyUp
	}
	if event.Key.ScanCode != 0 {
		scanCode := event.Key.ScanCode
		if scanCode&extendedKeyPrefix == extendedKeyPrefix {
			flags |= keyEventFExtended
			scanCode &= 0x00ff
		}
		return keyboardInput{ScanCode: uint16(scanCode), Flags: flags | keyEventFScancode}, nil
	}
	return keyboardInput{VirtualKey: uint16(event.Key.VirtualKey), Flags: flags}, nil
}

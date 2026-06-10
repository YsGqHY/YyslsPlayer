//go:build windows

package keysim

import (
	"context"

	"golang.org/x/sys/windows"
)

const (
	inputKeyboard     uint32 = 1
	keyEventFKeyUp    uint32 = 0x0002
	keyEventFScancode uint32 = 0x0008
	keyEventFExtended uint32 = 0x0001
	extendedKeyPrefix        = 0xE000 // equals (0xE0 << 8); marks VK_LCONTROL..VK_RMENU and similar extended keys

	// yyslsExtraInfo 设置非零 ExtraInfo，绕过简易的 ExtraInfo==0 SendInput 黑名单。
	yyslsExtraInfo = 0x5959534C // "YYSL" in little-endian ASCII
)

var (
	user32        = windows.NewLazySystemDLL("user32.dll")
	procSendInput = user32.NewProc("SendInput")
)

type unavailableDriver struct{}

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
	if d := NewHookLaunderDriver(); d != nil {
		return d
	}
	return unavailableDriver{}
}

func (unavailableDriver) Send(_ context.Context, _ KeyEvent, opts RunOptions) error {
	if opts.DryRun {
		return nil
	}
	return ErrHookLaunderUnavailable
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
		ki := keyboardInput{
			ScanCode:  uint16(scanCode),
			Flags:     flags | keyEventFScancode,
			ExtraInfo: yyslsExtraInfo,
		}
		if includeVirtualKeyWithScanCode() && event.Key.VirtualKey != 0 {
			ki.VirtualKey = uint16(event.Key.VirtualKey)
		}
		return ki, nil
	}
	return keyboardInput{
		VirtualKey: uint16(event.Key.VirtualKey),
		Flags:      flags,
		ExtraInfo:  yyslsExtraInfo,
	}, nil
}

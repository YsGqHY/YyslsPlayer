//go:build windows

package keysim

import (
	"context"

	"golang.org/x/sys/windows"
)

const (
	inputKeyboard     uint32 = 1
	inputMouse        uint32 = 0
	keyEventFKeyUp    uint32 = 0x0002
	keyEventFScancode uint32 = 0x0008
	keyEventFExtended uint32 = 0x0001
	keyEventFUnicode  uint32 = 0x0004
	extendedKeyPrefix        = 0xE000 // equals (0xE0 << 8); marks VK_LCONTROL..VK_RMENU and similar extended keys

	mouseEventFMove       uint32 = 0x0001
	mouseEventFLeftDown   uint32 = 0x0002
	mouseEventFLeftUp     uint32 = 0x0004
	mouseEventFRightDown  uint32 = 0x0008
	mouseEventFRightUp    uint32 = 0x0010
	mouseEventFMiddleDown uint32 = 0x0020
	mouseEventFMiddleUp   uint32 = 0x0040
	mouseEventFXDown      uint32 = 0x0080
	mouseEventFXUp        uint32 = 0x0100
	mouseEventFWheel      uint32 = 0x0800
	mouseEventFHWheel     uint32 = 0x1000

	xButton1 uint32 = 0x0001
	xButton2 uint32 = 0x0002

	// wheelDelta is one notch of scroll (WHEEL_DELTA), the standard click unit.
	wheelDelta int32 = 120

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

func (unavailableDriver) Send(ctx context.Context, event KeyEvent, opts RunOptions) error {
	if opts.DryRun {
		return nil
	}
	// Mouse events do not depend on the keyboard laundering hook, so they can be
	// dispatched even when the hook launder driver is unavailable.
	if event.Key.IsMouse() {
		return sendMouseEventCtx(ctx, event)
	}
	return ErrHookLaunderUnavailable
}

// MoveMouse moves the cursor by a relative offset. Like other mouse events it
// does not require the keyboard laundering hook, so it works even when the hook
// launder driver is unavailable.
func (unavailableDriver) MoveMouse(_ context.Context, dx, dy int, opts RunOptions) error {
	if opts.DryRun {
		return nil
	}
	return sendMouseMove(dx, dy)
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

// unicodeInput builds a KEYEVENTF_UNICODE keyboard input for a single UTF-16
// code unit. Up sends the key-up half of the synthetic keystroke.
func unicodeInput(codeUnit uint16, up bool) keyboardInput {
	flags := keyEventFUnicode
	if up {
		flags |= keyEventFKeyUp
	}
	return keyboardInput{
		ScanCode:  codeUnit,
		Flags:     flags,
		ExtraInfo: yyslsExtraInfo,
	}
}

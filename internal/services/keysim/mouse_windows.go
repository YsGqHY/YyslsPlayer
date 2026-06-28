//go:build windows

package keysim

import (
	"context"
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

// mouseInput mirrors the Win32 MOUSEINPUT structure.
type mouseInput struct {
	Dx        int32
	Dy        int32
	MouseData uint32
	Flags     uint32
	Time      uint32
	ExtraInfo uintptr
}

// mouseInputUnion mirrors the Win32 INPUT structure when the union holds a
// MOUSEINPUT. Its total size matches the keyboard `input` struct (sizeof INPUT),
// so SendInput receives a consistent element size regardless of input type.
type mouseInputUnion struct {
	Type uint32
	Mi   mouseInput
}

// mouseInputFromEvent builds a MOUSEINPUT for the given mouse button event.
func mouseInputFromEvent(event KeyEvent) (mouseInput, error) {
	if !event.Key.IsMouse() {
		return mouseInput{}, fmt.Errorf("%w: %s", ErrInvalidKey, event.Key.Label)
	}
	up := event.Kind == PhysicalUp
	var flags uint32
	var data uint32
	switch event.Key.VirtualKey {
	case MouseButtonLeft:
		if up {
			flags = mouseEventFLeftUp
		} else {
			flags = mouseEventFLeftDown
		}
	case MouseButtonRight:
		if up {
			flags = mouseEventFRightUp
		} else {
			flags = mouseEventFRightDown
		}
	case MouseButtonMiddle:
		if up {
			flags = mouseEventFMiddleUp
		} else {
			flags = mouseEventFMiddleDown
		}
	case MouseButtonX1:
		data = xButton1
		if up {
			flags = mouseEventFXUp
		} else {
			flags = mouseEventFXDown
		}
	case MouseButtonX2:
		data = xButton2
		if up {
			flags = mouseEventFXUp
		} else {
			flags = mouseEventFXDown
		}
	case MouseWheelUp:
		flags = mouseEventFWheel
		data = wheelData(wheelDelta)
	case MouseWheelDown:
		flags = mouseEventFWheel
		data = wheelData(-wheelDelta)
	case MouseWheelRight:
		flags = mouseEventFHWheel
		data = wheelData(wheelDelta)
	case MouseWheelLeft:
		flags = mouseEventFHWheel
		data = wheelData(-wheelDelta)
	default:
		return mouseInput{}, fmt.Errorf("%w: %s", ErrInvalidKey, event.Key.Label)
	}
	return mouseInput{
		MouseData: data,
		Flags:     flags,
		ExtraInfo: yyslsExtraInfo,
	}, nil
}

// sendMouseEvent dispatches a mouse button event via SendInput. Mouse events do
// not pass through the keyboard WH_KEYBOARD_LL laundering hook; they carry the
// yyslsExtraInfo marker for parity with keyboard injection.
func sendMouseEvent(event KeyEvent) error {
	mi, err := mouseInputFromEvent(event)
	if err != nil {
		return err
	}
	in := mouseInputUnion{Type: inputMouse, Mi: mi}
	ret, _, callErr := procSendInput.Call(
		uintptr(1),
		uintptr(unsafe.Pointer(&in)),
		unsafe.Sizeof(in),
	)
	if ret != 1 {
		if callErr != nil && callErr != windows.ERROR_SUCCESS {
			return fmt.Errorf("%w: %v", ErrSendFailed, callErr)
		}
		return fmt.Errorf("%w: sent=%d", ErrSendFailed, ret)
	}
	return nil
}

// sendMouseEventCtx is the context-aware entry used by drivers.
func sendMouseEventCtx(_ context.Context, event KeyEvent) error {
	return sendMouseEvent(event)
}

// sendMouseMove dispatches a relative cursor move via SendInput using
// MOUSEEVENTF_MOVE. Offsets are signed pixel deltas; the move carries the
// yyslsExtraInfo marker for parity with other injected mouse events.
func sendMouseMove(dx, dy int) error {
	mi := mouseInput{
		Dx:        int32(dx),
		Dy:        int32(dy),
		Flags:     mouseEventFMove,
		ExtraInfo: yyslsExtraInfo,
	}
	in := mouseInputUnion{Type: inputMouse, Mi: mi}
	ret, _, callErr := procSendInput.Call(
		uintptr(1),
		uintptr(unsafe.Pointer(&in)),
		unsafe.Sizeof(in),
	)
	if ret != 1 {
		if callErr != nil && callErr != windows.ERROR_SUCCESS {
			return fmt.Errorf("%w: %v", ErrSendFailed, callErr)
		}
		return fmt.Errorf("%w: sent=%d", ErrSendFailed, ret)
	}
	return nil
}

// wheelData packs a signed wheel delta into the unsigned MOUSEINPUT.mouseData
// field. Windows reinterprets the low 32 bits as a signed value.
func wheelData(delta int32) uint32 {
	return uint32(delta)
}

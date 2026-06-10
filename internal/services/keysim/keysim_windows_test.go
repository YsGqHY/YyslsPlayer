//go:build windows

package keysim

import "testing"

func TestKeyboardInputFromEventPrefersScanCode(t *testing.T) {
	ki, err := keyboardInputFromEvent(KeyEvent{Kind: PhysicalDown, Key: keyA()})
	if err != nil {
		t.Fatalf("keyboardInputFromEvent failed: %v", err)
	}
	if ki.ScanCode != 30 || ki.VirtualKey != 65 || ki.Flags != keyEventFScancode {
		t.Fatalf("keyboard input = %+v, want ScanCode=30 VirtualKey=65 Flags=%d", ki, keyEventFScancode)
	}
}

func TestKeyboardInputFromEventAddsKeyUpFlag(t *testing.T) {
	ki, err := keyboardInputFromEvent(KeyEvent{Kind: PhysicalUp, Key: keyA()})
	if err != nil {
		t.Fatalf("keyboardInputFromEvent failed: %v", err)
	}
	wantFlags := keyEventFScancode | keyEventFKeyUp
	if ki.ScanCode != 30 || ki.Flags != wantFlags {
		t.Fatalf("keyboard input = %+v, want flags=%d", ki, wantFlags)
	}
}

func TestKeyboardInputFromEventFallsBackToVirtualKey(t *testing.T) {
	ki, err := keyboardInputFromEvent(KeyEvent{Kind: PhysicalDown, Key: Key{Label: "VK", VirtualKey: 65}})
	if err != nil {
		t.Fatalf("keyboardInputFromEvent failed: %v", err)
	}
	if ki.VirtualKey != 65 || ki.ScanCode != 0 || ki.Flags != 0 {
		t.Fatalf("keyboard input = %+v", ki)
	}
}

func TestKeyboardInputFromEventHandlesExtendedScanCode(t *testing.T) {
	ki, err := keyboardInputFromEvent(KeyEvent{Kind: PhysicalDown, Key: Key{Label: "RightCtrl", VirtualKey: 17, ScanCode: extendedKeyPrefix | 0x1D}})
	if err != nil {
		t.Fatalf("keyboardInputFromEvent failed: %v", err)
	}
	wantFlags := keyEventFScancode | keyEventFExtended
	if ki.ScanCode != 0x1D || ki.Flags != wantFlags {
		t.Fatalf("keyboard input = %+v, want flags=%d", ki, wantFlags)
	}
}

func TestKeyboardInputSetsExtraInfo(t *testing.T) {
	ki, err := keyboardInputFromEvent(KeyEvent{Kind: PhysicalDown, Key: keyA()})
	if err != nil {
		t.Fatalf("keyboardInputFromEvent failed: %v", err)
	}
	if ki.ExtraInfo != yyslsExtraInfo {
		t.Fatalf("ExtraInfo = %d, want %d", ki.ExtraInfo, yyslsExtraInfo)
	}
}

func TestKeyboardInputFillsBothVKAndScanCode(t *testing.T) {
	ki, err := keyboardInputFromEvent(KeyEvent{
		Kind: PhysicalDown,
		Key:  Key{Label: "Dual", VirtualKey: 68, ScanCode: 32},
	})
	if err != nil {
		t.Fatalf("keyboardInputFromEvent failed: %v", err)
	}
	if ki.ScanCode != 32 || ki.VirtualKey != 68 || ki.Flags != keyEventFScancode {
		t.Fatalf("keyboard input = %+v, want ScanCode=32 VirtualKey=68", ki)
	}
}

func TestKeyboardInputOnlyScanCodeZeroVK(t *testing.T) {
	ki, err := keyboardInputFromEvent(KeyEvent{
		Kind: PhysicalDown,
		Key:  Key{Label: "ScanOnly", ScanCode: 44},
	})
	if err != nil {
		t.Fatalf("keyboardInputFromEvent failed: %v", err)
	}
	if ki.ScanCode != 44 || ki.VirtualKey != 0 || ki.Flags != keyEventFScancode {
		t.Fatalf("keyboard input = %+v, want ScanCode=44 VirtualKey=0", ki)
	}
}

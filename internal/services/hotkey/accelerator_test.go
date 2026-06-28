package hotkey

import (
	"errors"
	"testing"
)

func TestParseAccelerator(t *testing.T) {
	tests := []struct {
		name      string
		raw       string
		wantText  string
		wantMods  int
		wantVKStr string // 主键名
		wantErr   bool
	}{
		{name: "ctrl alt backspace", raw: "Ctrl+Alt+Backspace", wantText: "Ctrl+Alt+Backspace", wantMods: ModControl | ModAlt, wantVKStr: "Backspace"},
		{name: "function key", raw: "F9", wantText: "F9", wantMods: 0, wantVKStr: "F9"},
		{name: "lowercase + aliases", raw: "control+option+p", wantText: "Ctrl+Alt+P", wantMods: ModControl | ModAlt, wantVKStr: "P"},
		{name: "reorders to canonical", raw: "Alt+Ctrl+S", wantText: "Ctrl+Alt+S", wantMods: ModControl | ModAlt, wantVKStr: "S"},
		{name: "win alias", raw: "Cmd+Space", wantText: "Win+Space", wantMods: ModWin, wantVKStr: "Space"},
		{name: "whitespace tolerant", raw: " Ctrl + Shift + Up ", wantText: "Ctrl+Shift+Up", wantMods: ModControl | ModShift, wantVKStr: "Up"},
		{name: "empty", raw: "", wantErr: true},
		{name: "modifier only", raw: "Ctrl+Alt", wantErr: true},
		{name: "unknown key", raw: "Ctrl+Banana", wantErr: true},
		{name: "two main keys", raw: "Ctrl+A+B", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			acc, err := parseAccelerator(tt.raw)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil (acc=%+v)", acc)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if acc.text != tt.wantText {
				t.Errorf("text = %q, want %q", acc.text, tt.wantText)
			}
			if acc.modifiers != tt.wantMods {
				t.Errorf("modifiers = %d, want %d", acc.modifiers, tt.wantMods)
			}
			if acc.keyName != tt.wantVKStr {
				t.Errorf("keyName = %q, want %q", acc.keyName, tt.wantVKStr)
			}
		})
	}
}

func TestParseAcceleratorSupportsExtendedKeyboardKeys(t *testing.T) {
	tests := []struct {
		raw     string
		want    string
		wantKey string
	}{
		{raw: "Ctrl+Semicolon", want: "Ctrl+Semicolon", wantKey: "Semicolon"},
		{raw: "Ctrl+Equal", want: "Ctrl+Equal", wantKey: "Equal"},
		{raw: "Ctrl+Comma", want: "Ctrl+Comma", wantKey: "Comma"},
		{raw: "Ctrl+Minus", want: "Ctrl+Minus", wantKey: "Minus"},
		{raw: "Ctrl+Period", want: "Ctrl+Period", wantKey: "Period"},
		{raw: "Ctrl+Slash", want: "Ctrl+Slash", wantKey: "Slash"},
		{raw: "Ctrl+Backquote", want: "Ctrl+Backquote", wantKey: "Backquote"},
		{raw: "Ctrl+BracketLeft", want: "Ctrl+BracketLeft", wantKey: "BracketLeft"},
		{raw: "Ctrl+Backslash", want: "Ctrl+Backslash", wantKey: "Backslash"},
		{raw: "Ctrl+BracketRight", want: "Ctrl+BracketRight", wantKey: "BracketRight"},
		{raw: "Ctrl+Quote", want: "Ctrl+Quote", wantKey: "Quote"},
		{raw: "Ctrl+IntlBackslash", want: "Ctrl+IntlBackslash", wantKey: "IntlBackslash"},
		{raw: "Ctrl+Numpad0", want: "Ctrl+Numpad0", wantKey: "Numpad0"},
		{raw: "Ctrl+Numpad9", want: "Ctrl+Numpad9", wantKey: "Numpad9"},
		{raw: "Ctrl+NumpadMultiply", want: "Ctrl+NumpadMultiply", wantKey: "NumpadMultiply"},
		{raw: "Ctrl+NumpadAdd", want: "Ctrl+NumpadAdd", wantKey: "NumpadAdd"},
		{raw: "Ctrl+NumpadSubtract", want: "Ctrl+NumpadSubtract", wantKey: "NumpadSubtract"},
		{raw: "Ctrl+NumpadDecimal", want: "Ctrl+NumpadDecimal", wantKey: "NumpadDecimal"},
		{raw: "Ctrl+NumpadDivide", want: "Ctrl+NumpadDivide", wantKey: "NumpadDivide"},
		{raw: "Ctrl+Cancel", want: "Ctrl+Cancel", wantKey: "Cancel"},
		{raw: "Ctrl+CapsLock", want: "Ctrl+CapsLock", wantKey: "CapsLock"},
		{raw: "Ctrl+KanaMode", want: "Ctrl+KanaMode", wantKey: "KanaMode"},
		{raw: "Ctrl+Convert", want: "Ctrl+Convert", wantKey: "Convert"},
		{raw: "Ctrl+NonConvert", want: "Ctrl+NonConvert", wantKey: "NonConvert"},
		{raw: "Ctrl+NumLock", want: "Ctrl+NumLock", wantKey: "NumLock"},
		{raw: "Ctrl+ScrollLock", want: "Ctrl+ScrollLock", wantKey: "ScrollLock"},
		{raw: "Ctrl+Select", want: "Ctrl+Select", wantKey: "Select"},
		{raw: "Ctrl+Print", want: "Ctrl+Print", wantKey: "Print"},
		{raw: "Ctrl+Execute", want: "Ctrl+Execute", wantKey: "Execute"},
		{raw: "Ctrl+PrintScreen", want: "Ctrl+PrintScreen", wantKey: "PrintScreen"},
		{raw: "Ctrl+Pause", want: "Ctrl+Pause", wantKey: "Pause"},
		{raw: "Ctrl+ShiftLeft", want: "Ctrl+ShiftLeft", wantKey: "ShiftLeft"},
		{raw: "Alt+ControlRight", want: "Alt+ControlRight", wantKey: "ControlRight"},
		{raw: "Ctrl+MetaRight", want: "Ctrl+MetaRight", wantKey: "MetaRight"},
		{raw: "Ctrl+Apps", want: "Ctrl+Apps", wantKey: "Apps"},
		{raw: "Ctrl+BrowserBack", want: "Ctrl+BrowserBack", wantKey: "BrowserBack"},
		{raw: "Ctrl+VolumeUp", want: "Ctrl+VolumeUp", wantKey: "VolumeUp"},
		{raw: "Ctrl+MediaPlayPause", want: "Ctrl+MediaPlayPause", wantKey: "MediaPlayPause"},
		{raw: "Ctrl+LaunchMail", want: "Ctrl+LaunchMail", wantKey: "LaunchMail"},
		{raw: "Ctrl+Num1", want: "Ctrl+Numpad1", wantKey: "Numpad1"},
		{raw: "Ctrl+PrtSc", want: "Ctrl+PrintScreen", wantKey: "PrintScreen"},
		{raw: "Ctrl+ContextMenu", want: "Ctrl+Apps", wantKey: "Apps"},
	}
	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			acc, err := parseAccelerator(tt.raw)
			if err != nil {
				t.Fatalf("parseAccelerator(%q): %v", tt.raw, err)
			}
			if acc.text != tt.want {
				t.Fatalf("text = %q, want %q", acc.text, tt.want)
			}
			if acc.keyName != tt.wantKey {
				t.Fatalf("keyName = %q, want %q", acc.keyName, tt.wantKey)
			}
		})
	}
}

func TestAllNamedMainKeysNormalizeWithSafeModifier(t *testing.T) {
	for _, def := range mainKeys {
		raw := "Ctrl+" + def.name
		acc, err := normalizeAccelerator(raw)
		if err != nil {
			t.Fatalf("normalizeAccelerator(%q): %v", raw, err)
		}
		if acc.keyName != def.name {
			t.Fatalf("%q keyName = %q, want %q", raw, acc.keyName, def.name)
		}
		if acc.vk != def.vk {
			t.Fatalf("%q vk = %#x, want %#x", raw, acc.vk, def.vk)
		}
	}
}

func TestParseAcceleratorSupportsVirtualKeyFallback(t *testing.T) {
	tests := []struct {
		raw  string
		want string
		vk   int
	}{
		{raw: "Ctrl+VK0xC1", want: "Ctrl+VK0xC1", vk: 0xC1},
		{raw: "Alt+VK193", want: "Alt+VK0xC1", vk: 0xC1},
		{raw: "Win+VK0xe7", want: "Win+VK0xE7", vk: 0xE7},
	}
	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			acc, err := normalizeAccelerator(tt.raw)
			if err != nil {
				t.Fatalf("normalizeAccelerator(%q): %v", tt.raw, err)
			}
			if acc.text != tt.want {
				t.Fatalf("text = %q, want %q", acc.text, tt.want)
			}
			if acc.vk != tt.vk {
				t.Fatalf("vk = %#x, want %#x", acc.vk, tt.vk)
			}
		})
	}

	invalid := []string{"Ctrl+VK0", "Ctrl+VK0x00", "Ctrl+VK0x01", "Ctrl+VK0x02", "Ctrl+VK0x04", "Ctrl+VK0x07", "Ctrl+VK0xFF", "Ctrl+VKXYZ"}
	for _, raw := range invalid {
		if _, err := parseAccelerator(raw); err == nil {
			t.Fatalf("parseAccelerator(%q) should fail", raw)
		}
	}
}

func TestIsSafeForGlobal(t *testing.T) {
	safe := []string{
		"Ctrl+Alt+Backspace",
		"F1", "F9", "F12", "F24",
		"Ctrl+S",
		"Alt+Space",
		"Win+Up",
	}
	unsafe := []string{
		"Space",          // 裸普通键
		"A",              // 裸字母
		"5",              // 裸数字
		"Enter",          // 裸回车
		"Tab",            // 裸 Tab
		"Semicolon",      // 裸标点键
		"Numpad1",        // 裸数字小键盘
		"VolumeUp",       // 裸媒体键仍可能影响系统行为
		"MediaPlayPause", // 裸媒体键仍可能影响系统行为
		"Shift+A",        // 仅 Shift 不算安全（仍是普通输入）
		"Shift+Space",
		"Shift+NumpadAdd",
	}
	for _, raw := range safe {
		acc, err := parseAccelerator(raw)
		if err != nil {
			t.Fatalf("parse %q: %v", raw, err)
		}
		if !isSafeForGlobal(acc) {
			t.Errorf("%q should be safe for global", raw)
		}
	}
	for _, raw := range unsafe {
		acc, err := parseAccelerator(raw)
		if err != nil {
			t.Fatalf("parse %q: %v", raw, err)
		}
		if isSafeForGlobal(acc) {
			t.Errorf("%q should be unsafe for global", raw)
		}
	}
}

func TestNormalizeAccelerator(t *testing.T) {
	if _, err := normalizeAccelerator("Space"); !errors.Is(err, ErrUnsafeAccelerator) {
		t.Errorf("normalize bare Space: want ErrUnsafeAccelerator, got %v", err)
	}
	if _, err := normalizeAccelerator("Ctrl+Banana"); !errors.Is(err, ErrInvalidAccelerator) {
		t.Errorf("normalize invalid: want ErrInvalidAccelerator, got %v", err)
	}
	acc, err := normalizeAccelerator("ctrl+alt+f9")
	if err != nil {
		t.Fatalf("normalize valid: %v", err)
	}
	if acc.text != "Ctrl+Alt+F9" {
		t.Errorf("text = %q, want Ctrl+Alt+F9", acc.text)
	}
}

func TestFindConflicts(t *testing.T) {
	bindings := map[string]string{
		ActionPlayPause:        "F9",
		ActionStop:             "F9", // 与 playPause 冲突
		ActionPreviewToggle:    "F11",
		ActionEmergencyRelease: "Ctrl+Alt+Backspace",
	}
	enabled := map[string]bool{
		ActionPlayPause:        true,
		ActionStop:             true,
		ActionPreviewToggle:    true,
		ActionEmergencyRelease: true,
	}
	groups := findConflicts(bindings, enabled)
	if len(groups) != 1 {
		t.Fatalf("want 1 conflict group, got %d (%v)", len(groups), groups)
	}
	if len(groups[0]) != 2 {
		t.Errorf("want 2 actions in conflict, got %v", groups[0])
	}

	// 停用其中一个后冲突消失。
	enabled[ActionStop] = false
	if groups := findConflicts(bindings, enabled); len(groups) != 0 {
		t.Errorf("disabling one binding should clear conflict, got %v", groups)
	}

	bindings = map[string]string{
		"first":  "Ctrl+Backslash",
		"second": "Ctrl+IntlYen",
	}
	enabled = map[string]bool{"first": true, "second": true}
	if groups := findConflicts(bindings, enabled); len(groups) != 1 || len(groups[0]) != 2 {
		t.Errorf("same VK with different names should conflict, got %v", groups)
	}
}

func TestDefaultBindingsAreSafe(t *testing.T) {
	for _, d := range defaultBindings {
		acc, err := parseAccelerator(d.Accelerator)
		if err != nil {
			t.Fatalf("default %s=%q parse failed: %v", d.ActionID, d.Accelerator, err)
		}
		if !isSafeForGlobal(acc) {
			t.Errorf("default %s=%q is unsafe for global hotkey", d.ActionID, d.Accelerator)
		}
	}
}

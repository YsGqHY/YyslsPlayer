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

func TestIsSafeForGlobal(t *testing.T) {
	safe := []string{
		"Ctrl+Alt+Backspace",
		"F1", "F9", "F12", "F24",
		"Ctrl+S",
		"Alt+Space",
		"Win+Up",
	}
	unsafe := []string{
		"Space",   // 裸普通键
		"A",       // 裸字母
		"5",       // 裸数字
		"Enter",   // 裸回车
		"Tab",     // 裸 Tab
		"Shift+A", // 仅 Shift 不算安全（仍是普通输入）
		"Shift+Space",
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

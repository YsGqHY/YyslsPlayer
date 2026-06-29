//go:build completion

package macro

import (
	"testing"

	"gopkg.in/yaml.v3"

	"YyslsPlayer/internal/services/keysim"
	"YyslsPlayer/internal/storage"
)

func sampleDetail() storage.MacroDetail {
	return storage.MacroDetail{
		Profile: storage.MacroProfile{
			ID:                 7,
			Name:               "Combo",
			Description:        "demo",
			TriggerAccelerator: "Ctrl+Alt+1",
			Enabled:            true,
			RepeatMode:         RepeatModeCount,
			RepeatCount:        5,
			RepeatIntervalMs:   30,
			InterruptPolicy:    InterruptIgnore,
		},
		Steps: []storage.MacroStep{
			{Kind: StepKeyTap, KeyLabel: "A", VirtualKey: 65, ScanCode: 30, DeviceKind: DeviceKeyboard, ModifierKeysJSON: "[]", PayloadJSON: "{}", DurationMs: 40},
			{Kind: StepDelay, ModifierKeysJSON: "[]", PayloadJSON: "{}", WaitMs: 100},
			{Kind: StepChordTap, KeyLabel: "C", VirtualKey: 67, ScanCode: 46, DeviceKind: DeviceKeyboard, ModifierKeysJSON: `[{"label":"Ctrl","virtualKey":17,"scanCode":29}]`, PayloadJSON: "{}", DurationMs: 40},
			{Kind: StepText, DeviceKind: DeviceKeyboard, ModifierKeysJSON: "[]", PayloadJSON: `{"text":"hello","perCharDelayMs":5}`, DurationMs: 40},
			{Kind: StepMouseMove, DeviceKind: DeviceMouse, ModifierKeysJSON: "[]", PayloadJSON: `{"dx":100,"dy":-20,"jitter":3}`, DurationMs: 10},
		},
	}
}

func TestPortableRoundTrip(t *testing.T) {
	doc, err := toPortableDoc([]storage.MacroDetail{sampleDetail()})
	if err != nil {
		t.Fatalf("toPortableDoc: %v", err)
	}
	data, err := yaml.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var back portableDoc
	if err := yaml.Unmarshal(data, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	reqs, err := fromPortableDoc(back)
	if err != nil {
		t.Fatalf("fromPortableDoc: %v", err)
	}
	if len(reqs) != 1 {
		t.Fatalf("expected 1 macro, got %d", len(reqs))
	}
	req := reqs[0]
	if req.ID != 0 {
		t.Errorf("imported macro should have fresh ID, got %d", req.ID)
	}
	if req.Enabled {
		t.Errorf("imported macro should be disabled")
	}
	if req.Name != "Combo" || req.RepeatMode != RepeatModeCount || req.RepeatCount != 5 || req.RepeatIntervalMs != 30 {
		t.Errorf("profile fields not preserved: %+v", req)
	}
	if req.TriggerAccelerator != "Ctrl+Alt+1" {
		t.Errorf("trigger not preserved: %q", req.TriggerAccelerator)
	}
	if len(req.Steps) != 5 {
		t.Fatalf("expected 5 steps, got %d", len(req.Steps))
	}
	// Steps pass the same validation used by SaveMacro.
	rows, err := normalizeAndValidateSteps(req.Steps)
	if err != nil {
		t.Fatalf("normalizeAndValidateSteps: %v", err)
	}
	chord := rows[2]
	mods, err := keysim.DecodeModifiers(chord.ModifierKeysJSON)
	if err != nil || len(mods) != 1 || mods[0].VirtualKey != 17 {
		t.Errorf("chord modifiers not preserved: %v %v", chord.ModifierKeysJSON, err)
	}
	textPayload, err := DecodeTextPayload(rows[3].PayloadJSON)
	if err != nil || textPayload.Text != "hello" || textPayload.PerCharDelayMs != 5 {
		t.Errorf("text payload not preserved: %+v %v", textPayload, err)
	}
	movePayload, err := DecodeMousePayload(rows[4].PayloadJSON)
	if err != nil || movePayload.Dx != 100 || movePayload.Dy != -20 || movePayload.Jitter != 3 {
		t.Errorf("move payload not preserved: %+v %v", movePayload, err)
	}
}

func TestFromPortableDocRejectsBadVersion(t *testing.T) {
	_, err := fromPortableDoc(portableDoc{Version: 99, Macros: []portableMacro{{Name: "x"}}})
	if err == nil {
		t.Fatal("expected error for unsupported version")
	}
}

func TestFromPortableDocRejectsEmpty(t *testing.T) {
	_, err := fromPortableDoc(portableDoc{Version: currentPortableVersion, Macros: nil})
	if err == nil {
		t.Fatal("expected error for empty macros")
	}
}

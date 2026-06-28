//go:build completion

package macro

import (
	"testing"

	"YyslsPlayer/internal/services/keysim"
	"YyslsPlayer/internal/storage"
)

func TestPlanStepsLinearTimeline(t *testing.T) {
	steps := []storage.MacroStep{
		{Kind: StepKeyTap, KeyLabel: "A", VirtualKey: 65, ScanCode: 30, DurationMs: 40, ModifierKeysJSON: "[]", PayloadJSON: "{}"},
		{Kind: StepDelay, WaitMs: 100, ModifierKeysJSON: "[]", PayloadJSON: "{}"},
		{Kind: StepKeyTap, KeyLabel: "B", VirtualKey: 66, ScanCode: 48, DurationMs: 30, ModifierKeysJSON: "[]", PayloadJSON: "{}"},
	}
	planned, err := planSteps(steps)
	if err != nil {
		t.Fatalf("planSteps failed: %v", err)
	}
	if len(planned.steps) != 3 {
		t.Fatalf("planned steps = %d, want 3", len(planned.steps))
	}
	if planned.durationMs != 170 {
		t.Fatalf("duration = %d, want 170", planned.durationMs)
	}
	if len(planned.actions) != 4 {
		t.Fatalf("actions = %d, want 4", len(planned.actions))
	}
	wantTimes := []int64{0, 40, 140, 170}
	for i, want := range wantTimes {
		if planned.actions[i].TimeMs != want {
			t.Fatalf("action %d time = %d, want %d", i, planned.actions[i].TimeMs, want)
		}
	}
	if planned.steps[1].endMs != 140 || len(planned.steps[1].actionIndexes) != 0 {
		t.Fatalf("delay step = %+v, want endMs=140 and no actions", planned.steps[1])
	}
}

func TestNormalizeStepRejectsMixedDelay(t *testing.T) {
	_, err := normalizeStep(MacroStepDTO{Kind: StepDelay, KeyLabel: "A", VirtualKey: 65, ScanCode: 30, WaitMs: 100})
	if err == nil {
		t.Fatal("delay with key should be rejected")
	}
}

func TestPlanMouseScrollEmitsSinglePressNoDuration(t *testing.T) {
	steps := []storage.MacroStep{
		{Kind: StepMouseScroll, KeyLabel: "Wheel Up", VirtualKey: 6, DeviceKind: DeviceMouse, ModifierKeysJSON: "[]", PayloadJSON: "{}"},
		{Kind: StepMouseScroll, KeyLabel: "Wheel Down", VirtualKey: 7, DeviceKind: DeviceMouse, ModifierKeysJSON: "[]", PayloadJSON: "{}"},
	}
	planned, err := planSteps(steps)
	if err != nil {
		t.Fatalf("planSteps failed: %v", err)
	}
	// Each scroll is a single one-shot press action and consumes no timeline.
	if len(planned.actions) != 2 {
		t.Fatalf("actions = %d, want 2", len(planned.actions))
	}
	if planned.durationMs != 0 {
		t.Fatalf("duration = %d, want 0", planned.durationMs)
	}
	for i, a := range planned.actions {
		if a.TimeMs != 0 {
			t.Fatalf("action %d time = %d, want 0", i, a.TimeMs)
		}
		if a.Key.Kind != "mouse" {
			t.Fatalf("action %d kind = %q, want mouse", i, a.Key.Kind)
		}
	}
}

func TestNormalizeMouseScrollValidatesWheelButton(t *testing.T) {
	// A valid wheel direction passes.
	if _, err := normalizeStep(MacroStepDTO{Kind: StepMouseScroll, KeyLabel: "Wheel Up", VirtualKey: 6}); err != nil {
		t.Fatalf("valid wheel step rejected: %v", err)
	}
	// A click button id is not a valid scroll direction.
	if _, err := normalizeStep(MacroStepDTO{Kind: StepMouseScroll, KeyLabel: "Mouse Left", VirtualKey: 1}); err == nil {
		t.Fatal("click button should be rejected for mouseScroll")
	}
	// Scroll cannot carry a duration.
	if _, err := normalizeStep(MacroStepDTO{Kind: StepMouseScroll, KeyLabel: "Wheel Up", VirtualKey: 6, DurationMs: 40}); err == nil {
		t.Fatal("mouseScroll with duration should be rejected")
	}
}

func TestPlanTextEmitsSingleTextAction(t *testing.T) {
	steps := []storage.MacroStep{
		{Kind: StepText, DeviceKind: DeviceKeyboard, ModifierKeysJSON: "[]", PayloadJSON: `{"text":"hi 世界","perCharDelayMs":20}`, DurationMs: 100},
	}
	planned, err := planSteps(steps)
	if err != nil {
		t.Fatalf("planSteps failed: %v", err)
	}
	if len(planned.actions) != 1 {
		t.Fatalf("actions = %d, want 1", len(planned.actions))
	}
	a := planned.actions[0]
	if a.Action != keysim.ActionText {
		t.Fatalf("action kind = %q, want text", a.Action)
	}
	if a.Text != "hi 世界" {
		t.Fatalf("text = %q, want %q", a.Text, "hi 世界")
	}
	if a.TextDelayMs != 20 {
		t.Fatalf("text delay = %d, want 20", a.TextDelayMs)
	}
	// DurationMs from the step row drives the timeline cursor.
	if planned.durationMs != 100 {
		t.Fatalf("duration = %d, want 100", planned.durationMs)
	}
}

func TestNormalizeTextStep(t *testing.T) {
	// Valid text normalizes and derives a duration.
	row, err := normalizeStep(MacroStepDTO{Kind: StepText, PayloadJSON: `{"text":"abc","perCharDelayMs":10}`})
	if err != nil {
		t.Fatalf("valid text step rejected: %v", err)
	}
	if row.DeviceKind != DeviceKeyboard {
		t.Fatalf("deviceKind = %q, want keyboard", row.DeviceKind)
	}
	if row.DurationMs <= 0 {
		t.Fatalf("derived duration = %d, want > 0", row.DurationMs)
	}
	// Empty text is rejected.
	if _, err := normalizeStep(MacroStepDTO{Kind: StepText, PayloadJSON: `{"text":""}`}); err == nil {
		t.Fatal("empty text should be rejected")
	}
	// Text cannot carry a key.
	if _, err := normalizeStep(MacroStepDTO{Kind: StepText, KeyLabel: "A", VirtualKey: 65, PayloadJSON: `{"text":"x"}`}); err == nil {
		t.Fatal("text with key should be rejected")
	}
}

func TestPlanMouseMoveEmitsSingleMoveAction(t *testing.T) {
	steps := []storage.MacroStep{
		{Kind: StepMouseMove, DeviceKind: DeviceMouse, ModifierKeysJSON: "[]", PayloadJSON: `{"dx":120,"dy":-40}`, DurationMs: mouseMoveDurationMs},
	}
	planned, err := planSteps(steps)
	if err != nil {
		t.Fatalf("planSteps failed: %v", err)
	}
	if len(planned.actions) != 1 {
		t.Fatalf("actions = %d, want 1", len(planned.actions))
	}
	a := planned.actions[0]
	if a.Action != keysim.ActionMouseMove {
		t.Fatalf("action kind = %q, want mouseMove", a.Action)
	}
	if a.Dx != 120 || a.Dy != -40 {
		t.Fatalf("offset = (%d,%d), want (120,-40)", a.Dx, a.Dy)
	}
	if planned.durationMs != mouseMoveDurationMs {
		t.Fatalf("duration = %d, want %d", planned.durationMs, mouseMoveDurationMs)
	}
}

func TestNormalizeMouseMoveStep(t *testing.T) {
	// Valid offset normalizes, forces mouse device, and sets a fixed duration.
	row, err := normalizeStep(MacroStepDTO{Kind: StepMouseMove, PayloadJSON: `{"dx":10,"dy":20}`})
	if err != nil {
		t.Fatalf("valid move step rejected: %v", err)
	}
	if row.DeviceKind != DeviceMouse {
		t.Fatalf("deviceKind = %q, want mouse", row.DeviceKind)
	}
	if row.DurationMs != mouseMoveDurationMs {
		t.Fatalf("duration = %d, want %d", row.DurationMs, mouseMoveDurationMs)
	}
	// Move cannot carry a key/button.
	if _, err := normalizeStep(MacroStepDTO{Kind: StepMouseMove, KeyLabel: "Mouse Left", VirtualKey: 1, PayloadJSON: `{"dx":1,"dy":1}`}); err == nil {
		t.Fatal("move with key should be rejected")
	}
	// Move cannot carry a waitMs.
	if _, err := normalizeStep(MacroStepDTO{Kind: StepMouseMove, WaitMs: 50, PayloadJSON: `{"dx":1,"dy":1}`}); err == nil {
		t.Fatal("move with waitMs should be rejected")
	}
	// Offset out of range is rejected.
	if _, err := normalizeStep(MacroStepDTO{Kind: StepMouseMove, PayloadJSON: `{"dx":999999,"dy":0}`}); err == nil {
		t.Fatal("out-of-range move offset should be rejected")
	}
	// A user-set travel duration is preserved (controls move speed).
	row, err = normalizeStep(MacroStepDTO{Kind: StepMouseMove, DurationMs: 200, PayloadJSON: `{"dx":100,"dy":0,"jitter":8}`})
	if err != nil {
		t.Fatalf("move step with duration/jitter rejected: %v", err)
	}
	if row.DurationMs != 200 {
		t.Fatalf("duration = %d, want 200 (user value preserved)", row.DurationMs)
	}
	// Jitter out of range is rejected.
	if _, err := normalizeStep(MacroStepDTO{Kind: StepMouseMove, PayloadJSON: `{"dx":1,"dy":1,"jitter":99999}`}); err == nil {
		t.Fatal("out-of-range jitter should be rejected")
	}
}

func TestPlanMouseMoveSpreadsAcrossDuration(t *testing.T) {
	// A long-duration move expands into multiple timed sub-moves whose deltas
	// sum to the exact net offset, with the last sub-move landing at duration.
	steps := []storage.MacroStep{
		{Kind: StepMouseMove, DeviceKind: DeviceMouse, ModifierKeysJSON: "[]", PayloadJSON: `{"dx":100,"dy":-50}`, DurationMs: 100},
	}
	planned, err := planSteps(steps)
	if err != nil {
		t.Fatalf("planSteps failed: %v", err)
	}
	if len(planned.actions) != 10 {
		t.Fatalf("actions = %d, want 10 (100ms / 10ms segments)", len(planned.actions))
	}
	sumX, sumY := 0, 0
	for _, a := range planned.actions {
		if a.Action != keysim.ActionMouseMove {
			t.Fatalf("action kind = %q, want mouseMove", a.Action)
		}
		sumX += a.Dx
		sumY += a.Dy
	}
	if sumX != 100 || sumY != -50 {
		t.Fatalf("sub-move sum = (%d,%d), want (100,-50)", sumX, sumY)
	}
	last := planned.actions[len(planned.actions)-1]
	if last.TimeMs != 100 {
		t.Fatalf("last sub-move time = %d, want 100", last.TimeMs)
	}
	if planned.durationMs != 100 {
		t.Fatalf("duration = %d, want 100", planned.durationMs)
	}
}

func TestPlanMoveSegmentsJitterStillLandsExact(t *testing.T) {
	// With jitter on, intermediate points wobble but the accumulated deltas must
	// still sum to the exact net offset (the final point is forced to target).
	payload := MousePayload{Dx: 80, Dy: 40, Jitter: 6}
	seq := 0
	rnd := func(n int) int {
		// Deterministic pseudo-random stand-in so the test is stable.
		seq++
		return seq % n
	}
	segs := planMoveSegments(payload, 80, rnd)
	if len(segs) < 2 {
		t.Fatalf("segments = %d, want multiple for an 80ms move", len(segs))
	}
	sumX, sumY := 0, 0
	for _, s := range segs {
		sumX += s.dx
		sumY += s.dy
	}
	if sumX != 80 || sumY != 40 {
		t.Fatalf("jittered sum = (%d,%d), want (80,40)", sumX, sumY)
	}
}

package keysim

import (
	"context"
	"errors"
	"testing"
)

func TestDryRunExpandsModifierSequence(t *testing.T) {
	svc := New(NewStubDriver())
	actions := []KeyAction{
		action(ActionPress, keyA(), shift()),
		action(ActionRelease, keyA(), shift()),
	}

	result, err := svc.Run(context.Background(), actions, RunOptions{DryRun: true, DryRunLogLimit: 10})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if result.TotalKeyframes != 2 || len(result.Keyframes) != 2 || result.KeyframesTruncated {
		t.Fatalf("keyframes result = %+v", result)
	}
	if result.Keyframes[0].Action != ActionPress || result.Keyframes[1].Action != ActionRelease {
		t.Fatalf("keyframes = %+v", result.Keyframes)
	}
	if result.TotalEvents != 4 || len(result.Events) != 4 || result.Truncated {
		t.Fatalf("result = %+v", result)
	}
	assertEvent(t, result.Events[0], PhysicalDown, "Shift", true)
	assertEvent(t, result.Events[1], PhysicalDown, "A", false)
	assertEvent(t, result.Events[2], PhysicalUp, "A", false)
	assertEvent(t, result.Events[3], PhysicalUp, "Shift", true)
	if got := svc.Snapshot().Pressed; len(got) != 0 {
		t.Fatalf("pressed = %+v, want empty", got)
	}
}

func TestDryRunKeepsSharedModifierPressedUntilLastRelease(t *testing.T) {
	svc := New(NewStubDriver())
	actions := []KeyAction{
		action(ActionPress, keyA(), shift()),
		action(ActionPress, keyB(), shift()),
		action(ActionRelease, keyA(), shift()),
		action(ActionRelease, keyB(), shift()),
	}

	result, err := svc.Run(context.Background(), actions, RunOptions{DryRun: true, DryRunLogLimit: 10})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if result.TotalEvents != 6 || len(result.Events) != 6 {
		t.Fatalf("events = %+v", result.Events)
	}
	assertEvent(t, result.Events[0], PhysicalDown, "Shift", true)
	assertEvent(t, result.Events[1], PhysicalDown, "A", false)
	assertEvent(t, result.Events[2], PhysicalDown, "B", false)
	assertEvent(t, result.Events[3], PhysicalUp, "A", false)
	assertEvent(t, result.Events[4], PhysicalUp, "B", false)
	assertEvent(t, result.Events[5], PhysicalUp, "Shift", true)
}

func TestReleaseAllClearsPressedSetInReverseOrder(t *testing.T) {
	svc := New(NewStubDriver())
	_, err := svc.Apply(context.Background(), action(ActionPress, keyA(), shift()), RunOptions{DryRun: true})
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}
	if got := svc.Snapshot().Pressed; len(got) != 2 {
		t.Fatalf("pressed len = %d, want 2: %+v", len(got), got)
	}

	result, err := svc.ReleaseAll(context.Background(), RunOptions{DryRun: true, DryRunLogLimit: 10})
	if err != nil {
		t.Fatalf("ReleaseAll failed: %v", err)
	}
	if result.ReleasedKeys != 2 || result.TotalEvents != 2 {
		t.Fatalf("result = %+v", result)
	}
	assertEvent(t, result.Events[0], PhysicalUp, "A", false)
	assertEvent(t, result.Events[1], PhysicalUp, "Shift", true)
	if got := svc.Snapshot().Pressed; len(got) != 0 {
		t.Fatalf("pressed = %+v, want empty", got)
	}
}

func TestStubRejectsActualSend(t *testing.T) {
	svc := New(NewStubDriver())
	_, err := svc.Apply(context.Background(), action(ActionPress, keyA()), RunOptions{DryRun: false})
	if !errors.Is(err, ErrUnsupportedPlatform) {
		t.Fatalf("err = %v, want KEYSIM_UNSUPPORTED_PLATFORM", err)
	}
	if got := svc.Snapshot().Pressed; len(got) != 0 {
		t.Fatalf("pressed = %+v, want empty", got)
	}
}

func TestApplyFailureReleasesPressedModifiers(t *testing.T) {
	driver := &fakeDriver{failLabel: "A", failKind: PhysicalDown}
	svc := New(driver)

	result, err := svc.Apply(context.Background(), action(ActionPress, keyA(), shift()), RunOptions{DryRun: false, DryRunLogLimit: 10})
	if !errors.Is(err, errFakeSend) {
		t.Fatalf("err = %v, want fake send error", err)
	}
	if result.RecoveryReleasedKeys != 1 || result.TotalEvents != 2 {
		t.Fatalf("result = %+v", result)
	}
	assertEvent(t, result.Events[0], PhysicalDown, "Shift", true)
	assertEvent(t, result.Events[1], PhysicalUp, "Shift", true)
	if got := svc.Snapshot().Pressed; len(got) != 0 {
		t.Fatalf("pressed = %+v, want empty", got)
	}
}

func TestReleaseAllContinuesAfterSingleReleaseFailure(t *testing.T) {
	driver := &fakeDriver{failLabel: "B", failKind: PhysicalUp}
	svc := New(driver)
	if _, err := svc.Apply(context.Background(), action(ActionPress, keyA()), RunOptions{DryRun: false}); err != nil {
		t.Fatalf("press A failed: %v", err)
	}
	if _, err := svc.Apply(context.Background(), action(ActionPress, keyB()), RunOptions{DryRun: false}); err != nil {
		t.Fatalf("press B failed: %v", err)
	}

	result, err := svc.ReleaseAll(context.Background(), RunOptions{DryRun: false, DryRunLogLimit: 10})
	if !errors.Is(err, ErrReleaseFailed) || !errors.Is(err, errFakeSend) {
		t.Fatalf("err = %v, want release and fake send errors", err)
	}
	if result.ReleasedKeys != 1 || result.TotalEvents != 1 {
		t.Fatalf("result = %+v", result)
	}
	assertEvent(t, result.Events[0], PhysicalUp, "A", false)
	pressed := svc.Snapshot().Pressed
	if len(pressed) != 1 || pressed[0].Key.Label != "B" {
		t.Fatalf("pressed = %+v, want B only", pressed)
	}
}

func TestDecodeModifiers(t *testing.T) {
	mods, err := DecodeModifiers(`[{"label":"Shift","virtualKey":16,"scanCode":42}]`)
	if err != nil {
		t.Fatalf("DecodeModifiers failed: %v", err)
	}
	if len(mods) != 1 || mods[0].Label != "Shift" || mods[0].VirtualKey != 16 || mods[0].ScanCode != 42 {
		t.Fatalf("mods = %+v", mods)
	}
}

func TestDryRunLogLimitTruncatesResult(t *testing.T) {
	svc := New(NewStubDriver())
	result, err := svc.Run(context.Background(), []KeyAction{
		action(ActionPress, keyA()),
		action(ActionRelease, keyA()),
		action(ActionPress, keyB()),
	}, RunOptions{DryRun: true, DryRunLogLimit: 2})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if result.TotalKeyframes != 3 || len(result.Keyframes) != 2 || !result.KeyframesTruncated {
		t.Fatalf("keyframes result = %+v", result)
	}
	if result.TotalEvents != 3 || len(result.Events) != 2 || !result.Truncated {
		t.Fatalf("result = %+v", result)
	}
}

var errFakeSend = errors.New("fake send failed")

type fakeDriver struct {
	failLabel string
	failKind  PhysicalKind
}

func (d *fakeDriver) Send(_ context.Context, event KeyEvent, _ RunOptions) error {
	if event.Key.Label == d.failLabel && event.Kind == d.failKind {
		return errFakeSend
	}
	return nil
}

func action(kind ActionKind, key Key, modifiers ...Key) KeyAction {
	return KeyAction{
		TimeMs:         12,
		Action:         kind,
		Lane:           3,
		SourceNote:     60,
		NormalizedNote: 60,
		Velocity:       90,
		Key:            key,
		Modifiers:      modifiers,
	}
}

func keyA() Key {
	return Key{Label: "A", VirtualKey: 65, ScanCode: 30}
}

func keyB() Key {
	return Key{Label: "B", VirtualKey: 66, ScanCode: 48}
}

func shift() Key {
	return Key{Label: "Shift", VirtualKey: 16, ScanCode: 42}
}

func assertEvent(t *testing.T, event KeyEvent, kind PhysicalKind, label string, modifier bool) {
	t.Helper()
	if event.Kind != kind || event.Key.Label != label || event.Modifier != modifier {
		t.Fatalf("event = %+v, want kind=%s label=%s modifier=%v", event, kind, label, modifier)
	}
}

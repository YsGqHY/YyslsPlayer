package keysim

import (
	"context"
	"testing"
)

func mouseLeft() Key {
	return Key{Label: "MouseLeft", VirtualKey: MouseButtonLeft, Kind: KeyKindMouse}
}

func mouseX1() Key {
	return Key{Label: "MouseX1", VirtualKey: MouseButtonX1, Kind: KeyKindMouse}
}

func TestMouseKeyValidation(t *testing.T) {
	if err := validateKey(mouseLeft()); err != nil {
		t.Fatalf("validateKey(mouseLeft) = %v, want nil", err)
	}
	// Mouse key with an unknown button identifier must be rejected.
	bad := Key{Label: "MouseBad", VirtualKey: 99, Kind: KeyKindMouse}
	if err := validateKey(bad); err == nil {
		t.Fatalf("validateKey(bad mouse) = nil, want error")
	}
	// Mouse key validity does not depend on ScanCode.
	noScan := Key{Label: "MouseRight", VirtualKey: MouseButtonRight, Kind: KeyKindMouse}
	if err := validateKey(noScan); err != nil {
		t.Fatalf("validateKey(mouseRight without scan) = %v, want nil", err)
	}
}

func TestMouseKeyIDNamespaceIsolated(t *testing.T) {
	// A mouse button must not collide with a keyboard key that happens to share
	// the same VirtualKey integer.
	kbd := Key{Label: "Backspace", VirtualKey: MouseButtonLeft, ScanCode: 14}
	if keyID(mouseLeft()) == keyID(kbd) {
		t.Fatalf("mouse keyID collides with keyboard keyID: %s", keyID(mouseLeft()))
	}
	if got := keyID(mouseLeft()); got != "mouse:1" {
		t.Fatalf("keyID(mouseLeft) = %s, want mouse:1", got)
	}
}

func TestMouseDryRunPressReleaseTracksPressedSet(t *testing.T) {
	svc := New(NewStubDriver())
	if _, err := svc.Apply(context.Background(), action(ActionPress, mouseLeft()), RunOptions{DryRun: true}); err != nil {
		t.Fatalf("press mouseLeft failed: %v", err)
	}
	if _, err := svc.Apply(context.Background(), action(ActionPress, mouseX1()), RunOptions{DryRun: true}); err != nil {
		t.Fatalf("press mouseX1 failed: %v", err)
	}
	pressed := svc.Snapshot().Pressed
	if len(pressed) != 2 {
		t.Fatalf("pressed len = %d, want 2: %+v", len(pressed), pressed)
	}

	result, err := svc.ReleaseAll(context.Background(), RunOptions{DryRun: true, DryRunLogLimit: 10})
	if err != nil {
		t.Fatalf("ReleaseAll failed: %v", err)
	}
	if result.ReleasedKeys != 2 || result.TotalEvents != 2 {
		t.Fatalf("result = %+v", result)
	}
	if got := svc.Snapshot().Pressed; len(got) != 0 {
		t.Fatalf("pressed = %+v, want empty", got)
	}
}

func wheelUp() Key {
	return Key{Label: "WheelUp", VirtualKey: MouseWheelUp, Kind: KeyKindMouse}
}

func TestMouseWheelValidation(t *testing.T) {
	for _, btn := range []int{MouseWheelUp, MouseWheelDown, MouseWheelLeft, MouseWheelRight} {
		k := Key{Label: "Wheel", VirtualKey: btn, Kind: KeyKindMouse}
		if err := validateKey(k); err != nil {
			t.Fatalf("validateKey(wheel %d) = %v, want nil", btn, err)
		}
		if !k.IsMouseWheel() {
			t.Fatalf("IsMouseWheel(%d) = false, want true", btn)
		}
	}
	// Click buttons are not wheel directions.
	if mouseLeft().IsMouseWheel() {
		t.Fatal("mouseLeft should not be a wheel button")
	}
}

func TestMouseWheelIsOneShotNotTracked(t *testing.T) {
	svc := New(NewStubDriver())
	// A wheel "press" emits a notch but must not enter the pressed set, and a
	// wheel "release" is a no-op.
	res, err := svc.Apply(context.Background(), action(ActionPress, wheelUp()), RunOptions{DryRun: true, DryRunLogLimit: 10})
	if err != nil {
		t.Fatalf("scroll up failed: %v", err)
	}
	if res.TotalEvents != 1 {
		t.Fatalf("scroll emitted %d events, want 1", res.TotalEvents)
	}
	if got := svc.Snapshot().Pressed; len(got) != 0 {
		t.Fatalf("wheel left pressed entries %+v, want empty", got)
	}
	// ReleaseAll has nothing to release.
	rel, err := svc.ReleaseAll(context.Background(), RunOptions{DryRun: true})
	if err != nil {
		t.Fatalf("ReleaseAll failed: %v", err)
	}
	if rel.ReleasedKeys != 0 {
		t.Fatalf("ReleasedKeys = %d, want 0", rel.ReleasedKeys)
	}
}

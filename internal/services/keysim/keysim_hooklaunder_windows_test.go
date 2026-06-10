//go:build windows

package keysim

import (
	"context"
	"testing"
)

func TestNewHookLaunderDriverCreatesDriver(t *testing.T) {
	d := NewHookLaunderDriver()
	if d == nil {
		t.Skip("hook launder driver not available on this system — skipping integration test")
	}
	// Verify Send works in dry-run mode (no actual key injection).
	err := d.Send(context.Background(), KeyEvent{
		Kind: PhysicalDown,
		Key:  keyA(),
	}, RunOptions{DryRun: true})
	if err != nil {
		t.Fatalf("dry-run Send failed: %v", err)
	}
}

func TestHookLaunderDriverSendRequiresValidKey(t *testing.T) {
	d := NewHookLaunderDriver()
	if d == nil {
		t.Skip("hook launder driver not available")
	}
	err := d.Send(context.Background(), KeyEvent{
		Kind: PhysicalDown,
		Key:  Key{Label: "Invalid"}, // zero VK and scan code
	}, RunOptions{DryRun: false})
	if err == nil {
		t.Fatal("expected error for invalid key")
	}
}

func TestHookLaunderDriverSendUsesExtraInfo(t *testing.T) {
	// Verify that keyboardInputFromEvent still produces the correct ExtraInfo
	// for the hook to identify our events.
	ki, err := keyboardInputFromEvent(KeyEvent{
		Kind: PhysicalDown,
		Key:  keyA(),
	})
	if err != nil {
		t.Fatalf("keyboardInputFromEvent failed: %v", err)
	}
	if ki.ExtraInfo != yyslsExtraInfo {
		t.Fatalf("ExtraInfo = %d, want %d (hook launder depends on this)", ki.ExtraInfo, yyslsExtraInfo)
	}
}

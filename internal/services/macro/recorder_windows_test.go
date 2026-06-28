//go:build windows && completion

package macro

import "testing"

// TestAcceptKeySuppressesAutoRepeat verifies the recorder records one keyDown
// per physical press (ignoring OS auto-repeat) and only records a keyUp that
// pairs with a prior down.
func TestAcceptKeySuppressesAutoRepeat(t *testing.T) {
	r := &windowsRecorder{downKeys: make(map[uint32]struct{})}

	if !r.acceptKey(65, false) {
		t.Fatal("first keyDown should be accepted")
	}
	if r.acceptKey(65, false) {
		t.Fatal("auto-repeat keyDown should be suppressed")
	}
	if r.acceptKey(65, false) {
		t.Fatal("subsequent auto-repeat keyDown should be suppressed")
	}
	if !r.acceptKey(65, true) {
		t.Fatal("keyUp matching a prior down should be accepted")
	}
	if r.acceptKey(65, true) {
		t.Fatal("stray keyUp without a prior down should be suppressed")
	}
	// A fresh press after release is a new physical press.
	if !r.acceptKey(65, false) {
		t.Fatal("keyDown after release should be accepted again")
	}
}

// TestAcceptKeyIndependentKeys verifies held keys are tracked independently.
func TestAcceptKeyIndependentKeys(t *testing.T) {
	r := &windowsRecorder{downKeys: make(map[uint32]struct{})}

	if !r.acceptKey(16, false) { // Shift down
		t.Fatal("Shift down should be accepted")
	}
	if !r.acceptKey(65, false) { // A down while Shift held
		t.Fatal("A down should be accepted while Shift held")
	}
	if r.acceptKey(16, false) {
		t.Fatal("Shift auto-repeat should be suppressed")
	}
	if !r.acceptKey(65, true) {
		t.Fatal("A up should be accepted")
	}
	if !r.acceptKey(16, true) {
		t.Fatal("Shift up should be accepted")
	}
}

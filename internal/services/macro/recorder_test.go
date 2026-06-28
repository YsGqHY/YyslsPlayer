//go:build completion

package macro

import "testing"

// TestOnCapturedInsertsDelaysWhenEnabled verifies that, with delay capture on,
// the recorder inserts a delay step reflecting the gap between events.
func TestOnCapturedInsertsDelaysWhenEnabled(t *testing.T) {
	s := &Service{}
	sess := &recordSession{captureDelays: true}

	s.onCaptured(sess, capturedEvent{vk: 65, scan: 30, atMs: 0})
	s.onCaptured(sess, capturedEvent{vk: 65, scan: 30, keyUp: true, atMs: 40})
	s.onCaptured(sess, capturedEvent{vk: 66, scan: 48, atMs: 200})

	// Expected: keyDown, delay(40), keyUp, delay(160), keyDown.
	wantKinds := []string{StepKeyDown, StepDelay, StepKeyUp, StepDelay, StepKeyDown}
	if len(sess.steps) != len(wantKinds) {
		t.Fatalf("steps = %d, want %d (%+v)", len(sess.steps), len(wantKinds), sess.steps)
	}
	for i, want := range wantKinds {
		if sess.steps[i].Kind != want {
			t.Fatalf("step %d kind = %q, want %q", i, sess.steps[i].Kind, want)
		}
	}
	if sess.steps[1].WaitMs != 40 {
		t.Fatalf("first delay = %d, want 40", sess.steps[1].WaitMs)
	}
	if sess.steps[3].WaitMs != 160 {
		t.Fatalf("second delay = %d, want 160", sess.steps[3].WaitMs)
	}
}

// TestOnCapturedSkipsDelaysWhenDisabled verifies that, with delay capture off,
// events are recorded back-to-back with no interleaved delay steps.
func TestOnCapturedSkipsDelaysWhenDisabled(t *testing.T) {
	s := &Service{}
	sess := &recordSession{captureDelays: false}

	s.onCaptured(sess, capturedEvent{vk: 65, scan: 30, atMs: 0})
	s.onCaptured(sess, capturedEvent{vk: 65, scan: 30, keyUp: true, atMs: 40})
	s.onCaptured(sess, capturedEvent{vk: 66, scan: 48, atMs: 200})

	wantKinds := []string{StepKeyDown, StepKeyUp, StepKeyDown}
	if len(sess.steps) != len(wantKinds) {
		t.Fatalf("steps = %d, want %d (%+v)", len(sess.steps), len(wantKinds), sess.steps)
	}
	for i, want := range wantKinds {
		if sess.steps[i].Kind != want {
			t.Fatalf("step %d kind = %q, want %q", i, sess.steps[i].Kind, want)
		}
		if sess.steps[i].Kind == StepDelay {
			t.Fatalf("step %d should not be a delay", i)
		}
	}
}

// TestOnCapturedDropsMovesWhenDisabled verifies that, with move capture off,
// cursor-move events are discarded while key events still record. This mirrors
// G HUB's "record mouse movement" toggle keeping keyboard macros clean.
func TestOnCapturedDropsMovesWhenDisabled(t *testing.T) {
	s := &Service{}
	sess := &recordSession{captureDelays: false, captureMoves: false}

	s.onCaptured(sess, capturedEvent{vk: 65, scan: 30, atMs: 0})
	s.onCaptured(sess, capturedEvent{move: true, mouse: true, dx: 12, dy: -7, atMs: 10})
	s.onCaptured(sess, capturedEvent{move: true, mouse: true, dx: 5, dy: 3, atMs: 20})
	s.onCaptured(sess, capturedEvent{vk: 65, scan: 30, keyUp: true, atMs: 30})

	wantKinds := []string{StepKeyDown, StepKeyUp}
	if len(sess.steps) != len(wantKinds) {
		t.Fatalf("steps = %d, want %d (%+v)", len(sess.steps), len(wantKinds), sess.steps)
	}
	for i, want := range wantKinds {
		if sess.steps[i].Kind != want {
			t.Fatalf("step %d kind = %q, want %q", i, sess.steps[i].Kind, want)
		}
	}
	if sess.pendingMove {
		t.Fatalf("pendingMove should stay false when move capture is disabled")
	}
}

// TestOnCapturedCoalescesMovesWhenEnabled verifies that, with move capture on,
// consecutive cursor moves collapse into a single move step carrying the net
// offset, flushed before the following key event.
func TestOnCapturedCoalescesMovesWhenEnabled(t *testing.T) {
	s := &Service{}
	sess := &recordSession{captureDelays: false, captureMoves: true}

	s.onCaptured(sess, capturedEvent{move: true, mouse: true, dx: 12, dy: -7, atMs: 10})
	s.onCaptured(sess, capturedEvent{move: true, mouse: true, dx: 5, dy: 3, atMs: 20})
	s.onCaptured(sess, capturedEvent{vk: 65, scan: 30, atMs: 30})

	wantKinds := []string{StepMouseMove, StepKeyDown}
	if len(sess.steps) != len(wantKinds) {
		t.Fatalf("steps = %d, want %d (%+v)", len(sess.steps), len(wantKinds), sess.steps)
	}
	for i, want := range wantKinds {
		if sess.steps[i].Kind != want {
			t.Fatalf("step %d kind = %q, want %q", i, sess.steps[i].Kind, want)
		}
	}
	payload, err := DecodeMousePayload(sess.steps[0].PayloadJSON)
	if err != nil {
		t.Fatalf("decode move payload: %v", err)
	}
	if payload.Dx != 17 || payload.Dy != -4 {
		t.Fatalf("coalesced offset = (%d,%d), want (17,-4)", payload.Dx, payload.Dy)
	}
}

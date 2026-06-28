//go:build completion

package macro

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	// maxMouseMoveDelta bounds a single relative cursor move on either axis so a
	// macro cannot fling the pointer an absurd distance in one step.
	maxMouseMoveDelta = 10_000
	// mouseMoveDurationMs is the default/minimum timeline cost of a move step so
	// the planner cursor advances. Larger user-set durations spread the motion
	// across many sub-moves to control travel speed.
	mouseMoveDurationMs = 10
	// maxMouseJitter bounds the per-axis random wobble applied while a move is in
	// flight, so "unstable" movement cannot teleport the cursor off course.
	maxMouseJitter = 500
	// moveSegmentMs is the nominal wall-clock spacing between intermediate
	// sub-moves; a longer move duration yields more segments (smoother + slower).
	moveSegmentMs = 10
	// maxMoveSegments caps how many sub-moves one step expands into, bounding the
	// timeline action count for very long durations.
	maxMoveSegments = 300
)

// MousePayload is the decoded JSON shape stored in MacroStep.PayloadJSON for
// StepMouseMove. It carries a relative cursor offset in pixels plus an optional
// per-axis random jitter applied while the move is in flight. Reusing the
// payload column means zero DB migration, mirroring the text-block step.
type MousePayload struct {
	Dx int `json:"dx"`
	Dy int `json:"dy"`
	// Jitter is the maximum random per-axis pixel wobble added to intermediate
	// move points to make the motion unstable. Zero means a straight path.
	Jitter int `json:"jitter,omitempty"`
}

// DecodeMousePayload parses and validates a mouse-move payload. An empty or "{}"
// payload decodes to a zero offset, which the planner treats as a no-op move.
func DecodeMousePayload(raw string) (MousePayload, error) {
	var payload MousePayload
	trimmed := strings.TrimSpace(raw)
	if trimmed != "" && trimmed != "{}" {
		if err := json.Unmarshal([]byte(trimmed), &payload); err != nil {
			return MousePayload{}, fmt.Errorf("invalid mouse payload: %w", err)
		}
	}
	if payload.Dx < -maxMouseMoveDelta || payload.Dx > maxMouseMoveDelta ||
		payload.Dy < -maxMouseMoveDelta || payload.Dy > maxMouseMoveDelta {
		return MousePayload{}, fmt.Errorf("mouse move offset out of range")
	}
	if payload.Jitter < 0 || payload.Jitter > maxMouseJitter {
		return MousePayload{}, fmt.Errorf("mouse move jitter out of range")
	}
	return payload, nil
}

// EncodeMousePayload serializes a mouse-move payload back to JSON for storage.
func EncodeMousePayload(payload MousePayload) (string, error) {
	out, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("encode mouse payload: %w", err)
	}
	return string(out), nil
}

// moveSegment is one timed relative sub-move that the planner schedules along
// the macro timeline. offsetMs is relative to the move step's start cursor.
type moveSegment struct {
	offsetMs int64
	dx       int
	dy       int
}

// planMoveSegments splits a single move (net dx/dy over durationMs) into a
// sequence of timed relative sub-moves. Intermediate points get a random
// per-axis wobble of up to payload.Jitter px so the path looks unstable, while
// the segment deltas always sum to the exact net offset so the cursor still
// lands where the user asked. A zero/short duration collapses to one instant
// sub-move (the legacy behaviour). rnd supplies the jitter; pass nil for a
// deterministic straight path (used by tests / when jitter is zero).
func planMoveSegments(payload MousePayload, durationMs int64, rnd func(n int) int) []moveSegment {
	if payload.Dx == 0 && payload.Dy == 0 {
		return nil
	}
	segments := 1
	if durationMs > moveSegmentMs {
		segments = int(durationMs / moveSegmentMs)
	}
	if segments < 1 {
		segments = 1
	}
	if segments > maxMoveSegments {
		segments = maxMoveSegments
	}
	if rnd == nil || payload.Jitter <= 0 {
		// Straight, evenly spaced path: cumulative target along each axis is the
		// rounded linear interpolation, and each delta is the difference so the
		// sum is exact.
		out := make([]moveSegment, 0, segments)
		prevX, prevY := 0, 0
		for i := 1; i <= segments; i++ {
			tx := payload.Dx * i / segments
			ty := payload.Dy * i / segments
			out = append(out, moveSegment{
				offsetMs: int64(i) * durationMs / int64(segments),
				dx:       tx - prevX,
				dy:       ty - prevY,
			})
			prevX, prevY = tx, ty
		}
		return out
	}
	// Jittered path: each intermediate target is the linear point plus a random
	// per-axis wobble; the final target is forced to the exact net offset so the
	// accumulated deltas still sum correctly.
	out := make([]moveSegment, 0, segments)
	prevX, prevY := 0, 0
	for i := 1; i <= segments; i++ {
		var tx, ty int
		if i == segments {
			tx, ty = payload.Dx, payload.Dy
		} else {
			tx = payload.Dx*i/segments + jitterDelta(rnd, payload.Jitter)
			ty = payload.Dy*i/segments + jitterDelta(rnd, payload.Jitter)
		}
		out = append(out, moveSegment{
			offsetMs: int64(i) * durationMs / int64(segments),
			dx:       tx - prevX,
			dy:       ty - prevY,
		})
		prevX, prevY = tx, ty
	}
	return out
}

// jitterDelta returns a random integer in [-jitter, jitter] using rnd, which
// yields a non-negative value in [0, n).
func jitterDelta(rnd func(n int) int, jitter int) int {
	if jitter <= 0 {
		return 0
	}
	return rnd(2*jitter+1) - jitter
}

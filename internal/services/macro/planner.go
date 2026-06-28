//go:build completion

package macro

import (
	"fmt"
	"math/rand"
	"sort"

	"YyslsPlayer/internal/services/keysim"
	"YyslsPlayer/internal/storage"
)

// randIntn returns a pseudo-random int in [0, n) for movement jitter. A
// non-positive n yields 0. Cursor wobble does not need cryptographic randomness.
func randIntn(n int) int {
	if n <= 0 {
		return 0
	}
	return rand.Intn(n)
}

type plannedMacro struct {
	actions    []keysim.KeyAction
	steps      []plannedStep
	durationMs int64
}

type plannedStep struct {
	step          storage.MacroStep
	actionIndexes []int
	endMs         int64
}

func planSteps(steps []storage.MacroStep) (plannedMacro, error) {
	if len(steps) == 0 {
		return plannedMacro{}, ErrMacroNoSteps
	}
	sorted := append([]storage.MacroStep(nil), steps...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].OrderIndex == sorted[j].OrderIndex {
			return sorted[i].ID < sorted[j].ID
		}
		return sorted[i].OrderIndex < sorted[j].OrderIndex
	})

	planned := plannedMacro{steps: make([]plannedStep, 0, len(sorted))}
	cursor := int64(0)
	for idx, step := range sorted {
		start := len(planned.actions)
		switch step.Kind {
		case StepDelay:
			cursor += step.WaitMs
		case StepKeyTap:
			planned.actions = append(planned.actions,
				keyAction(cursor, keysim.ActionPress, step, nil),
				keyAction(cursor+step.DurationMs, keysim.ActionRelease, step, nil),
			)
			cursor += step.DurationMs
		case StepKeyDown:
			planned.actions = append(planned.actions, keyAction(cursor, keysim.ActionPress, step, nil))
		case StepKeyUp:
			planned.actions = append(planned.actions, keyAction(cursor, keysim.ActionRelease, step, nil))
		case StepChordTap:
			mods, err := keysim.DecodeModifiers(step.ModifierKeysJSON)
			if err != nil {
				return plannedMacro{}, fmt.Errorf("%w: step %d modifiers: %w", ErrMacroInvalid, idx+1, err)
			}
			planned.actions = append(planned.actions,
				keyAction(cursor, keysim.ActionPress, step, mods),
				keyAction(cursor+step.DurationMs, keysim.ActionRelease, step, mods),
			)
			cursor += step.DurationMs
		case StepMouseTap:
			planned.actions = append(planned.actions,
				keyAction(cursor, keysim.ActionPress, step, nil),
				keyAction(cursor+step.DurationMs, keysim.ActionRelease, step, nil),
			)
			cursor += step.DurationMs
		case StepMouseDown:
			planned.actions = append(planned.actions, keyAction(cursor, keysim.ActionPress, step, nil))
		case StepMouseUp:
			planned.actions = append(planned.actions, keyAction(cursor, keysim.ActionRelease, step, nil))
		case StepMouseScroll:
			// Scroll is a one-shot notch: keysim emits on press and ignores the
			// release, so a single press action is sufficient per wheel step.
			planned.actions = append(planned.actions, keyAction(cursor, keysim.ActionPress, step, nil))
		case StepMouseMove:
			payload, err := DecodeMousePayload(step.PayloadJSON)
			if err != nil {
				return plannedMacro{}, fmt.Errorf("%w: step %d mouse move: %w", ErrMacroInvalid, idx+1, err)
			}
			// Spread the move across its duration as timed sub-moves so the
			// cursor travels at a controlled speed; jitter wobbles intermediate
			// points for an unstable path. Segments always sum to the exact net
			// offset, and the planner cursor advances by the full duration.
			segments := planMoveSegments(payload, step.DurationMs, randIntn)
			for _, seg := range segments {
				planned.actions = append(planned.actions, moveAction(cursor+seg.offsetMs, seg.dx, seg.dy))
			}
			cursor += step.DurationMs
		case StepText:
			payload, err := DecodeTextPayload(step.PayloadJSON)
			if err != nil {
				return plannedMacro{}, fmt.Errorf("%w: step %d text: %w", ErrMacroInvalid, idx+1, err)
			}
			planned.actions = append(planned.actions, textAction(cursor, payload))
			cursor += step.DurationMs
		default:
			return plannedMacro{}, fmt.Errorf("%w: unknown step kind %q", ErrMacroInvalid, step.Kind)
		}
		pstep := plannedStep{step: step, endMs: cursor}
		for i := start; i < len(planned.actions); i++ {
			pstep.actionIndexes = append(pstep.actionIndexes, i)
		}
		planned.steps = append(planned.steps, pstep)
	}
	planned.durationMs = cursor
	return planned, nil
}

func keyAction(timeMs int64, action keysim.ActionKind, step storage.MacroStep, modifiers []keysim.Key) keysim.KeyAction {
	return keysim.KeyAction{
		TimeMs:    timeMs,
		Action:    action,
		Lane:      -1,
		Velocity:  100,
		Key:       keyFromStep(step),
		Modifiers: modifiers,
	}
}

// textAction builds a Unicode text-injection keysim action from a decoded payload.
func textAction(timeMs int64, payload TextPayload) keysim.KeyAction {
	return keysim.KeyAction{
		TimeMs:      timeMs,
		Action:      keysim.ActionText,
		Lane:        -1,
		Velocity:    100,
		Text:        payload.Text,
		TextDelayMs: payload.PerCharDelayMs,
	}
}

// moveAction builds a relative cursor-move keysim action from a decoded offset.
func moveAction(timeMs int64, dx, dy int) keysim.KeyAction {
	return keysim.KeyAction{
		TimeMs:   timeMs,
		Action:   keysim.ActionMouseMove,
		Lane:     -1,
		Velocity: 100,
		Dx:       dx,
		Dy:       dy,
	}
}

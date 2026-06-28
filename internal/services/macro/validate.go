//go:build completion

package macro

import (
	"fmt"
	"strings"

	"YyslsPlayer/internal/services/keysim"
	"YyslsPlayer/internal/storage"
)

const (
	defaultKeyTapMs   = 40
	maxStepCount      = 500
	maxDurationMs     = 60_000
	maxRepeatCount    = 1000
	defaultRepeatWait = 30
)

func normalizeProfile(row *storage.MacroProfile) {
	row.Name = strings.TrimSpace(row.Name)
	if row.Name == "" {
		row.Name = "New Macro"
	}
	row.Description = strings.TrimSpace(row.Description)
	switch row.RepeatMode {
	case RepeatModeOnce, RepeatModeCount, RepeatModeHold, RepeatModeToggle:
		// valid
	default:
		row.RepeatMode = RepeatModeOnce
	}
	if row.RepeatCount <= 0 {
		row.RepeatCount = 1
	}
	if row.RepeatCount > maxRepeatCount {
		row.RepeatCount = maxRepeatCount
	}
	if row.RepeatMode != RepeatModeCount {
		// RepeatCount only applies to the fixed-count mode.
		row.RepeatCount = 1
	}
	if row.RepeatIntervalMs < 0 {
		row.RepeatIntervalMs = 0
	}
	if row.RepeatIntervalMs > maxDurationMs {
		row.RepeatIntervalMs = maxDurationMs
	}
	// RepeatInterval only meaningful for repeating modes (count/hold/toggle).
	if row.RepeatMode == RepeatModeOnce {
		row.RepeatIntervalMs = 0
	}
	if row.InterruptPolicy == "" {
		row.InterruptPolicy = InterruptIgnore
	}
	if row.InterruptPolicy != InterruptIgnore && row.InterruptPolicy != InterruptRestart {
		row.InterruptPolicy = InterruptIgnore
	}
}

func normalizeAndValidateSteps(steps []MacroStepDTO) ([]storage.MacroStep, error) {
	if len(steps) > maxStepCount {
		return nil, fmt.Errorf("%w: too many steps", ErrMacroInvalid)
	}
	out := make([]storage.MacroStep, 0, len(steps))
	for i, step := range steps {
		row, err := normalizeStep(step)
		if err != nil {
			return nil, fmt.Errorf("%w: step %d: %w", ErrMacroInvalid, i+1, err)
		}
		row.OrderIndex = i
		out = append(out, row)
	}
	return out, nil
}

func normalizeStep(step MacroStepDTO) (storage.MacroStep, error) {
	row := storage.MacroStep{
		Kind:             strings.TrimSpace(step.Kind),
		KeyLabel:         strings.TrimSpace(step.KeyLabel),
		VirtualKey:       step.VirtualKey,
		ScanCode:         step.ScanCode,
		DeviceKind:       strings.TrimSpace(step.DeviceKind),
		ModifierKeysJSON: strings.TrimSpace(step.ModifierKeysJSON),
		DurationMs:       step.DurationMs,
		WaitMs:           step.WaitMs,
		PayloadJSON:      strings.TrimSpace(step.PayloadJSON),
	}
	if row.ModifierKeysJSON == "" {
		row.ModifierKeysJSON = "[]"
	}
	if row.PayloadJSON == "" {
		row.PayloadJSON = "{}"
	}
	if row.DurationMs < 0 || row.WaitMs < 0 || row.DurationMs > maxDurationMs || row.WaitMs > maxDurationMs {
		return storage.MacroStep{}, fmt.Errorf("duration out of range")
	}
	switch row.Kind {
	case StepDelay:
		row.DeviceKind = ""
		if row.WaitMs <= 0 {
			return storage.MacroStep{}, fmt.Errorf("delay requires waitMs")
		}
		if row.DurationMs != 0 || hasKey(row) || row.ModifierKeysJSON != "[]" {
			return storage.MacroStep{}, fmt.Errorf("delay cannot include key or duration")
		}
	case StepKeyTap:
		row.DeviceKind = DeviceKeyboard
		if err := validateKeyRow(row); err != nil {
			return storage.MacroStep{}, err
		}
		if row.DurationMs <= 0 {
			row.DurationMs = defaultKeyTapMs
		}
		if row.WaitMs != 0 || row.ModifierKeysJSON != "[]" {
			return storage.MacroStep{}, fmt.Errorf("keyTap cannot include waitMs or modifiers")
		}
	case StepKeyDown, StepKeyUp:
		row.DeviceKind = DeviceKeyboard
		if err := validateKeyRow(row); err != nil {
			return storage.MacroStep{}, err
		}
		if row.DurationMs != 0 || row.WaitMs != 0 || row.ModifierKeysJSON != "[]" {
			return storage.MacroStep{}, fmt.Errorf("%s cannot include duration, waitMs, or modifiers", row.Kind)
		}
	case StepChordTap:
		row.DeviceKind = DeviceKeyboard
		if err := validateKeyRow(row); err != nil {
			return storage.MacroStep{}, err
		}
		if row.DurationMs <= 0 {
			row.DurationMs = defaultKeyTapMs
		}
		if row.WaitMs != 0 {
			return storage.MacroStep{}, fmt.Errorf("chordTap cannot include waitMs")
		}
		mods, err := keysim.DecodeModifiers(row.ModifierKeysJSON)
		if err != nil || len(mods) == 0 {
			return storage.MacroStep{}, fmt.Errorf("chordTap requires modifiers")
		}
	case StepMouseTap:
		row.DeviceKind = DeviceMouse
		if err := validateMouseRow(row); err != nil {
			return storage.MacroStep{}, err
		}
		if row.DurationMs <= 0 {
			row.DurationMs = defaultKeyTapMs
		}
		if row.WaitMs != 0 || row.ModifierKeysJSON != "[]" {
			return storage.MacroStep{}, fmt.Errorf("mouseTap cannot include waitMs or modifiers")
		}
	case StepMouseDown, StepMouseUp:
		row.DeviceKind = DeviceMouse
		if err := validateMouseRow(row); err != nil {
			return storage.MacroStep{}, err
		}
		if row.DurationMs != 0 || row.WaitMs != 0 || row.ModifierKeysJSON != "[]" {
			return storage.MacroStep{}, fmt.Errorf("%s cannot include duration, waitMs, or modifiers", row.Kind)
		}
	case StepMouseScroll:
		row.DeviceKind = DeviceMouse
		if err := validateWheelRow(row); err != nil {
			return storage.MacroStep{}, err
		}
		// Scroll is a one-shot notch with no hold/release semantics.
		if row.DurationMs != 0 || row.WaitMs != 0 || row.ModifierKeysJSON != "[]" {
			return storage.MacroStep{}, fmt.Errorf("mouseScroll cannot include duration, waitMs, or modifiers")
		}
	case StepMouseMove:
		row.DeviceKind = DeviceMouse
		// Movement carries no key/button; the offset lives in PayloadJSON.
		if hasKey(row) || row.ModifierKeysJSON != "[]" {
			return storage.MacroStep{}, fmt.Errorf("mouseMove cannot include key or modifiers")
		}
		if row.WaitMs != 0 {
			return storage.MacroStep{}, fmt.Errorf("mouseMove cannot include waitMs")
		}
		payload, err := DecodeMousePayload(row.PayloadJSON)
		if err != nil {
			return storage.MacroStep{}, err
		}
		encoded, err := EncodeMousePayload(payload)
		if err != nil {
			return storage.MacroStep{}, err
		}
		row.PayloadJSON = encoded
		// DurationMs is the travel time the move is spread across (controls
		// speed). It is user-editable but floored to the nominal minimum so the
		// planner cursor always advances and at least one sub-move is emitted.
		if row.DurationMs < mouseMoveDurationMs {
			row.DurationMs = mouseMoveDurationMs
		}
	case StepText:
		row.DeviceKind = DeviceKeyboard
		if hasKey(row) || row.ModifierKeysJSON != "[]" {
			return storage.MacroStep{}, fmt.Errorf("text cannot include key or modifiers")
		}
		if row.WaitMs != 0 {
			return storage.MacroStep{}, fmt.Errorf("text cannot include waitMs")
		}
		payload, err := DecodeTextPayload(row.PayloadJSON)
		if err != nil {
			return storage.MacroStep{}, err
		}
		// Re-encode so the persisted payload is normalized (trimmed delay bounds).
		encoded, err := EncodeTextPayload(payload)
		if err != nil {
			return storage.MacroStep{}, err
		}
		row.PayloadJSON = encoded
		// DurationMs is derived from the text length so the timeline planner can
		// advance the cursor; it is not user-editable for text steps.
		row.DurationMs = textDurationMs(payload)
	default:
		return storage.MacroStep{}, fmt.Errorf("unknown kind %q", row.Kind)
	}
	return row, nil
}

func validateKeyRow(row storage.MacroStep) error {
	key := keysim.Key{Label: row.KeyLabel, VirtualKey: row.VirtualKey, ScanCode: row.ScanCode}
	if key.Label == "" {
		return fmt.Errorf("missing key label")
	}
	if key.ScanCode == 0 && key.VirtualKey == 0 {
		return fmt.Errorf("missing key code")
	}
	return nil
}

func validateMouseRow(row storage.MacroStep) error {
	if row.KeyLabel == "" {
		return fmt.Errorf("missing mouse button label")
	}
	switch row.VirtualKey {
	case keysim.MouseButtonLeft, keysim.MouseButtonRight, keysim.MouseButtonMiddle,
		keysim.MouseButtonX1, keysim.MouseButtonX2:
		return nil
	default:
		return fmt.Errorf("invalid mouse button %d", row.VirtualKey)
	}
}

// validateWheelRow accepts only the four scroll-wheel direction ids.
func validateWheelRow(row storage.MacroStep) error {
	if row.KeyLabel == "" {
		return fmt.Errorf("missing scroll direction label")
	}
	switch row.VirtualKey {
	case keysim.MouseWheelUp, keysim.MouseWheelDown, keysim.MouseWheelLeft, keysim.MouseWheelRight:
		return nil
	default:
		return fmt.Errorf("invalid scroll direction %d", row.VirtualKey)
	}
}

func hasKey(row storage.MacroStep) bool {
	return row.KeyLabel != "" || row.VirtualKey != 0 || row.ScanCode != 0
}

func stepDTO(row storage.MacroStep) MacroStepDTO {
	return MacroStepDTO{
		ID:               row.ID,
		MacroID:          row.MacroID,
		OrderIndex:       row.OrderIndex,
		Kind:             row.Kind,
		KeyLabel:         row.KeyLabel,
		VirtualKey:       row.VirtualKey,
		ScanCode:         row.ScanCode,
		DeviceKind:       row.DeviceKind,
		ModifierKeysJSON: row.ModifierKeysJSON,
		DurationMs:       row.DurationMs,
		WaitMs:           row.WaitMs,
		PayloadJSON:      row.PayloadJSON,
	}
}

func keyFromStep(row storage.MacroStep) keysim.Key {
	kind := keysim.KeyKindKeyboard
	if row.DeviceKind == DeviceMouse {
		kind = keysim.KeyKindMouse
	}
	return keysim.Key{Label: row.KeyLabel, VirtualKey: row.VirtualKey, ScanCode: row.ScanCode, Kind: kind}
}

//go:build completion

package macro

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	// maxTextRunes caps a single text block to keep injection bounded and avoid
	// pathological macros. G HUB practically tops out around a few hundred chars.
	maxTextRunes = 1000
	// maxTextPerCharDelayMs bounds the optional per-character throttle.
	maxTextPerCharDelayMs = 1000
	// textDurationFloorMs keeps the planner cursor advancing even for short text.
	textDurationFloorMs = 40
)

// TextPayload is the decoded JSON shape stored in MacroStep.PayloadJSON for
// StepText. Reusing the long-reserved payload column means zero DB migration.
type TextPayload struct {
	Text           string `json:"text"`
	PerCharDelayMs int64  `json:"perCharDelayMs,omitempty"`
}

// DecodeTextPayload parses and validates a text-block payload. An empty or "{}"
// payload is treated as an empty (invalid) text block so callers get a clear error.
func DecodeTextPayload(raw string) (TextPayload, error) {
	var payload TextPayload
	trimmed := strings.TrimSpace(raw)
	if trimmed != "" && trimmed != "{}" {
		if err := json.Unmarshal([]byte(trimmed), &payload); err != nil {
			return TextPayload{}, fmt.Errorf("invalid text payload: %w", err)
		}
	}
	if payload.Text == "" {
		return TextPayload{}, fmt.Errorf("text block requires non-empty text")
	}
	if !utf8.ValidString(payload.Text) {
		return TextPayload{}, fmt.Errorf("text block contains invalid UTF-8")
	}
	if n := utf8.RuneCountInString(payload.Text); n > maxTextRunes {
		return TextPayload{}, fmt.Errorf("text block too long: %d > %d", n, maxTextRunes)
	}
	if payload.PerCharDelayMs < 0 {
		payload.PerCharDelayMs = 0
	}
	if payload.PerCharDelayMs > maxTextPerCharDelayMs {
		payload.PerCharDelayMs = maxTextPerCharDelayMs
	}
	return payload, nil
}

// EncodeTextPayload serializes a normalized payload back to JSON for storage.
func EncodeTextPayload(payload TextPayload) (string, error) {
	out, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("encode text payload: %w", err)
	}
	return string(out), nil
}

// textDurationMs estimates how long the timeline cursor should advance for a
// text step, factoring in any per-character throttle.
func textDurationMs(payload TextPayload) int64 {
	runes := int64(utf8.RuneCountInString(payload.Text))
	dur := runes * payload.PerCharDelayMs
	if dur < textDurationFloorMs {
		return textDurationFloorMs
	}
	return dur
}

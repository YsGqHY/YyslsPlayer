//go:build completion

package macro

import (
	"encoding/json"
	"fmt"
	"strings"

	"YyslsPlayer/internal/services/keysim"
	"YyslsPlayer/internal/storage"
)

const (
	// currentPortableVersion is the schema version stamped into exported YAML
	// documents. Import rejects any document with a different version so future
	// breaking changes can be detected explicitly rather than mis-parsed.
	currentPortableVersion = 1
	// maxImportFileSize bounds how much YAML we read so a malformed/huge file
	// cannot exhaust memory.
	maxImportFileSize = 1 << 20 // 1 MiB
	// maxImportMacros bounds how many macros one file may carry.
	maxImportMacros = 200
)

// portableDoc is the top-level YAML shape for macro import/export. A single
// macro is exported as a one-element list so the format is symmetric and can
// carry many macros without a schema change.
type portableDoc struct {
	Version int             `yaml:"version"`
	Macros  []portableMacro `yaml:"macros"`
}

// portableMacro is one macro rendered in a human-readable YAML form. The nested
// JSON columns (modifiers / payload) are decoded into native YAML structures so
// the file is editable by hand.
type portableMacro struct {
	Name             string         `yaml:"name"`
	Description      string         `yaml:"description,omitempty"`
	Trigger          string         `yaml:"trigger,omitempty"`
	RepeatMode       string         `yaml:"repeatMode,omitempty"`
	RepeatCount      int            `yaml:"repeatCount,omitempty"`
	RepeatIntervalMs int64          `yaml:"repeatIntervalMs,omitempty"`
	InterruptPolicy  string         `yaml:"interruptPolicy,omitempty"`
	Steps            []portableStep `yaml:"steps"`
}

// portableStep mirrors MacroStepDTO but with optional, kind-specific fields so
// only the relevant data appears per step.
type portableStep struct {
	Kind       string        `yaml:"kind"`
	Key        *portableKey  `yaml:"key,omitempty"`
	Modifiers  []portableKey `yaml:"modifiers,omitempty"`
	Text       *portableText `yaml:"text,omitempty"`
	Move       *portableMove `yaml:"move,omitempty"`
	DurationMs int64         `yaml:"durationMs,omitempty"`
	WaitMs     int64         `yaml:"waitMs,omitempty"`
}

type portableKey struct {
	Label      string `yaml:"label"`
	VirtualKey int    `yaml:"virtualKey"`
	ScanCode   int    `yaml:"scanCode"`
}

type portableText struct {
	Value          string `yaml:"value"`
	PerCharDelayMs int64  `yaml:"perCharDelayMs,omitempty"`
}

type portableMove struct {
	Dx     int `yaml:"dx"`
	Dy     int `yaml:"dy"`
	Jitter int `yaml:"jitter,omitempty"`
}

// toPortableDoc converts persisted macro details into the export document,
// decoding the JSON columns into readable YAML structures.
func toPortableDoc(details []storage.MacroDetail) (portableDoc, error) {
	doc := portableDoc{Version: currentPortableVersion, Macros: make([]portableMacro, 0, len(details))}
	for _, detail := range details {
		pm := portableMacro{
			Name:             detail.Profile.Name,
			Description:      detail.Profile.Description,
			Trigger:          detail.Profile.TriggerAccelerator,
			RepeatMode:       detail.Profile.RepeatMode,
			RepeatCount:      detail.Profile.RepeatCount,
			RepeatIntervalMs: detail.Profile.RepeatIntervalMs,
			InterruptPolicy:  detail.Profile.InterruptPolicy,
			Steps:            make([]portableStep, 0, len(detail.Steps)),
		}
		for _, row := range detail.Steps {
			ps, err := toPortableStep(row)
			if err != nil {
				return portableDoc{}, err
			}
			pm.Steps = append(pm.Steps, ps)
		}
		doc.Macros = append(doc.Macros, pm)
	}
	return doc, nil
}

func toPortableStep(row storage.MacroStep) (portableStep, error) {
	ps := portableStep{
		Kind:       row.Kind,
		DurationMs: row.DurationMs,
		WaitMs:     row.WaitMs,
	}
	if hasKey(row) {
		ps.Key = &portableKey{Label: row.KeyLabel, VirtualKey: row.VirtualKey, ScanCode: row.ScanCode}
	}
	switch row.Kind {
	case StepChordTap:
		mods, err := keysim.DecodeModifiers(row.ModifierKeysJSON)
		if err != nil {
			return portableStep{}, fmt.Errorf("decode chord modifiers: %w", err)
		}
		for _, m := range mods {
			ps.Modifiers = append(ps.Modifiers, portableKey{Label: m.Label, VirtualKey: m.VirtualKey, ScanCode: m.ScanCode})
		}
	case StepText:
		payload, err := DecodeTextPayload(row.PayloadJSON)
		if err != nil {
			return portableStep{}, fmt.Errorf("decode text payload: %w", err)
		}
		ps.Text = &portableText{Value: payload.Text, PerCharDelayMs: payload.PerCharDelayMs}
		// DurationMs is derived for text steps; drop it from the portable form.
		ps.DurationMs = 0
	case StepMouseMove:
		payload, err := DecodeMousePayload(row.PayloadJSON)
		if err != nil {
			return portableStep{}, fmt.Errorf("decode mouse payload: %w", err)
		}
		ps.Move = &portableMove{Dx: payload.Dx, Dy: payload.Dy, Jitter: payload.Jitter}
	}
	return ps, nil
}

// fromPortableDoc converts an imported document into save requests. Each macro
// gets a fresh ID and is disabled so importing never registers a hotkey that
// could collide with an existing binding. An invalid trigger is cleared on that
// macro rather than failing the whole import.
func fromPortableDoc(doc portableDoc) ([]SaveMacroRequest, error) {
	if doc.Version != currentPortableVersion {
		return nil, fmt.Errorf("%w: unsupported version %d", ErrMacroImportInvalid, doc.Version)
	}
	if len(doc.Macros) == 0 {
		return nil, fmt.Errorf("%w: no macros in file", ErrMacroImportInvalid)
	}
	if len(doc.Macros) > maxImportMacros {
		return nil, fmt.Errorf("%w: too many macros (%d)", ErrMacroImportInvalid, len(doc.Macros))
	}
	out := make([]SaveMacroRequest, 0, len(doc.Macros))
	for i, pm := range doc.Macros {
		req, err := fromPortableMacro(pm)
		if err != nil {
			return nil, fmt.Errorf("%w: macro %d: %w", ErrMacroImportInvalid, i+1, err)
		}
		out = append(out, req)
	}
	return out, nil
}

func fromPortableMacro(pm portableMacro) (SaveMacroRequest, error) {
	req := SaveMacroRequest{
		ID:                 0,
		Name:               strings.TrimSpace(pm.Name),
		Description:        pm.Description,
		Enabled:            false,
		RepeatMode:         pm.RepeatMode,
		RepeatCount:        pm.RepeatCount,
		RepeatIntervalMs:   pm.RepeatIntervalMs,
		InterruptPolicy:    pm.InterruptPolicy,
		TriggerAccelerator: strings.TrimSpace(pm.Trigger),
		Steps:              make([]MacroStepDTO, 0, len(pm.Steps)),
	}
	for i, ps := range pm.Steps {
		step, err := fromPortableStep(ps)
		if err != nil {
			return SaveMacroRequest{}, fmt.Errorf("step %d: %w", i+1, err)
		}
		req.Steps = append(req.Steps, step)
	}
	return req, nil
}

func fromPortableStep(ps portableStep) (MacroStepDTO, error) {
	step := MacroStepDTO{
		Kind:       strings.TrimSpace(ps.Kind),
		DurationMs: ps.DurationMs,
		WaitMs:     ps.WaitMs,
	}
	if ps.Key != nil {
		step.KeyLabel = ps.Key.Label
		step.VirtualKey = ps.Key.VirtualKey
		step.ScanCode = ps.Key.ScanCode
	}
	if len(ps.Modifiers) > 0 {
		mods := make([]portableKey, len(ps.Modifiers))
		copy(mods, ps.Modifiers)
		encoded, err := json.Marshal(mods)
		if err != nil {
			return MacroStepDTO{}, fmt.Errorf("encode modifiers: %w", err)
		}
		step.ModifierKeysJSON = string(encoded)
	}
	if ps.Text != nil {
		encoded, err := EncodeTextPayload(TextPayload{Text: ps.Text.Value, PerCharDelayMs: ps.Text.PerCharDelayMs})
		if err != nil {
			return MacroStepDTO{}, fmt.Errorf("encode text: %w", err)
		}
		step.PayloadJSON = encoded
	}
	if ps.Move != nil {
		encoded, err := EncodeMousePayload(MousePayload{Dx: ps.Move.Dx, Dy: ps.Move.Dy, Jitter: ps.Move.Jitter})
		if err != nil {
			return MacroStepDTO{}, fmt.Errorf("encode move: %w", err)
		}
		step.PayloadJSON = encoded
	}
	return step, nil
}

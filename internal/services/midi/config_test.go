package midi

import (
	"errors"
	"testing"

	"YyslsPlayer/internal/storage"
)

func TestValidateMidiConfigAcceptsDefaultsAndBoundaries(t *testing.T) {
	defaults := DefaultMidiConfig()
	validated, err := ValidateMidiConfig(defaults)
	if err != nil {
		t.Fatalf("ValidateMidiConfig(defaults) failed: %v", err)
	}
	if validated != defaults {
		t.Fatalf("validated defaults = %+v, want %+v", validated, defaults)
	}

	boundary := MidiConfigDTO{
		BaseNote:         MaxBaseNote,
		Transpose:        MinTranspose,
		OctaveShift:      MaxOctaveShift,
		Speed:            MaxSpeed,
		OutOfRangePolicy: OutOfRangeNearest,
		MinPressMs:       MinPressMs,
		ReleaseGapMs:     MaxReleaseGapMs,
		KeymapProfileID:  storage.DefaultKeymapProfileID,
	}
	if _, err := ValidateMidiConfig(boundary); err != nil {
		t.Fatalf("ValidateMidiConfig(boundary) failed: %v", err)
	}

	boundary.BaseNote = MinBaseNote
	boundary.Transpose = MaxTranspose
	boundary.OctaveShift = MinOctaveShift
	boundary.Speed = MinSpeed
	boundary.OutOfRangePolicy = OutOfRangeOctaveFold
	boundary.MinPressMs = MaxPressMs
	boundary.ReleaseGapMs = MinReleaseGapMs
	if _, err := ValidateMidiConfig(boundary); err != nil {
		t.Fatalf("ValidateMidiConfig(boundary low/high swap) failed: %v", err)
	}
}

func TestValidateMidiConfigDefaultsEmptyPolicyToDrop(t *testing.T) {
	cfg := DefaultMidiConfig()
	cfg.OutOfRangePolicy = "   "
	validated, err := ValidateMidiConfig(cfg)
	if err != nil {
		t.Fatalf("ValidateMidiConfig failed: %v", err)
	}
	if validated.OutOfRangePolicy != OutOfRangeDrop {
		t.Fatalf("policy = %q, want drop", validated.OutOfRangePolicy)
	}
}

func TestValidateMidiConfigRejectsInvalidFields(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*MidiConfigDTO)
	}{
		{name: "base low", mutate: func(c *MidiConfigDTO) { c.BaseNote = MinBaseNote - 1 }},
		{name: "base high", mutate: func(c *MidiConfigDTO) { c.BaseNote = MaxBaseNote + 1 }},
		{name: "transpose low", mutate: func(c *MidiConfigDTO) { c.Transpose = MinTranspose - 1 }},
		{name: "transpose high", mutate: func(c *MidiConfigDTO) { c.Transpose = MaxTranspose + 1 }},
		{name: "octave low", mutate: func(c *MidiConfigDTO) { c.OctaveShift = MinOctaveShift - 1 }},
		{name: "octave high", mutate: func(c *MidiConfigDTO) { c.OctaveShift = MaxOctaveShift + 1 }},
		{name: "speed low", mutate: func(c *MidiConfigDTO) { c.Speed = MinSpeed - 0.01 }},
		{name: "speed high", mutate: func(c *MidiConfigDTO) { c.Speed = MaxSpeed + 0.01 }},
		{name: "policy", mutate: func(c *MidiConfigDTO) { c.OutOfRangePolicy = "wrap" }},
		{name: "press low", mutate: func(c *MidiConfigDTO) { c.MinPressMs = MinPressMs - 1 }},
		{name: "press high", mutate: func(c *MidiConfigDTO) { c.MinPressMs = MaxPressMs + 1 }},
		{name: "gap low", mutate: func(c *MidiConfigDTO) { c.ReleaseGapMs = MinReleaseGapMs - 1 }},
		{name: "gap high", mutate: func(c *MidiConfigDTO) { c.ReleaseGapMs = MaxReleaseGapMs + 1 }},
		{name: "keymap zero", mutate: func(c *MidiConfigDTO) { c.KeymapProfileID = 0 }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := DefaultMidiConfig()
			tc.mutate(&cfg)
			if _, err := ValidateMidiConfig(cfg); !errors.Is(err, ErrInvalidMidiConfig) {
				t.Fatalf("ValidateMidiConfig error = %v, want INVALID_MIDI_CONFIG", err)
			}
		})
	}
}

func TestMidiConfigFromProfile(t *testing.T) {
	profile := storage.MidiProfile{
		BaseNote:         50,
		Transpose:        -2,
		OctaveShift:      1,
		Speed:            1.5,
		OutOfRangePolicy: OutOfRangeNearest,
		MinPressMs:       40,
		ReleaseGapMs:     12,
		KeymapProfileID:  7,
	}
	cfg := MidiConfigFromProfile(profile)
	if cfg.BaseNote != profile.BaseNote || cfg.Transpose != profile.Transpose || cfg.KeymapProfileID != profile.KeymapProfileID {
		t.Fatalf("config from profile = %+v", cfg)
	}
}

func TestValidateMidiProfileDTOAppliesSanitizedConfig(t *testing.T) {
	profile := MidiProfileDTO{
		ID:               99,
		Name:             "Custom",
		BaseNote:         storage.DefaultMidiBaseNote,
		Transpose:        storage.DefaultMidiTranspose,
		OctaveShift:      storage.DefaultMidiOctaveShift,
		Speed:            storage.DefaultMidiSpeed,
		OutOfRangePolicy: "",
		MinPressMs:       storage.DefaultMinPressMs,
		ReleaseGapMs:     storage.DefaultReleaseGapMs,
		KeymapProfileID:  storage.DefaultKeymapProfileID,
	}

	validated, err := ValidateMidiProfileDTO(profile)
	if err != nil {
		t.Fatalf("ValidateMidiProfileDTO failed: %v", err)
	}
	if validated.ID != profile.ID || validated.Name != profile.Name {
		t.Fatalf("identity fields changed: %+v", validated)
	}
	if validated.OutOfRangePolicy != OutOfRangeDrop {
		t.Fatalf("policy = %q, want drop", validated.OutOfRangePolicy)
	}
}

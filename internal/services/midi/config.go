package midi

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"YyslsPlayer/internal/storage"
)

const (
	OutOfRangeDrop       = "drop"
	OutOfRangeOctaveFold = "octaveFold"
	OutOfRangeNearest    = "nearest"

	MinBaseNote     = 0
	MaxBaseNote     = 127
	MinTranspose    = -24
	MaxTranspose    = 24
	MinOctaveShift  = -3
	MaxOctaveShift  = 3
	MinSpeed        = 0.25
	MaxSpeed        = 3.0
	MinPressMs      = 10
	MaxPressMs      = 300
	MinReleaseGapMs = 0
	MaxReleaseGapMs = 200
	MinTrackNumber  = 0
	MaxTrackNumber  = 127
)

var ErrInvalidMidiConfig = errors.New("INVALID_MIDI_CONFIG")

type MidiConfigDTO struct {
	BaseNote         int     `json:"baseNote"`
	Transpose        int     `json:"transpose"`
	OctaveShift      int     `json:"octaveShift"`
	Speed            float64 `json:"speed"`
	OutOfRangePolicy string  `json:"outOfRangePolicy"`
	MinPressMs       int     `json:"minPressMs"`
	ReleaseGapMs     int     `json:"releaseGapMs"`
	KeymapProfileID  uint    `json:"keymapProfileId"`
	EnabledTracks    *[]int  `json:"enabledTracks"`
}

func DefaultMidiConfig() MidiConfigDTO {
	return MidiConfigDTO{
		BaseNote:         storage.DefaultMidiBaseNote,
		Transpose:        storage.DefaultMidiTranspose,
		OctaveShift:      storage.DefaultMidiOctaveShift,
		Speed:            storage.DefaultMidiSpeed,
		OutOfRangePolicy: storage.DefaultOutOfRangePolicy,
		MinPressMs:       storage.DefaultMinPressMs,
		ReleaseGapMs:     storage.DefaultReleaseGapMs,
		KeymapProfileID:  storage.DefaultKeymapProfileID,
	}
}

func ValidateMidiConfig(cfg MidiConfigDTO) (MidiConfigDTO, error) {
	cfg.OutOfRangePolicy = strings.TrimSpace(cfg.OutOfRangePolicy)
	if cfg.OutOfRangePolicy == "" {
		cfg.OutOfRangePolicy = OutOfRangeDrop
	}

	if cfg.BaseNote < MinBaseNote || cfg.BaseNote > MaxBaseNote {
		return cfg, configError("baseNote", cfg.BaseNote, "0..127")
	}
	if cfg.Transpose < MinTranspose || cfg.Transpose > MaxTranspose {
		return cfg, configError("transpose", cfg.Transpose, "-24..24")
	}
	if cfg.OctaveShift < MinOctaveShift || cfg.OctaveShift > MaxOctaveShift {
		return cfg, configError("octaveShift", cfg.OctaveShift, "-3..3")
	}
	if cfg.Speed < MinSpeed || cfg.Speed > MaxSpeed {
		return cfg, configError("speed", cfg.Speed, "0.25..3.0")
	}
	if !isAllowedOutOfRangePolicy(cfg.OutOfRangePolicy) {
		return cfg, configError("outOfRangePolicy", cfg.OutOfRangePolicy, "drop/octaveFold/nearest")
	}
	if cfg.MinPressMs < MinPressMs || cfg.MinPressMs > MaxPressMs {
		return cfg, configError("minPressMs", cfg.MinPressMs, "10..300")
	}
	if cfg.ReleaseGapMs < MinReleaseGapMs || cfg.ReleaseGapMs > MaxReleaseGapMs {
		return cfg, configError("releaseGapMs", cfg.ReleaseGapMs, "0..200")
	}
	if cfg.KeymapProfileID == 0 {
		return cfg, configError("keymapProfileId", cfg.KeymapProfileID, "> 0")
	}
	tracks, err := normalizeEnabledTracks(cfg.EnabledTracks)
	if err != nil {
		return cfg, err
	}
	cfg.EnabledTracks = tracks
	return cfg, nil
}

func MidiConfigFromProfile(profile storage.MidiProfile) MidiConfigDTO {
	return MidiConfigDTO{
		BaseNote:         profile.BaseNote,
		Transpose:        profile.Transpose,
		OctaveShift:      profile.OctaveShift,
		Speed:            profile.Speed,
		OutOfRangePolicy: profile.OutOfRangePolicy,
		MinPressMs:       profile.MinPressMs,
		ReleaseGapMs:     profile.ReleaseGapMs,
		KeymapProfileID:  profile.KeymapProfileID,
		EnabledTracks:    decodeEnabledTracks(profile.EnabledTracksJSON),
	}
}

func MidiConfigFromProfileDTO(profile MidiProfileDTO) MidiConfigDTO {
	return MidiConfigDTO{
		BaseNote:         profile.BaseNote,
		Transpose:        profile.Transpose,
		OctaveShift:      profile.OctaveShift,
		Speed:            profile.Speed,
		OutOfRangePolicy: profile.OutOfRangePolicy,
		MinPressMs:       profile.MinPressMs,
		ReleaseGapMs:     profile.ReleaseGapMs,
		KeymapProfileID:  profile.KeymapProfileID,
		EnabledTracks:    cloneIntSlice(profile.EnabledTracks),
	}
}

func ApplyConfigToProfileDTO(profile MidiProfileDTO, cfg MidiConfigDTO) MidiProfileDTO {
	profile.BaseNote = cfg.BaseNote
	profile.Transpose = cfg.Transpose
	profile.OctaveShift = cfg.OctaveShift
	profile.Speed = cfg.Speed
	profile.OutOfRangePolicy = cfg.OutOfRangePolicy
	profile.MinPressMs = cfg.MinPressMs
	profile.ReleaseGapMs = cfg.ReleaseGapMs
	profile.KeymapProfileID = cfg.KeymapProfileID
	profile.EnabledTracks = cloneIntSlice(cfg.EnabledTracks)
	return profile
}

func ValidateMidiProfileDTO(profile MidiProfileDTO) (MidiProfileDTO, error) {
	cfg, err := ValidateMidiConfig(MidiConfigFromProfileDTO(profile))
	if err != nil {
		return profile, err
	}
	return ApplyConfigToProfileDTO(profile, cfg), nil
}

func isAllowedOutOfRangePolicy(policy string) bool {
	switch policy {
	case OutOfRangeDrop, OutOfRangeOctaveFold, OutOfRangeNearest:
		return true
	default:
		return false
	}
}

func configError(field string, value any, allowed string) error {
	return fmt.Errorf("%w: %s=%v outside %s", ErrInvalidMidiConfig, field, value, allowed)
}

func normalizeEnabledTracks(tracks *[]int) (*[]int, error) {
	if tracks == nil {
		return nil, nil
	}
	seen := make(map[int]bool, len(*tracks))
	out := make([]int, 0, len(*tracks))
	for _, track := range *tracks {
		if track < MinTrackNumber || track > MaxTrackNumber {
			return nil, configError("enabledTracks", track, "0..127")
		}
		if seen[track] {
			continue
		}
		seen[track] = true
		out = append(out, track)
	}
	sort.Ints(out)
	return &out, nil
}

func decodeEnabledTracks(raw string) *[]int {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "null" {
		return nil
	}
	var tracks []int
	if err := json.Unmarshal([]byte(raw), &tracks); err != nil {
		return nil
	}
	tracksRef, err := normalizeEnabledTracks(&tracks)
	if err != nil {
		return nil
	}
	return tracksRef
}

func encodeEnabledTracks(tracks *[]int) string {
	if tracks == nil {
		return "null"
	}
	tracksRef, err := normalizeEnabledTracks(tracks)
	if err != nil || tracksRef == nil {
		return "null"
	}
	data, err := json.Marshal(*tracksRef)
	if err != nil {
		return "null"
	}
	return string(data)
}

func cloneIntSlice(values *[]int) *[]int {
	if values == nil {
		return nil
	}
	out := make([]int, len(*values))
	copy(out, *values)
	return &out
}

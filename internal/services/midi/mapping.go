package midi

import (
	"errors"
	"fmt"

	"YyslsPlayer/internal/storage"
)

const (
	LaneCount = 36
	MinLane   = 0
	MaxLane   = LaneCount - 1
)

var (
	ErrLaneOutOfRange   = errors.New("LANE_OUT_OF_RANGE")
	ErrKeymapIncomplete = errors.New("KEYMAP_INCOMPLETE")
)

type LaneMappingDTO struct {
	SourceNote     int           `json:"sourceNote"`
	NormalizedNote int           `json:"normalizedNote"`
	RawLane        int           `json:"rawLane"`
	Lane           int           `json:"lane"`
	Policy         string        `json:"policy"`
	Dropped        bool          `json:"dropped"`
	Folded         bool          `json:"folded"`
	Clamped        bool          `json:"clamped"`
	Key            KeymapLaneDTO `json:"key"`
}

type Key36Mapping struct {
	ProfileID   uint
	ProfileName string
	Lanes       [LaneCount]KeymapLaneDTO
}

type MappingStats struct {
	TotalNotes      int
	PlayableNotes   int
	OutOfRangeCount int
	DroppedCount    int
	FoldedCount     int
	ClampedCount    int
	MinLane         int
	MaxLane         int
}

func MapNoteToLane(note int, cfg MidiConfigDTO) (int, int, error) {
	result, err := MapNoteWithPolicy(note, cfg)
	if err != nil {
		return result.NormalizedNote, result.RawLane, err
	}
	if result.Dropped || result.Lane < MinLane || result.Lane > MaxLane {
		return result.NormalizedNote, result.RawLane, fmt.Errorf("%w: note=%d normalized=%d lane=%d", ErrLaneOutOfRange, note, result.NormalizedNote, result.RawLane)
	}
	return result.NormalizedNote, result.Lane, nil
}

func MapNoteWithPolicy(note int, cfg MidiConfigDTO) (LaneMappingDTO, error) {
	validated, err := ValidateMidiConfig(cfg)
	if err != nil {
		return LaneMappingDTO{SourceNote: note}, err
	}
	normalized := note + validated.Transpose + validated.OctaveShift*12
	rawLane := normalized - validated.BaseNote
	result := LaneMappingDTO{
		SourceNote:     note,
		NormalizedNote: normalized,
		RawLane:        rawLane,
		Lane:           rawLane,
		Policy:         validated.OutOfRangePolicy,
	}
	if rawLane >= MinLane && rawLane <= MaxLane {
		return result, nil
	}

	switch validated.OutOfRangePolicy {
	case OutOfRangeDrop:
		result.Dropped = true
		return result, nil
	case OutOfRangeOctaveFold:
		result.Folded = true
		result.Lane = foldLaneToRange(rawLane)
		return result, nil
	case OutOfRangeNearest:
		result.Clamped = true
		result.Lane = clampLane(rawLane)
		return result, nil
	default:
		return result, configError("outOfRangePolicy", validated.OutOfRangePolicy, "drop/octaveFold/nearest")
	}
}

func MappingStatsFromResults(results []LaneMappingDTO) MappingStats {
	stats := MappingStats{MinLane: -1, MaxLane: -1}
	for _, result := range results {
		stats.TotalNotes++
		if result.RawLane < MinLane || result.RawLane > MaxLane {
			stats.OutOfRangeCount++
		}
		if result.Dropped {
			stats.DroppedCount++
			continue
		}
		stats.PlayableNotes++
		if result.Folded {
			stats.FoldedCount++
		}
		if result.Clamped {
			stats.ClampedCount++
		}
		if stats.MinLane == -1 || result.Lane < stats.MinLane {
			stats.MinLane = result.Lane
		}
		if stats.MaxLane == -1 || result.Lane > stats.MaxLane {
			stats.MaxLane = result.Lane
		}
	}
	return stats
}

func foldLaneToRange(lane int) int {
	folded := lane % 12
	if folded < 0 {
		folded += 12
	}
	return folded
}

func clampLane(lane int) int {
	if lane < MinLane {
		return MinLane
	}
	if lane > MaxLane {
		return MaxLane
	}
	return lane
}

func LoadKey36Mapping(store *storage.Store, profileID uint) (Key36Mapping, error) {
	if profileID == 0 {
		return Key36Mapping{}, fmt.Errorf("%w: profile id required", ErrKeymapIncomplete)
	}
	return Key36MappingFromRows(profileID, store.ListKeymapProfile(profileID))
}

func Key36MappingFromRows(profileID uint, rows []storage.Keymap36) (Key36Mapping, error) {
	mapping := Key36Mapping{ProfileID: profileID}
	seen := make(map[int]bool, LaneCount)
	for _, row := range rows {
		if row.Lane < MinLane || row.Lane > MaxLane {
			return Key36Mapping{}, fmt.Errorf("%w: lane=%d", ErrKeymapIncomplete, row.Lane)
		}
		if seen[row.Lane] {
			return Key36Mapping{}, fmt.Errorf("%w: duplicate lane=%d", ErrKeymapIncomplete, row.Lane)
		}
		seen[row.Lane] = true
		if mapping.ProfileName == "" {
			mapping.ProfileName = row.ProfileName
		}
		mapping.Lanes[row.Lane] = keymapLaneDTO(row)
	}
	if len(seen) != LaneCount {
		return Key36Mapping{}, fmt.Errorf("%w: got %d lanes, want %d", ErrKeymapIncomplete, len(seen), LaneCount)
	}
	return mapping, nil
}

func MapEventToLaneKey(event storage.MidiEvent, cfg MidiConfigDTO, mapping Key36Mapping) (LaneMappingDTO, error) {
	result, err := MapNoteWithPolicy(event.Note, cfg)
	if err != nil {
		return result, err
	}
	if result.Dropped {
		return result, nil
	}
	if result.Lane < MinLane || result.Lane > MaxLane {
		return result, fmt.Errorf("%w: note=%d normalized=%d lane=%d", ErrLaneOutOfRange, event.Note, result.NormalizedNote, result.Lane)
	}
	result.Key = mapping.Lanes[result.Lane]
	return result, nil
}

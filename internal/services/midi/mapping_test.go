package midi

import (
	"errors"
	"testing"

	"YyslsPlayer/internal/storage"
)

func TestMapNoteToLaneDefaultRange(t *testing.T) {
	cfg := DefaultMidiConfig()

	normalized, lane, err := MapNoteToLane(48, cfg)
	if err != nil {
		t.Fatalf("MapNoteToLane C3 failed: %v", err)
	}
	if normalized != 48 || lane != 0 {
		t.Fatalf("C3 normalized/lane = %d/%d", normalized, lane)
	}

	normalized, lane, err = MapNoteToLane(83, cfg)
	if err != nil {
		t.Fatalf("MapNoteToLane B5 failed: %v", err)
	}
	if normalized != 83 || lane != 35 {
		t.Fatalf("B5 normalized/lane = %d/%d", normalized, lane)
	}
}

func TestMapNoteToLaneWithTransposeAndOctaveShift(t *testing.T) {
	cfg := DefaultMidiConfig()
	cfg.Transpose = 2
	cfg.OctaveShift = -1

	normalized, lane, err := MapNoteToLane(60, cfg)
	if err != nil {
		t.Fatalf("MapNoteToLane failed: %v", err)
	}
	if normalized != 50 || lane != 2 {
		t.Fatalf("normalized/lane = %d/%d, want 50/2", normalized, lane)
	}
}

func TestMapNoteToLaneRejectsOutOfRange(t *testing.T) {
	cfg := DefaultMidiConfig()
	if normalized, lane, err := MapNoteToLane(47, cfg); !errors.Is(err, ErrLaneOutOfRange) {
		t.Fatalf("low note result normalized/lane/err = %d/%d/%v", normalized, lane, err)
	}
	if normalized, lane, err := MapNoteToLane(84, cfg); !errors.Is(err, ErrLaneOutOfRange) {
		t.Fatalf("high note result normalized/lane/err = %d/%d/%v", normalized, lane, err)
	}
}

func TestMapNoteWithPolicyDrop(t *testing.T) {
	cfg := DefaultMidiConfig()
	result, err := MapNoteWithPolicy(84, cfg)
	if err != nil {
		t.Fatalf("MapNoteWithPolicy drop failed: %v", err)
	}
	if !result.Dropped || result.Folded || result.Clamped {
		t.Fatalf("drop flags = %+v", result)
	}
	if result.NormalizedNote != 84 || result.RawLane != 36 || result.Lane != 36 || result.Policy != OutOfRangeDrop {
		t.Fatalf("drop result = %+v", result)
	}
}

func TestMapNoteWithPolicyOctaveFold(t *testing.T) {
	cfg := DefaultMidiConfig()
	cfg.OutOfRangePolicy = OutOfRangeOctaveFold

	low, err := MapNoteWithPolicy(35, cfg)
	if err != nil {
		t.Fatalf("MapNoteWithPolicy low fold failed: %v", err)
	}
	if !low.Folded || low.Dropped || low.Clamped || low.RawLane != -13 || low.Lane != 11 {
		t.Fatalf("low fold result = %+v", low)
	}

	high, err := MapNoteWithPolicy(96, cfg)
	if err != nil {
		t.Fatalf("MapNoteWithPolicy high fold failed: %v", err)
	}
	if !high.Folded || high.RawLane != 48 || high.Lane != 0 {
		t.Fatalf("high fold result = %+v", high)
	}
}

func TestMapNoteWithPolicyNearest(t *testing.T) {
	cfg := DefaultMidiConfig()
	cfg.OutOfRangePolicy = OutOfRangeNearest

	low, err := MapNoteWithPolicy(47, cfg)
	if err != nil {
		t.Fatalf("MapNoteWithPolicy low nearest failed: %v", err)
	}
	if !low.Clamped || low.Dropped || low.Folded || low.RawLane != -1 || low.Lane != 0 {
		t.Fatalf("low nearest result = %+v", low)
	}

	high, err := MapNoteWithPolicy(84, cfg)
	if err != nil {
		t.Fatalf("MapNoteWithPolicy high nearest failed: %v", err)
	}
	if !high.Clamped || high.RawLane != 36 || high.Lane != 35 {
		t.Fatalf("high nearest result = %+v", high)
	}
}

func TestKey36MappingFromRowsValidatesCompleteness(t *testing.T) {
	rows := defaultKeymapRowsForTest()
	mapping, err := Key36MappingFromRows(storage.DefaultKeymapProfileID, rows)
	if err != nil {
		t.Fatalf("Key36MappingFromRows failed: %v", err)
	}
	if mapping.ProfileID != storage.DefaultKeymapProfileID || mapping.ProfileName == "" {
		t.Fatalf("unexpected mapping header: %+v", mapping)
	}
	if mapping.Lanes[0].Lane != 0 || mapping.Lanes[35].Lane != 35 {
		t.Fatalf("unexpected mapping lanes: %+v %+v", mapping.Lanes[0], mapping.Lanes[35])
	}

	if _, err := Key36MappingFromRows(storage.DefaultKeymapProfileID, rows[:35]); !errors.Is(err, ErrKeymapIncomplete) {
		t.Fatalf("missing lane error = %v", err)
	}
	duplicate := append([]storage.Keymap36{}, rows...)
	duplicate[35].Lane = 0
	if _, err := Key36MappingFromRows(storage.DefaultKeymapProfileID, duplicate); !errors.Is(err, ErrKeymapIncomplete) {
		t.Fatalf("duplicate lane error = %v", err)
	}
	invalid := append([]storage.Keymap36{}, rows...)
	invalid[0].Lane = -1
	if _, err := Key36MappingFromRows(storage.DefaultKeymapProfileID, invalid); !errors.Is(err, ErrKeymapIncomplete) {
		t.Fatalf("invalid lane error = %v", err)
	}
}

func TestLoadKey36MappingLoadsDefaultKeymap(t *testing.T) {
	db := openTestDB(t, "load_keymap.json")
	mapping, err := LoadKey36Mapping(db.Store, storage.DefaultKeymapProfileID)
	if err != nil {
		t.Fatalf("LoadKey36Mapping failed: %v", err)
	}
	if mapping.ProfileID != storage.DefaultKeymapProfileID {
		t.Fatalf("profile id = %d", mapping.ProfileID)
	}
	if mapping.Lanes[0].Lane != 0 || mapping.Lanes[35].Lane != 35 {
		t.Fatalf("default lanes not loaded: %+v %+v", mapping.Lanes[0], mapping.Lanes[35])
	}
}

func TestMapEventToLaneKeySeparatesPitchAndPhysicalKeymap(t *testing.T) {
	rows := defaultKeymapRowsForTest()
	mapping, err := Key36MappingFromRows(storage.DefaultKeymapProfileID, rows)
	if err != nil {
		t.Fatalf("Key36MappingFromRows failed: %v", err)
	}

	cfg := DefaultMidiConfig()
	mapped, err := MapEventToLaneKey(storage.MidiEvent{Note: 60}, cfg, mapping)
	if err != nil {
		t.Fatalf("MapEventToLaneKey failed: %v", err)
	}
	if mapped.SourceNote != 60 || mapped.NormalizedNote != 60 || mapped.Lane != 12 {
		t.Fatalf("mapped note = %+v", mapped)
	}
	if mapped.Key.Label != mapping.Lanes[12].Label || mapped.Key.VirtualKey != mapping.Lanes[12].VirtualKey {
		t.Fatalf("mapped key = %+v, want lane 12 %+v", mapped.Key, mapping.Lanes[12])
	}

	cfg.BaseNote = 47
	shifted, err := MapEventToLaneKey(storage.MidiEvent{Note: 60}, cfg, mapping)
	if err != nil {
		t.Fatalf("MapEventToLaneKey shifted failed: %v", err)
	}
	if shifted.Lane != 13 {
		t.Fatalf("shifted lane = %d, want 13", shifted.Lane)
	}
	if shifted.Key.VirtualKey != mapping.Lanes[13].VirtualKey {
		t.Fatalf("shifted physical key = %+v, want lane 13 %+v", shifted.Key, mapping.Lanes[13])
	}
	if mapping.Lanes[12].VirtualKey == 0 || mapping.Lanes[13].VirtualKey == 0 {
		t.Fatalf("physical keymap was mutated or incomplete")
	}
}

func TestMapEventToLaneKeyAppliesOutOfRangePolicy(t *testing.T) {
	mapping, err := Key36MappingFromRows(storage.DefaultKeymapProfileID, defaultKeymapRowsForTest())
	if err != nil {
		t.Fatalf("Key36MappingFromRows failed: %v", err)
	}

	cfg := DefaultMidiConfig()
	dropped, err := MapEventToLaneKey(storage.MidiEvent{Note: 84}, cfg, mapping)
	if err != nil {
		t.Fatalf("drop MapEventToLaneKey failed: %v", err)
	}
	if !dropped.Dropped || dropped.Key.VirtualKey != 0 {
		t.Fatalf("dropped event should not have key: %+v", dropped)
	}

	cfg.OutOfRangePolicy = OutOfRangeOctaveFold
	folded, err := MapEventToLaneKey(storage.MidiEvent{Note: 96}, cfg, mapping)
	if err != nil {
		t.Fatalf("fold MapEventToLaneKey failed: %v", err)
	}
	if !folded.Folded || folded.Lane != 0 || folded.Key.VirtualKey != mapping.Lanes[0].VirtualKey {
		t.Fatalf("folded event = %+v", folded)
	}

	cfg.OutOfRangePolicy = OutOfRangeNearest
	clamped, err := MapEventToLaneKey(storage.MidiEvent{Note: 84}, cfg, mapping)
	if err != nil {
		t.Fatalf("nearest MapEventToLaneKey failed: %v", err)
	}
	if !clamped.Clamped || clamped.Lane != 35 || clamped.Key.VirtualKey != mapping.Lanes[35].VirtualKey {
		t.Fatalf("clamped event = %+v", clamped)
	}
}

func TestMappingStatsFromResults(t *testing.T) {
	results := []LaneMappingDTO{
		{RawLane: 0, Lane: 0},
		{RawLane: 36, Lane: 36, Dropped: true},
		{RawLane: 48, Lane: 0, Folded: true},
		{RawLane: -2, Lane: 0, Clamped: true},
	}
	stats := MappingStatsFromResults(results)
	if stats.TotalNotes != 4 || stats.PlayableNotes != 3 {
		t.Fatalf("stats totals = %+v", stats)
	}
	if stats.OutOfRangeCount != 3 || stats.DroppedCount != 1 || stats.FoldedCount != 1 || stats.ClampedCount != 1 {
		t.Fatalf("stats policy counts = %+v", stats)
	}
	if stats.MinLane != 0 || stats.MaxLane != 0 {
		t.Fatalf("stats range = %+v", stats)
	}
}

func defaultKeymapRowsForTest() []storage.Keymap36 {
	rows := make([]storage.Keymap36, 0, LaneCount)
	for lane := 0; lane < LaneCount; lane++ {
		rows = append(rows, storage.Keymap36{
			ID:               uint(lane + 1),
			ProfileID:        storage.DefaultKeymapProfileID,
			ProfileName:      storage.DefaultKeymapProfileName,
			Lane:             lane,
			Label:            "lane",
			PitchClass:       lane % 12,
			IsBlackKey:       isBlackMidiNote(lane),
			VirtualKey:       100 + lane,
			ScanCode:         200 + lane,
			ModifierKeysJSON: "[]",
			DisplayOrder:     lane,
		})
	}
	return rows
}

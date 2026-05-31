package storage

import (
	"path/filepath"
	"testing"
)

func TestAllModelsRegistersMidiCollections(t *testing.T) {
	expected := map[string]struct {
		labelKey  string
		clearable bool
	}{
		MidiProjectsTable:   {labelKey: "midiProjects", clearable: false},
		MidiEventsTable:     {labelKey: "midiEvents", clearable: false},
		MidiProfilesTable:   {labelKey: "midiProfiles", clearable: false},
		Keymap36Table:       {labelKey: "keymap36", clearable: false},
		PlayHistoryTable:    {labelKey: "playHistory", clearable: true},
		HotkeyBindingsTable: {labelKey: "hotkeyBindings", clearable: false},
	}

	seen := make(map[string]ModelDescriptor, len(AllModels))
	for _, desc := range AllModels {
		if desc.Model == nil {
			t.Fatalf("model for collection %q is nil", desc.TableName)
		}
		if desc.TableName == "" || desc.LabelKey == "" {
			t.Fatalf("invalid descriptor: %+v", desc)
		}
		if _, ok := seen[desc.TableName]; ok {
			t.Fatalf("duplicate descriptor for %q", desc.TableName)
		}
		seen[desc.TableName] = desc
		if got := FindDescriptor(desc.TableName); got == nil {
			t.Fatalf("FindDescriptor(%q) returned nil", desc.TableName)
		}
	}

	for tableName, want := range expected {
		desc, ok := seen[tableName]
		if !ok {
			t.Fatalf("missing collection %q in AllModels", tableName)
		}
		if desc.LabelKey != want.labelKey || desc.Clearable != want.clearable {
			t.Fatalf("descriptor %q = %+v, want label=%q clearable=%v", tableName, desc, want.labelKey, want.clearable)
		}
	}
}

func TestOpenSeedsDefaultMidiStateIdempotently(t *testing.T) {
	db := openStorageTestDB(t, "defaults.json")

	profile, ok := db.Store.GetGlobalDefaultProfile()
	if !ok {
		t.Fatalf("default profile missing")
	}
	if profile.ID != DefaultMidiProfileID || profile.Name != DefaultMidiProfileName {
		t.Fatalf("default profile = %+v", profile)
	}
	if profile.BaseNote != DefaultMidiBaseNote || profile.Transpose != DefaultMidiTranspose || profile.OctaveShift != DefaultMidiOctaveShift {
		t.Fatalf("default pitch config = %+v", profile)
	}
	if profile.Speed != DefaultMidiSpeed || profile.OutOfRangePolicy != DefaultOutOfRangePolicy {
		t.Fatalf("default playback config = %+v", profile)
	}
	if profile.MinPressMs != DefaultMinPressMs || profile.ReleaseGapMs != DefaultReleaseGapMs || profile.KeymapProfileID != DefaultKeymapProfileID {
		t.Fatalf("default timing/keymap config = %+v", profile)
	}

	keymap := db.Store.ListKeymapProfile(DefaultKeymapProfileID)
	if len(keymap) != 36 {
		t.Fatalf("default keymap rows = %d, want 36", len(keymap))
	}
	for lane, row := range keymap {
		if row.Lane != lane || row.PitchClass != lane%12 || row.DisplayOrder != lane || row.ProfileName != DefaultKeymapProfileName {
			t.Fatalf("keymap row %d = %+v", lane, row)
		}
	}
	assertDefaultLane(t, keymap[0], "C3", false, 90, 44, modifierKeysNone)
	assertDefaultLane(t, keymap[1], "C#3", true, 90, 44, modifierKeysShift)
	assertDefaultLane(t, keymap[12], "C4", false, 65, 30, modifierKeysNone)
	assertDefaultLane(t, keymap[25], "C#5", true, 81, 16, modifierKeysShift)
	assertDefaultLane(t, keymap[35], "B5", false, 85, 22, modifierKeysNone)

	if err := ensureDefaultMidiState(db.Store); err != nil {
		t.Fatalf("ensureDefaultMidiState() second run failed: %v", err)
	}
	if got := db.Store.CountGlobalDefaultProfiles(); got != 1 {
		t.Fatalf("default profile count after second run = %d, want 1", got)
	}
	if got := db.Store.CountKeymapRows(DefaultKeymapProfileID); got != 36 {
		t.Fatalf("default keymap count after second run = %d, want 36", got)
	}

	if err := db.Store.UpdateKeymapLane(DefaultKeymapProfileID, 0, func(row *Keymap36) { row.ScanCode = 999 }); err != nil {
		t.Fatalf("update default keymap lane failed: %v", err)
	}
	if err := ensureDefaultMidiState(db.Store); err != nil {
		t.Fatalf("ensureDefaultMidiState() after calibration failed: %v", err)
	}
	calibrated := db.Store.ListKeymapProfile(DefaultKeymapProfileID)[0]
	if calibrated.ScanCode != 999 {
		t.Fatalf("default keymap lane was overwritten, scan code = %d, want 999", calibrated.ScanCode)
	}

	const customProfileName = "User Calibrated Profile"
	profile.Name = customProfileName
	if _, err := db.Store.SaveProfile(profile); err != nil {
		t.Fatalf("rename default profile failed: %v", err)
	}
	if err := ensureDefaultMidiState(db.Store); err != nil {
		t.Fatalf("ensureDefaultMidiState() after profile edit failed: %v", err)
	}
	renamed, _ := db.Store.GetGlobalDefaultProfile()
	if renamed.Name != customProfileName {
		t.Fatalf("default profile name was overwritten = %q, want %q", renamed.Name, customProfileName)
	}
	if got := db.Store.CountGlobalDefaultProfiles(); got != 1 {
		t.Fatalf("total default profile count after rename = %d, want 1", got)
	}
}

func TestOpenCreatesJSONFile(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "state.json")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	defer db.Close()
	if size, err := FileSize(dbPath); err != nil || size <= 0 {
		t.Fatalf("FileSize() = %d, %v", size, err)
	}
}

func assertDefaultLane(t *testing.T, row Keymap36, label string, blackKey bool, virtualKey, scanCode int, modifiers string) {
	t.Helper()
	if row.Label != label || row.IsBlackKey != blackKey || row.VirtualKey != virtualKey || row.ScanCode != scanCode || row.ModifierKeysJSON != modifiers {
		t.Fatalf("lane %d = %+v", row.Lane, row)
	}
}

func openStorageTestDB(t *testing.T, filename string) *DB {
	t.Helper()
	db, err := Open(filepath.Join(t.TempDir(), filename))
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Fatalf("Close() failed: %v", err)
		}
	})
	return db
}

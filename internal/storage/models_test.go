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
	assertDefaultLane(t, keymap[3], "D#3", true, 67, 46, modifierKeysCtrl)
	assertDefaultLane(t, keymap[6], "F#3", true, 86, 47, modifierKeysShift)
	assertDefaultLane(t, keymap[10], "A#3", true, 77, 50, modifierKeysCtrl)
	assertDefaultLane(t, keymap[12], "C4", false, 65, 30, modifierKeysNone)
	assertDefaultLane(t, keymap[25], "C#5", true, 81, 16, modifierKeysShift)
	assertDefaultLane(t, keymap[27], "D#5", true, 69, 18, modifierKeysCtrl)
	assertDefaultLane(t, keymap[34], "A#5", true, 85, 22, modifierKeysCtrl)
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

func TestDefaultKeymapUsesGameSemitoneModifierGroups(t *testing.T) {
	rows := defaultKeymap36Rows()
	byLane := keymapRowsByLane(rows)

	assertDefaultLane(t, byLane[25], "C#5", true, 81, 16, modifierKeysShift) // Shift+Q
	assertDefaultLane(t, byLane[13], "C#4", true, 65, 30, modifierKeysShift) // Shift+A
	assertDefaultLane(t, byLane[1], "C#3", true, 90, 44, modifierKeysShift)  // Shift+Z
	assertDefaultLane(t, byLane[30], "F#5", true, 82, 19, modifierKeysShift) // Shift+R
	assertDefaultLane(t, byLane[18], "F#4", true, 70, 33, modifierKeysShift) // Shift+F
	assertDefaultLane(t, byLane[6], "F#3", true, 86, 47, modifierKeysShift)  // Shift+V
	assertDefaultLane(t, byLane[32], "G#5", true, 84, 20, modifierKeysShift) // Shift+T
	assertDefaultLane(t, byLane[20], "G#4", true, 71, 34, modifierKeysShift) // Shift+G
	assertDefaultLane(t, byLane[8], "G#3", true, 66, 48, modifierKeysShift)  // Shift+B

	assertDefaultLane(t, byLane[27], "D#5", true, 69, 18, modifierKeysCtrl) // Ctrl+E
	assertDefaultLane(t, byLane[15], "D#4", true, 68, 32, modifierKeysCtrl) // Ctrl+D
	assertDefaultLane(t, byLane[3], "D#3", true, 67, 46, modifierKeysCtrl)  // Ctrl+C
	assertDefaultLane(t, byLane[34], "A#5", true, 85, 22, modifierKeysCtrl) // Ctrl+U
	assertDefaultLane(t, byLane[22], "A#4", true, 74, 36, modifierKeysCtrl) // Ctrl+J
	assertDefaultLane(t, byLane[10], "A#3", true, 77, 50, modifierKeysCtrl) // Ctrl+M
}

func TestEnsureDefaultMidiStateMigratesLegacySemitoneDefaults(t *testing.T) {
	db := openStorageTestDB(t, "legacy_semitone_defaults.json")
	if err := db.Store.db().Where("profile_id = ?", DefaultKeymapProfileID).Delete(&Keymap36{}).Error; err != nil {
		t.Fatalf("clear default keymap failed: %v", err)
	}
	rows := legacyDefaultKeymap36Rows()
	now := nowMillis()
	for i := range rows {
		rows[i].CreatedAt = now
		rows[i].UpdatedAt = now
	}
	if err := db.Store.db().Create(&rows).Error; err != nil {
		t.Fatalf("seed legacy keymap failed: %v", err)
	}

	if err := ensureDefaultMidiState(db.Store); err != nil {
		t.Fatalf("ensureDefaultMidiState() failed: %v", err)
	}
	keymap := db.Store.ListKeymapProfile(DefaultKeymapProfileID)
	assertDefaultLane(t, keymap[3], "D#3", true, 67, 46, modifierKeysCtrl)
	assertDefaultLane(t, keymap[10], "A#3", true, 77, 50, modifierKeysCtrl)
	assertDefaultLane(t, keymap[27], "D#5", true, 69, 18, modifierKeysCtrl)
	assertDefaultLane(t, keymap[34], "A#5", true, 85, 22, modifierKeysCtrl)
}

func TestEnsureDefaultMidiStateDoesNotOverwriteCustomizedSemitoneLane(t *testing.T) {
	db := openStorageTestDB(t, "customized_semitone_defaults.json")
	if err := db.Store.db().Where("profile_id = ?", DefaultKeymapProfileID).Delete(&Keymap36{}).Error; err != nil {
		t.Fatalf("clear default keymap failed: %v", err)
	}
	rows := legacyDefaultKeymap36Rows()
	now := nowMillis()
	for i := range rows {
		rows[i].CreatedAt = now
		rows[i].UpdatedAt = now
		if rows[i].Lane == 3 {
			rows[i].ScanCode = 999
		}
	}
	if err := db.Store.db().Create(&rows).Error; err != nil {
		t.Fatalf("seed customized legacy keymap failed: %v", err)
	}

	if err := ensureDefaultMidiState(db.Store); err != nil {
		t.Fatalf("ensureDefaultMidiState() failed: %v", err)
	}
	keymap := db.Store.ListKeymapProfile(DefaultKeymapProfileID)
	assertDefaultLane(t, keymap[3], "D#3", true, 88, 999, modifierKeysShift)
	assertDefaultLane(t, keymap[10], "A#3", true, 77, 50, modifierKeysCtrl)
}

func TestOpenCreatesDBFile(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "state.db")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	defer db.Close()
	if size, err := FileSize(dbPath); err != nil || size <= 0 {
		t.Fatalf("FileSize() = %d, %v", size, err)
	}
}

func TestDeleteProjectsBatchRemovesRelatedRowsOnce(t *testing.T) {
	db := openStorageTestDB(t, "batch_delete.json")
	store := db.Store
	first, err := store.ImportProject(ProjectImportData{
		Project: MidiProject{DisplayName: "First", FileName: "first.mid", FileHash: "sha256:first", PPQ: 480},
		Events:  []MidiEvent{{Track: 0, Channel: 0, Note: 60, Velocity: 90, StartMs: 0, DurationMs: 100}},
	})
	if err != nil {
		t.Fatalf("first import failed: %v", err)
	}
	second, err := store.ImportProject(ProjectImportData{
		Project: MidiProject{DisplayName: "Second", FileName: "second.mid", FileHash: "sha256:second", PPQ: 480},
		Events:  []MidiEvent{{Track: 0, Channel: 0, Note: 62, Velocity: 80, StartMs: 0, DurationMs: 100}},
	})
	if err != nil {
		t.Fatalf("second import failed: %v", err)
	}
	profileProjectID := first.ID
	profile, err := store.SaveProfile(MidiProfile{ProjectID: &profileProjectID, Name: "First profile", BaseNote: DefaultMidiBaseNote, Speed: DefaultMidiSpeed, OutOfRangePolicy: DefaultOutOfRangePolicy, KeymapProfileID: DefaultKeymapProfileID})
	if err != nil {
		t.Fatalf("save profile failed: %v", err)
	}
	if _, err := store.AddPlayHistory(PlayHistory{ProjectID: first.ID, ProfileID: profile.ID, StartedAt: 1, DurationMs: 100, Completed: true}); err != nil {
		t.Fatalf("add history failed: %v", err)
	}

	results, err := store.DeleteProjectsBatch([]uint{0, first.ID, first.ID, 999})
	if err != nil {
		t.Fatalf("DeleteProjectsBatch failed: %v", err)
	}
	if len(results) != 4 {
		t.Fatalf("results = %d, want 4", len(results))
	}
	if results[0].Deleted || results[0].Reason != ProjectDeleteReasonInvalidID {
		t.Fatalf("invalid id result = %+v", results[0])
	}
	if !results[1].Deleted || results[1].Project.ID != first.ID || results[1].Project.DisplayName != first.DisplayName {
		t.Fatalf("deleted result = %+v", results[1])
	}
	if results[2].Deleted || results[2].Reason != ProjectDeleteReasonDuplicate {
		t.Fatalf("duplicate result = %+v", results[2])
	}
	if results[3].Deleted || results[3].Reason != ProjectDeleteReasonNotFound {
		t.Fatalf("missing result = %+v", results[3])
	}
	if _, ok := store.GetProject(first.ID); ok {
		t.Fatalf("first project still exists")
	}
	if _, ok := store.GetProject(second.ID); !ok {
		t.Fatalf("second project was deleted")
	}
	if got := store.CountEventsByProject(first.ID); got != 0 {
		t.Fatalf("first events = %d, want 0", got)
	}
	if got := store.CountProjectProfiles(first.ID); got != 0 {
		t.Fatalf("first profiles = %d, want 0", got)
	}
	if got := store.CountHistoryByProject(first.ID); got != 0 {
		t.Fatalf("first history = %d, want 0", got)
	}
	if got := store.CountEventsByProject(second.ID); got != 1 {
		t.Fatalf("second events = %d, want 1", got)
	}
	if got := store.CountGlobalDefaultProfiles(); got != 1 {
		t.Fatalf("global default profiles = %d, want 1", got)
	}
}

func TestImportProjectsBatchReturnsPerInputResultsAndHashIndex(t *testing.T) {
	db := openStorageTestDB(t, "batch_import.json")
	store := db.Store
	seed, err := store.ImportProject(ProjectImportData{
		Project: MidiProject{DisplayName: "Seed", FileName: "seed.mid", FileHash: "sha256:seed", PPQ: 480},
		Events:  []MidiEvent{{Track: 0, Channel: 0, Note: 60, Velocity: 90, StartMs: 0, DurationMs: 100}},
	})
	if err != nil {
		t.Fatalf("seed import failed: %v", err)
	}

	index := store.ProjectHashIndex()
	if got, ok := index["sha256:seed"]; !ok || got.ID != seed.ID || got.DisplayName != seed.DisplayName {
		t.Fatalf("ProjectHashIndex seed = %+v, ok=%v", got, ok)
	}

	results, err := store.ImportProjectsBatch([]ProjectImportData{
		{
			Project: MidiProject{DisplayName: "Alpha", FileName: "alpha.mid", FileHash: "sha256:alpha", PPQ: 480},
			Events:  []MidiEvent{{Track: 1, Channel: 0, Note: 62, Velocity: 80, StartMs: 10, DurationMs: 90}},
		},
		{
			Project: MidiProject{DisplayName: "Alpha Duplicate", FileName: "alpha-copy.mid", FileHash: "sha256:alpha", PPQ: 480},
			Events:  []MidiEvent{{Track: 2, Channel: 0, Note: 64, Velocity: 70, StartMs: 20, DurationMs: 80}},
		},
		{
			Project: MidiProject{DisplayName: "Seed Duplicate", FileName: "seed-copy.mid", FileHash: "sha256:seed", PPQ: 480},
			Events:  []MidiEvent{{Track: 3, Channel: 0, Note: 65, Velocity: 60, StartMs: 30, DurationMs: 70}},
		},
	})
	if err != nil {
		t.Fatalf("ImportProjectsBatch failed: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("batch results = %d, want 3", len(results))
	}
	if results[0].Status != ProjectBatchImportStatusImported || results[0].Project.ID == 0 || results[0].Project.FileHash != "sha256:alpha" {
		t.Fatalf("result 0 = %+v", results[0])
	}
	if results[1].Status != ProjectBatchImportStatusSkipped || results[1].Reason != ProjectBatchImportReasonDuplicateInLibrary || results[1].Project.ID != results[0].Project.ID {
		t.Fatalf("result 1 = %+v, imported=%+v", results[1], results[0])
	}
	if results[2].Status != ProjectBatchImportStatusSkipped || results[2].Project.ID != seed.ID {
		t.Fatalf("result 2 = %+v, seed=%+v", results[2], seed)
	}
	if got := store.CountProjects(); got != 2 {
		t.Fatalf("project count = %d, want 2", got)
	}
	if got := store.CountEventsByProject(results[0].Project.ID); got != 1 {
		t.Fatalf("alpha event count = %d, want 1", got)
	}
	if got := store.CountEventsByProject(seed.ID); got != 1 {
		t.Fatalf("seed event count = %d, want 1", got)
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

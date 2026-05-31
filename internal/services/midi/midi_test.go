package midi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"YyslsPlayer/internal/storage"
)

func TestReadMidiFileComputesHashAndValidatesInput(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.mid")
	data := minimalMidiBytes()
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write midi fixture failed: %v", err)
	}

	file, err := readMidiFile(path)
	if err != nil {
		t.Fatalf("readMidiFile failed: %v", err)
	}
	if file.Name != "sample.mid" {
		t.Fatalf("file name = %q", file.Name)
	}
	if file.Size != int64(len(data)) {
		t.Fatalf("file size = %d, want %d", file.Size, len(data))
	}
	if string(file.Bytes) != string(data) {
		t.Fatalf("file bytes mismatch")
	}
	wantHash := sha256Hex(data)
	if file.SHA256 != wantHash {
		t.Fatalf("sha256 = %q, want %q", file.SHA256, wantHash)
	}
	if file.FileHash != "sha256:"+file.SHA256 {
		t.Fatalf("file hash = %q", file.FileHash)
	}

	if _, err := readMidiFile(filepath.Join(dir, "missing.mid")); !errors.Is(err, ErrMidiFileNotFound) {
		t.Fatalf("missing file error = %v", err)
	}
	unsupportedPath := filepath.Join(dir, "sample.txt")
	if err := os.WriteFile(unsupportedPath, data, 0o644); err != nil {
		t.Fatalf("write unsupported fixture failed: %v", err)
	}
	if _, err := readMidiFile(unsupportedPath); !errors.Is(err, ErrMidiUnsupportedFormat) {
		t.Fatalf("unsupported format error = %v", err)
	}
	largePath := filepath.Join(dir, "large.mid")
	large, err := os.Create(largePath)
	if err != nil {
		t.Fatalf("create large fixture failed: %v", err)
	}
	if err := large.Truncate(maxMidiFileSizeBytes + 1); err != nil {
		t.Fatalf("truncate large fixture failed: %v", err)
	}
	if err := large.Close(); err != nil {
		t.Fatalf("close large fixture failed: %v", err)
	}
	if _, err := readMidiFile(largePath); !errors.Is(err, ErrMidiTooLarge) {
		t.Fatalf("too large error = %v", err)
	}
}

func TestImportFileCreatesProjectAndDeduplicatesByHash(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t, "import_file.json")
	svc := New(storage.NewHolder(db))

	path := filepath.Join(t.TempDir(), "dedupe.MIDI")
	if err := os.WriteFile(path, minimalMidiBytes(), 0o644); err != nil {
		t.Fatalf("write midi fixture failed: %v", err)
	}

	first, err := svc.ImportFile(ctx, path)
	if err != nil {
		t.Fatalf("ImportFile first call failed: %v", err)
	}
	if first.Project.ID == 0 {
		t.Fatalf("imported project id is zero")
	}
	if first.Project.DisplayName != "dedupe" {
		t.Fatalf("display name = %q, want dedupe", first.Project.DisplayName)
	}
	if first.Project.FileName != "dedupe.MIDI" {
		t.Fatalf("file name = %q", first.Project.FileName)
	}
	wantHash := "sha256:" + sha256Hex(minimalMidiBytes())
	if first.Project.FileHash != wantHash {
		t.Fatalf("file hash = %q, want %q", first.Project.FileHash, wantHash)
	}
	if first.Project.SourcePath == "" {
		t.Fatalf("source path is empty")
	}
	if first.Project.PPQ != 480 || first.Project.TrackCount != 1 || first.Project.ChannelCount != 1 {
		t.Fatalf("unexpected parsed metadata: %+v", first.Project)
	}
	if first.Project.NoteCount != 1 || first.EventCount != 1 || first.Project.DurationMs != 500 {
		t.Fatalf("unexpected parsed note summary: %+v", first)
	}
	if first.Project.FileSizeBytes != int64(len(minimalMidiBytes())) {
		t.Fatalf("file size bytes = %d, want %d", first.Project.FileSizeBytes, len(minimalMidiBytes()))
	}
	if first.DefaultProfile.ID == 0 || len(first.DefaultKeymap.Lanes) != 36 {
		t.Fatalf("default profile/keymap missing in detail: %+v", first)
	}
	if first.QualityReport.TotalNotes != 1 || first.QualityReport.NoteRange.Min != 60 || first.QualityReport.NoteRange.Max != 60 {
		t.Fatalf("unexpected quality report after import: %+v", first.QualityReport)
	}
	if first.QualityReport.ChordDensity != 1 || first.QualityReport.PlayableRatio != 1 {
		t.Fatalf("unexpected quality density/ratio: %+v", first.QualityReport)
	}

	second, err := svc.ImportFile(ctx, path)
	if err != nil {
		t.Fatalf("ImportFile second call failed: %v", err)
	}
	if second.Project.ID != first.Project.ID {
		t.Fatalf("dedupe returned project id %d, want %d", second.Project.ID, first.Project.ID)
	}
	count := db.Store.CountProjects()
	if count != 1 {
		t.Fatalf("project count = %d, want 1", count)
	}
}

func TestImportFilesReportsImportedSkippedAndFailed(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t, "import_files.json")
	svc := New(storage.NewHolder(db))
	dir := t.TempDir()

	existingPath := filepath.Join(dir, "existing.mid")
	existingDuplicatePath := filepath.Join(dir, "existing-copy.mid")
	firstPath := filepath.Join(dir, "first.mid")
	duplicatePath := filepath.Join(dir, "duplicate.midi")
	badPath := filepath.Join(dir, "bad.mid")
	if err := os.WriteFile(existingPath, minimalMidiBytes(), 0o644); err != nil {
		t.Fatalf("write existing fixture failed: %v", err)
	}
	if err := os.WriteFile(existingDuplicatePath, minimalMidiBytes(), 0o644); err != nil {
		t.Fatalf("write existing duplicate fixture failed: %v", err)
	}
	if err := os.WriteFile(firstPath, singleNoteMidiBytes(62), 0o644); err != nil {
		t.Fatalf("write first fixture failed: %v", err)
	}
	if err := os.WriteFile(duplicatePath, singleNoteMidiBytes(62), 0o644); err != nil {
		t.Fatalf("write duplicate fixture failed: %v", err)
	}
	if err := os.WriteFile(badPath, []byte("not midi"), 0o644); err != nil {
		t.Fatalf("write bad fixture failed: %v", err)
	}

	existing, err := svc.ImportFile(ctx, existingPath)
	if err != nil {
		t.Fatalf("seed existing import failed: %v", err)
	}

	result, err := svc.ImportFiles(ctx, []string{existingPath, existingDuplicatePath, firstPath, duplicatePath, badPath})
	if err != nil {
		t.Fatalf("ImportFiles failed: %v", err)
	}
	if result.TotalCount != 5 || len(result.Items) != 5 {
		t.Fatalf("unexpected result size: %+v", result)
	}
	if result.ImportedCount != 1 || result.SkippedCount != 3 || result.FailedCount != 1 {
		t.Fatalf("unexpected counters: %+v", result)
	}
	if result.Items[0].Status != importStatusSkipped || result.Items[0].Reason != importReasonDuplicateInLibrary || result.Items[0].ProjectID == nil || *result.Items[0].ProjectID != existing.Project.ID {
		t.Fatalf("existing duplicate item = %+v", result.Items[0])
	}
	if result.Items[1].Status != importStatusSkipped || result.Items[1].Reason != importReasonDuplicateInBatch || result.Items[1].ProjectID == nil || *result.Items[1].ProjectID != existing.Project.ID {
		t.Fatalf("existing batch duplicate item = %+v", result.Items[1])
	}
	if result.Items[2].Status != importStatusImported || result.Items[2].ProjectID == nil {
		t.Fatalf("first import item = %+v", result.Items[2])
	}
	importedProjectID := *result.Items[2].ProjectID
	if result.Items[3].Status != importStatusSkipped || result.Items[3].Reason != importReasonDuplicateInBatch || result.Items[3].ProjectID == nil || *result.Items[3].ProjectID != importedProjectID {
		t.Fatalf("batch duplicate item = %+v", result.Items[3])
	}
	if result.Items[4].Status != importStatusFailed || result.Items[4].Error == "" {
		t.Fatalf("failed item = %+v", result.Items[4])
	}
	if result.FirstProjectID == nil || *result.FirstProjectID != existing.Project.ID {
		t.Fatalf("first project id = %v, want %d", result.FirstProjectID, existing.Project.ID)
	}
	if result.FirstImportedProjectID == nil || *result.FirstImportedProjectID != importedProjectID {
		t.Fatalf("first imported project id = %v, want %d", result.FirstImportedProjectID, importedProjectID)
	}

	count := db.Store.CountProjects()
	if count != 2 {
		t.Fatalf("project count = %d, want 2", count)
	}
}

func TestImportDirectoryRecursivelyFindsMidiFiles(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t, "import_directory.json")
	svc := New(storage.NewHolder(db))
	dir := t.TempDir()
	nested := filepath.Join(dir, "nested")
	if err := os.Mkdir(nested, 0o755); err != nil {
		t.Fatalf("create nested dir failed: %v", err)
	}
	rootMidi := filepath.Join(dir, "a-root.mid")
	nestedMidi := filepath.Join(nested, "b-nested.MIDI")
	ignored := filepath.Join(nested, "ignored.txt")
	if err := os.WriteFile(rootMidi, singleNoteMidiBytes(61), 0o644); err != nil {
		t.Fatalf("write root midi failed: %v", err)
	}
	if err := os.WriteFile(nestedMidi, singleNoteMidiBytes(63), 0o644); err != nil {
		t.Fatalf("write nested midi failed: %v", err)
	}
	if err := os.WriteFile(ignored, minimalMidiBytes(), 0o644); err != nil {
		t.Fatalf("write ignored fixture failed: %v", err)
	}

	result, err := svc.ImportDirectory(ctx, dir)
	if err != nil {
		t.Fatalf("ImportDirectory failed: %v", err)
	}
	if result.TotalCount != 2 || result.ImportedCount != 2 || result.SkippedCount != 0 || result.FailedCount != 0 {
		t.Fatalf("unexpected directory result: %+v", result)
	}
	if len(result.Items) != 2 {
		t.Fatalf("directory items = %d, want 2", len(result.Items))
	}
	if result.Items[0].Path != rootMidi || result.Items[1].Path != nestedMidi {
		t.Fatalf("directory import order/items = %+v", result.Items)
	}

	count := db.Store.CountProjects()
	if count != 2 {
		t.Fatalf("project count = %d, want 2", count)
	}
}

func TestParseNormalizedScoreHandlesTempoChangesAndVelocityZeroNoteOff(t *testing.T) {
	score, err := parseNormalizedScore(tempoChangeMidiBytes())
	if err != nil {
		t.Fatalf("parseNormalizedScore failed: %v", err)
	}
	if score.PPQ != 480 {
		t.Fatalf("PPQ = %d, want 480", score.PPQ)
	}
	if score.TrackCount != 2 {
		t.Fatalf("track count = %d, want 2", score.TrackCount)
	}
	if score.ChannelCount != 2 {
		t.Fatalf("channel count = %d, want 2", score.ChannelCount)
	}
	if score.DurationMs != 1250 {
		t.Fatalf("duration = %d, want 1250", score.DurationMs)
	}
	if len(score.Events) != 2 {
		t.Fatalf("events = %d, want 2", len(score.Events))
	}
	first := score.Events[0]
	if first.Track != 1 || first.Channel != 0 || first.Note != 60 || first.Velocity != 90 || first.StartMs != 0 || first.DurationMs != 500 {
		t.Fatalf("event 0 = %+v", first)
	}
	second := score.Events[1]
	if second.Track != 1 || second.Channel != 1 || second.Note != 64 || second.Velocity != 70 || second.StartMs != 500 || second.DurationMs != 750 {
		t.Fatalf("event 1 = %+v", second)
	}
}

func TestParseNormalizedScoreRejectsEmptyMidi(t *testing.T) {
	if _, err := parseNormalizedScore(emptyMidiBytes()); !errors.Is(err, ErrMidiEmpty) {
		t.Fatalf("empty midi error = %v", err)
	}
}

func TestQualityReportFromEvents(t *testing.T) {
	project := storage.MidiProject{TrackCount: 2, ChannelCount: 2}
	events := []storage.MidiEvent{
		{Track: 0, Channel: 0, Note: 60, StartMs: 0, DurationMs: 100},
		{Track: 0, Channel: 0, Note: 61, StartMs: 50, DurationMs: 150},
		{Track: 1, Channel: 1, Note: 64, StartMs: 75, DurationMs: 25},
		{Track: 1, Channel: 1, Note: 70, StartMs: 200, DurationMs: 100},
	}

	report := qualityReportFromEvents(project, events)
	if report.NoteRange.Min != 60 || report.NoteRange.Max != 70 {
		t.Fatalf("note range = %+v", report.NoteRange)
	}
	if report.TotalNotes != 4 || report.PlayableNotes != 4 || report.PlayableRatio != 1 {
		t.Fatalf("note totals = %+v", report)
	}
	if report.BlackKeyCount != 2 {
		t.Fatalf("black key count = %d, want 2", report.BlackKeyCount)
	}
	if report.ChordDensity != 3 {
		t.Fatalf("chord density = %d, want 3", report.ChordDensity)
	}
	if report.TrackCount != 2 || report.ChannelCount != 2 {
		t.Fatalf("track/channel = %d/%d", report.TrackCount, report.ChannelCount)
	}
	if report.MappedRange.MinLane != -1 || report.MappedRange.MaxLane != -1 {
		t.Fatalf("mapped range before M3 = %+v", report.MappedRange)
	}
}

func TestQualityReportFromEventsWithConfigDropAndSuggestions(t *testing.T) {
	project := storage.MidiProject{TrackCount: 1, ChannelCount: 1}
	events := []storage.MidiEvent{
		{Track: 0, Channel: 0, Note: 72, StartMs: 0, DurationMs: 100},
		{Track: 0, Channel: 0, Note: 84, StartMs: 100, DurationMs: 100},
	}
	cfg := DefaultMidiConfig()
	report := qualityReportFromEventsWithConfig(project, events, cfg)
	if report.MappedRange.MinLane != 24 || report.MappedRange.MaxLane != 24 {
		t.Fatalf("mapped range = %+v", report.MappedRange)
	}
	if report.TotalNotes != 2 || report.PlayableNotes != 1 || report.OutOfRangeCount != 1 || report.DroppedCount != 1 {
		t.Fatalf("mapping counts = %+v", report)
	}
	if report.SuggestedTranspose != -1 || report.SuggestedOctaveShift != 0 {
		t.Fatalf("suggestions = transpose %d octave %d", report.SuggestedTranspose, report.SuggestedOctaveShift)
	}
	if len(report.Warnings) < 2 {
		t.Fatalf("warnings = %+v", report.Warnings)
	}
}

func TestQualityReportFromEventsWithConfigFoldAndNearest(t *testing.T) {
	project := storage.MidiProject{TrackCount: 1, ChannelCount: 1}
	events := []storage.MidiEvent{
		{Track: 0, Channel: 0, Note: 35, StartMs: 0, DurationMs: 100},
		{Track: 0, Channel: 0, Note: 96, StartMs: 100, DurationMs: 100},
	}
	cfg := DefaultMidiConfig()
	cfg.OutOfRangePolicy = OutOfRangeOctaveFold
	fold := qualityReportFromEventsWithConfig(project, events, cfg)
	if fold.FoldedCount != 2 || fold.DroppedCount != 0 || fold.PlayableNotes != 2 {
		t.Fatalf("fold report = %+v", fold)
	}
	if fold.MappedRange.MinLane != 0 || fold.MappedRange.MaxLane != 11 {
		t.Fatalf("fold mapped range = %+v", fold.MappedRange)
	}

	cfg.OutOfRangePolicy = OutOfRangeNearest
	nearest := qualityReportFromEventsWithConfig(project, events, cfg)
	if nearest.ClampedCount != 2 || nearest.PlayableNotes != 2 || nearest.DroppedCount != 0 {
		t.Fatalf("nearest report = %+v", nearest)
	}
	if nearest.MappedRange.MinLane != 0 || nearest.MappedRange.MaxLane != 35 {
		t.Fatalf("nearest mapped range = %+v", nearest.MappedRange)
	}
}

func TestQualityReportSuggestsLargeTransposeAndOctaveShift(t *testing.T) {
	project := storage.MidiProject{TrackCount: 1, ChannelCount: 1}
	events := []storage.MidiEvent{
		{Track: 0, Channel: 0, Note: 92, StartMs: 0, DurationMs: 100},
		{Track: 0, Channel: 0, Note: 95, StartMs: 100, DurationMs: 100},
	}
	report := qualityReportFromEventsWithConfig(project, events, DefaultMidiConfig())
	if report.SuggestedTranspose != -12 || report.SuggestedOctaveShift != -1 {
		t.Fatalf("suggestions = transpose %d octave %d", report.SuggestedTranspose, report.SuggestedOctaveShift)
	}
}

func TestQualityReportEmptyEvents(t *testing.T) {
	report := qualityReportFromEvents(storage.MidiProject{}, nil)
	if report.NoteRange.Min != -1 || report.NoteRange.Max != -1 {
		t.Fatalf("empty note range = %+v", report.NoteRange)
	}
	if report.TotalNotes != 0 || report.PlayableRatio != 0 || report.ChordDensity != 0 {
		t.Fatalf("empty report = %+v", report)
	}
}

func TestServiceListGetDeleteProject(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t, "midi_crud.json")
	svc := New(storage.NewHolder(db))

	sourcePath := `C:\scores\song.mid`
	project := storage.MidiProject{
		DisplayName:  "Song Alpha",
		FileName:     "song.mid",
		SourcePath:   &sourcePath,
		FileHash:     "hash-alpha",
		PPQ:          480,
		TrackCount:   2,
		ChannelCount: 1,
		DurationMs:   12345,
		NoteCount:    3,
	}
	var err error
	project, err = db.Store.SaveProject(project)
	if err != nil {
		t.Fatalf("create project failed: %v", err)
	}
	profile := storage.MidiProfile{
		ProjectID:        &project.ID,
		Name:             "Project profile",
		BaseNote:         50,
		Transpose:        2,
		OctaveShift:      1,
		Speed:            1.25,
		OutOfRangePolicy: "nearest",
		MinPressMs:       40,
		ReleaseGapMs:     20,
		KeymapProfileID:  storage.DefaultKeymapProfileID,
	}
	profile, err = db.Store.SaveProfile(profile)
	if err != nil {
		t.Fatalf("create project profile failed: %v", err)
	}
	if err := db.Store.UpdateProjectDefaultProfile(project.ID, profile.ID); err != nil {
		t.Fatalf("set default profile failed: %v", err)
	}
	project, _ = db.Store.GetProject(project.ID)
	if err := db.Store.AddEvents([]storage.MidiEvent{
		{ProjectID: project.ID, Track: 0, Channel: 0, Note: 60, Velocity: 90, StartMs: 0, DurationMs: 120},
		{ProjectID: project.ID, Track: 0, Channel: 0, Note: 64, Velocity: 88, StartMs: 250, DurationMs: 100},
		{ProjectID: project.ID, Track: 1, Channel: 1, Note: 67, Velocity: 80, StartMs: 500, DurationMs: 200},
	}); err != nil {
		t.Fatalf("create events failed: %v", err)
	}
	if _, err := db.Store.AddPlayHistory(storage.PlayHistory{
		ProjectID:  project.ID,
		ProfileID:  profile.ID,
		StartedAt:  1000,
		DurationMs: 12345,
		Completed:  true,
		DryRun:     true,
	}); err != nil {
		t.Fatalf("create play history failed: %v", err)
	}

	list, err := svc.ListProjects(ctx, ListProjectsRequest{Limit: 10, Query: "Alpha"})
	if err != nil {
		t.Fatalf("ListProjects failed: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("ListProjects count = %d, want 1", len(list))
	}
	if list[0].ID != project.ID || list[0].SourcePath != sourcePath || list[0].NoteCount != 3 {
		t.Fatalf("unexpected project summary: %+v", list[0])
	}
	if list[0].DefaultProfileID == nil || *list[0].DefaultProfileID != profile.ID {
		t.Fatalf("summary default profile id = %v, want %d", list[0].DefaultProfileID, profile.ID)
	}

	detail, err := svc.GetProject(ctx, project.ID)
	if err != nil {
		t.Fatalf("GetProject failed: %v", err)
	}
	if detail.Project.ID != project.ID {
		t.Fatalf("detail project id = %d, want %d", detail.Project.ID, project.ID)
	}
	if detail.DefaultProfile.ID != profile.ID {
		t.Fatalf("detail default profile id = %d, want %d", detail.DefaultProfile.ID, profile.ID)
	}
	if detail.EventCount != 3 {
		t.Fatalf("detail event count = %d, want 3", detail.EventCount)
	}
	if detail.PlayHistoryCount != 1 {
		t.Fatalf("detail history count = %d, want 1", detail.PlayHistoryCount)
	}
	if detail.DefaultKeymap.ProfileID != storage.DefaultKeymapProfileID {
		t.Fatalf("detail keymap profile id = %d", detail.DefaultKeymap.ProfileID)
	}
	if len(detail.DefaultKeymap.Lanes) != 36 {
		t.Fatalf("detail keymap lanes = %d, want 36", len(detail.DefaultKeymap.Lanes))
	}
	if detail.QualityReport.MappedRange.MinLane != 10 || detail.QualityReport.MappedRange.MaxLane != 17 {
		t.Fatalf("detail mapped range = %+v", detail.QualityReport.MappedRange)
	}
	if detail.QualityReport.SuggestedTranspose != 0 || detail.QualityReport.PlayableRatio != 1 {
		t.Fatalf("detail quality report = %+v", detail.QualityReport)
	}
	if len(detail.Profiles) < 2 {
		t.Fatalf("detail profiles = %d, want project profile plus global default", len(detail.Profiles))
	}

	if err := svc.DeleteProject(ctx, project.ID); err != nil {
		t.Fatalf("DeleteProject failed: %v", err)
	}
	assertProjectRowsDeleted(t, db, project.ID)
	assertDefaultRowsPreserved(t, db)
}

func TestServiceDeleteProjectsReportsBatchResult(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t, "midi_batch_delete.json")
	svc := New(storage.NewHolder(db))

	first, err := db.Store.SaveProject(storage.MidiProject{DisplayName: "First", FileName: "first.mid", FileHash: "hash-first", PPQ: 480})
	if err != nil {
		t.Fatalf("create first project failed: %v", err)
	}
	second, err := db.Store.SaveProject(storage.MidiProject{DisplayName: "Second", FileName: "second.mid", FileHash: "hash-second", PPQ: 480})
	if err != nil {
		t.Fatalf("create second project failed: %v", err)
	}
	if err := db.Store.AddEvents([]storage.MidiEvent{{ProjectID: first.ID, Track: 0, Channel: 0, Note: 60, Velocity: 90, StartMs: 0, DurationMs: 100}}); err != nil {
		t.Fatalf("add first events failed: %v", err)
	}
	if err := db.Store.AddEvents([]storage.MidiEvent{{ProjectID: second.ID, Track: 0, Channel: 0, Note: 62, Velocity: 90, StartMs: 0, DurationMs: 100}}); err != nil {
		t.Fatalf("add second events failed: %v", err)
	}

	result, err := svc.DeleteProjects(ctx, []uint{first.ID, 999, second.ID})
	if err != nil {
		t.Fatalf("DeleteProjects failed: %v", err)
	}
	if result.TotalCount != 3 || result.DeletedCount != 2 || result.FailedCount != 1 || len(result.Items) != 3 {
		t.Fatalf("unexpected batch delete result: %+v", result)
	}
	if result.Items[0].Status != projectBatchStatusDeleted || result.Items[0].DisplayName != first.DisplayName {
		t.Fatalf("first delete item = %+v", result.Items[0])
	}
	if result.Items[1].Status != projectBatchStatusFailed || result.Items[1].Reason == "" {
		t.Fatalf("missing delete item = %+v", result.Items[1])
	}
	if result.Items[2].Status != projectBatchStatusDeleted || result.Items[2].ProjectID != second.ID {
		t.Fatalf("second delete item = %+v", result.Items[2])
	}
	if got := db.Store.CountProjects(); got != 0 {
		t.Fatalf("project count = %d, want 0", got)
	}
	if got := db.Store.CountEventsByProject(first.ID) + db.Store.CountEventsByProject(second.ID); got != 0 {
		t.Fatalf("deleted event count = %d, want 0", got)
	}
}

func TestServiceUpdateProfileCreatesProjectScopedCopy(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t, "midi_update_profile.json")
	svc := New(storage.NewHolder(db))

	project := storage.MidiProject{DisplayName: "Song", FileName: "song.mid", FileHash: "hash-update", PPQ: 480}
	project, err := db.Store.SaveProject(project)
	if err != nil {
		t.Fatalf("create project failed: %v", err)
	}

	global, err := loadDefaultProfile(db.Store, nil, nil)
	if err != nil {
		t.Fatalf("load default profile failed: %v", err)
	}
	input := profileDTO(global)
	input.ProjectID = &project.ID
	input.Name = "Song profile"
	input.BaseNote = 52
	input.Transpose = -3
	input.Speed = 1.5
	input.OutOfRangePolicy = OutOfRangeNearest

	updated, err := svc.UpdateProfile(ctx, input)
	if err != nil {
		t.Fatalf("UpdateProfile failed: %v", err)
	}
	if updated.ID == global.ID || updated.ProjectID == nil || *updated.ProjectID != project.ID {
		t.Fatalf("expected project scoped copy, got %+v", updated)
	}
	if updated.BaseNote != 52 || updated.Transpose != -3 || updated.Speed != 1.5 || updated.OutOfRangePolicy != OutOfRangeNearest {
		t.Fatalf("updated config = %+v", updated)
	}

	reloaded, ok := db.Store.GetProject(project.ID)
	if !ok {
		t.Fatalf("reload project failed")
	}
	if reloaded.DefaultProfileID == nil || *reloaded.DefaultProfileID != updated.ID {
		t.Fatalf("default profile id = %v, want %d", reloaded.DefaultProfileID, updated.ID)
	}

	updated.Transpose = 4
	again, err := svc.UpdateProfile(ctx, updated)
	if err != nil {
		t.Fatalf("UpdateProfile second save failed: %v", err)
	}
	if again.ID != updated.ID || again.Transpose != 4 {
		t.Fatalf("expected in-place update, got %+v", again)
	}
}

func TestServiceUsesCurrentHolderDatabase(t *testing.T) {
	ctx := context.Background()
	first := openTestDB(t, "first.json")
	second := openTestDB(t, "second.json")
	holder := storage.NewHolder(first)
	svc := New(holder)

	if _, err := first.Store.SaveProject(storage.MidiProject{
		DisplayName: "First DB",
		FileName:    "first.mid",
		FileHash:    "hash-first",
		PPQ:         480,
	}); err != nil {
		t.Fatalf("create first project failed: %v", err)
	}
	if _, err := second.Store.SaveProject(storage.MidiProject{
		DisplayName: "Second DB",
		FileName:    "second.mid",
		FileHash:    "hash-second",
		PPQ:         480,
	}); err != nil {
		t.Fatalf("create second project failed: %v", err)
	}

	before, err := svc.ListProjects(ctx, ListProjectsRequest{Limit: 10})
	if err != nil {
		t.Fatalf("ListProjects before swap failed: %v", err)
	}
	if len(before) != 1 || before[0].DisplayName != "First DB" {
		t.Fatalf("before swap projects = %+v", before)
	}

	old := holder.Swap(second)
	if old != first {
		t.Fatalf("Swap returned unexpected old db")
	}
	after, err := svc.ListProjects(ctx, ListProjectsRequest{Limit: 10})
	if err != nil {
		t.Fatalf("ListProjects after swap failed: %v", err)
	}
	if len(after) != 1 || after[0].DisplayName != "Second DB" {
		t.Fatalf("after swap projects = %+v", after)
	}
}

func TestGetDefaultKeymap(t *testing.T) {
	db := openTestDB(t, "default_keymap.json")
	svc := New(storage.NewHolder(db))

	keymap, err := svc.GetDefaultKeymap(context.Background())
	if err != nil {
		t.Fatalf("GetDefaultKeymap failed: %v", err)
	}
	if keymap.ProfileID != storage.DefaultKeymapProfileID {
		t.Fatalf("keymap profile id = %d, want %d", keymap.ProfileID, storage.DefaultKeymapProfileID)
	}
	if len(keymap.Lanes) != 36 {
		t.Fatalf("keymap lanes = %d, want 36", len(keymap.Lanes))
	}
}

func TestDefaultProfileCanBeUpdatedAndReset(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t, "default_profile.json")
	svc := New(storage.NewHolder(db))

	profile, err := svc.GetDefaultProfile(ctx)
	if err != nil {
		t.Fatalf("GetDefaultProfile failed: %v", err)
	}
	profile.Name = "Global Custom"
	profile.BaseNote = 50
	profile.Transpose = -2
	profile.OctaveShift = 1
	profile.Speed = 1.25
	profile.OutOfRangePolicy = OutOfRangeOctaveFold
	profile.MinPressMs = 44
	profile.ReleaseGapMs = 18

	updated, err := svc.UpdateDefaultProfile(ctx, profile)
	if err != nil {
		t.Fatalf("UpdateDefaultProfile failed: %v", err)
	}
	if updated.ProjectID != nil || updated.BaseNote != 50 || updated.OutOfRangePolicy != OutOfRangeOctaveFold {
		t.Fatalf("updated default profile = %+v", updated)
	}

	reset, err := svc.ResetDefaultProfile(ctx)
	if err != nil {
		t.Fatalf("ResetDefaultProfile failed: %v", err)
	}
	if reset.Name != storage.DefaultMidiProfileName || reset.BaseNote != storage.DefaultMidiBaseNote || reset.OutOfRangePolicy != storage.DefaultOutOfRangePolicy {
		t.Fatalf("reset default profile = %+v", reset)
	}
}

func TestUpdateDefaultProfileRejectsProjectScopedProfile(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t, "default_profile_scoped.json")
	svc := New(storage.NewHolder(db))

	profile, err := svc.GetDefaultProfile(ctx)
	if err != nil {
		t.Fatalf("GetDefaultProfile failed: %v", err)
	}
	projectID := uint(42)
	profile.ProjectID = &projectID
	if _, err := svc.UpdateDefaultProfile(ctx, profile); err == nil {
		t.Fatalf("UpdateDefaultProfile accepted project-scoped profile")
	}
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func minimalMidiBytes() []byte {
	return []byte{
		'M', 'T', 'h', 'd',
		0x00, 0x00, 0x00, 0x06,
		0x00, 0x00,
		0x00, 0x01,
		0x01, 0xE0,
		'M', 'T', 'r', 'k',
		0x00, 0x00, 0x00, 0x0D,
		0x00, 0x90, 0x3C, 0x40,
		0x83, 0x60, 0x80, 0x3C, 0x00,
		0x00, 0xFF, 0x2F, 0x00,
	}
}

func singleNoteMidiBytes(note byte) []byte {
	data := minimalMidiBytes()
	data[24] = note
	data[29] = note
	return data
}

func emptyMidiBytes() []byte {
	return []byte{
		'M', 'T', 'h', 'd',
		0x00, 0x00, 0x00, 0x06,
		0x00, 0x00,
		0x00, 0x01,
		0x01, 0xE0,
		'M', 'T', 'r', 'k',
		0x00, 0x00, 0x00, 0x04,
		0x00, 0xFF, 0x2F, 0x00,
	}
}

func tempoChangeMidiBytes() []byte {
	return []byte{
		'M', 'T', 'h', 'd',
		0x00, 0x00, 0x00, 0x06,
		0x00, 0x01,
		0x00, 0x02,
		0x01, 0xE0,
		'M', 'T', 'r', 'k',
		0x00, 0x00, 0x00, 0x13,
		0x00, 0xFF, 0x51, 0x03, 0x07, 0xA1, 0x20,
		0x85, 0x50, 0xFF, 0x51, 0x03, 0x0F, 0x42, 0x40,
		0x00, 0xFF, 0x2F, 0x00,
		'M', 'T', 'r', 'k',
		0x00, 0x00, 0x00, 0x16,
		0x00, 0x90, 0x3C, 0x5A,
		0x83, 0x60, 0x80, 0x3C, 0x00,
		0x00, 0x91, 0x40, 0x46,
		0x83, 0x60, 0x91, 0x40, 0x00,
		0x00, 0xFF, 0x2F, 0x00,
	}
}

func openTestDB(t *testing.T, filename string) *storage.DB {
	t.Helper()

	db, err := storage.Open(filepath.Join(t.TempDir(), filename))
	if err != nil {
		t.Fatalf("open db failed: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Fatalf("close db failed: %v", err)
		}
	})
	return db
}

func assertProjectRowsDeleted(t *testing.T, db *storage.DB, projectID uint) {
	t.Helper()

	if _, ok := db.Store.GetProject(projectID); ok {
		t.Fatalf("project %d still exists", projectID)
	}
	if got := db.Store.CountEventsByProject(projectID); got != 0 {
		t.Fatalf("event count = %d, want 0", got)
	}
	if got := db.Store.CountProjectProfiles(projectID); got != 0 {
		t.Fatalf("project profile count = %d, want 0", got)
	}
	if got := db.Store.CountHistoryByProject(projectID); got != 0 {
		t.Fatalf("history count = %d, want 0", got)
	}
}

func assertDefaultRowsPreserved(t *testing.T, db *storage.DB) {
	t.Helper()

	if got := db.Store.CountGlobalDefaultProfiles(); got != 1 {
		t.Fatalf("global default profile count = %d, want 1", got)
	}
	if got := db.Store.CountKeymapRows(storage.DefaultKeymapProfileID); got != 36 {
		t.Fatalf("default keymap row count = %d, want 36", got)
	}
}

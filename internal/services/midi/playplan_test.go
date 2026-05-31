package midi

import (
	"context"
	"errors"
	"testing"

	"YyslsPlayer/internal/storage"
)

func TestBuildPlayPlanGeneratesSortedPressReleaseFrames(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t, "playplan.json")
	svc := New(storage.NewHolder(db))
	project := createPlayPlanProject(t, db, "PlayPlan")
	profile := createPlayPlanProfile(t, db, project.ID, MidiConfigDTO{
		BaseNote:         48,
		Transpose:        0,
		OctaveShift:      0,
		Speed:            1,
		OutOfRangePolicy: OutOfRangeDrop,
		MinPressMs:       35,
		ReleaseGapMs:     15,
		KeymapProfileID:  storage.DefaultKeymapProfileID,
	})
	createPlayPlanEvents(t, db, project.ID, []storage.MidiEvent{
		{Track: 0, Channel: 0, Note: 60, Velocity: 90, StartMs: 0, DurationMs: 20},
		{Track: 0, Channel: 0, Note: 64, Velocity: 80, StartMs: 10, DurationMs: 50},
		{Track: 0, Channel: 0, Note: 84, Velocity: 70, StartMs: 30, DurationMs: 40},
	})

	plan, err := svc.BuildPlayPlan(ctx, project.ID, profile.ID)
	if err != nil {
		t.Fatalf("BuildPlayPlan failed: %v", err)
	}
	if plan.ProjectID != project.ID || plan.ProfileID != profile.ID {
		t.Fatalf("plan ids = %d/%d", plan.ProjectID, plan.ProfileID)
	}
	if plan.DurationMs != 60 {
		t.Fatalf("duration = %d, want 60", plan.DurationMs)
	}
	if len(plan.Frames) != 4 {
		t.Fatalf("frames = %d, want 4: %+v", len(plan.Frames), plan.Frames)
	}
	assertFrame(t, plan.Frames[0], 0, KeyActionPress, 12, 60)
	assertFrame(t, plan.Frames[1], 10, KeyActionPress, 16, 64)
	assertFrame(t, plan.Frames[2], 35, KeyActionRelease, 12, 60)
	assertFrame(t, plan.Frames[3], 60, KeyActionRelease, 16, 64)
	if plan.Report.TotalNotes != 3 || plan.Report.PlayableNotes != 2 || plan.Report.DroppedCount != 1 || plan.Report.OutOfRangeCount != 1 {
		t.Fatalf("report = %+v", plan.Report)
	}
	if plan.Report.MappedRange.MinLane != 12 || plan.Report.MappedRange.MaxLane != 16 {
		t.Fatalf("mapped range = %+v", plan.Report.MappedRange)
	}
}

func TestBuildPlayPlanAppliesSpeedAndNearestPolicy(t *testing.T) {
	db := openTestDB(t, "playplan_speed.json")
	svc := New(storage.NewHolder(db))
	project := createPlayPlanProject(t, db, "Fast")
	profile := createPlayPlanProfile(t, db, project.ID, MidiConfigDTO{
		BaseNote:         48,
		Transpose:        0,
		OctaveShift:      0,
		Speed:            2,
		OutOfRangePolicy: OutOfRangeNearest,
		MinPressMs:       10,
		ReleaseGapMs:     0,
		KeymapProfileID:  storage.DefaultKeymapProfileID,
	})
	createPlayPlanEvents(t, db, project.ID, []storage.MidiEvent{
		{Track: 0, Channel: 0, Note: 84, Velocity: 64, StartMs: 100, DurationMs: 100},
	})

	plan, err := svc.BuildPlayPlan(context.Background(), project.ID, profile.ID)
	if err != nil {
		t.Fatalf("BuildPlayPlan failed: %v", err)
	}
	if len(plan.Frames) != 2 {
		t.Fatalf("frames = %+v", plan.Frames)
	}
	assertFrame(t, plan.Frames[0], 50, KeyActionPress, 35, 84)
	assertFrame(t, plan.Frames[1], 100, KeyActionRelease, 35, 84)
	if plan.Report.ClampedCount != 1 || plan.Report.DroppedCount != 0 || plan.Report.PlayableNotes != 1 {
		t.Fatalf("report = %+v", plan.Report)
	}
}

func TestBuildPlayPlanRejectsEmptyPlayablePlan(t *testing.T) {
	db := openTestDB(t, "playplan_empty.json")
	svc := New(storage.NewHolder(db))
	project := createPlayPlanProject(t, db, "Empty")
	profile := createPlayPlanProfile(t, db, project.ID, DefaultMidiConfig())
	createPlayPlanEvents(t, db, project.ID, []storage.MidiEvent{
		{Track: 0, Channel: 0, Note: 84, Velocity: 64, StartMs: 0, DurationMs: 100},
	})

	_, err := svc.BuildPlayPlan(context.Background(), project.ID, profile.ID)
	if !errors.Is(err, ErrPlayPlanEmpty) {
		t.Fatalf("BuildPlayPlan error = %v, want PLAYPLAN_EMPTY", err)
	}
}

func TestBuildKeyFramesAppliesReleaseGapForSameLane(t *testing.T) {
	cfg := DefaultMidiConfig()
	cfg.ReleaseGapMs = 20
	mapping, err := Key36MappingFromRows(storage.DefaultKeymapProfileID, defaultKeymapRowsForTest())
	if err != nil {
		t.Fatalf("Key36MappingFromRows failed: %v", err)
	}
	frames, _, durationMs := buildKeyFrames([]storage.MidiEvent{
		{Note: 60, Velocity: 90, StartMs: 0, DurationMs: 50},
		{Note: 60, Velocity: 90, StartMs: 55, DurationMs: 30},
	}, cfg, mapping)
	if len(frames) != 4 {
		t.Fatalf("frames = %+v", frames)
	}
	assertFrame(t, frames[0], 0, KeyActionPress, 12, 60)
	assertFrame(t, frames[1], 50, KeyActionRelease, 12, 60)
	assertFrame(t, frames[2], 70, KeyActionPress, 12, 60)
	assertFrame(t, frames[3], 105, KeyActionRelease, 12, 60)
	if durationMs != 105 {
		t.Fatalf("duration = %d, want 105", durationMs)
	}
}

func TestBuildKeyFramesOrdersReleaseBeforePressAtSameTime(t *testing.T) {
	cfg := DefaultMidiConfig()
	mapping, err := Key36MappingFromRows(storage.DefaultKeymapProfileID, defaultKeymapRowsForTest())
	if err != nil {
		t.Fatalf("Key36MappingFromRows failed: %v", err)
	}
	frames, _, _ := buildKeyFrames([]storage.MidiEvent{
		{Note: 60, Velocity: 90, StartMs: 0, DurationMs: 100},
		{Note: 62, Velocity: 90, StartMs: 100, DurationMs: 100},
	}, cfg, mapping)
	if len(frames) != 4 {
		t.Fatalf("frames = %+v", frames)
	}
	if frames[1].Action != KeyActionRelease || frames[2].Action != KeyActionPress || frames[1].TimeMs != frames[2].TimeMs {
		t.Fatalf("same-time order = %+v", frames)
	}
}

func createPlayPlanProject(t *testing.T, db *storage.DB, name string) storage.MidiProject {
	t.Helper()
	project := storage.MidiProject{
		DisplayName:  name,
		FileName:     name + ".mid",
		FileHash:     "hash-" + name,
		PPQ:          480,
		TrackCount:   1,
		ChannelCount: 1,
		DurationMs:   1000,
	}
	saved, err := db.Store.SaveProject(project)
	if err != nil {
		t.Fatalf("create project failed: %v", err)
	}
	return saved
}

func createPlayPlanProfile(t *testing.T, db *storage.DB, projectID uint, cfg MidiConfigDTO) storage.MidiProfile {
	t.Helper()
	profile := storage.MidiProfile{
		ProjectID:        &projectID,
		Name:             "Plan profile",
		BaseNote:         cfg.BaseNote,
		Transpose:        cfg.Transpose,
		OctaveShift:      cfg.OctaveShift,
		Speed:            cfg.Speed,
		OutOfRangePolicy: cfg.OutOfRangePolicy,
		MinPressMs:       cfg.MinPressMs,
		ReleaseGapMs:     cfg.ReleaseGapMs,
		KeymapProfileID:  cfg.KeymapProfileID,
	}
	saved, err := db.Store.SaveProfile(profile)
	if err != nil {
		t.Fatalf("create profile failed: %v", err)
	}
	return saved
}

func createPlayPlanEvents(t *testing.T, db *storage.DB, projectID uint, events []storage.MidiEvent) {
	t.Helper()
	for i := range events {
		events[i].ProjectID = projectID
	}
	if err := db.Store.AddEvents(events); err != nil {
		t.Fatalf("create events failed: %v", err)
	}
}

func assertFrame(t *testing.T, frame KeyFrameDTO, timeMs int64, action string, lane int, sourceNote int) {
	t.Helper()
	if frame.TimeMs != timeMs || frame.Action != action || frame.Lane != lane || frame.SourceNote != sourceNote {
		t.Fatalf("frame = %+v, want time=%d action=%s lane=%d source=%d", frame, timeMs, action, lane, sourceNote)
	}
	if frame.Key.VirtualKey == 0 {
		t.Fatalf("frame key missing: %+v", frame)
	}
}

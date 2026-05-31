package acceptance

import (
	"context"
	"encoding/binary"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"YyslsPlayer/internal/services/keysim"
	midisvc "YyslsPlayer/internal/services/midi"
	"YyslsPlayer/internal/services/player"
	"YyslsPlayer/internal/storage"
)

type sampleNote struct {
	note          byte
	startTick     int
	durationTicks int
	velocity      byte
}

type sampleExpectation struct {
	totalNotes      int
	noteMin         int
	noteMax         int
	blackKeys       int
	playableNotes   int
	outOfRange      int
	dropped         int
	mappedMin       int
	mappedMax       int
	frameCount      int
	requireBlackKey bool
}

func TestSampleMIDIAcceptanceImportPlanAndDryRun(t *testing.T) {
	ctx := context.Background()
	db := openAcceptanceDB(t)
	svc := midisvc.New(storage.NewHolder(db))

	cases := []struct {
		name  string
		file  string
		notes []sampleNote
		want  sampleExpectation
	}{
		{
			name: "natural scale sample maps natural notes to 36 lanes",
			file: "sample-natural.mid",
			notes: []sampleNote{
				{note: 60, startTick: 0, durationTicks: 96, velocity: 90},
				{note: 62, startTick: 240, durationTicks: 96, velocity: 88},
				{note: 64, startTick: 480, durationTicks: 96, velocity: 86},
			},
			want: sampleExpectation{totalNotes: 3, noteMin: 60, noteMax: 64, blackKeys: 0, playableNotes: 3, mappedMin: 12, mappedMax: 16, frameCount: 6},
		},
		{
			name: "chromatic sample preserves semitone lanes",
			file: "sample-chromatic.mid",
			notes: []sampleNote{
				{note: 60, startTick: 0, durationTicks: 96, velocity: 90},
				{note: 61, startTick: 240, durationTicks: 96, velocity: 88},
				{note: 63, startTick: 480, durationTicks: 96, velocity: 86},
			},
			want: sampleExpectation{totalNotes: 3, noteMin: 60, noteMax: 63, blackKeys: 2, playableNotes: 3, mappedMin: 12, mappedMax: 15, frameCount: 6, requireBlackKey: true},
		},
		{
			name: "out of range sample drops only invalid notes by default",
			file: "sample-out-of-range.mid",
			notes: []sampleNote{
				{note: 72, startTick: 0, durationTicks: 96, velocity: 90},
				{note: 84, startTick: 240, durationTicks: 96, velocity: 88},
			},
			want: sampleExpectation{totalNotes: 2, noteMin: 72, noteMax: 84, blackKeys: 0, playableNotes: 1, outOfRange: 1, dropped: 1, mappedMin: 24, mappedMax: 24, frameCount: 2},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := writeMIDISample(t, tc.file, tc.notes)
			detail, err := svc.ImportFile(ctx, path)
			if err != nil {
				t.Fatalf("ImportFile failed: %v", err)
			}
			assertQualityReport(t, detail.QualityReport, tc.want)
			if detail.Project.NoteCount != tc.want.totalNotes || detail.EventCount != int64(tc.want.totalNotes) {
				t.Fatalf("project/event counts = %d/%d, want %d", detail.Project.NoteCount, detail.EventCount, tc.want.totalNotes)
			}

			plan, err := svc.BuildPlayPlan(ctx, detail.Project.ID, detail.DefaultProfile.ID)
			if err != nil {
				t.Fatalf("BuildPlayPlan failed: %v", err)
			}
			assertQualityReport(t, plan.Report, tc.want)
			if len(plan.Frames) != tc.want.frameCount {
				t.Fatalf("plan frames = %d, want %d: %+v", len(plan.Frames), tc.want.frameCount, plan.Frames)
			}
			assertPlanUsesExpectedLanes(t, plan, tc.want)
			runPlayerDryRun(t, plan)
		})
	}
}

func TestSampleMIDIAcceptanceKeepsPitchAndPhysicalMappingSeparated(t *testing.T) {
	ctx := context.Background()
	db := openAcceptanceDB(t)
	svc := midisvc.New(storage.NewHolder(db))
	path := writeMIDISample(t, "sample-mapping-separation.mid", []sampleNote{
		{note: 60, startTick: 0, durationTicks: 96, velocity: 90},
	})

	detail, err := svc.ImportFile(ctx, path)
	if err != nil {
		t.Fatalf("ImportFile failed: %v", err)
	}
	before, err := svc.BuildPlayPlan(ctx, detail.Project.ID, detail.DefaultProfile.ID)
	if err != nil {
		t.Fatalf("BuildPlayPlan before keymap change failed: %v", err)
	}
	firstBefore := firstPressFrame(t, before)
	if firstBefore.Lane != 12 || firstBefore.SourceNote != 60 || firstBefore.Key.ScanCode == 123 {
		t.Fatalf("unexpected initial frame: %+v", firstBefore)
	}

	if err := db.Store.UpdateKeymapLane(storage.DefaultKeymapProfileID, 12, func(row *storage.Keymap36) {
		row.Label = "Custom-C4"
		row.VirtualKey = 222
		row.ScanCode = 123
	}); err != nil {
		t.Fatalf("update keymap lane failed: %v", err)
	}
	afterKeymap, err := svc.BuildPlayPlan(ctx, detail.Project.ID, detail.DefaultProfile.ID)
	if err != nil {
		t.Fatalf("BuildPlayPlan after keymap change failed: %v", err)
	}
	firstAfterKeymap := firstPressFrame(t, afterKeymap)
	if firstAfterKeymap.Lane != 12 || firstAfterKeymap.SourceNote != 60 || firstAfterKeymap.NormalizedNote != 60 {
		t.Fatalf("physical keymap changed pitch mapping: %+v", firstAfterKeymap)
	}
	if firstAfterKeymap.Key.ScanCode != 123 || firstAfterKeymap.Key.VirtualKey != 222 || firstAfterKeymap.Key.Label != "Custom-C4" {
		t.Fatalf("physical keymap change was not reflected in frame key: %+v", firstAfterKeymap.Key)
	}

	updated := detail.DefaultProfile
	updated.ProjectID = &detail.Project.ID
	updated.Name = "Sample shifted profile"
	updated.BaseNote = 60
	updated.Transpose = 0
	updatedProfile, err := svc.UpdateProfile(ctx, updated)
	if err != nil {
		t.Fatalf("UpdateProfile failed: %v", err)
	}
	afterPitchConfig, err := svc.BuildPlayPlan(ctx, detail.Project.ID, updatedProfile.ID)
	if err != nil {
		t.Fatalf("BuildPlayPlan after pitch config change failed: %v", err)
	}
	firstAfterPitch := firstPressFrame(t, afterPitchConfig)
	if firstAfterPitch.Lane != 0 || firstAfterPitch.SourceNote != 60 || firstAfterPitch.NormalizedNote != 60 {
		t.Fatalf("baseNote change did not move pitch mapping as expected: %+v", firstAfterPitch)
	}

	var customized storage.Keymap36
	for _, row := range db.Store.ListKeymapProfile(storage.DefaultKeymapProfileID) {
		if row.Lane == 12 {
			customized = row
			break
		}
	}
	if customized.ScanCode != 123 || customized.VirtualKey != 222 || customized.Label != "Custom-C4" {
		t.Fatalf("pitch config update modified physical keymap row: %+v", customized)
	}
}

func openAcceptanceDB(t *testing.T) *storage.DB {
	t.Helper()
	db, err := storage.Open(filepath.Join(t.TempDir(), "sample_acceptance.json"))
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

func writeMIDISample(t *testing.T, name string, notes []sampleNote) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, midiBytes(notes), 0o644); err != nil {
		t.Fatalf("write midi sample failed: %v", err)
	}
	return path
}

func midiBytes(notes []sampleNote) []byte {
	const ppq uint16 = 480
	track := make([]byte, 0, len(notes)*8+16)
	track = append(track, encodeVLQ(0)...)
	track = append(track, 0xFF, 0x51, 0x03, 0x01, 0x86, 0xA0) // 100000 us/quarter keeps acceptance tests fast.

	sorted := append([]sampleNote(nil), notes...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].startTick < sorted[j].startTick })
	currentTick := 0
	for _, note := range sorted {
		start := note.startTick
		if start < currentTick {
			start = currentTick
		}
		track = append(track, encodeVLQ(start-currentTick)...)
		track = append(track, 0x90, note.note, note.velocity)
		currentTick = start
		duration := note.durationTicks
		if duration < 0 {
			duration = 0
		}
		track = append(track, encodeVLQ(duration)...)
		track = append(track, 0x80, note.note, 0x00)
		currentTick += duration
	}
	track = append(track, encodeVLQ(0)...)
	track = append(track, 0xFF, 0x2F, 0x00)

	out := []byte{
		'M', 'T', 'h', 'd',
		0x00, 0x00, 0x00, 0x06,
		0x00, 0x00,
		0x00, 0x01,
		0x00, 0x00,
		'M', 'T', 'r', 'k',
		0x00, 0x00, 0x00, 0x00,
	}
	binary.BigEndian.PutUint16(out[12:14], ppq)
	binary.BigEndian.PutUint32(out[18:22], uint32(len(track)))
	out = append(out, track...)
	return out
}

func encodeVLQ(value int) []byte {
	if value <= 0 {
		return []byte{0}
	}
	buffer := []byte{byte(value & 0x7F)}
	value >>= 7
	for value > 0 {
		buffer = append([]byte{byte(value&0x7F) | 0x80}, buffer...)
		value >>= 7
	}
	return buffer
}

func assertQualityReport(t *testing.T, report midisvc.QualityReportDTO, want sampleExpectation) {
	t.Helper()
	if report.TotalNotes != want.totalNotes || report.PlayableNotes != want.playableNotes || report.OutOfRangeCount != want.outOfRange || report.DroppedCount != want.dropped {
		t.Fatalf("report counts = %+v, want total=%d playable=%d out=%d dropped=%d", report, want.totalNotes, want.playableNotes, want.outOfRange, want.dropped)
	}
	if report.NoteRange.Min != want.noteMin || report.NoteRange.Max != want.noteMax {
		t.Fatalf("note range = %+v, want %d..%d", report.NoteRange, want.noteMin, want.noteMax)
	}
	if report.BlackKeyCount != want.blackKeys {
		t.Fatalf("black key count = %d, want %d", report.BlackKeyCount, want.blackKeys)
	}
	if report.MappedRange.MinLane != want.mappedMin || report.MappedRange.MaxLane != want.mappedMax {
		t.Fatalf("mapped range = %+v, want %d..%d", report.MappedRange, want.mappedMin, want.mappedMax)
	}
	if want.totalNotes > 0 {
		wantRatio := float64(want.playableNotes) / float64(want.totalNotes)
		if report.PlayableRatio != wantRatio {
			t.Fatalf("playable ratio = %v, want %v", report.PlayableRatio, wantRatio)
		}
	}
}

func assertPlanUsesExpectedLanes(t *testing.T, plan midisvc.PlayPlanDTO, want sampleExpectation) {
	t.Helper()
	seenBlackKey := false
	for i, frame := range plan.Frames {
		if frame.Lane < want.mappedMin || frame.Lane > want.mappedMax {
			t.Fatalf("frame %d lane = %d outside expected %d..%d: %+v", i, frame.Lane, want.mappedMin, want.mappedMax, frame)
		}
		if frame.Key.VirtualKey == 0 || frame.Key.ScanCode == 0 {
			t.Fatalf("frame %d missing physical key data: %+v", i, frame)
		}
		if frame.Key.IsBlackKey {
			seenBlackKey = true
		}
	}
	if want.requireBlackKey && !seenBlackKey {
		t.Fatalf("expected at least one black-key frame: %+v", plan.Frames)
	}
}

func runPlayerDryRun(t *testing.T, plan midisvc.PlayPlanDTO) {
	t.Helper()
	ctx := context.Background()
	sim := keysim.New(keysim.NewStubDriver())
	svc := player.New(sim)
	session, err := svc.Start(ctx, player.StartRequest{Plan: plan, DryRun: true, LookaheadMs: player.MinLookaheadMs})
	if err != nil {
		t.Fatalf("player Start failed: %v", err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		state, err := svc.GetState(ctx, session.SessionID)
		if err != nil {
			t.Fatalf("player GetState failed: %v", err)
		}
		if state.State == player.StateCompleted {
			if pressed := sim.Snapshot().Pressed; len(pressed) != 0 {
				t.Fatalf("pressed keys after completion = %+v", pressed)
			}
			return
		}
		if state.State == player.StateError {
			t.Fatalf("player entered error state: %+v", state)
		}
		time.Sleep(5 * time.Millisecond)
	}
	state, _ := svc.GetState(ctx, session.SessionID)
	t.Fatalf("timed out waiting for dry-run completion: %+v", state)
}

func firstPressFrame(t *testing.T, plan midisvc.PlayPlanDTO) midisvc.KeyFrameDTO {
	t.Helper()
	for _, frame := range plan.Frames {
		if frame.Action == midisvc.KeyActionPress {
			return frame
		}
	}
	t.Fatalf("no press frame in plan: %+v", plan.Frames)
	return midisvc.KeyFrameDTO{}
}

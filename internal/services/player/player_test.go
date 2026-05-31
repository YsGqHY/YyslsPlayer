package player

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"YyslsPlayer/internal/services/keysim"
	"YyslsPlayer/internal/services/midi"
)

func TestStartPauseResumeStopStateMachine(t *testing.T) {
	ctx := context.Background()
	driver := &recordingDriver{}
	sim := keysim.New(driver)
	svc := New(sim)

	session, err := svc.Start(ctx, StartRequest{Plan: testPlan(), DryRun: false})
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	if session.State != StatePlaying || session.DurationMs != 1100 || session.FrameCount != 2 || session.LookaheadMs != DefaultLookaheadMs {
		t.Fatalf("session = %+v", session)
	}

	paused, err := svc.Pause(ctx, session.SessionID)
	if err != nil {
		t.Fatalf("Pause failed: %v", err)
	}
	if paused.State != StatePaused {
		t.Fatalf("paused state = %+v", paused)
	}

	resumed, err := svc.Resume(ctx, session.SessionID)
	if err != nil {
		t.Fatalf("Resume failed: %v", err)
	}
	if resumed.State != StatePlaying {
		t.Fatalf("resumed state = %+v", resumed)
	}

	if _, err := sim.Apply(ctx, keysim.KeyAction{Action: keysim.ActionPress, Key: keyA()}, keysim.RunOptions{DryRun: false}); err != nil {
		t.Fatalf("pre-press failed: %v", err)
	}
	stopped, err := svc.Stop(ctx, session.SessionID)
	if err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
	if stopped.State != StateStopped {
		t.Fatalf("stopped state = %+v", stopped)
	}
	if got := sim.Snapshot().Pressed; len(got) != 0 {
		t.Fatalf("pressed = %+v, want empty", got)
	}
	events := driver.Events()
	if len(events) != 2 || events[1].Kind != keysim.PhysicalUp || events[1].Key.Label != "A" {
		t.Fatalf("driver events = %+v", events)
	}
}

func TestSchedulerCompletesShortPlayPlan(t *testing.T) {
	ctx := context.Background()
	driver := &recordingDriver{}
	svc := New(keysim.New(driver))
	session, err := svc.Start(ctx, StartRequest{Plan: timedPlan(0, 15), LookaheadMs: MinLookaheadMs, DryRun: false})
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	events := waitForEvents(t, driver, 2, 250*time.Millisecond)
	assertEvent(t, events[0], keysim.PhysicalDown, "A")
	assertEvent(t, events[1], keysim.PhysicalUp, "A")
	state := waitForState(t, svc, session.SessionID, StateCompleted, 250*time.Millisecond)
	if state.PositionMs != state.DurationMs {
		t.Fatalf("completed state = %+v", state)
	}
}

func TestPlayerEventsEmitStateAndPosition(t *testing.T) {
	ctx := context.Background()
	emitter := &recordingEmitter{}
	svc := New(keysim.New(&recordingDriver{}))
	svc.AttachEmitter(emitter)
	session, err := svc.Start(ctx, StartRequest{Plan: testPlan(), DryRun: false})
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	waitForPlayerEvent(t, emitter, EventState, 100*time.Millisecond, func(payload any) bool {
		state, ok := payload.(PlayerStateDTO)
		return ok && state.SessionID == session.SessionID && state.State == StatePlaying
	})
	waitForPlayerEvent(t, emitter, EventPosition, 100*time.Millisecond, func(payload any) bool {
		position, ok := payload.(PlayerPositionDTO)
		return ok && position.SessionID == session.SessionID && position.State == StatePlaying
	})
	if _, err := svc.Pause(ctx, session.SessionID); err != nil {
		t.Fatalf("Pause failed: %v", err)
	}
	waitForPlayerEvent(t, emitter, EventState, 100*time.Millisecond, func(payload any) bool {
		state, ok := payload.(PlayerStateDTO)
		return ok && state.SessionID == session.SessionID && state.State == StatePaused
	})
	if _, err := svc.Resume(ctx, session.SessionID); err != nil {
		t.Fatalf("Resume failed: %v", err)
	}
	waitForPlayerEvent(t, emitter, EventState, 100*time.Millisecond, func(payload any) bool {
		state, ok := payload.(PlayerStateDTO)
		return ok && state.SessionID == session.SessionID && state.State == StatePlaying && state.Message == "resumed"
	})
	if _, err := svc.Stop(ctx, session.SessionID); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
	waitForPlayerEvent(t, emitter, EventState, 100*time.Millisecond, func(payload any) bool {
		state, ok := payload.(PlayerStateDTO)
		return ok && state.SessionID == session.SessionID && state.State == StateStopped
	})
}

func TestPlayerEventsEmitCompletionAndError(t *testing.T) {
	ctx := context.Background()
	completeEmitter := &recordingEmitter{}
	completeSvc := New(keysim.New(&recordingDriver{}))
	completeSvc.AttachEmitter(completeEmitter)
	completeSession, err := completeSvc.Start(ctx, StartRequest{Plan: timedPlan(0, 10), LookaheadMs: MinLookaheadMs, DryRun: false})
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	waitForState(t, completeSvc, completeSession.SessionID, StateCompleted, 250*time.Millisecond)
	waitForPlayerEvent(t, completeEmitter, EventState, 100*time.Millisecond, func(payload any) bool {
		state, ok := payload.(PlayerStateDTO)
		return ok && state.SessionID == completeSession.SessionID && state.State == StateCompleted
	})

	errorEmitter := &recordingEmitter{}
	errorSvc := New(keysim.New(failingDriver{}))
	errorSvc.AttachEmitter(errorEmitter)
	errorSession, err := errorSvc.Start(ctx, StartRequest{Plan: timedPlan(0, 10), LookaheadMs: MinLookaheadMs, DryRun: false})
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	waitForState(t, errorSvc, errorSession.SessionID, StateError, 250*time.Millisecond)
	waitForPlayerEvent(t, errorEmitter, EventError, 100*time.Millisecond, func(payload any) bool {
		event, ok := payload.(PlayerErrorDTO)
		return ok && event.SessionID == errorSession.SessionID && event.ErrorCode == "KEYSIM_SEND_FAILED"
	})
}

func TestLookaheadDoesNotSendFutureFrameEarly(t *testing.T) {
	ctx := context.Background()
	driver := &recordingDriver{}
	svc := New(keysim.New(driver))
	session, err := svc.Start(ctx, StartRequest{Plan: timedPlan(40, 55), LookaheadMs: 20, DryRun: false})
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	time.Sleep(10 * time.Millisecond)
	if got := len(driver.Events()); got != 0 {
		t.Fatalf("events sent early = %d", got)
	}
	if _, err := svc.Stop(ctx, session.SessionID); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
}

func TestPauseReleasesPressedKeysAndBlocksSchedulerUntilResume(t *testing.T) {
	ctx := context.Background()
	driver := &recordingDriver{}
	sim := keysim.New(driver)
	svc := New(sim)
	session, err := svc.Start(ctx, StartRequest{Plan: timedPlan(0, 50), LookaheadMs: MinLookaheadMs, DryRun: false})
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	waitForEvents(t, driver, 1, 150*time.Millisecond)
	if _, err := svc.Pause(ctx, session.SessionID); err != nil {
		t.Fatalf("Pause failed: %v", err)
	}
	events := waitForEvents(t, driver, 2, 100*time.Millisecond)
	assertEvent(t, events[1], keysim.PhysicalUp, "A")
	if got := sim.Snapshot().Pressed; len(got) != 0 {
		t.Fatalf("pressed while paused = %+v, want empty", got)
	}
	time.Sleep(70 * time.Millisecond)
	if got := len(driver.Events()); got != 2 {
		t.Fatalf("events while paused = %d, want pause release only", got)
	}
	if _, err := svc.Resume(ctx, session.SessionID); err != nil {
		t.Fatalf("Resume failed: %v", err)
	}
	waitForState(t, svc, session.SessionID, StateCompleted, 200*time.Millisecond)
	if got := len(driver.Events()); got != 2 {
		t.Fatalf("events after resume = %d, want no duplicate release", got)
	}
}

func TestStopCancelsSchedulerAndReleasesPressedKeys(t *testing.T) {
	ctx := context.Background()
	driver := &recordingDriver{}
	sim := keysim.New(driver)
	svc := New(sim)
	session, err := svc.Start(ctx, StartRequest{Plan: timedPlan(0, 80), LookaheadMs: MinLookaheadMs, DryRun: false})
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	waitForEvents(t, driver, 1, 150*time.Millisecond)
	stopped, err := svc.Stop(ctx, session.SessionID)
	if err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
	if stopped.State != StateStopped {
		t.Fatalf("stopped = %+v", stopped)
	}
	if got := sim.Snapshot().Pressed; len(got) != 0 {
		t.Fatalf("pressed = %+v, want empty", got)
	}
	time.Sleep(110 * time.Millisecond)
	events := driver.Events()
	if len(events) != 2 {
		t.Fatalf("events after stop = %+v, want down + stop release only", events)
	}
	assertEvent(t, events[1], keysim.PhysicalUp, "A")
}

func TestStartRejectsBusySession(t *testing.T) {
	svc := New(keysim.New(&recordingDriver{}))
	session, err := svc.Start(context.Background(), StartRequest{Plan: testPlan(), DryRun: true})
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer func() { _, _ = svc.Stop(context.Background(), session.SessionID) }()
	_, err = svc.Start(context.Background(), StartRequest{Plan: testPlan(), DryRun: true})
	if !errors.Is(err, ErrPlayerBusy) {
		t.Fatalf("err = %v, want PLAYER_BUSY", err)
	}
}

func TestStartRejectsEmptyPlan(t *testing.T) {
	svc := New(keysim.New(&recordingDriver{}))
	_, err := svc.Start(context.Background(), StartRequest{Plan: midi.PlayPlanDTO{}, DryRun: true})
	if !errors.Is(err, ErrPlayPlanEmpty) {
		t.Fatalf("err = %v, want PLAYPLAN_EMPTY", err)
	}
}

func TestStartRejectsInvalidLookahead(t *testing.T) {
	svc := New(keysim.New(&recordingDriver{}))
	_, err := svc.Start(context.Background(), StartRequest{Plan: testPlan(), LookaheadMs: MaxLookaheadMs + 1, DryRun: true})
	if !errors.Is(err, ErrInvalidLookahead) {
		t.Fatalf("err = %v, want PLAYER_INVALID_LOOKAHEAD", err)
	}
}

func TestPauseRejectsInvalidTransition(t *testing.T) {
	ctx := context.Background()
	svc := New(keysim.New(&recordingDriver{}))
	session, err := svc.Start(ctx, StartRequest{Plan: testPlan(), DryRun: true})
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer func() { _, _ = svc.Stop(context.Background(), session.SessionID) }()
	if _, err := svc.Pause(ctx, session.SessionID); err != nil {
		t.Fatalf("Pause failed: %v", err)
	}
	_, err = svc.Pause(ctx, session.SessionID)
	if !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("err = %v, want PLAYER_INVALID_STATE", err)
	}
}

func TestGetStateIdleAndNotFound(t *testing.T) {
	svc := New(keysim.New(&recordingDriver{}))
	idle, err := svc.GetState(context.Background(), "")
	if err != nil {
		t.Fatalf("GetState idle failed: %v", err)
	}
	if idle.State != StateIdle {
		t.Fatalf("idle = %+v", idle)
	}
	_, err = svc.GetState(context.Background(), "missing")
	if !errors.Is(err, ErrPlayerNotFound) {
		t.Fatalf("err = %v, want PLAYER_NOT_FOUND", err)
	}
}

func TestReleaseAllUsesCurrentSessionDryRunOption(t *testing.T) {
	ctx := context.Background()
	driver := &recordingDriver{}
	sim := keysim.New(driver)
	svc := New(sim)
	session, err := svc.Start(ctx, StartRequest{Plan: testPlan(), DryRun: true})
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer func() { _, _ = svc.Stop(context.Background(), session.SessionID) }()
	if _, err := sim.Apply(ctx, keysim.KeyAction{Action: keysim.ActionPress, Key: keyA()}, keysim.RunOptions{DryRun: true}); err != nil {
		t.Fatalf("pre-press failed: %v", err)
	}
	if err := svc.ReleaseAll(ctx); err != nil {
		t.Fatalf("ReleaseAll failed: %v", err)
	}
	if got := sim.Snapshot().Pressed; len(got) != 0 {
		t.Fatalf("pressed = %+v, want empty", got)
	}
	events := driver.Events()
	dryRuns := driver.DryRuns()
	if len(events) != 2 || !dryRuns[1] || events[1].Kind != keysim.PhysicalUp {
		t.Fatalf("events = %+v dryRuns = %+v", events, dryRuns)
	}
}

func testPlan() midi.PlayPlanDTO {
	return timedPlan(1000, 1100)
}

func timedPlan(pressAt int64, releaseAt int64) midi.PlayPlanDTO {
	return midi.PlayPlanDTO{
		ProjectID:  10,
		ProfileID:  20,
		DurationMs: releaseAt,
		Frames: []midi.KeyFrameDTO{
			{TimeMs: pressAt, Action: midi.KeyActionPress, Lane: 0, SourceNote: 60, NormalizedNote: 60, Velocity: 90, Key: laneKeyA()},
			{TimeMs: releaseAt, Action: midi.KeyActionRelease, Lane: 0, SourceNote: 60, NormalizedNote: 60, Velocity: 90, Key: laneKeyA()},
		},
	}
}

func laneKeyA() midi.KeymapLaneDTO {
	return midi.KeymapLaneDTO{Label: "A", VirtualKey: 65, ScanCode: 30, ModifierKeysJSON: "[]"}
}

type failingDriver struct{}

func (failingDriver) Send(_ context.Context, _ keysim.KeyEvent, _ keysim.RunOptions) error {
	return keysim.ErrSendFailed
}

type eventRecord struct {
	name    string
	payload any
}

type recordingEmitter struct {
	mu     sync.Mutex
	events []eventRecord
}

func (e *recordingEmitter) Emit(name string, payload any) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.events = append(e.events, eventRecord{name: name, payload: payload})
}

func (e *recordingEmitter) Events() []eventRecord {
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]eventRecord(nil), e.events...)
}

type recordingDriver struct {
	mu      sync.Mutex
	events  []keysim.KeyEvent
	dryRuns []bool
}

func (d *recordingDriver) Send(_ context.Context, event keysim.KeyEvent, opts keysim.RunOptions) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.events = append(d.events, event)
	d.dryRuns = append(d.dryRuns, opts.DryRun)
	return nil
}

func (d *recordingDriver) Events() []keysim.KeyEvent {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]keysim.KeyEvent(nil), d.events...)
}

func (d *recordingDriver) DryRuns() []bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]bool(nil), d.dryRuns...)
}

func waitForEvents(t *testing.T, driver *recordingDriver, want int, timeout time.Duration) []keysim.KeyEvent {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		events := driver.Events()
		if len(events) >= want {
			return events
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %d events, got %+v", want, driver.Events())
	return nil
}

func waitForPlayerEvent(t *testing.T, emitter *recordingEmitter, name string, timeout time.Duration, ok func(any) bool) eventRecord {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		for _, event := range emitter.Events() {
			if event.name == name && ok(event.payload) {
				return event
			}
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for player event %s, got %+v", name, emitter.Events())
	return eventRecord{}
}

func waitForState(t *testing.T, svc *Service, sessionID string, want PlayerState, timeout time.Duration) PlayerStateDTO {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		state, err := svc.GetState(context.Background(), sessionID)
		if err != nil {
			t.Fatalf("GetState failed: %v", err)
		}
		if state.State == want {
			return state
		}
		time.Sleep(2 * time.Millisecond)
	}
	state, _ := svc.GetState(context.Background(), sessionID)
	t.Fatalf("timed out waiting for state %s, got %+v", want, state)
	return PlayerStateDTO{}
}

func assertEvent(t *testing.T, event keysim.KeyEvent, kind keysim.PhysicalKind, label string) {
	t.Helper()
	if event.Kind != kind || event.Key.Label != label {
		t.Fatalf("event = %+v, want kind=%s label=%s", event, kind, label)
	}
}

func keyA() keysim.Key {
	return keysim.Key{Label: "A", VirtualKey: 65, ScanCode: 30}
}

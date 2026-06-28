//go:build completion

package macro

import (
	"context"
	"sync"

	"YyslsPlayer/internal/storage"
	"YyslsPlayer/internal/utils/logx"
)

// capturedEvent is a single low-level keyboard or mouse transition delivered by
// the platform recorder hook.
type capturedEvent struct {
	vk     int
	scan   int
	keyUp  bool
	mouse  bool
	wheel  bool // scroll-wheel notch; button carries the wheel direction id
	move   bool // relative cursor move; dx/dy carry the accumulated offset
	dx     int
	dy     int
	button int
	atMs   int64 // monotonic milliseconds since session start
}

// keyRecorder is the platform hook that streams raw key/mouse transitions.
type keyRecorder interface {
	// Start installs the hook and invokes onEvent for every transition until
	// Stop is called. atMs is filled in by the recorder relative to start.
	Start(onEvent func(capturedEvent)) error
	Stop()
}

type recordSession struct {
	mu            sync.Mutex
	steps         []storage.MacroStep
	lastAtMs      int64
	startWall     int64
	rec           keyRecorder
	captureDelays bool // when false, events are recorded back-to-back (G HUB "record delays" off)
	captureMoves  bool // when false, cursor-move events are dropped (G HUB "record mouse movement" off)
	// pendingMove accumulates consecutive WM_MOUSEMOVE deltas so a continuous
	// drag collapses into one move step, flushed before the next non-move event.
	pendingMove   bool
	pendingDx     int
	pendingDy     int
	pendingMoveAt int64
}

// StartRecording installs a low-level hook and begins capturing key/mouse
// transitions into a fresh timeline. Recording is mutually exclusive with macro
// playback. captureDelays controls whether the gaps between events are recorded
// as delay steps; when false, events are appended back-to-back so the user can
// insert precise delays manually (mirrors Logitech G HUB's "record delays" toggle).
// captureMoves controls whether relative cursor-move events are recorded; when
// false, mouse movement is ignored so a keyboard/click macro is not polluted by
// pointer drift (mirrors G HUB's "record mouse movement" toggle).
func (s *Service) StartRecording(ctx context.Context, captureDelays bool, captureMoves bool) (RecordStateDTO, error) {
	_ = ctx
	s.mu.Lock()
	if s.current != nil {
		s.mu.Unlock()
		return RecordStateDTO{}, ErrMacroBusy
	}
	if s.recording != nil {
		st := s.recordState
		s.mu.Unlock()
		return st, nil
	}
	rec := newKeyRecorder()
	sess := &recordSession{startWall: nowMillis(), rec: rec, captureDelays: captureDelays, captureMoves: captureMoves}
	if err := rec.Start(func(ev capturedEvent) { s.onCaptured(sess, ev) }); err != nil {
		s.mu.Unlock()
		s.recordError(err)
		return RecordStateDTO{}, err
	}
	s.recording = sess
	s.recordState = RecordStateDTO{State: RecordStateRecording, StartedAt: sess.startWall, UpdatedAt: sess.startWall}
	state := s.recordState
	s.mu.Unlock()
	s.emitRecordState(state)
	logx.For("macro").Info("macro recording started")
	return state, nil
}

// StopRecording tears down the hook and returns the captured timeline.
func (s *Service) StopRecording(ctx context.Context) (RecordResultDTO, error) {
	_ = ctx
	s.mu.Lock()
	sess := s.recording
	if sess == nil {
		s.mu.Unlock()
		return RecordResultDTO{}, nil
	}
	s.recording = nil
	s.mu.Unlock()

	sess.rec.Stop()

	sess.mu.Lock()
	// Flush any cursor move still accumulating when recording stopped.
	flushed := sess.flushPendingMoveLocked()
	steps := append([]storage.MacroStep(nil), sess.steps...)
	duration := sess.lastAtMs
	sess.mu.Unlock()

	for _, e := range flushed {
		s.emitRecordStep(e)
	}

	dtos := make([]MacroStepDTO, 0, len(steps))
	for i := range steps {
		steps[i].OrderIndex = i
		dtos = append(dtos, stepDTO(steps[i]))
	}

	s.mu.Lock()
	s.recordState = RecordStateDTO{State: RecordStateStopped, StepCount: len(dtos), StartedAt: sess.startWall, UpdatedAt: nowMillis(), Message: "stopped"}
	state := s.recordState
	s.mu.Unlock()
	s.emitRecordState(state)
	logx.For("macro").Info("macro recording stopped", "steps", len(dtos))
	return RecordResultDTO{Steps: dtos, DurationMs: duration}, nil
}

// GetRecordState reports the current recording status.
func (s *Service) GetRecordState(ctx context.Context) (RecordStateDTO, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.recordState.State == "" {
		return RecordStateDTO{State: RecordStateIdle}, nil
	}
	return s.recordState, nil
}

// onCaptured converts a raw transition into a delay+key/mouse step pair and
// appends it to the active session, emitting a live step event. Consecutive
// cursor moves are coalesced into a single move step that is flushed before the
// next non-move event.
func (s *Service) onCaptured(sess *recordSession, ev capturedEvent) {
	sess.mu.Lock()
	if ev.move {
		// Drop cursor moves entirely when move capture is disabled so a
		// keyboard/click macro is not polluted by pointer drift.
		if !sess.captureMoves {
			sess.mu.Unlock()
			return
		}
		// Accumulate the net offset; defer emitting until a non-move event or stop.
		if !sess.pendingMove {
			sess.pendingMove = true
			sess.pendingMoveAt = ev.atMs
		}
		sess.pendingDx += ev.dx
		sess.pendingDy += ev.dy
		sess.mu.Unlock()
		return
	}
	emits := sess.flushPendingMoveLocked()
	emits = append(emits, sess.appendEventLocked(ev)...)
	sess.mu.Unlock()

	for _, e := range emits {
		s.emitRecordStep(e)
	}
}

// flushPendingMoveLocked materializes any accumulated cursor move into a move
// step. Caller must hold sess.mu. Returns the record-step events to emit after
// unlocking. A zero net offset is discarded.
func (sess *recordSession) flushPendingMoveLocked() []RecordStepEventDTO {
	if !sess.pendingMove {
		return nil
	}
	dx, dy, at := sess.pendingDx, sess.pendingDy, sess.pendingMoveAt
	sess.pendingMove = false
	sess.pendingDx, sess.pendingDy, sess.pendingMoveAt = 0, 0, 0
	if dx == 0 && dy == 0 {
		return nil
	}
	return sess.appendEventLocked(capturedEvent{move: true, mouse: true, dx: dx, dy: dy, atMs: at})
}

// appendEventLocked records the delay gap (if enabled) and the step for a single
// captured event. Caller must hold sess.mu. Returns the record-step events to
// emit after unlocking.
func (sess *recordSession) appendEventLocked(ev capturedEvent) []RecordStepEventDTO {
	var emits []RecordStepEventDTO
	// Insert a delay step to preserve the gap since the previous event, unless
	// the session was started with delay capture disabled. The delay step is
	// emitted as a live record-step event too, otherwise the front-end timeline
	// (which is built from these events) would drop every recorded delay.
	if sess.captureDelays {
		if gap := ev.atMs - sess.lastAtMs; gap > 0 && len(sess.steps) > 0 {
			delayStep := storage.MacroStep{Kind: StepDelay, WaitMs: clampDuration(gap), ModifierKeysJSON: "[]", PayloadJSON: "{}"}
			sess.steps = append(sess.steps, delayStep)
			delayIndex := len(sess.steps) - 1
			emits = append(emits, RecordStepEventDTO{StepIndex: delayIndex, Step: stepDTO(delayStep), At: nowMillis()})
		}
	}
	sess.lastAtMs = ev.atMs

	step := captureToStep(ev)
	sess.steps = append(sess.steps, step)
	index := len(sess.steps) - 1
	emits = append(emits, RecordStepEventDTO{StepIndex: index, Step: stepDTO(step), At: nowMillis()})
	return emits
}

func captureToStep(ev capturedEvent) storage.MacroStep {
	if ev.move {
		payload, _ := EncodeMousePayload(MousePayload{Dx: ev.dx, Dy: ev.dy})
		return storage.MacroStep{
			Kind:             StepMouseMove,
			DeviceKind:       DeviceMouse,
			ModifierKeysJSON: "[]",
			PayloadJSON:      payload,
			DurationMs:       mouseMoveDurationMs,
		}
	}
	if ev.wheel {
		// Scroll notch: one-shot, no up/down pairing.
		return storage.MacroStep{
			Kind:             StepMouseScroll,
			KeyLabel:         mouseButtonLabel(ev.button),
			VirtualKey:       ev.button,
			DeviceKind:       DeviceMouse,
			ModifierKeysJSON: "[]",
			PayloadJSON:      "{}",
		}
	}
	if ev.mouse {
		kind := StepMouseDown
		if ev.keyUp {
			kind = StepMouseUp
		}
		return storage.MacroStep{
			Kind:             kind,
			KeyLabel:         mouseButtonLabel(ev.button),
			VirtualKey:       ev.button,
			DeviceKind:       DeviceMouse,
			ModifierKeysJSON: "[]",
			PayloadJSON:      "{}",
		}
	}
	kind := StepKeyDown
	if ev.keyUp {
		kind = StepKeyUp
	}
	return storage.MacroStep{
		Kind:             kind,
		KeyLabel:         keyLabel(ev.vk, ev.scan),
		VirtualKey:       ev.vk,
		ScanCode:         ev.scan,
		DeviceKind:       DeviceKeyboard,
		ModifierKeysJSON: "[]",
		PayloadJSON:      "{}",
	}
}

func clampDuration(ms int64) int64 {
	if ms > maxDurationMs {
		return maxDurationMs
	}
	return ms
}

func (s *Service) recordError(err error) {
	s.mu.Lock()
	s.recording = nil
	s.recordState = RecordStateDTO{State: RecordStateError, UpdatedAt: nowMillis(), ErrorCode: "MACRO_RECORD_FAILED", Message: err.Error()}
	state := s.recordState
	s.mu.Unlock()
	s.emitRecordState(state)
	logx.For("macro").Error("macro recording failed", "error", err)
}

func (s *Service) emitRecordState(dto RecordStateDTO) {
	s.mu.Lock()
	emitter := s.emitter
	s.mu.Unlock()
	if emitter != nil {
		emitter.Emit(EventRecordSt, dto)
	}
}

func (s *Service) emitRecordStep(dto RecordStepEventDTO) {
	s.mu.Lock()
	emitter := s.emitter
	s.mu.Unlock()
	if emitter != nil {
		emitter.Emit(EventRecordStep, dto)
	}
}

// stopRecordingInternal is used by Stop() to tear down without returning data.
func (s *Service) stopRecordingInternal() {
	s.mu.Lock()
	sess := s.recording
	s.recording = nil
	s.mu.Unlock()
	if sess != nil {
		sess.rec.Stop()
	}
}

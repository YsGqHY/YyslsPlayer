//go:build completion

package macro

import (
	"context"
	"errors"
	"time"

	"YyslsPlayer/internal/services/hotkey"
	"YyslsPlayer/internal/services/keysim"
	"YyslsPlayer/internal/services/player"
	"YyslsPlayer/internal/utils/logx"
)

func (s *Service) RunMacro(ctx context.Context, id uint) (MacroStateDTO, error) {
	_ = ctx
	// Toggle mode: a second trigger on the currently running toggle macro stops it.
	s.mu.Lock()
	if s.current != nil && s.current.macroID == id && s.current.repeatMode == RepeatModeToggle {
		s.mu.Unlock()
		return s.stopRunning(context.Background(), StateStopped, "toggled-off")
	}
	recording := s.recording != nil
	s.mu.Unlock()

	if recording {
		return MacroStateDTO{}, ErrMacroRecording
	}

	if err := s.ensurePlayerIdle(); err != nil {
		return MacroStateDTO{}, err
	}
	detail, ok := s.store().GetMacroDetail(id)
	if !ok {
		return MacroStateDTO{}, ErrMacroNotFound
	}
	planned, err := planSteps(detail.Steps)
	if err != nil {
		return MacroStateDTO{}, err
	}
	sess := &runSession{
		macroID:    detail.Profile.ID,
		macroName:  detail.Profile.Name,
		steps:      detail.Steps,
		planned:    planned,
		startedAt:  nowMillis(),
		done:       make(chan struct{}),
		repeatMode: detail.Profile.RepeatMode,
		repeatN:    detail.Profile.RepeatCount,
		intervalMs: detail.Profile.RepeatIntervalMs,
	}
	if vk, ok := hotkey.AcceleratorMainVK(detail.Profile.TriggerAccelerator); ok {
		sess.triggerVK = vk
	}
	runCtx, cancel := context.WithCancel(context.Background())
	sess.cancel = cancel

	s.mu.Lock()
	if s.current != nil {
		if detail.Profile.InterruptPolicy != InterruptRestart {
			state := s.state
			s.mu.Unlock()
			return state, ErrMacroBusy
		}
		prev := s.current
		if prev.cancel != nil {
			prev.cancel()
		}
		s.state = MacroStateDTO{State: StateStopping, MacroID: prev.macroID, MacroName: prev.macroName, UpdatedAt: nowMillis(), Message: "stopping"}
		s.mu.Unlock()
		<-prev.done
		s.mu.Lock()
	}
	s.current = sess
	s.state = MacroStateDTO{State: StateRunning, MacroID: sess.macroID, MacroName: sess.macroName, StepIndex: -1, StepCount: len(sess.steps), StartedAt: sess.startedAt, UpdatedAt: sess.startedAt}
	state := s.state
	s.mu.Unlock()

	s.emitState(state)
	go s.run(runCtx, sess)
	return state, nil
}

func (s *Service) StopMacro(ctx context.Context) (MacroStateDTO, error) {
	return s.stopRunning(ctx, StateStopped, "stopped")
}

func (s *Service) stopRunning(ctx context.Context, finalState string, message string) (MacroStateDTO, error) {
	_ = ctx
	s.mu.Lock()
	sess := s.current
	if sess == nil {
		state := s.state
		s.mu.Unlock()
		return state, nil
	}
	if sess.cancel != nil {
		sess.cancel()
	}
	s.state = MacroStateDTO{State: StateStopping, MacroID: sess.macroID, MacroName: sess.macroName, StepIndex: s.state.StepIndex, StepCount: s.state.StepCount, StartedAt: sess.startedAt, UpdatedAt: nowMillis(), Message: "stopping"}
	stopping := s.state
	s.mu.Unlock()
	s.emitState(stopping)
	<-sess.done
	s.mu.Lock()
	state := s.state
	if state.State == StateStopping {
		state.State = finalState
		state.Message = message
		state.UpdatedAt = nowMillis()
		s.state = state
	}
	s.mu.Unlock()
	s.emitState(state)
	return state, nil
}

// passResult reports how a single macro pass ended.
type passResult int

const (
	passCompleted passResult = iota
	passStopped
	passFailed
)

func (s *Service) run(ctx context.Context, sess *runSession) {
	defer close(sess.done)
	defer func() {
		s.mu.Lock()
		if s.current == sess {
			s.current = nil
		}
		s.mu.Unlock()
	}()

	if err := s.keysim.RefreshChainHead(ctx, keysim.RunOptions{}); err != nil {
		s.fail(sess, err)
		return
	}

	iteration := 0
	for {
		if ctx.Err() != nil {
			s.stopped(sess)
			return
		}
		switch s.runOnce(ctx, sess) {
		case passFailed:
			return
		case passStopped:
			s.stopped(sess)
			return
		}
		iteration++

		more, wait := s.nextIteration(sess, iteration)
		if !more {
			break
		}
		if wait > 0 && !sleepContext(ctx, wait) {
			s.stopped(sess)
			return
		}
	}

	if _, err := s.keysim.ReleaseAll(context.Background(), keysim.RunOptions{}); err != nil {
		s.fail(sess, err)
		return
	}
	s.complete(sess)
}

// nextIteration decides whether to run another pass and how long to wait first.
func (s *Service) nextIteration(sess *runSession, completedPasses int) (bool, time.Duration) {
	interval := time.Duration(sess.intervalMs) * time.Millisecond
	switch sess.repeatMode {
	case RepeatModeCount:
		if completedPasses >= sess.repeatN {
			return false, 0
		}
		return true, interval
	case RepeatModeHold:
		if !triggerKeyDown(sess.triggerVK) {
			return false, 0
		}
		return true, interval
	case RepeatModeToggle:
		// Loop until an explicit stop (cancel) or second trigger arrives.
		return true, interval
	default: // RepeatModeOnce
		return false, 0
	}
}

func (s *Service) runOnce(ctx context.Context, sess *runSession) passResult {
	start := time.Now()
	for stepIndex, plannedStep := range sess.planned.steps {
		if !s.markStep(sess, stepIndex) {
			return passStopped
		}
		for _, actionIndex := range plannedStep.actionIndexes {
			if actionIndex < 0 || actionIndex >= len(sess.planned.actions) {
				continue
			}
			action := sess.planned.actions[actionIndex]
			wait := start.Add(time.Duration(action.TimeMs) * time.Millisecond).Sub(time.Now())
			if wait > 0 && !sleepContext(ctx, wait) {
				return passStopped
			}
			if ctx.Err() != nil {
				return passStopped
			}
			if _, err := s.keysim.Apply(ctx, action, keysim.RunOptions{}); err != nil {
				s.fail(sess, err)
				return passFailed
			}
		}
		wait := start.Add(time.Duration(plannedStep.endMs) * time.Millisecond).Sub(time.Now())
		if wait > 0 && !sleepContext(ctx, wait) {
			return passStopped
		}
	}
	return passCompleted
}

func (s *Service) markStep(sess *runSession, index int) bool {
	s.mu.Lock()
	if s.current != sess || s.state.State != StateRunning {
		s.mu.Unlock()
		return false
	}
	s.state.StepIndex = index
	s.state.UpdatedAt = nowMillis()
	state := s.state
	s.mu.Unlock()
	s.emitState(state)
	if index >= 0 && index < len(sess.steps) {
		s.emitStep(MacroStepEventDTO{MacroID: sess.macroID, StepIndex: index, Step: stepDTO(sess.steps[index]), At: nowMillis()})
	}
	return true
}

func (s *Service) complete(sess *runSession) {
	s.mu.Lock()
	if s.current != sess {
		s.mu.Unlock()
		return
	}
	s.state = MacroStateDTO{State: StateCompleted, MacroID: sess.macroID, MacroName: sess.macroName, StepIndex: len(sess.steps) - 1, StepCount: len(sess.steps), StartedAt: sess.startedAt, UpdatedAt: nowMillis(), Message: "completed"}
	state := s.state
	s.mu.Unlock()
	s.emitState(state)
	logx.For("macro").Info("macro completed", "macroId", sess.macroID, "steps", len(sess.steps))
}

func (s *Service) stopped(sess *runSession) {
	_, _ = s.keysim.ReleaseAll(context.Background(), keysim.RunOptions{})
	s.mu.Lock()
	if s.current != sess {
		s.mu.Unlock()
		return
	}
	s.state = MacroStateDTO{State: StateStopped, MacroID: sess.macroID, MacroName: sess.macroName, StepIndex: s.state.StepIndex, StepCount: len(sess.steps), StartedAt: sess.startedAt, UpdatedAt: nowMillis(), Message: "stopped"}
	state := s.state
	s.mu.Unlock()
	s.emitState(state)
}

func (s *Service) fail(sess *runSession, err error) {
	_, releaseErr := s.keysim.ReleaseAll(context.Background(), keysim.RunOptions{})
	combined := errors.Join(err, releaseErr)
	code := macroErrorCode(combined)
	s.mu.Lock()
	if s.current == sess {
		s.state = MacroStateDTO{State: StateError, MacroID: sess.macroID, MacroName: sess.macroName, StepIndex: s.state.StepIndex, StepCount: len(sess.steps), StartedAt: sess.startedAt, UpdatedAt: nowMillis(), ErrorCode: code, Message: combined.Error()}
	}
	state := s.state
	s.mu.Unlock()
	s.emitState(state)
	s.emitError(sess.macroID, code, combined.Error())
	logx.For("macro").Error("macro failed", "macroId", sess.macroID, "error", combined)
}

func (s *Service) ensurePlayerIdle() error {
	if s.player == nil {
		return nil
	}
	state, err := s.player.GetState(context.Background(), "")
	if err != nil {
		if errors.Is(err, player.ErrPlayerNotFound) {
			return nil
		}
		return err
	}
	if state.State == player.StatePlaying || state.State == player.StatePaused || state.State == player.StateReady {
		return ErrMacroPlayerActive
	}
	return nil
}

func sleepContext(ctx context.Context, delay time.Duration) bool {
	if delay <= 0 {
		return true
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func macroErrorCode(err error) string {
	switch {
	case errors.Is(err, keysim.ErrUnsupportedPlatform):
		return "KEYSIM_UNSUPPORTED_PLATFORM"
	case errors.Is(err, keysim.ErrInvalidKey):
		return "KEYSIM_INVALID_KEY"
	case errors.Is(err, ErrMacroPlayerActive):
		return "MACRO_PLAYER_ACTIVE"
	case errors.Is(err, ErrMacroRecording):
		return "MACRO_RECORDING_ACTIVE"
	case errors.Is(err, ErrMacroBusy):
		return "MACRO_BUSY"
	case errors.Is(err, ErrMacroNotFound):
		return "MACRO_NOT_FOUND"
	case errors.Is(err, ErrMacroNoSteps):
		return "MACRO_NO_STEPS"
	case errors.Is(err, ErrMacroInvalid):
		return "MACRO_INVALID"
	default:
		return "MACRO_ERROR"
	}
}

func (s *Service) emitState(dto MacroStateDTO) {
	s.mu.Lock()
	emitter := s.emitter
	s.mu.Unlock()
	if emitter != nil {
		emitter.Emit(EventState, dto)
	}
}

func (s *Service) emitStep(dto MacroStepEventDTO) {
	s.mu.Lock()
	emitter := s.emitter
	s.mu.Unlock()
	if emitter != nil {
		emitter.Emit(EventStep, dto)
	}
}

func (s *Service) emitError(macroID uint, code string, message string) {
	s.mu.Lock()
	emitter := s.emitter
	s.mu.Unlock()
	if emitter != nil {
		emitter.Emit(EventError, MacroErrorDTO{MacroID: macroID, ErrorCode: code, Message: message, At: nowMillis()})
	}
}

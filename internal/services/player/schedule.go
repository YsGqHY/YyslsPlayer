package player

import (
	"context"
	"errors"
	"fmt"
	"time"

	"YyslsPlayer/internal/services/keysim"
	"YyslsPlayer/internal/services/midi"
	"YyslsPlayer/internal/utils/logx"
)

type scheduleDecision int

const (
	decisionExit scheduleDecision = iota
	decisionPaused
	decisionWait
	decisionRun
	decisionComplete
)

func normalizeLookahead(value int) (int, error) {
	if value == 0 {
		return DefaultLookaheadMs, nil
	}
	if value < MinLookaheadMs || value > MaxLookaheadMs {
		return 0, fmt.Errorf("%w: %d", ErrInvalidLookahead, value)
	}
	return value, nil
}

func keyActionsFromPlan(plan midi.PlayPlanDTO) ([]keysim.KeyAction, error) {
	if len(plan.Frames) == 0 {
		return nil, ErrPlayPlanEmpty
	}
	actions := make([]keysim.KeyAction, 0, len(plan.Frames))
	for _, frame := range plan.Frames {
		action, err := keyActionFromFrame(frame)
		if err != nil {
			return nil, err
		}
		actions = append(actions, action)
	}
	return actions, nil
}

func playPlanDuration(durationMs int64, actions []keysim.KeyAction) int64 {
	duration := durationMs
	for _, action := range actions {
		if action.TimeMs > duration {
			duration = action.TimeMs
		}
	}
	return duration
}

func keyActionFromFrame(frame midi.KeyFrameDTO) (keysim.KeyAction, error) {
	var kind keysim.ActionKind
	switch frame.Action {
	case midi.KeyActionPress:
		kind = keysim.ActionPress
	case midi.KeyActionRelease:
		kind = keysim.ActionRelease
	default:
		return keysim.KeyAction{}, fmt.Errorf("%w: action=%s lane=%d", ErrInvalidKeyFrame, frame.Action, frame.Lane)
	}
	if frame.TimeMs < 0 {
		return keysim.KeyAction{}, fmt.Errorf("%w: negative timeMs=%d lane=%d", ErrInvalidKeyFrame, frame.TimeMs, frame.Lane)
	}
	key := keysim.Key{Label: frame.Key.Label, VirtualKey: frame.Key.VirtualKey, ScanCode: frame.Key.ScanCode}
	if key.ScanCode == 0 && key.VirtualKey == 0 {
		return keysim.KeyAction{}, fmt.Errorf("%w: missing key lane=%d", ErrInvalidKeyFrame, frame.Lane)
	}
	modifiers, err := keysim.DecodeModifiers(frame.Key.ModifierKeysJSON)
	if err != nil {
		return keysim.KeyAction{}, fmt.Errorf("%w: modifier lane=%d: %w", ErrInvalidKeyFrame, frame.Lane, err)
	}
	return keysim.KeyAction{
		TimeMs:         frame.TimeMs,
		Action:         kind,
		Lane:           frame.Lane,
		SourceNote:     frame.SourceNote,
		NormalizedNote: frame.NormalizedNote,
		Velocity:       frame.Velocity,
		Key:            key,
		Modifiers:      modifiers,
	}, nil
}

func (s *Service) runScheduler(ctx context.Context, sess *session) {
	defer close(sess.done)
	for {
		action, dryRun, wait, version, decision := s.nextScheduleStep(sess)
		switch decision {
		case decisionExit:
			return
		case decisionPaused:
			if !sleepOrDone(ctx, 5*time.Millisecond) {
				return
			}
		case decisionWait:
			if !sleepOrWake(ctx, sess, schedulerWait(wait, sess.lookaheadMs)) {
				return
			}
		case decisionRun:
			applied, err := s.applyScheduledAction(ctx, sess, action, dryRun, version)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				s.failSession(ctx, sess, err)
				return
			}
			if applied {
				s.advanceFrame(sess, action.TimeMs, version)
			}
		case decisionComplete:
			s.completeSession(ctx, sess)
			return
		}
	}
}

func (s *Service) nextScheduleStep(sess *session) (keysim.KeyAction, bool, time.Duration, uint64, scheduleDecision) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.current != sess {
		return keysim.KeyAction{}, false, 0, 0, decisionExit
	}
	switch sess.state {
	case StatePaused:
		return keysim.KeyAction{}, sess.dryRun, 0, sess.scheduleVersion, decisionPaused
	case StatePlaying:
		if sess.nextFrame >= len(sess.actions) {
			return keysim.KeyAction{}, sess.dryRun, 0, sess.scheduleVersion, decisionComplete
		}
		now := time.Now()
		s.refreshPositionLocked(sess, now)
		action := sess.actions[sess.nextFrame]
		delayMs := action.TimeMs - sess.positionMs
		if delayMs > 0 {
			return action, sess.dryRun, time.Duration(delayMs) * time.Millisecond, sess.scheduleVersion, decisionWait
		}
		return action, sess.dryRun, 0, sess.scheduleVersion, decisionRun
	default:
		return keysim.KeyAction{}, sess.dryRun, 0, sess.scheduleVersion, decisionExit
	}
}

func (s *Service) applyScheduledAction(ctx context.Context, sess *session, action keysim.KeyAction, dryRun bool, version uint64) (bool, error) {
	s.sendMu.Lock()
	defer s.sendMu.Unlock()
	if ctx.Err() != nil {
		return false, nil
	}
	s.mu.Lock()
	active := s.current == sess && sess.state == StatePlaying && sess.scheduleVersion == version
	s.mu.Unlock()
	if !active {
		return false, nil
	}
	_, err := s.keysim.Apply(ctx, action, keysim.RunOptions{DryRun: dryRun})
	return err == nil, err
}

func (s *Service) advanceFrame(sess *session, frameTimeMs int64, version uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.current != sess || sess.state != StatePlaying || sess.scheduleVersion != version || sess.nextFrame >= len(sess.actions) || sess.actions[sess.nextFrame].TimeMs != frameTimeMs {
		return
	}
	sess.nextFrame++
	if frameTimeMs > sess.positionMs {
		sess.positionMs = frameTimeMs
	}
	sess.updatedAt = unixMillis()
}

func (s *Service) completeSession(ctx context.Context, sess *session) {
	s.mu.Lock()
	if s.current != sess || sess.state != StatePlaying {
		s.mu.Unlock()
		return
	}
	sess.positionMs = sess.durationMs
	dryRun := sess.dryRun
	sessionID := sess.id
	s.mu.Unlock()

	if _, err := s.releaseAll(ctx, dryRun); err != nil {
		if ctx.Err() != nil {
			return
		}
		s.markError(sessionID, errorCode(err), err.Error())
		return
	}

	s.mu.Lock()
	if s.current != sess || sess.state != StatePlaying {
		s.mu.Unlock()
		return
	}
	sess.positionMs = sess.durationMs
	if err := s.transitionLocked(sess, StateCompleted, "completed"); err != nil {
		s.mu.Unlock()
		return
	}
	dto := stateDTO(sess)
	s.mu.Unlock()
	s.emitStateAndPosition(dto)
	logx.For("player").Info("player session completed", "sessionId", sessionID)
}

func (s *Service) failSession(ctx context.Context, sess *session, err error) {
	_, releaseErr := s.releaseAll(ctx, sess.dryRun)
	combined := errors.Join(err, releaseErr)
	s.markError(sess.id, errorCode(combined), combined.Error())
	logx.For("player").Error("player scheduler failed", "sessionId", sess.id, "error", combined)
}

func schedulerWait(wait time.Duration, lookaheadMs int) time.Duration {
	if wait <= 0 {
		return 0
	}
	lookahead := time.Duration(lookaheadMs) * time.Millisecond
	if wait > lookahead {
		return wait - lookahead
	}
	if wait > 2*time.Millisecond {
		return 2 * time.Millisecond
	}
	return wait
}

func sleepOrDone(ctx context.Context, delay time.Duration) bool {
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

func sleepOrWake(ctx context.Context, sess *session, delay time.Duration) bool {
	if delay <= 0 {
		return true
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-sess.wakeCh:
		return true
	case <-timer.C:
		return true
	}
}

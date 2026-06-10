package player

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"YyslsPlayer/internal/services/keysim"
	"YyslsPlayer/internal/services/midi"
	"YyslsPlayer/internal/utils/logx"
)

type Service struct {
	keysim  *keysim.Service
	mu      sync.Mutex
	sendMu  sync.Mutex
	eventMu sync.RWMutex
	emitter EventEmitter
	current *session
}

type session struct {
	id                 string
	plan               midi.PlayPlanDTO
	actions            []keysim.KeyAction
	nextFrame          int
	state              PlayerState
	positionMs         int64
	durationMs         int64
	dryRun             bool
	lookaheadMs        int
	errorCode          string
	message            string
	startedAt          int64
	updatedAt          int64
	startTime          time.Time
	pausedAt           time.Time
	pausedDuration     time.Duration
	startDelay         time.Duration
	chainHeadRefreshed bool
	cancel             context.CancelFunc
	done               chan struct{}
	wakeCh             chan struct{}
	scheduleVersion    uint64
}

func New(sim *keysim.Service) *Service {
	if sim == nil {
		sim = keysim.New(nil)
	}
	return &Service{keysim: sim}
}

func (s *Service) Start(_ context.Context, req StartRequest) (PlayerSessionDTO, error) {
	if len(req.Plan.Frames) == 0 {
		return PlayerSessionDTO{}, ErrPlayPlanEmpty
	}
	lookaheadMs, err := normalizeLookahead(req.LookaheadMs)
	if err != nil {
		return PlayerSessionDTO{}, err
	}
	actions, err := keyActionsFromPlan(req.Plan)
	if err != nil {
		return PlayerSessionDTO{}, err
	}

	runCtx, cancel := context.WithCancel(context.Background())
	now := unixMillis()
	durationMs := playPlanDuration(req.Plan.DurationMs, actions)
	startPositionMs := clampPosition(req.StartPositionMs, durationMs)
	startedAt := time.Now()
	startDelay := normalizeStartDelay(req.StartDelayMs)
	sess := &session{
		id:          newSessionID(),
		plan:        req.Plan,
		actions:     actions,
		nextFrame:   nextFrameIndex(actions, startPositionMs),
		state:       StateReady,
		positionMs:  startPositionMs,
		durationMs:  durationMs,
		dryRun:      resolveDryRun(req.DryRun),
		lookaheadMs: lookaheadMs,
		startedAt:   now,
		updatedAt:   now,
		startTime:   startedAt.Add(startDelay).Add(-time.Duration(startPositionMs) * time.Millisecond),
		startDelay:  startDelay,
		cancel:      cancel,
		done:        make(chan struct{}),
		wakeCh:      make(chan struct{}, 1),
	}
	s.mu.Lock()
	if s.current != nil && isActiveState(s.current.state) {
		s.mu.Unlock()
		cancel()
		return PlayerSessionDTO{}, ErrPlayerBusy
	}
	if err := s.transitionLocked(sess, StatePlaying, "started"); err != nil {
		s.mu.Unlock()
		cancel()
		return PlayerSessionDTO{}, err
	}
	s.current = sess
	dto := sessionDTO(sess)
	state := stateDTO(sess)
	s.mu.Unlock()

	s.emitStateAndPosition(state)
	go s.runPositionEmitter(runCtx, sess)
	go s.runScheduler(runCtx, sess)
	logx.For("player").Info("player session started", "sessionId", sess.id, "projectId", sess.plan.ProjectID, "profileId", sess.plan.ProfileID, "durationMs", sess.durationMs, "frames", len(sess.plan.Frames), "dryRun", sess.dryRun, "lookaheadMs", sess.lookaheadMs)
	return dto, nil
}

func (s *Service) Pause(ctx context.Context, sessionID string) (PlayerStateDTO, error) {
	s.mu.Lock()
	sess, err := s.currentSessionLocked(sessionID)
	if err != nil {
		s.mu.Unlock()
		return PlayerStateDTO{}, err
	}
	if !canTransition(sess.state, StatePaused) {
		s.mu.Unlock()
		return PlayerStateDTO{}, transitionError(sess.state, StatePaused)
	}
	now := time.Now()
	sess.positionMs = sess.elapsedMs(now)
	sess.nextFrame = nextFrameAfterPosition(sess.actions, sess.positionMs)
	sess.pausedAt = now
	if err := s.transitionLocked(sess, StatePaused, "paused"); err != nil {
		s.mu.Unlock()
		return PlayerStateDTO{}, err
	}
	dryRun := sess.dryRun
	dto := stateDTO(sess)
	s.mu.Unlock()
	s.emitStateAndPosition(dto)
	if _, releaseErr := s.releaseAll(ctx, dryRun); releaseErr != nil {
		return s.markError(dto.SessionID, "KEYSIM_RELEASE_FAILED", releaseErr.Error()), releaseErr
	}
	return dto, nil
}

func (s *Service) Resume(_ context.Context, sessionID string) (PlayerStateDTO, error) {
	s.mu.Lock()
	sess, err := s.currentSessionLocked(sessionID)
	if err != nil {
		s.mu.Unlock()
		return PlayerStateDTO{}, err
	}
	if !canTransition(sess.state, StatePlaying) {
		s.mu.Unlock()
		return PlayerStateDTO{}, transitionError(sess.state, StatePlaying)
	}
	if !sess.pausedAt.IsZero() {
		sess.pausedDuration += time.Since(sess.pausedAt)
		sess.pausedAt = time.Time{}
	}
	if err := s.transitionLocked(sess, StatePlaying, "resumed"); err != nil {
		s.mu.Unlock()
		return PlayerStateDTO{}, err
	}
	sess.chainHeadRefreshed = false
	dto := stateDTO(sess)
	s.mu.Unlock()
	s.emitStateAndPosition(dto)
	s.wakeSession(sess)
	return dto, nil
}

func (s *Service) Seek(ctx context.Context, sessionID string, positionMs int64) (PlayerStateDTO, error) {
	s.mu.Lock()
	sess, err := s.currentSessionLocked(sessionID)
	if err != nil {
		s.mu.Unlock()
		return PlayerStateDTO{}, err
	}
	if sess.state != StatePlaying && sess.state != StatePaused {
		state := sess.state
		s.mu.Unlock()
		return PlayerStateDTO{}, fmt.Errorf("%w: state=%s", ErrInvalidSeek, state)
	}
	wasPlaying := sess.state == StatePlaying
	dryRun := sess.dryRun
	nextPosition := clampPosition(positionMs, sess.durationMs)
	s.seekLocked(sess, nextPosition, time.Now())
	dto := stateDTO(sess)
	s.mu.Unlock()

	if _, releaseErr := s.releaseAll(ctx, dryRun); releaseErr != nil {
		return s.markError(dto.SessionID, "KEYSIM_RELEASE_FAILED", releaseErr.Error()), releaseErr
	}
	s.emitStateAndPosition(dto)
	if wasPlaying {
		s.wakeSession(sess)
	}
	return dto, nil
}

func (s *Service) Stop(ctx context.Context, sessionID string) (PlayerStateDTO, error) {
	sess, dryRun, cancel, dto, err := s.stopState(sessionID)
	if err != nil {
		return PlayerStateDTO{}, err
	}
	if cancel != nil {
		cancel()
	}
	s.emitStateAndPosition(dto)
	_, releaseErr := s.releaseAll(ctx, dryRun)
	if releaseErr != nil {
		return s.markError(sess.id, "KEYSIM_RELEASE_FAILED", releaseErr.Error()), releaseErr
	}
	logx.For("player").Info("player session stopped", "sessionId", sess.id)
	return dto, nil
}

func (s *Service) GetState(_ context.Context, sessionID string) (PlayerStateDTO, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.current == nil {
		if sessionID == "" {
			return PlayerStateDTO{State: StateIdle}, nil
		}
		return PlayerStateDTO{}, ErrPlayerNotFound
	}
	sess, err := s.currentSessionLocked(sessionID)
	if err != nil {
		return PlayerStateDTO{}, err
	}
	s.refreshPositionLocked(sess, time.Now())
	return stateDTO(sess), nil
}

func (s *Service) ReleaseAll(ctx context.Context) error {
	dryRun := false
	s.mu.Lock()
	if s.current != nil {
		dryRun = s.current.dryRun
	}
	s.mu.Unlock()
	_, err := s.releaseAll(ctx, dryRun)
	if err != nil {
		s.markCurrentError("KEYSIM_RELEASE_FAILED", err.Error())
	}
	return err
}

func (s *Service) releaseAll(ctx context.Context, dryRun bool) (keysim.RunResult, error) {
	s.sendMu.Lock()
	defer s.sendMu.Unlock()
	return s.keysim.ReleaseAll(ctx, keysim.RunOptions{DryRun: dryRun})
}

func (s *Service) wakeSession(sess *session) {
	if sess == nil || sess.wakeCh == nil {
		return
	}
	select {
	case sess.wakeCh <- struct{}{}:
	default:
	}
}

func (s *Service) stopState(sessionID string) (*session, bool, context.CancelFunc, PlayerStateDTO, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, err := s.currentSessionLocked(sessionID)
	if err != nil {
		return nil, false, nil, PlayerStateDTO{}, err
	}
	s.refreshPositionLocked(sess, time.Now())
	if sess.state != StateStopped {
		if err := s.transitionLocked(sess, StateStopped, "stopped"); err != nil {
			return nil, false, nil, PlayerStateDTO{}, err
		}
	}
	return sess, sess.dryRun, sess.cancel, stateDTO(sess), nil
}

func (s *Service) currentSessionLocked(sessionID string) (*session, error) {
	if s.current == nil {
		return nil, ErrPlayerNotFound
	}
	if sessionID != "" && s.current.id != sessionID {
		return nil, ErrPlayerNotFound
	}
	return s.current, nil
}

func (s *Service) transitionLocked(sess *session, next PlayerState, message string) error {
	if !canTransition(sess.state, next) {
		return transitionError(sess.state, next)
	}
	sess.state = next
	sess.message = message
	sess.updatedAt = unixMillis()
	return nil
}

func (s *Service) markError(sessionID string, code string, message string) PlayerStateDTO {
	s.mu.Lock()
	var dto PlayerStateDTO
	if s.current == nil || s.current.id != sessionID {
		dto = PlayerStateDTO{SessionID: sessionID, State: StateError, ErrorCode: code, Message: message}
	} else {
		s.refreshPositionLocked(s.current, time.Now())
		s.setErrorLocked(s.current, code, message)
		dto = stateDTO(s.current)
	}
	s.mu.Unlock()
	s.emitError(dto)
	return dto
}

func (s *Service) markCurrentError(code string, message string) PlayerStateDTO {
	s.mu.Lock()
	var dto PlayerStateDTO
	if s.current == nil {
		dto = PlayerStateDTO{State: StateError, ErrorCode: code, Message: message}
	} else {
		s.refreshPositionLocked(s.current, time.Now())
		s.setErrorLocked(s.current, code, message)
		dto = stateDTO(s.current)
	}
	s.mu.Unlock()
	s.emitError(dto)
	return dto
}

func (s *Service) setErrorLocked(sess *session, code string, message string) {
	sess.state = StateError
	sess.errorCode = code
	sess.message = message
	sess.updatedAt = unixMillis()
	if sess.cancel != nil {
		sess.cancel()
	}
}

func (s *Service) refreshPositionLocked(sess *session, now time.Time) {
	switch sess.state {
	case StatePlaying:
		sess.positionMs = sess.elapsedMs(now)
	case StatePaused:
		if !sess.pausedAt.IsZero() {
			sess.positionMs = sess.elapsedMs(sess.pausedAt)
		}
	}
	sess.positionMs = clampPosition(sess.positionMs, sess.durationMs)
}

func (s *Service) seekLocked(sess *session, positionMs int64, now time.Time) {
	sess.positionMs = clampPosition(positionMs, sess.durationMs)
	sess.nextFrame = nextFrameIndex(sess.actions, sess.positionMs)
	sess.startTime = now.Add(-time.Duration(sess.positionMs) * time.Millisecond)
	sess.pausedDuration = 0
	if sess.state == StatePaused {
		sess.pausedAt = now
	} else {
		sess.pausedAt = time.Time{}
	}
	if sess.state == StatePlaying {
		sess.chainHeadRefreshed = false
	}
	sess.message = "seeked"
	sess.updatedAt = unixMillis()
	sess.scheduleVersion++
}

func clampPosition(positionMs int64, durationMs int64) int64 {
	if positionMs < 0 {
		return 0
	}
	if durationMs > 0 && positionMs > durationMs {
		return durationMs
	}
	return positionMs
}

func nextFrameIndex(actions []keysim.KeyAction, positionMs int64) int {
	for index, action := range actions {
		if action.TimeMs >= positionMs {
			return index
		}
	}
	return len(actions)
}

func nextFrameAfterPosition(actions []keysim.KeyAction, positionMs int64) int {
	for index, action := range actions {
		if action.TimeMs > positionMs {
			return index
		}
	}
	return len(actions)
}

func (sess *session) elapsedMs(now time.Time) int64 {
	elapsed := now.Sub(sess.startTime) - sess.pausedDuration
	if elapsed < 0 {
		return 0
	}
	return elapsed.Milliseconds()
}

func sessionDTO(sess *session) PlayerSessionDTO {
	return PlayerSessionDTO{
		SessionID:   sess.id,
		State:       sess.state,
		PositionMs:  sess.positionMs,
		DurationMs:  sess.durationMs,
		DryRun:      sess.dryRun,
		LookaheadMs: sess.lookaheadMs,
		ErrorCode:   sess.errorCode,
		Message:     sess.message,
		ProjectID:   sess.plan.ProjectID,
		ProfileID:   sess.plan.ProfileID,
		FrameCount:  len(sess.plan.Frames),
		StartedAt:   sess.startedAt,
		UpdatedAt:   sess.updatedAt,
	}
}

func stateDTO(sess *session) PlayerStateDTO {
	return PlayerStateDTO{
		SessionID:   sess.id,
		State:       sess.state,
		PositionMs:  sess.positionMs,
		DurationMs:  sess.durationMs,
		DryRun:      sess.dryRun,
		LookaheadMs: sess.lookaheadMs,
		ErrorCode:   sess.errorCode,
		Message:     sess.message,
	}
}

func newSessionID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err == nil {
		return hex.EncodeToString(buf)
	}
	return fmt.Sprintf("session-%d", time.Now().UnixNano())
}

func unixMillis() int64 {
	return time.Now().UnixMilli()
}

func errorCode(err error) string {
	switch {
	case errors.Is(err, keysim.ErrUnsupportedPlatform):
		return "KEYSIM_UNSUPPORTED_PLATFORM"
	case errors.Is(err, keysim.ErrSendFailed):
		return "KEYSIM_SEND_FAILED"
	case errors.Is(err, keysim.ErrReleaseFailed):
		return "KEYSIM_RELEASE_FAILED"
	case errors.Is(err, keysim.ErrInvalidKey):
		return "KEYSIM_INVALID_KEY"
	case errors.Is(err, ErrInvalidKeyFrame):
		return "PLAYER_INVALID_KEYFRAME"
	default:
		return "PLAYER_ERROR"
	}
}

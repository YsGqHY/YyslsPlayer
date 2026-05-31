package player

import (
	"context"
	"time"
)

type EventEmitter interface {
	Emit(name string, payload any)
}

type EventEmitterFunc func(name string, payload any)

func (f EventEmitterFunc) Emit(name string, payload any) {
	f(name, payload)
}

//wails:ignore
func (s *Service) AttachEmitter(emitter EventEmitter) {
	s.eventMu.Lock()
	defer s.eventMu.Unlock()
	s.emitter = emitter
}

func (s *Service) emitEvent(name string, payload any) {
	s.eventMu.RLock()
	emitter := s.emitter
	s.eventMu.RUnlock()
	if emitter == nil {
		return
	}
	emitter.Emit(name, payload)
}

func (s *Service) emitState(dto PlayerStateDTO) {
	s.emitEvent(EventState, dto)
}

func (s *Service) emitPosition(dto PlayerPositionDTO) {
	s.emitEvent(EventPosition, dto)
}

func (s *Service) emitError(dto PlayerStateDTO) {
	s.emitState(dto)
	s.emitEvent(EventError, errorEventDTO(dto))
}

func (s *Service) emitStateAndPosition(dto PlayerStateDTO) {
	s.emitState(dto)
	s.emitPosition(positionEventDTO(dto))
}

func (s *Service) runPositionEmitter(ctx context.Context, sess *session) {
	ticker := time.NewTicker(time.Second / DefaultProgressHz)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			position, ok := s.currentPositionEvent(sess)
			if !ok {
				return
			}
			s.emitPosition(position)
		}
	}
}

func (s *Service) currentPositionEvent(sess *session) (PlayerPositionDTO, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.current != sess {
		return PlayerPositionDTO{}, false
	}
	switch sess.state {
	case StatePlaying, StatePaused:
		s.refreshPositionLocked(sess, time.Now())
		return positionEventDTO(stateDTO(sess)), true
	default:
		return PlayerPositionDTO{}, false
	}
}

func positionEventDTO(dto PlayerStateDTO) PlayerPositionDTO {
	progress := 0.0
	if dto.DurationMs > 0 {
		progress = float64(dto.PositionMs) / float64(dto.DurationMs)
		if progress < 0 {
			progress = 0
		}
		if progress > 1 {
			progress = 1
		}
	}
	return PlayerPositionDTO{
		SessionID:  dto.SessionID,
		State:      dto.State,
		PositionMs: dto.PositionMs,
		DurationMs: dto.DurationMs,
		Progress:   progress,
		UpdatedAt:  unixMillis(),
	}
}

func errorEventDTO(dto PlayerStateDTO) PlayerErrorDTO {
	return PlayerErrorDTO{
		SessionID:   dto.SessionID,
		State:       dto.State,
		PositionMs:  dto.PositionMs,
		DurationMs:  dto.DurationMs,
		DryRun:      dto.DryRun,
		LookaheadMs: dto.LookaheadMs,
		ErrorCode:   dto.ErrorCode,
		Message:     dto.Message,
		UpdatedAt:   unixMillis(),
	}
}

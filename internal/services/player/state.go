package player

import "fmt"

func isActiveState(state PlayerState) bool {
	switch state {
	case StateReady, StatePlaying, StatePaused:
		return true
	default:
		return false
	}
}

func canTransition(from PlayerState, to PlayerState) bool {
	if from == to && to == StateStopped {
		return true
	}
	switch from {
	case StateIdle, StateStopped, StateCompleted, StateError:
		return to == StateReady
	case StateReady:
		return to == StatePlaying || to == StateStopped
	case StatePlaying:
		return to == StatePaused || to == StateStopped || to == StateCompleted || to == StateError
	case StatePaused:
		return to == StatePlaying || to == StateStopped || to == StateError
	default:
		return false
	}
}

func transitionError(from PlayerState, to PlayerState) error {
	return fmt.Errorf("%w: %s -> %s", ErrInvalidTransition, from, to)
}

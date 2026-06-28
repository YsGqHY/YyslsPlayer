//go:build completion

package macro

const (
	EventState      = "macro:state"
	EventStep       = "macro:step"
	EventError      = "macro:error"
	EventRecordSt   = "macro:record:state"
	EventRecordStep = "macro:record:step"
)

type EventEmitter interface {
	Emit(name string, payload any)
}

type EventEmitterFunc func(name string, payload any)

func (f EventEmitterFunc) Emit(name string, payload any) { f(name, payload) }

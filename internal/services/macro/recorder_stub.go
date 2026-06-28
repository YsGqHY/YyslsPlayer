//go:build !windows && completion

package macro

// newKeyRecorder returns a no-op recorder on non-Windows platforms. Recording
// requires low-level Windows hooks, so Start fails fast elsewhere.
func newKeyRecorder() keyRecorder { return noopRecorder{} }

type noopRecorder struct{}

func (noopRecorder) Start(func(capturedEvent)) error { return ErrMacroRecordUnsup }
func (noopRecorder) Stop()                           {}

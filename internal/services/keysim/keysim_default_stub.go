//go:build !windows

package keysim

func NewDefaultDriver() Driver {
	return NewStubDriver()
}

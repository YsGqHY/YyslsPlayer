package keysim

import "context"

type stubDriver struct{}

func NewStubDriver() Driver {
	return stubDriver{}
}

func (stubDriver) Send(_ context.Context, _ KeyEvent, opts RunOptions) error {
	if opts.DryRun {
		return nil
	}
	return ErrUnsupportedPlatform
}

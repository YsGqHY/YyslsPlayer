//go:build !windows

package app

func EnsureAdminRelaunch() error {
	return nil
}

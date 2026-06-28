//go:build !windows && completion

package macro

// triggerKeyDown is a no-op on non-Windows platforms. Without a way to poll the
// physical key state, "hold to repeat" degrades to a single pass (reported as
// not held after the first run).
func triggerKeyDown(vk int) bool {
	_ = vk
	return false
}

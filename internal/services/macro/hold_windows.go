//go:build windows && completion

package macro

import "golang.org/x/sys/windows"

var (
	user32Macro          = windows.NewLazySystemDLL("user32.dll")
	procGetAsyncKeyState = user32Macro.NewProc("GetAsyncKeyState")
)

// triggerKeyDown reports whether the physical key identified by virtual key vk
// is currently held down. Used by the "hold to repeat" macro mode to keep
// replaying while the trigger key remains pressed.
//
// GetAsyncKeyState returns a SHORT whose high-order bit (0x8000) is set when the
// key is down. vk <= 0 means there is no resolvable trigger key, so we report
// false to avoid an infinite loop.
func triggerKeyDown(vk int) bool {
	if vk <= 0 {
		return false
	}
	ret, _, _ := procGetAsyncKeyState.Call(uintptr(vk))
	return ret&0x8000 != 0
}

//go:build windows

package keysim

const mapvkVsckToVk = 1 // MAPVK_VSC_TO_VK

var procMapVirtualKeyW = user32Hook.NewProc("MapVirtualKeyW")

func scanToVK(scanCode int) int {
	vk, _, _ := procMapVirtualKeyW.Call(uintptr(scanCode), mapvkVsckToVk)
	return int(vk)
}

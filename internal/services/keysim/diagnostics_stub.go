//go:build !windows

package keysim

func scanToVK(scanCode int) int { return 0 }

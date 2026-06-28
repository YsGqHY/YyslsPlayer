package keysim

// Key kinds. Empty string is treated as KeyKindKeyboard for backward
// compatibility with persisted keyframes and macro steps.
const (
	KeyKindKeyboard = "keyboard"
	KeyKindMouse    = "mouse"
)

// Mouse button identifiers carried in Key.VirtualKey when Key.Kind == KeyKindMouse.
// Values are internal abstractions, decoupled from the Windows VK_* button codes;
// the windows driver maps them to MOUSEEVENTF_* flags.
//
// Ids 1..5 are click buttons (press/release pairs). Ids 6..9 are scroll-wheel
// directions: one-shot notch events with no release, mirroring Logitech G HUB's
// "scroll up/down/left/right" macro actions.
const (
	MouseButtonLeft   = 1
	MouseButtonRight  = 2
	MouseButtonMiddle = 3
	MouseButtonX1     = 4
	MouseButtonX2     = 5

	MouseWheelUp    = 6
	MouseWheelDown  = 7
	MouseWheelLeft  = 8
	MouseWheelRight = 9
)

// IsMouse reports whether the key represents a mouse button or wheel direction
// rather than a keyboard key.
func (k Key) IsMouse() bool {
	return k.Kind == KeyKindMouse
}

// IsMouseWheel reports whether the key represents a scroll-wheel direction. Wheel
// events are one-shot (no release) and are never tracked in the pressed set.
func (k Key) IsMouseWheel() bool {
	return k.Kind == KeyKindMouse && isWheelButton(k.VirtualKey)
}

// validMouseButton reports whether button is a known mouse button or wheel id.
func validMouseButton(button int) bool {
	switch button {
	case MouseButtonLeft, MouseButtonRight, MouseButtonMiddle, MouseButtonX1, MouseButtonX2:
		return true
	default:
		return isWheelButton(button)
	}
}

// isWheelButton reports whether button is one of the scroll-wheel directions.
func isWheelButton(button int) bool {
	switch button {
	case MouseWheelUp, MouseWheelDown, MouseWheelLeft, MouseWheelRight:
		return true
	default:
		return false
	}
}

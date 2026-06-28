//go:build completion

package macro

import "strconv"

// assignableKeys returns the catalogue of keys/buttons the macro editor can
// assign to a step. Keyboard entries carry Windows virtual-key and scan-code
// pairs (scan codes with the 0xE000 prefix are extended keys); mouse entries
// carry the keysim mouse button id in VirtualKey and DeviceKind == "mouse".
func assignableKeys() []AssignableKeyDTO {
	keys := make([]AssignableKeyDTO, 0, 160)

	// Modifiers.
	keys = append(keys,
		kb("Ctrl", 17, 29, true),
		kb("Shift", 16, 42, true),
		kb("Alt", 18, 56, true),
		kb("Win", 91, 0xE05B, true),
		kb("CtrlRight", 163, 0xE01D, true),
		kb("ShiftRight", 161, 54, true),
		kb("AltRight", 165, 0xE038, true),
	)

	// Editing / navigation cluster.
	keys = append(keys,
		kb("Space", 32, 57, false),
		kb("Enter", 13, 28, false),
		kb("Tab", 9, 15, false),
		kb("Backspace", 8, 14, false),
		kb("Escape", 27, 1, false),
		kb("CapsLock", 20, 58, false),
		kb("PrintScreen", 44, 0xE037, false),
		kb("ScrollLock", 145, 70, false),
		kb("Pause", 19, 0xE046, false),
		kb("Insert", 45, 0xE052, false),
		kb("Delete", 46, 0xE053, false),
		kb("Home", 36, 0xE047, false),
		kb("End", 35, 0xE04F, false),
		kb("PageUp", 33, 0xE049, false),
		kb("PageDown", 34, 0xE051, false),
		kb("Left", 37, 0xE04B, false),
		kb("Up", 38, 0xE048, false),
		kb("Right", 39, 0xE04D, false),
		kb("Down", 40, 0xE050, false),
		kb("Apps", 93, 0xE05D, false),
	)

	// Punctuation / OEM keys.
	keys = append(keys,
		kb("`", 192, 41, false),
		kb("-", 189, 12, false),
		kb("=", 187, 13, false),
		kb("[", 219, 26, false),
		kb("]", 221, 27, false),
		kb("\\", 220, 43, false),
		kb(";", 186, 39, false),
		kb("'", 222, 40, false),
		kb(",", 188, 51, false),
		kb(".", 190, 52, false),
		kb("/", 191, 53, false),
	)

	// Letters A-Z.
	letterScans := map[rune]int{
		'A': 30, 'B': 48, 'C': 46, 'D': 32, 'E': 18, 'F': 33, 'G': 34, 'H': 35, 'I': 23, 'J': 36, 'K': 37, 'L': 38, 'M': 50,
		'N': 49, 'O': 24, 'P': 25, 'Q': 16, 'R': 19, 'S': 31, 'T': 20, 'U': 22, 'V': 47, 'W': 17, 'X': 45, 'Y': 21, 'Z': 44,
	}
	for c := 'A'; c <= 'Z'; c++ {
		keys = append(keys, kb(string(c), int(c), letterScans[c], false))
	}

	// Digits 0-9 (main row).
	digitScans := []int{11, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	for d := 0; d <= 9; d++ {
		keys = append(keys, kb(string(rune('0'+d)), 0x30+d, digitScans[d], false))
	}

	// Function keys F1-F24.
	for f := 1; f <= 24; f++ {
		keys = append(keys, kb("F"+strconv.Itoa(f), 0x6F+f, functionScanCode(f), false))
	}

	// Numpad.
	numpadScans := []int{82, 79, 80, 81, 75, 76, 77, 71, 72, 73} // 0..9
	for d := 0; d <= 9; d++ {
		keys = append(keys, kb("Numpad"+strconv.Itoa(d), 0x60+d, numpadScans[d], false))
	}
	keys = append(keys,
		kb("NumpadMultiply", 106, 55, false),
		kb("NumpadAdd", 107, 78, false),
		kb("NumpadSubtract", 109, 74, false),
		kb("NumpadDecimal", 110, 83, false),
		kb("NumpadDivide", 111, 0xE035, false),
		kb("NumpadEnter", 13, 0xE01C, false),
		kb("NumLock", 144, 69, false),
	)

	// Media keys.
	keys = append(keys,
		kb("VolumeMute", 173, 0xE020, false),
		kb("VolumeDown", 174, 0xE02E, false),
		kb("VolumeUp", 175, 0xE030, false),
		kb("MediaNext", 176, 0xE019, false),
		kb("MediaPrevious", 177, 0xE010, false),
		kb("MediaStop", 178, 0xE024, false),
		kb("MediaPlayPause", 179, 0xE022, false),
		kb("LaunchMail", 180, 0xE06C, false),
		kb("LaunchMediaSelect", 181, 0xE06D, false),
		kb("LaunchApp1", 182, 0xE06B, false),
		kb("LaunchApp2", 183, 0xE021, false),
	)

	// Browser keys.
	keys = append(keys,
		kb("BrowserBack", 166, 0xE06A, false),
		kb("BrowserForward", 167, 0xE069, false),
		kb("BrowserRefresh", 168, 0xE067, false),
		kb("BrowserStop", 169, 0xE068, false),
		kb("BrowserSearch", 170, 0xE065, false),
		kb("BrowserFavorites", 171, 0xE066, false),
		kb("BrowserHome", 172, 0xE032, false),
	)

	// IME / language keys.
	keys = append(keys,
		kb("KanaMode", 21, 0, false),
		kb("HanjaMode", 25, 0, false),
		kb("Convert", 28, 0, false),
		kb("NonConvert", 29, 0, false),
		kb("ImeOn", 22, 0, false),
		kb("ImeOff", 26, 0, false),
	)

	// Mouse buttons (DeviceKind == "mouse"; VirtualKey carries the keysim button id).
	keys = append(keys,
		mouse("Mouse Left", mouseButtonLeft),
		mouse("Mouse Right", mouseButtonRight),
		mouse("Mouse Middle", mouseButtonMiddle),
		mouse("Mouse X1", mouseButtonX1),
		mouse("Mouse X2", mouseButtonX2),
	)

	// Scroll-wheel directions (DeviceKind == "mouse"; one-shot notch events).
	keys = append(keys,
		mouse("Wheel Up", mouseWheelUp),
		mouse("Wheel Down", mouseWheelDown),
		mouse("Wheel Left", mouseWheelLeft),
		mouse("Wheel Right", mouseWheelRight),
	)

	return keys
}

// Mouse button / wheel ids mirror keysim.MouseButton* / keysim.MouseWheel*
// without importing keysim here so the catalogue stays a pure data table;
// validate.go enforces the mapping.
const (
	mouseButtonLeft   = 1
	mouseButtonRight  = 2
	mouseButtonMiddle = 3
	mouseButtonX1     = 4
	mouseButtonX2     = 5

	mouseWheelUp    = 6
	mouseWheelDown  = 7
	mouseWheelLeft  = 8
	mouseWheelRight = 9
)

func kb(label string, vk, scan int, modifier bool) AssignableKeyDTO {
	return AssignableKeyDTO{Label: label, VirtualKey: vk, ScanCode: scan, Modifier: modifier, DeviceKind: DeviceKeyboard}
}

func mouse(label string, button int) AssignableKeyDTO {
	return AssignableKeyDTO{Label: label, VirtualKey: button, ScanCode: 0, Modifier: false, DeviceKind: DeviceMouse}
}

func functionScanCode(f int) int {
	scans := []int{0, 59, 60, 61, 62, 63, 64, 65, 66, 67, 68, 87, 88}
	if f >= 1 && f < len(scans) {
		return scans[f]
	}
	return 0
}

// keyLabel resolves a captured (vk, scan) pair back to a catalogue label so
// recorded steps display meaningfully. Falls back to a VK hex token.
func keyLabel(vk, scan int) string {
	for _, k := range assignableKeys() {
		if k.DeviceKind != DeviceKeyboard {
			continue
		}
		if k.VirtualKey == vk && (scan == 0 || k.ScanCode == 0 || k.ScanCode == scan) {
			return k.Label
		}
	}
	if vk > 0 {
		return "VK0x" + strconv.FormatInt(int64(vk), 16)
	}
	return "Unknown"
}

// mouseButtonLabel resolves a keysim mouse button id to its catalogue label.
func mouseButtonLabel(button int) string {
	switch button {
	case mouseButtonLeft:
		return "Mouse Left"
	case mouseButtonRight:
		return "Mouse Right"
	case mouseButtonMiddle:
		return "Mouse Middle"
	case mouseButtonX1:
		return "Mouse X1"
	case mouseButtonX2:
		return "Mouse X2"
	case mouseWheelUp:
		return "Wheel Up"
	case mouseWheelDown:
		return "Wheel Down"
	case mouseWheelLeft:
		return "Wheel Left"
	case mouseWheelRight:
		return "Wheel Right"
	default:
		return "Mouse"
	}
}

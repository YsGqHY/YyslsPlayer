package hotkey

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// keyDef 描述一个可绑定的"主键"：规范名 + Win32 虚拟键码。
type keyDef struct {
	name string
	vk   int
}

// mainKeys 是支持的主键集合。
// 用切片保持稳定的解析与展示顺序；名称大小写不敏感。
var mainKeys = func() []keyDef {
	defs := []keyDef{
		{"Cancel", 0x03},
		{"Backspace", 0x08},
		{"Tab", 0x09},
		{"Clear", 0x0C},
		{"Enter", 0x0D},
		{"Pause", 0x13},
		{"CapsLock", 0x14},
		{"KanaMode", 0x15},
		{"ImeOn", 0x16},
		{"JunjaMode", 0x17},
		{"FinalMode", 0x18},
		{"HanjaMode", 0x19},
		{"ImeOff", 0x1A},
		{"Escape", 0x1B},
		{"Convert", 0x1C},
		{"NonConvert", 0x1D},
		{"Accept", 0x1E},
		{"ModeChange", 0x1F},
		{"Space", 0x20},
		{"PageUp", 0x21},
		{"PageDown", 0x22},
		{"End", 0x23},
		{"Home", 0x24},
		{"Left", 0x25},
		{"Up", 0x26},
		{"Right", 0x27},
		{"Down", 0x28},
		{"Select", 0x29},
		{"Print", 0x2A},
		{"Execute", 0x2B},
		{"PrintScreen", 0x2C},
		{"Insert", 0x2D},
		{"Delete", 0x2E},
		{"Help", 0x2F},
		{"NumLock", 0x90},
		{"ScrollLock", 0x91},
		{"ShiftLeft", 0xA0},
		{"ShiftRight", 0xA1},
		{"ControlLeft", 0xA2},
		{"ControlRight", 0xA3},
		{"AltLeft", 0xA4},
		{"AltRight", 0xA5},
		{"Numpad0", 0x60},
		{"Numpad1", 0x61},
		{"Numpad2", 0x62},
		{"Numpad3", 0x63},
		{"Numpad4", 0x64},
		{"Numpad5", 0x65},
		{"Numpad6", 0x66},
		{"Numpad7", 0x67},
		{"Numpad8", 0x68},
		{"Numpad9", 0x69},
		{"NumpadMultiply", 0x6A},
		{"NumpadAdd", 0x6B},
		{"NumpadSeparator", 0x6C},
		{"NumpadSubtract", 0x6D},
		{"NumpadDecimal", 0x6E},
		{"NumpadDivide", 0x6F},
		{"Semicolon", 0xBA},
		{"Equal", 0xBB},
		{"Comma", 0xBC},
		{"Minus", 0xBD},
		{"Period", 0xBE},
		{"Slash", 0xBF},
		{"Backquote", 0xC0},
		{"BracketLeft", 0xDB},
		{"Backslash", 0xDC},
		{"BracketRight", 0xDD},
		{"Quote", 0xDE},
		{"Oem8", 0xDF},
		{"IntlYen", 0xDC},
		{"OemAX", 0xE1},
		{"IntlBackslash", 0xE2},
		{"ProcessKey", 0xE5},
		{"Packet", 0xE7},
		{"MetaLeft", 0x5B},
		{"MetaRight", 0x5C},
		{"Apps", 0x5D},
		{"Sleep", 0x5F},
		{"Attn", 0xF6},
		{"CrSel", 0xF7},
		{"ExSel", 0xF8},
		{"EraseEOF", 0xF9},
		{"Play", 0xFA},
		{"Zoom", 0xFB},
		{"Pa1", 0xFD},
		{"OemClear", 0xFE},
		{"BrowserBack", 0xA6},
		{"BrowserForward", 0xA7},
		{"BrowserRefresh", 0xA8},
		{"BrowserStop", 0xA9},
		{"BrowserSearch", 0xAA},
		{"BrowserFavorites", 0xAB},
		{"BrowserHome", 0xAC},
		{"VolumeMute", 0xAD},
		{"VolumeDown", 0xAE},
		{"VolumeUp", 0xAF},
		{"MediaNext", 0xB0},
		{"MediaPrevious", 0xB1},
		{"MediaStop", 0xB2},
		{"MediaPlayPause", 0xB3},
		{"LaunchMail", 0xB4},
		{"LaunchMediaSelect", 0xB5},
		{"LaunchApp1", 0xB6},
		{"LaunchApp2", 0xB7},
	}
	// A-Z
	for c := 'A'; c <= 'Z'; c++ {
		defs = append(defs, keyDef{string(c), int(c)})
	}
	// 0-9
	for d := 0; d <= 9; d++ {
		defs = append(defs, keyDef{fmt.Sprintf("%d", d), 0x30 + d})
	}
	// F1-F24
	for f := 1; f <= 24; f++ {
		defs = append(defs, keyDef{fmt.Sprintf("F%d", f), 0x6F + f}) // F1 = 0x70
	}
	return defs
}()

// vkByName / nameByVK 是 mainKeys 的索引。
var (
	vkByName   = map[string]keyDef{}
	nameByVKID = map[int]string{}
)

func init() {
	for _, d := range mainKeys {
		vkByName[strings.ToLower(d.name)] = d
		nameByVKID[d.vk] = d.name
	}
	aliases := map[string]string{
		"esc":                "Escape",
		"pgup":               "PageUp",
		"page up":            "PageUp",
		"pgdn":               "PageDown",
		"page down":          "PageDown",
		"print":              "PrintScreen",
		"prtsc":              "PrintScreen",
		"prtscr":             "PrintScreen",
		"printscreen":        "PrintScreen",
		"hangulmode":         "KanaMode",
		"lang1":              "KanaMode",
		"kanjimode":          "HanjaMode",
		"lang2":              "HanjaMode",
		"ime on":             "ImeOn",
		"imeoff":             "ImeOff",
		"ime off":            "ImeOff",
		"hangul":             "KanaMode",
		"hanja":              "HanjaMode",
		"contextmenu":        "Apps",
		"menu":               "Apps",
		"volume mute":        "VolumeMute",
		"volume down":        "VolumeDown",
		"volume up":          "VolumeUp",
		"medianexttrack":     "MediaNext",
		"mediaprevioustrack": "MediaPrevious",
		"mediaplaypause":     "MediaPlayPause",
		"mediastop":          "MediaStop",
		"launchmediaplayer":  "LaunchMediaSelect",
		"mediaselect":        "LaunchMediaSelect",
		"apps":               "Apps",
		"application":        "Apps",
		"multiply":           "NumpadMultiply",
		"add":                "NumpadAdd",
		"subtract":           "NumpadSubtract",
		"decimal":            "NumpadDecimal",
		"divide":             "NumpadDivide",
		"num*":               "NumpadMultiply",
		"num+":               "NumpadAdd",
		"num-":               "NumpadSubtract",
		"num.":               "NumpadDecimal",
		"num/":               "NumpadDivide",
	}
	for i := 0; i <= 9; i++ {
		aliases[fmt.Sprintf("num%d", i)] = fmt.Sprintf("Numpad%d", i)
		aliases[fmt.Sprintf("numpad%d", i)] = fmt.Sprintf("Numpad%d", i)
	}
	for alias, canonical := range aliases {
		aliasKey := strings.ToLower(alias)
		if _, exists := vkByName[aliasKey]; exists {
			continue
		}
		if def, ok := vkByName[strings.ToLower(canonical)]; ok {
			vkByName[aliasKey] = def
		}
	}
}

// isFunctionKey 判断 vk 是否为功能键 F1..F24。
func isFunctionKey(vk int) bool {
	return vk >= 0x70 && vk <= 0x87
}

// accelerator 是解析后的组合：修饰位掩码 + 主键 vk + 规范化文本。
type accelerator struct {
	modifiers int    // ModAlt | ModControl | ...（不含 ModNoRepeat）
	vk        int    // 主键虚拟键码
	keyName   string // 主键规范名
	text      string // 规范化后的可读组合
}

// parseAccelerator 解析 "Ctrl+Alt+Backspace" 这类组合。
//
// 规则：
//   - 用 '+' 分隔，忽略空白与大小写
//   - 修饰键别名：Ctrl/Control、Alt/Option、Shift、Win/Cmd/Meta/Super
//   - 必须恰好一个主键（mainKeys 内）
//
// 返回规范化后的 accelerator；无法解析返回 ErrInvalidAccelerator。
func parseAccelerator(raw string) (accelerator, error) {
	if strings.TrimSpace(raw) == "" {
		return accelerator{}, ErrInvalidAccelerator
	}
	parts := strings.Split(raw, "+")
	var (
		mods    int
		mainDef *keyDef
	)
	for _, part := range parts {
		token := strings.TrimSpace(part)
		if token == "" {
			return accelerator{}, ErrInvalidAccelerator
		}
		switch strings.ToLower(token) {
		case "ctrl", "control":
			mods |= ModControl
			continue
		case "alt", "option":
			mods |= ModAlt
			continue
		case "shift":
			mods |= ModShift
			continue
		case "win", "cmd", "meta", "super":
			mods |= ModWin
			continue
		}
		// 非修饰键 —— 必须是唯一主键
		def, ok := vkByName[strings.ToLower(token)]
		if !ok {
			var parsed bool
			def, parsed = parseVirtualKeyToken(token)
			if !parsed {
				return accelerator{}, fmt.Errorf("%w: %s", ErrInvalidAccelerator, token)
			}
		}
		if mainDef != nil {
			return accelerator{}, fmt.Errorf("%w: multiple main keys", ErrInvalidAccelerator)
		}
		d := def
		mainDef = &d
	}
	if mainDef == nil {
		return accelerator{}, fmt.Errorf("%w: missing main key", ErrInvalidAccelerator)
	}
	acc := accelerator{modifiers: mods, vk: mainDef.vk, keyName: mainDef.name}
	acc.text = formatAccelerator(mods, mainDef.name)
	return acc, nil
}

func isKeyboardVirtualKey(vk int) bool {
	if vk <= 0 || vk > 0xFE {
		return false
	}
	// 鼠标按钮与未定义槽位不属于键盘触发键。
	return vk == 0x03 || vk > 0x07
}

func parseVirtualKeyToken(token string) (keyDef, bool) {
	raw := strings.TrimSpace(token)
	if len(raw) < 3 || !strings.EqualFold(raw[:2], "VK") {
		return keyDef{}, false
	}
	value := raw[2:]
	base := 10
	if len(value) > 2 && (strings.HasPrefix(value, "0x") || strings.HasPrefix(value, "0X")) {
		value = value[2:]
		base = 16
	}
	vk, err := strconv.ParseInt(value, base, 32)
	if err != nil || !isKeyboardVirtualKey(int(vk)) {
		return keyDef{}, false
	}
	return keyDef{name: fmt.Sprintf("VK0x%02X", vk), vk: int(vk)}, true
}

// formatAccelerator 把修饰位 + 主键名拼成规范文本，顺序固定 Ctrl→Alt→Shift→Win。
func formatAccelerator(mods int, keyName string) string {
	var out []string
	if mods&ModControl != 0 {
		out = append(out, "Ctrl")
	}
	if mods&ModAlt != 0 {
		out = append(out, "Alt")
	}
	if mods&ModShift != 0 {
		out = append(out, "Shift")
	}
	if mods&ModWin != 0 {
		out = append(out, "Win")
	}
	out = append(out, keyName)
	return strings.Join(out, "+")
}

// isSafeForGlobal 判断组合是否适合注册为 OS 级全局热键。
//
// 拒绝"会污染系统输入"的组合：没有 Ctrl/Alt/Win 修饰、且主键是普通键
// （字母 / 数字 / Space / Enter / Tab / 方向键等）。这类裸键被全局注册后，
// 用户在任何程序里按它都会被本应用吞掉。
//
// 放行条件（满足其一）：
//   - 含 Ctrl / Alt / Win 任一修饰键（Shift 单独不算 —— Shift+A 仍是普通输入）
//   - 主键是功能键 F1..F24（这些键很少用于文本输入）
func isSafeForGlobal(acc accelerator) bool {
	if acc.modifiers&(ModControl|ModAlt|ModWin) != 0 {
		return true
	}
	return isFunctionKey(acc.vk)
}

// NormalizeAccelerator 解析 + 安全校验，返回规范化文本。
// 解析失败返回 ErrInvalidAccelerator；不安全返回 ErrUnsafeAccelerator。
func NormalizeAccelerator(raw string) (string, error) {
	acc, err := normalizeAccelerator(raw)
	if err != nil {
		return "", err
	}
	return acc.text, nil
}

// NormalizeAcceleratorAllowUnsafe 与 NormalizeAccelerator 相同，但放行"裸普通键"
// （无 Ctrl/Alt/Win 修饰且非功能键）。仅供用户显式开启"允许单键触发"的场景使用，
// 调用方需自行承担全局吞键的副作用。解析失败仍返回 ErrInvalidAccelerator。
func NormalizeAcceleratorAllowUnsafe(raw string) (string, error) {
	acc, err := parseAccelerator(raw)
	if err != nil {
		return "", err
	}
	return acc.text, nil
}

// AcceleratorMainVK 返回组合键的主键虚拟键码，供"按住重复"等需要轮询触发键
// 物理状态的场景使用。解析失败返回 (0, false)。
func AcceleratorMainVK(raw string) (int, bool) {
	acc, err := parseAccelerator(raw)
	if err != nil {
		return 0, false
	}
	return acc.vk, true
}

// normalizeAccelerator 解析 + 安全校验，返回规范化结构。
// 解析失败返回 ErrInvalidAccelerator；不安全返回 ErrUnsafeAccelerator。
func normalizeAccelerator(raw string) (accelerator, error) {
	return normalizeAcceleratorWithPolicy(raw, false)
}

// normalizeAcceleratorWithPolicy 在 normalizeAccelerator 基础上允许调用方放行裸普通键。
// allowUnsafe 为 true 时跳过 isSafeForGlobal 校验，仅做解析。
func normalizeAcceleratorWithPolicy(raw string, allowUnsafe bool) (accelerator, error) {
	acc, err := parseAccelerator(raw)
	if err != nil {
		return accelerator{}, err
	}
	if !allowUnsafe && !isSafeForGlobal(acc) {
		return accelerator{}, fmt.Errorf("%w: %s", ErrUnsafeAccelerator, acc.text)
	}
	return acc, nil
}

func acceleratorIdentity(acc accelerator) string {
	return fmt.Sprintf("%d:%d", acc.modifiers, acc.vk)
}

// findConflicts 在一组绑定中找出重复使用同一组合的动作 ID。
// 仅对 enabled 的绑定判定，返回按动作 ID 排序的冲突分组。
func findConflicts(bindings map[string]string, enabled map[string]bool) [][]string {
	byIdentity := map[string][]string{}
	for actionID, raw := range bindings {
		if !enabled[actionID] {
			continue
		}
		acc, err := parseAccelerator(raw)
		if err != nil {
			continue
		}
		byIdentity[acceleratorIdentity(acc)] = append(byIdentity[acceleratorIdentity(acc)], actionID)
	}
	var groups [][]string
	for _, ids := range byIdentity {
		if len(ids) > 1 {
			sort.Strings(ids)
			groups = append(groups, ids)
		}
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i][0] < groups[j][0] })
	return groups
}

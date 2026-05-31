package hotkey

import (
	"fmt"
	"sort"
	"strings"
)

// keyDef 描述一个可绑定的"主键"：规范名 + Win32 虚拟键码。
type keyDef struct {
	name string
	vk   int
}

// mainKeys 是支持的主键集合（不含修饰键本身）。
// 用切片保持稳定的解析与展示顺序；名称大小写不敏感。
var mainKeys = func() []keyDef {
	defs := []keyDef{
		{"Space", 0x20},
		{"Enter", 0x0D},
		{"Tab", 0x09},
		{"Backspace", 0x08},
		{"Escape", 0x1B},
		{"Insert", 0x2D},
		{"Delete", 0x2E},
		{"Home", 0x24},
		{"End", 0x23},
		{"PageUp", 0x21},
		{"PageDown", 0x22},
		{"Left", 0x25},
		{"Up", 0x26},
		{"Right", 0x27},
		{"Down", 0x28},
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
			return accelerator{}, fmt.Errorf("%w: %s", ErrInvalidAccelerator, token)
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

// normalizeAccelerator 解析 + 安全校验，返回规范化文本。
// 解析失败返回 ErrInvalidAccelerator；不安全返回 ErrUnsafeAccelerator。
func normalizeAccelerator(raw string) (accelerator, error) {
	acc, err := parseAccelerator(raw)
	if err != nil {
		return accelerator{}, err
	}
	if !isSafeForGlobal(acc) {
		return accelerator{}, fmt.Errorf("%w: %s", ErrUnsafeAccelerator, acc.text)
	}
	return acc, nil
}

// findConflicts 在一组绑定中找出重复使用同一组合的动作 ID。
// 仅对 enabled 的绑定判定，返回按动作 ID 排序的冲突分组。
func findConflicts(bindings map[string]string, enabled map[string]bool) [][]string {
	byText := map[string][]string{}
	for actionID, raw := range bindings {
		if !enabled[actionID] {
			continue
		}
		acc, err := parseAccelerator(raw)
		if err != nil {
			continue
		}
		byText[acc.text] = append(byText[acc.text], actionID)
	}
	var groups [][]string
	for _, ids := range byText {
		if len(ids) > 1 {
			sort.Strings(ids)
			groups = append(groups, ids)
		}
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i][0] < groups[j][0] })
	return groups
}

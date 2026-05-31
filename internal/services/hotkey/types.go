// Package hotkey 提供 OS 级全局快捷键：即使焦点不在本应用（例如切到
// 《燕云十六声》游戏窗口），也能控制演奏。
//
// 与 Wails app.KeyBinding 的区别：后者仅在应用窗口聚焦时触发，无法满足
// "切到游戏也生效"。本包在 Windows 上用 RegisterHotKey 注册 OS 级热键，
// 配合独立线程的 GetMessage 消息循环接收 WM_HOTKEY。
package hotkey

import "errors"

// 动作 ID：与前端 settings.shortcuts.actions.<id> 文案、HotkeyBinding.ActionID 对齐。
const (
	ActionPlayPause        = "play-pause"
	ActionStop             = "stop"
	ActionPreviewToggle    = "preview-toggle"
	ActionEmergencyRelease = "emergency-release"

	// EventTriggered 是后端→前端的事件名；与 frontend/src/shared/events.ts 对齐。
	EventTriggered = "hotkey:triggered"
)

// Win32 修饰位（与 user32 RegisterHotKey 的 fsModifiers 对齐）。
const (
	ModAlt      = 0x0001
	ModControl  = 0x0002
	ModShift    = 0x0004
	ModWin      = 0x0008
	ModNoRepeat = 0x4000
)

// 错误码（前端按 errorCode 选择 i18n 文案）。
var (
	// ErrUnknownAction 动作 ID 不在已知集合内。
	ErrUnknownAction = errors.New("HOTKEY_UNKNOWN_ACTION")
	// ErrUnsafeAccelerator 组合会污染系统输入（无修饰裸普通键），拒绝写入。
	ErrUnsafeAccelerator = errors.New("HOTKEY_UNSAFE_ACCELERATOR")
	// ErrInvalidAccelerator 无法解析的组合文本。
	ErrInvalidAccelerator = errors.New("HOTKEY_INVALID_ACCELERATOR")
)

// per-binding 运行时错误码（字符串，随 BindingDTO 返回给前端）。
const (
	CodeAlreadyRegistered = "HOTKEY_ALREADY_REGISTERED"
	CodeRegisterFailed    = "HOTKEY_REGISTER_FAILED"
)

// BindingDTO 是单个动作的绑定快照（含运行时注册状态）。
type BindingDTO struct {
	ActionID    string `json:"actionId"`
	Accelerator string `json:"accelerator"`
	Enabled     bool   `json:"enabled"`
	// Registered 表示当前是否已成功注册到 OS。Enabled 但 Registered=false
	// 通常意味着该组合被其它程序占用（见 ErrorCode）。
	Registered bool `json:"registered"`
	// ErrorCode 注册失败原因；为空表示无错误。
	ErrorCode string `json:"errorCode"`
}

// StateDTO 是整体快照，前端一次拉取渲染。
type StateDTO struct {
	// Supported 表示当前 Windows 热键管理器是否可用。
	Supported bool         `json:"supported"`
	Bindings  []BindingDTO `json:"bindings"`
}

// TriggeredDTO 是热键触发事件载荷。
type TriggeredDTO struct {
	ActionID    string `json:"actionId"`
	Accelerator string `json:"accelerator"`
	// HandledByBackend=true 表示后端已直接对 player 执行了动作（stop / 紧急松键 /
	// 暂停继续），前端无需再调用，只用于 UI 反馈 / 导航类动作。
	HandledByBackend bool  `json:"handledByBackend"`
	At               int64 `json:"at"`
}

// defaultBinding 是单条默认快捷键定义。
type defaultBinding struct {
	ActionID    string
	Accelerator string
}

// defaultBindings 默认快捷键表。
//
// 全部使用功能键或带修饰键的组合，绝不用裸普通键 —— OS 级热键会在系统范围
// 吞掉该键，注册裸 Space / 字母会导致用户在任何程序里都无法输入。
var defaultBindings = []defaultBinding{
	{ActionID: ActionPlayPause, Accelerator: "F9"},
	{ActionID: ActionStop, Accelerator: "F10"},
	{ActionID: ActionPreviewToggle, Accelerator: "F11"},
	{ActionID: ActionEmergencyRelease, Accelerator: "Ctrl+Alt+Backspace"},
}

// actionOrder 决定 List / StateDTO 的稳定顺序。
var actionOrder = []string{
	ActionPlayPause,
	ActionStop,
	ActionPreviewToggle,
	ActionEmergencyRelease,
}

// isKnownAction 校验动作 ID。
func isKnownAction(id string) bool {
	for _, a := range actionOrder {
		if a == id {
			return true
		}
	}
	return false
}

// defaultAccelerator 返回某动作的默认组合（找不到返回空串）。
func defaultAccelerator(actionID string) string {
	for _, b := range defaultBindings {
		if b.ActionID == actionID {
			return b.Accelerator
		}
	}
	return ""
}

package storage

// HotkeyBindingsTable 是全局快捷键绑定集合名。
const HotkeyBindingsTable = "hotkey_bindings"

// HotkeyBinding 持久化单个动作的全局热键绑定。
//
// 每个动作（playPause / stop / previewToggle / emergencyRelease）一行，
// ActionID 作主键。Accelerator 是规范化后的可读组合（如 "Ctrl+Alt+Backspace"），
// 后端解析为 Win32 修饰位 + 虚拟键码后注册。Enabled=false 表示用户暂时停用该热键。
//
// 注意：Registered 状态（是否成功注册到 OS、是否被占用）是运行时态，不落库，
// 由 hotkey 服务在内存里维护并随 StateDTO 返回。
type HotkeyBinding struct {
	ActionID    string `gorm:"primaryKey;size:64" json:"actionId"`
	Accelerator string `gorm:"size:128" json:"accelerator"`
	Enabled     bool   `json:"enabled"`
	UpdatedAt   int64  `gorm:"autoUpdateTime:false" json:"updatedAt"`
}

func (HotkeyBinding) TableName() string { return HotkeyBindingsTable }

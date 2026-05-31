package hotkey

// resolved 是要注册到 OS 的一条解析后绑定。
type resolved struct {
	actionID    string
	accelerator string // 规范化文本
	modifiers   int    // 不含 ModNoRepeat
	vk          int
}

// registerResult 是单条绑定的注册结果（per-binding）。
type registerResult struct {
	actionID  string
	ok        bool
	errorCode string // ok=false 时填，对应 CodeAlreadyRegistered / CodeRegisterFailed
}

// triggerFunc 是热键触发回调，由 Service 注入；actionID 为被触发的动作。
type triggerFunc func(actionID string)

// manager 抽象不同平台的全局热键注册。
//
//   - Windows: RegisterHotKey + GetMessage 消息循环（register_windows.go）
//   - 其它平台: no-op（register_stub.go）
//
// Apply 是幂等的"全量同步"：传入当前应启用的绑定集合，manager 负责注销旧的、
// 注册新的，并为每条返回注册结果（成功 / 被占用 / 失败）。
type manager interface {
	// Supported 返回当前平台是否支持全局热键。
	Supported() bool
	// Start 启动后台消息循环，trigger 在热键触发时被调用。
	Start(trigger triggerFunc) error
	// Apply 全量同步当前启用的绑定，返回每条的注册结果。
	Apply(bindings []resolved) []registerResult
	// Stop 注销所有热键并停止消息循环。
	Stop()
}

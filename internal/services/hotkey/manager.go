package hotkey

const (
	targetBuiltin  = "builtin"
	targetExternal = "external"
)

// target identifies one hotkey target, either a built-in action or an external source target.
type target struct {
	kind     string
	actionID string
	source   string
	targetID string
}

func builtinTarget(actionID string) target {
	return target{kind: targetBuiltin, actionID: actionID}
}

func externalTarget(source string, targetID string) target {
	return target{kind: targetExternal, source: source, targetID: targetID}
}

func (t target) key() string {
	if t.kind == targetExternal {
		return targetExternal + ":" + t.source + ":" + t.targetID
	}
	return targetBuiltin + ":" + t.actionID
}

// resolved 是要注册到 OS 的一条解析后绑定。
type resolved struct {
	target      target
	accelerator string // 规范化文本
	modifiers   int    // 不含 ModNoRepeat
	vk          int
}

// registerResult 是单条绑定的注册结果（per-binding）。
type registerResult struct {
	target    target
	ok        bool
	errorCode string // ok=false 时填，对应 CodeAlreadyRegistered / CodeRegisterFailed
}

// triggerFunc 是热键触发回调，由 Service 注入；target 为被触发的目标。
type triggerFunc func(target target)

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

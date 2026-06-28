package hotkey

import (
	"context"
	"errors"
	"sync"
	"time"

	"YyslsPlayer/internal/services/player"
	"YyslsPlayer/internal/storage"
	"YyslsPlayer/internal/utils/logx"
)

// EventEmitter 抽象向前端 Emit 事件的能力（仿 player 的模式，便于测试注入）。
type EventEmitter interface {
	Emit(name string, payload any)
}

// EventEmitterFunc 适配普通函数为 EventEmitter。
type EventEmitterFunc func(name string, payload any)

func (f EventEmitterFunc) Emit(name string, payload any) { f(name, payload) }

// ExternalHandler receives an external hotkey target trigger.
type ExternalHandler func(targetID string)

// Service 是 Wails 服务，方法自动暴露给前端。
//
// 职责：
//   - 持久化每个内置动作的全局热键绑定（SQLite，holder.Current()）
//   - 统一持有内置绑定与外部 source 绑定快照，并全量同步到平台 manager
//   - 收到 OS 热键触发时：内置 playback 类动作直接调用 player，外部 target
//     派发给对应 source handler（例如 macro）
type Service struct {
	holder  *storage.Holder
	player  *player.Service
	manager manager

	mu               sync.RWMutex
	emitter          EventEmitter
	regState         map[string]registerResult // actionID -> 最近一次内置注册结果
	externalBindings map[string][]ExternalBinding
	externalState    map[string]map[string]ExternalBindingState
	externalHandlers map[string]ExternalHandler
	started          bool
}

// New 构造服务；manager 由平台 newManager() 提供（windows / stub）。
func New(holder *storage.Holder, playerSvc *player.Service) *Service {
	return &Service{
		holder:           holder,
		player:           playerSvc,
		manager:          newManager(),
		regState:         make(map[string]registerResult),
		externalBindings: make(map[string][]ExternalBinding),
		externalState:    make(map[string]map[string]ExternalBindingState),
		externalHandlers: make(map[string]ExternalHandler),
	}
}

func (s *Service) store() *storage.Store {
	return s.holder.Current().Store
}

// AttachEmitter 注入前端事件发射器（app 创建后回填）。
//
//wails:ignore
func (s *Service) AttachEmitter(emitter EventEmitter) {
	s.mu.Lock()
	s.emitter = emitter
	s.mu.Unlock()
}

// RegisterExternalHandler registers a trigger handler for one external source.
//
//wails:ignore
func (s *Service) RegisterExternalHandler(source string, handler ExternalHandler) {
	if source == "" {
		return
	}
	s.mu.Lock()
	if handler == nil {
		delete(s.externalHandlers, source)
	} else {
		s.externalHandlers[source] = handler
	}
	s.mu.Unlock()
}

// SetExternalBindings replaces one source's full external binding snapshot.
//
//wails:ignore
func (s *Service) SetExternalBindings(source string, bindings []ExternalBinding) []ExternalBindingState {
	if source == "" {
		return nil
	}
	copied := make([]ExternalBinding, 0, len(bindings))
	for _, b := range bindings {
		copied = append(copied, b)
	}
	s.mu.Lock()
	s.externalBindings[source] = copied
	s.mu.Unlock()
	s.reapply(context.Background())
	return s.GetExternalBindingStates(source)
}

// ClearExternalBindings removes all external bindings for one source.
//
//wails:ignore
func (s *Service) ClearExternalBindings(source string) {
	if source == "" {
		return
	}
	s.mu.Lock()
	delete(s.externalBindings, source)
	delete(s.externalState, source)
	s.mu.Unlock()
	s.reapply(context.Background())
}

// GetExternalBindingStates returns the last parse/register snapshot for a source.
//
//wails:ignore
func (s *Service) GetExternalBindingStates(source string) []ExternalBindingState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	byTarget := s.externalState[source]
	if len(byTarget) == 0 {
		return []ExternalBindingState{}
	}
	out := make([]ExternalBindingState, 0, len(byTarget))
	for _, st := range byTarget {
		out = append(out, st)
	}
	return out
}

// Start 在应用启动后调用：播种默认绑定、启动 manager、注册启用项。
//
//wails:ignore
func (s *Service) Start() error {
	if err := s.ensureSeeded(context.Background()); err != nil {
		logx.For("hotkey").Error("seed default bindings failed", "error", err)
	}
	if !s.manager.Supported() {
		logx.For("hotkey").Info("global hotkeys unsupported on this platform")
		return nil
	}
	if err := s.manager.Start(s.onTrigger); err != nil {
		logx.For("hotkey").Error("hotkey manager start failed", "error", err)
		return err
	}
	s.mu.Lock()
	s.started = true
	s.mu.Unlock()
	s.reapply(context.Background())
	return nil
}

// Stop 注销所有热键并停止消息循环（应用退出时调用）。
//
//wails:ignore
func (s *Service) Stop() {
	s.manager.Stop()
}

// ensureSeeded 在表为空时写入默认绑定。
func (s *Service) ensureSeeded(ctx context.Context) error {
	_ = ctx
	if s.store().CountHotkeyBindings() > 0 {
		return nil
	}
	for _, d := range defaultBindings {
		row := storage.HotkeyBinding{ActionID: d.ActionID, Accelerator: d.Accelerator, Enabled: true}
		if err := s.store().SaveHotkeyBinding(row); err != nil {
			return err
		}
	}
	return nil
}

// loadBindings 读取所有绑定，缺失的动作用默认值补齐（不写库，仅内存补齐）。
func (s *Service) loadBindings(ctx context.Context) ([]storage.HotkeyBinding, error) {
	_ = ctx
	rows := s.store().ListHotkeyBindings()
	byAction := make(map[string]storage.HotkeyBinding, len(rows))
	for _, r := range rows {
		byAction[r.ActionID] = r
	}
	out := make([]storage.HotkeyBinding, 0, len(actionOrder))
	for _, actionID := range actionOrder {
		if r, ok := byAction[actionID]; ok {
			out = append(out, r)
			continue
		}
		out = append(out, storage.HotkeyBinding{
			ActionID:    actionID,
			Accelerator: defaultAccelerator(actionID),
			Enabled:     true,
		})
	}
	return out, nil
}

// GetState 返回整体快照（含运行时注册状态），前端一次拉取。
func (s *Service) GetState(ctx context.Context) (StateDTO, error) {
	rows, err := s.loadBindings(ctx)
	if err != nil {
		return StateDTO{}, err
	}
	supported := s.manager.Supported()
	s.mu.RLock()
	regState := make(map[string]registerResult, len(s.regState))
	for k, v := range s.regState {
		regState[k] = v
	}
	s.mu.RUnlock()

	bindings := make([]BindingDTO, 0, len(rows))
	for _, r := range rows {
		dto := BindingDTO{
			ActionID:    r.ActionID,
			Accelerator: r.Accelerator,
			Enabled:     r.Enabled,
		}
		if supported && r.Enabled {
			if res, ok := regState[r.ActionID]; ok {
				dto.Registered = res.ok
				if !res.ok {
					dto.ErrorCode = res.errorCode
				}
			}
		}
		bindings = append(bindings, dto)
	}
	return StateDTO{Supported: supported, Bindings: bindings}, nil
}

// SetBinding 更新某动作的组合；先做安全校验，不安全 / 非法直接拒绝。
func (s *Service) SetBinding(ctx context.Context, actionID, accel string) (StateDTO, error) {
	if !isKnownAction(actionID) {
		return StateDTO{}, ErrUnknownAction
	}
	acc, err := normalizeAccelerator(accel)
	if err != nil {
		return StateDTO{}, err
	}
	if err := s.upsert(ctx, actionID, func(row *storage.HotkeyBinding) {
		row.Accelerator = acc.text
	}); err != nil {
		return StateDTO{}, err
	}
	s.reapply(ctx)
	return s.GetState(ctx)
}

// SetEnabled 启用 / 停用某动作的热键。
func (s *Service) SetEnabled(ctx context.Context, actionID string, enabled bool) (StateDTO, error) {
	if !isKnownAction(actionID) {
		return StateDTO{}, ErrUnknownAction
	}
	if err := s.upsert(ctx, actionID, func(row *storage.HotkeyBinding) {
		row.Enabled = enabled
	}); err != nil {
		return StateDTO{}, err
	}
	s.reapply(ctx)
	return s.GetState(ctx)
}

// Reset 恢复全部默认绑定（清表后重新播种）。
func (s *Service) Reset(ctx context.Context) (StateDTO, error) {
	if err := s.store().ClearHotkeyBindings(); err != nil {
		return StateDTO{}, err
	}
	if err := s.ensureSeeded(ctx); err != nil {
		return StateDTO{}, err
	}
	s.reapply(ctx)
	return s.GetState(ctx)
}

// upsert 读取或新建某动作行，应用 mutate 后保存。
func (s *Service) upsert(ctx context.Context, actionID string, mutate func(*storage.HotkeyBinding)) error {
	_ = ctx
	row, ok := s.store().GetHotkeyBinding(actionID)
	if !ok {
		row = storage.HotkeyBinding{
			ActionID:    actionID,
			Accelerator: defaultAccelerator(actionID),
			Enabled:     true,
		}
	}
	mutate(&row)
	return s.store().SaveHotkeyBinding(row)
}

// reapply 解析启用的绑定并全量同步到 manager，记录 per-binding 注册结果。
func (s *Service) reapply(ctx context.Context) {
	rows, err := s.loadBindings(ctx)
	if err != nil {
		logx.For("hotkey").Error("load bindings failed", "error", err)
		return
	}

	s.mu.RLock()
	started := s.started
	externalBindings := copyExternalBindings(s.externalBindings)
	s.mu.RUnlock()

	var toRegister []resolved
	builtinState := make(map[string]registerResult, len(rows))
	externalState := make(map[string]map[string]ExternalBindingState, len(externalBindings))
	seen := make(map[string]target)

	for _, r := range rows {
		if !r.Enabled {
			continue
		}
		tgt := builtinTarget(r.ActionID)
		acc, err := normalizeAccelerator(r.Accelerator)
		if err != nil {
			builtinState[r.ActionID] = registerResult{target: tgt, ok: false, errorCode: acceleratorErrorCode(err)}
			logx.For("hotkey").Warn("skip invalid accelerator", "actionId", r.ActionID, "accel", r.Accelerator, "error", err)
			continue
		}
		identity := acceleratorIdentity(acc)
		if _, exists := seen[identity]; exists {
			builtinState[r.ActionID] = registerResult{target: tgt, ok: false, errorCode: CodeAppConflict}
			continue
		}
		seen[identity] = tgt
		toRegister = append(toRegister, resolved{target: tgt, accelerator: acc.text, modifiers: acc.modifiers, vk: acc.vk})
	}

	for source, bindings := range externalBindings {
		if _, ok := externalState[source]; !ok {
			externalState[source] = make(map[string]ExternalBindingState, len(bindings))
		}
		for _, b := range bindings {
			st := ExternalBindingState{Source: source, TargetID: b.TargetID, Accelerator: b.Accelerator, Enabled: b.Enabled}
			if !b.Enabled {
				externalState[source][b.TargetID] = st
				continue
			}
			tgt := externalTarget(source, b.TargetID)
			acc, err := normalizeAccelerator(b.Accelerator)
			if err != nil {
				st.ErrorCode = acceleratorErrorCode(err)
				externalState[source][b.TargetID] = st
				continue
			}
			st.Accelerator = acc.text
			identity := acceleratorIdentity(acc)
			if _, exists := seen[identity]; exists {
				st.ErrorCode = CodeAppConflict
				externalState[source][b.TargetID] = st
				continue
			}
			seen[identity] = tgt
			externalState[source][b.TargetID] = st
			toRegister = append(toRegister, resolved{target: tgt, accelerator: acc.text, modifiers: acc.modifiers, vk: acc.vk})
		}
	}

	if s.manager.Supported() && started {
		results := s.manager.Apply(toRegister)
		for _, res := range results {
			switch res.target.kind {
			case targetBuiltin:
				builtinState[res.target.actionID] = res
			case targetExternal:
				bySource := externalState[res.target.source]
				if bySource == nil {
					bySource = map[string]ExternalBindingState{}
					externalState[res.target.source] = bySource
				}
				st := bySource[res.target.targetID]
				st.Source = res.target.source
				st.TargetID = res.target.targetID
				st.Enabled = true
				st.Registered = res.ok
				if !res.ok {
					st.ErrorCode = res.errorCode
				}
				bySource[res.target.targetID] = st
			}
		}
	}

	s.mu.Lock()
	s.regState = builtinState
	s.externalState = externalState
	s.mu.Unlock()

	for _, res := range builtinState {
		if !res.ok && res.errorCode != "" {
			logx.For("hotkey").Warn("hotkey register failed", "target", res.target.key(), "code", res.errorCode)
		}
	}
	for _, bySource := range externalState {
		for _, st := range bySource {
			if st.Enabled && !st.Registered && st.ErrorCode != "" {
				logx.For("hotkey").Warn("external hotkey register failed", "source", st.Source, "targetId", st.TargetID, "code", st.ErrorCode)
			}
		}
	}
}

func copyExternalBindings(in map[string][]ExternalBinding) map[string][]ExternalBinding {
	out := make(map[string][]ExternalBinding, len(in))
	for source, bindings := range in {
		copied := make([]ExternalBinding, len(bindings))
		copy(copied, bindings)
		out[source] = copied
	}
	return out
}

func acceleratorErrorCode(err error) string {
	switch {
	case errors.Is(err, ErrUnsafeAccelerator):
		return CodeUnsafeAccelerator
	case errors.Is(err, ErrInvalidAccelerator):
		return CodeInvalidAccelerator
	default:
		return CodeRegisterFailed
	}
}

// onTrigger 由 manager 在 OS 热键触发时回调（独立 goroutine）。
func (s *Service) onTrigger(tgt target) {
	if tgt.kind == targetExternal {
		s.handleExternal(tgt)
		return
	}
	actionID := tgt.actionID
	handled := s.handleBackend(actionID)
	accel := ""
	if rows, err := s.loadBindings(context.Background()); err == nil {
		for _, r := range rows {
			if r.ActionID == actionID {
				accel = r.Accelerator
				break
			}
		}
	}
	s.emit(TriggeredDTO{
		ActionID:         actionID,
		Accelerator:      accel,
		HandledByBackend: handled,
		At:               time.Now().UnixMilli(),
	})
}

func (s *Service) handleExternal(tgt target) {
	s.mu.RLock()
	handler := s.externalHandlers[tgt.source]
	s.mu.RUnlock()
	if handler == nil {
		logx.For("hotkey").Warn("external hotkey handler missing", "source", tgt.source, "targetId", tgt.targetID)
		return
	}
	handler(tgt.targetID)
}

// handleBackend 对可由后端直接执行的动作调用 player；返回是否已处理。
func (s *Service) handleBackend(actionID string) bool {
	ctx := context.Background()
	switch actionID {
	case ActionEmergencyRelease:
		if err := s.player.ReleaseAll(ctx); err != nil {
			logx.For("hotkey").Error("emergency release failed", "error", err)
		}
		return true
	case ActionStop:
		// 对当前 session 停止；无会话时安全忽略。
		if _, err := s.player.Stop(ctx, ""); err != nil && !errors.Is(err, player.ErrPlayerNotFound) {
			logx.For("hotkey").Warn("hotkey stop failed", "error", err)
		}
		return true
	case ActionPlayPause:
		// 仅处理"暂停 <-> 继续"（开始演奏需要前端的 PlayPlan，交给前端）。
		state, err := s.player.GetState(ctx, "")
		if err != nil {
			return false
		}
		switch state.State {
		case player.StatePlaying:
			if _, err := s.player.Pause(ctx, ""); err != nil {
				logx.For("hotkey").Warn("hotkey pause failed", "error", err)
			}
			return true
		case player.StatePaused:
			if _, err := s.player.Resume(ctx, ""); err != nil {
				logx.For("hotkey").Warn("hotkey resume failed", "error", err)
			}
			return true
		default:
			return false // idle/ready/completed/stopped -> 让前端发起 start
		}
	default:
		return false // preview-toggle 等纯前端动作
	}
}

func (s *Service) emit(dto TriggeredDTO) {
	s.mu.RLock()
	emitter := s.emitter
	s.mu.RUnlock()
	if emitter == nil {
		return
	}
	emitter.Emit(EventTriggered, dto)
}

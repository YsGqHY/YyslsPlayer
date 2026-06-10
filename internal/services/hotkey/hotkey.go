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

// Service 是 Wails 服务，方法自动暴露给前端。
//
// 职责：
//   - 持久化每个动作的全局热键绑定（SQLite，holder.Current()）
//   - 把启用的绑定解析并 Apply 到平台 manager（OS 注册）
//   - 收到 OS 热键触发时：playback 类动作直接调用 player（游戏聚焦也可靠生效），
//     并向前端 Emit hotkey:triggered 供导航 / UI 反馈
type Service struct {
	holder  *storage.Holder
	player  *player.Service
	manager manager

	mu       sync.RWMutex
	emitter  EventEmitter
	regState map[string]registerResult // actionID -> 最近一次注册结果
	started  bool
}

// New 构造服务；manager 由平台 newManager() 提供（windows / stub）。
func New(holder *storage.Holder, playerSvc *player.Service) *Service {
	return &Service{
		holder:   holder,
		player:   playerSvc,
		manager:  newManager(),
		regState: make(map[string]registerResult),
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
	s.mu.RLock()
	started := s.started
	s.mu.RUnlock()
	if !s.manager.Supported() || !started {
		return
	}
	rows, err := s.loadBindings(ctx)
	if err != nil {
		logx.For("hotkey").Error("load bindings failed", "error", err)
		return
	}
	var toRegister []resolved
	for _, r := range rows {
		if !r.Enabled {
			continue
		}
		acc, err := parseAccelerator(r.Accelerator)
		if err != nil {
			logx.For("hotkey").Warn("skip invalid accelerator", "actionId", r.ActionID, "accel", r.Accelerator, "error", err)
			continue
		}
		toRegister = append(toRegister, resolved{
			actionID:    r.ActionID,
			accelerator: acc.text,
			modifiers:   acc.modifiers,
			vk:          acc.vk,
		})
	}
	results := s.manager.Apply(toRegister)
	s.mu.Lock()
	s.regState = make(map[string]registerResult, len(results))
	for _, res := range results {
		s.regState[res.actionID] = res
	}
	s.mu.Unlock()
	for _, res := range results {
		if !res.ok {
			logx.For("hotkey").Warn("hotkey register failed", "actionId", res.actionID, "code", res.errorCode)
		}
	}
}

// onTrigger 由 manager 在 OS 热键触发时回调（独立 goroutine）。
//
// playback 类动作直接对 player 当前 session 执行，保证游戏聚焦时也生效；
// 全部动作都 Emit hotkey:triggered 给前端做导航 / UI 反馈。
func (s *Service) onTrigger(actionID string) {
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

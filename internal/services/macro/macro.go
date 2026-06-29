//go:build completion

package macro

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"YyslsPlayer/internal/services/hotkey"
	"YyslsPlayer/internal/services/keysim"
	"YyslsPlayer/internal/services/player"
	"YyslsPlayer/internal/storage"
	"YyslsPlayer/internal/utils/filex"
	"YyslsPlayer/internal/utils/logx"
	"gopkg.in/yaml.v3"
	"gorm.io/gorm"
)

type HotkeyBinder interface {
	RegisterExternalHandler(source string, handler hotkey.ExternalHandler)
	SetExternalBindings(source string, bindings []hotkey.ExternalBinding) []hotkey.ExternalBindingState
	ClearExternalBindings(source string)
	GetExternalBindingStates(source string) []hotkey.ExternalBindingState
}

type PlaybackStateProvider interface {
	GetState(ctx context.Context, sessionID string) (player.PlayerStateDTO, error)
}

type Service struct {
	holder *storage.Holder
	keysim *keysim.Service
	hotkey HotkeyBinder
	player PlaybackStateProvider

	mu          sync.Mutex
	emitter     EventEmitter
	current     *runSession
	state       MacroStateDTO
	recording   *recordSession
	recordState RecordStateDTO
}

type runSession struct {
	macroID    uint
	macroName  string
	steps      []storage.MacroStep
	planned    plannedMacro
	cancel     context.CancelFunc
	done       chan struct{}
	startedAt  int64
	repeatMode string
	repeatN    int
	intervalMs int64
	triggerVK  int
}

func New(holder *storage.Holder, sim *keysim.Service, binder HotkeyBinder, playerState PlaybackStateProvider) *Service {
	if sim == nil {
		sim = keysim.New(nil)
	}
	return &Service{
		holder: holder,
		keysim: sim,
		hotkey: binder,
		player: playerState,
		state:  MacroStateDTO{State: StateIdle},
	}
}

func (s *Service) store() *storage.Store { return s.holder.Current().Store }

// AttachEmitter 注入前端事件发射器。
//
//wails:ignore
func (s *Service) AttachEmitter(fn func(name string, payload any)) {
	s.mu.Lock()
	s.emitter = EventEmitterFunc(fn)
	s.mu.Unlock()
}

// Start loads enabled macro triggers into the hotkey service.
//
//wails:ignore
func (s *Service) Start() {
	if s.hotkey == nil {
		return
	}
	s.hotkey.RegisterExternalHandler(SourceHotkey, s.onHotkey)
	s.syncHotkeys()
}

// Stop stops any running macro and clears external hotkeys.
//
//wails:ignore
func (s *Service) Stop() {
	s.stopRecordingInternal()
	_, _ = s.stopRunning(context.Background(), StateStopped, "stopped")
	if s.hotkey != nil {
		s.hotkey.ClearExternalBindings(SourceHotkey)
	}
}

func (s *Service) ListMacros(ctx context.Context) ([]MacroSummaryDTO, error) {
	_ = ctx
	profiles := s.store().ListMacroProfiles()
	states := s.hotkeyStates()
	out := make([]MacroSummaryDTO, 0, len(profiles))
	for _, p := range profiles {
		out = append(out, s.summaryDTO(p, len(s.store().ListMacroSteps(p.ID)), states))
	}
	return out, nil
}

func (s *Service) GetMacro(ctx context.Context, id uint) (MacroDetailDTO, error) {
	_ = ctx
	detail, ok := s.store().GetMacroDetail(id)
	if !ok {
		return MacroDetailDTO{}, ErrMacroNotFound
	}
	return s.detailDTO(detail), nil
}

func (s *Service) CreateMacro(ctx context.Context, name string) (MacroDetailDTO, error) {
	return s.SaveMacro(ctx, SaveMacroRequest{Name: name, Enabled: false, RepeatMode: RepeatModeOnce, RepeatCount: 1, InterruptPolicy: InterruptIgnore})
}

func (s *Service) SaveMacro(ctx context.Context, req SaveMacroRequest) (MacroDetailDTO, error) {
	_ = ctx
	profile := storage.MacroProfile{
		ID:                 req.ID,
		Name:               req.Name,
		Description:        req.Description,
		TriggerAccelerator: strings.TrimSpace(req.TriggerAccelerator),
		AllowUnsafeTrigger: req.AllowUnsafeTrigger,
		Enabled:            req.Enabled,
		RepeatMode:         req.RepeatMode,
		RepeatCount:        req.RepeatCount,
		RepeatIntervalMs:   req.RepeatIntervalMs,
		InterruptPolicy:    req.InterruptPolicy,
	}
	normalizeProfile(&profile)
	if profile.TriggerAccelerator != "" {
		acc, err := normalizeTrigger(profile.TriggerAccelerator, profile.AllowUnsafeTrigger)
		if err != nil {
			return MacroDetailDTO{}, fmt.Errorf("%w: %w", ErrMacroTriggerInvalid, err)
		}
		profile.TriggerAccelerator = acc
	}
	steps, err := normalizeAndValidateSteps(req.Steps)
	if err != nil {
		return MacroDetailDTO{}, err
	}
	detail, err := s.store().SaveMacroDetail(storage.MacroDetail{Profile: profile, Steps: steps})
	if err != nil {
		return MacroDetailDTO{}, err
	}
	s.syncHotkeys()
	return s.detailDTO(detail), nil
}

func (s *Service) DeleteMacro(ctx context.Context, id uint) error {
	_ = ctx
	if err := s.store().DeleteMacro(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrMacroNotFound
		}
		return err
	}
	s.syncHotkeys()
	return nil
}

// ExportMacro writes a single macro (with all steps) to a YAML file at
// targetPath. The macro is rendered in a human-readable, hand-editable form.
func (s *Service) ExportMacro(ctx context.Context, id uint, targetPath string) error {
	_ = ctx
	detail, ok := s.store().GetMacroDetail(id)
	if !ok {
		return ErrMacroNotFound
	}
	doc, err := toPortableDoc([]storage.MacroDetail{detail})
	if err != nil {
		return err
	}
	data, err := yaml.Marshal(doc)
	if err != nil {
		return fmt.Errorf("marshal macro yaml: %w", err)
	}
	return filex.WriteAtomic(targetPath, data, 0644)
}

// ImportMacros reads a YAML file and creates one new macro per entry. Imported
// macros get fresh IDs and are disabled (so no hotkey is registered on import).
// Returns the summaries of the created macros.
func (s *Service) ImportMacros(ctx context.Context, sourcePath string) ([]MacroSummaryDTO, error) {
	data, err := filex.ReadLimit(sourcePath, maxImportFileSize)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrMacroImportInvalid, err)
	}
	var doc portableDoc
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrMacroImportInvalid, err)
	}
	reqs, err := fromPortableDoc(doc)
	if err != nil {
		return nil, err
	}
	out := make([]MacroSummaryDTO, 0, len(reqs))
	for i, req := range reqs {
		detail, err := s.SaveMacro(ctx, req)
		if err != nil {
			// A bad trigger should not abort the whole file; retry without it.
			if errors.Is(err, ErrMacroTriggerInvalid) {
				req.TriggerAccelerator = ""
				detail, err = s.SaveMacro(ctx, req)
			}
			if err != nil {
				return nil, fmt.Errorf("%w: macro %d: %w", ErrMacroImportInvalid, i+1, err)
			}
		}
		out = append(out, detail.Profile)
	}
	return out, nil
}

func (s *Service) SetEnabled(ctx context.Context, id uint, enabled bool) (MacroDetailDTO, error) {
	_ = ctx
	row, err := s.store().UpdateMacroProfile(id, func(p *storage.MacroProfile) { p.Enabled = enabled })
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return MacroDetailDTO{}, ErrMacroNotFound
		}
		return MacroDetailDTO{}, err
	}
	s.syncHotkeys()
	steps := s.store().ListMacroSteps(row.ID)
	return s.detailDTO(storage.MacroDetail{Profile: row, Steps: steps}), nil
}

func (s *Service) SetTrigger(ctx context.Context, id uint, accelerator string, allowUnsafe bool) (MacroDetailDTO, error) {
	_ = ctx
	accel := strings.TrimSpace(accelerator)
	if accel != "" {
		var err error
		accel, err = normalizeTrigger(accel, allowUnsafe)
		if err != nil {
			return MacroDetailDTO{}, fmt.Errorf("%w: %w", ErrMacroTriggerInvalid, err)
		}
	}
	row, err := s.store().UpdateMacroProfile(id, func(p *storage.MacroProfile) {
		p.TriggerAccelerator = accel
		p.AllowUnsafeTrigger = allowUnsafe
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return MacroDetailDTO{}, ErrMacroNotFound
		}
		return MacroDetailDTO{}, err
	}
	s.syncHotkeys()
	steps := s.store().ListMacroSteps(row.ID)
	return s.detailDTO(storage.MacroDetail{Profile: row, Steps: steps}), nil
}

func (s *Service) ListAssignableKeys(ctx context.Context) ([]AssignableKeyDTO, error) {
	_ = ctx
	return assignableKeys(), nil
}

func (s *Service) GetState(ctx context.Context) (MacroStateDTO, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state, nil
}

func (s *Service) ValidateMacro(ctx context.Context, req SaveMacroRequest) error {
	_ = ctx
	profile := storage.MacroProfile{Name: req.Name, RepeatMode: req.RepeatMode, RepeatCount: req.RepeatCount, RepeatIntervalMs: req.RepeatIntervalMs, InterruptPolicy: req.InterruptPolicy}
	normalizeProfile(&profile)
	_, err := normalizeAndValidateSteps(req.Steps)
	return err
}

func (s *Service) syncHotkeys() {
	if s.hotkey == nil {
		return
	}
	profiles := s.store().ListEnabledMacroProfiles()
	bindings := make([]hotkey.ExternalBinding, 0, len(profiles))
	for _, p := range profiles {
		if p.TriggerAccelerator == "" {
			continue
		}
		bindings = append(bindings, hotkey.ExternalBinding{
			TargetID:    macroTargetID(p.ID),
			Accelerator: p.TriggerAccelerator,
			Enabled:     p.Enabled,
			Label:       p.Name,
			AllowUnsafe: p.AllowUnsafeTrigger,
		})
	}
	s.hotkey.SetExternalBindings(SourceHotkey, bindings)
}

// normalizeTrigger 解析并规范化触发组合键。allowUnsafe 为 true 时放行裸普通键
// （单键，无 Ctrl/Alt/Win 且非功能键）。
func normalizeTrigger(raw string, allowUnsafe bool) (string, error) {
	if allowUnsafe {
		return hotkey.NormalizeAcceleratorAllowUnsafe(raw)
	}
	return hotkey.NormalizeAccelerator(raw)
}

func (s *Service) hotkeyStates() map[uint]hotkey.ExternalBindingState {
	out := map[uint]hotkey.ExternalBindingState{}
	if s.hotkey == nil {
		return out
	}
	for _, st := range s.hotkey.GetExternalBindingStates(SourceHotkey) {
		if id, ok := parseMacroTargetID(st.TargetID); ok {
			out[id] = st
		}
	}
	return out
}

func (s *Service) summaryDTO(row storage.MacroProfile, stepCount int, states map[uint]hotkey.ExternalBindingState) MacroSummaryDTO {
	st := states[row.ID]
	return MacroSummaryDTO{
		ID:                 row.ID,
		Name:               row.Name,
		Description:        row.Description,
		TriggerAccelerator: row.TriggerAccelerator,
		AllowUnsafeTrigger: row.AllowUnsafeTrigger,
		Enabled:            row.Enabled,
		RepeatMode:         row.RepeatMode,
		RepeatCount:        row.RepeatCount,
		RepeatIntervalMs:   row.RepeatIntervalMs,
		InterruptPolicy:    row.InterruptPolicy,
		StepCount:          stepCount,
		Registered:         st.Registered,
		ErrorCode:          st.ErrorCode,
		CreatedAt:          row.CreatedAt,
		UpdatedAt:          row.UpdatedAt,
	}
}

func (s *Service) detailDTO(detail storage.MacroDetail) MacroDetailDTO {
	states := s.hotkeyStates()
	steps := make([]MacroStepDTO, 0, len(detail.Steps))
	for _, row := range detail.Steps {
		steps = append(steps, stepDTO(row))
	}
	return MacroDetailDTO{Profile: s.summaryDTO(detail.Profile, len(steps), states), Steps: steps}
}

func (s *Service) onHotkey(targetID string) {
	id, ok := parseMacroTargetID(targetID)
	if !ok {
		logx.For("macro").Warn("invalid macro target", "targetId", targetID)
		return
	}
	// Hotkey-triggered runs have no caller to surface the rejection, so any
	// failure (player busy, recording active, no steps, etc.) is bubbled to the
	// frontend via macro:error. Without this a global trigger that is silently
	// blocked (e.g. a MIDI track is loaded) looks like "the hotkey does nothing".
	if _, err := s.RunMacro(context.Background(), id); err != nil {
		s.emitError(id, macroErrorCode(err), err.Error())
		logx.For("macro").Warn("macro hotkey run failed", "macroId", id, "error", err)
	}
}

func macroTargetID(id uint) string { return fmt.Sprintf("macro:%d", id) }

func parseMacroTargetID(targetID string) (uint, bool) {
	if !strings.HasPrefix(targetID, "macro:") {
		return 0, false
	}
	var id uint
	if _, err := fmt.Sscanf(targetID, "macro:%d", &id); err != nil || id == 0 {
		return 0, false
	}
	return id, true
}

func nowMillis() int64 { return time.Now().UnixMilli() }

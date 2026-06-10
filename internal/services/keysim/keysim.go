package keysim

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"YyslsPlayer/internal/utils/logx"
)

type Service struct {
	driver  Driver
	mu      sync.Mutex
	pressed map[string]*pressedEntry
	order   int
}

type pressedEntry struct {
	key      Key
	count    int
	order    int
	modifier bool
}

type eventCollector struct {
	limit              int
	keyframes          []KeyAction
	totalKeyframes     int
	keyframesTruncated bool
	events             []KeyEvent
	total              int
	truncated          bool
}

func New(driver Driver) *Service {
	if driver == nil {
		driver = NewDefaultDriver()
	}
	return &Service{
		driver:  driver,
		pressed: make(map[string]*pressedEntry),
	}
}

func (s *Service) Apply(ctx context.Context, action KeyAction, opts RunOptions) (RunResult, error) {
	opts = normalizeOptions(opts)
	collector := newEventCollector(opts.DryRunLogLimit)
	collector.addKeyframe(opts, action)
	if err := s.apply(ctx, action, opts, collector); err != nil {
		recovered, releaseErr := s.releaseAll(ctx, opts, collector)
		combined := errors.Join(err, releaseErr)
		logRecovery("keysim apply failed", opts, recovered, combined)
		return collector.result(opts, 0, recovered), combined
	}
	return collector.result(opts, 0, 0), nil
}

func (s *Service) Run(ctx context.Context, actions []KeyAction, opts RunOptions) (RunResult, error) {
	opts = normalizeOptions(opts)
	collector := newEventCollector(opts.DryRunLogLimit)
	if err := s.refreshChainHead(ctx, opts); err != nil {
		recovered, releaseErr := s.releaseAll(ctx, opts, collector)
		combined := errors.Join(err, releaseErr)
		logRecovery("keysim refresh hook chain failed", opts, recovered, combined)
		return collector.result(opts, 0, recovered), combined
	}
	for _, action := range actions {
		collector.addKeyframe(opts, action)
		if err := s.apply(ctx, action, opts, collector); err != nil {
			recovered, releaseErr := s.releaseAll(ctx, opts, collector)
			combined := errors.Join(err, releaseErr)
			logRecovery("keysim run failed", opts, recovered, combined)
			return collector.result(opts, 0, recovered), combined
		}
	}
	result := collector.result(opts, 0, 0)
	logx.For("keysim").Info("keysim run finished", "dryRun", opts.DryRun, "actions", len(actions), "keyframes", result.TotalKeyframes, "events", result.TotalEvents, "keyframesTruncated", result.KeyframesTruncated, "eventsTruncated", result.Truncated)
	return result, nil
}

func (s *Service) ReleaseAll(ctx context.Context, opts RunOptions) (RunResult, error) {
	opts = normalizeOptions(opts)
	collector := newEventCollector(opts.DryRunLogLimit)
	released, err := s.releaseAll(ctx, opts, collector)
	return collector.result(opts, released, 0), err
}

func (s *Service) Snapshot() StateSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	return StateSnapshot{Pressed: s.pressedSnapshotLocked()}
}

func (s *Service) RefreshChainHead(ctx context.Context, opts RunOptions) error {
	opts = normalizeOptions(opts)
	return s.refreshChainHead(ctx, opts)
}

func (s *Service) refreshChainHead(ctx context.Context, opts RunOptions) error {
	if opts.DryRun {
		return nil
	}
	refresher, ok := s.driver.(ChainHeadRefresher)
	if !ok {
		return nil
	}
	return refresher.RefreshChainHead(ctx)
}

func (s *Service) apply(ctx context.Context, action KeyAction, opts RunOptions, collector *eventCollector) error {
	if err := validateAction(action); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	switch action.Action {
	case ActionPress:
		if len(action.Modifiers) == 0 {
			_, err := s.downLocked(ctx, eventFromAction(action, action.Key, PhysicalDown, false), opts, collector)
			return err
		}
		chordOpts := tightChordOptions(opts)
		for _, modifier := range action.Modifiers {
			if _, err := s.downLocked(ctx, eventFromAction(action, modifier, PhysicalDown, true), chordOpts, collector); err != nil {
				return err
			}
		}
		if _, err := s.downLocked(ctx, eventFromAction(action, action.Key, PhysicalDown, false), chordOpts, collector); err != nil {
			return err
		}
		if err := s.upLocked(ctx, eventFromAction(action, action.Key, PhysicalUp, false), chordOpts, collector); err != nil {
			return err
		}
		for i := len(action.Modifiers) - 1; i >= 0; i-- {
			if err := s.upLocked(ctx, eventFromAction(action, action.Modifiers[i], PhysicalUp, true), chordOpts, collector); err != nil {
				return err
			}
		}
		return nil
	case ActionRelease:
		if len(action.Modifiers) == 0 {
			return s.upLocked(ctx, eventFromAction(action, action.Key, PhysicalUp, false), opts, collector)
		}
		chordOpts := tightChordOptions(opts)
		if err := s.upLocked(ctx, eventFromAction(action, action.Key, PhysicalUp, false), chordOpts, collector); err != nil {
			return err
		}
		for i := len(action.Modifiers) - 1; i >= 0; i-- {
			if err := s.upLocked(ctx, eventFromAction(action, action.Modifiers[i], PhysicalUp, true), chordOpts, collector); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("%w: %s", ErrInvalidAction, action.Action)
	}
}

func (s *Service) downLocked(ctx context.Context, event KeyEvent, opts RunOptions, collector *eventCollector) (bool, error) {
	id := keyID(event.Key)
	entry := s.pressed[id]
	if entry != nil {
		entry.count++
		return false, nil
	}
	if err := s.driver.Send(ctx, event, opts); err != nil {
		logSendFailure(event, err)
		return false, err
	}
	if !opts.DryRun && opts.InterKeyDelayMs > 0 {
		time.Sleep(time.Duration(opts.InterKeyDelayMs) * time.Millisecond)
	}
	s.order++
	s.pressed[id] = &pressedEntry{key: event.Key, count: 1, order: s.order, modifier: event.Modifier}
	collector.add(event)
	logDryRunEvent(opts, event)
	return true, nil
}

func (s *Service) upLocked(ctx context.Context, event KeyEvent, opts RunOptions, collector *eventCollector) error {
	id := keyID(event.Key)
	entry := s.pressed[id]
	if entry == nil {
		return nil
	}
	entry.count--
	if entry.count > 0 {
		return nil
	}
	if err := s.driver.Send(ctx, event, opts); err != nil {
		entry.count++
		logSendFailure(event, err)
		return err
	}
	if !opts.DryRun && opts.InterKeyDelayMs > 0 {
		time.Sleep(time.Duration(opts.InterKeyDelayMs) * time.Millisecond)
	}
	delete(s.pressed, id)
	collector.add(event)
	logDryRunEvent(opts, event)
	return nil
}

func (s *Service) releaseAll(ctx context.Context, opts RunOptions, collector *eventCollector) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entries := make([]*pressedEntry, 0, len(s.pressed))
	for _, entry := range s.pressed {
		entries = append(entries, entry)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].order > entries[j].order })

	released := 0
	var releaseErr error
	for _, entry := range entries {
		event := KeyEvent{Kind: PhysicalUp, Key: entry.key, Modifier: entry.modifier}
		if err := s.driver.Send(ctx, event, opts); err != nil {
			logSendFailure(event, err)
			releaseErr = errors.Join(releaseErr, fmt.Errorf("%w: %s: %w", ErrReleaseFailed, entry.key.Label, err))
			continue
		}
		if !opts.DryRun && opts.InterKeyDelayMs > 0 {
			time.Sleep(time.Duration(opts.InterKeyDelayMs) * time.Millisecond)
		}
		delete(s.pressed, keyID(entry.key))
		collector.add(event)
		logDryRunEvent(opts, event)
		released++
	}
	return released, releaseErr
}

func (s *Service) pressedSnapshotLocked() []PressedKey {
	entries := make([]*pressedEntry, 0, len(s.pressed))
	for _, entry := range s.pressed {
		entries = append(entries, entry)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].order < entries[j].order })
	out := make([]PressedKey, 0, len(entries))
	for _, entry := range entries {
		out = append(out, PressedKey{Key: entry.key, Count: entry.count, Modifier: entry.modifier})
	}
	return out
}

func newEventCollector(limit int) *eventCollector {
	if limit <= 0 {
		limit = DefaultDryRunLogLimit
	}
	capacity := limit
	if capacity > 64 {
		capacity = 64
	}
	return &eventCollector{
		limit:     limit,
		keyframes: make([]KeyAction, 0, capacity),
		events:    make([]KeyEvent, 0, capacity),
	}
}

func (c *eventCollector) addKeyframe(opts RunOptions, action KeyAction) {
	c.totalKeyframes++
	if len(c.keyframes) < c.limit {
		c.keyframes = append(c.keyframes, action)
	} else {
		c.keyframesTruncated = true
	}
	logDryRunKeyframe(opts, action)
}

func (c *eventCollector) add(event KeyEvent) {
	c.total++
	if len(c.events) < c.limit {
		c.events = append(c.events, event)
		return
	}
	c.truncated = true
}

func (c *eventCollector) result(opts RunOptions, released int, recovered int) RunResult {
	return RunResult{
		DryRun:               opts.DryRun,
		Keyframes:            append([]KeyAction(nil), c.keyframes...),
		TotalKeyframes:       c.totalKeyframes,
		KeyframesTruncated:   c.keyframesTruncated,
		Events:               append([]KeyEvent(nil), c.events...),
		TotalEvents:          c.total,
		Truncated:            c.truncated,
		ReleasedKeys:         released,
		RecoveryReleasedKeys: recovered,
	}
}

func eventFromAction(action KeyAction, key Key, kind PhysicalKind, modifier bool) KeyEvent {
	return KeyEvent{
		TimeMs:         action.TimeMs,
		Kind:           kind,
		Key:            key,
		Lane:           action.Lane,
		SourceNote:     action.SourceNote,
		NormalizedNote: action.NormalizedNote,
		Velocity:       action.Velocity,
		Modifier:       modifier,
	}
}

func validateAction(action KeyAction) error {
	if action.Action != ActionPress && action.Action != ActionRelease {
		return fmt.Errorf("%w: %s", ErrInvalidAction, action.Action)
	}
	if err := validateKey(action.Key); err != nil {
		return err
	}
	for _, modifier := range action.Modifiers {
		if err := validateKey(modifier); err != nil {
			return err
		}
	}
	return nil
}

func validateKey(key Key) error {
	if key.ScanCode == 0 && key.VirtualKey == 0 {
		return fmt.Errorf("%w: %s", ErrInvalidKey, key.Label)
	}
	return nil
}

func normalizeOptions(opts RunOptions) RunOptions {
	if opts.DryRunLogLimit <= 0 {
		opts.DryRunLogLimit = DefaultDryRunLogLimit
	}
	if opts.InterKeyDelayMs <= 0 {
		opts.InterKeyDelayMs = 1
	}
	if opts.ModifierHoldDelayMs <= 0 {
		opts.ModifierHoldDelayMs = DefaultModifierHoldDelayMs
	}
	return opts
}

func tightChordOptions(opts RunOptions) RunOptions {
	chord := opts
	chord.InterKeyDelayMs = 0
	chord.ModifierHoldDelayMs = 0
	return chord
}

func keyID(key Key) string {
	if key.ScanCode != 0 {
		return fmt.Sprintf("scan:%d", key.ScanCode)
	}
	return fmt.Sprintf("vk:%d", key.VirtualKey)
}

func logDryRunKeyframe(opts RunOptions, action KeyAction) {
	if !opts.DryRun {
		return
	}
	logx.For("keysim").Debug(
		"keysim dry-run keyframe",
		"timeMs", action.TimeMs,
		"action", action.Action,
		"lane", action.Lane,
		"sourceNote", action.SourceNote,
		"normalizedNote", action.NormalizedNote,
		"velocity", action.Velocity,
		"label", action.Key.Label,
		"scanCode", action.Key.ScanCode,
		"virtualKey", action.Key.VirtualKey,
		"modifierCount", len(action.Modifiers),
	)
}

func logDryRunEvent(opts RunOptions, event KeyEvent) {
	if !opts.DryRun {
		return
	}
	logx.For("keysim").Debug(
		"keysim dry-run event",
		"kind", event.Kind,
		"label", event.Key.Label,
		"scanCode", event.Key.ScanCode,
		"virtualKey", event.Key.VirtualKey,
		"lane", event.Lane,
		"sourceNote", event.SourceNote,
		"normalizedNote", event.NormalizedNote,
		"modifier", event.Modifier,
	)
}

func logSendFailure(event KeyEvent, err error) {
	logx.For("keysim").Error(
		"keysim send failed",
		"kind", event.Kind,
		"label", event.Key.Label,
		"scanCode", event.Key.ScanCode,
		"virtualKey", event.Key.VirtualKey,
		"modifier", event.Modifier,
		"lane", event.Lane,
		"sourceNote", event.SourceNote,
		"normalizedNote", event.NormalizedNote,
		"error", err,
	)
}

func logRecovery(message string, opts RunOptions, recovered int, err error) {
	logx.For("keysim").Error(
		message,
		"dryRun", opts.DryRun,
		"recoveryReleasedKeys", recovered,
		"error", err,
	)
}

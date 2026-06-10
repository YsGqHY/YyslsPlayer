package storage

import (
	"errors"
	"fmt"
)

const (
	DefaultMidiProfileID     uint    = 1
	DefaultKeymapProfileID   uint    = 1
	DefaultMidiProfileName           = "Default MIDI Profile"
	DefaultKeymapProfileName         = "Default 36-Key Map"
	DefaultMidiBaseNote      int     = 48
	DefaultMidiTranspose     int     = 0
	DefaultMidiOctaveShift   int     = 0
	DefaultMidiSpeed         float64 = 1.0
	DefaultOutOfRangePolicy          = "drop"
	DefaultMinPressMs        int     = 35
	DefaultReleaseGapMs      int     = 15
)

const (
	modifierKeysNone  = "[]"
	modifierKeysShift = `[{"label":"Shift","virtualKey":16,"scanCode":42}]`
	modifierKeysCtrl  = `[{"label":"Ctrl","virtualKey":17,"scanCode":29}]`
)

type defaultKeySpec struct {
	keyLabel  string
	virtual   int
	scan      int
	modifier  string
	blackKey  bool
	pitchName string
}

// defaultAppSettings 返回 app_settings 的初始单行。
func defaultAppSettings() AppSettings {
	return AppSettings{ID: 1, ThemeChoice: "system", LocaleChoice: "auto"}
}

// ensureDefaults 在 Open 后补齐默认状态：app_settings 单行、全局默认 profile、36 键默认 keymap。
// 幂等：已存在的行保留，不覆盖用户校准。
func (s *Store) ensureDefaults() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureAppSettings(); err != nil {
		return err
	}
	if err := s.ensureDefaultKeymap(); err != nil {
		return err
	}
	return s.ensureDefaultProfile()
}

func (s *Store) ensureAppSettings() error {
	var count int64
	if err := s.db().Model(&AppSettings{}).Where("id = ?", uint(1)).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	row := defaultAppSettings()
	row.UpdatedAt = nowMillis()
	return s.db().Create(&row).Error
}

func (s *Store) ensureDefaultProfile() error {
	var count int64
	if err := s.db().Model(&MidiProfile{}).Where("project_id IS NULL").Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	profile := defaultMidiProfile()
	// 若 ID=1 被占用（项目级 profile 抢占），交给自增分配。
	var idTaken int64
	if err := s.db().Model(&MidiProfile{}).Where("id = ?", DefaultMidiProfileID).Count(&idTaken).Error; err != nil {
		return err
	}
	if idTaken > 0 {
		profile.ID = 0
	}
	now := nowMillis()
	profile.CreatedAt = now
	profile.UpdatedAt = now
	return s.db().Create(&profile).Error
}

func (s *Store) ensureDefaultKeymap() error {
	var existing []Keymap36
	if err := s.db().Where("profile_id = ?", DefaultKeymapProfileID).Find(&existing).Error; err != nil {
		return err
	}
	have := make(map[int]bool, len(existing))
	for _, row := range existing {
		have[row.Lane] = true
	}
	now := nowMillis()
	rows := make([]Keymap36, 0, 36)
	for _, row := range defaultKeymap36Rows() {
		if have[row.Lane] {
			continue
		}
		row.ID = 0
		row.CreatedAt = now
		row.UpdatedAt = now
		rows = append(rows, row)
	}
	if len(rows) > 0 {
		if err := s.db().Create(&rows).Error; err != nil {
			return err
		}
	}
	return s.repairLegacyDefaultSemitoneKeymap()
}

// ensureDefaultMidiState 重新补齐默认 MIDI 状态（幂等），供测试和路径切换后调用。
func ensureDefaultMidiState(store *Store) error {
	if store == nil {
		return errors.New("nil store")
	}
	return store.ensureDefaults()
}

func defaultMidiProfile() MidiProfile {
	return MidiProfile{
		ID:                DefaultMidiProfileID,
		Name:              DefaultMidiProfileName,
		BaseNote:          DefaultMidiBaseNote,
		Transpose:         DefaultMidiTranspose,
		OctaveShift:       DefaultMidiOctaveShift,
		Speed:             DefaultMidiSpeed,
		OutOfRangePolicy:  DefaultOutOfRangePolicy,
		MinPressMs:        DefaultMinPressMs,
		ReleaseGapMs:      DefaultReleaseGapMs,
		KeymapProfileID:   DefaultKeymapProfileID,
		EnabledTracksJSON: "null",
	}
}

func defaultKeymap36Rows() []Keymap36 {
	octaves := [][]defaultKeySpec{
		defaultOctaveSpecs([]defaultKeySpec{
			{keyLabel: "Z", virtual: 90, scan: 44},
			{keyLabel: "X", virtual: 88, scan: 45},
			{keyLabel: "C", virtual: 67, scan: 46},
			{keyLabel: "V", virtual: 86, scan: 47},
			{keyLabel: "B", virtual: 66, scan: 48},
			{keyLabel: "N", virtual: 78, scan: 49},
			{keyLabel: "M", virtual: 77, scan: 50},
		}),
		defaultOctaveSpecs([]defaultKeySpec{
			{keyLabel: "A", virtual: 65, scan: 30},
			{keyLabel: "S", virtual: 83, scan: 31},
			{keyLabel: "D", virtual: 68, scan: 32},
			{keyLabel: "F", virtual: 70, scan: 33},
			{keyLabel: "G", virtual: 71, scan: 34},
			{keyLabel: "H", virtual: 72, scan: 35},
			{keyLabel: "J", virtual: 74, scan: 36},
		}),
		defaultOctaveSpecs([]defaultKeySpec{
			{keyLabel: "Q", virtual: 81, scan: 16},
			{keyLabel: "W", virtual: 87, scan: 17},
			{keyLabel: "E", virtual: 69, scan: 18},
			{keyLabel: "R", virtual: 82, scan: 19},
			{keyLabel: "T", virtual: 84, scan: 20},
			{keyLabel: "Y", virtual: 89, scan: 21},
			{keyLabel: "U", virtual: 85, scan: 22},
		}),
	}

	rows := make([]Keymap36, 0, 36)
	for octaveIdx, specs := range octaves {
		for pitchClass, spec := range specs {
			lane := octaveIdx*12 + pitchClass
			rows = append(rows, Keymap36{
				ProfileID:        DefaultKeymapProfileID,
				ProfileName:      DefaultKeymapProfileName,
				Lane:             lane,
				Label:            fmt.Sprintf("%s%d", spec.pitchName, 3+octaveIdx),
				PitchClass:       pitchClass,
				IsBlackKey:       spec.blackKey,
				VirtualKey:       spec.virtual,
				ScanCode:         spec.scan,
				ModifierKeysJSON: spec.modifier,
				DisplayOrder:     lane,
			})
		}
	}
	return rows
}

func defaultOctaveSpecs(natural []defaultKeySpec) []defaultKeySpec {
	return octaveSpecs(natural, gameSemitoneKeySpec)
}

func legacyDefaultKeymap36Rows() []Keymap36 {
	octaves := [][]defaultKeySpec{
		legacyDefaultOctaveSpecs([]defaultKeySpec{
			{keyLabel: "Z", virtual: 90, scan: 44},
			{keyLabel: "X", virtual: 88, scan: 45},
			{keyLabel: "C", virtual: 67, scan: 46},
			{keyLabel: "V", virtual: 86, scan: 47},
			{keyLabel: "B", virtual: 66, scan: 48},
			{keyLabel: "N", virtual: 78, scan: 49},
			{keyLabel: "M", virtual: 77, scan: 50},
		}),
		legacyDefaultOctaveSpecs([]defaultKeySpec{
			{keyLabel: "A", virtual: 65, scan: 30},
			{keyLabel: "S", virtual: 83, scan: 31},
			{keyLabel: "D", virtual: 68, scan: 32},
			{keyLabel: "F", virtual: 70, scan: 33},
			{keyLabel: "G", virtual: 71, scan: 34},
			{keyLabel: "H", virtual: 72, scan: 35},
			{keyLabel: "J", virtual: 74, scan: 36},
		}),
		legacyDefaultOctaveSpecs([]defaultKeySpec{
			{keyLabel: "Q", virtual: 81, scan: 16},
			{keyLabel: "W", virtual: 87, scan: 17},
			{keyLabel: "E", virtual: 69, scan: 18},
			{keyLabel: "R", virtual: 82, scan: 19},
			{keyLabel: "T", virtual: 84, scan: 20},
			{keyLabel: "Y", virtual: 89, scan: 21},
			{keyLabel: "U", virtual: 85, scan: 22},
		}),
	}

	rows := make([]Keymap36, 0, 36)
	for octaveIdx, specs := range octaves {
		for pitchClass, spec := range specs {
			lane := octaveIdx*12 + pitchClass
			rows = append(rows, Keymap36{
				ProfileID:        DefaultKeymapProfileID,
				ProfileName:      DefaultKeymapProfileName,
				Lane:             lane,
				Label:            fmt.Sprintf("%s%d", spec.pitchName, 3+octaveIdx),
				PitchClass:       pitchClass,
				IsBlackKey:       spec.blackKey,
				VirtualKey:       spec.virtual,
				ScanCode:         spec.scan,
				ModifierKeysJSON: spec.modifier,
				DisplayOrder:     lane,
			})
		}
	}
	return rows
}

func legacyDefaultOctaveSpecs(natural []defaultKeySpec) []defaultKeySpec {
	return octaveSpecs(natural, legacySemitoneKeySpec)
}

func octaveSpecs(natural []defaultKeySpec, semitoneSpec func([]defaultKeySpec) map[int]defaultKeySpec) []defaultKeySpec {
	pitchNames := []string{"C", "C#", "D", "D#", "E", "F", "F#", "G", "G#", "A", "A#", "B"}
	naturalByPitch := map[int]defaultKeySpec{
		0:  natural[0],
		2:  natural[1],
		4:  natural[2],
		5:  natural[3],
		7:  natural[4],
		9:  natural[5],
		11: natural[6],
	}

	out := make([]defaultKeySpec, 12)
	blackBaseByPitch := semitoneSpec(natural)
	for pitchClass := range out {
		if spec, ok := naturalByPitch[pitchClass]; ok {
			spec.modifier = modifierKeysNone
			spec.pitchName = pitchNames[pitchClass]
			out[pitchClass] = spec
			continue
		}
		spec := blackBaseByPitch[pitchClass]
		spec.blackKey = true
		spec.pitchName = pitchNames[pitchClass]
		out[pitchClass] = spec
	}
	return out
}

func gameSemitoneKeySpec(natural []defaultKeySpec) map[int]defaultKeySpec {
	return map[int]defaultKeySpec{
		1:  withModifier(natural[0], modifierKeysShift), // #1
		3:  withModifier(natural[2], modifierKeysCtrl),  // b3
		6:  withModifier(natural[3], modifierKeysShift), // #4
		8:  withModifier(natural[4], modifierKeysShift), // #5
		10: withModifier(natural[6], modifierKeysCtrl),  // b7
	}
}

func legacySemitoneKeySpec(natural []defaultKeySpec) map[int]defaultKeySpec {
	return map[int]defaultKeySpec{
		1:  withModifier(natural[0], modifierKeysShift),
		3:  withModifier(natural[1], modifierKeysShift),
		6:  withModifier(natural[3], modifierKeysShift),
		8:  withModifier(natural[4], modifierKeysShift),
		10: withModifier(natural[5], modifierKeysShift),
	}
}

func withModifier(spec defaultKeySpec, modifier string) defaultKeySpec {
	spec.modifier = modifier
	return spec
}

func (s *Store) repairLegacyDefaultSemitoneKeymap() error {
	var existing []Keymap36
	if err := s.db().Where("profile_id = ?", DefaultKeymapProfileID).Find(&existing).Error; err != nil {
		return err
	}
	if len(existing) == 0 {
		return nil
	}
	legacyByLane := keymapRowsByLane(legacyDefaultKeymap36Rows())
	currentByLane := keymapRowsByLane(defaultKeymap36Rows())
	now := nowMillis()
	for _, row := range existing {
		if row.PitchClass != 3 && row.PitchClass != 10 {
			continue
		}
		legacy, ok := legacyByLane[row.Lane]
		if !ok || !matchesDefaultKeymap(row, legacy) {
			continue
		}
		current := currentByLane[row.Lane]
		updates := map[string]any{
			"virtual_key":        current.VirtualKey,
			"scan_code":          current.ScanCode,
			"modifier_keys_json": current.ModifierKeysJSON,
			"updated_at":         now,
		}
		if err := s.db().Model(&Keymap36{}).
			Where("id = ? AND profile_id = ?", row.ID, DefaultKeymapProfileID).
			Updates(updates).Error; err != nil {
			return err
		}
	}
	return nil
}

func keymapRowsByLane(rows []Keymap36) map[int]Keymap36 {
	out := make(map[int]Keymap36, len(rows))
	for _, row := range rows {
		out[row.Lane] = row
	}
	return out
}

func matchesDefaultKeymap(row Keymap36, expected Keymap36) bool {
	return row.ProfileID == expected.ProfileID &&
		row.ProfileName == expected.ProfileName &&
		row.Lane == expected.Lane &&
		row.Label == expected.Label &&
		row.PitchClass == expected.PitchClass &&
		row.IsBlackKey == expected.IsBlackKey &&
		row.VirtualKey == expected.VirtualKey &&
		row.ScanCode == expected.ScanCode &&
		row.ModifierKeysJSON == expected.ModifierKeysJSON &&
		row.DisplayOrder == expected.DisplayOrder
}

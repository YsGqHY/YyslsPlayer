package storage

import "fmt"

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
)

type defaultKeySpec struct {
	keyLabel  string
	virtual   int
	scan      int
	modifier  string
	blackKey  bool
	pitchName string
}

// ensureDefaultMidiState seeds the built-in global profile and editable 36-key map.
// It is intentionally idempotent: existing rows are preserved so user calibration is not overwritten.
func ensureDefaultMidiState(store *Store) error {
	return store.WithWrite(func(d *storeData) error {
		store.ensureDefaultsLocked()
		return nil
	})
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
	blackBaseByPitch := map[int]defaultKeySpec{
		1:  natural[0],
		3:  natural[1],
		6:  natural[3],
		8:  natural[4],
		10: natural[5],
	}

	out := make([]defaultKeySpec, 12)
	for pitchClass := range out {
		if spec, ok := naturalByPitch[pitchClass]; ok {
			spec.modifier = modifierKeysNone
			spec.pitchName = pitchNames[pitchClass]
			out[pitchClass] = spec
			continue
		}
		spec := blackBaseByPitch[pitchClass]
		spec.modifier = modifierKeysShift
		spec.blackKey = true
		spec.pitchName = pitchNames[pitchClass]
		out[pitchClass] = spec
	}
	return out
}

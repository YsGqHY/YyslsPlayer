import type { Messages } from '@/i18n';

export const midiDefaultsEnUS: Messages = {
  settings: {
    midiDefaults: {
      title: 'MIDI defaults',
      hint: 'Configure the global default MIDI profile. Scores without a saved project profile use these values; project profiles can still override them in the editor.',
      actions: {
        save: 'Save defaults',
        saving: 'Saving…',
        reload: 'Reload',
        reset: 'Restore built-in defaults',
      },
      feedback: {
        loading: 'Loading global defaults…',
        saved: 'Global defaults saved.',
        failed: 'Save failed: {{message}}',
      },
      fields: {
        name: {
          label: 'Profile name',
          description: 'Display name for the global default profile.',
        },
        keymapProfileId: {
          label: 'Keymap profile ID',
          description: '36-key physical mapping profile used to build PlayPlans. The built-in default profile id is 1.',
        },
        baseNote: {
          label: 'Lowest MIDI note',
          description: 'Lowest note for the 36 lanes. Default is 48, C3. Allowed range 0..127.',
        },
        transpose: {
          label: 'Semitone transpose',
          description: 'Move all notes by semitones. Allowed range -24..24.',
        },
        octaveShift: {
          label: 'Octave shift',
          description: 'Move all notes by octaves. Allowed range -3..3.',
        },
        speed: {
          label: 'Playback speed',
          description: 'Affects the shared PlayPlan timeline for preview and performance. Allowed range 0.25..3.0.',
        },
        minPressMs: {
          label: 'Minimum key press ms',
          description: 'Very short notes are stretched to reduce missed input. Allowed range 10..300 ms.',
        },
        releaseGapMs: {
          label: 'Same-key release gap ms',
          description: 'Release gap kept between repeated hits on the same physical key. Allowed range 0..200 ms.',
        },
        outOfRangePolicy: {
          label: 'Out-of-range policy',
          description: 'How notes outside the 36 lanes are handled. Drop is the safest default.',
        },
      },
      policies: {
        drop: 'Drop out-of-range notes',
        octaveFold: 'Fold by octave',
        nearest: 'Map to nearest lane',
      },
    },
  },
};

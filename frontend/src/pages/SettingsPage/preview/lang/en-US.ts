import type { Messages } from '@/i18n';

export const previewSettingsEnUS: Messages = {
  settings: {
    preview: {
      title: 'Preview and timeline',
      hint: 'Configure in-app Web Audio preview and PianoRoll rendering limits. Preview still consumes the same backend PlayPlan.',
      fields: {
        volume: {
          label: 'Preview volume',
          description: 'Web Audio master gain, allowed range 0..0.5. Default is 0.08.',
        },
        waveform: {
          label: 'Preview tone',
          description: 'Soft tones are recommended to reduce harsh high frequencies. This does not change MIDI mapping or game performance.',
        },
        progressHz: {
          label: 'Preview refresh rate',
          description: 'Preview progress update frequency, allowed range 1..30 Hz.',
        },
        pianoRollMaxNotes: {
          label: 'Maximum timeline notes',
          description: 'Limits PianoRoll note blocks per render to keep large MIDI files responsive. Allowed range 100..5000.',
        },
      },
      waveforms: {
        warmSine: 'Warm sine',
        softSine: 'Soft sine',
        mellowTriangle: 'Mellow triangle',
        roundedBell: 'Rounded bell',
        glassPad: 'Glass pad',
        mutedPluck: 'Muted pluck',
        softOrgan: 'Soft organ',
        warmPad: 'Warm pad',
        sine: 'Sine',
        triangle: 'Triangle',
        square: 'Square (bright)',
        sawtooth: 'Sawtooth (bright)',
      },
      actions: {
        reset: 'Restore preview defaults',
      },
    },
  },
};

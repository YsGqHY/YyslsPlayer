import type { Messages } from '@/i18n';

export const playbackEnUS: Messages = {
  settings: {
    playback: {
      title: 'Performance control',
      hint: 'Configure countdown and scheduler behavior for SendInput performance.',
      fields: {
        dryRunDefault: {
          label: 'Use dry-run by default',
          description: 'New performance panels start without injecting system keys so the schedule and PlayPlan can be checked first.',
        },
        lookahead: {
          label: 'Lookahead ms',
          description: 'Backend scheduler lead time, allowed range 5..50 ms. Higher is steadier; lower is tighter.',
        },
        countdown: {
          label: 'Start countdown seconds',
          description: 'Wait 0..10 seconds before performance starts so focus can be switched to the game window.',
        },
      },
      actions: {
        reset: 'Restore performance defaults',
      },
    },
  },
};

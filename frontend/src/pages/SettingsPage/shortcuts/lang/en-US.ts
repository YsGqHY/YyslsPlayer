import type { Messages } from '@/i18n';

export const shortcutsEnUS: Messages = {
  settings: {
    shortcuts: {
      title: 'Global shortcuts',
      hint: 'Register Windows system-wide hotkeys so playback can be controlled even when the game window is focused. Disabling an item unregisters its hotkey.',
      recording: 'Press a combo…',
      recordAria: 'Record shortcut for {{action}}',
      enableAria: 'Enable / disable {{action}} shortcut',
      error: 'Operation failed: {{message}}',
      actions: {
        reset: 'Restore default shortcuts',
        'play-pause': {
          label: 'Play / Pause',
          description: 'Pause while playing, resume while paused. Start from the performance panel when idle.',
        },
        stop: {
          label: 'Stop performance',
          description: 'Stop the current performance session and release all keys.',
        },
        'preview-toggle': {
          label: 'Toggle preview',
          description: 'Toggle the Web Audio preview playback (independent from in-game performance).',
        },
        'emergency-release': {
          label: 'Emergency release all keys',
          description: 'Immediately release every held key to prevent stuck keys. The most important safety net.',
        },
      },
      status: {
        listening: 'Listening. Press a combo with Ctrl / Alt / Win, or a function key F1–F12. Esc to cancel.',
        unsafe: 'That combo would capture system input. Use a Ctrl / Alt / Win combo or a function key.',
        invalid: 'Key not recognized. Try a different combo.',
        conflict: 'Conflicts with another shortcut. Pick a different combo.',
        occupied: 'This combo is already used by another program and could not be registered.',
        failed: 'Registration failed. Try a different combo.',
      },
    },
  },
};

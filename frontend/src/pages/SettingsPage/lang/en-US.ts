import type { Messages } from '@/i18n';

// SettingsPage top-level English copy: only "frame" layer (eyebrow/title, list, themes).
// Per-subpage copy lives in SettingsPage/<name>/lang/ and is registered by register<Name>Locales.
export const settingsPageEnUS: Messages = {
  settings: {
    eyebrow: 'Settings',
    title: 'Preferences',
    list: {
      personalization: {
        label: 'Personalization',
        description: 'Theme, display preferences, and custom palette',
      },
      midiDefaults: {
        label: 'MIDI defaults',
        description: 'Range, transpose, speed, and out-of-range policy',
      },
      library: {
        label: 'Library and import',
        description: 'List loading limit and post-import navigation',
      },
      playback: {
        label: 'Performance control',
        description: 'Countdown and scheduler lookahead',
      },
      shortcuts: {
        label: 'Shortcuts',
        description: 'Global hotkeys that work even when the game is focused',
      },
      preview: {
        label: 'Preview and timeline',
        description: 'Preview tone, volume, and PianoRoll limits',
      },
      language: {
        label: 'Language',
        description: 'Switch the UI language',
      },
      database: {
        label: 'Data storage',
        description: 'Data file location and disk usage',
      },
    },
    themes: {
      system: { label: 'Follow system', description: 'Tracks the OS light / dark preference' },
      light: { label: 'Light', description: 'Default white theme for daytime use' },
      dark: { label: 'Dark', description: 'Dim grey theme for low-light environments' },
      obsidian: { label: 'Obsidian', description: 'Pure black canvas with violet accent' },
      custom: { label: 'Custom', description: 'Your saved custom palette' },
    },
  },
};

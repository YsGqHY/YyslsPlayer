import type { Messages } from '@/i18n';

export const databaseEnUS: Messages = {
  settings: {
    database: {
      title: 'Data storage',
      hint: 'All preferences and settings live in a local data file. You can move the data file to any location you trust; switching migrates the data automatically.',
      currentPathLabel: 'Current location',
      defaultPathLabel: 'Default location',
      sizeLabel: 'Disk usage',
      customBadge: 'Custom',
      defaultBadge: 'Default',
      actions: {
        change: 'Change location…',
        reset: 'Reset to default',
      },
      dialog: {
        title: 'Choose a new data file location',
        message: 'Existing data will be copied to the chosen location.',
        confirm: 'Save here',
        filterDb: 'Data files (*.json)',
      },
      confirmReset: {
        title: 'Restore default location',
        message: 'The data file will be moved back to the platform default directory. Continue?',
        ok: 'Restore',
        cancel: 'Cancel',
      },
      confirmClear: {
        title: 'Clear {{label}}',
        message: 'This will delete all {{rows}} rows in "{{label}}". This action cannot be undone. Continue?',
        ok: 'Clear',
        cancel: 'Cancel',
      },
      feedback: {
        changing: 'Migrating data…',
        changed: 'Switched to the new location',
        resetting: 'Restoring default location…',
        reset: 'Restored to default location',
        clearing: 'Clearing "{{label}}"…',
        cleared: 'Cleared "{{label}}"',
        failed: 'Operation failed: {{message}}',
      },
      charts: {
        share: {
          title: 'Space share',
          hint: 'Relative footprint of each data category in storage.'
        },
        bytes: {
          title: 'Byte usage',
          hint: 'Storage usage by data category.'
        },
        empty: 'No data yet',
        estimatedNote: 'Usage values are estimated.',
        totalCaption: 'Data total',
      },
      tables: {
        title: 'Data collections',
        empty: 'No data collections to display.',
        meta: '{{rows}} rows · {{size}}',
        clear: 'Clear',
        clearAria: 'Clear {{label}}',
        protected: 'Protected',
        protectedAria: '{{label}} is protected from one-click clearing',
        estimatedTag: 'estimated',
        preferences: {
          label: 'Behavior preferences',
          description: 'UI toggles and visibility settings — safe to rebuild.',
        },
        appSettings: {
          label: 'App settings',
          description: 'Theme choice / custom palette / locale. Affects visual identity, not eligible for bulk clear.',
        },
        midiProjects: {
          label: 'MIDI projects',
          description: 'Imported score metadata, file hashes, and default profile references. Protected.',
        },
        midiEvents: {
          label: 'MIDI events',
          description: 'Normalized note events on the absolute timeline, managed with MIDI projects.',
        },
        midiProfiles: {
          label: 'MIDI profiles',
          description: 'Range, transpose, speed, out-of-range policy, and key timing settings. Protected.',
        },
        keymap36: {
          label: '36-key mapping',
          description: 'Lane-to-key, scan code, and modifier mapping profiles. Protected.',
        },
        playHistory: {
          label: 'Play history',
          description: 'Playback session history. Safe to rebuild.'
        },
        hotkeyBindings: {
          label: 'Global shortcuts',
          description: 'Per-action global hotkey bindings. Protected; use the shortcuts page to reset.',
        },
      },
    },
  },
};

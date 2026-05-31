import type { Messages } from '@/i18n';

export const librarySettingsEnUS: Messages = {
  settings: {
    library: {
      title: 'Library and import',
      hint: 'Configure library fetch limits and what happens after a MIDI file is imported.',
      fields: {
        autoOpen: {
          label: 'Open editor after import',
          description: 'After import succeeds, navigate directly to the editor. When disabled, the new score stays selected in the library workbench.',
        },
        listLimit: {
          label: 'Library list limit',
          description: 'Maximum number of scores loaded per refresh, allowed range 5..10000.',
        },
      },
      actions: {
        reset: 'Restore library defaults',
      },
    },
  },
};

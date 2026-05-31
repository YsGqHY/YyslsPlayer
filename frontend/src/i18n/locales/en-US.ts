import type { Locale } from '../types';

// English (US) — shared / framework-level keys only.
// Page-level copy lives in each page's own lang/ directory.
export const enUS: Locale = {
  code: 'en-US',
  englishName: 'English (US)',
  nativeName: 'English',
  messages: {
    app: {
      title: '燕云流音',
    },
    sidebar: {
      brand: 'Y',
      navAriaLabel: 'Primary navigation',
    },
    titleBar: {
      controls: {
        minimise: 'Minimise',
        maximise: 'Maximise / Restore',
        close: 'Close',
      },
    },
    route: {
      home: 'Home',
      library: 'Library',
      editor: 'Editor',
      settings: 'Settings',
    },
    common: {
      save: 'Save',
      cancel: 'Cancel',
      remove: 'Remove',
      reset: 'Reset',
      yes: 'Yes',
      no: 'No',
      loading: 'Loading…',
      error: 'Error',
      followSystem: 'Follow system',
      auto: 'Auto',
    },
    qualityReport: {
      metrics: {
        playableRatio: 'Playable ratio',
        playableSummary: '{{playable}} / {{total}} notes playable',
        noteRange: 'Raw note range',
        rawNotes: 'Total notes {{count}}',
        mappedRange: 'Mapped lanes',
        playableNotes: 'Playable {{count}}',
        outOfRange: 'Out of range',
        dropFoldClamp: 'drop {{dropped}} / fold {{folded}} / nearest {{clamped}}',
        blackKeyCount: 'Black-key lanes',
        chordDensity: 'Max simultaneous keys {{count}}',
        trackChannel: 'Tracks / channels',
        trackChannelHint: 'MIDI structure complexity',
        suggestion: 'Suggested transpose',
        octaveShift: 'Suggested octave {{shift}}',
      },
      warnings: {
        none: 'No obvious risk detected',
        out_of_range: 'Contains out-of-range notes',
        dropped_notes: 'Current policy drops some notes',
        high_chord_density: 'High chord density',
      },
    },
    previewPanel: {
      eyebrow: 'Preview',
      title: 'Web Audio Preview',
      subtitle: '{{duration}} · {{frames}} frames',
      empty: 'Load a score to start previewing.',
      loading: 'Loading preview engine…',
      status: 'State {{state}} · {{active}} active lanes',
      lanes: {
        title: 'Active lanes',
      },
      seek: {
        aria: 'Preview progress, drag or use arrow keys to seek',
      },
      actions: {
        play: 'Play',
        resume: 'Resume',
        pause: 'Pause',
        stop: 'Stop',
        restart: 'Restart',
        refresh: 'Refresh preview',
      },
      states: {
        idle: 'Idle',
        playing: 'Playing',
        paused: 'Paused',
        stopped: 'Stopped',
      },
      errors: {
        prefix: 'Preview failed: ',
        unknown: 'Unknown error',
      },
    },
    performPanel: {
      eyebrow: 'Performance',
      title: 'SendInput Performance',
      subtitle: '{{duration}} · {{frames}} frames · {{mode}}',
      empty: 'Generate a PlayPlan before starting real performance.',
      loading: 'Loading performance plan…',
      status: 'State {{state}} · {{progress}}',
      mode: {
        dryRun: 'Dry-run',
        real: 'Real input',
      },
      actions: {
        start: 'Start',
        resume: 'Resume',
        pause: 'Pause',
        stop: 'Stop',
        releaseAll: 'Release all',
      },
      fields: {
        dryRun: 'Dry-run mode',
        dryRunHelper: 'Do not inject system keys; only run scheduler and logging.',
        realHelper: 'Injects keyboard input. Switch the game to 36-key performance mode first.',
        lookahead: 'Lookahead ms',
      },
      stats: {
        state: 'State',
        session: 'Session',
        none: 'None',
        lookahead: 'Lookahead',
        lookaheadValue: '{{value}} ms',
      },
      states: {
        idle: 'Idle',
        ready: 'Ready',
        playing: 'Playing',
        paused: 'Paused',
        completed: 'Completed',
        stopped: 'Stopped',
        error: 'Error',
      },
      countdown: 'Performance starts in {{seconds}} seconds. Switch focus to the game window.',
      warnings: {
        realMode: 'Confirm the game is in 36-key performance mode before starting.',
      },
      errors: {
        prefix: 'Performance failed: ',
      },
    },
    pianoRoll: {
      eyebrow: '36-lane timeline',
      title: 'Piano Roll View',
      subtitle: '{{duration}} · {{notes}} note blocks',
      loading: 'Building timeline…',
      empty: 'Generate a PlayPlan to show the 36-key timeline.',
      truncated: 'Rendered {{rendered}} note blocks and hidden {{hidden}} to keep the UI responsive.',
      noteTitle: 'lane {{lane}} · source {{source}} · mapped {{normalized}} · start {{start}} · duration {{duration}}ms',
      stats: {
        rendered: 'Rendered {{count}}',
        active: 'Active lanes {{count}}',
        position: 'Position {{position}}',
      },
    },
  },
};

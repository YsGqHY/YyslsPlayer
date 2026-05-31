import { Box, Typography, useTheme } from '@mui/material';
import { useT } from '@/i18n';
import { usePreferences } from '@/preferences';
import type { PlayPlan } from '@/services';
import { pianoRollViewStyles } from './PianoRollView.styles';
import { usePianoRollView } from './usePianoRollView';

export interface PianoRollViewProps {
  plan: PlayPlan | null;
  loading?: boolean;
  compact?: boolean;
  maxNotes?: number;
}

export const PianoRollView = ({ plan, loading = false, compact = false, maxNotes }: PianoRollViewProps) => {
  const theme = useTheme();
  const styles = pianoRollViewStyles(theme);
  const t = useT();
  const { preferences } = usePreferences();
  const vm = usePianoRollView(plan, { maxNotes: maxNotes ?? preferences.pianoRollMaxNotes, tickCount: compact ? 4 : 6 });

  if (loading) {
    return (
      <Box sx={styles.root}>
        <Box sx={styles.empty}>{t('pianoRoll.loading')}</Box>
      </Box>
    );
  }

  if (!plan) {
    return (
      <Box sx={styles.root}>
        <Box sx={styles.empty}>{t('pianoRoll.empty')}</Box>
      </Box>
    );
  }

  return (
    <Box sx={styles.root}>
      <Box sx={styles.header}>
        <Box sx={styles.titleBlock}>
          <Typography sx={styles.eyebrow}>{t('pianoRoll.eyebrow')}</Typography>
          <Typography sx={styles.title}>{t('pianoRoll.title')}</Typography>
          <Typography sx={styles.meta}>{t('pianoRoll.subtitle', {
            duration: formatTime(vm.summary.durationMs),
            notes: String(vm.summary.totalNotes),
          })}</Typography>
        </Box>
        <Box sx={styles.stats}>
          <Box sx={styles.statChip}>{t('pianoRoll.stats.rendered', { count: String(vm.summary.renderedNotes) })}</Box>
          <Box sx={styles.statChip}>{t('pianoRoll.stats.active', { count: String(vm.summary.activeLaneCount) })}</Box>
          <Box sx={styles.statChip}>{t('pianoRoll.stats.position', { position: formatTime(vm.summary.positionMs) })}</Box>
        </Box>
      </Box>

      {vm.summary.hiddenNotes > 0 && (
        <Typography sx={styles.warning}>{t('pianoRoll.truncated', { hidden: String(vm.summary.hiddenNotes), rendered: String(vm.summary.renderedNotes) })}</Typography>
      )}

      <Box sx={styles.viewport}>
        <Box sx={styles.timeline}>
          <Box sx={styles.tickSpacer} />
          <Box sx={styles.tickLayer}>
            {vm.ticks.map((tick) => (
              <Box key={tick.id} sx={{ ...styles.tick, left: `${tick.leftPercent}%` }}>
                <span>{tick.label}</span>
              </Box>
            ))}
          </Box>

          <Box sx={styles.laneLabels}>
            {vm.lanes.map((lane) => (
              <Box key={lane} sx={styles.laneLabel} data-active={vm.activeLanes.has(lane)}>
                <span>{lane}</span>
                <span>{laneToNoteLabel(plan.baseNote, lane)}</span>
              </Box>
            ))}
          </Box>

          <Box sx={styles.laneCanvas}>
            {vm.notes.map((note) => (
              <Box
                key={note.id}
                sx={{
                  ...styles.note,
                  left: `${note.leftPercent}%`,
                  top: `${note.laneTopPercent}%`,
                  width: `${note.widthPercent}%`,
                  opacity: 0.45 + Math.min(0.5, note.velocity / 254),
                }}
                title={t('pianoRoll.noteTitle', {
                  lane: String(note.lane),
                  source: String(note.sourceNote),
                  normalized: String(note.normalizedNote),
                  start: formatTime(note.startMs),
                  duration: String(note.durationMs),
                })}
              />
            ))}
            <Box sx={{ ...styles.cursor, left: `${vm.summary.positionPercent}%` }} />
          </Box>
        </Box>
      </Box>
    </Box>
  );
};

const NOTE_NAMES = ['C', 'C#', 'D', 'D#', 'E', 'F', 'F#', 'G', 'G#', 'A', 'A#', 'B'];

const laneToNoteLabel = (baseNote: number, lane: number): string => {
  const midiNote = baseNote + lane;
  const pitch = ((midiNote % 12) + 12) % 12;
  const octave = Math.floor(midiNote / 12) - 1;
  return `${NOTE_NAMES[pitch] ?? 'C'}${octave}`;
};

const formatTime = (value: number): string => {
  const totalSeconds = Math.max(0, Math.floor(value / 1000));
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  return `${minutes}:${seconds.toString().padStart(2, '0')}`;
};

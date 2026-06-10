import {
  Box,
  Button,
  Checkbox,
  Typography,
  useTheme,
} from '@mui/material';
import { useT } from '@/i18n';
import { libraryPageStyles } from '../LibraryPage.styles';

interface TrackToggleSectionProps {
  enabledTracks: number[] | null;
  trackCount: number;
  onChange: (tracks: number[] | null) => void;
}

export const TrackToggleSection = ({ enabledTracks, trackCount, onChange }: TrackToggleSectionProps) => {
  const theme = useTheme();
  const styles = libraryPageStyles(theme);
  const t = useT();
  const tracks = Array.from({ length: Math.max(0, trackCount) }, (_, index) => index);
  const selected = enabledTracks === null ? tracks : enabledTracks;
  const selectedSet = new Set(selected);

  const toggleTrack = (track: number): void => {
    const next = selectedSet.has(track)
      ? selected.filter((value) => value !== track)
      : [...selected, track].sort((a, b) => a - b);
    onChange(next.length === trackCount ? null : next);
  };

  return (
    <Box sx={styles.section}>
      <Box sx={styles.sectionHeadingRow}>
        <Box>
          <Typography sx={styles.sectionTitle}>{t('library.tracks.title')}</Typography>
          <Typography sx={styles.fieldHelper}>{t('library.tracks.summary', { enabled: String(selected.length), total: String(trackCount) })}</Typography>
        </Box>
        <Box sx={styles.trackActions}>
          <Button variant="outlined" size="small" onClick={() => onChange(null)} disabled={selected.length === trackCount}>
            {t('library.tracks.enableAll')}
          </Button>
          <Button variant="outlined" size="small" onClick={() => onChange([])} disabled={selected.length === 0}>
            {t('library.tracks.disableAll')}
          </Button>
        </Box>
      </Box>
      {tracks.length === 0 ? (
        <Box sx={styles.hint}>{t('library.tracks.empty')}</Box>
      ) : (
        <Box sx={styles.trackToggleGrid}>
          {tracks.map((track) => {
            const checked = selectedSet.has(track);
            return (
              <Box
                key={track}
                component="button"
                type="button"
                sx={checked ? styles.trackToggleActive : styles.trackToggle}
                onClick={() => toggleTrack(track)}
              >
                <Checkbox checked={checked} size="small" tabIndex={-1} />
                <span>{t('library.tracks.item', { track: String(track + 1).padStart(2, '0') })}</span>
              </Box>
            );
          })}
        </Box>
      )}
    </Box>
  );
};

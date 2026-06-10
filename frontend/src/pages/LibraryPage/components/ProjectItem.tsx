import {
  Box,
  Checkbox,
  IconButton,
  Tooltip,
  Typography,
  useTheme,
} from '@mui/material';
import { DeleteRounded as DeleteRoundedIcon } from '@mui/icons-material';
import { PlaylistAddCheckRounded as PlaylistAddCheckRoundedIcon } from '@mui/icons-material';
import { PlaylistAddRounded as PlaylistAddRoundedIcon } from '@mui/icons-material';
import type { MouseEvent } from 'react';
import type { SxProps, Theme } from '@mui/material/styles';
import type { MidiProjectSummary, QualityReport } from '@/services';
import { useT } from '@/i18n';
import { libraryPageStyles } from '../LibraryPage.styles';
import { formatDuration, formatPercent } from '../utils/format';

interface ProjectItemProps {
  index: number;
  project: MidiProjectSummary;
  report: QualityReport | null;
  active: boolean;
  deleting: boolean;
  selectionMode: boolean;
  selected: boolean;
  queued: boolean;
  onSelect: () => void;
  onToggleSelect: () => void;
  onPlayNext: () => void;
  onAddToPlaylist: () => void;
  onDelete: () => void;
}

export const ProjectItem = ({
  index,
  project,
  report,
  active,
  deleting,
  selectionMode,
  selected,
  queued,
  onSelect,
  onToggleSelect,
  onPlayNext,
  onAddToPlaylist,
  onDelete,
}: ProjectItemProps) => {
  const theme = useTheme();
  const styles = libraryPageStyles(theme);
  const t = useT();
  const itemSx = (active || selected ? [styles.trackItem, active ? styles.trackItemActive : null, selected ? styles.trackItemSelected : null].filter(Boolean) : styles.trackItem) as SxProps<Theme>;

  const onDeleteClick = (event: MouseEvent<HTMLButtonElement>) => {
    event.stopPropagation();
    onDelete();
  };

  const onPlayNextClick = (event: MouseEvent<HTMLButtonElement>): void => {
    event.stopPropagation();
    onPlayNext();
  };

  const onAddToPlaylistClick = (event: MouseEvent<HTMLButtonElement>): void => {
    event.stopPropagation();
    onAddToPlaylist();
  };

  const onItemClick = (): void => {
    if (selectionMode) {
      onToggleSelect();
      return;
    }
    onSelect();
  };

  const onCheckboxClick = (event: MouseEvent<HTMLButtonElement>): void => {
    event.stopPropagation();
    onToggleSelect();
  };

  return (
    <Box component="button" type="button" sx={itemSx} onClick={onItemClick}>
      <Box sx={styles.trackIndex}>{selectionMode ? <Checkbox checked={selected} size="small" tabIndex={-1} onClick={onCheckboxClick} /> : index.toString().padStart(2, '0')}</Box>
      <Box sx={styles.trackMain}>
        <Typography sx={styles.trackTitle}>{project.displayName}</Typography>
        <Box sx={styles.trackMeta}>
          <span>{formatDuration(project.durationMs)}</span>
          <span>{t('library.item.notes', { count: String(project.noteCount) })}</span>
          <span>{t('library.item.trackChannel', { tracks: String(project.trackCount), channels: String(project.channelCount) })}</span>
        </Box>
      </Box>
      <Box sx={styles.trackAside}>
        <Typography sx={styles.trackRatio}>{report ? formatPercent(report.playableRatio) : t('library.item.notLoaded')}</Typography>
        {!selectionMode && (
          <Box sx={styles.trackItemActions}>
            <Tooltip title={t('library.playlist.playNext')} placement="top" arrow>
              <Box component="span" sx={styles.trackActionTooltipTarget}>
                <IconButton aria-label={t('library.playlist.playNext')} size="small" onClick={onPlayNextClick}>
                  <PlaylistAddCheckRoundedIcon fontSize="small" />
                </IconButton>
              </Box>
            </Tooltip>
            <Tooltip title={t('library.playlist.addToPlaylist')} placement="top" arrow>
              <Box component="span" sx={styles.trackActionTooltipTarget}>
                <IconButton aria-label={t('library.playlist.addToPlaylist')} size="small" onClick={onAddToPlaylistClick} disabled={queued}>
                  <PlaylistAddRoundedIcon fontSize="small" />
                </IconButton>
              </Box>
            </Tooltip>
            <Tooltip title={t('library.actions.deleteFromLibrary')} placement="top" arrow>
              <Box component="span" sx={styles.trackActionTooltipTarget}>
                <IconButton aria-label={t('library.actions.deleteFromLibrary')} size="small" onClick={onDeleteClick} disabled={deleting}>
                  <DeleteRoundedIcon fontSize="small" />
                </IconButton>
              </Box>
            </Tooltip>
          </Box>
        )}
      </Box>
    </Box>
  );
};

import {
  Box,
  Button,
  IconButton,
  Typography,
  useTheme,
} from '@mui/material';
import { DragIndicatorRounded as DragIndicatorRoundedIcon } from '@mui/icons-material';
import { PlaylistPlayRounded as PlaylistPlayRoundedIcon } from '@mui/icons-material';
import { DeleteRounded as DeleteRoundedIcon } from '@mui/icons-material';
import type { DragEvent, MouseEvent } from 'react';
import type { SxProps, Theme } from '@mui/material/styles';
import type { MidiProjectSummary } from '@/services';
import { useT } from '@/i18n';
import { libraryPageStyles } from '../LibraryPage.styles';
import { formatDuration } from '../utils/format';

interface PlaylistRowProps {
  index: number;
  item: MidiProjectSummary;
  active: boolean;
  dragging: boolean;
  onSelect: () => void;
  onPlay: () => void;
  onRemove: () => void;
  onDragStart: () => void;
  onDragEnd: () => void;
  onDragOver: (targetIndex: number) => void;
}

export const PlaylistRow = ({
  index,
  item,
  active,
  dragging,
  onSelect,
  onPlay,
  onRemove,
  onDragStart,
  onDragEnd,
  onDragOver,
}: PlaylistRowProps) => {
  const theme = useTheme();
  const styles = libraryPageStyles(theme);
  const t = useT();

  const rowSx = (active || dragging ? [styles.playlistRow, active ? styles.playlistRowActive : null, dragging ? styles.playlistRowDragging : null].filter(Boolean) : styles.playlistRow) as SxProps<Theme>;

  const handleDragStart = (event: DragEvent<HTMLDivElement>): void => {
    event.dataTransfer.effectAllowed = 'move';
    event.dataTransfer.setData('text/plain', String(index));
    onDragStart();
  };

  const handleDragOver = (event: DragEvent<HTMLDivElement>): void => {
    event.preventDefault();
    event.dataTransfer.dropEffect = 'move';
    onDragOver(index);
  };

  const handleRemove = (event: MouseEvent<HTMLButtonElement>): void => {
    event.stopPropagation();
    onRemove();
  };

  const handlePlay = (event: MouseEvent<HTMLButtonElement>): void => {
    event.stopPropagation();
    onPlay();
  };

  const dragSx: SxProps<Theme> = {
    ...rowSx,
    ...(dragging && {
      opacity: 0.58,
      transform: 'scale(0.98)',
      boxShadow: `0 4px 12px ${theme.palette.foundation?.bg?.hover ?? 'rgba(0,0,0,0.08)'}`,
    }),
  };

  return (
    <Box sx={dragSx} draggable onDragStart={handleDragStart} onDragEnd={onDragEnd} onDragOver={handleDragOver} onDrop={(event) => event.preventDefault()} onClick={onSelect}>
      <Box sx={styles.playlistDragHandle}><DragIndicatorRoundedIcon fontSize="small" /></Box>
      <Box sx={styles.playlistIndex}>{(index + 1).toString().padStart(2, '0')}</Box>
      <Box sx={styles.playlistRowMain}>
        <Typography sx={styles.trackTitle}>{item.displayName}</Typography>
        <Box sx={styles.trackMeta}>
          <span>{formatDuration(item.durationMs)}</span>
          <span>{t('library.item.notes', { count: String(item.noteCount) })}</span>
          <span>{t('library.item.trackChannel', { tracks: String(item.trackCount), channels: String(item.channelCount) })}</span>
        </Box>
      </Box>
      <Box sx={styles.playlistRowActions}>
        <Button variant={active ? 'contained' : 'outlined'} size="small" startIcon={<PlaylistPlayRoundedIcon />} onClick={handlePlay}>
          {active ? t('library.playlist.playCurrent') : t('library.playlist.play')}
        </Button>
        <IconButton aria-label={t('library.playlist.remove')} size="small" onClick={handleRemove}>
          <DeleteRoundedIcon fontSize="small" />
        </IconButton>
      </Box>
    </Box>
  );
};

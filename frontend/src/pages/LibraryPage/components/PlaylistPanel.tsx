import {
  Box,
  Button,
  Typography,
  useTheme,
} from '@mui/material';
import { AddRounded as AddRoundedIcon } from '@mui/icons-material';
import { ClearRounded as ClearRoundedIcon } from '@mui/icons-material';
import { PlaylistPlayRounded as PlaylistPlayRoundedIcon } from '@mui/icons-material';
import { SelectAllRounded as SelectAllRoundedIcon } from '@mui/icons-material';
import type { ReactNode } from 'react';
import type { MidiProjectSummary } from '@/services';
import { useT } from '@/i18n';
import { PerformPanel } from '@/components/PerformPanel';
import { libraryPageStyles } from '../LibraryPage.styles';
import { PlaylistRow } from './PlaylistRow';
import type { PlaylistMode } from '../useLibraryPage';
import type { UseLibraryPageResult } from '../useLibraryPage';

interface PlaylistPanelProps {
  modes: Array<{ id: PlaylistMode; label: string; icon: ReactNode }>;
  selectedProject: MidiProjectSummary | null;
  vm: UseLibraryPageResult;
}

export const PlaylistPanel = ({ modes, selectedProject, vm }: PlaylistPanelProps) => {
  const theme = useTheme();
  const styles = libraryPageStyles(theme);
  const t = useT();

  return (
    <Box sx={styles.playlistPanel}>
      <Box sx={styles.playlistHero}>
        <Box sx={styles.playlistHeroText}>
          <Typography sx={styles.eyebrow}>{t('library.playlist.eyebrow')}</Typography>
          <Typography component="h2" sx={styles.detailTitle}>{t('library.playlist.title')}</Typography>
          <Typography sx={styles.detailMeta}>{t('library.playlist.subtitle', { count: String(vm.playlistItems.length) })}</Typography>
        </Box>
        <Box sx={styles.playlistHeroActions}>
          <Button variant="outlined" size="small" startIcon={<AddRoundedIcon />} onClick={vm.addSelectedProjectToPlaylist} disabled={!selectedProject}>
            {t('library.playlist.addCurrent')}
          </Button>
          <Button variant="outlined" size="small" startIcon={<SelectAllRoundedIcon />} onClick={vm.addFilteredProjectsToPlaylist} disabled={vm.filteredProjects.length === 0}>
            {t('library.playlist.addFiltered')}
          </Button>
          <Button variant="outlined" size="small" startIcon={<ClearRoundedIcon />} onClick={vm.clearPlaylist} disabled={vm.playlistItems.length === 0}>
            {t('library.playlist.clear')}
          </Button>
        </Box>
      </Box>

      <Box sx={styles.playlistToolbar}>
        <Box sx={styles.playlistModeGroup}>
          {modes.map((mode) => (
            <Box
              key={mode.id}
              component="button"
              type="button"
              sx={vm.playlistMode === mode.id ? styles.playlistModeActive : styles.playlistMode}
              onClick={() => vm.setPlaylistMode(mode.id)}
            >
              {mode.icon}
              <span>{mode.label}</span>
            </Box>
          ))}
        </Box>
        <Typography sx={styles.panelSubtle}>{t('library.playlist.dragHint')}</Typography>
      </Box>

      {vm.playlistItems.length === 0 ? (
        <Box sx={styles.playlistEmpty}>
          <PlaylistPlayRoundedIcon fontSize="large" />
          <Typography sx={styles.emptyTitle}>{t('library.playlist.emptyTitle')}</Typography>
          <Typography sx={styles.emptyText}>{t('library.playlist.emptyText')}</Typography>
        </Box>
      ) : (
        <Box sx={styles.playlistList}>
          {vm.playlistItems.map((item, index) => (
            <PlaylistRow
              key={item.project.id}
              index={index}
              item={item.project}
              active={vm.playlistCurrentIndex === index}
              dragging={vm.playlistDraggingIndex === index}
              onSelect={() => void vm.selectPlaylistItem(index)}
              onPlay={() => void vm.playPlaylistItem(index)}
              onRemove={() => vm.removePlaylistItem(item.project.id)}
              onDragStart={() => vm.startPlaylistDrag(index)}
              onDragEnd={vm.finishPlaylistDrag}
              onDragOver={(targetIndex) => vm.movePlaylistItem(vm.playlistDraggingIndex ?? targetIndex, targetIndex)}
            />
          ))}
        </Box>
      )}

      <Box sx={styles.panelBody}>
        <PerformPanel
          plan={vm.previewPlan}
          loading={vm.previewLoading}
          error={vm.previewError}
          autoStartToken={vm.playlistAutoStartToken}
          onPlayerState={vm.handlePlayerState}
          onStart={vm.markPlaylistStarted}
        />
      </Box>
    </Box>
  );
};

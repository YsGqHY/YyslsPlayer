import AddRoundedIcon from '@mui/icons-material/AddRounded';
import ClearRoundedIcon from '@mui/icons-material/ClearRounded';
import DeleteRoundedIcon from '@mui/icons-material/DeleteRounded';
import DragIndicatorRoundedIcon from '@mui/icons-material/DragIndicatorRounded';
import DoneAllRoundedIcon from '@mui/icons-material/DoneAllRounded';
import DriveFolderUploadRoundedIcon from '@mui/icons-material/DriveFolderUploadRounded';
import FileUploadRoundedIcon from '@mui/icons-material/FileUploadRounded';
import GraphicEqRoundedIcon from '@mui/icons-material/GraphicEqRounded';
import LibraryAddCheckRoundedIcon from '@mui/icons-material/LibraryAddCheckRounded';
import KeyboardRoundedIcon from '@mui/icons-material/KeyboardRounded';
import MusicNoteRoundedIcon from '@mui/icons-material/MusicNoteRounded';
import PlaylistAddRoundedIcon from '@mui/icons-material/PlaylistAddRounded';
import PlaylistAddCheckRoundedIcon from '@mui/icons-material/PlaylistAddCheckRounded';
import PlaylistPlayRoundedIcon from '@mui/icons-material/PlaylistPlayRounded';
import RefreshRoundedIcon from '@mui/icons-material/RefreshRounded';
import RestartAltRoundedIcon from '@mui/icons-material/RestartAltRounded';
import SaveRoundedIcon from '@mui/icons-material/SaveRounded';
import SearchRoundedIcon from '@mui/icons-material/SearchRounded';
import SelectAllRoundedIcon from '@mui/icons-material/SelectAllRounded';
import ShuffleRoundedIcon from '@mui/icons-material/ShuffleRounded';
import SpeedRoundedIcon from '@mui/icons-material/SpeedRounded';
import SyncRoundedIcon from '@mui/icons-material/SyncRounded';
import TuneRoundedIcon from '@mui/icons-material/TuneRounded';
import {
  Box,
  Button,
  Checkbox,
  IconButton,
  MenuItem,
  Select,
  TextField,
  Tooltip,
  Typography,
  useTheme,
} from '@mui/material';
import { useEffect, useRef, useState } from 'react';
import type { DragEvent, MouseEvent, ReactNode } from 'react';
import type { SxProps, Theme } from '@mui/material/styles';
import { PerformPanel } from '@/components/PerformPanel';
import { PianoRollView } from '@/components/PianoRollView';
import { PreviewPanel } from '@/components/PreviewPanel';
import { QualityReportPanel } from '@/components/QualityReportPanel';
import { useT } from '@/i18n';
import { useRouter } from '@/router';
import { EditorSelectionService, MidiService, NativeDialogs, type MidiProfile, type MidiProjectSummary, type OutOfRangePolicy, type QualityReport } from '@/services';
import { libraryPageStyles } from './LibraryPage.styles';
import { useLibraryPage, type LibraryPanel, type LibraryProfileForm, type LibrarySortValue, type PlaylistMode } from './useLibraryPage';

export const LibraryPage = () => {
  const theme = useTheme();
  const styles = libraryPageStyles(theme);
  const t = useT();
  const router = useRouter();
  const vm = useLibraryPage();
  const selectedProject = vm.selected?.project ?? null;
  const selectedReport = vm.previewPlan?.report ?? vm.selected?.qualityReport ?? null;
  const profiles = vm.selected ? uniqueProfiles(vm.selected.defaultProfile, vm.selected.profiles) : [];
  const [dragActive, setDragActive] = useState(false);
  const dragDepthRef = useRef(0);

  const openImportedProject = (projectId: number | null): void => {
    if (projectId && vm.autoOpenImportedProject) {
      EditorSelectionService.setProjectId(projectId);
      router.navigate('editor');
    }
  };

  const importMidiFiles = async (): Promise<void> => {
    const projectId = await vm.importMidiFiles(t('library.dialog.openFilesTitle'), t('library.dialog.filterName'));
    openImportedProject(projectId);
  };

  const importMidiDirectory = async (): Promise<void> => {
    const projectId = await vm.importMidiDirectory(t('library.dialog.openDirectoryTitle'));
    openImportedProject(projectId);
  };

  // 文件拖放：真实路径只能由后端经 WindowFilesDropped 事件下发（浏览器拿不到本地路径）。
  // 这里订阅后端推送的路径并交给 ViewModel 导入，导入后按偏好自动打开项目。
  useEffect(() => {
    const off = MidiService.onFilesDropped((paths) => {
      void vm.importDroppedPaths(paths).then(openImportedProject);
    });
    return off;
    // openImportedProject 依赖 vm/ router，稳定性由 vm 内部 useCallback 保证；这里只需在挂载时订阅一次。
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [vm.importDroppedPaths, vm.autoOpenImportedProject]);

  // 下列 drag 处理器只维护高亮覆盖层的显隐；真正的文件路径解析与导入由 Wails 后端链路完成。
  // 用 dragenter/dragleave 计数避免子元素间移动导致的闪烁。
  const isFileDrag = (event: DragEvent<HTMLDivElement>): boolean =>
    Array.from(event.dataTransfer?.types ?? []).includes('Files');

  const handleDragEnter = (event: DragEvent<HTMLDivElement>): void => {
    if (!isFileDrag(event)) return;
    event.preventDefault();
    dragDepthRef.current += 1;
    setDragActive(true);
  };

  const handleDragOver = (event: DragEvent<HTMLDivElement>): void => {
    if (!isFileDrag(event)) return;
    event.preventDefault();
  };

  const handleDragLeave = (event: DragEvent<HTMLDivElement>): void => {
    if (!isFileDrag(event)) return;
    dragDepthRef.current = Math.max(0, dragDepthRef.current - 1);
    if (dragDepthRef.current === 0) {
      setDragActive(false);
    }
  };

  const handleDrop = (event: DragEvent<HTMLDivElement>): void => {
    if (!isFileDrag(event)) return;
    event.preventDefault();
    dragDepthRef.current = 0;
    setDragActive(false);
  };

  const confirmDeleteProject = async (project: MidiProjectSummary): Promise<void> => {
    const confirmed = await NativeDialogs.confirm({
      title: t('library.batch.deleteOneTitle'),
      message: t('library.batch.deleteOneMessage', { name: project.displayName }),
      okLabel: t('library.batch.deleteConfirm'),
      cancelLabel: t('library.batch.cancel'),
    });
    if (confirmed) {
      await vm.deleteProject(project.id);
    }
  };

  const confirmDeleteSelected = async (): Promise<void> => {
    if (vm.selectedCount === 0) return;
    const confirmed = await NativeDialogs.confirm({
      title: t('library.batch.deleteSelectedTitle'),
      message: t('library.batch.deleteSelectedMessage', { count: String(vm.selectedCount) }),
      okLabel: t('library.batch.deleteConfirm'),
      cancelLabel: t('library.batch.cancel'),
    });
    if (confirmed) {
      await vm.deleteSelectedProjects();
    }
  };

  const sortItems: Array<{ value: LibrarySortValue; label: string }> = [
    { value: 'createdAt:asc', label: t('library.sort.addedAsc') },
    { value: 'createdAt:desc', label: t('library.sort.addedDesc') },
    { value: 'fileName:asc', label: t('library.sort.fileNameAsc') },
    { value: 'fileName:desc', label: t('library.sort.fileNameDesc') },
    { value: 'durationMs:asc', label: t('library.sort.durationAsc') },
    { value: 'durationMs:desc', label: t('library.sort.durationDesc') },
    { value: 'fileSizeBytes:asc', label: t('library.sort.fileSizeAsc') },
    { value: 'fileSizeBytes:desc', label: t('library.sort.fileSizeDesc') },
  ];

  const panelItems: Array<{ id: LibraryPanel; label: string; description: string }> = [
    { id: 'perform', label: t('library.panels.perform.label'), description: t('library.panels.perform.description') },
    { id: 'overview', label: t('library.panels.overview.label'), description: t('library.panels.overview.description') },
    { id: 'settings', label: t('library.panels.settings.label'), description: t('library.panels.settings.description') },
    { id: 'preview', label: t('library.panels.preview.label'), description: t('library.panels.preview.description') },
  ];

  const playlistModes: Array<{ id: PlaylistMode; label: string; icon: ReactNode }> = [
    { id: 'shuffle', label: t('library.playlist.modes.shuffle'), icon: <ShuffleRoundedIcon fontSize="small" /> },
    { id: 'sequence', label: t('library.playlist.modes.sequence'), icon: <PlaylistPlayRoundedIcon fontSize="small" /> },
    { id: 'loop', label: t('library.playlist.modes.loop'), icon: <SyncRoundedIcon fontSize="small" /> },
    { id: 'singleLoop', label: t('library.playlist.modes.singleLoop'), icon: <RestartAltRoundedIcon fontSize="small" /> },
  ];

  return (
    <Box sx={styles.root}>
      <Box sx={styles.workspace}>
        <Box
          sx={styles.libraryColumn}
          data-file-drop-target=""
          onDragEnter={handleDragEnter}
          onDragOver={handleDragOver}
          onDragLeave={handleDragLeave}
          onDrop={handleDrop}
        >
          {dragActive && (
            <Box sx={styles.dropOverlay}>
              <FileUploadRoundedIcon fontSize="large" />
              <Typography sx={styles.dropOverlayTitle}>{t('library.drop.title')}</Typography>
              <Typography sx={styles.dropOverlayText}>{t('library.drop.hint')}</Typography>
            </Box>
          )}
          <Box sx={styles.brandBlock}>
            <Box sx={styles.brandHeaderRow}>
              <Box sx={styles.brandTitleBlock}>
                <Typography sx={styles.eyebrow}>{t('library.eyebrow')}</Typography>
                <Typography component="h1" sx={styles.title}>{t('library.title')}</Typography>
              </Box>
              <IconButton
                aria-label={t('library.playlist.toggle')}
                sx={vm.playlistOpen ? styles.playlistToggleActive : styles.playlistToggle}
                onClick={vm.togglePlaylist}
              >
                <PlaylistPlayRoundedIcon />
              </IconButton>
            </Box>
          </Box>

          <Box sx={styles.libraryTools}>
            <TextField
              value={vm.query}
              onChange={(event) => vm.setQuery(event.target.value)}
              placeholder={t('library.search.placeholder')}
              size="small"
              sx={styles.searchField}
              slotProps={{ input: { startAdornment: <SearchRoundedIcon fontSize="small" /> } }}
            />
            <Box sx={styles.toolButtons}>
              <Button
                variant="outlined"
                startIcon={<RefreshRoundedIcon />}
                onClick={() => void vm.refresh()}
                disabled={vm.loading || vm.importing}
              >
                {vm.loading ? t('library.actions.refreshing') : t('library.actions.refresh')}
              </Button>
              <Button
                variant="contained"
                startIcon={<FileUploadRoundedIcon />}
                onClick={() => void importMidiFiles()}
                disabled={vm.importing}
              >
                {vm.importing ? t('library.actions.importing') : t('library.actions.importFiles')}
              </Button>
              <Button
                variant="contained"
                startIcon={<DriveFolderUploadRoundedIcon />}
                onClick={() => void importMidiDirectory()}
                disabled={vm.importing}
              >
                {vm.importing ? t('library.actions.importing') : t('library.actions.importDirectory')}
              </Button>
            </Box>
          </Box>

          <Box sx={styles.listHeader}>
            <Box>
              <Typography sx={styles.panelTitle}>{t('library.list.title')}</Typography>
              <Typography sx={styles.panelSubtle}>{t('library.list.filtered', {
                shown: String(vm.filteredProjects.length),
                total: String(vm.projects.length),
              })}</Typography>
            </Box>
            <Box sx={styles.listHeaderActions}>
              <Select
                value={vm.sortValue}
                onChange={(event) => vm.setSortValue(event.target.value as LibrarySortValue)}
                size="small"
                sx={styles.sortSelect}
                displayEmpty
                aria-label={t('library.sort.label')}
              >
                {sortItems.map((item) => (
                  <MenuItem key={item.value} value={item.value}>{item.label}</MenuItem>
                ))}
              </Select>
              <Button
                variant={vm.selectionMode ? 'contained' : 'outlined'}
                size="small"
                startIcon={<LibraryAddCheckRoundedIcon />}
                onClick={() => {
                  if (vm.selectionMode) {
                    vm.exitSelectionMode();
                    return;
                  }
                  vm.enterSelectionMode();
                }}
                disabled={vm.batchDeleting || (!vm.selectionMode && (vm.projects.length === 0 || vm.importing))}
              >
                {vm.selectionMode ? t('library.batch.exit') : t('library.batch.enter')}
              </Button>
            </Box>
          </Box>

          {vm.selectionMode && (
            <Box sx={styles.batchToolbar}>
              <Box sx={styles.batchInfo}>
                <Typography sx={styles.batchTitle}>{t('library.batch.title')}</Typography>
                <Typography sx={styles.panelSubtle}>{t('library.batch.selected', { count: String(vm.selectedCount) })}</Typography>
              </Box>
              <Box sx={styles.batchActions}>
                <Button
                  variant="outlined"
                  size="small"
                  startIcon={vm.allFilteredSelected ? <DoneAllRoundedIcon /> : <SelectAllRoundedIcon />}
                  onClick={() => {
                    vm.selectAllFilteredProjects();
                  }}
                  disabled={vm.filteredProjects.length === 0 || vm.allFilteredSelected || vm.batchDeleting}
                >
                  {vm.allFilteredSelected ? t('library.batch.selectedAll') : t('library.batch.selectFiltered')}
                </Button>
                <Button
                  variant="outlined"
                  size="small"
                  startIcon={<ClearRoundedIcon />}
                  onClick={() => {
                    vm.clearProjectSelection();
                  }}
                  disabled={vm.selectedCount === 0 || vm.batchDeleting}
                >
                  {t('library.batch.clearSelection')}
                </Button>
                <Button
                  variant="contained"
                  size="small"
                  sx={styles.dangerButton}
                  startIcon={<DeleteRoundedIcon />}
                  onClick={() => void confirmDeleteSelected()}
                  disabled={vm.selectedCount === 0 || vm.batchDeleting}
                >
                  {vm.batchDeleting ? t('library.batch.deleting') : t('library.batch.deleteSelected')}
                </Button>
              </Box>
            </Box>
          )}

          {vm.error && <Box sx={styles.error}>{t('library.errors.prefix')}{vm.error}</Box>}
          {vm.importError && <Box sx={styles.error}>{t('library.errors.importPrefix')}{vm.importError}</Box>}
          {vm.importSummary && (
            <Box sx={vm.importSummary.failedCount > 0 ? styles.error : styles.hint}>
              {t('library.importSummary.result', {
                imported: String(vm.importSummary.importedCount),
                skipped: String(vm.importSummary.skippedCount),
                failed: String(vm.importSummary.failedCount),
                total: String(vm.importSummary.totalCount),
              })}
            </Box>
          )}
          {vm.batchResult && (
            <Box sx={vm.batchResult.failedCount > 0 ? styles.error : styles.hint}>
              {t('library.batch.result', {
                deleted: String(vm.batchResult.deletedCount),
                failed: String(vm.batchResult.failedCount),
                total: String(vm.batchResult.totalCount),
              })}
            </Box>
          )}

          {vm.filteredProjects.length === 0 ? (
            <Box sx={styles.emptyList}>
              <MusicNoteRoundedIcon fontSize="large" />
              <Typography>{vm.loading ? t('library.list.loading') : t(vm.query ? 'library.list.noSearchResult' : 'library.list.empty')}</Typography>
            </Box>
          ) : (
            <Box sx={styles.trackList}>
              {vm.filteredProjects.map((project, index) => (
                <ProjectItem
                  key={project.id}
                  index={index + 1}
                  project={project}
                  report={vm.selected?.project.id === project.id ? selectedReport : null}
                  active={vm.selected?.project.id === project.id}
                  deleting={vm.deletingId === project.id}
                  selectionMode={vm.selectionMode}
                  selected={vm.selectedProjectIds.includes(project.id)}
                  queued={vm.playlistItems.some((item) => item.project.id === project.id)}
                  onSelect={() => void vm.selectProject(project.id)}
                  onToggleSelect={() => {
                    vm.toggleProjectSelection(project.id);
                  }}
                  onPlayNext={() => vm.addProjectNextInPlaylist(project)}
                  onAddToPlaylist={() => vm.addProjectToPlaylist(project)}
                  onDelete={() => void confirmDeleteProject(project)}
                />
              ))}
            </Box>
          )}
        </Box>

        <Box sx={styles.detailColumn}>
          {vm.playlistOpen ? (
            <PlaylistPanel
              modes={playlistModes}
              selectedProject={selectedProject}
              vm={vm}
            />
          ) : (
            <>
          <Box sx={styles.detailHero}>
            {selectedProject ? (
              <>
                <Box sx={styles.detailTitleBlock}>
                  <Typography sx={styles.eyebrow}>{t('library.workbench.eyebrow')}</Typography>
                  <Typography component="h2" sx={styles.detailTitle}>{selectedProject.displayName}</Typography>
                  <Typography sx={styles.detailMeta}>{t('library.workbench.subtitle', {
                    file: selectedProject.fileName,
                    duration: formatDuration(selectedProject.durationMs),
                  })}</Typography>
                </Box>
                <Box sx={styles.quickStats}>
                  <StatChip icon={<MusicNoteRoundedIcon fontSize="small" />} label={t('library.stats.notes')} value={String(selectedProject.noteCount)} />
                  <StatChip icon={<GraphicEqRoundedIcon fontSize="small" />} label={t('library.stats.tracks')} value={String(selectedProject.trackCount)} />
                  <StatChip icon={<KeyboardRoundedIcon fontSize="small" />} label={t('library.stats.channels')} value={String(selectedProject.channelCount)} />
                  <StatChip icon={<TuneRoundedIcon fontSize="small" />} label={t('library.stats.playable')} value={selectedReport ? formatPercent(selectedReport.playableRatio) : '-'} />
                </Box>
              </>
            ) : (
              <Box sx={styles.detailEmptyHero}>
                <Typography sx={styles.eyebrow}>{t('library.workbench.eyebrow')}</Typography>
                <Typography component="h2" sx={styles.detailTitle}>{t('library.workbench.emptyTitle')}</Typography>
                <Typography sx={styles.detailMeta}>{t('library.workbench.emptySubtitle')}</Typography>
              </Box>
            )}
          </Box>

          {vm.detailLoading && <Box sx={styles.hint}>{t('library.workbench.loadingDetail')}</Box>}

          <Box sx={styles.panelTabs}>
            {panelItems.map((item) => {
              const active = vm.activePanel === item.id;
              return (
                <Box
                  key={item.id}
                  component="button"
                  type="button"
                  sx={active ? styles.panelTabActive : styles.panelTab}
                  onClick={() => vm.setActivePanel(item.id)}
                  disabled={!vm.selected}
                >
                  <Typography sx={styles.panelTabLabel}>{item.label}</Typography>
                  <Typography sx={styles.panelTabDesc}>{item.description}</Typography>
                </Box>
              );
            })}
          </Box>

          {!vm.selected ? (
            <Box sx={styles.emptyDetail}>
              <MusicNoteRoundedIcon fontSize="large" />
              <Typography sx={styles.emptyTitle}>{t('library.detail.emptyTitle')}</Typography>
              <Typography sx={styles.emptyText}>{t('library.detail.emptyText')}</Typography>
            </Box>
          ) : (
            <Box sx={styles.panelBody}>
              {vm.activePanel === 'overview' && (
                <OverviewPanel report={selectedReport} onRefresh={vm.refreshPreview} previewLoading={vm.previewLoading} />
              )}
              {vm.activePanel === 'settings' && (
                <SettingsPanel
                  profiles={profiles}
                  selectedProfileId={vm.selectedProfileId}
                  form={vm.form}
                  isDirty={vm.isDirty}
                  saving={vm.saving}
                  saveError={vm.saveError}
                  previewError={vm.previewError}
                  report={selectedReport}
                  trackCount={vm.selected.project.trackCount}
                  onSelectProfile={vm.selectProfile}
                  onUpdateField={vm.updateField}
                  onReset={vm.resetForm}
                  onSave={vm.save}
                />
              )}
              {vm.activePanel === 'preview' && (
                <Box sx={styles.panelStack}>
                  <PreviewPanel plan={vm.previewPlan} loading={vm.previewLoading} error={vm.previewError} compact onRefresh={vm.refreshPreview} />
                  <PianoRollView plan={vm.previewPlan} loading={vm.previewLoading} compact />
                </Box>
              )}
              {vm.activePanel === 'perform' && (
                <PerformPanel plan={vm.previewPlan} loading={vm.previewLoading} error={vm.previewError} />
              )}
            </Box>
          )}
            </>
          )}
        </Box>
      </Box>
    </Box>
  );
};

type LibraryPageVM = ReturnType<typeof useLibraryPage>;

interface PlaylistPanelProps {
  modes: Array<{ id: PlaylistMode; label: string; icon: ReactNode }>;
  selectedProject: MidiProjectSummary | null;
  vm: LibraryPageVM;
}

const PlaylistPanel = ({ modes, selectedProject, vm }: PlaylistPanelProps) => {
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

const PlaylistRow = ({ index, item, active, dragging, onSelect, onPlay, onRemove, onDragStart, onDragEnd, onDragOver }: PlaylistRowProps) => {
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

  return (
    <Box sx={rowSx} draggable onDragStart={handleDragStart} onDragEnd={onDragEnd} onDragOver={handleDragOver} onDrop={(event) => event.preventDefault()} onClick={onSelect}>
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

const ProjectItem = ({ index, project, report, active, deleting, selectionMode, selected, queued, onSelect, onToggleSelect, onPlayNext, onAddToPlaylist, onDelete }: ProjectItemProps) => {
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

const OverviewPanel = ({ report, previewLoading, onRefresh }: { report: QualityReport | null; previewLoading: boolean; onRefresh: () => Promise<void> }) => {
  const theme = useTheme();
  const styles = libraryPageStyles(theme);
  const t = useT();

  return (
    <Box sx={styles.panelStack}>
      <Box sx={styles.panelHeaderCard}>
        <Box>
          <Typography sx={styles.sectionEyebrow}>{t('library.overview.eyebrow')}</Typography>
          <Typography sx={styles.sectionBigTitle}>{t('library.overview.title')}</Typography>
          <Typography sx={styles.sectionDesc}>{t('library.overview.description')}</Typography>
        </Box>
        <Button variant="outlined" startIcon={<RefreshRoundedIcon />} onClick={() => void onRefresh()} disabled={previewLoading}>
          {t('library.actions.refreshPreview')}
        </Button>
      </Box>
      {report ? <QualityReportPanel report={report} /> : <Box sx={styles.hint}>{t('library.overview.noReport')}</Box>}
    </Box>
  );
};

type EditableProfileField = 'baseNote' | 'transpose' | 'octaveShift' | 'speed' | 'outOfRangePolicy' | 'minPressMs' | 'releaseGapMs' | 'keymapProfileId' | 'enabledTracks';
type EditableProfileForm = Pick<LibraryProfileForm, EditableProfileField>;

interface SettingsPanelProps {
  profiles: MidiProfile[];
  selectedProfileId: number;
  form: EditableProfileForm;
  isDirty: boolean;
  saving: boolean;
  saveError: string | null;
  previewError: string | null;
  report: QualityReport | null;
  trackCount: number;
  onSelectProfile: (profileId: number) => void;
  onUpdateField: <K extends EditableProfileField>(field: K, value: LibraryProfileForm[K]) => void;
  onReset: () => void;
  onSave: () => Promise<void>;
}

const SettingsPanel = ({
  profiles,
  selectedProfileId,
  form,
  isDirty,
  saving,
  saveError,
  previewError,
  report,
  trackCount,
  onSelectProfile,
  onUpdateField,
  onReset,
  onSave,
}: SettingsPanelProps) => {
  const theme = useTheme();
  const styles = libraryPageStyles(theme);
  const t = useT();

  return (
    <Box sx={styles.settingsGrid}>
      <Box sx={styles.settingsMain}>
        <Box sx={styles.section}>
          <Typography sx={styles.sectionTitle}>{t('library.inspector.profile')}</Typography>
          <Select value={String(selectedProfileId)} onChange={(event) => onSelectProfile(Number(event.target.value))} fullWidth size="small">
            {profiles.map((profile) => (
              <MenuItem key={profile.id} value={String(profile.id)}>{profile.name}</MenuItem>
            ))}
          </Select>
        </Box>

        <Box sx={styles.section}>
          <Box sx={styles.sectionHeadingRow}>
            <Typography sx={styles.sectionTitle}>{t('library.inspector.range')}</Typography>
            <Typography sx={styles.pill}>{t('library.inspector.baseRange')}</Typography>
          </Box>
          <Box sx={styles.formGrid}>
            <Field label={t('library.fields.baseNote')} helper={t('library.fields.baseNoteHelper')}>
              <NumberField value={form.baseNote} min={0} max={127} onChange={(value) => onUpdateField('baseNote', value)} />
            </Field>
            <Field label={t('library.fields.transpose')} helper={t('library.fields.transposeHelper')}>
              <NumberField value={form.transpose} min={-24} max={24} onChange={(value) => onUpdateField('transpose', value)} />
            </Field>
            <Field label={t('library.fields.octaveShift')} helper={t('library.fields.octaveShiftHelper')}>
              <NumberField value={form.octaveShift} min={-3} max={3} onChange={(value) => onUpdateField('octaveShift', value)} />
            </Field>
            <Field label={t('library.fields.speed')} helper={t('library.fields.speedHelper')}>
              <NumberField value={form.speed} min={0.25} max={3} step={0.05} onChange={(value) => onUpdateField('speed', value)} />
            </Field>
          </Box>
        </Box>

        <TrackToggleSection
          enabledTracks={form.enabledTracks}
          trackCount={trackCount}
          onChange={(tracks) => onUpdateField('enabledTracks', tracks)}
        />

        <Box sx={styles.section}>
          <Box sx={styles.sectionHeadingRow}>
            <Typography sx={styles.sectionTitle}>{t('library.inspector.mapping')}</Typography>
            <SpeedRoundedIcon fontSize="small" />
          </Box>
          <Field label={t('library.fields.outOfRangePolicy')} helper={t('library.fields.policyHelper')}>
            <Select value={form.outOfRangePolicy} onChange={(event) => onUpdateField('outOfRangePolicy', event.target.value as OutOfRangePolicy)} fullWidth size="small">
              <MenuItem value="drop">{t('library.policies.drop')}</MenuItem>
              <MenuItem value="octaveFold">{t('library.policies.octaveFold')}</MenuItem>
              <MenuItem value="nearest">{t('library.policies.nearest')}</MenuItem>
            </Select>
          </Field>
          <Box sx={styles.formGrid}>
            <Field label={t('library.fields.minPressMs')} helper={t('library.fields.minPressHelper')}>
              <NumberField value={form.minPressMs} min={10} max={300} onChange={(value) => onUpdateField('minPressMs', value)} />
            </Field>
            <Field label={t('library.fields.releaseGapMs')} helper={t('library.fields.releaseGapHelper')}>
              <NumberField value={form.releaseGapMs} min={0} max={200} onChange={(value) => onUpdateField('releaseGapMs', value)} />
            </Field>
          </Box>
          <Field label={t('library.fields.keymapProfileId')} helper={t('library.fields.keymapHelper')}>
            <NumberField value={form.keymapProfileId} min={1} onChange={(value) => onUpdateField('keymapProfileId', value)} />
          </Field>
        </Box>
      </Box>

      <Box sx={styles.settingsAside}>
        <Box sx={styles.section}>
          <Typography sx={styles.sectionTitle}>{t('library.inspector.summary')}</Typography>
          <MiniMetric label={t('library.summary.playableRatio')} value={report ? formatPercent(report.playableRatio) : '-'} />
          <MiniMetric label={t('library.summary.outOfRange')} value={report ? String(report.outOfRangeCount) : '-'} />
          <MiniMetric label={t('library.summary.suggestedTranspose')} value={report ? formatSigned(report.suggestedTranspose) : '-'} />
          <MiniMetric label={t('library.summary.suggestedOctave')} value={report ? formatSigned(report.suggestedOctaveShift) : '-'} />
        </Box>

        {saveError && <Box sx={styles.error}>{t('library.errors.savePrefix')}{saveError}</Box>}
        {previewError && <Box sx={styles.error}>{t('library.errors.previewPrefix')}{previewError}</Box>}
        {!isDirty && <Box sx={styles.hint}>{t('library.inspector.clean')}</Box>}

        <Box sx={styles.settingsActions}>
          <Button variant="outlined" startIcon={<RestartAltRoundedIcon />} onClick={onReset} disabled={!isDirty || saving}>
            {t('library.actions.reset')}
          </Button>
          <Button variant="contained" startIcon={<SaveRoundedIcon />} onClick={() => void onSave()} disabled={!isDirty || saving}>
            {saving ? t('library.actions.saving') : t('library.actions.save')}
          </Button>
        </Box>
      </Box>
    </Box>
  );
};

const TrackToggleSection = ({ enabledTracks, trackCount, onChange }: { enabledTracks: number[] | null; trackCount: number; onChange: (tracks: number[] | null) => void }) => {
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

const Field = ({ label, helper, children }: { label: string; helper: string; children: ReactNode }) => {
  const theme = useTheme();
  const styles = libraryPageStyles(theme);
  return (
    <Box sx={styles.field}>
      <Typography sx={styles.fieldLabel}>{label}</Typography>
      {children}
      <Typography sx={styles.fieldHelper}>{helper}</Typography>
    </Box>
  );
};

const NumberField = ({ value, min, max, step = 1, onChange }: { value: number; min?: number; max?: number; step?: number; onChange: (value: number) => void }) => (
  <TextField
    type="number"
    value={value}
    onChange={(event) => onChange(Number(event.target.value))}
    fullWidth
    size="small"
    slotProps={{ htmlInput: { min, max, step } }}
  />
);

const StatChip = ({ icon, label, value }: { icon: ReactNode; label: string; value: string }) => {
  const theme = useTheme();
  const styles = libraryPageStyles(theme);
  return (
    <Box sx={styles.statChip}>
      {icon}
      <Box>
        <Typography sx={styles.statValue}>{value}</Typography>
        <Typography sx={styles.statLabel}>{label}</Typography>
      </Box>
    </Box>
  );
};

const MiniMetric = ({ label, value }: { label: string; value: string }) => {
  const theme = useTheme();
  const styles = libraryPageStyles(theme);
  return (
    <Box sx={styles.miniMetric}>
      <Typography sx={styles.miniMetricLabel}>{label}</Typography>
      <Typography sx={styles.miniMetricValue}>{value}</Typography>
    </Box>
  );
};

const uniqueProfiles = (defaultProfile: MidiProfile, profiles: MidiProfile[]): MidiProfile[] => {
  const seen = new Set<number>();
  return [defaultProfile, ...profiles].filter((profile) => {
    if (seen.has(profile.id)) return false;
    seen.add(profile.id);
    return true;
  });
};

const formatDuration = (durationMs: number): string => {
  const totalSeconds = Math.max(0, Math.round(durationMs / 1000));
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  return `${minutes}:${seconds.toString().padStart(2, '0')}`;
};

const formatPercent = (ratio: number): string => `${Math.round(Math.max(0, Math.min(1, ratio)) * 100)}%`;

const formatSigned = (value: number): string => value > 0 ? `+${value}` : String(value);

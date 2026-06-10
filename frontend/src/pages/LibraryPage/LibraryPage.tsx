import {
  ClearRounded as ClearRoundedIcon,
  DeleteRounded as DeleteRoundedIcon,
  DoneAllRounded as DoneAllRoundedIcon,
  DriveFolderUploadRounded as DriveFolderUploadRoundedIcon,
  FileUploadRounded as FileUploadRoundedIcon,
  LibraryAddCheckRounded as LibraryAddCheckRoundedIcon,
  MusicNoteRounded as MusicNoteRoundedIcon,
  PlaylistPlayRounded as PlaylistPlayRoundedIcon,
  RefreshRounded as RefreshRoundedIcon,
  RestartAltRounded as RestartAltRoundedIcon,
  SearchRounded as SearchRoundedIcon,
  SelectAllRounded as SelectAllRoundedIcon,
  ShuffleRounded as ShuffleRoundedIcon,
  SyncRounded as SyncRoundedIcon,
  TuneRounded as TuneRoundedIcon,
  GraphicEqRounded as GraphicEqRoundedIcon,
  KeyboardRounded as KeyboardRoundedIcon,
} from '@mui/icons-material';
import {
  Box,
  Button,
  IconButton,
  MenuItem,
  Select,
  TextField,
  Typography,
  useTheme,
} from '@mui/material';
import { useEffect, useRef, useState } from 'react';
import type { DragEvent, ReactNode } from 'react';
import { PerformPanel } from '@/components/PerformPanel';
import { PianoRollView } from '@/components/PianoRollView';
import { PreviewPanel } from '@/components/PreviewPanel';
import { useT } from '@/i18n';
import { useRouter } from '@/router';
import { EditorSelectionService, MidiService, NativeDialogs, type MidiProjectSummary } from '@/services';
import { libraryPageStyles } from './LibraryPage.styles';
import { useLibraryPage, type LibraryPanel, type LibrarySortValue, type PlaylistMode } from './useLibraryPage';
import { ProjectItem } from './components/ProjectItem';
import { PlaylistPanel } from './components/PlaylistPanel';
import { OverviewPanel } from './components/OverviewPanel';
import { SettingsPanel } from './components/SettingsPanel';
import { StatChip } from './shared/StatChip';
import { uniqueProfiles } from './utils/profile';
import { formatDuration, formatPercent, midiNoteToName } from './utils/format';

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

  useEffect(() => {
    const off = MidiService.onFilesDropped((paths) => {
      void vm.importDroppedPaths(paths).then(openImportedProject);
    });
    return off;
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [vm.importDroppedPaths, vm.autoOpenImportedProject]);

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
    if (dragDepthRef.current === 0) setDragActive(false);
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
    if (confirmed) await vm.deleteProject(project.id);
  };

  const confirmDeleteSelected = async (): Promise<void> => {
    if (vm.selectedCount === 0) return;
    const confirmed = await NativeDialogs.confirm({
      title: t('library.batch.deleteSelectedTitle'),
      message: t('library.batch.deleteSelectedMessage', { count: String(vm.selectedCount) }),
      okLabel: t('library.batch.deleteConfirm'),
      cancelLabel: t('library.batch.cancel'),
    });
    if (confirmed) await vm.deleteSelectedProjects();
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

  // 可演奏率颜色判断
  const playableColor = (): 'success' | 'warning' | 'error' | 'default' => {
    if (!selectedReport) return 'default';
    const ratio = selectedReport.playableRatio;
    if (ratio >= 0.8) return 'success';
    if (ratio >= 0.5) return 'warning';
    return 'error';
  };

  // 音域范围显示
  const noteRangeLabel = selectedReport && selectedReport.noteRange
    ? `${midiNoteToName(selectedReport.noteRange.min)} - ${midiNoteToName(selectedReport.noteRange.max)}`
    : '-';

  // 是否为搜索无结果状态
  const isSearchNoResult = !vm.loading && vm.query.trim().length > 0 && vm.filteredProjects.length === 0;
  // 是否为完全空库
  const isLibraryEmpty = !vm.loading && vm.query.trim().length === 0 && vm.projects.length === 0;

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
                  if (vm.selectionMode) { vm.exitSelectionMode(); return; }
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
                  onClick={() => { vm.selectAllFilteredProjects(); }}
                  disabled={vm.filteredProjects.length === 0 || vm.allFilteredSelected || vm.batchDeleting}
                >
                  {vm.allFilteredSelected ? t('library.batch.selectedAll') : t('library.batch.selectFiltered')}
                </Button>
                <Button
                  variant="outlined"
                  size="small"
                  startIcon={<ClearRoundedIcon />}
                  onClick={() => { vm.clearProjectSelection(); }}
                  disabled={vm.selectedCount === 0 || vm.batchDeleting}
                >
                  {t('library.batch.clearSelection')}
                </Button>
                <Button
                  variant="contained"
                  size="small"
                  sx={styles.dangerButton}
                  startIcon={vm.batchDeleting ? undefined : <DeleteRoundedIcon />}
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

          {isSearchNoResult ? (
            <Box sx={styles.emptyList}>
              <MusicNoteRoundedIcon fontSize="large" />
              <Typography sx={{ mb: 0.5 }}>{t('library.list.noSearchResultQuery', { query: vm.query })}</Typography>
              <Button variant="outlined" size="small" startIcon={<ClearRoundedIcon />} onClick={() => vm.setQuery('')}>
                {t('library.search.clear')}
              </Button>
            </Box>
          ) : isLibraryEmpty ? (
            <Box sx={styles.emptyList}>
              <MusicNoteRoundedIcon fontSize="large" />
              <Typography sx={{ mb: 0.5 }}>{t('library.list.empty')}</Typography>
              <Box sx={{ ...styles.toolButtons, mt: 1.5, justifyContent: 'center' }}>
                <Button variant="contained" startIcon={<FileUploadRoundedIcon />} onClick={() => void importMidiFiles()} disabled={vm.importing}>
                  {t('library.actions.importFiles')}
                </Button>
              </Box>
              <Typography sx={{ mt: 1, fontSize: 12, color: 'text.secondary' }}>
                {t('library.list.dragHint')}
              </Typography>
            </Box>
          ) : vm.loading && vm.projects.length === 0 ? (
            <Box sx={styles.emptyList}>
              <Typography>{t('library.list.loading')}</Typography>
            </Box>
          ) : vm.filteredProjects.length > 0 ? (
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
                  onToggleSelect={() => { vm.toggleProjectSelection(project.id); }}
                  onPlayNext={() => vm.addProjectNextInPlaylist(project)}
                  onAddToPlaylist={() => vm.addProjectToPlaylist(project)}
                  onDelete={() => void confirmDeleteProject(project)}
                />
              ))}
            </Box>
          ) : (
            <Box sx={styles.emptyList}>
              <MusicNoteRoundedIcon fontSize="large" />
              <Typography>{t('library.list.empty')}</Typography>
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
                      <StatChip icon={<TuneRoundedIcon fontSize="small" />} label={t('library.stats.range')} value={noteRangeLabel} />
                      <StatChip icon={<TuneRoundedIcon fontSize="small" />} label={t('library.stats.playable')} value={selectedReport ? formatPercent(selectedReport.playableRatio) : '-'} color={playableColor()} />
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

import DeleteRoundedIcon from '@mui/icons-material/DeleteRounded';
import DriveFolderUploadRoundedIcon from '@mui/icons-material/DriveFolderUploadRounded';
import FileUploadRoundedIcon from '@mui/icons-material/FileUploadRounded';
import GraphicEqRoundedIcon from '@mui/icons-material/GraphicEqRounded';
import KeyboardRoundedIcon from '@mui/icons-material/KeyboardRounded';
import MusicNoteRoundedIcon from '@mui/icons-material/MusicNoteRounded';
import RefreshRoundedIcon from '@mui/icons-material/RefreshRounded';
import RestartAltRoundedIcon from '@mui/icons-material/RestartAltRounded';
import SaveRoundedIcon from '@mui/icons-material/SaveRounded';
import SearchRoundedIcon from '@mui/icons-material/SearchRounded';
import SpeedRoundedIcon from '@mui/icons-material/SpeedRounded';
import TuneRoundedIcon from '@mui/icons-material/TuneRounded';
import {
  Box,
  Button,
  Checkbox,
  IconButton,
  MenuItem,
  Select,
  TextField,
  Typography,
  useTheme,
} from '@mui/material';
import type { ReactNode } from 'react';
import type { SxProps, Theme } from '@mui/material/styles';
import { PerformPanel } from '@/components/PerformPanel';
import { PianoRollView } from '@/components/PianoRollView';
import { PreviewPanel } from '@/components/PreviewPanel';
import { QualityReportPanel } from '@/components/QualityReportPanel';
import { useT } from '@/i18n';
import { useRouter } from '@/router';
import { EditorSelectionService, type MidiProfile, type MidiProjectSummary, type OutOfRangePolicy, type QualityReport } from '@/services';
import { libraryPageStyles } from './LibraryPage.styles';
import { useLibraryPage, type LibraryPanel, type LibraryProfileForm } from './useLibraryPage';

export const LibraryPage = () => {
  const theme = useTheme();
  const styles = libraryPageStyles(theme);
  const t = useT();
  const router = useRouter();
  const vm = useLibraryPage();
  const selectedProject = vm.selected?.project ?? null;
  const selectedReport = vm.previewPlan?.report ?? vm.selected?.qualityReport ?? null;
  const profiles = vm.selected ? uniqueProfiles(vm.selected.defaultProfile, vm.selected.profiles) : [];

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

  const panelItems: Array<{ id: LibraryPanel; label: string; description: string }> = [
    { id: 'perform', label: t('library.panels.perform.label'), description: t('library.panels.perform.description') },
    { id: 'overview', label: t('library.panels.overview.label'), description: t('library.panels.overview.description') },
    { id: 'settings', label: t('library.panels.settings.label'), description: t('library.panels.settings.description') },
    { id: 'preview', label: t('library.panels.preview.label'), description: t('library.panels.preview.description') },
  ];

  return (
    <Box sx={styles.root}>
      <Box sx={styles.workspace}>
        <Box sx={styles.libraryColumn}>
          <Box sx={styles.brandBlock}>
            <Typography sx={styles.eyebrow}>{t('library.eyebrow')}</Typography>
            <Typography component="h1" sx={styles.title}>{t('library.title')}</Typography>
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
            <Typography sx={styles.pill}>{t('library.list.count', { count: String(vm.projects.length) })}</Typography>
          </Box>

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
                  onSelect={() => void vm.selectProject(project.id)}
                  onDelete={() => void vm.deleteProject(project.id)}
                />
              ))}
            </Box>
          )}
        </Box>

        <Box sx={styles.detailColumn}>
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
        </Box>
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
  onSelect: () => void;
  onDelete: () => void;
}

const ProjectItem = ({ index, project, report, active, deleting, onSelect, onDelete }: ProjectItemProps) => {
  const theme = useTheme();
  const styles = libraryPageStyles(theme);
  const t = useT();
  const itemSx = (active ? [styles.trackItem, styles.trackItemActive] : styles.trackItem) as SxProps<Theme>;

  const onDeleteClick = (event: React.MouseEvent<HTMLButtonElement>) => {
    event.stopPropagation();
    onDelete();
  };

  return (
    <Box component="button" type="button" sx={itemSx} onClick={onSelect}>
      <Box sx={styles.trackIndex}>{index.toString().padStart(2, '0')}</Box>
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
        <IconButton aria-label={t('library.actions.delete')} size="small" onClick={onDeleteClick} disabled={deleting}>
          <DeleteRoundedIcon fontSize="small" />
        </IconButton>
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

import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { DEFAULT_PREFERENCES, usePreferences } from '@/preferences';
import {
  MidiService,
  NativeDialogs,
  type ImportBatchResult,
  type MidiConfigSnapshot,
  type MidiProjectDetail,
  type MidiProjectSummary,
  type MidiProfile,
  type PlayPlan,
} from '@/services';

export type LibraryPanel = 'overview' | 'settings' | 'preview' | 'perform';

export interface LibraryProfileForm {
  id: number;
  projectId?: number;
  name: string;
  baseNote: number;
  transpose: number;
  octaveShift: number;
  speed: number;
  outOfRangePolicy: MidiConfigSnapshot['outOfRangePolicy'];
  minPressMs: number;
  releaseGapMs: number;
  keymapProfileId: number;
  enabledTracks: number[] | null;
}

export interface UseLibraryPageResult {
  projects: MidiProjectSummary[];
  filteredProjects: MidiProjectSummary[];
  selected: MidiProjectDetail | null;
  previewPlan: PlayPlan | null;
  form: LibraryProfileForm;
  query: string;
  activePanel: LibraryPanel;
  selectedProfileId: number;
  loading: boolean;
  detailLoading: boolean;
  importing: boolean;
  deletingId: number | null;
  saving: boolean;
  previewLoading: boolean;
  error: string | null;
  importError: string | null;
  importSummary: ImportBatchResult | null;
  saveError: string | null;
  previewError: string | null;
  isDirty: boolean;
  autoOpenImportedProject: boolean;
  setQuery: (query: string) => void;
  setActivePanel: (panel: LibraryPanel) => void;
  refresh: () => Promise<void>;
  importMidiFiles: (dialogTitle: string, filterName: string) => Promise<number | null>;
  importMidiDirectory: (dialogTitle: string) => Promise<number | null>;
  selectProject: (projectId: number) => Promise<void>;
  selectProfile: (profileId: number) => void;
  updateField: <K extends keyof LibraryProfileForm>(field: K, value: LibraryProfileForm[K]) => void;
  resetForm: () => void;
  refreshPreview: () => Promise<void>;
  save: () => Promise<void>;
  deleteProject: (projectId: number) => Promise<void>;
}

const errorMessage = (e: unknown): string => e instanceof Error ? e.message : String(e ?? '');

const DEFAULT_FORM: LibraryProfileForm = {
  id: 0,
  projectId: undefined,
  name: '',
  baseNote: 48,
  transpose: 0,
  octaveShift: 0,
  speed: 1,
  outOfRangePolicy: 'drop',
  minPressMs: 35,
  releaseGapMs: 15,
  keymapProfileId: 1,
  enabledTracks: null,
};

const profileToForm = (profile: MidiProfile, projectId?: number): LibraryProfileForm => ({
  id: profile.id,
  projectId: profile.projectId ?? projectId,
  name: profile.name,
  baseNote: profile.baseNote,
  transpose: profile.transpose,
  octaveShift: profile.octaveShift,
  speed: profile.speed,
  outOfRangePolicy: profile.outOfRangePolicy,
  minPressMs: profile.minPressMs,
  releaseGapMs: profile.releaseGapMs,
  keymapProfileId: profile.keymapProfileId,
  enabledTracks: profile.enabledTracks === null ? null : [...profile.enabledTracks],
});

const isSameTracks = (a: number[] | null, b: number[] | null): boolean => {
  if (a === null || b === null) return a === b;
  if (a.length !== b.length) return false;
  return a.every((value, index) => value === b[index]);
};

const isSameForm = (a: LibraryProfileForm, b: LibraryProfileForm): boolean => (
  a.id === b.id
  && a.name === b.name
  && a.baseNote === b.baseNote
  && a.transpose === b.transpose
  && a.octaveShift === b.octaveShift
  && a.speed === b.speed
  && a.outOfRangePolicy === b.outOfRangePolicy
  && a.minPressMs === b.minPressMs
  && a.releaseGapMs === b.releaseGapMs
  && a.keymapProfileId === b.keymapProfileId
  && isSameTracks(a.enabledTracks, b.enabledTracks)
);

const clampListLimit = (value: number): number => {
  if (!Number.isFinite(value)) return DEFAULT_PREFERENCES.libraryListLimit;
  return Math.max(5, Math.min(10000, Math.round(value)));
};

const pickProfile = (detail: MidiProjectDetail, preferredProfileId?: number): MidiProfile => {
  if (preferredProfileId) {
    const preferred = detail.profiles.find((p) => p.id === preferredProfileId);
    if (preferred) return preferred;
    if (detail.defaultProfile.id === preferredProfileId) return detail.defaultProfile;
  }

  const defaultProfile = detail.profiles.find((p) => p.id === detail.project.defaultProfileId);
  return defaultProfile ?? detail.defaultProfile;
};

export const useLibraryPage = (): UseLibraryPageResult => {
  const { preferences } = usePreferences();
  const listLimit = clampListLimit(preferences.libraryListLimit);
  const aliveRef = useRef(true);
  const selectedProjectIdRef = useRef<number | null>(null);
  const [projects, setProjects] = useState<MidiProjectSummary[]>([]);
  const [selected, setSelected] = useState<MidiProjectDetail | null>(null);
  const [previewPlan, setPreviewPlan] = useState<PlayPlan | null>(null);
  const [form, setForm] = useState<LibraryProfileForm>(DEFAULT_FORM);
  const [query, setQuery] = useState('');
  const [activePanel, setActivePanel] = useState<LibraryPanel>('perform');
  const [selectedProfileId, setSelectedProfileId] = useState(0);
  const [loading, setLoading] = useState(false);
  const [detailLoading, setDetailLoading] = useState(false);
  const [importing, setImporting] = useState(false);
  const [deletingId, setDeletingId] = useState<number | null>(null);
  const [saving, setSaving] = useState(false);
  const [previewLoading, setPreviewLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [importError, setImportError] = useState<string | null>(null);
  const [importSummary, setImportSummary] = useState<ImportBatchResult | null>(null);
  const [saveError, setSaveError] = useState<string | null>(null);
  const [previewError, setPreviewError] = useState<string | null>(null);

  useEffect(() => {
    aliveRef.current = true;
    return () => {
      aliveRef.current = false;
    };
  }, []);

  const clearSelection = useCallback((): void => {
    selectedProjectIdRef.current = null;
    setSelected(null);
    setPreviewPlan(null);
    setForm(DEFAULT_FORM);
    setActivePanel('perform');
    setSelectedProfileId(0);
    setSaveError(null);
    setPreviewError(null);
  }, []);

  const applyDetail = useCallback((detail: MidiProjectDetail, preferredProfileId?: number): MidiProfile => {
    const nextProfile = pickProfile(detail, preferredProfileId);
    selectedProjectIdRef.current = detail.project.id;
    setSelected(detail);
    setActivePanel('perform');
    setSelectedProfileId(nextProfile.id);
    setForm(profileToForm(nextProfile, detail.project.id));
    setSaveError(null);
    return nextProfile;
  }, []);

  const filteredProjects = useMemo(() => {
    const needle = query.trim().toLocaleLowerCase();
    if (!needle) return projects;
    return projects.filter((project) => (
      project.displayName.toLocaleLowerCase().includes(needle)
      || project.fileName.toLocaleLowerCase().includes(needle)
      || project.fileHash.toLocaleLowerCase().includes(needle)
    ));
  }, [projects, query]);

  const currentProfile = useMemo(() => {
    if (!selected) return null;
    return selected.profiles.find((p) => p.id === selectedProfileId)
      ?? (selected.defaultProfile.id === selectedProfileId ? selected.defaultProfile : selected.defaultProfile);
  }, [selected, selectedProfileId]);

  const loadPreviewPlan = useCallback(async (projectId: number, profileId: number): Promise<void> => {
    setPreviewLoading(true);
    setPreviewError(null);
    try {
      const plan = await MidiService.buildPlayPlan(projectId, profileId);
      if (!aliveRef.current) return;
      setPreviewPlan(plan);
    } catch (e: unknown) {
      if (!aliveRef.current) return;
      setPreviewPlan(null);
      setPreviewError(errorMessage(e));
    } finally {
      if (aliveRef.current) setPreviewLoading(false);
    }
  }, []);

  const refresh = useCallback(async (): Promise<void> => {
    setLoading(true);
    setError(null);
    try {
      const rows = await MidiService.listProjects({ limit: listLimit });
      if (!aliveRef.current) return;
      setProjects(rows);
      const selectedProjectId = selectedProjectIdRef.current;
      if (selectedProjectId && rows.every((p) => p.id !== selectedProjectId)) {
        clearSelection();
      }
    } catch (e: unknown) {
      if (!aliveRef.current) return;
      setError(errorMessage(e));
    } finally {
      if (aliveRef.current) setLoading(false);
    }
  }, [clearSelection, listLimit]);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  const selectProject = useCallback(async (projectId: number): Promise<void> => {
    if (projectId <= 0) return;
    setDetailLoading(true);
    setError(null);
    try {
      const detail = await MidiService.getProject(projectId);
      if (!aliveRef.current) return;
      const nextProfile = applyDetail(detail);
      await loadPreviewPlan(detail.project.id, nextProfile.id);
    } catch (e: unknown) {
      if (!aliveRef.current) return;
      setError(errorMessage(e));
    } finally {
      if (aliveRef.current) setDetailLoading(false);
    }
  }, [applyDetail, loadPreviewPlan]);

  const applyImportResult = useCallback(async (result: ImportBatchResult): Promise<number | null> => {
    setImportSummary(result);
    const projectId = result.firstImportedProjectId ?? result.firstProjectId ?? null;
    const rows = await MidiService.listProjects({ limit: listLimit });
    if (!aliveRef.current) return null;
    setProjects(rows);
    if (!projectId) return null;
    const detail = await MidiService.getProject(projectId);
    if (!aliveRef.current) return null;
    const nextProfile = applyDetail(detail);
    await loadPreviewPlan(detail.project.id, nextProfile.id);
    return projectId;
  }, [applyDetail, listLimit, loadPreviewPlan]);

  const importMidiFiles = useCallback(async (dialogTitle: string, filterName: string): Promise<number | null> => {
    setImportError(null);
    setImportSummary(null);
    const paths = await NativeDialogs.openFiles({
      title: dialogTitle,
      filters: [{ displayName: filterName, pattern: '*.mid;*.midi' }],
    });
    if (paths.length === 0) return null;
    setImporting(true);
    try {
      const result = await MidiService.importFiles(paths);
      if (!aliveRef.current) return null;
      return await applyImportResult(result);
    } catch (e: unknown) {
      if (!aliveRef.current) return null;
      setImportError(errorMessage(e));
      return null;
    } finally {
      if (aliveRef.current) setImporting(false);
    }
  }, [applyImportResult]);

  const importMidiDirectory = useCallback(async (dialogTitle: string): Promise<number | null> => {
    setImportError(null);
    setImportSummary(null);
    const path = await NativeDialogs.openDirectory({ title: dialogTitle });
    if (!path) return null;
    setImporting(true);
    try {
      const result = await MidiService.importDirectory(path);
      if (!aliveRef.current) return null;
      return await applyImportResult(result);
    } catch (e: unknown) {
      if (!aliveRef.current) return null;
      setImportError(errorMessage(e));
      return null;
    } finally {
      if (aliveRef.current) setImporting(false);
    }
  }, [applyImportResult]);

  const selectProfile = useCallback((profileId: number): void => {
    if (!selected) return;
    const nextProfile = pickProfile(selected, profileId);
    setSelectedProfileId(nextProfile.id);
    setForm(profileToForm(nextProfile, selected.project.id));
    setSaveError(null);
    void loadPreviewPlan(selected.project.id, nextProfile.id);
  }, [loadPreviewPlan, selected]);

  const updateField = useCallback(<K extends keyof LibraryProfileForm>(field: K, value: LibraryProfileForm[K]) => {
    setForm((prev) => ({ ...prev, [field]: value }));
    setSaveError(null);
  }, []);

  const resetForm = useCallback((): void => {
    if (!currentProfile) return;
    setForm(profileToForm(currentProfile, selected?.project.id));
    setSaveError(null);
  }, [currentProfile, selected?.project.id]);

  const refreshPreview = useCallback(async (): Promise<void> => {
    const projectId = selected?.project.id;
    if (!projectId || selectedProfileId <= 0) return;
    await loadPreviewPlan(projectId, selectedProfileId);
  }, [loadPreviewPlan, selected?.project.id, selectedProfileId]);

  const save = useCallback(async (): Promise<void> => {
    if (!selected) return;
    setSaving(true);
    setSaveError(null);
    try {
      const payload: MidiProfile = {
        id: form.id,
        projectId: form.projectId ?? selected.project.id,
        name: form.name,
        baseNote: form.baseNote,
        transpose: form.transpose,
        octaveShift: form.octaveShift,
        speed: form.speed,
        outOfRangePolicy: form.outOfRangePolicy,
        minPressMs: form.minPressMs,
        releaseGapMs: form.releaseGapMs,
        keymapProfileId: form.keymapProfileId,
        enabledTracks: form.enabledTracks === null ? null : [...form.enabledTracks],
        createdAt: 0,
        updatedAt: 0,
      };
      const saved = await MidiService.updateProfile(payload);
      if (!aliveRef.current) return;
      const detail = await MidiService.getProject(saved.projectId ?? selected.project.id);
      if (!aliveRef.current) return;
      applyDetail(detail, saved.id);
      await loadPreviewPlan(detail.project.id, saved.id);
    } catch (e: unknown) {
      if (!aliveRef.current) return;
      setSaveError(errorMessage(e));
    } finally {
      if (aliveRef.current) setSaving(false);
    }
  }, [applyDetail, form, loadPreviewPlan, selected]);

  const deleteProject = useCallback(async (projectId: number): Promise<void> => {
    setDeletingId(projectId);
    setError(null);
    try {
      await MidiService.deleteProject(projectId);
      if (!aliveRef.current) return;
      setProjects((prev) => prev.filter((p) => p.id !== projectId));
      if (selectedProjectIdRef.current === projectId) {
        clearSelection();
      }
    } catch (e: unknown) {
      if (!aliveRef.current) return;
      setError(errorMessage(e));
    } finally {
      if (aliveRef.current) setDeletingId(null);
    }
  }, [clearSelection]);

  return {
    projects,
    filteredProjects,
    selected,
    previewPlan,
    form,
    query,
    activePanel,
    selectedProfileId,
    loading,
    detailLoading,
    importing,
    deletingId,
    saving,
    previewLoading,
    error,
    importError,
    importSummary,
    saveError,
    previewError,
    isDirty: currentProfile ? !isSameForm(form, profileToForm(currentProfile, selected?.project.id)) : false,
    autoOpenImportedProject: preferences.libraryAutoOpenImportedProject,
    setQuery,
    setActivePanel,
    refresh,
    importMidiFiles,
    importMidiDirectory,
    selectProject,
    selectProfile,
    updateField,
    resetForm,
    refreshPreview,
    save,
    deleteProject,
  };
};

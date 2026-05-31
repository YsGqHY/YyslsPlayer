import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { DEFAULT_PREFERENCES, usePreferences } from '@/preferences';
import { registerHotkeyHandler } from '@/shared/hotkeys';
import {
  MidiService,
  NativeDialogs,
  type ImportBatchResult,
  type MidiConfigSnapshot,
  type MidiProjectDetail,
  type MidiProjectSummary,
  type MidiProfile,
  type PlayerStateSnapshot,
  type PlayPlan,
  type ProjectBatchManageResult,
} from '@/services';

export type LibraryPanel = 'overview' | 'settings' | 'preview' | 'perform';
export type LibrarySortKey = 'createdAt' | 'fileName' | 'durationMs' | 'fileSizeBytes';
export type LibrarySortDirection = 'asc' | 'desc';
export type LibrarySortValue = `${LibrarySortKey}:${LibrarySortDirection}`;
export type PlaylistMode = 'shuffle' | 'sequence' | 'loop' | 'singleLoop';

export interface PlaylistItem {
  project: MidiProjectSummary;
  addedAt: number;
}

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
  sortValue: LibrarySortValue;
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
  batchResult: ProjectBatchManageResult | null;
  saveError: string | null;
  previewError: string | null;
  isDirty: boolean;
  selectionMode: boolean;
  selectedProjectIds: number[];
  selectedCount: number;
  allFilteredSelected: boolean;
  someFilteredSelected: boolean;
  batchDeleting: boolean;
  playlistOpen: boolean;
  playlistItems: PlaylistItem[];
  playlistMode: PlaylistMode;
  playlistCurrentIndex: number;
  playlistCurrentProjectId: number | null;
  playlistDraggingIndex: number | null;
  playlistAutoStartToken: number;
  autoOpenImportedProject: boolean;
  setQuery: (query: string) => void;
  setSortValue: (sortValue: LibrarySortValue) => void;
  setActivePanel: (panel: LibraryPanel) => void;
  refresh: () => Promise<void>;
  importMidiFiles: (dialogTitle: string, filterName: string) => Promise<number | null>;
  importMidiDirectory: (dialogTitle: string) => Promise<number | null>;
  importDroppedPaths: (paths: string[]) => Promise<number | null>;
  selectProject: (projectId: number) => Promise<void>;
  selectProfile: (profileId: number) => void;
  updateField: <K extends keyof LibraryProfileForm>(field: K, value: LibraryProfileForm[K]) => void;
  resetForm: () => void;
  refreshPreview: () => Promise<void>;
  save: () => Promise<void>;
  deleteProject: (projectId: number) => Promise<void>;
  togglePlaylist: () => void;
  setPlaylistOpen: (open: boolean) => void;
  setPlaylistMode: (mode: PlaylistMode) => void;
  addProjectToPlaylist: (project: MidiProjectSummary) => void;
  addProjectNextInPlaylist: (project: MidiProjectSummary) => void;
  addSelectedProjectToPlaylist: () => void;
  addFilteredProjectsToPlaylist: () => void;
  removePlaylistItem: (projectId: number) => void;
  clearPlaylist: () => void;
  selectPlaylistItem: (index: number) => Promise<void>;
  playPlaylistItem: (index: number) => Promise<void>;
  movePlaylistItem: (fromIndex: number, toIndex: number) => void;
  startPlaylistDrag: (index: number) => void;
  finishPlaylistDrag: () => void;
  handlePlayerState: (snapshot: PlayerStateSnapshot) => void;
  markPlaylistStarted: () => void;
  enterSelectionMode: () => void;
  exitSelectionMode: () => void;
  toggleProjectSelection: (projectId: number) => void;
  selectAllFilteredProjects: () => void;
  clearProjectSelection: () => void;
  deleteSelectedProjects: () => Promise<ProjectBatchManageResult | null>;
}

const errorMessage = (e: unknown): string => e instanceof Error ? e.message : String(e ?? '');

const DEFAULT_SORT_VALUE: LibrarySortValue = 'createdAt:desc';
const DEFAULT_PLAYLIST_MODE: PlaylistMode = 'sequence';

const fileNameCollator = new Intl.Collator(undefined, { numeric: true, sensitivity: 'base' });

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

const parseSortValue = (sortValue: LibrarySortValue): { key: LibrarySortKey; direction: LibrarySortDirection } => {
  const [key, direction] = sortValue.split(':') as [LibrarySortKey, LibrarySortDirection];
  return { key, direction };
};

const sortProjects = (rows: MidiProjectSummary[], sortValue: LibrarySortValue): MidiProjectSummary[] => {
  const { key, direction } = parseSortValue(sortValue);
  const factor = direction === 'asc' ? 1 : -1;
  return [...rows].sort((a, b) => {
    let result = 0;
    if (key === 'fileName') {
      result = fileNameCollator.compare(a.fileName || a.displayName, b.fileName || b.displayName);
    } else {
      result = Number(a[key] ?? 0) - Number(b[key] ?? 0);
    }
    if (result === 0) {
      result = Number(a.id) - Number(b.id);
    }
    return result * factor;
  });
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
  const [sortValue, setSortValue] = useState<LibrarySortValue>(DEFAULT_SORT_VALUE);
  const [activePanel, setActivePanel] = useState<LibraryPanel>('perform');
  const [selectedProfileId, setSelectedProfileId] = useState(0);
  const [loading, setLoading] = useState(false);
  const [detailLoading, setDetailLoading] = useState(false);
  const [importing, setImporting] = useState(false);
  const [deletingId, setDeletingId] = useState<number | null>(null);
  const [batchDeleting, setBatchDeleting] = useState(false);
  const [saving, setSaving] = useState(false);
  const [previewLoading, setPreviewLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [importError, setImportError] = useState<string | null>(null);
  const [importSummary, setImportSummary] = useState<ImportBatchResult | null>(null);
  const [batchResult, setBatchResult] = useState<ProjectBatchManageResult | null>(null);
  const [selectionMode, setSelectionMode] = useState(false);
  const [selectedProjectIds, setSelectedProjectIds] = useState<number[]>([]);
  const [playlistOpen, setPlaylistOpen] = useState(false);
  const [playlistItems, setPlaylistItems] = useState<PlaylistItem[]>([]);
  const [playlistMode, setPlaylistMode] = useState<PlaylistMode>(DEFAULT_PLAYLIST_MODE);
  const [playlistCurrentIndex, setPlaylistCurrentIndex] = useState(-1);
  const [playlistDraggingIndex, setPlaylistDraggingIndex] = useState<number | null>(null);
  const [playlistAutoStartToken, setPlaylistAutoStartToken] = useState(0);
  const playlistStartedRef = useRef(false);
  const playlistAdvanceRef = useRef(false);
  const playlistCompletedSessionRef = useRef('');
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
    const rows = needle
      ? projects.filter((project) => (
        project.displayName.toLocaleLowerCase().includes(needle)
        || project.fileName.toLocaleLowerCase().includes(needle)
        || project.fileHash.toLocaleLowerCase().includes(needle)
      ))
      : projects;
    return sortProjects(rows, sortValue);
  }, [projects, query, sortValue]);

  const selectedIdSet = useMemo(() => new Set(selectedProjectIds), [selectedProjectIds]);
  const filteredProjectIds = useMemo(() => filteredProjects.map((project) => project.id), [filteredProjects]);
  const selectedCount = selectedProjectIds.length;
  const allFilteredSelected = filteredProjectIds.length > 0 && filteredProjectIds.every((projectId) => selectedIdSet.has(projectId));
  const someFilteredSelected = filteredProjectIds.some((projectId) => selectedIdSet.has(projectId));
  const playlistCurrentProjectId = playlistCurrentIndex >= 0 ? playlistItems[playlistCurrentIndex]?.project.id ?? null : null;

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

  useEffect(() => {
    if (selectedProjectIds.length === 0) return;
    const existingIds = new Set(projects.map((project) => project.id));
    setSelectedProjectIds((prev) => prev.filter((projectId) => existingIds.has(projectId)));
  }, [projects, selectedProjectIds.length]);

  useEffect(() => {
    if (playlistItems.length === 0) {
      setPlaylistCurrentIndex(-1);
      return;
    }
    setPlaylistCurrentIndex((prev) => (prev >= playlistItems.length ? playlistItems.length - 1 : prev));
  }, [playlistItems.length]);

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

  const selectRelativeProject = useCallback((direction: -1 | 1): void => {
    if (filteredProjects.length === 0) return;
    const currentID = selectedProjectIdRef.current;
    const currentIndex = currentID ? filteredProjects.findIndex((project) => project.id === currentID) : -1;
    const nextIndex = currentIndex < 0
      ? (direction > 0 ? 0 : filteredProjects.length - 1)
      : (currentIndex + direction + filteredProjects.length) % filteredProjects.length;
    const nextProject = filteredProjects[nextIndex];
    if (nextProject) {
      void selectProject(nextProject.id);
    }
  }, [filteredProjects, selectProject]);

  useEffect(() => {
    const offPrevious = registerHotkeyHandler('previous-track', () => selectRelativeProject(-1));
    const offNext = registerHotkeyHandler('next-track', () => selectRelativeProject(1));
    return () => {
      offPrevious();
      offNext();
    };
  }, [selectRelativeProject]);

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

  const importDroppedPaths = useCallback(async (paths: string[]): Promise<number | null> => {
    if (paths.length === 0) return null;
    setImportError(null);
    setImportSummary(null);
    setImporting(true);
    try {
      const result = await MidiService.importPaths(paths);
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

  const togglePlaylist = useCallback((): void => {
    setPlaylistOpen((prev) => !prev);
  }, []);

  const addProjectToPlaylist = useCallback((project: MidiProjectSummary): void => {
    setPlaylistOpen(true);
    setPlaylistItems((prev) => {
      if (prev.some((item) => item.project.id === project.id)) return prev;
      return [...prev, { project, addedAt: Date.now() }];
    });
  }, []);

  const addProjectNextInPlaylist = useCallback((project: MidiProjectSummary): void => {
    setPlaylistOpen(true);
    setPlaylistItems((prev) => {
      const existingIndex = prev.findIndex((item) => item.project.id === project.id);
      const withoutProject = existingIndex >= 0 ? prev.filter((item) => item.project.id !== project.id) : prev;
      const currentProjectId = playlistCurrentIndex >= 0 ? prev[playlistCurrentIndex]?.project.id : undefined;
      const currentIndexAfterRemoval = currentProjectId ? withoutProject.findIndex((item) => item.project.id === currentProjectId) : -1;
      const insertIndex = Math.min(withoutProject.length, Math.max(0, currentIndexAfterRemoval + 1));
      const next = [...withoutProject];
      next.splice(insertIndex, 0, { project, addedAt: Date.now() });
      setPlaylistCurrentIndex(() => {
        if (!currentProjectId) return insertIndex;
        const nextCurrentIndex = next.findIndex((item) => item.project.id === currentProjectId);
        return nextCurrentIndex >= 0 ? nextCurrentIndex : insertIndex;
      });
      return next;
    });
  }, [playlistCurrentIndex]);

  const addSelectedProjectToPlaylist = useCallback((): void => {
    if (!selected?.project) return;
    addProjectToPlaylist(selected.project);
  }, [addProjectToPlaylist, selected?.project]);

  const addFilteredProjectsToPlaylist = useCallback((): void => {
    setPlaylistOpen(true);
    setPlaylistItems((prev) => {
      const existing = new Set(prev.map((item) => item.project.id));
      const next = [...prev];
      for (const project of filteredProjects) {
        if (existing.has(project.id)) continue;
        existing.add(project.id);
        next.push({ project, addedAt: Date.now() });
      }
      return next;
    });
  }, [filteredProjects]);

  const removePlaylistItem = useCallback((projectId: number): void => {
    setPlaylistItems((prev) => {
      const removeIndex = prev.findIndex((item) => item.project.id === projectId);
      if (removeIndex < 0) return prev;
      setPlaylistCurrentIndex((current) => {
        if (current < 0) return current;
        if (prev.length <= 1) return -1;
        if (current === removeIndex) return Math.min(current, prev.length - 2);
        if (current > removeIndex) return current - 1;
        return current;
      });
      return prev.filter((item) => item.project.id !== projectId);
    });
  }, []);

  const clearPlaylist = useCallback((): void => {
    setPlaylistItems([]);
    setPlaylistCurrentIndex(-1);
    playlistStartedRef.current = false;
    playlistAdvanceRef.current = false;
    playlistCompletedSessionRef.current = '';
  }, []);

  const selectPlaylistItem = useCallback(async (index: number): Promise<void> => {
    const item = playlistItems[index];
    if (!item) return;
    setPlaylistOpen(true);
    setPlaylistCurrentIndex(index);
    await selectProject(item.project.id);
  }, [playlistItems, selectProject]);

  const playPlaylistItem = useCallback(async (index: number): Promise<void> => {
    await selectPlaylistItem(index);
    if (!playlistItems[index]) return;
    playlistAdvanceRef.current = true;
    setPlaylistAutoStartToken((prev) => prev + 1);
  }, [playlistItems, selectPlaylistItem]);

  const movePlaylistItem = useCallback((fromIndex: number, toIndex: number): void => {
    setPlaylistItems((prev) => {
      if (fromIndex < 0 || toIndex < 0 || fromIndex >= prev.length || toIndex >= prev.length || fromIndex === toIndex) return prev;
      const next = [...prev];
      const [moved] = next.splice(fromIndex, 1);
      if (!moved) return prev;
      next.splice(toIndex, 0, moved);
      setPlaylistDraggingIndex((current) => current === fromIndex ? toIndex : current);
      setPlaylistCurrentIndex((current) => {
        if (current === fromIndex) return toIndex;
        if (fromIndex < current && current <= toIndex) return current - 1;
        if (toIndex <= current && current < fromIndex) return current + 1;
        return current;
      });
      return next;
    });
  }, []);

  const startPlaylistDrag = useCallback((index: number): void => {
    setPlaylistDraggingIndex(index);
  }, []);

  const finishPlaylistDrag = useCallback((): void => {
    setPlaylistDraggingIndex(null);
  }, []);

  const nextPlaylistIndex = useCallback((): number => {
    if (playlistItems.length === 0 || playlistCurrentIndex < 0) return -1;
    if (playlistMode === 'singleLoop') return playlistCurrentIndex;
    if (playlistMode === 'shuffle') {
      if (playlistItems.length === 1) return playlistCurrentIndex;
      const candidates = playlistItems.map((_, index) => index).filter((index) => index !== playlistCurrentIndex);
      return candidates[Math.floor(Math.random() * candidates.length)] ?? -1;
    }
    const next = playlistCurrentIndex + 1;
    if (next < playlistItems.length) return next;
    return playlistMode === 'loop' ? 0 : -1;
  }, [playlistCurrentIndex, playlistItems, playlistMode]);

  const handlePlayerState = useCallback((snapshot: PlayerStateSnapshot): void => {
    if (snapshot.state !== 'completed' || !playlistStartedRef.current || !playlistAdvanceRef.current) return;
    if (snapshot.sessionId && playlistCompletedSessionRef.current === snapshot.sessionId) return;
    playlistCompletedSessionRef.current = snapshot.sessionId;
    const nextIndex = nextPlaylistIndex();
    if (nextIndex < 0) {
      playlistAdvanceRef.current = false;
      return;
    }
    void playPlaylistItem(nextIndex);
  }, [nextPlaylistIndex, playPlaylistItem]);

  const markPlaylistStarted = useCallback((): void => {
    playlistStartedRef.current = true;
    playlistCompletedSessionRef.current = '';
    if (playlistCurrentIndex >= 0) {
      playlistAdvanceRef.current = true;
    }
  }, [playlistCurrentIndex]);

  const applyDeletedProjectIds = useCallback((deletedIds: number[]): void => {
    if (deletedIds.length === 0) return;
    const deleted = new Set(deletedIds);
    setProjects((prev) => prev.filter((p) => !deleted.has(p.id)));
    setSelectedProjectIds((prev) => prev.filter((projectId) => !deleted.has(projectId)));
    setPlaylistItems((prev) => {
      const currentProjectId = playlistCurrentIndex >= 0 ? prev[playlistCurrentIndex]?.project.id : undefined;
      const remaining = prev.filter((item) => !deleted.has(item.project.id));
      setPlaylistCurrentIndex((current) => {
        if (remaining.length === 0) return -1;
        if (currentProjectId && !deleted.has(currentProjectId)) {
          const nextIndex = remaining.findIndex((item) => item.project.id === currentProjectId);
          return nextIndex >= 0 ? nextIndex : Math.min(current, remaining.length - 1);
        }
        return Math.min(current, remaining.length - 1);
      });
      return remaining;
    });
    if (selectedProjectIdRef.current && deleted.has(selectedProjectIdRef.current)) {
      clearSelection();
    }
  }, [clearSelection, playlistCurrentIndex]);

  const deleteProject = useCallback(async (projectId: number): Promise<void> => {
    setDeletingId(projectId);
    setError(null);
    setBatchResult(null);
    try {
      await MidiService.deleteProject(projectId);
      if (!aliveRef.current) return;
      applyDeletedProjectIds([projectId]);
      setBatchResult({
        totalCount: 1,
        deletedCount: 1,
        failedCount: 0,
        items: [{ projectId, displayName: '', status: 'deleted', reason: '', error: '' }],
      });
    } catch (e: unknown) {
      if (!aliveRef.current) return;
      setError(errorMessage(e));
    } finally {
      if (aliveRef.current) setDeletingId(null);
    }
  }, [applyDeletedProjectIds]);

  const enterSelectionMode = useCallback((): void => {
    setSelectionMode(true);
    setBatchResult(null);
    setError(null);
  }, []);

  const exitSelectionMode = useCallback((): void => {
    setSelectionMode(false);
    setSelectedProjectIds([]);
  }, []);

  const toggleProjectSelection = useCallback((projectId: number): void => {
    setSelectionMode(true);
    setBatchResult(null);
    setSelectedProjectIds((prev) => prev.includes(projectId) ? prev.filter((id) => id !== projectId) : [...prev, projectId]);
  }, []);

  const selectAllFilteredProjects = useCallback((): void => {
    setSelectionMode(true);
    setBatchResult(null);
    setSelectedProjectIds((prev) => {
      const next = new Set(prev);
      for (const projectId of filteredProjectIds) {
        next.add(projectId);
      }
      return Array.from(next);
    });
  }, [filteredProjectIds]);

  const clearProjectSelection = useCallback((): void => {
    setSelectedProjectIds([]);
    setBatchResult(null);
  }, []);

  const deleteSelectedProjects = useCallback(async (): Promise<ProjectBatchManageResult | null> => {
    const requestedIds = [...selectedProjectIds];
    if (requestedIds.length === 0) return null;
    setBatchDeleting(true);
    setError(null);
    setBatchResult(null);
    try {
      const result = await MidiService.deleteProjects(requestedIds);
      if (!aliveRef.current) return null;
      setBatchResult(result);
      const deletedIds = result.items.filter((item) => item.status === 'deleted').map((item) => item.projectId);
      applyDeletedProjectIds(deletedIds);
      return result;
    } catch (e: unknown) {
      if (!aliveRef.current) return null;
      setError(errorMessage(e));
      return null;
    } finally {
      if (aliveRef.current) setBatchDeleting(false);
    }
  }, [applyDeletedProjectIds, selectedProjectIds]);

  return {
    projects,
    filteredProjects,
    selected,
    previewPlan,
    form,
    query,
    sortValue,
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
    batchResult,
    saveError,
    previewError,
    isDirty: currentProfile ? !isSameForm(form, profileToForm(currentProfile, selected?.project.id)) : false,
    selectionMode,
    selectedProjectIds,
    selectedCount,
    allFilteredSelected,
    someFilteredSelected,
    batchDeleting,
    playlistOpen,
    playlistItems,
    playlistMode,
    playlistCurrentIndex,
    playlistCurrentProjectId,
    playlistDraggingIndex,
    playlistAutoStartToken,
    autoOpenImportedProject: preferences.libraryAutoOpenImportedProject,
    setQuery,
    setSortValue,
    setActivePanel,
    refresh,
    importMidiFiles,
    importMidiDirectory,
    importDroppedPaths,
    selectProject,
    selectProfile,
    updateField,
    resetForm,
    refreshPreview,
    save,
    deleteProject,
    togglePlaylist,
    setPlaylistOpen,
    setPlaylistMode,
    addProjectToPlaylist,
    addProjectNextInPlaylist,
    addSelectedProjectToPlaylist,
    addFilteredProjectsToPlaylist,
    removePlaylistItem,
    clearPlaylist,
    selectPlaylistItem,
    playPlaylistItem,
    movePlaylistItem,
    startPlaylistDrag,
    finishPlaylistDrag,
    handlePlayerState,
    markPlaylistStarted,
    enterSelectionMode,
    exitSelectionMode,
    toggleProjectSelection,
    selectAllFilteredProjects,
    clearProjectSelection,
    deleteSelectedProjects,
  };
};

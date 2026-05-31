import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { EditorSelectionService, MidiService, type MidiConfigSnapshot, type MidiProjectDetail, type MidiProfile, type PlayPlan } from '@/services';

export interface EditorFormState {
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
}

export interface UseEditorPageResult {
  project: MidiProjectDetail | null;
  previewPlan: PlayPlan | null;
  form: EditorFormState;
  selectedProfileId: number;
  saving: boolean;
  loading: boolean;
  previewLoading: boolean;
  error: string | null;
  saveError: string | null;
  previewError: string | null;
  isDirty: boolean;
  selectProfile: (profileId: number) => void;
  updateField: <K extends keyof EditorFormState>(field: K, value: EditorFormState[K]) => void;
  resetForm: () => void;
  refreshPreview: () => Promise<void>;
  save: () => Promise<void>;
  backToLibrary: () => void;
}

const DEFAULT_FORM: EditorFormState = {
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
};

const errorMessage = (e: unknown): string => e instanceof Error ? e.message : String(e ?? '');

const profileToForm = (profile: MidiProfile): EditorFormState => ({
  id: profile.id,
  projectId: profile.projectId,
  name: profile.name,
  baseNote: profile.baseNote,
  transpose: profile.transpose,
  octaveShift: profile.octaveShift,
  speed: profile.speed,
  outOfRangePolicy: profile.outOfRangePolicy,
  minPressMs: profile.minPressMs,
  releaseGapMs: profile.releaseGapMs,
  keymapProfileId: profile.keymapProfileId,
});

const isSameForm = (a: EditorFormState, b: EditorFormState): boolean => (
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
);

export const useEditorPage = (): UseEditorPageResult => {
  const aliveRef = useRef(true);
  const [project, setProject] = useState<MidiProjectDetail | null>(null);
  const [previewPlan, setPreviewPlan] = useState<PlayPlan | null>(null);
  const [form, setForm] = useState<EditorFormState>(DEFAULT_FORM);
  const [selectedProfileId, setSelectedProfileId] = useState(0);
  const [loading, setLoading] = useState(false);
  const [previewLoading, setPreviewLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [saveError, setSaveError] = useState<string | null>(null);
  const [previewError, setPreviewError] = useState<string | null>(null);

  useEffect(() => {
    aliveRef.current = true;
    return () => {
      aliveRef.current = false;
    };
  }, []);

  const loadPreviewPlan = useCallback(async (projectId: number, profileId: number): Promise<void> => {
    setPreviewLoading(true);
    setPreviewError(null);
    try {
      const plan = await MidiService.buildPlayPlan(projectId, profileId);
      if (!aliveRef.current) return;
      setPreviewPlan(plan);
    } catch (e: unknown) {
      if (!aliveRef.current) return;
      setPreviewError(errorMessage(e));
    } finally {
      if (aliveRef.current) setPreviewLoading(false);
    }
  }, []);

  const load = useCallback(async (): Promise<void> => {
    const projectId = EditorSelectionService.getProjectId();
    if (!projectId) {
      setProject(null);
      setPreviewPlan(null);
      setForm(DEFAULT_FORM);
      setSelectedProfileId(0);
      setLoading(false);
      return;
    }

    setLoading(true);
    setError(null);
    try {
      const detail = await MidiService.getProject(projectId);
      if (!aliveRef.current) return;
      setProject(detail);
      const nextProfile = detail.profiles.find((p) => p.id === detail.project.defaultProfileId) ?? detail.defaultProfile;
      setSelectedProfileId(nextProfile.id);
      setForm({ ...profileToForm(nextProfile), projectId: detail.project.id });
      await loadPreviewPlan(detail.project.id, nextProfile.id);
    } catch (e: unknown) {
      if (!aliveRef.current) return;
      setError(errorMessage(e));
    } finally {
      if (aliveRef.current) setLoading(false);
    }
  }, [loadPreviewPlan]);

  useEffect(() => {
    void load();
  }, [load]);

  const currentProfile = useMemo(() => project?.profiles.find((p) => p.id === selectedProfileId) ?? project?.defaultProfile ?? null, [project, selectedProfileId]);

  const selectProfile = useCallback((profileId: number) => {
    if (!project) return;
    const next = project.profiles.find((p) => p.id === profileId) ?? (project.defaultProfile.id === profileId ? project.defaultProfile : null);
    if (!next) return;
    setSelectedProfileId(profileId);
    setForm({ ...profileToForm(next), projectId: project.project.id });
    setSaveError(null);
    void loadPreviewPlan(project.project.id, profileId);
  }, [project]);

  const updateField = useCallback(<K extends keyof EditorFormState>(field: K, value: EditorFormState[K]) => {
    setForm((prev) => ({ ...prev, [field]: value }));
    setSaveError(null);
  }, []);

  const resetForm = useCallback(() => {
    if (!currentProfile) return;
    setForm({ ...profileToForm(currentProfile), projectId: project?.project.id });
    setSaveError(null);
  }, [currentProfile, project]);

  const refreshPreview = useCallback(async (): Promise<void> => {
    const projectId = project?.project.id ?? EditorSelectionService.getProjectId();
    if (!projectId || selectedProfileId === 0) return;
    await loadPreviewPlan(projectId, selectedProfileId);
  }, [loadPreviewPlan, project?.project.id, selectedProfileId]);

  const save = useCallback(async (): Promise<void> => {
    setSaving(true);
    setSaveError(null);
    try {
      const payload: MidiProfile = {
        id: form.id,
        projectId: form.projectId,
        name: form.name,
        baseNote: form.baseNote,
        transpose: form.transpose,
        octaveShift: form.octaveShift,
        speed: form.speed,
        outOfRangePolicy: form.outOfRangePolicy,
        minPressMs: form.minPressMs,
        releaseGapMs: form.releaseGapMs,
        keymapProfileId: form.keymapProfileId,
        enabledTracks: null,
        createdAt: 0,
        updatedAt: 0,
      };
      const saved = await MidiService.updateProfile(payload);
      if (!aliveRef.current) return;
      setForm({ ...profileToForm(saved), projectId: saved.projectId ?? project?.project.id });
      setSelectedProfileId(saved.id);
      const detail = await MidiService.getProject(saved.projectId ?? project?.project.id ?? 0);
      if (!aliveRef.current) return;
      setProject(detail);
      EditorSelectionService.setProjectId(detail.project.id);
      await loadPreviewPlan(detail.project.id, saved.id);
    } catch (e: unknown) {
      if (!aliveRef.current) return;
      setSaveError(errorMessage(e));
    } finally {
      if (aliveRef.current) setSaving(false);
    }
  }, [form, loadPreviewPlan, project?.project.id]);

  const backToLibrary = useCallback(() => {
    EditorSelectionService.clear();
  }, []);

  return {
    project,
    previewPlan,
    form,
    selectedProfileId,
    saving,
    loading,
    previewLoading,
    error,
    saveError,
    previewError,
    isDirty: currentProfile ? !isSameForm(form, profileToForm(currentProfile)) : false,
    selectProfile,
    updateField,
    resetForm,
    refreshPreview,
    save,
    backToLibrary,
  };
};

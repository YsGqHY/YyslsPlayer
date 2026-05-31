import { useCallback, useEffect, useRef, useState } from 'react';
import { MidiService, type MidiProfile, type OutOfRangePolicy } from '@/services';

export interface MidiDefaultsForm {
  id: number;
  name: string;
  baseNote: number;
  transpose: number;
  octaveShift: number;
  speed: number;
  outOfRangePolicy: OutOfRangePolicy;
  minPressMs: number;
  releaseGapMs: number;
  keymapProfileId: number;
}

export interface UseMidiDefaultsResult {
  form: MidiDefaultsForm;
  loading: boolean;
  saving: boolean;
  error: string | null;
  saved: boolean;
  updateField: <K extends keyof MidiDefaultsForm>(field: K, value: MidiDefaultsForm[K]) => void;
  save: () => Promise<void>;
  reset: () => Promise<void>;
  reload: () => Promise<void>;
}

const DEFAULT_FORM: MidiDefaultsForm = {
  id: 0,
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

const profileToForm = (profile: MidiProfile): MidiDefaultsForm => ({
  id: profile.id,
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

const formToProfile = (form: MidiDefaultsForm): MidiProfile => ({
  id: form.id,
  projectId: undefined,
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
});

export const useMidiDefaults = (): UseMidiDefaultsResult => {
  const aliveRef = useRef(true);
  const [form, setForm] = useState<MidiDefaultsForm>(DEFAULT_FORM);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [saved, setSaved] = useState(false);

  useEffect(() => {
    aliveRef.current = true;
    return () => {
      aliveRef.current = false;
    };
  }, []);

  const reload = useCallback(async (): Promise<void> => {
    setLoading(true);
    setError(null);
    setSaved(false);
    try {
      const profile = await MidiService.getDefaultProfile();
      if (!aliveRef.current) return;
      setForm(profileToForm(profile));
    } catch (e: unknown) {
      if (!aliveRef.current) return;
      setError(errorMessage(e));
    } finally {
      if (aliveRef.current) setLoading(false);
    }
  }, []);

  useEffect(() => {
    void reload();
  }, [reload]);

  const updateField = useCallback(<K extends keyof MidiDefaultsForm>(field: K, value: MidiDefaultsForm[K]): void => {
    setForm((prev) => ({ ...prev, [field]: value }));
    setSaved(false);
    setError(null);
  }, []);

  const save = useCallback(async (): Promise<void> => {
    setSaving(true);
    setError(null);
    setSaved(false);
    try {
      const savedProfile = await MidiService.updateDefaultProfile(formToProfile(form));
      if (!aliveRef.current) return;
      setForm(profileToForm(savedProfile));
      setSaved(true);
    } catch (e: unknown) {
      if (!aliveRef.current) return;
      setError(errorMessage(e));
    } finally {
      if (aliveRef.current) setSaving(false);
    }
  }, [form]);

  const reset = useCallback(async (): Promise<void> => {
    setSaving(true);
    setError(null);
    setSaved(false);
    try {
      const resetProfile = await MidiService.resetDefaultProfile();
      if (!aliveRef.current) return;
      setForm(profileToForm(resetProfile));
      setSaved(true);
    } catch (e: unknown) {
      if (!aliveRef.current) return;
      setError(errorMessage(e));
    } finally {
      if (aliveRef.current) setSaving(false);
    }
  }, []);

  return { form, loading, saving, error, saved, updateField, save, reset, reload };
};

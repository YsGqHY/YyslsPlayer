import { useCallback } from 'react';
import { DEFAULT_PREFERENCES, usePreferences, type PreviewWaveform } from '@/preferences';

export interface UsePreviewSettingsResult {
  volume: number;
  waveform: PreviewWaveform;
  progressHz: number;
  pianoRollMaxNotes: number;
  setVolume: (value: number) => void;
  setWaveform: (value: PreviewWaveform) => void;
  setProgressHz: (value: number) => void;
  setPianoRollMaxNotes: (value: number) => void;
  reset: () => void;
}

export const WAVEFORMS: PreviewWaveform[] = [
  'warmSine',
  'softSine',
  'mellowTriangle',
  'roundedBell',
  'glassPad',
  'mutedPluck',
  'softOrgan',
  'warmPad',
  'sine',
  'triangle',
  'square',
  'sawtooth',
];

const clampNumber = (value: number, min: number, max: number, fallback: number): number => {
  if (!Number.isFinite(value)) return fallback;
  return Math.max(min, Math.min(max, value));
};

const clampInt = (value: number, min: number, max: number, fallback: number): number => Math.round(clampNumber(value, min, max, fallback));

const normalizeWaveform = (value: string): PreviewWaveform => WAVEFORMS.includes(value as PreviewWaveform) ? value as PreviewWaveform : DEFAULT_PREFERENCES.previewWaveform;

export const usePreviewSettings = (): UsePreviewSettingsResult => {
  const { preferences, setPreference } = usePreferences();

  const setVolume = useCallback((value: number): void => {
    setPreference('previewVolume', clampNumber(value, 0, 0.5, DEFAULT_PREFERENCES.previewVolume));
  }, [setPreference]);

  const setWaveform = useCallback((value: PreviewWaveform): void => {
    setPreference('previewWaveform', normalizeWaveform(value));
  }, [setPreference]);

  const setProgressHz = useCallback((value: number): void => {
    setPreference('previewProgressHz', clampInt(value, 1, 30, DEFAULT_PREFERENCES.previewProgressHz));
  }, [setPreference]);

  const setPianoRollMaxNotes = useCallback((value: number): void => {
    setPreference('pianoRollMaxNotes', clampInt(value, 100, 5000, DEFAULT_PREFERENCES.pianoRollMaxNotes));
  }, [setPreference]);

  const reset = useCallback((): void => {
    setPreference('previewVolume', DEFAULT_PREFERENCES.previewVolume);
    setPreference('previewWaveform', DEFAULT_PREFERENCES.previewWaveform);
    setPreference('previewProgressHz', DEFAULT_PREFERENCES.previewProgressHz);
    setPreference('pianoRollMaxNotes', DEFAULT_PREFERENCES.pianoRollMaxNotes);
  }, [setPreference]);

  return {
    volume: clampNumber(preferences.previewVolume, 0, 0.5, DEFAULT_PREFERENCES.previewVolume),
    waveform: normalizeWaveform(preferences.previewWaveform),
    progressHz: clampInt(preferences.previewProgressHz, 1, 30, DEFAULT_PREFERENCES.previewProgressHz),
    pianoRollMaxNotes: clampInt(preferences.pianoRollMaxNotes, 100, 5000, DEFAULT_PREFERENCES.pianoRollMaxNotes),
    setVolume,
    setWaveform,
    setProgressHz,
    setPianoRollMaxNotes,
    reset,
  };
};

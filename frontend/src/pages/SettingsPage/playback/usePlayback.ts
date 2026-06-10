import { useCallback } from 'react';
import { DEFAULT_PREFERENCES, usePreferences } from '@/preferences';

export interface UsePlaybackResult {
  lookaheadMs: number;
  countdownSeconds: number;
  setLookaheadMs: (value: number) => void;
  setCountdownSeconds: (value: number) => void;
  reset: () => void;
}

const clampInt = (value: number, min: number, max: number, fallback: number): number => {
  if (!Number.isFinite(value)) return fallback;
  return Math.max(min, Math.min(max, Math.round(value)));
};

export const usePlayback = (): UsePlaybackResult => {
  const { preferences, setPreference } = usePreferences();

  const setLookaheadMs = useCallback((value: number): void => {
    setPreference('performLookaheadMs', clampInt(value, 5, 50, DEFAULT_PREFERENCES.performLookaheadMs));
  }, [setPreference]);

  const setCountdownSeconds = useCallback((value: number): void => {
    setPreference('performCountdownSeconds', clampInt(value, 0, 10, DEFAULT_PREFERENCES.performCountdownSeconds));
  }, [setPreference]);

  const reset = useCallback((): void => {
    setPreference('performLookaheadMs', DEFAULT_PREFERENCES.performLookaheadMs);
    setPreference('performCountdownSeconds', DEFAULT_PREFERENCES.performCountdownSeconds);
  }, [setPreference]);

  return {
    lookaheadMs: clampInt(preferences.performLookaheadMs, 5, 50, DEFAULT_PREFERENCES.performLookaheadMs),
    countdownSeconds: clampInt(preferences.performCountdownSeconds, 0, 10, DEFAULT_PREFERENCES.performCountdownSeconds),
    setLookaheadMs,
    setCountdownSeconds,
    reset,
  };
};

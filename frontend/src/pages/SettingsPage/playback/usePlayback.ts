import { useCallback } from 'react';
import { DEFAULT_PREFERENCES, usePreferences } from '@/preferences';

export interface UsePlaybackResult {
  showDryRunDefault: boolean;
  dryRunDefault: boolean;
  lookaheadMs: number;
  countdownSeconds: number;
  setDryRunDefault: (value: boolean) => void;
  setLookaheadMs: (value: number) => void;
  setCountdownSeconds: (value: number) => void;
  reset: () => void;
}

const isProductionBuild = import.meta.env.PROD;

const clampInt = (value: number, min: number, max: number, fallback: number): number => {
  if (!Number.isFinite(value)) return fallback;
  return Math.max(min, Math.min(max, Math.round(value)));
};

export const usePlayback = (): UsePlaybackResult => {
  const { preferences, setPreference } = usePreferences();

  const setDryRunDefault = useCallback((value: boolean): void => {
    if (isProductionBuild) return;
    setPreference('performDryRunDefault', value);
  }, [setPreference]);

  const setLookaheadMs = useCallback((value: number): void => {
    setPreference('performLookaheadMs', clampInt(value, 5, 50, DEFAULT_PREFERENCES.performLookaheadMs));
  }, [setPreference]);

  const setCountdownSeconds = useCallback((value: number): void => {
    setPreference('performCountdownSeconds', clampInt(value, 0, 10, DEFAULT_PREFERENCES.performCountdownSeconds));
  }, [setPreference]);

  const reset = useCallback((): void => {
    if (!isProductionBuild) {
      setPreference('performDryRunDefault', DEFAULT_PREFERENCES.performDryRunDefault);
    }
    setPreference('performLookaheadMs', DEFAULT_PREFERENCES.performLookaheadMs);
    setPreference('performCountdownSeconds', DEFAULT_PREFERENCES.performCountdownSeconds);
  }, [setPreference]);

  return {
    showDryRunDefault: !isProductionBuild,
    dryRunDefault: isProductionBuild ? false : preferences.performDryRunDefault,
    lookaheadMs: clampInt(preferences.performLookaheadMs, 5, 50, DEFAULT_PREFERENCES.performLookaheadMs),
    countdownSeconds: clampInt(preferences.performCountdownSeconds, 0, 10, DEFAULT_PREFERENCES.performCountdownSeconds),
    setDryRunDefault,
    setLookaheadMs,
    setCountdownSeconds,
    reset,
  };
};

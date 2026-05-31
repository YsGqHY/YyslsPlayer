import { useCallback, useEffect, useState } from 'react';
import { DEFAULT_PREFERENCES, usePreferences } from '@/preferences';

export interface UseLibrarySettingsResult {
  autoOpenImportedProject: boolean;
  listLimitInput: string;
  setAutoOpenImportedProject: (value: boolean) => void;
  setListLimitInput: (value: string) => void;
  commitListLimit: () => void;
  reset: () => void;
}

const LIST_LIMIT_MIN = 5;
const LIST_LIMIT_MAX = 10000;

const clampLimit = (value: number): number => {
  if (!Number.isFinite(value)) return DEFAULT_PREFERENCES.libraryListLimit;
  return Math.max(LIST_LIMIT_MIN, Math.min(LIST_LIMIT_MAX, Math.round(value)));
};

export const useLibrarySettings = (): UseLibrarySettingsResult => {
  const { preferences, setPreference } = usePreferences();
  const listLimit = clampLimit(preferences.libraryListLimit);
  const [listLimitInput, setListLimitInput] = useState(() => String(listLimit));

  useEffect(() => {
    setListLimitInput(String(listLimit));
  }, [listLimit]);

  const setAutoOpenImportedProject = useCallback((value: boolean): void => {
    setPreference('libraryAutoOpenImportedProject', value);
  }, [setPreference]);

  const commitListLimit = useCallback((): void => {
    const value = Number(listLimitInput);
    const next = clampLimit(value);
    setListLimitInput(String(next));
    setPreference('libraryListLimit', next);
  }, [listLimitInput, setPreference]);

  const reset = useCallback((): void => {
    setPreference('libraryAutoOpenImportedProject', DEFAULT_PREFERENCES.libraryAutoOpenImportedProject);
    setPreference('libraryListLimit', DEFAULT_PREFERENCES.libraryListLimit);
    setListLimitInput(String(DEFAULT_PREFERENCES.libraryListLimit));
  }, [setPreference]);

  return {
    autoOpenImportedProject: preferences.libraryAutoOpenImportedProject,
    listLimitInput,
    setAutoOpenImportedProject,
    setListLimitInput,
    commitListLimit,
    reset,
  };
};

import { useCallback, useState } from 'react';

export interface UseLibrarySelectionResult {
  selectionMode: boolean;
  selectedProjectIds: number[];
  selectedCount: number;
  allFilteredSelected: boolean;
  someFilteredSelected: boolean;
  batchDeleting: boolean;
  enterSelectionMode: () => void;
  exitSelectionMode: () => void;
  toggleProjectSelection: (projectId: number) => void;
  selectAllFilteredProjects: () => void;
  clearProjectSelection: () => void;
}

export const useLibrarySelection = (
  filteredProjectIds: number[],
): UseLibrarySelectionResult => {
  const [selectionMode, setSelectionMode] = useState(false);
  const [selectedProjectIds, setSelectedProjectIds] = useState<number[]>([]);
  const [batchDeleting] = useState(false);

  const selectedCount = selectedProjectIds.length;
  const selectedIdSet = new Set(selectedProjectIds);
  const allFilteredSelected = filteredProjectIds.length > 0 && filteredProjectIds.every((id) => selectedIdSet.has(id));
  const someFilteredSelected = filteredProjectIds.some((id) => selectedIdSet.has(id));

  const enterSelectionMode = useCallback((): void => {
    setSelectionMode(true);
  }, []);

  const exitSelectionMode = useCallback((): void => {
    setSelectionMode(false);
    setSelectedProjectIds([]);
  }, []);

  const toggleProjectSelection = useCallback((projectId: number): void => {
    setSelectionMode(true);
    setSelectedProjectIds((prev) => prev.includes(projectId) ? prev.filter((id) => id !== projectId) : [...prev, projectId]);
  }, []);

  const selectAllFilteredProjects = useCallback((): void => {
    setSelectionMode(true);
    setSelectedProjectIds((prev) => {
      const next = new Set(prev);
      for (const id of filteredProjectIds) next.add(id);
      return Array.from(next);
    });
  }, [filteredProjectIds]);

  const clearProjectSelection = useCallback((): void => {
    setSelectedProjectIds([]);
  }, []);

  return {
    selectionMode,
    selectedProjectIds,
    selectedCount,
    allFilteredSelected,
    someFilteredSelected,
    batchDeleting,
    enterSelectionMode,
    exitSelectionMode,
    toggleProjectSelection,
    selectAllFilteredProjects,
    clearProjectSelection,
  };
};

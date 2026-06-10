import { useCallback, useMemo, useRef, useState } from 'react';
import { MidiService, type MidiProjectSummary } from '@/services';

type LibrarySortKey = 'createdAt' | 'fileName' | 'durationMs' | 'fileSizeBytes';
type LibrarySortDirection = 'asc' | 'desc';
type LibrarySortValue = `${LibrarySortKey}:${LibrarySortDirection}`;

const DEFAULT_SORT_VALUE: LibrarySortValue = 'createdAt:desc';

const parseSortValue = (sortValue: LibrarySortValue): { key: LibrarySortKey; direction: LibrarySortDirection } => {
  const [key, direction] = sortValue.split(':') as [LibrarySortKey, LibrarySortDirection];
  return { key, direction };
};

/** 文件名自然排序用的 Intl.Collator。 */
export const fileNameCollator = new Intl.Collator(undefined, { numeric: true, sensitivity: 'base' });

export const sortProjects = (rows: MidiProjectSummary[], sortValue: LibrarySortValue): MidiProjectSummary[] => {
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

const errorMessage = (e: unknown): string => e instanceof Error ? e.message : String(e ?? '');

export interface UseLibraryListResult {
  projects: MidiProjectSummary[];
  filteredProjects: MidiProjectSummary[];
  query: string;
  sortValue: LibrarySortValue;
  loading: boolean;
  error: string | null;
  setQuery: (query: string) => void;
  setSortValue: (sortValue: LibrarySortValue) => void;
  refresh: () => Promise<void>;
  selectRelativeProject: (direction: -1 | 1) => void;
}

export const useLibraryList = (
  listLimit: number,
  selectedProjectIdRef: React.MutableRefObject<number | null>,
  aliveRef: React.MutableRefObject<boolean>,
  onSelectProject: (projectId: number) => Promise<void>,
): UseLibraryListResult => {
  const [projects, setProjects] = useState<MidiProjectSummary[]>([]);
  const [query, setQuery] = useState('');
  const [sortValue, setSortValue] = useState<LibrarySortValue>(DEFAULT_SORT_VALUE);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const clearSelectionRef = useRef<() => void>(() => {});

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

  const refresh = useCallback(async (): Promise<void> => {
    setLoading(true);
    setError(null);
    try {
      const rows = await MidiService.listProjects({ limit: listLimit });
      if (!aliveRef.current) return;
      setProjects(rows);
      const currentId = selectedProjectIdRef.current;
      if (currentId && rows.every((p) => p.id !== currentId)) {
        clearSelectionRef.current();
      }
    } catch (e: unknown) {
      if (!aliveRef.current) return;
      setError(errorMessage(e));
    } finally {
      if (aliveRef.current) setLoading(false);
    }
  }, [aliveRef, listLimit, selectedProjectIdRef]);

  const selectRelativeProject = useCallback((direction: -1 | 1): void => {
    if (filteredProjects.length === 0) return;
    const currentID = selectedProjectIdRef.current;
    const currentIndex = currentID ? filteredProjects.findIndex((project) => project.id === currentID) : -1;
    const nextIndex = currentIndex < 0
      ? (direction > 0 ? 0 : filteredProjects.length - 1)
      : (currentIndex + direction + filteredProjects.length) % filteredProjects.length;
    const nextProject = filteredProjects[nextIndex];
    if (nextProject) {
      void onSelectProject(nextProject.id);
    }
  }, [filteredProjects, onSelectProject, selectedProjectIdRef]);

  const setClearSelection = useCallback((fn: () => void) => {
    clearSelectionRef.current = fn;
  }, []);

  return {
    projects,
    filteredProjects,
    query,
    sortValue,
    loading,
    error,
    setQuery,
    setSortValue,
    refresh,
    selectRelativeProject,
    _internal: { setProjects, setClearSelection },
  } as UseLibraryListResult & { _internal: { setProjects: typeof setProjects; setClearSelection: typeof setClearSelection } };
};

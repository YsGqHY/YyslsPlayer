import { useCallback, useState } from 'react';
import { MidiService, NativeDialogs, type ImportBatchResult, type MidiProjectDetail } from '@/services';
import type { MidiProfile } from '@/services';

const errorMessage = (e: unknown): string => e instanceof Error ? e.message : String(e ?? '');

export interface UseLibraryImportResult {
  importing: boolean;
  importError: string | null;
  importSummary: ImportBatchResult | null;
  importMidiFiles: (dialogTitle: string, filterName: string) => Promise<number | null>;
  importMidiDirectory: (dialogTitle: string) => Promise<number | null>;
  importDroppedPaths: (paths: string[]) => Promise<number | null>;
}

export const useLibraryImport = (
  listLimit: number,
  aliveRef: React.MutableRefObject<boolean>,
  applyDetail: (detail: MidiProjectDetail, preferredProfileId?: number) => MidiProfile,
  loadPreviewPlan: (projectId: number, profileId: number) => Promise<void>,
  setProjects: (projects: React.SetStateAction<any[]>) => void,
): UseLibraryImportResult => {
  const [importing, setImporting] = useState(false);
  const [importError, setImportError] = useState<string | null>(null);
  const [importSummary, setImportSummary] = useState<ImportBatchResult | null>(null);

  const applyImportResult = useCallback(async (result: ImportBatchResult): Promise<number | null> => {
    setImportSummary(result);
    const projectId = result.firstImportedProjectId ?? result.firstProjectId ?? null;
    const rows = await MidiService.listProjects({ limit: listLimit });
    if (!aliveRef.current) return null;
    (setProjects as any)(rows);
    if (!projectId) return null;
    const detail = await MidiService.getProject(projectId);
    if (!aliveRef.current) return null;
    const nextProfile = applyDetail(detail);
    await loadPreviewPlan(detail.project.id, nextProfile.id);
    return projectId;
  }, [aliveRef, applyDetail, listLimit, loadPreviewPlan, setProjects]);

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
  }, [aliveRef, applyImportResult]);

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
  }, [aliveRef, applyImportResult]);

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
  }, [aliveRef, applyImportResult]);

  return {
    importing,
    importError,
    importSummary,
    importMidiFiles,
    importMidiDirectory,
    importDroppedPaths,
  };
};

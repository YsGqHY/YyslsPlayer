import { useCallback, useEffect, useRef, useState } from 'react';
import { useRouter } from '@/router';
import { useT } from '@/i18n';
import {
  TranscriptionService,
  type TranscriptionTask,
  type TranscriptionTaskDetail,
  type TranscriptionConfig,
  type AudioProbeResult,
  type TranscriptionCapability,
  DEFAULT_TRANSCRIPTION_CONFIG,
} from '@/services/transcription/TranscriptionService';
import { NativeDialogs } from '@/services/dialogs/NativeDialogs';

export interface UseTranscriptionPageResult {
  capability: TranscriptionCapability | null;
  capabilityLoading: boolean;
  selectedFilePath: string | null;
  probeResult: AudioProbeResult | null;
  probeLoading: boolean;
  probeError: string | null;
  tasks: TranscriptionTask[];
  tasksLoading: boolean;
  selectedTask: TranscriptionTaskDetail | null;
  taskDetailLoading: boolean;
  config: TranscriptionConfig;
  configLoading: boolean;
  installingFfmpeg: boolean;
  ffmpegResult: string | null;
  installFfmpeg: () => Promise<void>;
  proxyAddress: string;
  setProxyAddress: (v: string) => void;
  installingModel: boolean;
  modelResult: string | null;
  installModel: () => Promise<void>;
  selectAudioFile: () => Promise<void>;
  createTranscriptionTask: () => Promise<void>;
  refreshTasks: () => Promise<void>;
  selectTask: (id: number) => Promise<void>;
  cancelTask: (id: number) => Promise<void>;
  deleteTask: (id: number) => Promise<void>;
  importResult: (id: number) => Promise<void>;
  exportResult: (id: number) => Promise<void>;
  updateConfig: () => Promise<void>;
  setConfigField: <K extends keyof TranscriptionConfig>(key: K, value: TranscriptionConfig[K]) => void;
}

export function useTranscriptionPage(): UseTranscriptionPageResult {
  const { navigate } = useRouter();
  const t = useT();

  const [capability, setCapability] = useState<TranscriptionCapability | null>(null);
  const [capabilityLoading, setCapabilityLoading] = useState(true);
  const [selectedFilePath, setSelectedFilePath] = useState<string | null>(null);
  const [probeResult, setProbeResult] = useState<AudioProbeResult | null>(null);
  const [probeLoading, setProbeLoading] = useState(false);
  const [probeError, setProbeError] = useState<string | null>(null);
  const [tasks, setTasks] = useState<TranscriptionTask[]>([]);
  const [tasksLoading, setTasksLoading] = useState(false);
  const [selectedTask, setSelectedTask] = useState<TranscriptionTaskDetail | null>(null);
  const [taskDetailLoading, setTaskDetailLoading] = useState(false);
  const [config, setConfig] = useState<TranscriptionConfig>(DEFAULT_TRANSCRIPTION_CONFIG);
  const [configLoading, setConfigLoading] = useState(true);
  const [installingFfmpeg, setInstallingFfmpeg] = useState(false);
  const [ffmpegResult, setFfmpegResult] = useState<string | null>(null);
  const [installingModel, setInstallingModel] = useState(false);
  const [modelResult, setModelResult] = useState<string | null>(null);
  const [proxyAddress, setProxyAddress] = useState('');

  const unavailable = !TranscriptionService.enabled;

  /** 解析转录错误码为人类可读消息。提取 "code: detail" 格式前缀，查找 i18n 映射。
   *  i18n 错误对象使用嵌套结构：errors.asset.ffmpeg_missing → transcription.errors.asset.ffmpeg_missing
   *  命中 i18n 时返回"指引文案（原始技术细节）"，供用户知晓如何操作同时可诊断具体原因。 */
  const resolveError = useCallback((raw: string): string => {
    if (!raw) return '';
    const match = raw.match(/^([\w.]+?):\s*(.*)/);
    if (match) {
      const code = match[1];
      const detail = match[2];
      const key = `transcription.errors.${code}`;
      const resolved = t(key);
      // t() 在未命中时返回 key 本身；命中则返回转换后的文案
      if (resolved !== key) {
        return detail ? `${resolved}（${detail}）` : resolved;
      }
    }
    return raw;
  }, [t]);

  useEffect(() => {
    if (unavailable) { setCapabilityLoading(false); return; }
    Promise.all([
      TranscriptionService.getCapability(),
      TranscriptionService.getConfig(),
      TranscriptionService.listTasks(),
    ])
      .then(([cap, cfg, taskList]) => { setCapability(cap); setConfig(cfg); setTasks(taskList); })
      .catch(() => { setCapability({ transcriptionEnabled: false, buildFlavor: 'unknown', missingComponents: [] }); })
      .finally(() => { setCapabilityLoading(false); setConfigLoading(false); });
  }, [unavailable]);

  const unsubRef = useRef<(() => void)[]>([]);
  useEffect(() => {
    if (unavailable) return;
    const unsubs = [
      TranscriptionService.onProgress((evt) => {
        setTasks((prev) => prev.map((t) => {
          if (String(t.id) === evt.taskId) return { ...t, status: evt.status, progress: evt.progress, stage: evt.message || t.stage };
          return t;
        }));
      }),
      TranscriptionService.onCompleted(() => { setTimeout(() => refreshTasksRef.current?.(), 500); }),
      TranscriptionService.onFailed(() => { setTimeout(() => refreshTasksRef.current?.(), 500); }),
      TranscriptionService.onCancelled(() => { setTimeout(() => refreshTasksRef.current?.(), 500); }),
      TranscriptionService.onFfmpegInstalled(async () => {
        setFfmpegResult(t('transcription.missing.ffmpegInstalled'));
        try { setCapability(await TranscriptionService.getCapability()); } catch { /* keep current */ }
      }),
      TranscriptionService.onFfmpegInstallFailed((error) => {
        setFfmpegResult(error || t('transcription.missing.ffmpegInstallFailed'));
      }),
    ];
    unsubRef.current = unsubs;
    return () => unsubs.forEach((u) => u());
  }, [unavailable, t]);

  const refreshTasksRef = useRef<() => Promise<void>>(() => Promise.resolve());
  const refreshTasks = useCallback(async () => {
    setTasksLoading(true);
    try { setTasks(await TranscriptionService.listTasks()); } finally { setTasksLoading(false); }
  }, []);
  refreshTasksRef.current = refreshTasks;

  const selectAudioFile = useCallback(async () => {
    const file = await NativeDialogs.openFile({
      title: t('transcription.import.selectFileTitle'),
      filters: [
        { displayName: t('transcription.import.audioFilter'), pattern: '*.mp3;*.ogg;*.wav;*.flac;*.m4a;*.aac;*.opus' },
        { displayName: t('transcription.import.allFilesFilter'), pattern: '*.*' },
      ],
    });
    if (!file) return;
    setSelectedFilePath(file);
    setProbeResult(null); setProbeError(null); setProbeLoading(true);
    try { setProbeResult(await TranscriptionService.probeAudio(file)); }
    catch (e: any) { setProbeError(resolveError(e?.message || String(e))); }
    finally { setProbeLoading(false); }
  }, [t, resolveError]);

  const createTranscriptionTask = useCallback(async () => {
    if (!selectedFilePath) return;
    try { await TranscriptionService.createTask(selectedFilePath, config); await refreshTasks(); }
    catch (e: any) {
      await NativeDialogs.confirm({
        title: t('transcription.confirm.createFailTitle'),
        message: `${t('transcription.confirm.createFailMessage')} ${resolveError(e?.message || String(e))}`,
        okLabel: t('transcription.confirm.createFailOk'),
        cancelLabel: t('transcription.confirm.createFailCancel'),
      });
    }
  }, [selectedFilePath, config, refreshTasks, t, resolveError]);

  const selectTask = useCallback(async (id: number) => {
    setTaskDetailLoading(true);
    try { setSelectedTask(await TranscriptionService.getTask(id)); }
    catch (e: any) { console.error('Failed to load task detail', e); }
    finally { setTaskDetailLoading(false); }
  }, []);

  const cancelTask = useCallback(async (id: number) => {
    try { await TranscriptionService.cancelTask(id); await refreshTasks(); }
    catch (e: any) { console.error('Failed to cancel task', e); }
  }, [refreshTasks]);

  const deleteTask = useCallback(async (id: number) => {
    const ok = await NativeDialogs.confirm({
      title: t('transcription.confirm.deleteTitle'),
      message: t('transcription.confirm.deleteMessage'),
      okLabel: t('transcription.confirm.deleteOk'),
      cancelLabel: t('transcription.actions.delete'),
    });
    if (!ok) return;
    try {
      await TranscriptionService.deleteTask(id);
      if (selectedTask?.task.id === id) setSelectedTask(null);
      await refreshTasks();
    } catch (e: any) { console.error('Failed to delete task', e); }
  }, [refreshTasks, selectedTask, t]);

  const importResult = useCallback(async (id: number) => {
    try {
      const result = await TranscriptionService.importResultAsMidiProject(id);
      const ok = await NativeDialogs.confirm({
        title: t('transcription.confirm.importSuccess'),
        message: t('transcription.confirm.importMessage', { name: result.displayName, count: String(result.noteCount) }),
        okLabel: t('transcription.confirm.goToLibrary'),
        cancelLabel: t('transcription.confirm.stayHere'),
      });
      if (ok) navigate('library');
      await refreshTasks();
    } catch (e: any) {
      const errStr = e?.message || String(e);
      // smoke 验证失败的 warning 而非 error
      if (errStr.includes('midi.playplan_failed')) {
        await NativeDialogs.confirm({
          title: t('transcription.confirm.importWarningTitle'),
          message: t('transcription.confirm.importWarningMessage'),
          okLabel: t('transcription.confirm.importWarningOk'),
          cancelLabel: t('transcription.confirm.importWarningCancel'),
        });
        await refreshTasks();
        return;
      }
      await NativeDialogs.confirm({
        title: t('transcription.confirm.importFailTitle'),
        message: resolveError(errStr),
        okLabel: t('transcription.confirm.importFailOk'),
        cancelLabel: t('transcription.confirm.importFailCancel'),
      });
    }
  }, [refreshTasks, navigate, t, resolveError]);

  const exportResult = useCallback(async (id: number) => {
    const filePath = await NativeDialogs.saveFile({
      title: t('transcription.confirm.exportTitle'),
      filename: `transcription_${id}.mid`,
      filters: [{ displayName: t('transcription.confirm.exportFilter'), pattern: '*.mid' }],
    });
    if (!filePath) return;
    try { await TranscriptionService.exportResultMidi(id, filePath); }
    catch (e: any) { console.error('Export failed', e); }
  }, [t]);

  const updateConfig = useCallback(async () => {
    try { await TranscriptionService.updateConfig(config); }
    catch (e: any) { console.error('Failed to update config', e); }
  }, [config]);

  const setConfigField = useCallback(<K extends keyof TranscriptionConfig>(key: K, value: TranscriptionConfig[K]) => {
    setConfig((prev) => ({ ...prev, [key]: value }));
  }, []);

  const installFfmpeg = useCallback(async () => {
    setInstallingFfmpeg(true);
    setFfmpegResult(null);
    try {
      // 先设置下载代理
      if (proxyAddress.trim()) {
        await TranscriptionService.setDownloadProxy(proxyAddress.trim());
      } else {
        await TranscriptionService.setDownloadProxy('');
      }
      const msg = await TranscriptionService.installFfmpeg();
      setFfmpegResult(msg);
      try { setCapability(await TranscriptionService.getCapability()); } catch { /* keep current */ }
    } catch (e: any) {
      setFfmpegResult(resolveError(String(e?.message || e)));
    } finally {
      setInstallingFfmpeg(false);
    }
  }, [proxyAddress, resolveError]);

  const installModel = useCallback(async () => {
    setInstallingModel(true);
    setModelResult(null);
    try {
      const msg = await TranscriptionService.installModel();
      setModelResult(msg);
      try { setCapability(await TranscriptionService.getCapability()); } catch { /* keep current */ }
    } catch (e: any) {
      setModelResult(resolveError(String(e?.message || e)));
    } finally {
      setInstallingModel(false);
    }
  }, [resolveError]);

  return {
    capability, capabilityLoading,
    selectedFilePath, probeResult, probeLoading, probeError,
    tasks, tasksLoading,
    selectedTask, taskDetailLoading,
    config, configLoading,
    selectAudioFile, createTranscriptionTask, refreshTasks, selectTask,
    cancelTask, deleteTask, importResult, exportResult,
    updateConfig,
    setConfigField,
    installingFfmpeg,
    ffmpegResult,
    installFfmpeg,
    proxyAddress,
    setProxyAddress,
    installingModel,
    modelResult,
    installModel,
  };
}

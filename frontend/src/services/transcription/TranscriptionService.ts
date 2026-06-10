// TranscriptionService：音频转 MIDI 扒谱功能的前端封装。
//
// 使用 Call.ByID (numeric ID) 直接调用 Go service，不依赖 @bindings/.../transcription 静态 import。
// lite 构建时 bindings 目录被清理，静态 import 会失败；Call.ByID 无需模块解析，兼容双 flavor。
import { Call, Events } from '@wailsio/runtime';
import { AppEvents, type WailsEventPayload } from '@/shared/events';
import { FEATURES } from '@/shared/featureFlags';

// ===== 方法 ID（由 wails3 generate bindings 生成，Go 方法签名不变则 ID 稳定） =====

const METHOD_IDS: Record<string, number> = {
  CancelTask: 2709197560,
  CreateTask: 2870317166,
  DeleteTask: 1285601513,
  ExportResultMidi: 139977093,
  GetCapability: 1239525611,
  GetConfig: 469002571,
  GetTask: 1237367896,
  ImportResultAsMidiProject: 2452619,
  InstallFfmpeg: 3324406547,
  InstallModel: 569714259,
  ListTasks: 500807431,
  ProbeAudio: 3528892919,
  RetryTask: 4139993462,
  SetDownloadProxy: 2484605647,
  UpdateConfig: 3363917592,
};

// ===== DTO 类型 =====

export interface TranscriptionConfig {
  mode: 'melody' | 'polyphonic';
  minConfidence: number;
  minDurationMs: number;
  mergeGapMs: number;
  quantize: 'off' | 'light' | 'strong';
  maxPolyphony: number;
  targetBaseNote: number;
  targetLaneCount: number;
  outOfRangePolicy: string;
  preferMelodyRegister: boolean;
}

export interface TranscriptionTask {
  id: number;
  sourceFileName: string;
  status: string;
  stage: string;
  progress: number;
  durationMs: number;
  errorCode?: string;
  errorMessage?: string;
  createdAt: number;
  updatedAt: number;
}

export interface TranscriptionNote {
  id: number;
  taskId: number;
  note: number;
  velocity: number;
  startMs: number;
  durationMs: number;
  confidence: number;
  source: string;
  flagsJson?: string;
}

export interface TranscriptionAnalysis {
  id: number;
  taskId: number;
  kind: string;
  payloadJson: string;
  createdAt: number;
}

export interface TranscriptionTaskDetail {
  task: TranscriptionTask;
  configJson: string;
  engine: string;
  engineVersion: string;
  sampleRate: number;
  channels: number;
  sourceHash: string;
  resultMidiPath?: string;
  importedProjectId?: number;
  summaryJson?: string;
  reportJson?: string;
  notes: TranscriptionNote[];
  analysis: TranscriptionAnalysis[];
}

export interface AudioProbeResult {
  format: string;
  durationMs: number;
  sampleRate: number;
  channels: number;
  bitrate: number;
  codec: string;
  container: string;
  fileSizeBytes: number;
}

export interface TranscriptionCapability {
  transcriptionEnabled: boolean;
  buildFlavor: string;
  missingComponents: string[];
}

export interface MidiProjectImportResult {
  projectId: number;
  displayName: string;
  noteCount: number;
  durationMs: number;
  fileHash: string;
}

export interface TranscriptionProgress {
  taskId: string;
  status: string;
  progress: number;
  message?: string;
}

export interface TranscriptionResult {
  taskId: string;
  midiProjectId?: string;
  totalNotes: number;
  inRangeNotes: number;
  outRangeNotes: number;
  estimatedBpm: number;
  suggestedOctaveShift: number;
  coveragePercent: number;
  qualityReport?: string;
}

export interface TranscriptionError {
  taskId: string;
  errorCode: string;
  errorMessage: string;
}

// ===== 默认配置 =====

export const DEFAULT_TRANSCRIPTION_CONFIG: TranscriptionConfig = {
  mode: 'melody',
  minConfidence: 0.55,
  minDurationMs: 60,
  mergeGapMs: 40,
  quantize: 'light',
  maxPolyphony: 2,
  targetBaseNote: 48,
  targetLaneCount: 36,
  outOfRangePolicy: 'drop',
  preferMelodyRegister: true,
};

// ===== 调用辅助 =====

const callByID = async (method: string, ...args: unknown[]): Promise<unknown> => {
  const id = METHOD_IDS[method];
  if (id == null) throw new Error(`Method ID not found: ${method}`);
  return Call.ByID(id, ...args);
};

// ===== 数据归一化 =====

const asTask = (raw: any): TranscriptionTask => ({
  id: raw?.id ?? 0,
  sourceFileName: raw?.sourceFileName ?? '',
  status: raw?.status ?? '',
  stage: raw?.stage ?? '',
  progress: raw?.progress ?? 0,
  durationMs: raw?.durationMs ?? 0,
  errorCode: raw?.errorCode,
  errorMessage: raw?.errorMessage,
  createdAt: raw?.createdAt ?? 0,
  updatedAt: raw?.updatedAt ?? 0,
});

const asNote = (raw: any): TranscriptionNote => ({
  id: raw?.id ?? 0,
  taskId: raw?.taskId ?? 0,
  note: raw?.note ?? 0,
  velocity: raw?.velocity ?? 0,
  startMs: raw?.startMs ?? 0,
  durationMs: raw?.durationMs ?? 0,
  confidence: raw?.confidence ?? 0,
  source: raw?.source ?? '',
  flagsJson: raw?.flagsJson,
});

const asAnalysis = (raw: any): TranscriptionAnalysis => ({
  id: raw?.id ?? 0,
  taskId: raw?.taskId ?? 0,
  kind: raw?.kind ?? '',
  payloadJson: raw?.payloadJson ?? '',
  createdAt: raw?.createdAt ?? 0,
});

const asStr = (v: unknown): string => String(v ?? '');
const asNum = (v: unknown): number => Number(v ?? 0);
const asBool = (v: unknown): boolean => Boolean(v);
const asOptStr = (v: unknown): string | undefined => (v != null ? String(v) : undefined);
const asOptNum = (v: unknown): number | undefined => (v != null ? Number(v) : undefined);

// ===== Service =====

export const TranscriptionService = {
  get enabled(): boolean {
    return FEATURES.transcription;
  },

  async probeAudio(path: string): Promise<AudioProbeResult> {
    const raw: any = await callByID('ProbeAudio', path);
    return {
      format: asStr(raw?.format),
      durationMs: asNum(raw?.durationMs),
      sampleRate: asNum(raw?.sampleRate),
      channels: asNum(raw?.channels),
      bitrate: asNum(raw?.bitrate),
      codec: asStr(raw?.codec),
      container: asStr(raw?.container),
      fileSizeBytes: asNum(raw?.fileSizeBytes),
    };
  },

  async createTask(sourcePath: string, config: TranscriptionConfig = DEFAULT_TRANSCRIPTION_CONFIG): Promise<TranscriptionTask> {
    const result: any = await callByID('CreateTask', { sourcePath, config });
    return asTask(result);
  },

  async listTasks(limit = 50, offset = 0): Promise<TranscriptionTask[]> {
    const result: any = await callByID('ListTasks', limit, offset);
    return Array.isArray(result) ? result.map(asTask) : [];
  },

  async getTask(id: number): Promise<TranscriptionTaskDetail> {
    const raw: any = await callByID('GetTask', id);
    return {
      task: asTask(raw?.task),
      configJson: asStr(raw?.configJson),
      engine: asStr(raw?.engine),
      engineVersion: asStr(raw?.engineVersion),
      sampleRate: asNum(raw?.sampleRate),
      channels: asNum(raw?.channels),
      sourceHash: asStr(raw?.sourceHash),
      resultMidiPath: asOptStr(raw?.resultMidiPath),
      importedProjectId: asOptNum(raw?.importedProjectId),
      summaryJson: asOptStr(raw?.summaryJson),
      reportJson: asOptStr(raw?.reportJson),
      notes: Array.isArray(raw?.notes) ? raw.notes.map(asNote) : [],
      analysis: Array.isArray(raw?.analysis) ? raw.analysis.map(asAnalysis) : [],
    };
  },

  async cancelTask(id: number): Promise<void> {
    await callByID('CancelTask', id);
  },

  async retryTask(id: number, config: TranscriptionConfig): Promise<TranscriptionTask> {
    const result: any = await callByID('RetryTask', id, config);
    return asTask(result);
  },

  async deleteTask(id: number): Promise<void> {
    await callByID('DeleteTask', id);
  },

  async getCapability(): Promise<TranscriptionCapability> {
    const raw: any = await callByID('GetCapability');
    return {
      transcriptionEnabled: asBool(raw?.transcriptionEnabled),
      buildFlavor: asStr(raw?.buildFlavor),
      missingComponents: Array.isArray(raw?.missingComponents) ? raw.missingComponents : [],
    };
  },

  async setDownloadProxy(proxyAddress: string): Promise<void> {
    await callByID('SetDownloadProxy', proxyAddress);
  },

  async installFfmpeg(): Promise<string> {
    const result: any = await callByID('InstallFfmpeg');
    return asStr(result);
  },

  async installModel(): Promise<string> {
    const result: any = await callByID('InstallModel');
    return asStr(result);
  },

  async getConfig(): Promise<TranscriptionConfig> {
    const raw: any = await callByID('GetConfig');
    return {
      mode: (asStr(raw?.mode) || 'melody') as TranscriptionConfig['mode'],
      minConfidence: asNum(raw?.minConfidence) || 0.55,
      minDurationMs: asNum(raw?.minDurationMs) || 60,
      mergeGapMs: asNum(raw?.mergeGapMs) || 40,
      quantize: (asStr(raw?.quantize) || 'light') as TranscriptionConfig['quantize'],
      maxPolyphony: asNum(raw?.maxPolyphony) || 2,
      targetBaseNote: asNum(raw?.targetBaseNote) || 48,
      targetLaneCount: asNum(raw?.targetLaneCount) || 36,
      outOfRangePolicy: asStr(raw?.outOfRangePolicy) || 'drop',
      preferMelodyRegister: asBool(raw?.preferMelodyRegister),
    };
  },

  async updateConfig(config: TranscriptionConfig): Promise<void> {
    await callByID('UpdateConfig', config);
  },

  async importResultAsMidiProject(id: number): Promise<MidiProjectImportResult> {
    const raw: any = await callByID('ImportResultAsMidiProject', id);
    return {
      projectId: asNum(raw?.projectId),
      displayName: asStr(raw?.displayName),
      noteCount: asNum(raw?.noteCount),
      durationMs: asNum(raw?.durationMs),
      fileHash: asStr(raw?.fileHash),
    };
  },

  async exportResultMidi(id: number, targetPath: string): Promise<void> {
    await callByID('ExportResultMidi', id, targetPath);
  },

  // ===== 事件订阅 =====

  onProgress(handler: (event: TranscriptionProgress) => void): () => void {
    return Events.On(AppEvents.TranscriptionTaskProgress, (payload: WailsEventPayload<unknown>) => {
      const raw: any = payload?.data;
      handler({
        taskId: asStr(raw?.taskId),
        status: asStr(raw?.status),
        progress: asNum(raw?.progress),
        message: asOptStr(raw?.message),
      });
    });
  },

  onCompleted(handler: (event: TranscriptionResult) => void): () => void {
    return Events.On(AppEvents.TranscriptionTaskCompleted, (payload: WailsEventPayload<unknown>) => {
      const raw: any = payload?.data;
      handler({
        taskId: asStr(raw?.taskId),
        midiProjectId: asOptStr(raw?.midiProjectId),
        totalNotes: asNum(raw?.totalNotes),
        inRangeNotes: asNum(raw?.inRangeNotes),
        outRangeNotes: asNum(raw?.outRangeNotes),
        estimatedBpm: asNum(raw?.estimatedBpm),
        suggestedOctaveShift: asNum(raw?.suggestedOctaveShift),
        coveragePercent: asNum(raw?.coveragePercent),
        qualityReport: asOptStr(raw?.qualityReport),
      });
    });
  },

  onFailed(handler: (event: TranscriptionError) => void): () => void {
    return Events.On(AppEvents.TranscriptionTaskFailed, (payload: WailsEventPayload<unknown>) => {
      const raw: any = payload?.data;
      handler({
        taskId: asStr(raw?.taskId),
        errorCode: asStr(raw?.errorCode),
        errorMessage: asStr(raw?.errorMessage),
      });
    });
  },

  onCancelled(handler: (event: TranscriptionProgress) => void): () => void {
    return Events.On(AppEvents.TranscriptionTaskCancelled, () => {
      handler({ taskId: '', status: 'cancelled', progress: 0 });
    });
  },

  onFfmpegInstalled(handler: () => void): () => void {
    return Events.On(AppEvents.TranscriptionFfmpegInstalled, () => handler());
  },

  onFfmpegInstallFailed(handler: (error: string) => void): () => void {
    return Events.On(AppEvents.TranscriptionFfmpegInstallFailed, (payload: WailsEventPayload<unknown>) => {
      const raw: any = payload?.data;
      handler(String(raw?.error ?? ''));
    });
  },
};

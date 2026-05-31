// MidiService：MIDI 导入、曲库、配置详情、质量报告与 PlayPlan 的前端封装。
// View / ViewModel 不直接 import @bindings；所有后端字段在这里归一化为 camelCase。
import { Call } from '@wailsio/runtime';
import {
  ListProjectsRequest as BindingListProjectsRequest,
  MidiProfileDTO as BindingMidiProfileDTO,
  Service as Binding,
} from '@bindings/YyslsPlayer/internal/services/midi';

export type OutOfRangePolicy = 'drop' | 'octaveFold' | 'nearest';
export type KeyFrameAction = 'press' | 'release';

export interface ListProjectsRequest {
  query?: string;
  limit?: number;
  offset?: number;
}

export interface MidiProjectSummary {
  id: number;
  displayName: string;
  fileName: string;
  sourcePath: string;
  fileHash: string;
  ppq: number;
  trackCount: number;
  channelCount: number;
  durationMs: number;
  noteCount: number;
  defaultProfileId?: number;
  createdAt: number;
  updatedAt: number;
}

export interface MidiProfile {
  id: number;
  projectId?: number;
  name: string;
  baseNote: number;
  transpose: number;
  octaveShift: number;
  speed: number;
  outOfRangePolicy: OutOfRangePolicy;
  minPressMs: number;
  releaseGapMs: number;
  keymapProfileId: number;
  enabledTracks: number[] | null;
  createdAt: number;
  updatedAt: number;
}

export interface KeymapLane {
  id: number;
  profileId: number;
  profileName: string;
  lane: number;
  label: string;
  pitchClass: number;
  isBlackKey: boolean;
  virtualKey: number;
  scanCode: number;
  modifierKeysJson: string;
  displayOrder: number;
  createdAt: number;
  updatedAt: number;
}

export interface KeymapProfile {
  profileId: number;
  profileName: string;
  lanes: KeymapLane[];
}

export interface NoteRange {
  min: number;
  max: number;
}

export interface MappedRange {
  minLane: number;
  maxLane: number;
}

export interface QualityReport {
  noteRange: NoteRange;
  mappedRange: MappedRange;
  totalNotes: number;
  playableNotes: number;
  outOfRangeCount: number;
  droppedCount: number;
  foldedCount: number;
  clampedCount: number;
  blackKeyCount: number;
  playableRatio: number;
  chordDensity: number;
  trackCount: number;
  channelCount: number;
  suggestedTranspose: number;
  suggestedOctaveShift: number;
  warnings: string[];
}

export type ImportBatchStatus = 'imported' | 'skipped' | 'failed';

export interface ImportBatchItem {
  path: string;
  fileName: string;
  fileHash: string;
  projectId?: number;
  displayName: string;
  status: ImportBatchStatus;
  reason: string;
  error: string;
}

export interface ImportBatchResult {
  totalCount: number;
  importedCount: number;
  skippedCount: number;
  failedCount: number;
  firstProjectId?: number;
  lastProjectId?: number;
  firstImportedProjectId?: number;
  lastImportedProjectId?: number;
  items: ImportBatchItem[];
}

export interface MidiProjectDetail {
  project: MidiProjectSummary;
  defaultProfile: MidiProfile;
  profiles: MidiProfile[];
  defaultKeymap: KeymapProfile;
  qualityReport: QualityReport;
  eventCount: number;
  profileCount: number;
  playHistoryCount: number;
}

export interface MidiConfigSnapshot {
  baseNote: number;
  transpose: number;
  octaveShift: number;
  speed: number;
  outOfRangePolicy: OutOfRangePolicy;
  minPressMs: number;
  releaseGapMs: number;
  keymapProfileId: number;
  enabledTracks: number[] | null;
}

export interface KeyFrame {
  timeMs: number;
  action: KeyFrameAction;
  lane: number;
  sourceNote: number;
  normalizedNote: number;
  rawLane: number;
  velocity: number;
  key: KeymapLane;
}

export interface PlayPlan {
  projectId: number;
  profileId: number;
  durationMs: number;
  speed: number;
  baseNote: number;
  configSnapshot: MidiConfigSnapshot;
  frames: KeyFrame[];
  report: QualityReport;
}

type RawObject = Record<string, unknown>;
type DefaultProfileBinding = typeof Binding & {
  GetDefaultProfile?: () => Promise<unknown>;
  UpdateDefaultProfile?: (profile: BindingMidiProfileDTO) => Promise<unknown>;
  ResetDefaultProfile?: () => Promise<unknown>;
};
type BatchImportBinding = typeof Binding & {
  ImportFiles?: (paths: string[]) => Promise<unknown>;
  ImportDirectory?: (path: string) => Promise<unknown>;
};

const defaultProfileBinding = Binding as DefaultProfileBinding;
const batchImportBinding = Binding as BatchImportBinding;

const requireDefaultProfileBinding = <K extends keyof Pick<DefaultProfileBinding, 'GetDefaultProfile' | 'UpdateDefaultProfile' | 'ResetDefaultProfile'>>(name: K): NonNullable<DefaultProfileBinding[K]> => {
  const fn = defaultProfileBinding[name];
  if (!fn) {
    throw new Error(`Wails binding ${String(name)} is not generated yet`);
  }
  return fn as NonNullable<DefaultProfileBinding[K]>;
};

const callBindingByName = async (method: string, ...args: unknown[]): Promise<unknown> => {
  const names = [
    `YyslsPlayer/internal/services/midi.Service.${method}`,
    `YyslsPlayer/internal/services/midi.(*Service).${method}`,
    `YyslsPlayer/internal/services/midi.${method}`,
  ];
  let lastError: unknown;
  for (const name of names) {
    try {
      return await Call.ByName(name, ...args);
    } catch (error) {
      lastError = error;
    }
  }
  throw lastError;
};

const asObject = (value: unknown): RawObject => (value && typeof value === 'object' ? value as RawObject : {});
const asArray = <T>(value: unknown, map: (item: unknown) => T): T[] => Array.isArray(value) ? value.map(map) : [];
const asNumber = (value: unknown, fallback = 0): number => Number(value ?? fallback);
const asString = (value: unknown, fallback = ''): string => String(value ?? fallback);
const asBoolean = (value: unknown): boolean => Boolean(value);

const maybeNumber = (value: unknown): number | undefined => {
  if (value === null || value === undefined) return undefined;
  const n = Number(value);
  return Number.isFinite(n) ? n : undefined;
};

const normalizePolicy = (value: unknown): OutOfRangePolicy => {
  if (value === 'octaveFold' || value === 'nearest') return value;
  return 'drop';
};

const normalizeAction = (value: unknown): KeyFrameAction => value === 'release' ? 'release' : 'press';

const normalizeImportStatus = (value: unknown): ImportBatchStatus => {
  if (value === 'skipped' || value === 'failed') return value;
  return 'imported';
};

const normalizeEnabledTracks = (value: unknown): number[] | null => {
  if (value === null || value === undefined) return null;
  if (!Array.isArray(value)) return null;
  const seen = new Set<number>();
  const out: number[] = [];
  for (const item of value) {
    const track = Number(item);
    if (!Number.isInteger(track) || track < 0 || track > 127 || seen.has(track)) continue;
    seen.add(track);
    out.push(track);
  }
  return out.sort((a, b) => a - b);
};

const mapProject = (value: unknown): MidiProjectSummary => {
  const r = asObject(value);
  return {
    id: asNumber(r.id),
    displayName: asString(r.displayName),
    fileName: asString(r.fileName),
    sourcePath: asString(r.sourcePath),
    fileHash: asString(r.fileHash),
    ppq: asNumber(r.ppq),
    trackCount: asNumber(r.trackCount),
    channelCount: asNumber(r.channelCount),
    durationMs: asNumber(r.durationMs),
    noteCount: asNumber(r.noteCount),
    defaultProfileId: maybeNumber(r.defaultProfileId),
    createdAt: asNumber(r.createdAt),
    updatedAt: asNumber(r.updatedAt),
  };
};

const mapProfile = (value: unknown): MidiProfile => {
  const r = asObject(value);
  return {
    id: asNumber(r.id),
    projectId: maybeNumber(r.projectId),
    name: asString(r.name),
    baseNote: asNumber(r.baseNote, 48),
    transpose: asNumber(r.transpose),
    octaveShift: asNumber(r.octaveShift),
    speed: asNumber(r.speed, 1),
    outOfRangePolicy: normalizePolicy(r.outOfRangePolicy),
    minPressMs: asNumber(r.minPressMs, 35),
    releaseGapMs: asNumber(r.releaseGapMs, 15),
    keymapProfileId: asNumber(r.keymapProfileId, 1),
    enabledTracks: normalizeEnabledTracks(r.enabledTracks),
    createdAt: asNumber(r.createdAt),
    updatedAt: asNumber(r.updatedAt),
  };
};

const mapKeymapLane = (value: unknown): KeymapLane => {
  const r = asObject(value);
  return {
    id: asNumber(r.id),
    profileId: asNumber(r.profileId),
    profileName: asString(r.profileName),
    lane: asNumber(r.lane),
    label: asString(r.label),
    pitchClass: asNumber(r.pitchClass),
    isBlackKey: asBoolean(r.isBlackKey),
    virtualKey: asNumber(r.virtualKey),
    scanCode: asNumber(r.scanCode),
    modifierKeysJson: asString(r.modifierKeysJson, '[]'),
    displayOrder: asNumber(r.displayOrder),
    createdAt: asNumber(r.createdAt),
    updatedAt: asNumber(r.updatedAt),
  };
};

const mapKeymapProfile = (value: unknown): KeymapProfile => {
  const r = asObject(value);
  return {
    profileId: asNumber(r.profileId),
    profileName: asString(r.profileName),
    lanes: asArray(r.lanes, mapKeymapLane),
  };
};

const mapNoteRange = (value: unknown): NoteRange => {
  const r = asObject(value);
  return { min: asNumber(r.min, -1), max: asNumber(r.max, -1) };
};

const mapMappedRange = (value: unknown): MappedRange => {
  const r = asObject(value);
  return { minLane: asNumber(r.minLane, -1), maxLane: asNumber(r.maxLane, -1) };
};

const mapQualityReport = (value: unknown): QualityReport => {
  const r = asObject(value);
  return {
    noteRange: mapNoteRange(r.noteRange),
    mappedRange: mapMappedRange(r.mappedRange),
    totalNotes: asNumber(r.totalNotes),
    playableNotes: asNumber(r.playableNotes),
    outOfRangeCount: asNumber(r.outOfRangeCount),
    droppedCount: asNumber(r.droppedCount),
    foldedCount: asNumber(r.foldedCount),
    clampedCount: asNumber(r.clampedCount),
    blackKeyCount: asNumber(r.blackKeyCount),
    playableRatio: asNumber(r.playableRatio),
    chordDensity: asNumber(r.chordDensity),
    trackCount: asNumber(r.trackCount),
    channelCount: asNumber(r.channelCount),
    suggestedTranspose: asNumber(r.suggestedTranspose),
    suggestedOctaveShift: asNumber(r.suggestedOctaveShift),
    warnings: Array.isArray(r.warnings) ? r.warnings.map((w) => String(w)) : [],
  };
};

const mapBatchItem = (value: unknown): ImportBatchItem => {
  const r = asObject(value);
  return {
    path: asString(r.path),
    fileName: asString(r.fileName),
    fileHash: asString(r.fileHash),
    projectId: maybeNumber(r.projectId),
    displayName: asString(r.displayName),
    status: normalizeImportStatus(r.status),
    reason: asString(r.reason),
    error: asString(r.error),
  };
};

const mapBatchResult = (value: unknown): ImportBatchResult => {
  const r = asObject(value);
  return {
    totalCount: asNumber(r.totalCount),
    importedCount: asNumber(r.importedCount),
    skippedCount: asNumber(r.skippedCount),
    failedCount: asNumber(r.failedCount),
    firstProjectId: maybeNumber(r.firstProjectId),
    lastProjectId: maybeNumber(r.lastProjectId),
    firstImportedProjectId: maybeNumber(r.firstImportedProjectId),
    lastImportedProjectId: maybeNumber(r.lastImportedProjectId),
    items: asArray(r.items, mapBatchItem),
  };
};

const mapDetail = (value: unknown): MidiProjectDetail => {
  const r = asObject(value);
  return {
    project: mapProject(r.project),
    defaultProfile: mapProfile(r.defaultProfile),
    profiles: asArray(r.profiles, mapProfile),
    defaultKeymap: mapKeymapProfile(r.defaultKeymap),
    qualityReport: mapQualityReport(r.qualityReport),
    eventCount: asNumber(r.eventCount),
    profileCount: asNumber(r.profileCount),
    playHistoryCount: asNumber(r.playHistoryCount),
  };
};

const mapConfig = (value: unknown): MidiConfigSnapshot => {
  const r = asObject(value);
  return {
    baseNote: asNumber(r.baseNote, 48),
    transpose: asNumber(r.transpose),
    octaveShift: asNumber(r.octaveShift),
    speed: asNumber(r.speed, 1),
    outOfRangePolicy: normalizePolicy(r.outOfRangePolicy),
    minPressMs: asNumber(r.minPressMs, 35),
    releaseGapMs: asNumber(r.releaseGapMs, 15),
    keymapProfileId: asNumber(r.keymapProfileId, 1),
    enabledTracks: normalizeEnabledTracks(r.enabledTracks),
  };
};

const toBindingListProjectsRequest = (request: ListProjectsRequest): BindingListProjectsRequest =>
  new BindingListProjectsRequest({
    query: request.query ?? '',
    limit: request.limit ?? 0,
    offset: request.offset ?? 0,
  });

const toBindingMidiProfile = (profile: MidiProfile): BindingMidiProfileDTO => {
  const payload: Record<string, unknown> = {
    id: profile.id,
    projectId: profile.projectId ?? null,
    name: profile.name,
    baseNote: profile.baseNote,
    transpose: profile.transpose,
    octaveShift: profile.octaveShift,
    speed: profile.speed,
    outOfRangePolicy: profile.outOfRangePolicy,
    minPressMs: profile.minPressMs,
    releaseGapMs: profile.releaseGapMs,
    keymapProfileId: profile.keymapProfileId,
    enabledTracks: profile.enabledTracks,
    createdAt: profile.createdAt,
    updatedAt: profile.updatedAt,
  };
  return new BindingMidiProfileDTO(payload as Partial<BindingMidiProfileDTO>);
};

const mapFrame = (value: unknown): KeyFrame => {
  const r = asObject(value);
  return {
    timeMs: asNumber(r.timeMs),
    action: normalizeAction(r.action),
    lane: asNumber(r.lane),
    sourceNote: asNumber(r.sourceNote),
    normalizedNote: asNumber(r.normalizedNote),
    rawLane: asNumber(r.rawLane),
    velocity: asNumber(r.velocity),
    key: mapKeymapLane(r.key),
  };
};

const mapPlayPlan = (value: unknown): PlayPlan => {
  const r = asObject(value);
  return {
    projectId: asNumber(r.projectId),
    profileId: asNumber(r.profileId),
    durationMs: asNumber(r.durationMs),
    speed: asNumber(r.speed, 1),
    baseNote: asNumber(r.baseNote, 48),
    configSnapshot: mapConfig(r.configSnapshot),
    frames: asArray(r.frames, mapFrame),
    report: mapQualityReport(r.report),
  };
};

export const MidiService = {
  async importFile(path: string): Promise<MidiProjectDetail> {
    return mapDetail(await Binding.ImportFile(path));
  },

  async importFiles(paths: string[]): Promise<ImportBatchResult> {
    const fn = batchImportBinding.ImportFiles;
    return mapBatchResult(await (fn ? fn(paths) : callBindingByName('ImportFiles', paths)));
  },

  async importDirectory(path: string): Promise<ImportBatchResult> {
    const fn = batchImportBinding.ImportDirectory;
    return mapBatchResult(await (fn ? fn(path) : callBindingByName('ImportDirectory', path)));
  },

  async listProjects(request: ListProjectsRequest = {}): Promise<MidiProjectSummary[]> {
    return asArray(await Binding.ListProjects(toBindingListProjectsRequest(request)), mapProject);
  },

  async getProject(projectId: number): Promise<MidiProjectDetail> {
    return mapDetail(await Binding.GetProject(projectId));
  },

  async deleteProject(projectId: number): Promise<void> {
    await Binding.DeleteProject(projectId);
  },

  async updateProfile(profile: MidiProfile): Promise<MidiProfile> {
    return mapProfile(await Binding.UpdateProfile(toBindingMidiProfile(profile)));
  },

  async getDefaultProfile(): Promise<MidiProfile> {
    const fn = requireDefaultProfileBinding('GetDefaultProfile');
    return mapProfile(await fn());
  },

  async updateDefaultProfile(profile: MidiProfile): Promise<MidiProfile> {
    const fn = requireDefaultProfileBinding('UpdateDefaultProfile');
    return mapProfile(await fn(toBindingMidiProfile({ ...profile, projectId: undefined })));
  },

  async resetDefaultProfile(): Promise<MidiProfile> {
    const fn = requireDefaultProfileBinding('ResetDefaultProfile');
    return mapProfile(await fn());
  },

  async buildPlayPlan(projectId: number, profileId = 0): Promise<PlayPlan> {
    return mapPlayPlan(await Binding.BuildPlayPlan(projectId, profileId));
  },

  async getDefaultKeymap(): Promise<KeymapProfile> {
    return mapKeymapProfile(await Binding.GetDefaultKeymap());
  },
} as const;

import { Call, Events } from '@wailsio/runtime';
import { AppEvents, type WailsEventPayload } from '@/shared/events';
import { FEATURES } from '@/shared/featureFlags';

const METHOD_IDS = {
  CreateMacro: 1743859037,
  DeleteMacro: 1771404784,
  ExportMacro: 1436054307,
  GetMacro: 219203127,
  GetRecordState: 2105863525,
  GetState: 3517949026,
  ImportMacros: 1200929723,
  ListAssignableKeys: 2984877270,
  ListMacros: 3992847946,
  RunMacro: 1872574452,
  SaveMacro: 396519726,
  SetEnabled: 852229894,
  SetTrigger: 3975139451,
  StartRecording: 2402283796,
  StopMacro: 4010787021,
  StopRecording: 637258650,
  ValidateMacro: 1293516511,
} as const;

export type MacroStepKind =
  | 'delay'
  | 'keyTap'
  | 'keyDown'
  | 'keyUp'
  | 'chordTap'
  | 'mouseTap'
  | 'mouseDown'
  | 'mouseUp'
  | 'mouseScroll'
  | 'mouseMove'
  | 'text';
export type MacroRunState = 'idle' | 'running' | 'stopping' | 'completed' | 'stopped' | 'error';
export type MacroRepeatMode = 'once' | 'count' | 'hold' | 'toggle';
export type MacroDeviceKind = 'keyboard' | 'mouse' | '';
export type RecordRunState = 'idle' | 'recording' | 'stopped' | 'error';

export interface MacroSummary {
  id: number;
  name: string;
  description: string;
  triggerAccelerator: string;
  allowUnsafeTrigger: boolean;
  enabled: boolean;
  repeatMode: MacroRepeatMode;
  repeatCount: number;
  repeatIntervalMs: number;
  interruptPolicy: string;
  stepCount: number;
  registered: boolean;
  errorCode: string;
  createdAt: number;
  updatedAt: number;
}

export interface MacroStep {
  id: number;
  macroId: number;
  orderIndex: number;
  kind: MacroStepKind;
  keyLabel: string;
  virtualKey: number;
  scanCode: number;
  deviceKind: MacroDeviceKind;
  modifierKeysJson: string;
  durationMs: number;
  waitMs: number;
  payloadJson: string;
}

export interface MacroDetail {
  profile: MacroSummary;
  steps: MacroStep[];
}

export interface SaveMacroRequest {
  id: number;
  name: string;
  description: string;
  triggerAccelerator: string;
  allowUnsafeTrigger: boolean;
  enabled: boolean;
  repeatMode: MacroRepeatMode;
  repeatCount: number;
  repeatIntervalMs: number;
  interruptPolicy: string;
  steps: MacroStep[];
}

export interface MacroState {
  state: MacroRunState;
  macroId: number;
  macroName: string;
  stepIndex: number;
  stepCount: number;
  startedAt: number;
  updatedAt: number;
  errorCode: string;
  message: string;
}

export interface MacroStepEvent {
  macroId: number;
  stepIndex: number;
  step: MacroStep;
  at: number;
}

export interface MacroErrorEvent {
  macroId: number;
  errorCode: string;
  message: string;
  at: number;
}

export interface AssignableKey {
  label: string;
  virtualKey: number;
  scanCode: number;
  modifier: boolean;
  deviceKind: MacroDeviceKind;
}

export interface RecordState {
  state: RecordRunState;
  stepCount: number;
  startedAt: number;
  updatedAt: number;
  errorCode: string;
  message: string;
}

export interface RecordStepEvent {
  stepIndex: number;
  step: MacroStep;
  at: number;
}

export interface RecordResult {
  steps: MacroStep[];
  durationMs: number;
}

type RawObject = Record<string, unknown>;

const asObject = (value: unknown): RawObject => (value && typeof value === 'object' ? (value as RawObject) : {});
const asString = (value: unknown, fallback = ''): string => String(value ?? fallback);
const asNumber = (value: unknown, fallback = 0): number => Number(value ?? fallback);
const asBoolean = (value: unknown): boolean => Boolean(value);

const mapStep = (value: unknown): MacroStep => {
  const r = asObject(value);
  return {
    id: asNumber(r.id),
    macroId: asNumber(r.macroId),
    orderIndex: asNumber(r.orderIndex),
    kind: asString(r.kind, 'delay') as MacroStepKind,
    keyLabel: asString(r.keyLabel),
    virtualKey: asNumber(r.virtualKey),
    scanCode: asNumber(r.scanCode),
    deviceKind: asString(r.deviceKind) as MacroDeviceKind,
    modifierKeysJson: asString(r.modifierKeysJson, '[]'),
    durationMs: asNumber(r.durationMs),
    waitMs: asNumber(r.waitMs),
    payloadJson: asString(r.payloadJson, '{}'),
  };
};

const mapSummary = (value: unknown): MacroSummary => {
  const r = asObject(value);
  return {
    id: asNumber(r.id),
    name: asString(r.name),
    description: asString(r.description),
    triggerAccelerator: asString(r.triggerAccelerator),
    allowUnsafeTrigger: asBoolean(r.allowUnsafeTrigger),
    enabled: asBoolean(r.enabled),
    repeatMode: asString(r.repeatMode, 'once') as MacroRepeatMode,
    repeatCount: asNumber(r.repeatCount, 1),
    repeatIntervalMs: asNumber(r.repeatIntervalMs),
    interruptPolicy: asString(r.interruptPolicy, 'ignore'),
    stepCount: asNumber(r.stepCount),
    registered: asBoolean(r.registered),
    errorCode: asString(r.errorCode),
    createdAt: asNumber(r.createdAt),
    updatedAt: asNumber(r.updatedAt),
  };
};

const mapDetail = (value: unknown): MacroDetail => {
  const r = asObject(value);
  const steps = Array.isArray(r.steps) ? r.steps : [];
  return { profile: mapSummary(r.profile), steps: steps.map(mapStep) };
};

const mapState = (value: unknown): MacroState => {
  const r = asObject(value);
  return {
    state: asString(r.state, 'idle') as MacroRunState,
    macroId: asNumber(r.macroId),
    macroName: asString(r.macroName),
    stepIndex: asNumber(r.stepIndex, -1),
    stepCount: asNumber(r.stepCount),
    startedAt: asNumber(r.startedAt),
    updatedAt: asNumber(r.updatedAt),
    errorCode: asString(r.errorCode),
    message: asString(r.message),
  };
};

const mapStepEvent = (value: unknown): MacroStepEvent => {
  const r = asObject(value);
  return { macroId: asNumber(r.macroId), stepIndex: asNumber(r.stepIndex), step: mapStep(r.step), at: asNumber(r.at) };
};

const mapErrorEvent = (value: unknown): MacroErrorEvent => {
  const r = asObject(value);
  return { macroId: asNumber(r.macroId), errorCode: asString(r.errorCode), message: asString(r.message), at: asNumber(r.at) };
};

const mapKey = (value: unknown): AssignableKey => {
  const r = asObject(value);
  return {
    label: asString(r.label),
    virtualKey: asNumber(r.virtualKey),
    scanCode: asNumber(r.scanCode),
    modifier: asBoolean(r.modifier),
    deviceKind: asString(r.deviceKind) as MacroDeviceKind,
  };
};

const mapRecordState = (value: unknown): RecordState => {
  const r = asObject(value);
  return {
    state: asString(r.state, 'idle') as RecordRunState,
    stepCount: asNumber(r.stepCount),
    startedAt: asNumber(r.startedAt),
    updatedAt: asNumber(r.updatedAt),
    errorCode: asString(r.errorCode),
    message: asString(r.message),
  };
};

const mapRecordStepEvent = (value: unknown): RecordStepEvent => {
  const r = asObject(value);
  return { stepIndex: asNumber(r.stepIndex), step: mapStep(r.step), at: asNumber(r.at) };
};

const mapRecordResult = (value: unknown): RecordResult => {
  const r = asObject(value);
  const steps = Array.isArray(r.steps) ? r.steps : [];
  return { steps: steps.map(mapStep), durationMs: asNumber(r.durationMs) };
};

const toBackendStep = (step: MacroStep): MacroStep => ({
  id: step.id,
  macroId: step.macroId,
  orderIndex: step.orderIndex,
  kind: step.kind,
  keyLabel: step.keyLabel,
  virtualKey: step.virtualKey,
  scanCode: step.scanCode,
  deviceKind: step.deviceKind,
  modifierKeysJson: step.modifierKeysJson,
  durationMs: step.durationMs,
  waitMs: step.waitMs,
  payloadJson: step.payloadJson,
});

const toBackendRequest = (req: SaveMacroRequest): SaveMacroRequest => ({
  id: req.id,
  name: req.name,
  description: req.description,
  triggerAccelerator: req.triggerAccelerator,
  allowUnsafeTrigger: req.allowUnsafeTrigger,
  enabled: req.enabled,
  repeatMode: req.repeatMode,
  repeatCount: req.repeatCount,
  repeatIntervalMs: req.repeatIntervalMs,
  interruptPolicy: req.interruptPolicy,
  steps: req.steps.map(toBackendStep),
});

export const MacroService = {
  get enabled(): boolean {
    return FEATURES.macros;
  },

  async listMacros(): Promise<MacroSummary[]> {
    const rows = await Call.ByID(METHOD_IDS.ListMacros);
    return (Array.isArray(rows) ? rows : []).map(mapSummary);
  },

  async getMacro(id: number): Promise<MacroDetail> {
    return mapDetail(await Call.ByID(METHOD_IDS.GetMacro, id));
  },

  async createMacro(name: string): Promise<MacroDetail> {
    return mapDetail(await Call.ByID(METHOD_IDS.CreateMacro, name));
  },

  async saveMacro(req: SaveMacroRequest): Promise<MacroDetail> {
    return mapDetail(await Call.ByID(METHOD_IDS.SaveMacro, toBackendRequest(req)));
  },

  async deleteMacro(id: number): Promise<void> {
    await Call.ByID(METHOD_IDS.DeleteMacro, id);
  },

  async exportMacro(id: number, targetPath: string): Promise<void> {
    await Call.ByID(METHOD_IDS.ExportMacro, id, targetPath);
  },

  async importMacros(sourcePath: string): Promise<MacroSummary[]> {
    const rows = await Call.ByID(METHOD_IDS.ImportMacros, sourcePath);
    return (Array.isArray(rows) ? rows : []).map(mapSummary);
  },

  async setEnabled(id: number, enabled: boolean): Promise<MacroDetail> {
    return mapDetail(await Call.ByID(METHOD_IDS.SetEnabled, id, enabled));
  },

  async setTrigger(id: number, accelerator: string, allowUnsafe: boolean): Promise<MacroDetail> {
    return mapDetail(await Call.ByID(METHOD_IDS.SetTrigger, id, accelerator, allowUnsafe));
  },

  async runMacro(id: number): Promise<MacroState> {
    return mapState(await Call.ByID(METHOD_IDS.RunMacro, id));
  },

  async stopMacro(): Promise<MacroState> {
    return mapState(await Call.ByID(METHOD_IDS.StopMacro));
  },

  async getState(): Promise<MacroState> {
    return mapState(await Call.ByID(METHOD_IDS.GetState));
  },

  async listAssignableKeys(): Promise<AssignableKey[]> {
    const rows = await Call.ByID(METHOD_IDS.ListAssignableKeys);
    return (Array.isArray(rows) ? rows : []).map(mapKey);
  },

  async startRecording(captureDelays: boolean, captureMoves: boolean): Promise<RecordState> {
    return mapRecordState(await Call.ByID(METHOD_IDS.StartRecording, captureDelays, captureMoves));
  },

  async stopRecording(): Promise<RecordResult> {
    return mapRecordResult(await Call.ByID(METHOD_IDS.StopRecording));
  },

  async getRecordState(): Promise<RecordState> {
    return mapRecordState(await Call.ByID(METHOD_IDS.GetRecordState));
  },

  onState(handler: (event: MacroState) => void): () => void {
    return Events.On(AppEvents.MacroState, (event: WailsEventPayload<unknown>) => handler(mapState(event?.data)));
  },

  onStep(handler: (event: MacroStepEvent) => void): () => void {
    return Events.On(AppEvents.MacroStep, (event: WailsEventPayload<unknown>) => handler(mapStepEvent(event?.data)));
  },

  onError(handler: (event: MacroErrorEvent) => void): () => void {
    return Events.On(AppEvents.MacroError, (event: WailsEventPayload<unknown>) => handler(mapErrorEvent(event?.data)));
  },

  onRecordState(handler: (event: RecordState) => void): () => void {
    return Events.On(AppEvents.MacroRecordState, (event: WailsEventPayload<unknown>) => handler(mapRecordState(event?.data)));
  },

  onRecordStep(handler: (event: RecordStepEvent) => void): () => void {
    return Events.On(AppEvents.MacroRecordStep, (event: WailsEventPayload<unknown>) => handler(mapRecordStepEvent(event?.data)));
  },
} as const;

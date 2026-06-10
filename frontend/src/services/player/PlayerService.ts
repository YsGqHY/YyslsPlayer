// PlayerService：游戏演奏控制与 Wails player:* 事件订阅封装。
// View / ViewModel 不直接 import @bindings 或 @wailsio/runtime Events。
import { Call, Events } from '@wailsio/runtime';
import { Service as Binding } from '@bindings/YyslsPlayer/internal/services/player';
import { PlayPlanDTO } from '@bindings/YyslsPlayer/internal/services/midi';
import { AppEvents, type WailsEventPayload } from '@/shared/events';
import type { PlayPlan } from '@/services/midi/MidiService';

export type PlayerState = 'idle' | 'ready' | 'playing' | 'paused' | 'completed' | 'stopped' | 'error';

export interface StartPlayerRequest {
  plan: PlayPlan;
  lookaheadMs?: number;
  startPositionMs?: number;
  startDelayMs?: number;
}

export interface PlayerSession {
  sessionId: string;
  state: PlayerState;
  positionMs: number;
  durationMs: number;
  dryRun: boolean;
  lookaheadMs: number;
  errorCode: string;
  message: string;
  projectId: number;
  profileId: number;
  frameCount: number;
  startedAt: number;
  updatedAt: number;
}

export interface PlayerStateSnapshot {
  sessionId: string;
  state: PlayerState;
  positionMs: number;
  durationMs: number;
  dryRun: boolean;
  lookaheadMs: number;
  errorCode: string;
  message: string;
}

export interface PlayerPositionEvent {
  sessionId: string;
  state: PlayerState;
  positionMs: number;
  durationMs: number;
  progress: number;
  updatedAt: number;
}

export interface PlayerErrorEvent extends PlayerStateSnapshot {
  updatedAt: number;
}

type RawObject = Record<string, unknown>;

const asObject = (value: unknown): RawObject => (value && typeof value === 'object' ? value as RawObject : {});
const asNumber = (value: unknown, fallback = 0): number => Number(value ?? fallback);
const asString = (value: unknown, fallback = ''): string => String(value ?? fallback);
const asBoolean = (value: unknown): boolean => Boolean(value);

const normalizeState = (value: unknown): PlayerState => {
  switch (value) {
    case 'ready':
    case 'playing':
    case 'paused':
    case 'completed':
    case 'stopped':
    case 'error':
      return value;
    default:
      return 'idle';
  }
};

const mapState = (value: unknown): PlayerStateSnapshot => {
  const r = asObject(value);
  return {
    sessionId: asString(r.sessionId),
    state: normalizeState(r.state),
    positionMs: asNumber(r.positionMs),
    durationMs: asNumber(r.durationMs),
    dryRun: asBoolean(r.dryRun),
    lookaheadMs: asNumber(r.lookaheadMs, 20),
    errorCode: asString(r.errorCode),
    message: asString(r.message),
  };
};

const mapSession = (value: unknown): PlayerSession => {
  const r = asObject(value);
  return {
    ...mapState(r),
    projectId: asNumber(r.projectId),
    profileId: asNumber(r.profileId),
    frameCount: asNumber(r.frameCount),
    startedAt: asNumber(r.startedAt),
    updatedAt: asNumber(r.updatedAt),
  };
};

const mapPosition = (value: unknown): PlayerPositionEvent => {
  const r = asObject(value);
  return {
    sessionId: asString(r.sessionId),
    state: normalizeState(r.state),
    positionMs: asNumber(r.positionMs),
    durationMs: asNumber(r.durationMs),
    progress: Math.max(0, Math.min(1, asNumber(r.progress))),
    updatedAt: asNumber(r.updatedAt),
  };
};

const mapError = (value: unknown): PlayerErrorEvent => {
  const r = asObject(value);
  return {
    ...mapState(r),
    updatedAt: asNumber(r.updatedAt),
  };
};

const subscribe = <T>(name: string, map: (value: unknown) => T, handler: (event: T) => void): (() => void) => {
  return Events.On(name, (event: WailsEventPayload<unknown>) => {
    handler(map(event?.data));
  });
};

const callBindingByName = async (method: string, ...args: unknown[]): Promise<unknown> => {
  const names = [
    `YyslsPlayer/internal/services/player.Service.${method}`,
    `YyslsPlayer/internal/services/player.(*Service).${method}`,
    `YyslsPlayer/internal/services/player.${method}`,
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

const toBindingPlayPlan = (plan: PlayPlan): PlayPlanDTO => new PlayPlanDTO({
  ...plan,
  configSnapshot: {
    ...plan.configSnapshot,
    enabledTracks: plan.configSnapshot.enabledTracks,
  },
});

export const PlayerService = {
  async start(req: StartPlayerRequest): Promise<PlayerSession> {
    const payload: Parameters<typeof Binding.Start>[0] & { startPositionMs?: number; startDelayMs?: number } = {
      plan: toBindingPlayPlan(req.plan),
      dryRun: false,
      lookaheadMs: req.lookaheadMs ?? 20,
      startPositionMs: req.startPositionMs ?? 0,
      startDelayMs: req.startDelayMs ?? 0,
    };
    return mapSession(await Binding.Start(payload));
  },

  async pause(sessionId: string): Promise<PlayerStateSnapshot> {
    return mapState(await Binding.Pause(sessionId));
  },

  async resume(sessionId: string): Promise<PlayerStateSnapshot> {
    return mapState(await Binding.Resume(sessionId));
  },

  async seek(sessionId: string, positionMs: number): Promise<PlayerStateSnapshot> {
    const dynamicBinding = Binding as typeof Binding & { Seek?: (sessionId: string, positionMs: number) => Promise<unknown> };
    if (typeof dynamicBinding.Seek === 'function') {
      return mapState(await dynamicBinding.Seek(sessionId, positionMs));
    }
    return mapState(await callBindingByName('Seek', sessionId, positionMs));
  },

  async stop(sessionId: string): Promise<PlayerStateSnapshot> {
    return mapState(await Binding.Stop(sessionId));
  },

  async getState(sessionId = ''): Promise<PlayerStateSnapshot> {
    return mapState(await Binding.GetState(sessionId));
  },

  async releaseAll(): Promise<void> {
    await Binding.ReleaseAll();
  },

  onState(handler: (event: PlayerStateSnapshot) => void): () => void {
    return subscribe(AppEvents.PlayerState, mapState, handler);
  },

  onPosition(handler: (event: PlayerPositionEvent) => void): () => void {
    return subscribe(AppEvents.PlayerPosition, mapPosition, handler);
  },

  onError(handler: (event: PlayerErrorEvent) => void): () => void {
    return subscribe(AppEvents.PlayerError, mapError, handler);
  },
} as const;

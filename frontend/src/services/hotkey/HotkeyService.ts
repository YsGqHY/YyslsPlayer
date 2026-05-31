// HotkeyService：全局快捷键配置与 hotkey:triggered 事件订阅封装。
// View / ViewModel 不直接 import @bindings 或 @wailsio/runtime Events。
import { Events } from '@wailsio/runtime';
import { Service as Binding } from '@bindings/YyslsPlayer/internal/services/hotkey';
import { AppEvents, type WailsEventPayload } from '@/shared/events';

// 动作 ID —— 与后端 internal/services/hotkey/types.go 对齐。
export type HotkeyAction =
  | 'previous-track'
  | 'next-track'
  | 'play-pause'
  | 'stop'
  | 'preview-toggle'
  | 'emergency-release';

export interface HotkeyBinding {
  actionId: string;
  accelerator: string;
  enabled: boolean;
  // 是否已成功注册到 OS；enabled 但 registered=false 通常意味着被占用。
  registered: boolean;
  // 注册失败原因码（空表示无错误）。
  errorCode: string;
}

export interface HotkeyState {
  // Windows 热键管理器是否可用。
  supported: boolean;
  bindings: HotkeyBinding[];
}

export interface HotkeyTriggeredEvent {
  actionId: string;
  accelerator: string;
  // 后端是否已直接执行（stop / 紧急松键 / 暂停继续）。
  handledByBackend: boolean;
  at: number;
}

type RawObject = Record<string, unknown>;

const asObject = (value: unknown): RawObject =>
  value && typeof value === 'object' ? (value as RawObject) : {};
const asString = (value: unknown, fallback = ''): string => String(value ?? fallback);
const asBoolean = (value: unknown): boolean => Boolean(value);
const asNumber = (value: unknown, fallback = 0): number => Number(value ?? fallback);

const mapBinding = (value: unknown): HotkeyBinding => {
  const r = asObject(value);
  return {
    actionId: asString(r.actionId),
    accelerator: asString(r.accelerator),
    enabled: asBoolean(r.enabled),
    registered: asBoolean(r.registered),
    errorCode: asString(r.errorCode),
  };
};

const mapState = (value: unknown): HotkeyState => {
  const r = asObject(value);
  const list = Array.isArray(r.bindings) ? r.bindings : [];
  return {
    supported: asBoolean(r.supported),
    bindings: list.map(mapBinding),
  };
};

const mapTriggered = (value: unknown): HotkeyTriggeredEvent => {
  const r = asObject(value);
  return {
    actionId: asString(r.actionId),
    accelerator: asString(r.accelerator),
    handledByBackend: asBoolean(r.handledByBackend),
    at: asNumber(r.at),
  };
};

export const HotkeyService = {
  async getState(): Promise<HotkeyState> {
    return mapState(await Binding.GetState());
  },

  async setBinding(actionId: string, accelerator: string): Promise<HotkeyState> {
    return mapState(await Binding.SetBinding(actionId, accelerator));
  },

  async setEnabled(actionId: string, enabled: boolean): Promise<HotkeyState> {
    return mapState(await Binding.SetEnabled(actionId, enabled));
  },

  async reset(): Promise<HotkeyState> {
    return mapState(await Binding.Reset());
  },

  // 订阅 OS 热键触发事件；返回取消订阅函数。
  onTriggered(handler: (event: HotkeyTriggeredEvent) => void): () => void {
    return Events.On(AppEvents.HotkeyTriggered, (event: WailsEventPayload<unknown>) => {
      handler(mapTriggered(event?.data));
    });
  },
} as const;

import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { HotkeyService, recordFromEvent, type HotkeyBinding, type HotkeyState } from '@/services';

// 录制状态：哪个动作正在录制 + 最近一次本地校验结果（不安全时给提示，不写回后端）。
export interface RecordingState {
  actionId: string;
  // 'safe' 表示已捕获到安全组合（等待写入）；'unsafe' / 'invalid' 为本地校验失败原因
  hint: 'listening' | 'unsafe' | 'invalid' | null;
}

export interface ShortcutRow extends HotkeyBinding {
  // 是否与其它启用项的组合冲突（前端按文本判定，做轻量提示）。
  conflict: boolean;
}

export interface UseShortcutsResult {
  supported: boolean;
  rows: ShortcutRow[];
  loading: boolean;
  error: string | null;
  recordingActionId: string | null;
  recordingHint: RecordingState['hint'];
  busy: boolean;
  startRecording: (actionId: string) => void;
  cancelRecording: () => void;
  setEnabled: (actionId: string, enabled: boolean) => void;
  reset: () => void;
}

const errMsg = (e: unknown): string => (e instanceof Error ? e.message : String(e ?? ''));

// 计算与其它启用项冲突的动作 ID 集合。
const computeConflicts = (bindings: HotkeyBinding[]): Set<string> => {
  const byText = new Map<string, string[]>();
  for (const b of bindings) {
    if (!b.enabled) continue;
    const list = byText.get(b.accelerator) ?? [];
    list.push(b.actionId);
    byText.set(b.accelerator, list);
  }
  const conflicts = new Set<string>();
  for (const ids of byText.values()) {
    if (ids.length > 1) ids.forEach((id) => conflicts.add(id));
  }
  return conflicts;
};

export const useShortcuts = (): UseShortcutsResult => {
  const aliveRef = useRef(true);
  const [state, setState] = useState<HotkeyState>({ supported: false, bindings: [] });
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [recording, setRecording] = useState<RecordingState | null>(null);

  useEffect(() => {
    aliveRef.current = true;
    return () => {
      aliveRef.current = false;
    };
  }, []);

  const reload = useCallback(async (): Promise<void> => {
    setLoading(true);
    setError(null);
    try {
      const next = await HotkeyService.getState();
      if (!aliveRef.current) return;
      setState(next);
    } catch (e: unknown) {
      if (!aliveRef.current) return;
      setError(errMsg(e));
    } finally {
      if (aliveRef.current) setLoading(false);
    }
  }, []);

  useEffect(() => {
    void reload();
  }, [reload]);

  const cancelRecording = useCallback((): void => {
    setRecording(null);
  }, []);

  const startRecording = useCallback((actionId: string): void => {
    setError(null);
    setRecording({ actionId, hint: 'listening' });
  }, []);

  const setEnabled = useCallback((actionId: string, enabled: boolean): void => {
    setBusy(true);
    void HotkeyService.setEnabled(actionId, enabled)
      .then((next) => {
        if (aliveRef.current) setState(next);
      })
      .catch((e: unknown) => {
        if (aliveRef.current) setError(errMsg(e));
      })
      .finally(() => {
        if (aliveRef.current) setBusy(false);
      });
  }, []);

  const reset = useCallback((): void => {
    setBusy(true);
    setRecording(null);
    void HotkeyService.reset()
      .then((next) => {
        if (aliveRef.current) setState(next);
      })
      .catch((e: unknown) => {
        if (aliveRef.current) setError(errMsg(e));
      })
      .finally(() => {
        if (aliveRef.current) setBusy(false);
      });
  }, []);

  // 录制阶段：在 window 上监听一次 keydown，捕获组合并写回后端。
  useEffect(() => {
    if (!recording) return;
    const onKeyDown = (event: KeyboardEvent): void => {
      event.preventDefault();
      event.stopPropagation();
      const recorded = recordFromEvent(event);
      if (recorded.modifiersOnly) {
        // 仅按下修饰键，继续等待主键。
        setRecording((prev) => (prev ? { ...prev, hint: 'listening' } : prev));
        return;
      }
      if (event.code === 'Escape') {
        setRecording(null);
        return;
      }
      if (!recorded.accelerator) {
        setRecording((prev) => (prev ? { ...prev, hint: 'invalid' } : prev));
        return;
      }
      if (!recorded.safe) {
        setRecording((prev) => (prev ? { ...prev, hint: 'unsafe' } : prev));
        return;
      }
      const actionId = recording.actionId;
      const accel = recorded.accelerator;
      setRecording(null);
      setBusy(true);
      void HotkeyService.setBinding(actionId, accel)
        .then((next) => {
          if (aliveRef.current) setState(next);
        })
        .catch((e: unknown) => {
          if (aliveRef.current) setError(errMsg(e));
        })
        .finally(() => {
          if (aliveRef.current) setBusy(false);
        });
    };
    window.addEventListener('keydown', onKeyDown, true);
    return () => {
      window.removeEventListener('keydown', onKeyDown, true);
    };
  }, [recording]);

  const rows = useMemo<ShortcutRow[]>(() => {
    const conflicts = computeConflicts(state.bindings);
    return state.bindings.map((b) => ({ ...b, conflict: conflicts.has(b.actionId) }));
  }, [state.bindings]);

  return {
    supported: state.supported,
    rows,
    loading,
    error,
    recordingActionId: recording?.actionId ?? null,
    recordingHint: recording?.hint ?? null,
    busy,
    startRecording,
    cancelRecording,
    setEnabled,
    reset,
  };
};

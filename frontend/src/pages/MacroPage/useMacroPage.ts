import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useT } from '@/i18n';
import {
  MacroService,
  recordFromEvent,
  type AssignableKey,
  type MacroDetail,
  type MacroRepeatMode,
  type MacroRunState,
  type MacroStep,
  type MacroStepKind,
  type MacroSummary,
  type RecordRunState,
  type SaveMacroRequest,
} from '@/services';

export interface DraftMacro {
  id: number;
  name: string;
  description: string;
  triggerAccelerator: string;
  enabled: boolean;
  repeatMode: MacroRepeatMode;
  repeatCount: number;
  repeatIntervalMs: number;
  interruptPolicy: string;
  steps: MacroStep[];
}

export interface UseMacroPageResult {
  macros: MacroSummary[];
  activeId: number;
  draft: DraftMacro | null;
  keys: AssignableKey[];
  selectedStepIndex: number;
  recordingTrigger: boolean;
  recordingHint: 'listening' | 'unsafe' | 'invalid' | null;
  recordState: RecordRunState;
  recordStepCount: number;
  captureDelays: boolean;
  captureMoves: boolean;
  runningState: MacroRunState;
  runningMacroId: number;
  runningStepIndex: number;
  loading: boolean;
  busy: boolean;
  dirty: boolean;
  error: string | null;
  selectMacro: (id: number) => void;
  createMacro: (name: string) => void;
  saveDraft: () => void;
  deleteActive: () => void;
  runActive: () => void;
  stopRunning: () => void;
  setEnabled: (enabled: boolean) => void;
  updateDraft: (patch: Partial<DraftMacro>) => void;
  selectStep: (index: number) => void;
  addStep: (kind: MacroStepKind) => void;
  removeStep: (index: number) => void;
  moveStep: (index: number, direction: -1 | 1) => void;
  reorderStep: (fromIndex: number, toIndex: number) => void;
  duplicateStep: (index: number) => void;
  updateStep: (index: number, patch: Partial<MacroStep>) => void;
  startTriggerRecording: () => void;
  cancelTriggerRecording: () => void;
  startStepRecording: () => void;
  stopStepRecording: () => void;
  setCaptureDelays: (value: boolean) => void;
  setCaptureMoves: (value: boolean) => void;
}

const errMsg = (e: unknown): string => (e instanceof Error ? e.message : String(e ?? ''));

// resolveRuntimeError turns a backend macro error event into a human sentence.
// Hotkey-triggered failures arrive as a stable errorCode (e.g. MACRO_PLAYER_ACTIVE);
// we prefer the localized runtimeErrors copy and fall back to the raw message /
// code when no translation exists (t returns the key itself on a miss).
const resolveRuntimeError = (
  t: ReturnType<typeof useT>,
  errorCode: string,
  message: string,
): string => {
  if (errorCode) {
    const key = `settings.macros.runtimeErrors.${errorCode}`;
    const translated = t(key);
    if (translated !== key) return translated;
  }
  return message || errorCode || t('settings.macros.runtimeErrors.MACRO_ERROR');
};

const isMouseKind = (kind: MacroStepKind): boolean =>
  kind === 'mouseTap' || kind === 'mouseDown' || kind === 'mouseUp' || kind === 'mouseScroll';

const emptyStep = (kind: MacroStepKind, key?: AssignableKey): MacroStep => {
  if (kind === 'text') {
    // Text block stores its Unicode payload in payloadJson; no key/duration.
    return {
      id: 0,
      macroId: 0,
      orderIndex: 0,
      kind,
      keyLabel: '',
      virtualKey: 0,
      scanCode: 0,
      deviceKind: 'keyboard',
      modifierKeysJson: '[]',
      durationMs: 0,
      waitMs: 0,
      payloadJson: JSON.stringify({ text: '' }),
    };
  }
  if (kind === 'mouseScroll') {
    // Scroll defaults to one Wheel Up notch (keysim wheel id 6); one-shot, no
    // duration or release.
    return {
      id: 0,
      macroId: 0,
      orderIndex: 0,
      kind,
      keyLabel: 'Wheel Up',
      virtualKey: 6,
      scanCode: 0,
      deviceKind: 'mouse',
      modifierKeysJson: '[]',
      durationMs: 0,
      waitMs: 0,
      payloadJson: '{}',
    };
  }
  if (kind === 'mouseMove') {
    // Relative cursor move; dx/dy live in payloadJson, no key/button. Duration
    // mirrors the backend nominal move cost (mouseMoveDurationMs).
    return {
      id: 0,
      macroId: 0,
      orderIndex: 0,
      kind,
      keyLabel: '',
      virtualKey: 0,
      scanCode: 0,
      deviceKind: 'mouse',
      modifierKeysJson: '[]',
      durationMs: 100,
      waitMs: 0,
      payloadJson: JSON.stringify({ dx: 100, dy: 0, jitter: 0 }),
    };
  }
  if (isMouseKind(kind)) {
    return {
      id: 0,
      macroId: 0,
      orderIndex: 0,
      kind,
      keyLabel: 'Mouse Left',
      virtualKey: 1,
      scanCode: 0,
      deviceKind: 'mouse',
      modifierKeysJson: '[]',
      durationMs: kind === 'mouseTap' ? 40 : 0,
      waitMs: 0,
      payloadJson: '{}',
    };
  }
  return {
    id: 0,
    macroId: 0,
    orderIndex: 0,
    kind,
    keyLabel: kind === 'delay' ? '' : key?.label ?? 'A',
    virtualKey: kind === 'delay' ? 0 : key?.virtualKey ?? 65,
    scanCode: kind === 'delay' ? 0 : key?.scanCode ?? 30,
    deviceKind: kind === 'delay' ? '' : 'keyboard',
    modifierKeysJson: kind === 'chordTap' ? '[{"label":"Ctrl","virtualKey":17,"scanCode":29}]' : '[]',
    durationMs: kind === 'keyTap' || kind === 'chordTap' ? 40 : 0,
    waitMs: kind === 'delay' ? 100 : 0,
    payloadJson: '{}',
  };
};

const detailToDraft = (detail: MacroDetail): DraftMacro => ({
  id: detail.profile.id,
  name: detail.profile.name,
  description: detail.profile.description,
  triggerAccelerator: detail.profile.triggerAccelerator,
  enabled: detail.profile.enabled,
  repeatMode: detail.profile.repeatMode || 'once',
  repeatCount: detail.profile.repeatCount || 1,
  repeatIntervalMs: detail.profile.repeatIntervalMs || 0,
  interruptPolicy: detail.profile.interruptPolicy || 'ignore',
  steps: detail.steps.map((s, index) => ({ ...s, orderIndex: index })),
});

const draftToRequest = (draft: DraftMacro): SaveMacroRequest => ({
  id: draft.id,
  name: draft.name,
  description: draft.description,
  triggerAccelerator: draft.triggerAccelerator,
  enabled: draft.enabled,
  repeatMode: draft.repeatMode,
  repeatCount: draft.repeatCount,
  repeatIntervalMs: draft.repeatIntervalMs,
  interruptPolicy: draft.interruptPolicy || 'ignore',
  steps: draft.steps.map((s, index) => ({ ...s, orderIndex: index })),
});

const draftSignature = (draft: DraftMacro | null): string => JSON.stringify(draft ?? null);

export const useMacroPage = (): UseMacroPageResult => {
  const t = useT();
  const aliveRef = useRef(true);
  const savedSigRef = useRef('null');
  const [macros, setMacros] = useState<MacroSummary[]>([]);
  const [activeId, setActiveId] = useState(0);
  const [draft, setDraft] = useState<DraftMacro | null>(null);
  const [keys, setKeys] = useState<AssignableKey[]>([]);
  const [selectedStepIndex, setSelectedStepIndex] = useState(-1);
  const [recordingTrigger, setRecordingTrigger] = useState(false);
  const [recordingHint, setRecordingHint] = useState<UseMacroPageResult['recordingHint']>(null);
  const [runningState, setRunningState] = useState<MacroRunState>('idle');
  const [runningMacroId, setRunningMacroId] = useState(0);
  const [runningStepIndex, setRunningStepIndex] = useState(-1);
  const [recordState, setRecordState] = useState<RecordRunState>('idle');
  const [recordStepCount, setRecordStepCount] = useState(0);
  const [captureDelays, setCaptureDelays] = useState(true);
  const [captureMoves, setCaptureMoves] = useState(false);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    aliveRef.current = true;
    return () => {
      aliveRef.current = false;
    };
  }, []);

  const loadDetail = useCallback(async (id: number): Promise<void> => {
    const detail = await MacroService.getMacro(id);
    if (!aliveRef.current) return;
    const nextDraft = detailToDraft(detail);
    setActiveId(id);
    setDraft(nextDraft);
    savedSigRef.current = draftSignature(nextDraft);
    setSelectedStepIndex(nextDraft.steps.length > 0 ? 0 : -1);
  }, []);

  const reload = useCallback(async (): Promise<void> => {
    setLoading(true);
    setError(null);
    try {
      const [list, keyRows, state] = await Promise.all([
        MacroService.listMacros(),
        MacroService.listAssignableKeys(),
        MacroService.getState(),
      ]);
      if (!aliveRef.current) return;
      setMacros(list);
      setKeys(keyRows);
      setRunningState(state.state);
      setRunningMacroId(state.macroId);
      setRunningStepIndex(state.stepIndex);
      const nextID = activeId || list[0]?.id || 0;
      if (nextID) {
        await loadDetail(nextID);
      } else {
        setDraft(null);
        savedSigRef.current = 'null';
      }
    } catch (e: unknown) {
      if (aliveRef.current) setError(errMsg(e));
    } finally {
      if (aliveRef.current) setLoading(false);
    }
  }, [activeId, loadDetail]);

  useEffect(() => {
    void reload();
  }, [reload]);

  useEffect(() => {
    const offState = MacroService.onState((state) => {
      setRunningState(state.state);
      setRunningMacroId(state.macroId);
      setRunningStepIndex(state.stepIndex);
    });
    const offStep = MacroService.onStep((event) => {
      setRunningMacroId(event.macroId);
      setRunningStepIndex(event.stepIndex);
    });
    const offError = MacroService.onError((event) => {
      setError(resolveRuntimeError(t, event.errorCode, event.message));
      // A blocked/failed trigger means OS-level registration succeeded (the key
      // fired) but execution was rejected; refresh the list so any registration
      // state badge stays in sync.
      void MacroService.listMacros().then((list) => {
        if (aliveRef.current) setMacros(list);
      }).catch(() => undefined);
    });
    const offRecordState = MacroService.onRecordState((state) => {
      setRecordState(state.state);
      setRecordStepCount(state.stepCount);
      if (state.state === 'error') setError(resolveRuntimeError(t, state.errorCode, state.message));
    });
    const offRecordStep = MacroService.onRecordStep((event) => {
      setRecordStepCount(event.stepIndex + 1);
      setDraft((prev) => {
        if (!prev) return prev;
        const steps = [...prev.steps, { ...event.step, id: 0, macroId: 0, orderIndex: prev.steps.length }];
        setSelectedStepIndex(steps.length - 1);
        return { ...prev, steps };
      });
    });
    return () => {
      offState();
      offStep();
      offError();
      offRecordState();
      offRecordStep();
    };
  }, [t]);

  useEffect(() => {
    if (!recordingTrigger) return;
    const onKeyDown = (event: KeyboardEvent): void => {
      event.preventDefault();
      event.stopPropagation();
      if (event.code === 'Escape' && !event.ctrlKey && !event.altKey && !event.shiftKey && !event.metaKey) {
        setRecordingTrigger(false);
        setRecordingHint(null);
        return;
      }
      const recorded = recordFromEvent(event);
      if (recorded.modifiersOnly) {
        setRecordingHint('listening');
        return;
      }
      if (!recorded.accelerator) {
        setRecordingHint('invalid');
        return;
      }
      if (!recorded.safe) {
        setRecordingHint('unsafe');
        return;
      }
      setDraft((prev) => (prev ? { ...prev, triggerAccelerator: recorded.accelerator ?? '' } : prev));
      setRecordingTrigger(false);
      setRecordingHint(null);
    };
    window.addEventListener('keydown', onKeyDown, true);
    return () => window.removeEventListener('keydown', onKeyDown, true);
  }, [recordingTrigger]);

  const selectMacro = useCallback((id: number): void => {
    setError(null);
    void loadDetail(id).catch((e: unknown) => setError(errMsg(e)));
  }, [loadDetail]);

  const createMacro = useCallback((name: string): void => {
    setBusy(true);
    void MacroService.createMacro(name)
      .then((detail) => {
        if (!aliveRef.current) return;
        setMacros((prev) => [detail.profile, ...prev]);
        const nextDraft = detailToDraft(detail);
        setActiveId(detail.profile.id);
        setDraft(nextDraft);
        savedSigRef.current = draftSignature(nextDraft);
      })
      .catch((e: unknown) => setError(errMsg(e)))
      .finally(() => aliveRef.current && setBusy(false));
  }, []);

  const saveDraft = useCallback((): void => {
    if (!draft) return;
    setBusy(true);
    setError(null);
    void MacroService.saveMacro(draftToRequest(draft))
      .then((detail) => {
        if (!aliveRef.current) return;
        const nextDraft = detailToDraft(detail);
        setDraft(nextDraft);
        savedSigRef.current = draftSignature(nextDraft);
        setActiveId(detail.profile.id);
        setMacros((prev) => {
          const rest = prev.filter((m) => m.id !== detail.profile.id);
          return [detail.profile, ...rest];
        });
      })
      .catch((e: unknown) => setError(errMsg(e)))
      .finally(() => aliveRef.current && setBusy(false));
  }, [draft]);

  const deleteActive = useCallback((): void => {
    if (!activeId) return;
    setBusy(true);
    void MacroService.deleteMacro(activeId)
      .then(async () => {
        if (!aliveRef.current) return;
        const list = await MacroService.listMacros();
        if (!aliveRef.current) return;
        setMacros(list);
        const nextID = list[0]?.id ?? 0;
        if (nextID) {
          await loadDetail(nextID);
        } else {
          setActiveId(0);
          setDraft(null);
          savedSigRef.current = 'null';
        }
      })
      .catch((e: unknown) => setError(errMsg(e)))
      .finally(() => aliveRef.current && setBusy(false));
  }, [activeId, loadDetail]);

  const runActive = useCallback((): void => {
    if (!activeId) return;
    setError(null);
    void MacroService.runMacro(activeId).catch((e: unknown) => setError(errMsg(e)));
  }, [activeId]);

  const stopRunning = useCallback((): void => {
    void MacroService.stopMacro().catch((e: unknown) => setError(errMsg(e)));
  }, []);

  const setEnabled = useCallback((enabled: boolean): void => {
    setDraft((prev) => (prev ? { ...prev, enabled } : prev));
  }, []);

  const updateDraft = useCallback((patch: Partial<DraftMacro>): void => {
    setDraft((prev) => (prev ? { ...prev, ...patch } : prev));
  }, []);

  const addStep = useCallback((kind: MacroStepKind): void => {
    const firstKey = keys.find((k) => !k.modifier);
    setDraft((prev) => {
      if (!prev) return prev;
      // Insert right after the selected step so users can splice precise delays
      // into a recorded timeline (mirrors G HUB inserting at the playhead).
      // Falls back to appending when nothing is selected.
      const insertAt = selectedStepIndex >= 0 && selectedStepIndex < prev.steps.length ? selectedStepIndex + 1 : prev.steps.length;
      const newStep = emptyStep(kind, firstKey);
      const steps = [...prev.steps.slice(0, insertAt), newStep, ...prev.steps.slice(insertAt)].map((s, i) => ({ ...s, orderIndex: i }));
      setSelectedStepIndex(insertAt);
      return { ...prev, steps };
    });
  }, [keys, selectedStepIndex]);

  const removeStep = useCallback((index: number): void => {
    setDraft((prev) => {
      if (!prev) return prev;
      const steps = prev.steps.filter((_, i) => i !== index).map((s, i) => ({ ...s, orderIndex: i }));
      setSelectedStepIndex(steps.length === 0 ? -1 : Math.min(index, steps.length - 1));
      return { ...prev, steps };
    });
  }, []);

  const moveStep = useCallback((index: number, direction: -1 | 1): void => {
    setDraft((prev) => {
      if (!prev) return prev;
      const nextIndex = index + direction;
      if (nextIndex < 0 || nextIndex >= prev.steps.length) return prev;
      const steps = [...prev.steps];
      const current = steps[index];
      const target = steps[nextIndex];
      if (!current || !target) return prev;
      steps[index] = target;
      steps[nextIndex] = current;
      setSelectedStepIndex(nextIndex);
      return { ...prev, steps: steps.map((s, i) => ({ ...s, orderIndex: i })) };
    });
  }, []);

  const reorderStep = useCallback((fromIndex: number, toIndex: number): void => {
    setDraft((prev) => {
      if (!prev) return prev;
      if (fromIndex === toIndex) return prev;
      if (fromIndex < 0 || fromIndex >= prev.steps.length) return prev;
      if (toIndex < 0 || toIndex >= prev.steps.length) return prev;
      const steps = [...prev.steps];
      const [moved] = steps.splice(fromIndex, 1);
      if (!moved) return prev;
      steps.splice(toIndex, 0, moved);
      setSelectedStepIndex(toIndex);
      return { ...prev, steps: steps.map((s, i) => ({ ...s, orderIndex: i })) };
    });
  }, []);

  const duplicateStep = useCallback((index: number): void => {
    setDraft((prev) => {
      if (!prev) return prev;
      const src = prev.steps[index];
      if (!src) return prev;
      const steps = [...prev.steps.slice(0, index + 1), { ...src, id: 0 }, ...prev.steps.slice(index + 1)].map((s, i) => ({ ...s, orderIndex: i }));
      setSelectedStepIndex(index + 1);
      return { ...prev, steps };
    });
  }, []);

  const updateStep = useCallback((index: number, patch: Partial<MacroStep>): void => {
    setDraft((prev) => {
      if (!prev || !prev.steps[index]) return prev;
      const steps = prev.steps.map((step, i) => (i === index ? { ...step, ...patch } : step));
      return { ...prev, steps };
    });
  }, []);

  const startStepRecording = useCallback((): void => {
    setError(null);
    void MacroService.startRecording(captureDelays, captureMoves)
      .then((state) => {
        if (!aliveRef.current) return;
        setRecordState(state.state);
        setRecordStepCount(state.stepCount);
      })
      .catch((e: unknown) => setError(errMsg(e)));
  }, [captureDelays, captureMoves]);

  const stopStepRecording = useCallback((): void => {
    void MacroService.stopRecording()
      .then(() => {
        if (!aliveRef.current) return;
        setRecordState('idle');
      })
      .catch((e: unknown) => setError(errMsg(e)));
  }, []);

  const dirty = useMemo(() => draftSignature(draft) !== savedSigRef.current, [draft]);

  return {
    macros,
    activeId,
    draft,
    keys,
    selectedStepIndex,
    recordingTrigger,
    recordingHint,
    recordState,
    recordStepCount,
    captureDelays,
    captureMoves,
    runningState,
    runningMacroId,
    runningStepIndex,
    loading,
    busy,
    dirty,
    error,
    selectMacro,
    createMacro,
    saveDraft,
    deleteActive,
    runActive,
    stopRunning,
    setEnabled,
    updateDraft,
    selectStep: setSelectedStepIndex,
    addStep,
    removeStep,
    moveStep,
    reorderStep,
    duplicateStep,
    updateStep,
    startTriggerRecording: () => {
      setRecordingTrigger(true);
      setRecordingHint('listening');
    },
    cancelTriggerRecording: () => {
      setRecordingTrigger(false);
      setRecordingHint(null);
    },
    startStepRecording,
    stopStepRecording,
    setCaptureDelays,
    setCaptureMoves,
  };
};

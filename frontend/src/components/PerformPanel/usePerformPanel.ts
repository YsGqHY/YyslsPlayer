import { useEffect, useRef, useState } from 'react';
import { usePreferences } from '@/preferences';
import { PlayerService, type PlayerPositionEvent, type PlayerStateSnapshot, type PlayPlan } from '@/services';
import { registerHotkeyHandler } from '@/shared/hotkeys';

export interface PerformPanelState {
  snapshot: PlayerStateSnapshot;
  position: PlayerPositionEvent;
  dryRun: boolean;
  lookaheadMs: number;
  busy: boolean;
  error: string | null;
  countdown: number;
  canStart: boolean;
  canPause: boolean;
  canResume: boolean;
  canStop: boolean;
  canReleaseAll: boolean;
  displayPositionMs: number;
  displayDurationMs: number;
  displayProgress: number;
  canSeek: boolean;
  setDryRun: (value: boolean) => void;
  setLookaheadMs: (value: number) => void;
  setSeekPreview: (value: number) => void;
  commitSeek: (value: number) => void;
  start: () => void;
  pause: () => void;
  resume: () => void;
  stop: () => void;
  releaseAll: () => void;
}

const INITIAL_STATE: PlayerStateSnapshot = {
  sessionId: '',
  state: 'idle',
  positionMs: 0,
  durationMs: 0,
  dryRun: false,
  lookaheadMs: 20,
  errorCode: '',
  message: '',
};

const INITIAL_POSITION: PlayerPositionEvent = {
  sessionId: '',
  state: 'idle',
  positionMs: 0,
  durationMs: 0,
  progress: 0,
  updatedAt: 0,
};

const isProductionBuild = import.meta.env.PROD;
const isActive = (state: PlayerStateSnapshot['state']): boolean => state === 'playing' || state === 'paused';
const isTerminal = (state: PlayerStateSnapshot['state']): boolean => state === 'completed' || state === 'stopped' || state === 'error';
const clampLookahead = (value: number): number => Math.max(5, Math.min(50, Number.isFinite(value) ? Math.round(value) : 20));
const clampCountdown = (value: number): number => Math.max(0, Math.min(10, Number.isFinite(value) ? Math.round(value) : 0));
const clampPosition = (value: number, durationMs: number): number => {
  const next = Number.isFinite(value) ? Math.round(value) : 0;
  if (next < 0) return 0;
  if (durationMs > 0 && next > durationMs) return durationMs;
  return next;
};

export const usePerformPanel = (plan: PlayPlan | null, loading = false): PerformPanelState => {
  const { preferences } = usePreferences();
  const [snapshot, setSnapshot] = useState<PlayerStateSnapshot>(INITIAL_STATE);
  const [position, setPosition] = useState<PlayerPositionEvent>(INITIAL_POSITION);
  const [dryRun, setDryRunState] = useState(isProductionBuild ? false : preferences.performDryRunDefault);
  const [lookaheadMs, setLookaheadMsState] = useState(clampLookahead(preferences.performLookaheadMs));
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [countdown, setCountdown] = useState(0);
  const [seekPositionMs, setSeekPositionMs] = useState(0);
  const [seeking, setSeeking] = useState(false);
  const sessionRef = useRef('');
  const countdownRef = useRef<number | null>(null);
  const seekingRef = useRef(false);
  const pendingTerminalPositionRef = useRef<number | null>(null);
  // 始终指向最新的 start/stop/pause/resume，供全局热键处理器调用（避免闭包过期）。
  const actionsRef = useRef<{ start: () => void; pause: () => void; resume: () => void; stop: () => void }>({
    start: () => {},
    pause: () => {},
    resume: () => {},
    stop: () => {},
  });

  useEffect(() => {
    seekingRef.current = seeking;
  }, [seeking]);

  useEffect(() => {
    const offState = PlayerService.onState((event) => {
      sessionRef.current = event.sessionId || sessionRef.current;
      setSnapshot(event);
      const pendingTerminalPosition = pendingTerminalPositionRef.current;
      if (isTerminal(event.state) && pendingTerminalPosition !== null) {
        pendingTerminalPositionRef.current = null;
        setSeekPositionMs(pendingTerminalPosition);
      } else if (!seekingRef.current) {
        setSeekPositionMs(event.positionMs);
      }
      if (isTerminal(event.state)) {
        setPosition((prev) => ({
          sessionId: event.sessionId,
          state: event.state,
          positionMs: event.positionMs,
          durationMs: event.durationMs,
          progress: event.durationMs > 0 ? Math.max(0, Math.min(1, event.positionMs / event.durationMs)) : prev.progress,
          updatedAt: Date.now(),
        }));
      }
    });
    const offPosition = PlayerService.onPosition((event) => {
      if (sessionRef.current && event.sessionId && event.sessionId !== sessionRef.current) return;
      setPosition(event);
      if (!seekingRef.current) {
        setSeekPositionMs(event.positionMs);
      }
      setSnapshot((prev) => ({
        ...prev,
        sessionId: event.sessionId || prev.sessionId,
        state: event.state,
        positionMs: event.positionMs,
        durationMs: event.durationMs,
      }));
    });
    const offError = PlayerService.onError((event) => {
      sessionRef.current = event.sessionId || sessionRef.current;
      setSnapshot(event);
      setError(event.message || event.errorCode || 'PLAYER_ERROR');
    });
    return () => {
      offState();
      offPosition();
      offError();
    };
  }, []);

  const setDryRun = (value: boolean): void => {
    setDryRunState(isProductionBuild ? false : value);
  };

  useEffect(() => {
    if (snapshot.state === 'playing' || snapshot.state === 'paused') return;
    setDryRunState(isProductionBuild ? false : preferences.performDryRunDefault);
  }, [preferences.performDryRunDefault, snapshot.state]);

  useEffect(() => {
    if (snapshot.state === 'playing' || snapshot.state === 'paused') return;
    setLookaheadMsState(clampLookahead(preferences.performLookaheadMs));
  }, [preferences.performLookaheadMs, snapshot.state]);

  useEffect(() => {
    if (plan) {
      const durationMs = snapshot.durationMs || position.durationMs || plan.durationMs;
      setSeekPositionMs((value) => clampPosition(value, durationMs));
      return;
    }
    sessionRef.current = '';
    setSnapshot(INITIAL_STATE);
    setPosition(INITIAL_POSITION);
    setError(null);
    setCountdown(0);
    setSeekPositionMs(0);
    setSeeking(false);
  }, [plan]);

  useEffect(() => {
    return () => {
      if (countdownRef.current !== null) {
        window.clearInterval(countdownRef.current);
        countdownRef.current = null;
      }
      const sessionId = sessionRef.current;
      if (sessionId) {
        void PlayerService.stop(sessionId).catch(() => undefined);
      }
    };
  }, []);

  const run = (operation: () => Promise<PlayerStateSnapshot | unknown>) => {
    setBusy(true);
    setError(null);
    void operation()
      .then((result) => {
        if (result && typeof result === 'object' && 'state' in result) {
          const next = result as PlayerStateSnapshot;
          sessionRef.current = next.sessionId || sessionRef.current;
          setSnapshot(next);
          setSeekPositionMs(next.positionMs);
        }
      })
      .catch((e: unknown) => {
        setError(e instanceof Error && e.message ? e.message : String(e));
      })
      .finally(() => setBusy(false));
  };

  const start = () => {
    if (!plan || loading || snapshot.state === 'playing' || countdown > 0) return;
    if (snapshot.state === 'paused' && sessionRef.current) {
      run(() => PlayerService.resume(sessionRef.current));
      return;
    }
    run(async () => {
      const startSession = async () => {
        const durationMs = snapshot.durationMs || position.durationMs || plan.durationMs;
        const startPositionMs = clampPosition(seekPositionMs, durationMs);
        const session = await PlayerService.start({ plan, dryRun, lookaheadMs: clampLookahead(lookaheadMs), startPositionMs });
        sessionRef.current = session.sessionId;
        const progress = session.durationMs > 0 ? Math.max(0, Math.min(1, session.positionMs / session.durationMs)) : 0;
        setSeekPositionMs(session.positionMs);
        setPosition({
          sessionId: session.sessionId,
          state: session.state,
          positionMs: session.positionMs,
          durationMs: session.durationMs,
          progress,
          updatedAt: session.updatedAt,
        });
        return session;
      };

      const waitSeconds = clampCountdown(preferences.performCountdownSeconds);
      if (waitSeconds <= 0) return startSession();

      setCountdown(waitSeconds);
      await new Promise<void>((resolve) => {
        let remaining = waitSeconds;
        countdownRef.current = window.setInterval(() => {
          remaining -= 1;
          setCountdown(Math.max(0, remaining));
          if (remaining <= 0 && countdownRef.current !== null) {
            window.clearInterval(countdownRef.current);
            countdownRef.current = null;
            resolve();
          }
        }, 1000);
      });
      return startSession();
    });
  };

  const pause = () => {
    if (snapshot.state !== 'playing' || !sessionRef.current) return;
    run(() => PlayerService.pause(sessionRef.current));
  };

  const resume = () => {
    if (snapshot.state !== 'paused' || !sessionRef.current) return;
    run(() => PlayerService.resume(sessionRef.current));
  };

  const stop = () => {
    if (!sessionRef.current) return;
    run(() => PlayerService.stop(sessionRef.current));
  };

  const releaseAll = () => {
    run(async () => {
      await PlayerService.releaseAll();
      return PlayerService.getState(sessionRef.current);
    });
  };

  const currentDurationMs = snapshot.durationMs || position.durationMs || (plan?.durationMs ?? 0);
  const currentPositionMs = isActive(snapshot.state) ? snapshot.positionMs : seekPositionMs;
  const displayDurationMs = currentDurationMs;
  const displayPositionMs = clampPosition(seeking ? seekPositionMs : currentPositionMs, displayDurationMs);
  const displayProgress = displayDurationMs > 0 ? Math.max(0, Math.min(1, displayPositionMs / displayDurationMs)) : 0;
  const canSeek = Boolean(plan) && displayDurationMs > 0 && countdown === 0 && !busy;

  const setSeekPreview = (value: number) => {
    setSeeking(true);
    setSeekPositionMs(clampPosition(value, displayDurationMs));
  };

  const commitSeek = (value: number) => {
    const nextPosition = clampPosition(value, displayDurationMs);
    setSeeking(false);
    setSeekPositionMs(nextPosition);
    if (!sessionRef.current || !isActive(snapshot.state)) return;
    run(async () => {
      try {
        return await PlayerService.seek(sessionRef.current, nextPosition);
      } catch (seekError) {
        if (!plan) throw seekError;
        pendingTerminalPositionRef.current = nextPosition;
        const stopped = await PlayerService.stop(sessionRef.current);
        if (snapshot.state === 'paused') return { ...stopped, positionMs: nextPosition };
        const session = await PlayerService.start({ plan, dryRun, lookaheadMs: clampLookahead(lookaheadMs), startPositionMs: nextPosition });
        sessionRef.current = session.sessionId;
        return session;
      }
    });
  };

  // 每次渲染刷新动作引用，让全局热键处理器始终调用到最新闭包。
  actionsRef.current = { start, pause, resume, stop };

  // 注册全局热键处理器（仅一次，稳定）。后端已直接处理 stop / 暂停继续，
  // 但空闲态的 play 需要前端发起（依赖当前 PlayPlan）；这里覆盖 play-pause 的"开始"路径。
  useEffect(() => {
    const offPlayPause = registerHotkeyHandler('play-pause', (event) => {
      // 后端已处理暂停/继续；这里只在它没处理时（空闲/就绪/已停止）发起开始。
      if (!event.handledByBackend) actionsRef.current.start();
    });
    const offStop = registerHotkeyHandler('stop', (event) => {
      if (!event.handledByBackend) actionsRef.current.stop();
    });
    return () => {
      offPlayPause();
      offStop();
    };
  }, []);

  return {
    snapshot,
    position,
    dryRun,
    lookaheadMs,
    busy,
    error,
    countdown,
    displayPositionMs,
    displayDurationMs,
    displayProgress,
    canSeek,
    canStart: Boolean(plan) && !loading && !busy && countdown === 0 && snapshot.state !== 'playing',
    canPause: snapshot.state === 'playing' && !busy,
    canResume: snapshot.state === 'paused' && !busy,
    canStop: isActive(snapshot.state) && !busy,
    canReleaseAll: !busy,
    setDryRun,
    setLookaheadMs: (value) => setLookaheadMsState(clampLookahead(value)),
    setSeekPreview,
    commitSeek,
    start,
    pause,
    resume,
    stop,
    releaseAll,
  };
};

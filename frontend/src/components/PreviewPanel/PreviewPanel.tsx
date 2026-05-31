import PlayArrowRoundedIcon from '@mui/icons-material/PlayArrowRounded';
import PauseRoundedIcon from '@mui/icons-material/PauseRounded';
import StopRoundedIcon from '@mui/icons-material/StopRounded';
import ReplayRoundedIcon from '@mui/icons-material/ReplayRounded';
import RefreshRoundedIcon from '@mui/icons-material/RefreshRounded';
import { Box, Button, Typography, useTheme } from '@mui/material';
import { useEffect, useMemo, useRef, useState, type KeyboardEvent as ReactKeyboardEvent, type PointerEvent as ReactPointerEvent } from 'react';
import { useT } from '@/i18n';
import { usePreferences } from '@/preferences';
import { PreviewEngineService, type PlayPlan, type PreviewEngineSnapshot } from '@/services';
import { registerHotkeyHandler } from '@/shared/hotkeys';
import { previewPanelStyles } from './PreviewPanel.styles';

export interface PreviewPanelProps {
  plan: PlayPlan | null;
  loading?: boolean;
  error?: string | null;
  compact?: boolean;
  onRefresh?: () => void | Promise<void>;
}

export const PreviewPanel = ({ plan, loading = false, error = null, compact = false, onRefresh }: PreviewPanelProps) => {
  const theme = useTheme();
  const styles = previewPanelStyles(theme);
  const t = useT();
  const { preferences } = usePreferences();
  const [playerError, setPlayerError] = useState<string | null>(null);
  const [snapshot, setSnapshot] = useState<PreviewEngineSnapshot>({ state: 'idle', positionMs: 0, durationMs: 0, activeLanes: [] });
  // 拖拽中临时位置（ms）；非拖拽时为 null，显示走引擎快照。
  const [dragMs, setDragMs] = useState<number | null>(null);
  const seekBarRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    return PreviewEngineService.subscribe(setSnapshot);
  }, []);

  useEffect(() => {
    PreviewEngineService.configure({
      masterGain: preferences.previewVolume,
      waveform: preferences.previewWaveform,
      progressHz: preferences.previewProgressHz,
    });
  }, [preferences.previewProgressHz, preferences.previewVolume, preferences.previewWaveform]);

  useEffect(() => {
    setPlayerError(null);
    if (!plan) {
      PreviewEngineService.stop();
      return;
    }
    PreviewEngineService.load(plan);
    return () => {
      PreviewEngineService.stop();
    };
  }, [plan]);

  const hasPlan = Boolean(plan) && snapshot.durationMs > 0;
  // 显示位置：拖拽中用临时值，否则用引擎快照。
  const displayMs = dragMs ?? snapshot.positionMs;
  const ratio = useMemo(() => {
    if (snapshot.durationMs <= 0) return 0;
    return Math.max(0, Math.min(1, displayMs / snapshot.durationMs));
  }, [displayMs, snapshot.durationMs]);
  const progress = `${Math.round(ratio * 100)}%`;

  const activeLanesSet = useMemo(() => new Set(snapshot.activeLanes), [snapshot.activeLanes]);
  const stateLabel = t(`previewPanel.states.${snapshot.state}`);
  const canStop = snapshot.state === 'playing' || snapshot.state === 'paused' || snapshot.positionMs > 0;

  // 根据指针 x 坐标算出对应的毫秒位置。
  const msFromClientX = (clientX: number): number => {
    const el = seekBarRef.current?.querySelector('[data-seek-rail="true"]') as HTMLElement | null;
    const rect = (el ?? seekBarRef.current)?.getBoundingClientRect();
    if (!rect || rect.width <= 0) return 0;
    const r = Math.max(0, Math.min(1, (clientX - rect.left) / rect.width));
    return Math.round(r * snapshot.durationMs);
  };

  const handleSeekPointerDown = (event: ReactPointerEvent<HTMLDivElement>) => {
    if (!hasPlan) return;
    event.preventDefault();
    event.currentTarget.setPointerCapture(event.pointerId);
    setDragMs(msFromClientX(event.clientX));
  };

  const handleSeekPointerMove = (event: ReactPointerEvent<HTMLDivElement>) => {
    if (dragMs === null) return;
    setDragMs(msFromClientX(event.clientX));
  };

  const commitSeek = (event: ReactPointerEvent<HTMLDivElement>) => {
    if (dragMs === null) return;
    const target = msFromClientX(event.clientX);
    if (event.currentTarget.hasPointerCapture(event.pointerId)) {
      event.currentTarget.releasePointerCapture(event.pointerId);
    }
    setDragMs(null);
    PreviewEngineService.seek(target);
  };

  // 键盘可达性：左右箭头 ±1s，Home/End 跳到首尾。
  const handleSeekKeyDown = (event: ReactKeyboardEvent<HTMLDivElement>) => {
    if (!hasPlan) return;
    const step = 1000;
    let next: number | null = null;
    switch (event.key) {
      case 'ArrowLeft':
      case 'ArrowDown':
        next = snapshot.positionMs - step;
        break;
      case 'ArrowRight':
      case 'ArrowUp':
        next = snapshot.positionMs + step;
        break;
      case 'Home':
        next = 0;
        break;
      case 'End':
        next = snapshot.durationMs;
        break;
      default:
        return;
    }
    event.preventDefault();
    PreviewEngineService.seek(Math.max(0, Math.min(snapshot.durationMs, next)));
  };

  const handlePlayerFailure = (e: unknown) => {
    setPlayerError(e instanceof Error && e.message ? e.message : t('previewPanel.errors.unknown'));
  };

  const handlePlay = () => {
    if (!plan) return;
    setPlayerError(null);
    const operation = snapshot.state === 'paused' ? PreviewEngineService.resume() : PreviewEngineService.play(plan, 0);
    void operation.catch(handlePlayerFailure);
  };

  const handlePause = () => {
    PreviewEngineService.pause();
  };

  const handleStop = () => {
    PreviewEngineService.stop();
  };

  const handleRestart = () => {
    if (!plan) return;
    setPlayerError(null);
    void PreviewEngineService.play(plan, 0).catch(handlePlayerFailure);
  };

  const handleRefresh = () => {
    if (!onRefresh) return;
    setPlayerError(null);
    PreviewEngineService.stop();
    try {
      void Promise.resolve(onRefresh()).catch(handlePlayerFailure);
    } catch (e: unknown) {
      handlePlayerFailure(e);
    }
  };

  // 全局热键 preview-toggle：在播放↔暂停间切换（空闲时从头播放）。
  // 用 ref 持有最新的 toggle 逻辑，只注册一次稳定处理器。
  const toggleRef = useRef<() => void>(() => {});
  toggleRef.current = () => {
    if (snapshot.state === 'playing') {
      handlePause();
    } else {
      handlePlay();
    }
  };
  useEffect(() => {
    return registerHotkeyHandler('preview-toggle', () => toggleRef.current());
  }, []);

  return (
    <Box sx={styles.root}>
      <Box sx={styles.header}>
        <Box sx={styles.titleBlock}>
          <Typography sx={styles.eyebrow}>{t('previewPanel.eyebrow')}</Typography>
          <Typography sx={styles.meta}>{plan ? t('previewPanel.subtitle', { duration: formatTime(plan.durationMs), frames: String(plan.frames.length) }) : t('previewPanel.empty')}</Typography>
        </Box>
        <Box sx={styles.controls}>
          {onRefresh && (
            <Button variant="outlined" startIcon={<RefreshRoundedIcon />} onClick={handleRefresh} disabled={loading}>
              {t('previewPanel.actions.refresh')}
            </Button>
          )}
          <Button variant="outlined" startIcon={<ReplayRoundedIcon />} onClick={handleRestart} disabled={!plan || loading}>
            {t('previewPanel.actions.restart')}
          </Button>
          <Button variant="outlined" startIcon={<PauseRoundedIcon />} onClick={handlePause} disabled={snapshot.state !== 'playing'}>
            {t('previewPanel.actions.pause')}
          </Button>
          <Button variant="contained" startIcon={<PlayArrowRoundedIcon />} onClick={handlePlay} disabled={!plan || loading || snapshot.state === 'playing'}>
            {snapshot.state === 'paused' ? t('previewPanel.actions.resume') : t('previewPanel.actions.play')}
          </Button>
          <Button variant="outlined" startIcon={<StopRoundedIcon />} onClick={handleStop} disabled={!canStop}>
            {t('previewPanel.actions.stop')}
          </Button>
        </Box>
      </Box>

      <Box sx={styles.progress}>
        <Box
          ref={seekBarRef}
          sx={styles.seekBar}
          role="slider"
          tabIndex={hasPlan ? 0 : -1}
          aria-label={t('previewPanel.seek.aria')}
          aria-valuemin={0}
          aria-valuemax={Math.round(snapshot.durationMs)}
          aria-valuenow={Math.round(displayMs)}
          aria-valuetext={formatTime(displayMs)}
          aria-disabled={!hasPlan}
          onPointerDown={handleSeekPointerDown}
          onPointerMove={handleSeekPointerMove}
          onPointerUp={commitSeek}
          onPointerCancel={commitSeek}
          onKeyDown={handleSeekKeyDown}
        >
          <Box data-seek-rail="true" sx={styles.seekRail}>
            <Box sx={{ ...styles.seekFill, width: progress }} />
          </Box>
          {hasPlan && <Box sx={{ ...styles.seekThumb, left: progress }} />}
        </Box>
        <Box sx={styles.progressRow}>
          <span>{formatTime(displayMs)}</span>
          <span>{formatTime(snapshot.durationMs)}</span>
          <span>{t('previewPanel.status', { state: stateLabel, active: String(snapshot.activeLanes.length) })}</span>
        </Box>
      </Box>

      {error && <Box sx={styles.error}>{t('previewPanel.errors.prefix')}{error}</Box>}
      {playerError && <Box sx={styles.error}>{t('previewPanel.errors.prefix')}{playerError}</Box>}
      {loading && <Box sx={styles.empty}>{t('previewPanel.loading')}</Box>}

      {!loading && !plan && !error && !playerError && <Box sx={styles.empty}>{t('previewPanel.empty')}</Box>}

      {plan && !loading && !compact && (
        <Box sx={styles.progress}>
          <Typography sx={styles.meta}>{t('previewPanel.lanes.title')}</Typography>
          <Box sx={styles.laneRow}>
            {Array.from({ length: 36 }, (_, lane) => (
              <Box key={lane} sx={styles.laneChip} data-active={activeLanesSet.has(lane)}>
                {lane}
              </Box>
            ))}
          </Box>
        </Box>
      )}
    </Box>
  );
};

const formatTime = (value: number): string => {
  const totalSeconds = Math.max(0, Math.floor(value / 1000));
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  return `${minutes}:${seconds.toString().padStart(2, '0')}`;
};

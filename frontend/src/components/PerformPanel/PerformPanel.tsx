import ReportProblemRoundedIcon from '@mui/icons-material/ReportProblemRounded';
import PauseRoundedIcon from '@mui/icons-material/PauseRounded';
import PlayArrowRoundedIcon from '@mui/icons-material/PlayArrowRounded';
import StopRoundedIcon from '@mui/icons-material/StopRounded';
import { Box, Button, FormControlLabel, Slider, Switch, TextField, Typography, useTheme } from '@mui/material';
import { useT } from '@/i18n';
import type { PlayPlan } from '@/services';
import { performPanelStyles } from './PerformPanel.styles';
import { usePerformPanel } from './usePerformPanel';

export interface PerformPanelProps {
  plan: PlayPlan | null;
  loading?: boolean;
  error?: string | null;
}

export const PerformPanel = ({ plan, loading = false, error = null }: PerformPanelProps) => {
  const theme = useTheme();
  const styles = performPanelStyles(theme);
  const t = useT();
  const vm = usePerformPanel(plan, loading);
  const showDryRunSwitch = !import.meta.env.PROD;
  const progressPct = `${Math.round(vm.displayProgress * 100)}%`;
  const stateLabel = t(`performPanel.states.${vm.snapshot.state}`);
  const modeLabel = vm.dryRun ? t('performPanel.mode.dryRun') : t('performPanel.mode.real');

  return (
    <Box sx={styles.root}>
      <Box sx={styles.header}>
        <Box sx={styles.titleBlock}>
          <Typography sx={styles.eyebrow}>{t('performPanel.eyebrow')}</Typography>
          <Typography sx={styles.meta}>
            {plan ? t('performPanel.subtitle', { duration: formatTime(plan.durationMs), frames: String(plan.frames.length), mode: modeLabel }) : t('performPanel.empty')}
          </Typography>
        </Box>
        <Box sx={styles.controls}>
          <Button variant="outlined" startIcon={<PauseRoundedIcon />} onClick={vm.pause} disabled={!vm.canPause}>
            {t('performPanel.actions.pause')}
          </Button>
          <Button variant="contained" startIcon={<PlayArrowRoundedIcon />} onClick={vm.snapshot.state === 'paused' ? vm.resume : vm.start} disabled={vm.snapshot.state === 'paused' ? !vm.canResume : !vm.canStart}>
            {vm.snapshot.state === 'paused' ? t('performPanel.actions.resume') : t('performPanel.actions.start')}
          </Button>
          <Button variant="outlined" startIcon={<StopRoundedIcon />} onClick={vm.stop} disabled={!vm.canStop}>
            {t('performPanel.actions.stop')}
          </Button>
          <Button variant="outlined" color="error" startIcon={<ReportProblemRoundedIcon />} onClick={vm.releaseAll} disabled={!vm.canReleaseAll}>
            {t('performPanel.actions.releaseAll')}
          </Button>
        </Box>
      </Box>

      <Box sx={showDryRunSwitch ? styles.configRow : styles.configRowCompact}>
        {showDryRunSwitch && (
          <Box sx={styles.configCard}>
            <FormControlLabel
              control={<Switch checked={vm.dryRun} onChange={(event) => vm.setDryRun(event.target.checked)} disabled={vm.snapshot.state === 'playing' || vm.snapshot.state === 'paused'} />}
              label={t('performPanel.fields.dryRun')}
            />
            <Typography sx={styles.meta}>{t(vm.dryRun ? 'performPanel.fields.dryRunHelper' : 'performPanel.fields.realHelper')}</Typography>
          </Box>
        )}
        <TextField
          type="number"
          label={t('performPanel.fields.lookahead')}
          value={vm.lookaheadMs}
          onChange={(event) => vm.setLookaheadMs(Number(event.target.value))}
          disabled={vm.snapshot.state === 'playing' || vm.snapshot.state === 'paused'}
          slotProps={{ htmlInput: { min: 5, max: 50, step: 1 } }}
          fullWidth
        />
      </Box>

      <Box sx={styles.progress}>
        <Slider
          value={vm.displayPositionMs}
          min={0}
          max={Math.max(0, vm.displayDurationMs)}
          step={1}
          disabled={!vm.canSeek}
          onChange={(_, value) => vm.setSeekPreview(Array.isArray(value) ? value[0] : value)}
          onChangeCommitted={(_, value) => vm.commitSeek(Array.isArray(value) ? value[0] : value)}
          sx={styles.progressSlider}
        />
        <Box sx={styles.progressRow}>
          <span>{formatTime(vm.displayPositionMs)}</span>
          <span>{formatTime(vm.displayDurationMs)}</span>
          <span>{t('performPanel.status', { state: stateLabel, progress: progressPct })}</span>
        </Box>
      </Box>

      {vm.countdown > 0 && <Box sx={styles.warning}>{t('performPanel.countdown', { seconds: String(vm.countdown) })}</Box>}
      {!vm.dryRun && <Box sx={styles.warning}>{t('performPanel.warnings.realMode')}</Box>}
      {loading && <Box sx={styles.empty}>{t('performPanel.loading')}</Box>}
      {error && <Box sx={styles.error}>{t('performPanel.errors.prefix')}{error}</Box>}
      {vm.error && <Box sx={styles.error}>{t('performPanel.errors.prefix')}{vm.error}</Box>}
      {!loading && !plan && !error && !vm.error && <Box sx={styles.empty}>{t('performPanel.empty')}</Box>}
    </Box>
  );
};

const formatTime = (value: number): string => {
  const totalSeconds = Math.max(0, Math.floor(value / 1000));
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  return `${minutes}:${seconds.toString().padStart(2, '0')}`;
};

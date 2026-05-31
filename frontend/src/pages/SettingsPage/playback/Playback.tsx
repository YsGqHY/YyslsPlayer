import RestartAltRoundedIcon from '@mui/icons-material/RestartAltRounded';
import { Box, Switch, TextField, Typography, useTheme } from '@mui/material';
import type { ChangeEvent } from 'react';
import { useT } from '@/i18n';
import { settingsPageStyles } from '../SettingsPage.styles';
import { playbackStyles } from './Playback.styles';
import { usePlayback } from './usePlayback';

export const Playback = () => {
  const theme = useTheme();
  const shared = settingsPageStyles(theme);
  const styles = playbackStyles(theme);
  const vm = usePlayback();
  const t = useT();

  return (
    <Box sx={shared.section}>
      <Box sx={shared.sectionHeader}>
        <Typography sx={shared.sectionTitle}>{t('settings.playback.title')}</Typography>
        <Typography sx={shared.sectionHint}>{t('settings.playback.hint')}</Typography>
      </Box>

      {vm.showDryRunDefault && (
        <ToggleRow
          label={t('settings.playback.fields.dryRunDefault.label')}
          description={t('settings.playback.fields.dryRunDefault.description')}
          checked={vm.dryRunDefault}
          onChange={vm.setDryRunDefault}
          styles={styles}
        />
      )}
      <Box sx={styles.numberGrid}>
        <NumberField
          label={t('settings.playback.fields.lookahead.label')}
          description={t('settings.playback.fields.lookahead.description')}
          value={vm.lookaheadMs}
          min={5}
          max={50}
          step={1}
          onChange={vm.setLookaheadMs}
          styles={styles}
        />
        <NumberField
          label={t('settings.playback.fields.countdown.label')}
          description={t('settings.playback.fields.countdown.description')}
          value={vm.countdownSeconds}
          min={0}
          max={10}
          step={1}
          onChange={vm.setCountdownSeconds}
          styles={styles}
        />
      </Box>

      <Box component="button" type="button" sx={styles.resetButton} onClick={vm.reset}>
        <RestartAltRoundedIcon fontSize="small" />
        {t('settings.playback.actions.reset')}
      </Box>
    </Box>
  );
};

type Styles = ReturnType<typeof playbackStyles>;

interface ToggleRowProps {
  label: string;
  description: string;
  checked: boolean;
  onChange: (value: boolean) => void;
  styles: Styles;
}

const ToggleRow = ({ label, description, checked, onChange, styles }: ToggleRowProps) => (
  <Box sx={styles.row}>
    <Box sx={styles.rowText}>
      <Typography sx={styles.label}>{label}</Typography>
      <Typography sx={styles.desc}>{description}</Typography>
    </Box>
    <Switch checked={checked} onChange={(e: ChangeEvent<HTMLInputElement>) => onChange(e.target.checked)} slotProps={{ input: { 'aria-label': label } }} />
  </Box>
);

interface NumberFieldProps {
  label: string;
  description: string;
  value: number;
  min: number;
  max: number;
  step: number;
  onChange: (value: number) => void;
  styles: Styles;
}

const NumberField = ({ label, description, value, min, max, step, onChange, styles }: NumberFieldProps) => (
  <Box sx={styles.field}>
    <Typography sx={styles.label}>{label}</Typography>
    <TextField
      type="number"
      value={value}
      onChange={(e) => onChange(Number(e.target.value))}
      slotProps={{ htmlInput: { min, max, step, 'aria-label': label } }}
      fullWidth
    />
    <Typography sx={styles.desc}>{description}</Typography>
  </Box>
);

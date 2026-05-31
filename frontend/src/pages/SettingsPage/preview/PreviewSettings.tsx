import RestartAltRoundedIcon from '@mui/icons-material/RestartAltRounded';
import { Box, MenuItem, Select, TextField, Typography, useTheme } from '@mui/material';
import type { ReactNode } from 'react';
import { useT } from '@/i18n';
import type { PreviewWaveform } from '@/preferences';
import { settingsPageStyles } from '../SettingsPage.styles';
import { previewSettingsStyles } from './PreviewSettings.styles';
import { usePreviewSettings, WAVEFORMS } from './usePreviewSettings';

export const PreviewSettings = () => {
  const theme = useTheme();
  const shared = settingsPageStyles(theme);
  const styles = previewSettingsStyles(theme);
  const vm = usePreviewSettings();
  const t = useT();

  return (
    <Box sx={shared.section}>
      <Box sx={shared.sectionHeader}>
        <Typography sx={shared.sectionTitle}>{t('settings.preview.title')}</Typography>
        <Typography sx={shared.sectionHint}>{t('settings.preview.hint')}</Typography>
      </Box>

      <Box sx={styles.fieldGrid}>
        <Field label={t('settings.preview.fields.volume.label')} description={t('settings.preview.fields.volume.description')} styles={styles}>
          <TextField type="number" value={vm.volume} onChange={(e) => vm.setVolume(Number(e.target.value))} slotProps={{ htmlInput: { min: 0, max: 0.5, step: 0.01 } }} fullWidth />
        </Field>
        <Field label={t('settings.preview.fields.waveform.label')} description={t('settings.preview.fields.waveform.description')} styles={styles}>
          <Select value={vm.waveform} onChange={(e) => vm.setWaveform(e.target.value as PreviewWaveform)} fullWidth>
            {WAVEFORMS.map((waveform) => (
              <MenuItem key={waveform} value={waveform}>{t(`settings.preview.waveforms.${waveform}`)}</MenuItem>
            ))}
          </Select>
        </Field>
        <Field label={t('settings.preview.fields.progressHz.label')} description={t('settings.preview.fields.progressHz.description')} styles={styles}>
          <TextField type="number" value={vm.progressHz} onChange={(e) => vm.setProgressHz(Number(e.target.value))} slotProps={{ htmlInput: { min: 1, max: 30, step: 1 } }} fullWidth />
        </Field>
        <Field label={t('settings.preview.fields.pianoRollMaxNotes.label')} description={t('settings.preview.fields.pianoRollMaxNotes.description')} styles={styles}>
          <TextField type="number" value={vm.pianoRollMaxNotes} onChange={(e) => vm.setPianoRollMaxNotes(Number(e.target.value))} slotProps={{ htmlInput: { min: 100, max: 5000, step: 100 } }} fullWidth />
        </Field>
      </Box>

      <Box component="button" type="button" sx={styles.resetButton} onClick={vm.reset}>
        <RestartAltRoundedIcon fontSize="small" />
        {t('settings.preview.actions.reset')}
      </Box>
    </Box>
  );
};

type Styles = ReturnType<typeof previewSettingsStyles>;

const Field = ({ label, description, children, styles }: { label: string; description: string; children: ReactNode; styles: Styles }) => (
  <Box sx={styles.field}>
    <Typography sx={styles.label}>{label}</Typography>
    {children}
    <Typography sx={styles.desc}>{description}</Typography>
  </Box>
);

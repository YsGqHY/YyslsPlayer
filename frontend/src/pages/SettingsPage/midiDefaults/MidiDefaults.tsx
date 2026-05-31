import RefreshRoundedIcon from '@mui/icons-material/RefreshRounded';
import RestartAltRoundedIcon from '@mui/icons-material/RestartAltRounded';
import SaveRoundedIcon from '@mui/icons-material/SaveRounded';
import { Box, MenuItem, Select, TextField, Typography, useTheme } from '@mui/material';
import type { ReactNode } from 'react';
import { useT } from '@/i18n';
import type { OutOfRangePolicy } from '@/services';
import { settingsPageStyles } from '../SettingsPage.styles';
import { midiDefaultsStyles } from './MidiDefaults.styles';
import { useMidiDefaults } from './useMidiDefaults';

export const MidiDefaults = () => {
  const theme = useTheme();
  const shared = settingsPageStyles(theme);
  const styles = midiDefaultsStyles(theme);
  const vm = useMidiDefaults();
  const t = useT();

  return (
    <Box sx={shared.section}>
      <Box sx={shared.sectionHeader}>
        <Typography sx={shared.sectionTitle}>{t('settings.midiDefaults.title')}</Typography>
        <Typography sx={shared.sectionHint}>{t('settings.midiDefaults.hint')}</Typography>
      </Box>

      <Box sx={styles.actions}>
        <Box component="button" type="button" sx={styles.actionPrimary} onClick={() => void vm.save()} disabled={vm.loading || vm.saving}>
          <SaveRoundedIcon fontSize="small" />
          {vm.saving ? t('settings.midiDefaults.actions.saving') : t('settings.midiDefaults.actions.save')}
        </Box>
        <Box component="button" type="button" sx={styles.actionSecondary} onClick={() => void vm.reload()} disabled={vm.loading || vm.saving}>
          <RefreshRoundedIcon fontSize="small" />
          {t('settings.midiDefaults.actions.reload')}
        </Box>
        <Box component="button" type="button" sx={styles.actionSecondary} onClick={() => void vm.reset()} disabled={vm.loading || vm.saving}>
          <RestartAltRoundedIcon fontSize="small" />
          {t('settings.midiDefaults.actions.reset')}
        </Box>
      </Box>

      {vm.loading && <Typography sx={styles.desc}>{t('settings.midiDefaults.feedback.loading')}</Typography>}
      {vm.saved && <Typography sx={styles.status}>{t('settings.midiDefaults.feedback.saved')}</Typography>}
      {vm.error && <Box sx={styles.error}>{t('settings.midiDefaults.feedback.failed', { message: vm.error })}</Box>}

      <Box sx={styles.fieldGrid}>
        <Field label={t('settings.midiDefaults.fields.name.label')} description={t('settings.midiDefaults.fields.name.description')} styles={styles}>
          <TextField value={vm.form.name} onChange={(e) => vm.updateField('name', e.target.value)} fullWidth />
        </Field>
        <Field label={t('settings.midiDefaults.fields.keymapProfileId.label')} description={t('settings.midiDefaults.fields.keymapProfileId.description')} styles={styles}>
          <TextField type="number" value={vm.form.keymapProfileId} onChange={(e) => vm.updateField('keymapProfileId', Number(e.target.value))} slotProps={{ htmlInput: { min: 1, step: 1 } }} fullWidth />
        </Field>
        <Field label={t('settings.midiDefaults.fields.baseNote.label')} description={t('settings.midiDefaults.fields.baseNote.description')} styles={styles}>
          <TextField type="number" value={vm.form.baseNote} onChange={(e) => vm.updateField('baseNote', Number(e.target.value))} slotProps={{ htmlInput: { min: 0, max: 127, step: 1 } }} fullWidth />
        </Field>
        <Field label={t('settings.midiDefaults.fields.transpose.label')} description={t('settings.midiDefaults.fields.transpose.description')} styles={styles}>
          <TextField type="number" value={vm.form.transpose} onChange={(e) => vm.updateField('transpose', Number(e.target.value))} slotProps={{ htmlInput: { min: -24, max: 24, step: 1 } }} fullWidth />
        </Field>
        <Field label={t('settings.midiDefaults.fields.octaveShift.label')} description={t('settings.midiDefaults.fields.octaveShift.description')} styles={styles}>
          <TextField type="number" value={vm.form.octaveShift} onChange={(e) => vm.updateField('octaveShift', Number(e.target.value))} slotProps={{ htmlInput: { min: -3, max: 3, step: 1 } }} fullWidth />
        </Field>
        <Field label={t('settings.midiDefaults.fields.speed.label')} description={t('settings.midiDefaults.fields.speed.description')} styles={styles}>
          <TextField type="number" value={vm.form.speed} onChange={(e) => vm.updateField('speed', Number(e.target.value))} slotProps={{ htmlInput: { min: 0.25, max: 3, step: 0.05 } }} fullWidth />
        </Field>
        <Field label={t('settings.midiDefaults.fields.minPressMs.label')} description={t('settings.midiDefaults.fields.minPressMs.description')} styles={styles}>
          <TextField type="number" value={vm.form.minPressMs} onChange={(e) => vm.updateField('minPressMs', Number(e.target.value))} slotProps={{ htmlInput: { min: 10, max: 300, step: 1 } }} fullWidth />
        </Field>
        <Field label={t('settings.midiDefaults.fields.releaseGapMs.label')} description={t('settings.midiDefaults.fields.releaseGapMs.description')} styles={styles}>
          <TextField type="number" value={vm.form.releaseGapMs} onChange={(e) => vm.updateField('releaseGapMs', Number(e.target.value))} slotProps={{ htmlInput: { min: 0, max: 200, step: 1 } }} fullWidth />
        </Field>
        <Field label={t('settings.midiDefaults.fields.outOfRangePolicy.label')} description={t('settings.midiDefaults.fields.outOfRangePolicy.description')} styles={styles}>
          <Select value={vm.form.outOfRangePolicy} onChange={(e) => vm.updateField('outOfRangePolicy', e.target.value as OutOfRangePolicy)} fullWidth>
            <MenuItem value="drop">{t('settings.midiDefaults.policies.drop')}</MenuItem>
            <MenuItem value="octaveFold">{t('settings.midiDefaults.policies.octaveFold')}</MenuItem>
            <MenuItem value="nearest">{t('settings.midiDefaults.policies.nearest')}</MenuItem>
          </Select>
        </Field>
      </Box>
    </Box>
  );
};

type Styles = ReturnType<typeof midiDefaultsStyles>;

const Field = ({ label, description, children, styles }: { label: string; description: string; children: ReactNode; styles: Styles }) => (
  <Box sx={styles.field}>
    <Typography sx={styles.label}>{label}</Typography>
    {children}
    <Typography sx={styles.desc}>{description}</Typography>
  </Box>
);

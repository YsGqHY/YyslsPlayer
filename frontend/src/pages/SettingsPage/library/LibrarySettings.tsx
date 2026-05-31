import RestartAltRoundedIcon from '@mui/icons-material/RestartAltRounded';
import { Box, Switch, TextField, Typography, useTheme } from '@mui/material';
import type { ChangeEvent } from 'react';
import { useT } from '@/i18n';
import { settingsPageStyles } from '../SettingsPage.styles';
import { librarySettingsStyles } from './LibrarySettings.styles';
import { useLibrarySettings } from './useLibrarySettings';

export const LibrarySettings = () => {
  const theme = useTheme();
  const shared = settingsPageStyles(theme);
  const styles = librarySettingsStyles(theme);
  const vm = useLibrarySettings();
  const t = useT();

  return (
    <Box sx={shared.section}>
      <Box sx={shared.sectionHeader}>
        <Typography sx={shared.sectionTitle}>{t('settings.library.title')}</Typography>
        <Typography sx={shared.sectionHint}>{t('settings.library.hint')}</Typography>
      </Box>

      <Box sx={styles.row}>
        <Box sx={styles.rowText}>
          <Typography sx={styles.label}>{t('settings.library.fields.autoOpen.label')}</Typography>
          <Typography sx={styles.desc}>{t('settings.library.fields.autoOpen.description')}</Typography>
        </Box>
        <Switch
          checked={vm.autoOpenImportedProject}
          onChange={(e: ChangeEvent<HTMLInputElement>) => vm.setAutoOpenImportedProject(e.target.checked)}
          slotProps={{ input: { 'aria-label': t('settings.library.fields.autoOpen.label') } }}
        />
      </Box>

      <Box sx={styles.field}>
        <Typography sx={styles.label}>{t('settings.library.fields.listLimit.label')}</Typography>
        <TextField
          type="number"
          value={vm.listLimitInput}
          onChange={(e) => vm.setListLimitInput(e.target.value)}
          onBlur={vm.commitListLimit}
          slotProps={{ htmlInput: { min: 5, max: 10000, step: 1 } }}
          fullWidth
        />
        <Typography sx={styles.desc}>{t('settings.library.fields.listLimit.description')}</Typography>
      </Box>

      <Box component="button" type="button" sx={styles.resetButton} onClick={vm.reset}>
        <RestartAltRoundedIcon fontSize="small" />
        {t('settings.library.actions.reset')}
      </Box>
    </Box>
  );
};

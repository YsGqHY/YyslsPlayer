import KeyboardRoundedIcon from '@mui/icons-material/KeyboardRounded';
import RestartAltRoundedIcon from '@mui/icons-material/RestartAltRounded';
import { Box, Switch, Typography, useTheme } from '@mui/material';
import type { SxProps, Theme } from '@mui/material';
import { useT } from '@/i18n';
import { settingsPageStyles } from '../SettingsPage.styles';
import { shortcutsStyles } from './Shortcuts.styles';
import { useShortcuts, type ShortcutRow } from './useShortcuts';

export const Shortcuts = () => {
  const theme = useTheme();
  const shared = settingsPageStyles(theme);
  const styles = shortcutsStyles(theme);
  const vm = useShortcuts();
  const t = useT();

  return (
    <Box sx={shared.section}>
      <Box sx={shared.sectionHeader}>
        <Typography sx={shared.sectionTitle}>{t('settings.shortcuts.title')}</Typography>
        <Typography sx={shared.sectionHint}>{t('settings.shortcuts.hint')}</Typography>
      </Box>

      {vm.error && <Typography sx={styles.errorText}>{t('settings.shortcuts.error', { message: vm.error })}</Typography>}

      <Box sx={styles.list}>
        {vm.rows.map((row) => (
          <ActionRow
            key={row.actionId}
            row={row}
            supported={vm.supported}
            busy={vm.busy}
            recording={vm.recordingActionId === row.actionId}
            recordingHint={vm.recordingActionId === row.actionId ? vm.recordingHint : null}
            onStartRecording={() => vm.startRecording(row.actionId)}
            onCancelRecording={vm.cancelRecording}
            onToggleEnabled={(enabled) => vm.setEnabled(row.actionId, enabled)}
            styles={styles}
          />
        ))}
      </Box>

      <Box component="button" type="button" sx={styles.resetButton} onClick={vm.reset} disabled={vm.busy}>
        <RestartAltRoundedIcon fontSize="small" />
        {t('settings.shortcuts.actions.reset')}
      </Box>
    </Box>
  );
};

type Styles = ReturnType<typeof shortcutsStyles>;

interface ActionRowProps {
  row: ShortcutRow;
  supported: boolean;
  busy: boolean;
  recording: boolean;
  recordingHint: 'listening' | 'unsafe' | 'invalid' | null;
  onStartRecording: () => void;
  onCancelRecording: () => void;
  onToggleEnabled: (enabled: boolean) => void;
  styles: Styles;
}

const ActionRow = ({
  row,
  supported,
  busy,
  recording,
  recordingHint,
  onStartRecording,
  onCancelRecording,
  onToggleEnabled,
  styles,
}: ActionRowProps) => {
  const t = useT();
  const actionLabel = t(`settings.shortcuts.actions.${row.actionId}.label`);
  const actionDesc = t(`settings.shortcuts.actions.${row.actionId}.description`);

  // 状态文案：录制提示 > 冲突 > 注册失败。
  let statusKey: string | null = null;
  let statusVar = false; // true=warning, false=danger
  if (recording) {
    if (recordingHint === 'unsafe') {
      statusKey = 'settings.shortcuts.status.unsafe';
      statusVar = true;
    } else if (recordingHint === 'invalid') {
      statusKey = 'settings.shortcuts.status.invalid';
      statusVar = true;
    } else {
      statusKey = 'settings.shortcuts.status.listening';
      statusVar = true;
    }
  } else if (row.enabled && row.conflict) {
    statusKey = 'settings.shortcuts.status.conflict';
    statusVar = false;
  } else if (supported && row.enabled && !row.registered && row.errorCode) {
    statusKey =
      row.errorCode === 'HOTKEY_ALREADY_REGISTERED'
        ? 'settings.shortcuts.status.occupied'
        : 'settings.shortcuts.status.failed';
    statusVar = false;
  }

  const chipLabel = recording ? t('settings.shortcuts.recording') : row.accelerator;

  return (
    <Box sx={styles.row}>
      <Box sx={styles.rowText}>
        <Typography sx={styles.label}>{actionLabel}</Typography>
        <Typography sx={styles.desc}>{actionDesc}</Typography>
        {statusKey && (
          <Typography sx={[styles.statusLine, statusVar ? styles.statusWarning : styles.statusDanger] as SxProps<Theme>}>
            {t(statusKey)}
          </Typography>
        )}
      </Box>
      <Box sx={styles.controls}>
        <Box
          component="button"
          type="button"
          sx={[
            styles.keyChip,
            recording ? styles.keyChipRecording : false,
            !row.enabled ? styles.keyChipDisabled : false,
          ] as SxProps<Theme>}
          onClick={recording ? onCancelRecording : onStartRecording}
          disabled={busy && !recording}
          aria-label={t('settings.shortcuts.recordAria', { action: actionLabel })}
        >
          <KeyboardRoundedIcon fontSize="small" />
          {chipLabel}
        </Box>
        <Switch
          checked={row.enabled}
          onChange={(e) => onToggleEnabled(e.target.checked)}
          disabled={busy}
          slotProps={{ input: { 'aria-label': t('settings.shortcuts.enableAria', { action: actionLabel }) } }}
        />
      </Box>
    </Box>
  );
};

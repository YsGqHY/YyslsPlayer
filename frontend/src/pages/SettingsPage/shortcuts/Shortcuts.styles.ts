import type { SxProps, Theme } from '@mui/material';

export const shortcutsStyles = (theme: Theme): Record<string, SxProps<Theme>> => {
  const fp = theme.palette.foundation;
  return {
    list: {
      display: 'flex',
      flexDirection: 'column',
      gap: 1,
    },
    row: {
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'space-between',
      gap: 2,
      px: 2.5,
      py: 1.75,
      borderRadius: 1.5,
      border: `1px solid ${fp.divider}`,
      backgroundColor: fp.bg.surface,
    },
    rowText: {
      display: 'flex',
      flexDirection: 'column',
      gap: 0.25,
      minWidth: 0,
      flex: 1,
    },
    label: {
      fontSize: 14,
      fontWeight: 600,
      color: fp.text.primary,
    },
    desc: {
      fontSize: 12,
      color: fp.text.muted,
      lineHeight: 1.5,
    },
    statusLine: {
      fontSize: 12,
      lineHeight: 1.5,
      mt: 0.25,
    },
    statusDanger: {
      color: fp.status.danger,
    },
    statusWarning: {
      color: fp.status.warning,
    },
    controls: {
      display: 'flex',
      alignItems: 'center',
      gap: 1.25,
      flexShrink: 0,
    },
    // 快捷键展示 chip / 录制按钮
    keyChip: {
      display: 'inline-flex',
      alignItems: 'center',
      gap: 0.5,
      minWidth: 132,
      justifyContent: 'center',
      px: 1.5,
      py: 0.75,
      borderRadius: 1,
      border: `1px solid ${fp.divider}`,
      backgroundColor: fp.bg.elevated,
      color: fp.text.primary,
      fontSize: 13,
      fontWeight: 600,
      cursor: 'pointer',
      transition: 'border-color 160ms, background-color 160ms',
      '&:hover': {
        backgroundColor: fp.bg.hover,
      },
      '&:disabled': {
        cursor: 'not-allowed',
        opacity: 0.6,
      },
    },
    keyChipRecording: {
      borderColor: fp.accent,
      color: fp.accent,
      backgroundColor: fp.bg.active,
    },
    keyChipDisabled: {
      opacity: 0.5,
    },
    resetButton: {
      alignSelf: 'flex-start',
      display: 'inline-flex',
      alignItems: 'center',
      gap: 0.75,
      fontSize: 12,
      px: 1.5,
      py: 0.75,
      borderRadius: 1,
      border: `1px solid ${fp.divider}`,
      backgroundColor: fp.bg.surface,
      color: fp.text.primary,
      cursor: 'pointer',
      '&:hover': { backgroundColor: fp.bg.hover },
      '&:disabled': { cursor: 'not-allowed', opacity: 0.6 },
    },
    errorText: {
      fontSize: 12,
      color: fp.status.danger,
      lineHeight: 1.5,
    },
  };
};

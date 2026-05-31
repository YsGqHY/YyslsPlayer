import type { SxProps, Theme } from '@mui/material';

export const previewSettingsStyles = (theme: Theme): Record<string, SxProps<Theme>> => {
  const fp = theme.palette.foundation;
  return {
    fieldGrid: {
      display: 'grid',
      gridTemplateColumns: 'repeat(auto-fit, minmax(220px, 1fr))',
      gap: 1.5,
    },
    field: {
      display: 'flex',
      flexDirection: 'column',
      gap: 0.75,
      p: 2,
      borderRadius: 1.5,
      border: `1px solid ${fp.divider}`,
      backgroundColor: fp.bg.surface,
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
    },
  };
};

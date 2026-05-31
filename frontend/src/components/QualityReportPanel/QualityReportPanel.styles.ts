import type { SxProps, Theme } from '@mui/material';

export const qualityReportPanelStyles = (theme: Theme): Record<string, SxProps<Theme>> => {
  const fp = theme.palette.foundation;
  return {
    root: {
      display: 'flex',
      flexDirection: 'column',
      gap: 2,
    },
    hero: {
      p: 2,
      borderRadius: 1.5,
      border: `1px solid ${fp.divider}`,
      backgroundColor: fp.bg.content,
      display: 'grid',
      gridTemplateColumns: { xs: '1fr', sm: 'minmax(160px, 0.6fr) minmax(0, 1fr)' },
      gap: 2,
      alignItems: 'center',
    },
    ratioValue: {
      color: fp.text.primary,
      fontSize: 36,
      fontWeight: 800,
      lineHeight: 1,
    },
    ratioLabel: {
      mt: 0.75,
      color: fp.text.muted,
      fontSize: 12,
      textTransform: 'uppercase',
      letterSpacing: 0.8,
    },
    progressTrack: {
      height: 10,
      borderRadius: 1,
      backgroundColor: fp.bg.elevated,
      border: `1px solid ${fp.divider}`,
      overflow: 'hidden',
    },
    progressFill: {
      height: '100%',
      backgroundColor: fp.accent,
    },
    grid: {
      display: 'grid',
      gridTemplateColumns: { xs: '1fr', sm: 'repeat(2, minmax(0, 1fr))' },
      gap: 1.25,
    },
    card: {
      p: 1.5,
      borderRadius: 1.5,
      border: `1px solid ${fp.divider}`,
      backgroundColor: fp.bg.content,
    },
    label: {
      fontSize: 11,
      color: fp.text.muted,
      textTransform: 'uppercase',
      letterSpacing: 0.8,
      mb: 0.75,
    },
    value: {
      color: fp.text.primary,
      fontSize: 18,
      fontWeight: 800,
      lineHeight: 1.2,
    },
    subValue: {
      mt: 0.5,
      color: fp.text.secondary,
      fontSize: 12,
    },
    warnings: {
      display: 'flex',
      flexWrap: 'wrap',
      gap: 0.75,
    },
    warningChip: {
      px: 1,
      py: 0.5,
      borderRadius: 1,
      border: `1px solid ${fp.status.warning}`,
      color: fp.status.warning,
      backgroundColor: fp.bg.elevated,
      fontSize: 12,
      fontWeight: 700,
    },
    okChip: {
      px: 1,
      py: 0.5,
      borderRadius: 1,
      border: `1px solid ${fp.status.success}`,
      color: fp.status.success,
      backgroundColor: fp.bg.elevated,
      fontSize: 12,
      fontWeight: 700,
    },
  };
};

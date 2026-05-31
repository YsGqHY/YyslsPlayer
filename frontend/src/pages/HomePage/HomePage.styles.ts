import type { SxProps, Theme } from '@mui/material';

export const homePageStyles = (theme: Theme): Record<string, SxProps<Theme>> => {
  const fp = theme.palette.foundation;
  return {
    root: {
      flex: 1,
      minHeight: 0,
      display: 'flex',
      flexDirection: 'column',
      backgroundColor: fp.bg.content,
      overflow: 'hidden',
    },
    body: {
      flex: 1,
      overflowY: 'auto',
      px: 6,
      py: 6,
      display: 'flex',
      flexDirection: 'column',
      gap: 4,
      alignItems: 'flex-start',
      maxWidth: 720,
      minHeight: 0,
    },
    eyebrow: {
      fontSize: 12,
      fontWeight: 600,
      letterSpacing: 1.2,
      textTransform: 'uppercase',
      color: fp.text.muted,
    },
    hero: {
      fontSize: 32,
      fontWeight: 700,
      color: fp.text.primary,
      lineHeight: 1.2,
      m: 0,
    },
    subtitle: {
      color: fp.text.secondary,
      fontSize: 15,
      lineHeight: 1.6,
    },
    footer: {
      mt: 'auto',
      pt: 4,
      color: fp.text.muted,
      fontSize: 12,
      display: 'flex',
      flexDirection: 'column',
      gap: 0.5,
    },
  };
};

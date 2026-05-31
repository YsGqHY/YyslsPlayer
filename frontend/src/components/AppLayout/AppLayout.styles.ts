import type { SxProps, Theme } from '@mui/material';

export const appLayoutStyles = (theme: Theme, backgroundImageDataUrl = ''): Record<string, SxProps<Theme>> => {
  const fp = theme.palette.foundation;
  const hasBackgroundImage = backgroundImageDataUrl.trim() !== '';
  return {
    root: {
      flex: 1,
      minHeight: 0,
      display: 'flex',
      flexDirection: 'row',
      backgroundColor: fp.bg.content,
      backgroundImage: hasBackgroundImage ? `linear-gradient(${fp.bg.active}, ${fp.bg.active}), url(${backgroundImageDataUrl})` : 'none',
      backgroundSize: hasBackgroundImage ? 'cover' : undefined,
      backgroundPosition: hasBackgroundImage ? 'center' : undefined,
      backgroundRepeat: hasBackgroundImage ? 'no-repeat' : undefined,
    },
    main: {
      flex: 1,
      minWidth: 0,
      display: 'flex',
      flexDirection: 'column',
      backgroundColor: hasBackgroundImage ? 'transparent' : fp.bg.content,
      overflow: 'hidden',
    },
  };
};

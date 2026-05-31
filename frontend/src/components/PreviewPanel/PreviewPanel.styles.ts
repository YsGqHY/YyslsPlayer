import type { SxProps, Theme } from '@mui/material';

export const previewPanelStyles = (theme: Theme): Record<string, SxProps<Theme>> => {
  const fp = theme.palette.foundation;
  return {
    root: {
      display: 'flex',
      flexDirection: 'column',
      gap: 2,
    },
    header: {
      p: 2,
      borderRadius: 1.5,
      border: `1px solid ${fp.divider}`,
      backgroundColor: fp.bg.content,
      display: 'flex',
      justifyContent: 'space-between',
      gap: 2,
      alignItems: 'center',
      flexWrap: 'wrap',
    },
    titleBlock: {
      display: 'flex',
      flexDirection: 'column',
      gap: 0.5,
    },
    eyebrow: {
      fontSize: 11,
      fontWeight: 700,
      letterSpacing: 0.8,
      textTransform: 'uppercase',
      color: fp.accent,
    },
    title: {
      fontSize: 18,
      fontWeight: 800,
      color: fp.text.primary,
    },
    meta: {
      fontSize: 12,
      color: fp.text.muted,
    },
    controls: {
      display: 'flex',
      gap: 1,
      flexWrap: 'wrap',
      alignItems: 'center',
    },
    progress: {
      p: 2,
      borderRadius: 1.5,
      border: `1px solid ${fp.divider}`,
      backgroundColor: fp.bg.content,
      display: 'flex',
      flexDirection: 'column',
      gap: 1,
    },
    progressTrack: {
      height: 12,
      borderRadius: 1,
      backgroundColor: fp.bg.elevated,
      border: `1px solid ${fp.divider}`,
      overflow: 'hidden',
    },
    progressFill: {
      height: '100%',
      backgroundColor: fp.accent,
      transition: 'width 80ms linear',
    },
    // —— 可拖拽 seek 条 ——
    // 外层提供更大的点击/拖拽热区（上下留白），视觉轨道在内层。
    seekBar: {
      position: 'relative',
      py: 1,
      cursor: 'pointer',
      touchAction: 'none',
      userSelect: 'none',
      '&[aria-disabled="true"]': {
        cursor: 'default',
        opacity: 0.6,
      },
      // 键盘聚焦时给可见的焦点环（取 accent）。
      outline: 'none',
      '&:focus-visible [data-seek-rail="true"]': {
        boxShadow: `0 0 0 2px ${fp.accent}`,
      },
    },
    seekRail: {
      height: 10,
      borderRadius: 1,
      backgroundColor: fp.bg.elevated,
      border: `1px solid ${fp.divider}`,
      overflow: 'hidden',
    },
    seekFill: {
      height: '100%',
      backgroundColor: fp.accent,
    },
    // 拖拽圆点：以百分比定位，垂直居中于 seekRail。
    seekThumb: {
      position: 'absolute',
      top: '50%',
      width: 14,
      height: 14,
      borderRadius: '50%',
      backgroundColor: fp.accent,
      border: `2px solid ${fp.bg.content}`,
      transform: 'translate(-50%, -50%)',
      boxShadow: '0 1px 3px rgba(0, 0, 0, 0.25)',
      pointerEvents: 'none',
    },
    progressRow: {
      display: 'flex',
      justifyContent: 'space-between',
      gap: 1,
      flexWrap: 'wrap',
      color: fp.text.secondary,
      fontSize: 12,
    },
    laneRow: {
      display: 'flex',
      flexWrap: 'wrap',
      gap: 0.5,
    },
    laneChip: {
      minWidth: 28,
      px: 0.75,
      py: 0.4,
      borderRadius: 1,
      border: `1px solid ${fp.divider}`,
      backgroundColor: fp.bg.elevated,
      color: fp.text.secondary,
      textAlign: 'center',
      fontSize: 12,
      fontVariantNumeric: 'tabular-nums',
      '&[data-active="true"]': {
        borderColor: fp.accent,
        color: fp.text.primary,
      },
    },
    empty: {
      p: 2,
      borderRadius: 1.5,
      border: `1px dashed ${fp.divider}`,
      color: fp.text.muted,
      textAlign: 'center',
      backgroundColor: fp.bg.content,
    },
    error: {
      p: 1.5,
      borderRadius: 1.5,
      border: `1px solid ${fp.status.danger}`,
      backgroundColor: fp.bg.elevated,
      color: fp.status.danger,
      fontSize: 13,
    },
  };
};

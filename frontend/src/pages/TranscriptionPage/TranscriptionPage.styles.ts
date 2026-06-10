import type { SxProps, Theme } from '@mui/material';

type StyleRecord = Record<string, SxProps<Theme>>;

export const styles = {
  root: {
    p: 3,
    height: '100%',
    overflow: 'auto',
    display: 'flex',
    flexDirection: 'column',
  },
  header: {
    mb: 3,
  },
  mainGrid: {
    display: 'grid',
    gridTemplateColumns: '320px minmax(0, 1fr)',
    gap: 2,
    flex: 1,
    minHeight: 0,
  },
  panel: {
    p: 2,
    display: 'flex',
    flexDirection: 'column',
    minHeight: 0,
    overflow: 'auto',
  },
  panelTitle: {
    fontWeight: 600,
    mb: 1,
  },
  panelHeader: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    mb: 1,
  },
  importArea: {
    display: 'flex',
    flexDirection: 'column',
  },
  probeInfo: {
    mt: 1.5,
    display: 'flex',
    flexDirection: 'column',
    gap: 0.3,
  },
  taskList: {
    flex: 1,
    overflow: 'auto',
    display: 'flex',
    flexDirection: 'column',
    gap: 0.5,
  },
  taskItem: {
    p: 1,
    borderRadius: 1,
    cursor: 'pointer',
    display: 'flex',
    alignItems: 'center',
    transition: 'background 0.15s',
    '&:hover': { bgcolor: 'action.hover' },
  },
  taskItemActive: {
    bgcolor: 'action.selected',
  },
  detailPanel: {
    mt: 2,
    p: 2,
  },
  detailGrid: {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fill, minmax(240px, 1fr))',
    gap: 0.5,
  },
  emptyState: {
    display: 'flex',
    flexDirection: 'column',
    alignItems: 'center',
    justifyContent: 'center',
    height: '100%',
    flex: 1,
  },
  emptyList: {
    display: 'flex',
    flexDirection: 'column',
    alignItems: 'center',
    justifyContent: 'center',
    py: 4,
  },
} satisfies StyleRecord;

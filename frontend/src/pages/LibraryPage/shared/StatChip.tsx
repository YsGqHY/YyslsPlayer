import { Box, Typography, useTheme } from '@mui/material';
import type { ReactNode } from 'react';
import { libraryPageStyles } from '../LibraryPage.styles';

interface StatChipProps {
  icon: ReactNode;
  label: string;
  value: string;
  color?: 'success' | 'warning' | 'error' | 'default';
}

export const StatChip = ({ icon, label, value, color = 'default' }: StatChipProps) => {
  const theme = useTheme();
  const styles = libraryPageStyles(theme);
  const fp = theme.palette.foundation;

  const colorMap: Record<string, string> = {
    success: fp.status.success,
    warning: theme.palette.warning?.main ?? fp.accent,
    error: fp.status.danger,
    default: fp.text.primary,
  };

  return (
    <Box sx={styles.statChip}>
      {icon}
      <Box>
        <Typography sx={{ ...styles.statValue, color: colorMap[color] }}>{value}</Typography>
        <Typography sx={styles.statLabel}>{label}</Typography>
      </Box>
    </Box>
  );
};

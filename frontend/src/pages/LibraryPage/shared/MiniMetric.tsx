import { Box, Typography, useTheme } from '@mui/material';
import { libraryPageStyles } from '../LibraryPage.styles';

interface MiniMetricProps {
  label: string;
  value: string;
}

export const MiniMetric = ({ label, value }: MiniMetricProps) => {
  const theme = useTheme();
  const styles = libraryPageStyles(theme);
  return (
    <Box sx={styles.miniMetric}>
      <Typography sx={styles.miniMetricLabel}>{label}</Typography>
      <Typography sx={styles.miniMetricValue}>{value}</Typography>
    </Box>
  );
};

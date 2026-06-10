import { Box, Typography } from '@mui/material';
import { useTheme } from '@mui/material';
import type { ReactNode } from 'react';
import { libraryPageStyles } from '../LibraryPage.styles';

interface FieldProps {
  label: string;
  helper: string;
  children: ReactNode;
}

export const Field = ({ label, helper, children }: FieldProps) => {
  const theme = useTheme();
  const styles = libraryPageStyles(theme);
  return (
    <Box sx={styles.field}>
      <Typography sx={styles.fieldLabel}>{label}</Typography>
      {children}
      <Typography sx={styles.fieldHelper}>{helper}</Typography>
    </Box>
  );
};

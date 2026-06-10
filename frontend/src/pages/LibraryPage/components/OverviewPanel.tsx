import {
  Box,
  Button,
  Typography,
  useTheme,
} from '@mui/material';
import { RefreshRounded as RefreshRoundedIcon } from '@mui/icons-material';
import type { QualityReport } from '@/services';
import { useT } from '@/i18n';
import { QualityReportPanel } from '@/components/QualityReportPanel';
import { libraryPageStyles } from '../LibraryPage.styles';

interface OverviewPanelProps {
  report: QualityReport | null;
  previewLoading: boolean;
  onRefresh: () => Promise<void>;
}

export const OverviewPanel = ({ report, previewLoading, onRefresh }: OverviewPanelProps) => {
  const theme = useTheme();
  const styles = libraryPageStyles(theme);
  const t = useT();

  return (
    <Box sx={styles.panelStack}>
      <Box sx={styles.panelHeaderCard}>
        <Box>
          <Typography sx={styles.sectionEyebrow}>{t('library.overview.eyebrow')}</Typography>
          <Typography sx={styles.sectionBigTitle}>{t('library.overview.title')}</Typography>
          <Typography sx={styles.sectionDesc}>{t('library.overview.description')}</Typography>
        </Box>
        <Button variant="outlined" startIcon={<RefreshRoundedIcon />} onClick={() => void onRefresh()} disabled={previewLoading}>
          {t('library.actions.refreshPreview')}
        </Button>
      </Box>
      {report ? <QualityReportPanel report={report} /> : <Box sx={styles.hint}>{t('library.overview.noReport')}</Box>}
    </Box>
  );
};

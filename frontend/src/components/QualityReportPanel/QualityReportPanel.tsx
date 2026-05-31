import { Box, Typography, useTheme } from '@mui/material';
import { useT } from '@/i18n';
import type { QualityReport } from '@/services';
import { qualityReportPanelStyles } from './QualityReportPanel.styles';

export interface QualityReportPanelProps {
  report: QualityReport;
  compact?: boolean;
}

export const QualityReportPanel = ({ report, compact = false }: QualityReportPanelProps) => {
  const theme = useTheme();
  const styles = qualityReportPanelStyles(theme);
  const t = useT();
  const ratio = Math.max(0, Math.min(1, report.playableRatio));
  const ratioText = `${Math.round(ratio * 100)}%`;
  const warnings = report.warnings ?? [];

  const cards = [
    {
      key: 'noteRange',
      label: t('qualityReport.metrics.noteRange'),
      value: formatRange(report.noteRange.min, report.noteRange.max),
      sub: t('qualityReport.metrics.rawNotes', { count: String(report.totalNotes) }),
    },
    {
      key: 'mappedRange',
      label: t('qualityReport.metrics.mappedRange'),
      value: formatLaneRange(report.mappedRange.minLane, report.mappedRange.maxLane),
      sub: t('qualityReport.metrics.playableNotes', { count: String(report.playableNotes) }),
    },
    {
      key: 'outOfRange',
      label: t('qualityReport.metrics.outOfRange'),
      value: String(report.outOfRangeCount),
      sub: t('qualityReport.metrics.dropFoldClamp', {
        dropped: String(report.droppedCount),
        folded: String(report.foldedCount),
        clamped: String(report.clampedCount),
      }),
    },
    {
      key: 'blackKeys',
      label: t('qualityReport.metrics.blackKeyCount'),
      value: String(report.blackKeyCount),
      sub: t('qualityReport.metrics.chordDensity', { count: String(report.chordDensity) }),
    },
    {
      key: 'tracks',
      label: t('qualityReport.metrics.trackChannel'),
      value: `${report.trackCount} / ${report.channelCount}`,
      sub: t('qualityReport.metrics.trackChannelHint'),
    },
    {
      key: 'suggestion',
      label: t('qualityReport.metrics.suggestion'),
      value: formatSigned(report.suggestedTranspose),
      sub: t('qualityReport.metrics.octaveShift', { shift: formatSigned(report.suggestedOctaveShift) }),
    },
  ];

  return (
    <Box sx={styles.root}>
      <Box sx={styles.hero}>
        <Box>
          <Typography sx={styles.ratioValue}>{ratioText}</Typography>
          <Typography sx={styles.ratioLabel}>{t('qualityReport.metrics.playableRatio')}</Typography>
        </Box>
        <Box>
          <Box sx={styles.progressTrack}>
            <Box sx={{ ...styles.progressFill, width: ratioText }} />
          </Box>
          <Typography sx={styles.subValue}>{t('qualityReport.metrics.playableSummary', {
            playable: String(report.playableNotes),
            total: String(report.totalNotes),
          })}</Typography>
        </Box>
      </Box>

      <Box sx={styles.grid}>
        {(compact ? cards.slice(0, 4) : cards).map((card) => (
          <Box key={card.key} sx={styles.card}>
            <Typography sx={styles.label}>{card.label}</Typography>
            <Typography sx={styles.value}>{card.value}</Typography>
            <Typography sx={styles.subValue}>{card.sub}</Typography>
          </Box>
        ))}
      </Box>

      <Box sx={styles.warnings}>
        {warnings.length === 0 ? (
          <Box sx={styles.okChip}>{t('qualityReport.warnings.none')}</Box>
        ) : warnings.map((warning) => (
          <Box key={warning} sx={styles.warningChip}>{warningLabel(warning, t)}</Box>
        ))}
      </Box>
    </Box>
  );
};

const formatRange = (min: number, max: number): string => {
  if (min < 0 || max < 0) return '-';
  return `${min} - ${max}`;
};

const formatLaneRange = (min: number, max: number): string => {
  if (min < 0 || max < 0) return '-';
  return `${min} - ${max}`;
};

const warningLabel = (warning: string, t: (key: string) => string): string => {
  switch (warning) {
    case 'out_of_range':
    case 'dropped_notes':
    case 'high_chord_density':
      return t(`qualityReport.warnings.${warning}`);
    default:
      return warning;
  }
};

const formatSigned = (value: number): string => value > 0 ? `+${value}` : String(value);

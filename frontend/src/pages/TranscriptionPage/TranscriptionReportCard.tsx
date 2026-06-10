import { Box, Typography, Chip, Alert } from '@mui/material';
import { useMemo } from 'react';
import { useT } from '@/i18n';

interface ReportCardProps {
  reportJson: string;
}

// 前端 side quality report 类型（从后端 JSON 解析）
interface ParsedReport {
  overallScore?: number;
  transcriptionConfidence?: number;
  playabilityScore?: number;
  audioQualityScore?: number;
  estimatedBpm?: number;
  bpmConfidence?: number;
  keyEstimate?: {
    tonic?: string;
    mode?: string;
    confidence?: number;
    method?: string;
    candidates?: Array<{ tonic: string; mode: string; confidence: number }>;
  };
  noteCount?: number;
  droppedCandidateCount?: number;
  lowConfidenceCount?: number;
  outOfRangeCount?: number;
  shortNoteCount?: number;
  maxPolyphony?: number;
  suggestedTranspose?: number;
  suggestedOctaveShift?: number;
  warnings?: string[];
}

function scoreColor(score: number): string {
  if (score >= 70) return '#4caf50';
  if (score >= 40) return '#ff9800';
  return '#f44336';
}

export function TranscriptionReportCard({ reportJson }: ReportCardProps) {
  const t = useT();

  const report: ParsedReport | null = useMemo(() => {
    try {
      return JSON.parse(reportJson) as ParsedReport;
    } catch {
      return null;
    }
  }, [reportJson]);

  if (!report) return null;

  const overall = report.overallScore ?? 0;
  const playScore = report.playabilityScore ?? 0;
  const bpm = report.estimatedBpm ?? 0;
  const bpmConf = report.bpmConfidence ?? 0;
  const key = report.keyEstimate;

  return (
    <Box sx={{ mt: 2, display: 'flex', flexDirection: 'column', gap: 1.5 }}>
      <Typography variant="subtitle2" sx={{ fontWeight: 600 }}>
        {t('transcription.report.title')}
      </Typography>

      {/* 评分 */}
      <Box sx={{ display: 'flex', gap: 2, alignItems: 'center', flexWrap: 'wrap' }}>
        <Box sx={{ textAlign: 'center' }}>
          <Box
            sx={{
              width: 64,
              height: 64,
              borderRadius: '50%',
              border: `4px solid ${scoreColor(overall)}`,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
            }}
          >
            <Typography variant="h6" sx={{ fontWeight: 700, color: scoreColor(overall) }}>
              {Math.round(overall)}
            </Typography>
          </Box>
          <Typography variant="caption" sx={{ opacity: 0.5 }}>
            {t('transcription.report.overall')}
          </Typography>
        </Box>

        <Box sx={{ textAlign: 'center' }}>
          <Box
            sx={{
              width: 48,
              height: 48,
              borderRadius: '50%',
              border: `3px solid ${scoreColor(playScore)}`,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
            }}
          >
            <Typography variant="body2" sx={{ fontWeight: 600, color: scoreColor(playScore) }}>
              {Math.round(playScore)}
            </Typography>
          </Box>
          <Typography variant="caption" sx={{ opacity: 0.5 }}>
            {t('transcription.report.playable')}
          </Typography>
        </Box>
      </Box>

      {/* BPM */}
      {bpm > 0 && (
        <Box>
          <Typography variant="caption" sx={{ opacity: 0.5 }}>
            BPM: {bpm.toFixed(0)}
            {bpmConf > 0 && ` (${t('transcription.report.bpmConfidence')} ${Math.round(bpmConf * 100)}%)`}
          </Typography>
        </Box>
      )}

      {/* 调性 */}
      {key && key.tonic && (
        <Box>
          <Typography variant="caption" sx={{ opacity: 0.5 }}>
            {t('transcription.report.key')}: {key.tonic} {key.mode}
            {(key.confidence ?? 0) > 0 && ` (${Math.round((key.confidence ?? 0) * 100)}%)`}
          </Typography>
          {key.candidates && key.candidates.length > 1 && (
            <Box sx={{ display: 'flex', gap: 0.5, mt: 0.5, flexWrap: 'wrap' }}>
              {key.candidates.slice(0, 3).map((c, i) => (
                <Chip
                  key={i}
                  size="small"
                  label={`${c.tonic} ${c.mode} ${Math.round(c.confidence * 100)}%`}
                  variant="outlined"
                />
              ))}
            </Box>
          )}
        </Box>
      )}

      {/* 建议 */}
      {((report.suggestedTranspose ?? 0) !== 0 || (report.suggestedOctaveShift ?? 0) !== 0) && (
        <Box>
          <Typography variant="caption" sx={{ opacity: 0.5 }}>
            {(report.suggestedTranspose ?? 0) !== 0 &&
              `${t('transcription.report.suggestedTranspose')}: ${(report.suggestedTranspose ?? 0) > 0 ? '+' : ''}${report.suggestedTranspose}`}
            {(report.suggestedOctaveShift ?? 0) !== 0 &&
              ` / ${t('transcription.report.suggestedOctaveShift')}: ${report.suggestedOctaveShift}`}
          </Typography>
        </Box>
      )}

      {/* 统计 */}
      <Box sx={{ display: 'flex', gap: 1, flexWrap: 'wrap' }}>
        <Chip size="small" label={`${report.noteCount ?? 0} ${t('transcription.report.notes')}`} variant="outlined" />
        {report.outOfRangeCount ? (
          <Chip size="small" label={`${report.outOfRangeCount} ${t('transcription.report.outOfRange')}`} color="warning" variant="outlined" />
        ) : null}
        {report.droppedCandidateCount ? (
          <Chip size="small" label={`${report.droppedCandidateCount} ${t('transcription.report.filtered')}`} variant="outlined" />
        ) : null}
      </Box>

      {/* 警告 */}
      {report.warnings && report.warnings.length > 0 && (
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5 }}>
          {report.warnings.map((w, i) => (
            <Alert key={i} severity="warning" sx={{ py: 0 }}>
              {w}
            </Alert>
          ))}
        </Box>
      )}
    </Box>
  );
}

import { Box, Typography, Chip, Paper } from '@mui/material';
import CheckCircleRoundedIcon from '@mui/icons-material/CheckCircleRounded';
import ErrorRoundedIcon from '@mui/icons-material/ErrorRounded';
import HourglassEmptyRoundedIcon from '@mui/icons-material/HourglassEmptyRounded';
import { useMemo } from 'react';

interface StageDebug {
  stage: string;
  status: string; // ok | failed | running | skipped
  startTime: number;
  endTime: number;
  durationMs: number;
  input: string;
  output: string;
  diagnostics: string;
  error?: string;
}

interface PipelineDebug {
  stages: StageDebug[];
}

interface Props {
  /** analysis[] 中 kind="pipelineDebug" 的 payloadJSON */
  debugJson: string;
}

const STAGE_LABELS: Record<string, string> = {
  probe: 'Probe 探测',
  decode: 'Decode 解码',
  analyze: 'Analyze 分析',
  transcribe: 'Transcribe 转录',
  postprocess: 'Postprocess 后处理',
  midi: 'MIDI 写出',
};

const STAGE_COLORS: Record<string, string> = {
  probe: '#6366f1',
  decode: '#8b5cf6',
  analyze: '#f59e0b',
  transcribe: '#10b981',
  postprocess: '#3b82f6',
  midi: '#06b6d4',
};

function fmtMs(ms: number): string {
  if (ms < 1000) return `${ms}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
  return `${Math.floor(ms / 60000)}m${Math.round((ms % 60000) / 1000)}s`;
}

export function PipelineDebugPanel({ debugJson }: Props) {
  const pipeline: PipelineDebug | null = useMemo(() => {
    try {
      return JSON.parse(debugJson) as PipelineDebug;
    } catch {
      return null;
    }
  }, [debugJson]);

  if (!pipeline || !pipeline.stages || pipeline.stages.length === 0) return null;

  const totalMs = pipeline.stages.reduce((sum, s) => sum + (s.durationMs || 0), 0);

  return (
    <Box sx={{ mt: 2 }}>
      <Typography variant="subtitle2" sx={{ fontWeight: 600, mb: 1 }}>
        Pipeline Debug ({pipeline.stages.length} stages, {fmtMs(totalMs)})
      </Typography>

      {pipeline.stages.map((stage, idx) => {
        const label = STAGE_LABELS[stage.stage] || stage.stage;
        const color = STAGE_COLORS[stage.stage] || '#9e9e9e';

        return (
          <Box key={idx} sx={{ mb: 1.5 }}>
            {/* Stage header */}
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 0.5 }}>
              {/* Status icon */}
              {stage.status === 'ok' ? (
                <CheckCircleRoundedIcon sx={{ fontSize: 16, color: '#4caf50' }} />
              ) : stage.status === 'failed' ? (
                <ErrorRoundedIcon sx={{ fontSize: 16, color: '#f44336' }} />
              ) : (
                <HourglassEmptyRoundedIcon sx={{ fontSize: 16, color: '#9e9e9e' }} />
              )}

              {/* Left rail line */}
              <Box
                sx={{
                  width: 3,
                  height: '100%',
                  borderRadius: 2,
                  bgcolor: stage.status === 'failed' ? '#f44336' : color,
                  opacity: stage.status !== 'running' ? 0.6 : 1,
                }}
              />

              <Typography variant="body2" sx={{ fontWeight: 600, fontSize: '0.82rem' }}>
                {idx + 1}. {label}
              </Typography>

              <Chip
                size="small"
                label={fmtMs(stage.durationMs)}
                sx={{
                  ml: 'auto',
                  fontSize: '0.7rem',
                  height: 20,
                  bgcolor: `${color}18`,
                  color: color,
                }}
              />

              {stage.status === 'failed' && (
                <Chip
                  size="small"
                  label="FAILED"
                  sx={{
                    fontSize: '0.7rem',
                    height: 20,
                    bgcolor: '#f4433618',
                    color: '#f44336',
                    fontWeight: 600,
                  }}
                />
              )}
            </Box>

            {/* Stage details */}
            <Box sx={{ ml: 4.5 }}>
              <Paper
                variant="outlined"
                sx={{
                  p: 1.5,
                  bgcolor: 'action.hover',
                  borderRadius: 1,
                }}
              >
                {stage.input && (
                  <DetailRow label="Input" value={stage.input} />
                )}
                {stage.output && (
                  <DetailRow label="Output" value={stage.output} />
                )}
                {stage.diagnostics && (
                  <DetailRow label="Diagnostics" value={stage.diagnostics} />
                )}
                {stage.error && (
                  <Box sx={{ mt: 0.5 }}>
                    <Typography
                      variant="caption"
                      sx={{ color: '#f44336', fontWeight: 500 }}
                    >
                      Error: {stage.error}
                    </Typography>
                  </Box>
                )}
              </Paper>
            </Box>
          </Box>
        );
      })}
    </Box>
  );
}

function DetailRow({ label, value }: { label: string; value: string }) {
  return (
    <Box sx={{ mb: 0.3, '&:last-child': { mb: 0 } }}>
      <Typography
        variant="caption"
        component="span"
        sx={{ opacity: 0.5, mr: 0.5 }}
      >
        {label}:
      </Typography>
      <Typography
        variant="caption"
        component="span"
        sx={{ wordBreak: 'break-all', fontFamily: 'monospace', fontSize: '0.7rem' }}
      >
        {value}
      </Typography>
    </Box>
  );
}

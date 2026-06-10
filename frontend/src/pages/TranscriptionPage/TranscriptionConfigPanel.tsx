import {
  Box, Typography, Slider, ToggleButton, ToggleButtonGroup,
  TextField, Accordion, AccordionSummary, AccordionDetails,
} from '@mui/material';
import ExpandMoreRoundedIcon from '@mui/icons-material/ExpandMoreRounded';
import { useT } from '@/i18n';
import type { TranscriptionConfig } from '@/services/transcription/TranscriptionService';

interface ConfigPanelProps {
  config: TranscriptionConfig;
  onChange: <K extends keyof TranscriptionConfig>(key: K, value: TranscriptionConfig[K]) => void;
}

export function TranscriptionConfigPanel({ config, onChange }: ConfigPanelProps) {
  const t = useT();

  const handleMode = (_: unknown, val: string | null) => {
    if (val) onChange('mode', val as TranscriptionConfig['mode']);
  };

  const handleQuantize = (_: unknown, val: string | null) => {
    if (val) onChange('quantize', val as TranscriptionConfig['quantize']);
  };

  return (
    <Accordion defaultExpanded={false} sx={{ mt: 1.5 }}>
      <AccordionSummary expandIcon={<ExpandMoreRoundedIcon />}>
        <Typography variant="body2" sx={{ fontWeight: 500 }}>
          {t('transcription.config.title')}
        </Typography>
      </AccordionSummary>
      <AccordionDetails>
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
          <Box>
            <Typography variant="caption" sx={{ opacity: 0.6, mb: 0.5, display: 'block' }}>
              {t('transcription.config.mode')}
            </Typography>
            <ToggleButtonGroup
              size="small"
              value={config.mode}
              exclusive
              onChange={handleMode}
              fullWidth
            >
              <ToggleButton value="melody">{t('transcription.config.melody')}</ToggleButton>
              <ToggleButton value="polyphonic">{t('transcription.config.polyphonic')}</ToggleButton>
            </ToggleButtonGroup>
          </Box>

          <Box>
            <Typography variant="caption" sx={{ opacity: 0.6 }}>
              {t('transcription.config.minConfidence')}: {config.minConfidence.toFixed(2)}
            </Typography>
            <Slider
              size="small"
              min={0.3}
              max={0.95}
              step={0.05}
              value={config.minConfidence}
              onChange={(_, v) => onChange('minConfidence', v as number)}
            />
          </Box>

          <Box>
            <Typography variant="caption" sx={{ opacity: 0.6, mb: 0.5, display: 'block' }}>
              {t('transcription.config.quantize')}
            </Typography>
            <ToggleButtonGroup
              size="small"
              value={config.quantize}
              exclusive
              onChange={handleQuantize}
              fullWidth
            >
              <ToggleButton value="off">{t('transcription.config.quantizeOff')}</ToggleButton>
              <ToggleButton value="light">{t('transcription.config.quantizeLight')}</ToggleButton>
              <ToggleButton value="strong">{t('transcription.config.quantizeStrong')}</ToggleButton>
            </ToggleButtonGroup>
          </Box>

          <Box>
            <Typography variant="caption" sx={{ opacity: 0.6, mb: 0.5, display: 'block' }}>
              {t('transcription.config.maxPolyphony')}
            </Typography>
            <TextField
              size="small"
              type="number"
              slotProps={{ htmlInput: { min: 1, max: 6 } }}
              value={config.maxPolyphony}
              onChange={(e) => {
                const v = parseInt(e.target.value, 10);
                if (v >= 1 && v <= 6) onChange('maxPolyphony', v);
              }}
              fullWidth
            />
          </Box>

          <Box>
            <Typography variant="caption" sx={{ opacity: 0.6, mb: 0.5, display: 'block' }}>
              {t('transcription.config.minDurationMs')}
            </Typography>
            <TextField
              size="small"
              type="number"
              slotProps={{ htmlInput: { min: 20, max: 500 } }}
              value={config.minDurationMs}
              onChange={(e) => {
                const v = parseInt(e.target.value, 10);
                if (v >= 20 && v <= 500) onChange('minDurationMs', v);
              }}
              fullWidth
            />
          </Box>
        </Box>
      </AccordionDetails>
    </Accordion>
  );
}

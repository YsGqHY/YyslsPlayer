import {
  Box,
  Button,
  MenuItem,
  Select,
  Typography,
  useTheme,
} from '@mui/material';
import { SaveRounded as SaveRoundedIcon } from '@mui/icons-material';
import { RestartAltRounded as RestartAltRoundedIcon } from '@mui/icons-material';
import { SpeedRounded as SpeedRoundedIcon } from '@mui/icons-material';
import type { OutOfRangePolicy, MidiProfile, QualityReport } from '@/services';
import { useT } from '@/i18n';
import { Field } from '../shared/Field';
import { NumberField } from '../shared/NumberField';
import { MiniMetric } from '../shared/MiniMetric';
import { TrackToggleSection } from './TrackToggleSection';
import { libraryPageStyles } from '../LibraryPage.styles';
import { formatPercent, formatSigned } from '../utils/format';
import type { LibraryProfileForm } from '../useLibraryPage';

type EditableProfileField = 'baseNote' | 'transpose' | 'octaveShift' | 'speed' | 'outOfRangePolicy' | 'minPressMs' | 'releaseGapMs' | 'keymapProfileId' | 'enabledTracks';
type EditableProfileForm = Pick<LibraryProfileForm, EditableProfileField>;

interface SettingsPanelProps {
  profiles: MidiProfile[];
  selectedProfileId: number;
  form: EditableProfileForm;
  isDirty: boolean;
  saving: boolean;
  saveError: string | null;
  previewError: string | null;
  report: QualityReport | null;
  trackCount: number;
  onSelectProfile: (profileId: number) => void;
  onUpdateField: <K extends EditableProfileField>(field: K, value: LibraryProfileForm[K]) => void;
  onReset: () => void;
  onSave: () => Promise<void>;
}

export const SettingsPanel = ({
  profiles,
  selectedProfileId,
  form,
  isDirty,
  saving,
  saveError,
  previewError,
  report,
  trackCount,
  onSelectProfile,
  onUpdateField,
  onReset,
  onSave,
}: SettingsPanelProps) => {
  const theme = useTheme();
  const styles = libraryPageStyles(theme);
  const t = useT();

  return (
    <Box sx={styles.settingsGrid}>
      <Box sx={styles.settingsMain}>
        <Box sx={styles.section}>
          <Typography sx={styles.sectionTitle}>{t('library.inspector.profile')}</Typography>
          <Select value={String(selectedProfileId)} onChange={(event) => onSelectProfile(Number(event.target.value))} fullWidth size="small">
            {profiles.map((profile) => (
              <MenuItem key={profile.id} value={String(profile.id)}>{profile.name}</MenuItem>
            ))}
          </Select>
        </Box>

        <Box sx={styles.section}>
          <Box sx={styles.sectionHeadingRow}>
            <Typography sx={styles.sectionTitle}>{t('library.inspector.range')}</Typography>
            <Typography sx={styles.pill}>{t('library.inspector.baseRange')}</Typography>
          </Box>
          <Box sx={styles.formGrid}>
            <Field label={t('library.fields.baseNote')} helper={t('library.fields.baseNoteHelper')}>
              <NumberField value={form.baseNote} min={0} max={127} onChange={(value) => onUpdateField('baseNote', value)} />
            </Field>
            <Field label={t('library.fields.transpose')} helper={t('library.fields.transposeHelper')}>
              <NumberField value={form.transpose} min={-24} max={24} onChange={(value) => onUpdateField('transpose', value)} />
            </Field>
            <Field label={t('library.fields.octaveShift')} helper={t('library.fields.octaveShiftHelper')}>
              <NumberField value={form.octaveShift} min={-3} max={3} onChange={(value) => onUpdateField('octaveShift', value)} />
            </Field>
            <Field label={t('library.fields.speed')} helper={t('library.fields.speedHelper')}>
              <NumberField value={form.speed} min={0.25} max={3} step={0.05} onChange={(value) => onUpdateField('speed', value)} />
            </Field>
          </Box>
        </Box>

        <TrackToggleSection
          enabledTracks={form.enabledTracks}
          trackCount={trackCount}
          onChange={(tracks) => onUpdateField('enabledTracks', tracks)}
        />

        <Box sx={styles.section}>
          <Box sx={styles.sectionHeadingRow}>
            <Typography sx={styles.sectionTitle}>{t('library.inspector.mapping')}</Typography>
            <SpeedRoundedIcon fontSize="small" />
          </Box>
          <Field label={t('library.fields.outOfRangePolicy')} helper={t('library.fields.policyHelper')}>
            <Select value={form.outOfRangePolicy} onChange={(event) => onUpdateField('outOfRangePolicy', event.target.value as OutOfRangePolicy)} fullWidth size="small">
              <MenuItem value="drop">{t('library.policies.drop')}</MenuItem>
              <MenuItem value="octaveFold">{t('library.policies.octaveFold')}</MenuItem>
              <MenuItem value="nearest">{t('library.policies.nearest')}</MenuItem>
            </Select>
          </Field>
          <Box sx={styles.formGrid}>
            <Field label={t('library.fields.minPressMs')} helper={t('library.fields.minPressHelper')}>
              <NumberField value={form.minPressMs} min={10} max={300} onChange={(value) => onUpdateField('minPressMs', value)} />
            </Field>
            <Field label={t('library.fields.releaseGapMs')} helper={t('library.fields.releaseGapHelper')}>
              <NumberField value={form.releaseGapMs} min={0} max={200} onChange={(value) => onUpdateField('releaseGapMs', value)} />
            </Field>
          </Box>
          <Field label={t('library.fields.keymapProfileId')} helper={t('library.fields.keymapHelper')}>
            <NumberField value={form.keymapProfileId} min={1} onChange={(value) => onUpdateField('keymapProfileId', value)} />
          </Field>
        </Box>
      </Box>

      <Box sx={styles.settingsAside}>
        <Box sx={styles.section}>
          <Typography sx={styles.sectionTitle}>{t('library.inspector.summary')}</Typography>
          <MiniMetric label={t('library.summary.playableRatio')} value={report ? formatPercent(report.playableRatio) : '-'} />
          <MiniMetric label={t('library.summary.outOfRange')} value={report ? String(report.outOfRangeCount) : '-'} />
          <MiniMetric label={t('library.summary.suggestedTranspose')} value={report ? formatSigned(report.suggestedTranspose) : '-'} />
          <MiniMetric label={t('library.summary.suggestedOctave')} value={report ? formatSigned(report.suggestedOctaveShift) : '-'} />
        </Box>

        {saveError && <Box sx={styles.error}>{t('library.errors.savePrefix')}{saveError}</Box>}
        {previewError && <Box sx={styles.error}>{t('library.errors.previewPrefix')}{previewError}</Box>}
        {!isDirty && <Box sx={styles.hint}>{t('library.inspector.clean')}</Box>}

        <Box sx={styles.settingsActions}>
          <Button variant="outlined" startIcon={<RestartAltRoundedIcon />} onClick={onReset} disabled={!isDirty || saving}>
            {t('library.actions.reset')}
          </Button>
          <Button variant="contained" startIcon={<SaveRoundedIcon />} onClick={() => void onSave()} disabled={!isDirty || saving}>
            {saving ? t('library.actions.saving') : t('library.actions.save')}
          </Button>
        </Box>
      </Box>
    </Box>
  );
};

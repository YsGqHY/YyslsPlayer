import ArrowBackRoundedIcon from '@mui/icons-material/ArrowBackRounded';
import SaveRoundedIcon from '@mui/icons-material/SaveRounded';
import { Box, Button, MenuItem, Select, TextField, Typography, useTheme } from '@mui/material';
import type { ReactNode } from 'react';
import { PerformPanel } from '@/components/PerformPanel';
import { PianoRollView } from '@/components/PianoRollView';
import { PreviewPanel } from '@/components/PreviewPanel';
import { QualityReportPanel } from '@/components/QualityReportPanel';
import { useRouter } from '@/router';
import { useT } from '@/i18n';
import type { OutOfRangePolicy } from '@/services';
import { editorPageStyles } from './EditorPage.styles';
import { useEditorPage } from './useEditorPage';

export const EditorPage = () => {
  const theme = useTheme();
  const styles = editorPageStyles(theme);
  const t = useT();
  const vm = useEditorPage();
  const router = useRouter();

  const goLibrary = () => {
    vm.backToLibrary();
    router.navigate('library');
  };

  if (vm.loading) {
    return (
      <Box sx={styles.root}>
        <Box sx={styles.body}>
          <Box sx={styles.hero}>
            <Box>
              <Typography sx={styles.eyebrow}>{t('editor.eyebrow')}</Typography>
              <Typography component="h1" sx={styles.title}>{t('editor.title')}</Typography>
              <Typography sx={styles.subtitle}>{t('editor.subtitleLoading')}</Typography>
            </Box>
          </Box>
          <Box sx={styles.panel}>
            <Box sx={styles.empty}>
              <Typography>{t('editor.loading')}</Typography>
            </Box>
          </Box>
        </Box>
      </Box>
    );
  }

  const project = vm.project;

  if (!project) {
    return (
      <Box sx={styles.root}>
        <Box sx={styles.body}>
          <Box sx={styles.hero}>
            <Box>
              <Typography sx={styles.eyebrow}>{t('editor.eyebrow')}</Typography>
              <Typography component="h1" sx={styles.title}>{t('editor.title')}</Typography>
              <Typography sx={styles.subtitle}>{t('editor.subtitle')}</Typography>
            </Box>
            <Box sx={styles.actions}>
              <Button variant="outlined" startIcon={<ArrowBackRoundedIcon />} onClick={goLibrary}>
                {t('editor.actions.back')}
              </Button>
            </Box>
          </Box>
          <Box sx={styles.panel}>
            <Box sx={styles.empty}>
              <Typography>{t('editor.empty')}</Typography>
            </Box>
          </Box>
        </Box>
      </Box>
    );
  }

  const currentProfile = project.profiles.find((p) => p.id === vm.selectedProfileId) ?? project.defaultProfile;
  const profileNote = currentProfile.projectId ? t('editor.profile.scopeProject') : t('editor.profile.scopeGlobal');

  return (
    <Box sx={styles.root}>
      <Box sx={styles.body}>
        <Box sx={styles.hero}>
          <Box>
            <Typography sx={styles.eyebrow}>{t('editor.eyebrow')}</Typography>
            <Typography component="h1" sx={styles.title}>{project.project.displayName}</Typography>
            <Typography sx={styles.subtitle}>{t('editor.subtitle', { file: project.project.fileName, hash: project.project.fileHash })}</Typography>
          </Box>
          <Box sx={styles.actions}>
            <Button variant="outlined" startIcon={<ArrowBackRoundedIcon />} onClick={goLibrary}>
              {t('editor.actions.back')}
            </Button>
            <Button variant="outlined" onClick={vm.resetForm} disabled={!vm.isDirty}>
              {t('editor.actions.reset')}
            </Button>
            <Button variant="contained" startIcon={<SaveRoundedIcon />} onClick={() => void vm.save()} disabled={!vm.isDirty || vm.saving}>
              {vm.saving ? t('editor.actions.saving') : t('editor.actions.save')}
            </Button>
          </Box>
        </Box>

        <Box sx={styles.panel}>
          <Box sx={styles.panelHeader}>
            <Typography sx={styles.panelTitle}>{t('editor.profiles.title')}</Typography>
            <Typography sx={styles.pill}>{t('editor.profiles.count', { count: String(project.profiles.length) })}</Typography>
          </Box>
          <Box sx={styles.content}>
            <Box sx={styles.section}>
              <Typography sx={styles.sectionTitle}>{t('editor.profiles.select')}</Typography>
              <Box sx={styles.profileRow}>
                {[project.defaultProfile, ...project.profiles.filter((p) => p.id !== project.defaultProfile.id)].map((profile) => (
                  <Button
                    key={profile.id}
                    type="button"
                    variant={vm.selectedProfileId === profile.id ? 'contained' : 'outlined'}
                    onClick={() => vm.selectProfile(profile.id)}
                  >
                    {profile.name}
                  </Button>
                ))}
              </Box>
            </Box>

            <Box sx={styles.section}>
              <Typography sx={styles.sectionTitle}>{t('editor.config.title')}</Typography>
              <Box sx={styles.fieldGrid}>
                <Field label={t('editor.fields.name')} helper={profileNote}>
                  <TextField value={vm.form.name} onChange={(e) => vm.updateField('name', e.target.value)} fullWidth />
                </Field>
                <Field label={t('editor.fields.keymapProfileId')} helper={t('editor.fields.keymapHelper')}>
                  <TextField
                    type="number"
                    value={vm.form.keymapProfileId}
                    onChange={(e) => vm.updateField('keymapProfileId', Number(e.target.value))}
                    fullWidth
                  />
                </Field>
                <Field label={t('editor.fields.baseNote')} helper={t('editor.fields.baseNoteHelper')}>
                  <TextField type="number" value={vm.form.baseNote} onChange={(e) => vm.updateField('baseNote', Number(e.target.value))} fullWidth />
                </Field>
                <Field label={t('editor.fields.transpose')} helper={t('editor.fields.transposeHelper')}>
                  <TextField type="number" value={vm.form.transpose} onChange={(e) => vm.updateField('transpose', Number(e.target.value))} fullWidth />
                </Field>
                <Field label={t('editor.fields.octaveShift')} helper={t('editor.fields.octaveShiftHelper')}>
                  <TextField type="number" value={vm.form.octaveShift} onChange={(e) => vm.updateField('octaveShift', Number(e.target.value))} fullWidth />
                </Field>
                <Field label={t('editor.fields.speed')} helper={t('editor.fields.speedHelper')}>
                  <TextField type="number" slotProps={{ htmlInput: { step: 0.05, min: 0.25, max: 3 } }} value={vm.form.speed} onChange={(e) => vm.updateField('speed', Number(e.target.value))} fullWidth />
                </Field>
                <Field label={t('editor.fields.minPressMs')} helper={t('editor.fields.minPressHelper')}>
                  <TextField type="number" value={vm.form.minPressMs} onChange={(e) => vm.updateField('minPressMs', Number(e.target.value))} fullWidth />
                </Field>
                <Field label={t('editor.fields.releaseGapMs')} helper={t('editor.fields.releaseGapHelper')}>
                  <TextField type="number" value={vm.form.releaseGapMs} onChange={(e) => vm.updateField('releaseGapMs', Number(e.target.value))} fullWidth />
                </Field>
                <Field label={t('editor.fields.outOfRangePolicy')} helper={t('editor.fields.policyHelper')}>
                  <Select
                    value={vm.form.outOfRangePolicy}
                    onChange={(e) => vm.updateField('outOfRangePolicy', e.target.value as OutOfRangePolicy)}
                    fullWidth
                  >
                    <MenuItem value="drop">{t('editor.policies.drop')}</MenuItem>
                    <MenuItem value="octaveFold">{t('editor.policies.octaveFold')}</MenuItem>
                    <MenuItem value="nearest">{t('editor.policies.nearest')}</MenuItem>
                  </Select>
                </Field>
              </Box>
            </Box>

            {vm.error && <Box sx={styles.error}>{t('editor.errors.prefix')}{vm.error}</Box>}
            {vm.saveError && <Box sx={styles.error}>{t('editor.errors.savePrefix')}{vm.saveError}</Box>}
            {!vm.isDirty && <Box sx={styles.hint}>{t('editor.hint.clean')}</Box>}
            <Box sx={styles.footer}>
              <Typography sx={styles.helper}>{t('editor.scopeHint', { project: project.project.displayName })}</Typography>
              <Typography sx={styles.helper}>{t('editor.profileHint', { name: currentProfile.name })}</Typography>
            </Box>
          </Box>
        </Box>

        <Box sx={styles.panel}>
          <Box sx={styles.panelHeader}>
            <Typography sx={styles.panelTitle}>{t('editor.summary.title')}</Typography>
            <Typography sx={styles.pill}>{t('editor.summary.profile', { name: currentProfile.name })}</Typography>
          </Box>
          <Box sx={styles.content}>
            <PreviewPanel plan={vm.previewPlan} loading={vm.previewLoading} error={vm.previewError} compact onRefresh={vm.refreshPreview} />
            <PerformPanel plan={vm.previewPlan} loading={vm.previewLoading} error={vm.previewError} />
            <PianoRollView plan={vm.previewPlan} loading={vm.previewLoading} compact />
            <QualityReportPanel report={project.qualityReport} />
            <Box sx={styles.section}>
              <Typography sx={styles.sectionTitle}>{t('editor.summary.detail')}</Typography>
              <Typography sx={styles.helper}>{t('editor.summary.file', { file: project.project.fileName })}</Typography>
              <Typography sx={styles.helper}>{t('editor.summary.noteCount', { count: String(project.project.noteCount) })}</Typography>
              <Typography sx={styles.helper}>{t('editor.summary.trackChannel', { tracks: String(project.project.trackCount), channels: String(project.project.channelCount) })}</Typography>
            </Box>
          </Box>
        </Box>
      </Box>
    </Box>
  );
};

const Field = ({ label, helper, children }: { label: string; helper: string; children: ReactNode }) => {
  const theme = useTheme();
  const styles = editorPageStyles(theme);
  return (
    <Box sx={styles.field}>
      <Typography sx={styles.fieldLabel}>{label}</Typography>
      {children}
      <Typography sx={styles.helper}>{helper}</Typography>
    </Box>
  );
};

import { Box, Typography, Paper, Button, LinearProgress, CircularProgress, TextField } from '@mui/material';
import AudioFileRoundedIcon from '@mui/icons-material/AudioFileRounded';
import DownloadRoundedIcon from '@mui/icons-material/DownloadRounded';
import MusicNoteRoundedIcon from '@mui/icons-material/MusicNoteRounded';
import TerminalRoundedIcon from '@mui/icons-material/TerminalRounded';
import ErrorOutlineRoundedIcon from '@mui/icons-material/ErrorOutlineRounded';
import { useTheme } from '@mui/material/styles';
import { useT } from '@/i18n';
import { useTranscriptionPage } from './useTranscriptionPage';
import { TranscriptionConfigPanel } from './TranscriptionConfigPanel';
import { TranscriptionReportCard } from './TranscriptionReportCard';
import { PipelineDebugPanel } from './PipelineDebugPanel';
import { styles } from './TranscriptionPage.styles';

export function TranscriptionPage() {
  const t = useT();
  const theme = useTheme();
  const vm = useTranscriptionPage();
  const { capability, capabilityLoading } = vm;

  const missing = capability?.missingComponents ?? [];
  const showUnavailable = !capabilityLoading &&
    (!capability?.transcriptionEnabled || !vm.capability?.transcriptionEnabled || missing.length > 0);

  // 不可用状态（含组件缺失：ffmpeg / model）
  if (showUnavailable) {
    return (
      <Box sx={styles.root}>
        <Box sx={styles.emptyState}>
          <AudioFileRoundedIcon sx={{ fontSize: 64, opacity: 0.3 }} />
          <Typography variant="h6" sx={{ mt: 2, opacity: 0.6 }}>
            {t('transcription.unavailable.title')}
          </Typography>
          <Typography variant="body2" sx={{ mt: 1, opacity: 0.4, maxWidth: 420, textAlign: 'center' }}>
            {capability?.buildFlavor === 'lite'
              ? t('transcription.unavailable.liteMessage')
              : t('transcription.unavailable.defaultMessage')
            }
          </Typography>

          {/* 缺失组件列表 + 操作 */}
          {missing.length > 0 && (
            <Box sx={{ mt: 3, display: 'flex', flexDirection: 'column', gap: 2, width: '100%', maxWidth: 360 }}>
              {missing.includes('ffmpeg') && (
                <Paper variant="outlined" sx={{ p: 1.5 }}>
                  <Box sx={{ display: 'flex', alignItems: 'flex-start', gap: 1 }}>
                    <Box sx={{ mt: 0.3, opacity: 0.5 }}><DownloadRoundedIcon /></Box>
                    <Box sx={{ flex: 1 }}>
                      <Typography variant="body2" sx={{ fontWeight: 500 }}>
                        {t('transcription.missing.ffmpeg')}
                      </Typography>
                      <Typography variant="caption" sx={{ opacity: 0.5, display: 'block', mt: 0.3 }}>
                        {t('transcription.missing.ffmpegDetail')}
                      </Typography>
                    </Box>
                  </Box>
                  <TextField
                    size="small"
                    fullWidth
                    sx={{ mt: 1 }}
                    placeholder={t('transcription.missing.proxyPlaceholder')}
                    value={vm.proxyAddress}
                    onChange={(e) => vm.setProxyAddress(e.target.value)}
                    slotProps={{ input: { style: { fontSize: '0.8rem' } } }}
                  />
                  <Button
                    size="small"
                    variant="outlined"
                    fullWidth
                    sx={{ mt: 1 }}
                    startIcon={vm.installingFfmpeg ? <CircularProgress size={14} /> : undefined}
                    onClick={vm.installFfmpeg}
                    disabled={vm.installingFfmpeg}
                  >
                    {vm.installingFfmpeg ? t('transcription.missing.installing') : t('transcription.missing.installFfmpeg')}
                  </Button>
                  {vm.ffmpegResult && (
                    <Typography variant="caption" sx={{ opacity: 0.6, display: 'block', mt: 0.5 }}>
                      {vm.ffmpegResult}
                    </Typography>
                  )}
                </Paper>
              )}
              {missing.includes('model') && (
                <MissingAction
                  icon={<TerminalRoundedIcon />}
                  label={t('transcription.missing.model')}
                  detail={t('transcription.missing.modelDetail')}
                  actionLabel={vm.installingModel ? t('transcription.missing.installing') : t('transcription.missing.installModel')}
                  loading={vm.installingModel}
                  onClick={vm.installModel}
                  result={vm.modelResult}
                />
              )}
            </Box>
          )}
        </Box>
      </Box>
    );
  }

  // 加载中
  if (capabilityLoading) {
    return (
      <Box sx={styles.root}>
        <Box sx={styles.emptyState}>
          <LinearProgress sx={{ width: 200 }} />
        </Box>
      </Box>
    );
  }

  return (
    <Box sx={styles.root}>
      {/* 顶部标题 */}
      <Box sx={styles.header}>
        <Typography variant="h5" sx={{ fontWeight: 600 }}>
          {t('transcription.title')}
        </Typography>
        <Typography variant="body2" sx={{ opacity: 0.6 }}>
          {t('transcription.subtitle')}
        </Typography>
      </Box>

      {/* 主布局 */}
      <Box sx={styles.mainGrid}>
        {/* 左侧：音频导入 + 配置 */}
        <Paper variant="outlined" sx={styles.panel}>
          <Typography variant="subtitle2" sx={styles.panelTitle} gutterBottom>
            {t('transcription.import.title')}
          </Typography>
          <Box sx={styles.importArea}>
            <Button
              variant="outlined"
              startIcon={<AudioFileRoundedIcon />}
              onClick={vm.selectAudioFile}
              fullWidth
            >
              {t('transcription.import.selectFile')}
            </Button>
            {vm.selectedFilePath && (
              <Typography variant="caption" sx={{ mt: 1, wordBreak: 'break-all' }}>
                {vm.selectedFilePath}
              </Typography>
            )}
            {vm.probeLoading && <LinearProgress sx={{ mt: 1 }} />}
            {vm.probeResult && (
              <Box sx={styles.probeInfo}>
                <InfoRow label={t('transcription.probe.format')} value={vm.probeResult.format} />
                <InfoRow label={t('transcription.probe.duration')} value={`${(vm.probeResult.durationMs / 1000).toFixed(1)}s`} />
                <InfoRow label={t('transcription.probe.sampleRate')} value={`${vm.probeResult.sampleRate} Hz`} />
                <InfoRow label={t('transcription.probe.channels')} value={String(vm.probeResult.channels)} />
                <InfoRow label={t('transcription.probe.codec')} value={vm.probeResult.codec} />
              </Box>
            )}
            {vm.probeError && (
              <Typography variant="caption" sx={{ color: theme.palette.foundation.status?.danger || '#f44336', mt: 1 }}>
                {vm.probeError}
              </Typography>
            )}
          </Box>

          {vm.probeResult && (
            <>
              <Button
                variant="contained"
                onClick={vm.createTranscriptionTask}
                sx={{ mt: 2 }}
                fullWidth
              >
                {t('transcription.import.start')}
              </Button>

              {/* 配置面板 */}
              <TranscriptionConfigPanel config={vm.config} onChange={vm.setConfigField} />
            </>
          )}
        </Paper>

        {/* 右侧：任务列表 */}
        <Paper variant="outlined" sx={styles.panel}>
          <Box sx={styles.panelHeader}>
            <Typography variant="subtitle2" sx={styles.panelTitle}>
              {t('transcription.tasks.title')}
            </Typography>
            <Button size="small" onClick={vm.refreshTasks}>
              {t('transcription.tasks.refresh')}
            </Button>
          </Box>
          {vm.tasks.length === 0 ? (
            <Box sx={styles.emptyList}>
              <MusicNoteRoundedIcon sx={{ fontSize: 40, opacity: 0.3 }} />
              <Typography variant="body2" sx={{ mt: 1, opacity: 0.5 }}>
                {t('transcription.tasks.empty')}
              </Typography>
            </Box>
          ) : (
            <Box sx={styles.taskList}>
              {vm.tasks.map((task) => (
                <Box
                  key={task.id}
                  sx={[
                    styles.taskItem,
                    vm.selectedTask?.task.id === task.id ? styles.taskItemActive : {},
                  ]}
                  onClick={() => vm.selectTask(task.id)}
                >
                  <Box sx={{ flex: 1, minWidth: 0 }}>
                    <Typography variant="body2" sx={{ fontWeight: 500 }} noWrap>
                      {task.sourceFileName}
                    </Typography>
                    <Box sx={{ display: 'flex', gap: 1, alignItems: 'center', mt: 0.5 }}>
                      <TaskStatusChip status={task.status} />
                      <Typography variant="caption" sx={{ opacity: 0.5 }}>
                        {task.stage}
                      </Typography>
                    </Box>
                    {task.status === 'failed' && task.errorMessage && (
                      <Box
                        sx={{
                          mt: 0.5,
                          p: 0.75,
                          borderRadius: 1,
                          bgcolor: theme.palette.foundation.status?.danger
                            ? `${theme.palette.foundation.status.danger}12`
                            : '#f4433612',
                          borderLeft: 3,
                          borderColor: theme.palette.foundation.status?.danger || '#f44336',
                        }}
                      >
                        <Box sx={{ display: 'flex', alignItems: 'flex-start', gap: 0.5 }}>
                          <ErrorOutlineRoundedIcon
                            sx={{
                              fontSize: 14,
                              mt: 0.15,
                              color: theme.palette.foundation.status?.danger || '#f44336',
                            }}
                          />
                          <Typography
                            variant="caption"
                            sx={{
                              color: theme.palette.foundation.status?.danger || '#d32f2f',
                              wordBreak: 'break-word',
                              lineHeight: 1.4,
                              fontSize: '0.72rem',
                            }}
                          >
                            {task.errorCode ? `[${task.errorCode}] ` : ''}{task.errorMessage}
                          </Typography>
                        </Box>
                      </Box>
                    )}
                    {task.status === 'running' && (
                      <LinearProgress
                        variant="determinate"
                        value={task.progress * 100}
                        sx={{ mt: 0.5, height: 3, borderRadius: 1 }}
                      />
                    )}
                  </Box>
                  <Box sx={{ display: 'flex', gap: 0.5, ml: 1 }}>
                    {(task.status === 'queued' || task.status === 'running') && (
                      <Button size="small" color="warning" onClick={(e) => { e.stopPropagation(); vm.cancelTask(task.id); }}>
                        {t('transcription.actions.cancel')}
                      </Button>
                    )}
                    {task.status === 'completed' && (
                      <>
                        <Button size="small" onClick={(e) => { e.stopPropagation(); vm.importResult(task.id); }}>
                          {t('transcription.actions.import')}
                        </Button>
                        <Button size="small" color="secondary" onClick={(e) => { e.stopPropagation(); vm.exportResult(task.id); }}>
                          {t('transcription.actions.export')}
                        </Button>
                      </>
                    )}
                    <Button size="small" color="error" onClick={(e) => { e.stopPropagation(); vm.deleteTask(task.id); }}>
                      {t('transcription.actions.delete')}
                    </Button>
                  </Box>
                </Box>
              ))}
            </Box>
          )}
        </Paper>
      </Box>

      {/* 详情面板 */}
      {vm.selectedTask && (
        <Paper variant="outlined" sx={styles.detailPanel}>
          <Typography variant="subtitle2" sx={styles.panelTitle} gutterBottom>
            {t('transcription.detail.title')}
          </Typography>
          {vm.taskDetailLoading ? (
            <LinearProgress />
          ) : (
            <>
              {/* 失败任务错误横幅 */}
              {vm.selectedTask.task.status === 'failed' && (
                <Box
                  sx={{
                    mb: 2,
                    p: 1.5,
                    borderRadius: 1,
                    bgcolor: theme.palette.foundation.status?.danger
                      ? `${theme.palette.foundation.status.danger}10`
                      : '#f4433610',
                    border: 1,
                    borderColor: theme.palette.foundation.status?.danger
                      ? `${theme.palette.foundation.status.danger}40`
                      : '#f4433640',
                  }}
                >
                  <Box sx={{ display: 'flex', alignItems: 'flex-start', gap: 1 }}>
                    <ErrorOutlineRoundedIcon
                      sx={{
                        fontSize: 18,
                        mt: 0.15,
                        color: theme.palette.foundation.status?.danger || '#f44336',
                      }}
                    />
                    <Box>
                      <Typography
                        variant="body2"
                        sx={{ fontWeight: 600, color: theme.palette.foundation.status?.danger || '#d32f2f' }}
                      >
                        {t('transcription.detail.failedAtStage', { stage: vm.selectedTask.task.stage })}
                      </Typography>
                      <Typography
                        variant="caption"
                        sx={{ mt: 0.3, display: 'block', wordBreak: 'break-word', opacity: 0.85 }}
                      >
                        {vm.selectedTask.task.errorCode ? `[${vm.selectedTask.task.errorCode}] ` : ''}
                        {vm.selectedTask.task.errorMessage}
                      </Typography>
                    </Box>
                  </Box>
                </Box>
              )}
              <Box sx={styles.detailGrid}>
                <InfoRow label={t('transcription.detail.engine')} value={vm.selectedTask.engine || '-'} />
                <InfoRow label={t('transcription.detail.version')} value={vm.selectedTask.engineVersion || '-'} />
                <InfoRow label={t('transcription.detail.sampleRate')} value={`${vm.selectedTask.sampleRate} Hz`} />
                <InfoRow label={t('transcription.detail.channels')} value={String(vm.selectedTask.channels)} />
                <InfoRow label={t('transcription.detail.noteCount')} value={String(vm.selectedTask.notes.length)} />
                {vm.selectedTask.importedProjectId && (
                  <InfoRow label={t('transcription.detail.importedProject')} value={`#${vm.selectedTask.importedProjectId}`} />
                )}
              </Box>
              {vm.selectedTask.reportJson && (
                <TranscriptionReportCard reportJson={vm.selectedTask.reportJson} />
              )}
              {/* 管道诊断面板：显示每个阶段的输入/输出/耗时/诊断 */}
              {vm.selectedTask.analysis
                .filter((a) => a.kind === 'pipelineDebug')
                .map((a) => (
                  <PipelineDebugPanel key={a.id} debugJson={a.payloadJson} />
                ))}
            </>
          )}
        </Paper>
      )}
    </Box>
  );
}

// ===== 子组件 =====

/** MissingAction: 缺失组件卡片（说明 + 操作按钮） */
function MissingAction({
  icon, label, detail, actionLabel, loading, onClick, result,
}: {
  icon: React.ReactNode;
  label: string;
  detail: string;
  actionLabel: string;
  loading: boolean;
  onClick: () => void;
  result: string | null;
}) {
  return (
    <Paper variant="outlined" sx={{ p: 1.5 }}>
      <Box sx={{ display: 'flex', alignItems: 'flex-start', gap: 1 }}>
        <Box sx={{ mt: 0.3, opacity: 0.5 }}>{icon}</Box>
        <Box sx={{ flex: 1 }}>
          <Typography variant="body2" sx={{ fontWeight: 500 }}>
            {label}
          </Typography>
          <Typography variant="caption" sx={{ opacity: 0.5, display: 'block', mt: 0.3 }}>
            {detail}
          </Typography>
        </Box>
      </Box>
      <Button
        size="small"
        variant="outlined"
        fullWidth
        sx={{ mt: 1 }}
        startIcon={loading ? <CircularProgress size={14} /> : undefined}
        onClick={onClick}
        disabled={loading}
      >
        {actionLabel}
      </Button>
      {result && (
        <Typography variant="caption" sx={{ opacity: 0.6, display: 'block', mt: 0.5 }}>
          {result}
        </Typography>
      )}
    </Paper>
  );
}

function InfoRow({ label, value }: { label: string; value: string }) {
  return (
    <Box sx={{ display: 'flex', gap: 1, py: 0.3 }}>
      <Typography variant="caption" sx={{ opacity: 0.5, minWidth: 60 }}>
        {label}
      </Typography>
      <Typography variant="caption" sx={{ wordBreak: 'break-all' }}>
        {value}
      </Typography>
    </Box>
  );
}

function TaskStatusChip({ status }: { status: string }) {
  const t = useT();
  const theme = useTheme();
  const fp = theme.palette.foundation;

  const statusColor: Record<string, string> = {
    queued: fp.text?.muted || '#9e9e9e',
    running: fp.accent || '#2196f3',
    cancelling: fp.status?.warning || '#ff9800',
    completed: fp.status?.success || '#4caf50',
    failed: fp.status?.danger || '#f44336',
    cancelled: fp.status?.warning || '#ff5722',
  };

  return (
    <Typography
      variant="caption"
      sx={{
        px: 0.8,
        py: 0.2,
        borderRadius: 1,
        bgcolor: `${statusColor[status] || '#9e9e9e'}20`,
        color: statusColor[status] || '#9e9e9e',
        fontWeight: 500,
        fontSize: '0.7rem',
      }}
    >
      {t(`transcription.status.${status}`, { _: status })}
    </Typography>
  );
}

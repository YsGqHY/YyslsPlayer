import { Box, Typography, useTheme } from '@mui/material';
import { useT } from '@/i18n';
import { WindowControls } from './WindowControls';
import { titleBarStyles } from './TitleBar.styles';

export interface TitleBarProps {
  // 可选：覆盖默认标题（默认从 i18n 取 app.title）
  title?: string;
}

// Windows 自绘标题栏：拖拽区 + 标题 + 右侧三联按钮。
export const TitleBar = ({ title }: TitleBarProps) => {
  const theme = useTheme();
  const styles = titleBarStyles(theme);
  const t = useT();
  const displayTitle = title ?? t('app.title');

  return (
    <Box sx={styles.root} style={{ '--wails-draggable': 'drag' } as React.CSSProperties}>
      <Typography variant="caption" sx={styles.title}>
        {displayTitle}
      </Typography>
      <Box sx={styles.spacer} />
      <WindowControls />
    </Box>
  );
};

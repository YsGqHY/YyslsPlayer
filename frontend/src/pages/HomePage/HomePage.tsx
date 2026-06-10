import { Box, Typography, useTheme } from '@mui/material';
import { useT } from '@/i18n';
import { useHomePage } from './useHomePage';
import { homePageStyles } from './HomePage.styles';

// Home 页面：首页介绍。
// 所有人类可见文本必须走 t() —— 不允许在 JSX 里硬编码中/英文。
export const HomePage = () => {
  const theme = useTheme();
  const styles = homePageStyles(theme);
  const t = useT();
  const { footerKey, flavor } = useHomePage();
  const editionLabel = flavor === 'completion' ? t('home.edition.completion') : t('home.edition.lite');

  return (
    <Box sx={styles.root}>
      <Box sx={styles.body}>
        <Typography sx={styles.eyebrow}>{t('home.eyebrow')}</Typography>
        <Box sx={styles.heroRow}>
          <Typography component="h1" sx={styles.hero}>
            {t('home.hero')}
          </Typography>
          <Box component="span" sx={styles.editionBadge}>
            {editionLabel}
          </Box>
        </Box>
        <Typography sx={styles.subtitle}>
          {t('home.subtitle')}
        </Typography>

        <Box sx={styles.footer}>
          <Typography variant="inherit">{t(footerKey)}</Typography>
          <Typography variant="inherit">{t('home.footer.scope')}</Typography>
        </Box>
      </Box>
    </Box>
  );
};

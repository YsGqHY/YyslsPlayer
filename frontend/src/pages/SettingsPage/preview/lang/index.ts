import { localeRegistry } from '@/i18n';
import { previewSettingsZhCN } from './zh-CN';
import { previewSettingsEnUS } from './en-US';

let registered = false;

export const registerPreviewSettingsLocales = (): void => {
  if (registered) return;
  localeRegistry.extend('zh-CN', previewSettingsZhCN);
  localeRegistry.extend('en-US', previewSettingsEnUS);
  registered = true;
};

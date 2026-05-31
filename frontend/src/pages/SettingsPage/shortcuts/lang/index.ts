import { localeRegistry } from '@/i18n';
import { shortcutsZhCN } from './zh-CN';
import { shortcutsEnUS } from './en-US';

let registered = false;

export const registerShortcutsLocales = (): void => {
  if (registered) return;
  localeRegistry.extend('zh-CN', shortcutsZhCN);
  localeRegistry.extend('en-US', shortcutsEnUS);
  registered = true;
};

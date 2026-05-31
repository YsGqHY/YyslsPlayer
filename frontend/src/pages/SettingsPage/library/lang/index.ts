import { localeRegistry } from '@/i18n';
import { librarySettingsZhCN } from './zh-CN';
import { librarySettingsEnUS } from './en-US';

let registered = false;

export const registerLibrarySettingsLocales = (): void => {
  if (registered) return;
  localeRegistry.extend('zh-CN', librarySettingsZhCN);
  localeRegistry.extend('en-US', librarySettingsEnUS);
  registered = true;
};

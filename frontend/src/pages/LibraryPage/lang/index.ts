import { localeRegistry } from '@/i18n';
import { libraryPageEnUS } from './en-US';
import { libraryPageZhCN } from './zh-CN';

let registered = false;

export const registerLibraryPageLocales = (): void => {
  if (registered) return;
  localeRegistry.extend('zh-CN', libraryPageZhCN);
  localeRegistry.extend('en-US', libraryPageEnUS);
  registered = true;
};

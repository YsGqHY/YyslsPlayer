import { localeRegistry } from '@/i18n';
import { macroPageZhCN } from './zh-CN';
import { macroPageEnUS } from './en-US';

let registered = false;

export const registerMacroPageLocales = (): void => {
  if (registered) return;
  localeRegistry.extend('zh-CN', macroPageZhCN);
  localeRegistry.extend('en-US', macroPageEnUS);
  registered = true;
};

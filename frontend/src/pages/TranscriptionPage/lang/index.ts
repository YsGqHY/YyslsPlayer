import { localeRegistry } from '@/i18n/registry';
import { transcriptionPageZhCN } from './zh-CN';
import { transcriptionPageEnUS } from './en-US';

let registered = false;

export const registerTranscriptionPageLocales = (): void => {
  if (registered) return;
  localeRegistry.extend('zh-CN', transcriptionPageZhCN);
  localeRegistry.extend('en-US', transcriptionPageEnUS);
  registered = true;
};

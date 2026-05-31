import { localeRegistry } from '@/i18n';
import { playbackZhCN } from './zh-CN';
import { playbackEnUS } from './en-US';

let registered = false;

export const registerPlaybackLocales = (): void => {
  if (registered) return;
  localeRegistry.extend('zh-CN', playbackZhCN);
  localeRegistry.extend('en-US', playbackEnUS);
  registered = true;
};

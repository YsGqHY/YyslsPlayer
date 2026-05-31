import { localeRegistry } from '@/i18n';
import { midiDefaultsZhCN } from './zh-CN';
import { midiDefaultsEnUS } from './en-US';

let registered = false;

export const registerMidiDefaultsLocales = (): void => {
  if (registered) return;
  localeRegistry.extend('zh-CN', midiDefaultsZhCN);
  localeRegistry.extend('en-US', midiDefaultsEnUS);
  registered = true;
};

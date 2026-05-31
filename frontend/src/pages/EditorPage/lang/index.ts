import { localeRegistry } from '@/i18n';
import { editorPageEnUS } from './en-US';
import { editorPageZhCN } from './zh-CN';

let registered = false;

export const registerEditorPageLocales = (): void => {
  if (registered) return;
  localeRegistry.extend('zh-CN', editorPageZhCN);
  localeRegistry.extend('en-US', editorPageEnUS);
  registered = true;
};

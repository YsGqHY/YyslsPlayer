import { Browser } from '@wailsio/runtime';

export const BrowserService = {
  async openURL(url: string): Promise<void> {
    await Browser.OpenURL(url);
  },
} as const;

import { Call } from '@wailsio/runtime';

export interface ImportedBackgroundImage {
  dataUrl: string;
  mime: string;
  size: number;
  name: string;
}

const asObject = (value: unknown): Record<string, unknown> => (value && typeof value === 'object' ? value as Record<string, unknown> : {});
const asString = (value: unknown, fallback = ''): string => String(value ?? fallback);
const asNumber = (value: unknown, fallback = 0): number => Number(value ?? fallback);

const callAppearance = async (method: string, ...args: unknown[]): Promise<unknown> => {
  const names = [
    `YyslsPlayer/internal/services/appearance.Service.${method}`,
    `YyslsPlayer/internal/services/appearance.(*Service).${method}`,
    `YyslsPlayer/internal/services/appearance.${method}`,
  ];
  let lastError: unknown;
  for (const name of names) {
    try {
      return await Call.ByName(name, ...args);
    } catch (error) {
      lastError = error;
    }
  }
  throw lastError;
};

const mapImportedBackgroundImage = (value: unknown): ImportedBackgroundImage => {
  const raw = asObject(value);
  return {
    dataUrl: asString(raw.dataUrl),
    mime: asString(raw.mime),
    size: asNumber(raw.size),
    name: asString(raw.name),
  };
};

export const AppearanceService = {
  async importBackgroundImage(path: string): Promise<ImportedBackgroundImage> {
    return mapImportedBackgroundImage(await callAppearance('ImportBackgroundImage', path));
  },
} as const;

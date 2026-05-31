// 用户界面偏好（行为开关）：
// - 主题不在这里（颜色 / 模式属于"配色"，归 styles/themes/）
// - 这里只放与"行为 / 显隐"相关的开关，比如菜单 logo / Tooltip 是否显示
// 任何新增项请保持 boolean 优先；复杂偏好（枚举 / 数值）也可以加，但请就近写默认值
export type PreviewTone = 'warmSine' | 'softSine' | 'mellowTriangle' | 'roundedBell' | 'glassPad' | 'mutedPluck' | 'softOrgan' | 'warmPad' | 'sine' | 'triangle' | 'square' | 'sawtooth';
export type PreviewWaveform = PreviewTone;

export interface Preferences {
  showLogo: boolean;       // Sidebar 顶部品牌方块是否显示
  showTooltip: boolean;    // Sidebar 菜单按钮是否显示 Tooltip
  backgroundImageDataUrl: string; // 应用背景图 data URL；空字符串表示使用主题背景

  performDryRunDefault: boolean;
  performLookaheadMs: number;
  performCountdownSeconds: number;

  previewVolume: number;
  previewWaveform: PreviewWaveform;
  previewProgressHz: number;
  pianoRollMaxNotes: number;

  libraryAutoOpenImportedProject: boolean;
  libraryListLimit: number;
}

export interface PreferencesContextValue {
  preferences: Preferences;
  setPreference: <K extends keyof Preferences>(key: K, value: Preferences[K]) => void;
  reset: () => void;
}

export const DEFAULT_PREFERENCES: Preferences = {
  showLogo: true,
  showTooltip: true,
  backgroundImageDataUrl: '',

  performDryRunDefault: false,
  performLookaheadMs: 20,
  performCountdownSeconds: 3,

  previewVolume: 0.08,
  previewWaveform: 'warmSine',
  previewProgressHz: 10,
  pianoRollMaxNotes: 900,

  libraryAutoOpenImportedProject: false,
  libraryListLimit: 200,
};

// 键盘事件 → 可读组合字符串（accelerator）的转换工具。
//
// 与后端 internal/services/hotkey/accelerator.go 的命名 / 安全规则保持一致：
//   - 修饰键顺序固定 Ctrl→Alt→Shift→Win
//   - 主键名：字母大写、数字、F1..F24、Space/Enter/Tab/Backspace/Escape/方向键等
//   - 全局热键安全规则：必须含 Ctrl/Alt/Win 之一，或主键是功能键 F1..F24
//
// 仅做"录制"阶段的本地校验与展示；最终仍由后端 SetBinding 二次校验后落库。

export interface RecordedAccelerator {
  // 规范化后的组合文本（如 "Ctrl+Alt+Backspace"）；无主键时为 null
  accelerator: string | null;
  // 是否仅按下了修饰键（还在等待主键）
  modifiersOnly: boolean;
  // 是否满足全局热键安全规则
  safe: boolean;
}

const codeNameMap: Record<string, string> = {
  Cancel: 'Cancel',
  Backspace: 'Backspace',
  Tab: 'Tab',
  Clear: 'Clear',
  Enter: 'Enter',
  NumpadEnter: 'Enter',
  Pause: 'Pause',
  CapsLock: 'CapsLock',
  KanaMode: 'KanaMode',
  Lang1: 'KanaMode',
  JunjaMode: 'JunjaMode',
  FinalMode: 'FinalMode',
  HanjaMode: 'HanjaMode',
  Lang2: 'HanjaMode',
  ImeOn: 'ImeOn',
  ImeOff: 'ImeOff',
  Escape: 'Escape',
  Convert: 'Convert',
  NonConvert: 'NonConvert',
  Accept: 'Accept',
  ModeChange: 'ModeChange',
  Space: 'Space',
  PageUp: 'PageUp',
  PageDown: 'PageDown',
  End: 'End',
  Home: 'Home',
  ArrowLeft: 'Left',
  ArrowUp: 'Up',
  ArrowRight: 'Right',
  ArrowDown: 'Down',
  Select: 'Select',
  Print: 'Print',
  Execute: 'Execute',
  PrintScreen: 'PrintScreen',
  Insert: 'Insert',
  Delete: 'Delete',
  Help: 'Help',
  NumLock: 'NumLock',
  ScrollLock: 'ScrollLock',
  ShiftLeft: 'ShiftLeft',
  ShiftRight: 'ShiftRight',
  ControlLeft: 'ControlLeft',
  ControlRight: 'ControlRight',
  AltLeft: 'AltLeft',
  AltRight: 'AltRight',
  NumpadMultiply: 'NumpadMultiply',
  NumpadAdd: 'NumpadAdd',
  NumpadComma: 'NumpadSeparator',
  NumpadSubtract: 'NumpadSubtract',
  NumpadDecimal: 'NumpadDecimal',
  NumpadDivide: 'NumpadDivide',
  Semicolon: 'Semicolon',
  Equal: 'Equal',
  Comma: 'Comma',
  Minus: 'Minus',
  Period: 'Period',
  Slash: 'Slash',
  Backquote: 'Backquote',
  BracketLeft: 'BracketLeft',
  Backslash: 'Backslash',
  BracketRight: 'BracketRight',
  Quote: 'Quote',
  IntlYen: 'IntlYen',
  IntlBackslash: 'IntlBackslash',
  IntlRo: 'VK0xC1',
  MetaLeft: 'MetaLeft',
  MetaRight: 'MetaRight',
  ContextMenu: 'Apps',
  Sleep: 'Sleep',
  BrowserBack: 'BrowserBack',
  BrowserForward: 'BrowserForward',
  BrowserRefresh: 'BrowserRefresh',
  BrowserStop: 'BrowserStop',
  BrowserSearch: 'BrowserSearch',
  BrowserFavorites: 'BrowserFavorites',
  BrowserHome: 'BrowserHome',
  AudioVolumeMute: 'VolumeMute',
  AudioVolumeDown: 'VolumeDown',
  AudioVolumeUp: 'VolumeUp',
  MediaTrackNext: 'MediaNext',
  MediaTrackPrevious: 'MediaPrevious',
  MediaStop: 'MediaStop',
  MediaPlayPause: 'MediaPlayPause',
  LaunchMail: 'LaunchMail',
  LaunchMediaPlayer: 'LaunchMediaSelect',
  LaunchApp1: 'LaunchApp1',
  LaunchApp2: 'LaunchApp2',
};

// event.code → 主键规范名。覆盖与后端 mainKeys 对齐的集合。
const codeToKeyName = (code: string): string | null => {
  if (/^Key[A-Z]$/.test(code)) return code.slice(3); // KeyA -> A
  if (/^Digit[0-9]$/.test(code)) return code.slice(5); // Digit1 -> 1
  if (/^Numpad[0-9]$/.test(code)) return code; // Numpad1 -> Numpad1
  if (/^F([1-9]|1[0-9]|2[0-4])$/.test(code)) return code; // F1..F24
  return codeNameMap[code] ?? null;
};

const isKeyboardVirtualKey = (vk: number): boolean => {
  if (!Number.isInteger(vk) || vk <= 0 || vk > 0xfe) return false;
  return vk === 0x03 || vk > 0x07;
};

const fallbackVirtualKeyName = (event: KeyboardEvent): string | null => {
  const maybeVK = event.keyCode || event.which;
  if (!isKeyboardVirtualKey(maybeVK)) {
    return null;
  }
  return `VK0x${maybeVK.toString(16).toUpperCase().padStart(2, '0')}`;
};

type ModifierName = 'Ctrl' | 'Alt' | 'Shift' | 'Win';

const modifierForCode = (code: string): ModifierName | null => {
  if (code === 'ControlLeft' || code === 'ControlRight') return 'Ctrl';
  if (code === 'AltLeft' || code === 'AltRight') return 'Alt';
  if (code === 'ShiftLeft' || code === 'ShiftRight') return 'Shift';
  if (code === 'MetaLeft' || code === 'MetaRight') return 'Win';
  return null;
};

const isFunctionKeyName = (name: string): boolean => /^F([1-9]|1[0-9]|2[0-4])$/.test(name);

// recordFromEvent 把一次 keydown 事件转成 RecordedAccelerator。
export const recordFromEvent = (event: KeyboardEvent): RecordedAccelerator => {
  const hasCtrl = event.ctrlKey;
  const hasAlt = event.altKey;
  const hasShift = event.shiftKey;
  const hasWin = event.metaKey;

  const mainModifier = modifierForCode(event.code);
  const activeModifierCount = [hasCtrl, hasAlt, hasShift, hasWin].filter(Boolean).length;
  if (mainModifier && activeModifierCount <= 1) {
    return { accelerator: null, modifiersOnly: true, safe: false };
  }

  const keyName = codeToKeyName(event.code) ?? fallbackVirtualKeyName(event);
  if (!keyName) {
    return { accelerator: null, modifiersOnly: false, safe: false };
  }

  const parts: string[] = [];
  if (hasCtrl && mainModifier !== 'Ctrl') parts.push('Ctrl');
  if (hasAlt && mainModifier !== 'Alt') parts.push('Alt');
  if (hasShift && mainModifier !== 'Shift') parts.push('Shift');
  if (hasWin && mainModifier !== 'Win') parts.push('Win');
  parts.push(keyName);

  const safe = parts.includes('Ctrl') || parts.includes('Alt') || parts.includes('Win') || isFunctionKeyName(keyName);

  return { accelerator: parts.join('+'), modifiersOnly: false, safe };
};

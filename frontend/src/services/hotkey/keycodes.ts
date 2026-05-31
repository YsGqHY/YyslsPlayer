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

// event.code → 主键规范名。覆盖与后端 mainKeys 对齐的集合。
const codeToKeyName = (code: string): string | null => {
  if (/^Key[A-Z]$/.test(code)) return code.slice(3); // KeyA -> A
  if (/^Digit[0-9]$/.test(code)) return code.slice(5); // Digit1 -> 1
  if (/^Numpad[0-9]$/.test(code)) return code.slice(6); // Numpad1 -> 1
  if (/^F([1-9]|1[0-9]|2[0-4])$/.test(code)) return code; // F1..F24
  switch (code) {
    case 'Space':
      return 'Space';
    case 'Enter':
    case 'NumpadEnter':
      return 'Enter';
    case 'Tab':
      return 'Tab';
    case 'Backspace':
      return 'Backspace';
    case 'Escape':
      return 'Escape';
    case 'Insert':
      return 'Insert';
    case 'Delete':
      return 'Delete';
    case 'Home':
      return 'Home';
    case 'End':
      return 'End';
    case 'PageUp':
      return 'PageUp';
    case 'PageDown':
      return 'PageDown';
    case 'ArrowLeft':
      return 'Left';
    case 'ArrowUp':
      return 'Up';
    case 'ArrowRight':
      return 'Right';
    case 'ArrowDown':
      return 'Down';
    default:
      return null;
  }
};

const isModifierCode = (code: string): boolean =>
  code === 'ControlLeft' ||
  code === 'ControlRight' ||
  code === 'AltLeft' ||
  code === 'AltRight' ||
  code === 'ShiftLeft' ||
  code === 'ShiftRight' ||
  code === 'MetaLeft' ||
  code === 'MetaRight';

const isFunctionKeyName = (name: string): boolean => /^F([1-9]|1[0-9]|2[0-4])$/.test(name);

// recordFromEvent 把一次 keydown 事件转成 RecordedAccelerator。
export const recordFromEvent = (event: KeyboardEvent): RecordedAccelerator => {
  const hasCtrl = event.ctrlKey;
  const hasAlt = event.altKey;
  const hasShift = event.shiftKey;
  const hasWin = event.metaKey;

  if (isModifierCode(event.code)) {
    return { accelerator: null, modifiersOnly: true, safe: false };
  }

  const keyName = codeToKeyName(event.code);
  if (!keyName) {
    return { accelerator: null, modifiersOnly: false, safe: false };
  }

  const parts: string[] = [];
  if (hasCtrl) parts.push('Ctrl');
  if (hasAlt) parts.push('Alt');
  if (hasShift) parts.push('Shift');
  if (hasWin) parts.push('Win');
  parts.push(keyName);

  const safe = hasCtrl || hasAlt || hasWin || isFunctionKeyName(keyName);

  return { accelerator: parts.join('+'), modifiersOnly: false, safe };
};

// NativeDialogsService：统一封装 Wails v3 的原生 dialog API。
//
// 规范：项目内所有"打开文件 / 保存文件 / 选择文件夹 / 确认对话框 / 错误提示"都
// 通过本服务调用，禁止使用浏览器 alert / confirm / prompt（在 webview 里观感差且
// 在 Wails 里行为不一致）。
//
// 实现细节：
//   - SaveFile / OpenFile / Info / Warning / Error / Question 全部走 @wailsio/runtime
//   - 把 PascalCase 字段名收口为本地的 camelCase API，对调用方更友好
//   - 用户取消时统一返回 null（而不是空字符串），避免上层判空写 if (path === '')
import { Dialogs, System } from '@wailsio/runtime';

export interface FileFilter {
  displayName: string;
  pattern: string; // e.g. "*.json"
}

export interface OpenFileOptions {
  title?: string;
  message?: string;
  buttonText?: string;
  directory?: string;
  filters?: FileFilter[];
  allowsMultiple?: boolean;
  // 选择目录而非文件
  chooseDirectories?: boolean;
}

export interface SaveFileOptions {
  title?: string;
  message?: string;
  buttonText?: string;
  directory?: string;
  filename?: string;
  filters?: FileFilter[];
  canCreateDirectories?: boolean;
}

export interface ConfirmOptions {
  title?: string;
  message: string;
  okLabel?: string;
  cancelLabel?: string;
}

const mapFilters = (filters?: FileFilter[]) =>
  filters?.map((f) => ({ DisplayName: f.displayName, Pattern: f.pattern }));

const normalizeDialogChoice = (choice: unknown): string => String(choice ?? '').trim().replace(/\(&?.+\)$/u, '').toLocaleLowerCase();

const isNegativeChoice = (choice: unknown, cancelLabel: string): boolean => {
  if (choice === false || choice === 1) return true;
  const normalized = normalizeDialogChoice(choice);
  const normalizedCancel = normalizeDialogChoice(cancelLabel);
  return normalized === '' || normalized === normalizedCancel || ['1', 'false', 'no', 'n', 'cancel', '否', '取消'].includes(normalized);
};

const isAffirmativeChoice = (choice: unknown, okLabel: string, cancelLabel: string): boolean => {
  if (choice === true || choice === 0) return true;
  const normalized = normalizeDialogChoice(choice);
  const normalizedOk = normalizeDialogChoice(okLabel);
  if (isNegativeChoice(choice, cancelLabel)) return false;
  return normalized === normalizedOk || ['0', 'true', 'yes', 'y', 'ok', '是', '确定'].includes(normalized) || normalized !== '';
};

type MessageDialogFn = (options: { Title?: string; Message?: string; Buttons?: Array<{ Label: string; IsDefault?: boolean; IsCancel?: boolean }> }) => Promise<unknown>;

// showMessageDialog drives Info / Warning / Error dialogs with an explicit OK
// button. This is mandatory on Windows: the Wails backend blocks on a response
// channel that is only fed by a button's OnClick callback, and it auto-injects
// a default button ONLY on darwin. With no buttons on Windows the channel never
// receives a value, so the JS Promise never resolves and any caller awaiting it
// hangs forever (which can wedge UI "busy" state). The Windows MessageBox maps
// MB_OK to the return string "Ok", so the button Label must be exactly "Ok" for
// the backend's label match to fire its callback.
const showMessageDialog = async (fn: MessageDialogFn, title: string, message: string): Promise<void> => {
  await fn({ Title: title, Message: message, Buttons: [{ Label: 'Ok', IsDefault: true }] });
};

export const NativeDialogs = {
  // 打开文件选择器（单选）。取消返回 null。
  async openFile(options: OpenFileOptions = {}): Promise<string | null> {
    const result = await Dialogs.OpenFile({
      Title: options.title,
      Message: options.message,
      ButtonText: options.buttonText,
      Directory: options.directory,
      Filters: mapFilters(options.filters),
      CanChooseDirectories: options.chooseDirectories ?? false,
      CanChooseFiles: !options.chooseDirectories,
      AllowsMultipleSelection: false,
    });
    return result && result !== '' ? result : null;
  },

  // 打开文件选择器（多选）。取消返回空数组。
  async openFiles(options: OpenFileOptions = {}): Promise<string[]> {
    const result = await Dialogs.OpenFile({
      Title: options.title,
      Message: options.message,
      ButtonText: options.buttonText,
      Directory: options.directory,
      Filters: mapFilters(options.filters),
      CanChooseDirectories: options.chooseDirectories ?? false,
      CanChooseFiles: !options.chooseDirectories,
      AllowsMultipleSelection: true,
    });
    return Array.isArray(result) ? result : [];
  },

  // 选择文件夹。取消返回 null。
  async openDirectory(options: Omit<OpenFileOptions, 'chooseDirectories' | 'filters'> = {}): Promise<string | null> {
    const result = await Dialogs.OpenFile({
      Title: options.title,
      Message: options.message,
      ButtonText: options.buttonText,
      Directory: options.directory,
      CanChooseDirectories: true,
      CanChooseFiles: false,
      CanCreateDirectories: true,
      AllowsMultipleSelection: false,
    });
    return result && result !== '' ? result : null;
  },

  // 保存文件对话框。取消返回 null。
  async saveFile(options: SaveFileOptions = {}): Promise<string | null> {
    const result = await Dialogs.SaveFile({
      Title: options.title,
      Message: options.message,
      ButtonText: options.buttonText,
      Directory: options.directory,
      Filename: options.filename,
      Filters: mapFilters(options.filters),
      CanCreateDirectories: options.canCreateDirectories ?? true,
    });
    return result && result !== '' ? result : null;
  },

  // 二选一确认。OK 返回 true，Cancel / 关闭返回 false。
  async confirm(options: ConfirmOptions): Promise<boolean> {
    const okLabel = options.okLabel ?? 'OK';
    const cancelLabel = options.cancelLabel ?? 'Cancel';

    // Windows 下 Wails 的 Question 对话框被强制渲染为系统 MessageBox(MB_YESNO)，
    // 自定义按钮文案不会显示，系统只会回传 "Yes" / "No"。而后端仅在
    // button.Label === 回传值 时才向响应 channel 写值，否则协程会永久阻塞、
    // 前端 Promise 永不 resolve。因此 Windows 必须把按钮 Label 设为 "Yes" / "No"。
    if (System.IsWindows()) {
      const choice = await Dialogs.Question({
        Title: options.title,
        Message: options.message,
        Buttons: [
          { Label: 'No', IsCancel: true },
          { Label: 'Yes', IsDefault: true },
        ],
      });
      return isAffirmativeChoice(choice, 'Yes', 'No');
    }

    const choice = await Dialogs.Question({
      Title: options.title,
      Message: options.message,
      Buttons: [
        { Label: okLabel, IsDefault: true },
        { Label: cancelLabel, IsCancel: true },
      ],
    });
    return isAffirmativeChoice(choice, okLabel, cancelLabel);
  },

  async info(title: string, message: string): Promise<void> {
    await showMessageDialog(Dialogs.Info, title, message);
  },

  async warning(title: string, message: string): Promise<void> {
    await showMessageDialog(Dialogs.Warning, title, message);
  },

  async error(title: string, message: string): Promise<void> {
    await showMessageDialog(Dialogs.Error, title, message);
  },
} as const;

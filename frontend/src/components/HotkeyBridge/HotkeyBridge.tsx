import { useHotkeyDispatcher } from '@/shared/hotkeys';

// HotkeyBridge 是无渲染组件：挂载全局热键派发器。
// 放在 AppLayout 内（RouterProvider 之下），确保派发时 useRouter 可用。
export const HotkeyBridge = (): null => {
  useHotkeyDispatcher();
  return null;
};

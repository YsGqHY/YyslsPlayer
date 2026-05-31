// 全局热键的前端动作注册表。
//
// 后端发出 hotkey:triggered 事件后，HotkeyBridge 派发到这里注册的处理器。
// 页面级 UI（演奏面板 / 预览面板）在挂载时注册自己能处理的动作，卸载时注销。
// 未注册的动作安全忽略（no-op）。
//
// 注意：stop / 紧急松键 / 暂停继续 已由后端直接作用于 player（游戏聚焦也生效），
// 这里的处理器主要用于：
//   - play-pause 在空闲态时由前端发起 start（需要当前 PlayPlan）
//   - preview-toggle 切换纯前端的 Web Audio 试听
//   - 任何动作触发后的额外 UI 反馈

import type { HotkeyAction, HotkeyTriggeredEvent } from '@/services';

export type HotkeyHandler = (event: HotkeyTriggeredEvent) => void;

const handlers = new Map<string, Set<HotkeyHandler>>();

// registerHotkeyHandler 注册某动作的处理器，返回注销函数。
export const registerHotkeyHandler = (action: HotkeyAction, handler: HotkeyHandler): (() => void) => {
  let set = handlers.get(action);
  if (!set) {
    set = new Set();
    handlers.set(action, set);
  }
  set.add(handler);
  return () => {
    const current = handlers.get(action);
    if (!current) return;
    current.delete(handler);
    if (current.size === 0) handlers.delete(action);
  };
};

// dispatchHotkey 把事件派发给已注册的处理器。
export const dispatchHotkey = (event: HotkeyTriggeredEvent): void => {
  const set = handlers.get(event.actionId);
  if (!set) return;
  for (const handler of set) {
    try {
      handler(event);
    } catch {
      // 单个处理器异常不应阻断其它处理器。
    }
  }
};

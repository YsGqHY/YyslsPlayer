import { useEffect } from 'react';
import { useRouter } from '@/router';
import { HotkeyService, PlayerService } from '@/services';
import { dispatchHotkey } from './registry';

// useHotkeyDispatcher 订阅后端 hotkey:triggered 事件，统一派发：
//   - openLibrary / openSettings 类（导航）→ 这里直接 navigate（当前未默认绑定，预留）
//   - emergency-release → 幂等再调一次 PlayerService.releaseAll() 兜底
//   - 其余 → 交给页面级注册的处理器（registry）
//
// 后端已对 stop / 暂停继续 / 紧急松键直接作用于 player，因此前端无需重复执行；
// 只在 handledByBackend=false 时才把动作交给页面处理器（如空闲态的 play 需要 PlayPlan）。
export const useHotkeyDispatcher = (): void => {
  const router = useRouter();

  useEffect(() => {
    const off = HotkeyService.onTriggered((event) => {
      switch (event.actionId) {
        case 'emergency-release':
          // 后端已执行；前端再幂等兜底一次，避免任何遗漏的按下态。
          void PlayerService.releaseAll().catch(() => undefined);
          dispatchHotkey(event);
          return;
        case 'open-library':
          router.navigate('library');
          return;
        case 'open-settings':
          router.navigate('settings');
          return;
        default:
          // play-pause / stop / preview-toggle：交给页面级处理器。
          dispatchHotkey(event);
          return;
      }
    });
    return off;
  }, [router]);
};

# Foundation 前端开发规范

本规范约束 `frontend/src/` 下所有 React 代码的组织方式。当前应用是 Windows-only 的《燕云流音》桌面端，前端负责 MIDI 曲库、编辑、预览、演奏控制和设置界面。

技术栈：React 19、TypeScript strict、MUI 9、Vite 8、MVVM 架构。

## 1. 架构：MVVM

| 层 | 角色 | 文件命名 |
|----|------|----------|
| View | 纯展示组件，仅消费 ViewModel 输出，不直接调用业务/服务 | `<Name>.tsx` |
| ViewModel | 状态、副作用、业务编排，导出一个 `use<Name>` hook | `use<Name>.ts` |
| Style | 该组件的所有 sx prop 与样式常量 | `<Name>.styles.ts` |
| Service | 后端通信封装（`bindings/*` 包装层） | `services/<domain>/<Name>Service.ts` |
| Shared | 多个组件复用的工具 / 类型 / hooks | `shared/...` |

铁律：

- View 文件里不写业务状态机、API 调用或数据转换。
- ViewModel hook 不返回 JSX，只返回数据与回调。
- ViewModel 不直接 import `@wailsio/runtime` 或 `@bindings/*`，必须通过 `services/` 调用。
- 所有人类可见字符串必须走 i18n 的 `t('key')`。

## 2. 目录与文件结构

每个组件 / 页面 = 一个文件夹，包含其全部内容：

```text
ComponentName/
├── ComponentName.tsx
├── useComponentName.ts
├── ComponentName.styles.ts
├── ComponentName.types.ts
├── SubComponent.tsx
├── useSubComponent.ts
└── index.ts
```

顶层目录约定：

```text
src/
├── components/         # 跨页面共享组件
├── pages/              # 路由级页面
├── services/           # 后端通信封装
├── shared/             # 事件、快捷键派发、共享工具
│   ├── events.ts       # player:* / hotkey:* 事件名
│   └── hotkeys/        # 全局快捷键前端派发桥
├── styles/             # 主题系统
├── App.tsx             # Provider 装配
└── main.tsx            # createRoot 入口
```

## 3. 命名规范

| 对象 | 风格 | 示例 |
|------|------|------|
| 组件文件 / 文件夹 / Type | PascalCase | `PerformPanel`, `UseHomePageResult` |
| 普通函数 / 变量 / hook | camelCase | `usePerformPanel`, `formatTime` |
| ViewModel hook | `use<ComponentName>` | `useTitleBar`, `useEditorPage` |
| 服务对象 | `<Domain>Service` | `MidiService`, `PlayerService`, `HotkeyService` |
| 事件名常量 | `AppEvents.<Name>` | `AppEvents.PlayerState` |
| Style 模块 | `<name>Styles` | `titleBarStyles` |
| Boolean 状态 | `is/has/should` 前缀 | `isMaximised`, `hasError` |

## 4. TypeScript 规则

- `tsconfig.json` 已开启 `strict` + `noUncheckedIndexedAccess`。
- 所有导出函数 / 组件 props 必须显式标注类型。
- 禁止 `any`；外部输入用 `unknown` 再窄化。
- React 组件不写 `React.FC`，直接使用普通函数组件。
- 公共 props 用 `interface`，联合 / 交叉用 `type`。

## 5. 样式规范

- 不写 CSS 文件，所有组件样式集中在 `<Name>.styles.ts`。
- 颜色只能从 `theme.palette.foundation.*` 取。
- 间距优先使用 `theme.spacing` 或 MUI spacing 数字。
- 按钮 / IconButton、输入框、Paper、容器遵循项目方形圆角设计。
- 拖拽区域用 `style={{ '--wails-draggable': 'drag' }}`，按钮区用 `'no-drag'` 阻止穿透。

## 6. 服务层规范

`services/<domain>/<Name>Service.ts` 只做两件事：

- import Wails bindings 或 runtime API。
- 暴露调用方友好的 Promise 方法和前端 camelCase 类型。

示例：

```ts
import { Service as Binding } from '@bindings/YyslsPlayer/internal/services/player';

export const PlayerService = {
  stop(sessionId: string): Promise<void> {
    return Binding.Stop(sessionId);
  },
} as const;
```

业务变更只改 Service，不把 bindings 泄漏到 ViewModel。

## 7. 事件订阅规范

- 事件名集中在 `shared/events.ts`，不要在组件内手写字符串。
- 当前业务事件包括 `player:state`、`player:position`、`player:error`、`hotkey:triggered`。
- Service 层封装通用订阅，例如 `PlayerService.onState()`、`HotkeyService.onTriggered()`。
- 单页面专用订阅可以写在该页面的 `use<Page>.ts` 中。
- 订阅必须在 `useEffect` cleanup 中取消。

## 8. Windows-only UI 边界

当前前端只面向 Windows 桌面端：

- 标题栏始终使用自绘三联按钮（最小化 / 最大化 / 关闭）。
- 不保留其它平台标题栏分支。
- 全局快捷键设置页面面向 Windows `RegisterHotKey`，不展示降级提示。

## 9. 路径别名

- `@/*` -> `src/*`
- `@bindings/*` -> `bindings/*`

正确：

```ts
import { MidiService, PlayerService } from '@/services';
```

错误：

```ts
import { MidiService } from '../../../services/midi/MidiService';
```

## 10. 检查清单

提交前自查：

- [ ] 组件有独立文件夹，View / ViewModel / Style 三件齐全。
- [ ] View 不写业务，ViewModel 不返回 JSX。
- [ ] ViewModel 不直接 import bindings 或 runtime。
- [ ] 没有硬编码颜色。
- [ ] 公共组件有 props interface 与 index.ts 出口。
- [ ] `pnpm --dir frontend typecheck` 通过。
- [ ] `pnpm --dir frontend build` 通过。
- [ ] 没有 `console.log`、没有死代码。

## 11. 反例对照

直接在 View 里调 bindings 是反例：

```tsx
import { Service } from '@bindings/YyslsPlayer/internal/services/player';

const onStop = () => Service.Stop(sessionId);
```

应通过 ViewModel + Service：

```ts
import { PlayerService } from '@/services';

const stop = () => PlayerService.stop(sessionId);
```

## 12. 何时打破规范

规范是默认值，不是教条。出现以下情况可绕行，但需要在代码审查中说明原因：

- 组件只有极少展示逻辑，ViewModel 可省略。
- 第三方库强制要求 hook 形态，且封装收益很低。
- 实验性 spike 放在 `src/experimental/`，不进入 `pages/` 或 `components/`。

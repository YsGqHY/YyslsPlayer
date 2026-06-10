import type { Messages } from '@/i18n';

// HomePage 中文文案。挂在 home.* 命名空间下。
// 由 src/pages/HomePage/lang/index.ts 通过 localeRegistry.extend('zh-CN', ...) 注入。
export const homePageZhCN: Messages = {
  home: {
    eyebrow: '燕云流音 · 燕云十六声 36 键模式',
    hero: '燕云流音 MIDI 演奏工具',
    subtitle: '燕云流音支持导入 MIDI 后，按高低音、音区映射和播放倍速配置进行预览或按键模拟演奏。当前业务目标仅支持 36 键模式。',
    footer: {
      windowsOnly: 'Windows 专用演奏链路：MIDI 解析、PlayPlan 调度、SendInput 按键模拟。',
      scope: '当前版本仅面向《燕云十六声》36 键模式。',
    },
    edition: {
      lite: '轻量版',
      completion: '完整版',
    },
  },
};

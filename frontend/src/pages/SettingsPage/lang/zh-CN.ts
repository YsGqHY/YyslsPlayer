import type { Messages } from '@/i18n';

// SettingsPage 顶层文案（中文）：仅包含"框架"层级 ——
//   - eyebrow / title（页头）
//   - list.*（左列入口）
//   - themes.*（主题预设的展示名，被 personalization 与全局通用引用）
// 子页面专属文案在 SettingsPage/<name>/lang/ 下，由 register<Name>Locales 注册。
export const settingsPageZhCN: Messages = {
  settings: {
    eyebrow: '设置',
    title: '偏好设置',
    list: {
      personalization: {
        label: '个性化',
        description: '主题、显示偏好与自定义配色',
      },
      midiDefaults: {
        label: 'MIDI 默认值',
        description: '音域、移调、倍速与超范围策略',
      },
      library: {
        label: '曲库与导入',
        description: '列表加载数量和导入后跳转',
      },
      playback: {
        label: '演奏控制',
        description: '倒计时与调度窗口',
      },
      shortcuts: {
        label: '快捷键',
        description: '全局热键，切到游戏也能控制演奏',
      },
      preview: {
        label: '预览与时间轴',
        description: '试听音色、音量与 PianoRoll 上限',
      },
      language: {
        label: '语言',
        description: '切换界面语言',
      },
      database: {
        label: '数据存储',
        description: '数据库位置与占用空间',
      },
    },
    themes: {
      system: { label: '跟随系统', description: '随操作系统的明暗模式自动切换' },
      light: { label: '明亮', description: '默认白色主题，适合白天使用' },
      dark: { label: '黑暗', description: '深灰主题，降低低光环境下的眩光' },
      obsidian: { label: '黑曜', description: '纯黑画布 + 紫罗兰强调，对比度最高' },
      custom: { label: '自定义', description: '你保存的自定义配色' },
    },
  },
};

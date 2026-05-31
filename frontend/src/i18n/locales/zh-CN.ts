import type { Locale } from '../types';

// 简体中文（默认）—— 公用文案。
// 仅放"跨页面 / 框架级"的 keys：
//   - app.*：应用元数据
//   - sidebar.* / titleBar.*：基座组件
//   - route.*：路由 label（多处消费：Sidebar tooltip / 设置页 / 未来面包屑）
//   - common.*：通用术语（保存 / 取消 / 删除 / 是 / 否 / 加载中 / 错误 / 跟随系统）
// 页面级文案（home / settings 等）不要放这里 —— 各 page 自带 lang/ 目录注册。
//
// 命名约定：点路径分组；模板插值用 {{var}}（与 t(key, vars) 配合）。
export const zhCN: Locale = {
  code: 'zh-CN',
  englishName: 'Chinese (Simplified)',
  nativeName: '简体中文',
  messages: {
    app: {
      title: '燕云流音',
    },
    sidebar: {
      brand: 'Y',
      navAriaLabel: '主导航',
    },
    titleBar: {
      controls: {
        minimise: '最小化',
        maximise: '最大化 / 还原',
        close: '关闭',
      },
    },
    route: {
      home: '首页',
      library: '曲库',
      editor: '编辑器',
      settings: '设置',
    },
    common: {
      save: '保存',
      cancel: '取消',
      remove: '删除',
      reset: '重置',
      yes: '是',
      no: '否',
      loading: '加载中…',
      error: '错误',
      followSystem: '跟随系统',
      auto: '自动',
    },
    qualityReport: {
      metrics: {
        playableRatio: '可演奏率',
        playableSummary: '{{playable}} / {{total}} 个音符可演奏',
        noteRange: '原始音域',
        rawNotes: '总音符 {{count}}',
        mappedRange: '映射 lane',
        playableNotes: '可演奏 {{count}}',
        outOfRange: '超范围',
        dropFoldClamp: 'drop {{dropped}} / fold {{folded}} / nearest {{clamped}}',
        blackKeyCount: '半音 lane',
        chordDensity: '最大同时按键 {{count}}',
        trackChannel: '轨道 / 通道',
        trackChannelHint: 'MIDI 结构复杂度',
        suggestion: '建议移调',
        octaveShift: '建议八度 {{shift}}',
      },
      warnings: {
        none: '未发现明显风险',
        out_of_range: '存在超范围音符',
        dropped_notes: '当前策略会丢弃部分音符',
        high_chord_density: '和弦密度较高',
      },
    },
    previewPanel: {
      eyebrow: '试听预览',
      title: 'Web Audio Preview',
      subtitle: '时长 {{duration}} · {{frames}} 个 frame',
      empty: '加载曲目后即可试听。',
      loading: '预览引擎加载中…',
      status: '状态 {{state}} · 当前按下 {{active}} 个 lane',
      lanes: {
        title: '当前活跃 lane',
      },
      seek: {
        aria: '试听进度，拖拽或用方向键跳转',
      },
      actions: {
        play: '播放',
        resume: '继续',
        pause: '暂停',
        stop: '停止',
        restart: '重播',
        refresh: '刷新预览',
      },
      states: {
        idle: '空闲',
        playing: '播放中',
        paused: '已暂停',
        stopped: '已停止',
      },
      errors: {
        prefix: '预览失败：',
        unknown: '未知错误',
      },
    },
    performPanel: {
      eyebrow: '游戏演奏',
      title: 'SendInput Performance',
      subtitle: '时长 {{duration}} · {{frames}} 个 frame · {{mode}}',
      empty: '生成 PlayPlan 后即可开始实际演奏。',
      loading: '演奏计划加载中…',
      status: '状态 {{state}} · 进度 {{progress}}',
      mode: {
        dryRun: 'Dry-run',
        real: '实际按键',
      },
      actions: {
        start: '开始演奏',
        resume: '继续',
        pause: '暂停',
        stop: '停止',
        releaseAll: '紧急释放',
      },
      fields: {
        dryRun: 'Dry-run 模式',
        dryRunHelper: '不注入系统按键，只走调度和日志链路。',
        realHelper: '会调用系统按键模拟，请先切到游戏 36 键模式。',
        lookahead: 'Lookahead ms',
      },
      stats: {
        state: '状态',
        session: 'Session',
        none: '无',
        lookahead: '调度窗口',
        lookaheadValue: '{{value}} ms',
      },
      states: {
        idle: '空闲',
        ready: '就绪',
        playing: '演奏中',
        paused: '已暂停',
        completed: '已完成',
        stopped: '已停止',
        error: '错误',
      },
      countdown: '演奏将在 {{seconds}} 秒后开始，请切换到游戏窗口。',
      warnings: {
        realMode: '开始前请确认游戏已处于 36 键演奏界面。',
      },
      errors: {
        prefix: '演奏失败：',
      },
    },
    pianoRoll: {
      eyebrow: '36 Lane 时间轴',
      title: 'Piano Roll View',
      subtitle: '时长 {{duration}} · {{notes}} 个音符块',
      loading: '正在生成时间轴…',
      empty: '生成 PlayPlan 后显示 36 键时间轴。',
      truncated: '已渲染 {{rendered}} 个音符块，隐藏 {{hidden}} 个以保持界面流畅。',
      noteTitle: 'lane {{lane}} · source {{source}} · mapped {{normalized}} · start {{start}} · duration {{duration}}ms',
      stats: {
        rendered: '渲染 {{count}}',
        active: '活跃 lane {{count}}',
        position: '位置 {{position}}',
      },
    },
  },
};

import type { Messages } from '@/i18n';

export const midiDefaultsZhCN: Messages = {
  settings: {
    midiDefaults: {
      title: 'MIDI 默认值',
      hint: '配置全局默认 MIDI profile。未单独保存项目配置的曲目会使用这些默认值；项目级配置仍可在编辑器中覆盖。',
      actions: {
        save: '保存默认值',
        saving: '保存中…',
        reload: '重新载入',
        reset: '恢复内置默认值',
      },
      feedback: {
        loading: '正在读取全局默认配置…',
        saved: '全局默认配置已保存。',
        failed: '保存失败：{{message}}',
      },
      fields: {
        name: {
          label: 'Profile 名称',
          description: '全局默认 profile 的展示名称。',
        },
        keymapProfileId: {
          label: 'Keymap Profile ID',
          description: '用于生成 PlayPlan 的 36 键物理映射 profile。当前默认 profile id 为 1。',
        },
        baseNote: {
          label: '最低音 MIDI Note',
          description: '36 lane 的最低音，默认 48 即 C3。允许 0..127。',
        },
        transpose: {
          label: '半音移调',
          description: '整体按半音移动，允许 -24..24。',
        },
        octaveShift: {
          label: '八度偏移',
          description: '整体按八度移动，允许 -3..3。',
        },
        speed: {
          label: '播放倍速',
          description: '影响预览和演奏共用的 PlayPlan 时间轴，允许 0.25..3.0。',
        },
        minPressMs: {
          label: '最短按键 ms',
          description: '过短音符会拉长到该时长，降低漏键概率。允许 10..300ms。',
        },
        releaseGapMs: {
          label: '同键间隔 ms',
          description: '同一物理键连续触发时保留的松开间隔，允许 0..200ms。',
        },
        outOfRangePolicy: {
          label: '超范围策略',
          description: 'MIDI note 映射到 36 lane 外时的处理方式。默认 drop 最安全。',
        },
      },
      policies: {
        drop: '丢弃超范围音符',
        octaveFold: '按八度折叠',
        nearest: '映射到最近 lane',
      },
    },
  },
};

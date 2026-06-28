import type { Messages } from '@/i18n';

export const databaseZhCN: Messages = {
  settings: {
    database: {
      title: '数据存储',
      hint: '应用所有偏好与设置保存在本地数据文件中。可以把数据文件移到任意你信任的位置；切换时会自动迁移数据。',
      currentPathLabel: '当前位置',
      defaultPathLabel: '默认位置',
      sizeLabel: '占用空间',
      customBadge: '自定义',
      defaultBadge: '默认',
      actions: {
        change: '更改位置…',
        reset: '重置为默认',
      },
      // 原生「另存为」对话框文案
      dialog: {
        title: '选择新的数据文件位置',
        message: '现有数据会自动复制到所选位置。',
        confirm: '保存到此处',
        filterDb: '数据文件 (*.json)',
      },
      // 重置确认对话框文案
      confirmReset: {
        title: '恢复默认位置',
        message: '将把数据文件迁回平台默认目录。是否继续？',
        ok: '迁回默认',
        cancel: '取消',
      },
      // 单表清空确认对话框
      confirmClear: {
        title: '清空 {{label}}',
        message: '将删除「{{label}}」中的全部 {{rows}} 行数据。该操作不可撤销，是否继续？',
        ok: '清空',
        cancel: '取消',
      },
      feedback: {
        changing: '正在迁移数据…',
        changed: '已切换到新位置',
        resetting: '正在恢复默认位置…',
        reset: '已恢复默认位置',
        clearing: '正在清空「{{label}}」…',
        cleared: '已清空「{{label}}」',
        failed: '操作失败：{{message}}',
      },
      charts: {
        share: {
          title: '空间占比',
          hint: '各类数据在存储中的相对占用比例。'
        },
        bytes: {
          title: '字节用量',
          hint: '各类数据在存储中的占用空间。'
        },
        empty: '暂无数据',
        estimatedNote: '占用为估算值。',
        totalCaption: '数据总量',
      },
      // 表清单
      tables: {
        title: '数据集合',
        empty: '没有可展示的数据集合。',
        meta: '行数 {{rows}} · {{size}}',
        clear: '清空',
        clearAria: '清空 {{label}}',
        protected: '受保护',
        protectedAria: '{{label}} 受保护，不可一键清空',
        estimatedTag: '估算',
        // 每个数据集合的标签 / 描述：键名与 internal/storage/models.go 的 ModelDescriptor.LabelKey 对齐
        preferences: {
          label: '行为偏好',
          description: '界面开关 / 显隐设置等键值对，可重建。',
        },
        appSettings: {
          label: '应用设置',
          description: '主题选择 / 自定义主题 / 语言。涉及视觉一致性，不允许一键清空。',
        },
        midiProjects: {
          label: 'MIDI 项目',
          description: '已导入曲目的元信息、文件 hash 和默认配置引用，受保护。',
        },
        midiEvents: {
          label: 'MIDI 事件',
          description: '标准化后的 note 事件时间轴，随 MIDI 项目一起管理。',
        },
        midiProfiles: {
          label: 'MIDI 配置',
          description: '音域、移调、倍速、超范围策略和按键时长配置，受保护。',
        },
        keymap36: {
          label: '36 键映射',
          description: 'lane 到物理按键 / 扫描码 / modifier 的映射 profile，受保护。',
        },
        playHistory: {
          label: '播放历史',
          description: '演奏历史记录，可重建。'
        },
        hotkeyBindings: {
          label: '全局快捷键',
          description: '各动作的全局热键绑定，受保护；如需重置请到快捷键设置页操作。',
        },
        macroProfiles: {
          label: '按键宏',
          description: '用户创建的宏配置、触发组合键和运行策略，受保护。',
        },
        macroSteps: {
          label: '宏积木块',
          description: '按键宏的线性动作块、延迟和按键参数，随宏一起管理。',
        },
      },
    },
  },
};

import type { Messages } from '@/i18n';

export const librarySettingsZhCN: Messages = {
  settings: {
    library: {
      title: '曲库与导入',
      hint: '配置曲库加载数量和 MIDI 导入后的默认跳转行为。',
      fields: {
        autoOpen: {
          label: '导入后自动打开编辑器',
          description: '导入成功后直接进入编辑器页面；关闭时仍留在曲库工作台并选中新曲目。',
        },
        listLimit: {
          label: '曲库列表加载上限',
          description: '每次刷新最多读取的曲目数量，允许 5..10000。',
        },
      },
      actions: {
        reset: '恢复曲库默认值',
      },
    },
  },
};

import type { Messages } from '@/i18n';

export const playbackZhCN: Messages = {
  settings: {
    playback: {
      title: '演奏控制',
      hint: '配置 SendInput 演奏的倒计时和调度窗口。',
      fields: {
        lookahead: {
          label: '调度窗口 ms',
          description: '后端调度提前量，允许 5..50ms；较大值更稳，较小值响应更紧。',
        },
        countdown: {
          label: '开始倒计时秒数',
          description: '开始演奏前等待 0..10 秒，留出切换焦点到游戏窗口的时间。',
        },
      },
      actions: {
        reset: '恢复演奏默认值',
      },
    },
  },
};

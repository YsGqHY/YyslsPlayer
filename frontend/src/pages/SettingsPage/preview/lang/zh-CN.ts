import type { Messages } from '@/i18n';

export const previewSettingsZhCN: Messages = {
  settings: {
    preview: {
      title: '预览与时间轴',
      hint: '配置应用内 Web Audio 试听和 PianoRoll 渲染上限。预览仍消费后端生成的同一套 PlayPlan。',
      fields: {
        volume: {
          label: '预览音量',
          description: 'Web Audio 主增益，允许 0..0.5；默认 0.08。',
        },
        waveform: {
          label: '振荡器波形',
          description: '简单预览音色，不改变 MIDI 映射或游戏演奏。',
        },
        progressHz: {
          label: '预览刷新频率',
          description: '预览进度刷新频率，允许 1..30 Hz。',
        },
        pianoRollMaxNotes: {
          label: '时间轴最大渲染音符',
          description: '限制 PianoRoll 一次渲染的音符块数量，避免大 MIDI 卡顿。允许 100..5000。',
        },
      },
      waveforms: {
        sine: '正弦波',
        triangle: '三角波',
        square: '方波',
        sawtooth: '锯齿波',
      },
      actions: {
        reset: '恢复预览默认值',
      },
    },
  },
};

import type { Messages } from '@/i18n';

export const shortcutsZhCN: Messages = {
  settings: {
    shortcuts: {
      title: '全局快捷键',
      hint: '注册 Windows 系统级热键，即使焦点在《燕云十六声》游戏窗口也能控制演奏。停用某项后该热键不再注册。',
      recording: '请按下组合…',
      recordAria: '为「{{action}}」录制快捷键',
      enableAria: '启用 / 停用「{{action}}」快捷键',
      error: '操作失败：{{message}}',
      actions: {
        reset: '恢复默认快捷键',
        'play-pause': {
          label: '播放 / 暂停',
          description: '演奏中暂停、暂停时继续。空闲时请在演奏面板点击开始。',
        },
        stop: {
          label: '停止演奏',
          description: '停止当前演奏会话并松开所有按键。',
        },
        'preview-toggle': {
          label: '试听 / 暂停预览',
          description: '切换 Web Audio 试听播放（与游戏演奏相互独立）。',
        },
        'emergency-release': {
          label: '紧急松开全部按键',
          description: '立即释放所有按下的按键，防止卡键。最关键的安全兜底。',
        },
      },
      status: {
        listening: '正在监听，请按下含 Ctrl / Alt / Win 的组合，或功能键 F1–F12。按 Esc 取消。',
        unsafe: '该组合会占用系统输入，请改用带 Ctrl / Alt / Win 的组合或功能键。',
        invalid: '无法识别该按键，请换一个组合。',
        conflict: '与其它快捷键冲突，请改键。',
        occupied: '该组合已被其它程序占用，未能注册。',
        failed: '注册失败，请尝试更换组合。',
      },
    },
  },
};

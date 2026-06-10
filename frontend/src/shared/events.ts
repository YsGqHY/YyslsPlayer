// 应用启动后由后端推送的事件名集合，
// 与 internal/events/events.go 和 internal/services/player/types.go 中的事件常量保持同步。
export const AppEvents = {
  PlayerState: 'player:state',
  PlayerPosition: 'player:position',
  PlayerError: 'player:error',
  HotkeyTriggered: 'hotkey:triggered',
  MidiFilesDropped: 'midi:filesDropped',
  TranscriptionTaskProgress: 'transcription:task:progress',
  TranscriptionTaskCompleted: 'transcription:task:completed',
  TranscriptionTaskFailed: 'transcription:task:failed',
  TranscriptionTaskCancelled: 'transcription:task:cancelled',
  TranscriptionFfmpegInstalled: 'transcription.ffmpeg.installed',
  TranscriptionFfmpegInstallFailed: 'transcription.ffmpeg.failed',
} as const;

export type AppEventName = (typeof AppEvents)[keyof typeof AppEvents];

export interface WailsEventPayload<T> {
  data: T;
  name: string;
  sender?: string;
}

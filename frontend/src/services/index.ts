export { BrowserService } from './browser/BrowserService';
export { PreferencesService } from './preferences/PreferencesService';
export { AppearanceService, type ImportedBackgroundImage } from './appearance/AppearanceService';
export {
  AppSettingsService,
  type AppSettingsSnapshot,
} from './appsettings/AppSettingsService';
export {
  StorageService,
  type StorageStats,
  type TableInfo,
  type TableStats,
} from './storage/StorageService';
export {
  NativeDialogs,
  type OpenFileOptions,
  type SaveFileOptions,
  type ConfirmOptions,
  type FileFilter,
} from './dialogs/NativeDialogs';
export { EditorSelectionService } from './editor/EditorSelectionService';
export {
  PreviewEngine,
  PreviewEngineService,
  type PreviewEngineOptions,
  type PreviewEngineSnapshot,
  type PreviewEngineState,
} from './preview/PreviewEngine';
export {
  MidiService,
  type OutOfRangePolicy,
  type KeyFrameAction,
  type ListProjectsRequest,
  type ImportBatchStatus,
  type ImportBatchItem,
  type ImportBatchResult,
  type ProjectBatchManageStatus,
  type ProjectBatchManageItem,
  type ProjectBatchManageResult,
  type MidiProjectSummary,
  type MidiProfile,
  type KeymapLane,
  type KeymapProfile,
  type NoteRange,
  type MappedRange,
  type QualityReport,
  type MidiProjectDetail,
  type MidiConfigSnapshot,
  type KeyFrame,
  type PlayPlan,
} from './midi/MidiService';
export {
  PlayerService,
  type StartPlayerRequest,
  type PlayerState,
  type PlayerSession,
  type PlayerStateSnapshot,
  type PlayerPositionEvent,
  type PlayerErrorEvent,
} from './player/PlayerService';
export {
  HotkeyService,
  type HotkeyAction,
  type HotkeyBinding,
  type HotkeyState,
  type HotkeyTriggeredEvent,
} from './hotkey/HotkeyService';
export { recordFromEvent, type RecordedAccelerator } from './hotkey/keycodes';
export {
  MacroService,
  type AssignableKey,
  type MacroDetail,
  type MacroDeviceKind,
  type MacroErrorEvent,
  type MacroInterruptPolicy,
  type MacroRepeatMode,
  type MacroRunState,
  type MacroState,
  type MacroStep,
  type MacroStepEvent,
  type MacroStepKind,
  type MacroSummary,
  type RecordRunState,
  type RecordResult,
  type RecordState,
  type RecordStepEvent,
  type SaveMacroRequest,
} from './macro/MacroService';
export {
  TranscriptionService,
  DEFAULT_TRANSCRIPTION_CONFIG,
  type TranscriptionConfig,
  type TranscriptionTask,
  type TranscriptionNote,
  type TranscriptionAnalysis,
  type TranscriptionTaskDetail,
  type AudioProbeResult,
  type TranscriptionCapability,
  type MidiProjectImportResult,
  type TranscriptionProgress,
  type TranscriptionResult,
  type TranscriptionError,
} from './transcription/TranscriptionService';

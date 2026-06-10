/** 格式化毫秒为 m:ss。 */
export const formatDuration = (durationMs: number): string => {
  const totalSeconds = Math.max(0, Math.round(durationMs / 1000));
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  return `${minutes}:${seconds.toString().padStart(2, '0')}`;
};

/** 将 0..1 的比率格式化为百分比字符串。 */
export const formatPercent = (ratio: number): string =>
  `${Math.round(Math.max(0, Math.min(1, ratio)) * 100)}%`;

/** 将数值格式化为带正号的字符串，非正数直接 String。 */
export const formatSigned = (value: number): string =>
  value > 0 ? `+${value}` : String(value);

/** MIDI note 编号 0..127 转换为音名 + 八度（如 "C3", "G#5"）。 */
const NOTE_NAMES = ['C', 'C#', 'D', 'D#', 'E', 'F', 'F#', 'G', 'G#', 'A', 'A#', 'B'];

export const midiNoteToName = (note: number): string => {
  if (!Number.isInteger(note) || note < 0 || note > 127) return '?';
  const name = NOTE_NAMES[note % 12];
  const octave = Math.floor(note / 12) - 1;
  return `${name}${octave}`;
};

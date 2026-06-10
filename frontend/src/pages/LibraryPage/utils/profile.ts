import type { MidiProfile } from '@/services';
import type { LibraryProfileForm } from '../useLibraryPage';

/** 去重合并 defaultProfile + profiles 列表。 */
export const uniqueProfiles = (defaultProfile: MidiProfile, profiles: MidiProfile[]): MidiProfile[] => {
  const seen = new Set<number>();
  return [defaultProfile, ...profiles].filter((profile) => {
    if (seen.has(profile.id)) return false;
    seen.add(profile.id);
    return true;
  });
};

/** 将 MidiProfile 转换为 LibraryProfileForm。 */
export const profileToForm = (profile: MidiProfile, projectId?: number): LibraryProfileForm => ({
  id: profile.id,
  projectId: profile.projectId ?? projectId,
  name: profile.name,
  baseNote: profile.baseNote,
  transpose: profile.transpose,
  octaveShift: profile.octaveShift,
  speed: profile.speed,
  outOfRangePolicy: profile.outOfRangePolicy,
  minPressMs: profile.minPressMs,
  releaseGapMs: profile.releaseGapMs,
  keymapProfileId: profile.keymapProfileId,
  enabledTracks: profile.enabledTracks === null ? null : [...profile.enabledTracks],
});

const isSameTracks = (a: number[] | null, b: number[] | null): boolean => {
  if (a === null || b === null) return a === b;
  if (a.length !== b.length) return false;
  return a.every((value, index) => value === b[index]);
};

/** 比较两个 LibraryProfileForm 是否内容相同。 */
export const isSameForm = (a: LibraryProfileForm, b: LibraryProfileForm): boolean => (
  a.id === b.id
  && a.name === b.name
  && a.baseNote === b.baseNote
  && a.transpose === b.transpose
  && a.octaveShift === b.octaveShift
  && a.speed === b.speed
  && a.outOfRangePolicy === b.outOfRangePolicy
  && a.minPressMs === b.minPressMs
  && a.releaseGapMs === b.releaseGapMs
  && a.keymapProfileId === b.keymapProfileId
  && isSameTracks(a.enabledTracks, b.enabledTracks)
);

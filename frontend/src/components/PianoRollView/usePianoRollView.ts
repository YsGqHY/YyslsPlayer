import { useEffect, useMemo, useState } from 'react';
import { PreviewEngineService, type KeyFrame, type PlayPlan, type PreviewEngineSnapshot } from '@/services';

export interface PianoRollNoteBlock {
  id: string;
  lane: number;
  sourceNote: number;
  normalizedNote: number;
  velocity: number;
  startMs: number;
  endMs: number;
  durationMs: number;
  leftPercent: number;
  widthPercent: number;
  laneTopPercent: number;
}

export interface PianoRollTick {
  id: string;
  timeMs: number;
  leftPercent: number;
  label: string;
}

export interface PianoRollSummary {
  durationMs: number;
  totalNotes: number;
  renderedNotes: number;
  hiddenNotes: number;
  activeLaneCount: number;
  positionMs: number;
  positionPercent: number;
}

export interface PianoRollViewModel {
  notes: PianoRollNoteBlock[];
  ticks: PianoRollTick[];
  lanes: number[];
  activeLanes: Set<number>;
  summary: PianoRollSummary;
}

export interface UsePianoRollViewOptions {
  maxNotes?: number;
  tickCount?: number;
}

const LANE_COUNT = 36;
const DEFAULT_MAX_NOTES = 900;
const DEFAULT_TICK_COUNT = 5;
const MIN_BLOCK_MS = 12;

export const usePianoRollView = (plan: PlayPlan | null, options: UsePianoRollViewOptions = {}): PianoRollViewModel => {
  const maxNotes = options.maxNotes ?? DEFAULT_MAX_NOTES;
  const tickCount = options.tickCount ?? DEFAULT_TICK_COUNT;
  const [snapshot, setSnapshot] = useState<PreviewEngineSnapshot>(() => PreviewEngineService.snapshot());

  useEffect(() => PreviewEngineService.subscribe(setSnapshot), []);

  const lanes = useMemo(() => Array.from({ length: LANE_COUNT }, (_, index) => LANE_COUNT - 1 - index), []);

  const pairedNotes = useMemo(() => pairFrames(plan), [plan]);
  const notes = useMemo(() => pairedNotes.slice(0, maxNotes), [maxNotes, pairedNotes]);
  const ticks = useMemo(() => buildTicks(plan?.durationMs ?? 0, tickCount), [plan?.durationMs, tickCount]);

  const durationMs = Math.max(0, plan?.durationMs ?? 0);
  const positionMs = durationMs > 0 ? clamp(snapshot.positionMs, 0, durationMs) : 0;

  return {
    notes,
    ticks,
    lanes,
    activeLanes: new Set(snapshot.activeLanes),
    summary: {
      durationMs,
      totalNotes: pairedNotes.length,
      renderedNotes: notes.length,
      hiddenNotes: Math.max(0, pairedNotes.length - notes.length),
      activeLaneCount: snapshot.activeLanes.length,
      positionMs,
      positionPercent: durationMs > 0 ? (positionMs / durationMs) * 100 : 0,
    },
  };
};

const pairFrames = (plan: PlayPlan | null): PianoRollNoteBlock[] => {
  if (!plan || plan.durationMs <= 0 || plan.frames.length === 0) return [];

  const durationMs = Math.max(1, plan.durationMs);
  const pending = new Map<number, KeyFrame[]>();
  const notes: PianoRollNoteBlock[] = [];

  for (const frame of [...plan.frames].sort((a, b) => a.timeMs - b.timeMs)) {
    if (frame.lane < 0 || frame.lane >= LANE_COUNT) continue;

    if (frame.action === 'press') {
      const queue = pending.get(frame.lane) ?? [];
      queue.push(frame);
      pending.set(frame.lane, queue);
      continue;
    }

    const queue = pending.get(frame.lane);
    const start = queue?.shift();
    if (!start) continue;
    if (queue && queue.length === 0) pending.delete(frame.lane);
    notes.push(toNoteBlock(start, frame.timeMs, durationMs, notes.length));
  }

  for (const queue of pending.values()) {
    for (const start of queue) {
      notes.push(toNoteBlock(start, durationMs, durationMs, notes.length));
    }
  }

  return notes.sort((a, b) => a.startMs - b.startMs || a.lane - b.lane);
};

const toNoteBlock = (start: KeyFrame, releaseMs: number, planDurationMs: number, index: number): PianoRollNoteBlock => {
  const startMs = clamp(start.timeMs, 0, planDurationMs);
  const endMs = clamp(Math.max(releaseMs, startMs + MIN_BLOCK_MS), 0, planDurationMs);
  const durationMs = Math.max(MIN_BLOCK_MS, endMs - startMs);
  const leftPercent = (startMs / planDurationMs) * 100;
  const rawWidthPercent = (durationMs / planDurationMs) * 100;
  const widthPercent = Math.min(Math.max(0.2, 100 - leftPercent), Math.max(0.2, rawWidthPercent));
  return {
    id: `${start.lane}-${startMs}-${index}`,
    lane: start.lane,
    sourceNote: start.sourceNote,
    normalizedNote: start.normalizedNote,
    velocity: start.velocity,
    startMs,
    endMs,
    durationMs,
    leftPercent,
    widthPercent,
    laneTopPercent: ((LANE_COUNT - 1 - start.lane) / LANE_COUNT) * 100,
  };
};

const buildTicks = (durationMs: number, tickCount: number): PianoRollTick[] => {
  if (durationMs <= 0) return [];
  const count = Math.max(2, tickCount);
  return Array.from({ length: count }, (_, index) => {
    const ratio = index / (count - 1);
    const timeMs = Math.round(durationMs * ratio);
    return {
      id: `${index}-${timeMs}`,
      timeMs,
      leftPercent: ratio * 100,
      label: formatTime(timeMs),
    };
  });
};

const formatTime = (value: number): string => {
  const totalSeconds = Math.max(0, Math.floor(value / 1000));
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  return `${minutes}:${seconds.toString().padStart(2, '0')}`;
};

const clamp = (value: number, min: number, max: number): number => Math.min(max, Math.max(min, value));

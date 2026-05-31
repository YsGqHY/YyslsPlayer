import type { KeyFrame, PlayPlan } from '@/services/midi/MidiService';

export type PreviewEngineState = 'idle' | 'playing' | 'paused' | 'stopped';

export interface PreviewEngineSnapshot {
  state: PreviewEngineState;
  positionMs: number;
  durationMs: number;
  activeLanes: number[];
}

export interface PreviewEngineOptions {
  progressHz?: number;
  masterGain?: number;
  waveform?: OscillatorType;
}

type Listener = (snapshot: PreviewEngineSnapshot) => void;

interface ActiveVoice {
  lane: number;
  oscillator: OscillatorNode;
  gain: GainNode;
}

const DEFAULT_PROGRESS_HZ = 10;
const DEFAULT_GAIN = 0.08;
const ATTACK_SECONDS = 0.006;
const RELEASE_SECONDS = 0.035;
const DEFAULT_MIDI_NOTE = 60; // C4, used as the preview pitch anchor.
export class PreviewEngine {
  private audioContext: AudioContext | null = null;
  private plan: PlayPlan | null = null;
  private state: PreviewEngineState = 'idle';
  private listeners = new Set<Listener>();
  private active = new Map<number, ActiveVoice>();
  private scheduled: number[] = [];
  private progressTimer: number | null = null;
  private startedAtSeconds = 0;
  private pausedAtMs = 0;
  private durationMs = 0;
  private progressHz: number;
  private masterGain: number;
  private waveform: OscillatorType;

  constructor(options: PreviewEngineOptions = {}) {
    this.progressHz = clamp(options.progressHz ?? DEFAULT_PROGRESS_HZ, 1, 30);
    this.masterGain = clamp(options.masterGain ?? DEFAULT_GAIN, 0.01, 0.5);
    this.waveform = options.waveform ?? 'sine';
  }

  subscribe(listener: Listener): () => void {
    this.listeners.add(listener);
    listener(this.snapshot());
    return () => {
      this.listeners.delete(listener);
    };
  }

  configure(options: PreviewEngineOptions): void {
    if (options.progressHz !== undefined) {
      this.progressHz = clamp(options.progressHz, 1, 30);
      if (this.state === 'playing') {
        this.startProgressTimer();
      }
    }
    if (options.masterGain !== undefined) {
      this.masterGain = clamp(options.masterGain, 0, 0.5);
    }
    if (options.waveform !== undefined) {
      this.waveform = options.waveform;
    }
  }

  load(plan: PlayPlan): void {
    this.stopScheduled(true);
    this.plan = plan;
    this.durationMs = Math.max(0, plan.durationMs);
    this.pausedAtMs = 0;
    this.startedAtSeconds = 0;
    this.state = 'idle';
    this.emit();
  }

  async play(plan: PlayPlan, startMs = 0): Promise<void> {
    await this.ensureAudioContext();
    if (!this.audioContext) return;
    this.load(plan);
    this.pausedAtMs = clamp(startMs, 0, this.durationMs);
    this.startedAtSeconds = this.audioContext.currentTime - this.pausedAtMs / 1000;
    this.state = 'playing';
    this.scheduleFrom(this.pausedAtMs);
    this.startProgressTimer();
    this.emit();
  }

  pause(): void {
    if (this.state !== 'playing') return;
    this.pausedAtMs = this.positionMs();
    this.stopScheduled(true);
    this.state = 'paused';
    this.emit();
  }

  async resume(): Promise<void> {
    if (!this.plan || this.state !== 'paused') return;
    await this.play(this.plan, this.pausedAtMs);
  }

  stop(): void {
    this.pausedAtMs = 0;
    this.stopScheduled(true);
    this.state = 'stopped';
    this.emit();
  }

  // seek 跳转到指定毫秒：
  //   - playing：停掉当前调度并从新位置重新排程，听感上立即跳转
  //   - paused / stopped / idle：只更新位置，等下次 play/resume 时从该处开始
  // 没有加载 plan 时忽略。
  seek(targetMs: number): void {
    if (!this.plan) return;
    const clamped = clamp(Math.round(targetMs), 0, this.durationMs);
    if (this.state === 'playing') {
      if (!this.audioContext) return;
      this.stopScheduled(true);
      this.pausedAtMs = clamped;
      this.startedAtSeconds = this.audioContext.currentTime - clamped / 1000;
      this.scheduleFrom(clamped);
      this.startProgressTimer();
      this.emit();
      return;
    }
    // 已暂停 / 已停止 / 空闲：仅记录目标位置，下次播放从此处起。
    this.pausedAtMs = clamped;
    if (this.state === 'idle' || this.state === 'stopped') {
      this.state = 'paused';
    }
    this.emit();
  }

  dispose(): void {
    this.stopScheduled(true);
    this.listeners.clear();
    void this.audioContext?.close();
    this.audioContext = null;
    this.plan = null;
    this.state = 'idle';
  }

  snapshot(): PreviewEngineSnapshot {
    return {
      state: this.state,
      positionMs: this.state === 'playing' ? this.positionMs() : this.pausedAtMs,
      durationMs: this.durationMs,
      activeLanes: Array.from(this.active.keys()).sort((a, b) => a - b),
    };
  }

  private async ensureAudioContext(): Promise<void> {
    if (this.audioContext) {
      if (this.audioContext.state === 'suspended') {
        await this.audioContext.resume();
      }
      return;
    }
    const Ctor = window.AudioContext ?? window.webkitAudioContext;
    if (!Ctor) {
      throw new Error('AudioContext is not available in this browser');
    }
    this.audioContext = new Ctor();
  }

  private scheduleFrom(startMs: number): void {
    if (!this.audioContext || !this.plan) return;
    const frames = this.plan.frames.filter((frame) => frame.timeMs >= startMs);
    for (const frame of frames) {
      const delayMs = Math.max(0, frame.timeMs - startMs);
      const timer = window.setTimeout(() => this.handleFrame(frame), delayMs);
      this.scheduled.push(timer);
    }
    const remainingMs = Math.max(0, this.durationMs - startMs);
    const doneTimer = window.setTimeout(() => {
      if (this.state === 'playing') {
        this.pausedAtMs = this.durationMs;
        this.stopScheduled(true);
        this.state = 'stopped';
        this.emit();
      }
    }, remainingMs + 60);
    this.scheduled.push(doneTimer);
  }

  private handleFrame(frame: KeyFrame): void {
    if (!this.audioContext || this.state !== 'playing') return;
    if (frame.action === 'press') {
      this.startVoice(frame);
      return;
    }
    this.stopVoice(frame.lane);
  }

  private startVoice(frame: KeyFrame): void {
    if (!this.audioContext) return;
    this.stopVoice(frame.lane);
    const oscillator = this.audioContext.createOscillator();
    const gain = this.audioContext.createGain();
    oscillator.type = this.waveform;
    oscillator.frequency.value = noteToFrequency(frame.normalizedNote);
    const velocityGain = clamp(frame.velocity / 127, 0.15, 1) * this.masterGain;
    const now = this.audioContext.currentTime;
    gain.gain.setValueAtTime(0, now);
    gain.gain.linearRampToValueAtTime(velocityGain, now + ATTACK_SECONDS);
    oscillator.connect(gain);
    gain.connect(this.audioContext.destination);
    oscillator.start(now);
    this.active.set(frame.lane, { lane: frame.lane, oscillator, gain });
    this.emit();
  }

  private stopVoice(lane: number): void {
    const voice = this.active.get(lane);
    if (!voice || !this.audioContext) return;
    const now = this.audioContext.currentTime;
    voice.gain.gain.cancelScheduledValues(now);
    voice.gain.gain.setValueAtTime(voice.gain.gain.value, now);
    voice.gain.gain.linearRampToValueAtTime(0, now + RELEASE_SECONDS);
    voice.oscillator.stop(now + RELEASE_SECONDS + 0.005);
    this.active.delete(lane);
    this.emit();
  }

  private stopScheduled(stopVoices: boolean): void {
    for (const id of this.scheduled) {
      window.clearTimeout(id);
    }
    this.scheduled = [];
    this.stopProgressTimer();
    if (stopVoices) {
      for (const lane of Array.from(this.active.keys())) {
        this.stopVoice(lane);
      }
      this.active.clear();
    }
  }

  private positionMs(): number {
    if (!this.audioContext || this.state !== 'playing') return this.pausedAtMs;
    const elapsedMs = (this.audioContext.currentTime - this.startedAtSeconds) * 1000;
    return clamp(Math.round(elapsedMs), 0, this.durationMs);
  }

  private startProgressTimer(): void {
    this.stopProgressTimer();
    this.progressTimer = window.setInterval(() => this.emit(), Math.round(1000 / this.progressHz));
  }

  private stopProgressTimer(): void {
    if (this.progressTimer !== null) {
      window.clearInterval(this.progressTimer);
      this.progressTimer = null;
    }
  }

  private emit(): void {
    const snapshot = this.snapshot();
    for (const listener of this.listeners) {
      listener(snapshot);
    }
  }
}

const noteToFrequency = (note: number): number => 440 * (2 ** ((note - DEFAULT_MIDI_NOTE) / 12));
const clamp = (value: number, min: number, max: number): number => Math.min(max, Math.max(min, value));

export const PreviewEngineService = new PreviewEngine();

declare global {
  interface Window {
    webkitAudioContext?: typeof AudioContext;
  }
}

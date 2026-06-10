//go:build completion

// Package postprocess 提供转录后处理管道：置信度过滤、量化、旋律抽取、36 键分析。
package postprocess

import (
	"fmt"
	"sort"

	"YyslsPlayer/internal/services/transcription/engine"
	"YyslsPlayer/internal/services/transcription/shared"
)

// Processor 后处理器。
type Processor struct {
	config shared.Config
}

// New 创建后处理器。
func New(cfg shared.Config) *Processor {
	if cfg.MinConfidence <= 0 {
		cfg.MinConfidence = 0.55
	}
	if cfg.MinDurationMs <= 0 {
		cfg.MinDurationMs = 60
	}
	if cfg.MergeGapMs <= 0 {
		cfg.MergeGapMs = 40
	}
	if cfg.MaxPolyphony <= 0 {
		cfg.MaxPolyphony = 2
	}
	if cfg.TargetLaneCount <= 0 {
		cfg.TargetLaneCount = 36
	}
	return &Processor{config: cfg}
}

// ProcessResult 后处理结果。
type ProcessResult struct {
	Notes              []engine.RawNote     `json:"notes"`
	QualityReport      shared.QualityReport `json:"qualityReport"`
	MelodySummary      shared.MelodySummary `json:"melodySummary"`
	DroppedCount       int                  `json:"droppedCount"`
	LowConfidenceCount int                  `json:"lowConfidenceCount"`
	OutOfRangeCount    int                  `json:"outOfRangeCount"`
}

// Process 执行完整后处理管道。
func (p *Processor) Process(raw []engine.RawNote, durationMs int64) (*ProcessResult, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("postprocess.empty_result: no notes to process")
	}

	notes := make([]engine.RawNote, len(raw))
	copy(notes, raw)

	result := &ProcessResult{}
	totalRaw := len(notes)

	afterCon, lowConf := p.filterConfidence(notes)
	result.LowConfidenceCount = lowConf
	notes = afterCon

	notes, shortNotes := p.filterShortNotes(notes)
	result.DroppedCount = totalRaw - len(notes)
	_ = shortNotes

	notes = p.mergeAdjacentSame(notes)

	sort.Slice(notes, func(i, j int) bool {
		if notes[i].StartMs == notes[j].StartMs {
			return notes[i].Confidence > notes[j].Confidence
		}
		return notes[i].StartMs < notes[j].StartMs
	})

	if p.config.Quantize == "light" || p.config.Quantize == "strong" {
		notes = p.quantizeLight(notes)
	}

	if p.config.Mode == "melody" || p.config.MaxPolyphony > 0 {
		notes = p.limitPolyphony(notes)
	}

	result.Notes = notes
	result.QualityReport = p.buildQualityReport(notes, result, durationMs)
	result.MelodySummary = p.buildMelodySummary(notes, durationMs)
	result.OutOfRangeCount = p.countOutOfRange(notes)

	return result, nil
}

func (p *Processor) filterConfidence(notes []engine.RawNote) ([]engine.RawNote, int) {
	lowCount := 0
	filtered := make([]engine.RawNote, 0, len(notes))
	for _, n := range notes {
		if n.Confidence >= p.config.MinConfidence {
			filtered = append(filtered, n)
		} else {
			lowCount++
		}
	}
	return filtered, lowCount
}

func (p *Processor) filterShortNotes(notes []engine.RawNote) ([]engine.RawNote, int) {
	dropped := 0
	filtered := make([]engine.RawNote, 0, len(notes))
	for _, n := range notes {
		if n.DurationMs >= int64(p.config.MinDurationMs) {
			filtered = append(filtered, n)
		} else {
			dropped++
		}
	}
	return filtered, dropped
}

func (p *Processor) mergeAdjacentSame(notes []engine.RawNote) []engine.RawNote {
	if len(notes) < 2 {
		return notes
	}

	sort.Slice(notes, func(i, j int) bool {
		if notes[i].Note == notes[j].Note {
			return notes[i].StartMs < notes[j].StartMs
		}
		return notes[i].Note < notes[j].Note || (notes[i].Note == notes[j].Note && notes[i].StartMs < notes[j].StartMs)
	})

	merged := make([]engine.RawNote, 0, len(notes))
	for i := 0; i < len(notes); {
		cur := notes[i]
		j := i + 1
		for j < len(notes) &&
			notes[j].Note == cur.Note &&
			(cur.StartMs+cur.DurationMs+int64(p.config.MergeGapMs)) >= notes[j].StartMs {
			end := notes[j].StartMs + notes[j].DurationMs
			if end > cur.StartMs+cur.DurationMs {
				cur.DurationMs = end - cur.StartMs
			}
			if notes[j].Confidence > cur.Confidence {
				cur.Confidence = notes[j].Confidence
			}
			j++
		}
		merged = append(merged, cur)
		i = j
	}
	return merged
}

func (p *Processor) quantizeLight(notes []engine.RawNote) []engine.RawNote {
	gridMs := int64(125)
	result := make([]engine.RawNote, len(notes))
	threshold := int64(40)
	if p.config.Quantize == "strong" {
		threshold = int64(80)
	}

	for i, n := range notes {
		result[i] = n
		nearest := (n.StartMs / gridMs) * gridMs
		dist := n.StartMs - nearest
		if dist > gridMs/2 {
			nearest += gridMs
			dist = nearest - n.StartMs
		}
		if dist < 0 {
			dist = -dist
		}
		if dist < threshold {
			result[i].StartMs = nearest
		}
	}
	return result
}

func (p *Processor) limitPolyphony(notes []engine.RawNote) []engine.RawNote {
	if p.config.MaxPolyphony <= 0 || len(notes) == 0 {
		return notes
	}

	type event struct {
		timeMs int64
		note   *engine.RawNote
		isEnd  bool
		score  float64
	}

	events := make([]event, 0, len(notes)*2)
	for i := range notes {
		score := notes[i].Confidence
		events = append(events,
			event{timeMs: notes[i].StartMs, note: &notes[i], isEnd: false, score: score},
			event{timeMs: notes[i].StartMs + notes[i].DurationMs, note: &notes[i], isEnd: true, score: score},
		)
	}

	sort.Slice(events, func(i, j int) bool {
		if events[i].timeMs == events[j].timeMs {
			return !events[i].isEnd && events[j].isEnd
		}
		return events[i].timeMs < events[j].timeMs
	})

	active := make(map[int]*engine.RawNote)
	kept := make(map[int]bool)

	for _, ev := range events {
		if ev.isEnd {
			for k, n := range active {
				if n == ev.note {
					delete(active, k)
					break
				}
			}
		} else {
			noteKey := ev.note.Note
			if len(active) >= p.config.MaxPolyphony {
				worstKey := -1
				worstScore := 1.1
				for k, n := range active {
					s := n.Confidence
					if s < worstScore {
						worstScore = s
						worstKey = k
					}
				}
				if ev.note.Confidence > worstScore {
					if worstKey >= 0 {
						delete(active, worstKey)
					}
					active[noteKey] = ev.note
					kept[getNoteIndex(notes, ev.note)] = true
				}
			} else {
				active[noteKey] = ev.note
				kept[getNoteIndex(notes, ev.note)] = true
			}
		}
	}

	if len(kept) == 0 {
		return notes
	}

	result := make([]engine.RawNote, 0, len(kept))
	for i := range notes {
		if kept[i] {
			result = append(result, notes[i])
		}
	}
	return result
}

func getNoteIndex(notes []engine.RawNote, target *engine.RawNote) int {
	for i := range notes {
		if &notes[i] == target {
			return i
		}
	}
	return -1
}

// detectDenseSegments 滑动窗口检测超过 maxPolyphony 的密集片段。
func detectDenseSegments(notes []engine.RawNote, maxPolyphony int) []shared.DenseSegment {
	if maxPolyphony <= 0 || len(notes) < 2 {
		return nil
	}

	type timeEvent struct {
		timeMs int64
		delta  int // +1 for note on, -1 for note off
	}
	events := make([]timeEvent, 0, len(notes)*2)
	for _, n := range notes {
		events = append(events, timeEvent{n.StartMs, 1}, timeEvent{n.StartMs + n.DurationMs, -1})
	}
	sort.Slice(events, func(i, j int) bool { return events[i].timeMs < events[j].timeMs })

	var segments []shared.DenseSegment
	activeNotes := 0
	segmentStart := int64(-1)
	segmentMaxPoly := 0
	segmentNoteCount := 0

	for _, ev := range events {
		prevActive := activeNotes
		activeNotes += ev.delta

		if activeNotes > maxPolyphony {
			if segmentStart < 0 {
				segmentStart = ev.timeMs
			}
			if activeNotes > segmentMaxPoly {
				segmentMaxPoly = activeNotes
			}
			segmentNoteCount++
		} else if segmentStart >= 0 && activeNotes <= maxPolyphony {
			segment := shared.DenseSegment{
				StartMs:     segmentStart,
				EndMs:       ev.timeMs,
				PolyphonyAt: segmentMaxPoly,
				NoteCount:   segmentNoteCount,
			}
			segments = append(segments, segment)
			segmentStart = -1
			segmentMaxPoly = 0
			segmentNoteCount = 0
		}
		_ = prevActive
	}

	// 限制最多 20 个密集片段
	if len(segments) > 20 {
		segments = segments[:20]
	}
	return segments
}

func (p *Processor) countOutOfRange(notes []engine.RawNote) int {
	baseNote := p.config.TargetBaseNote
	maxNote := baseNote + p.config.TargetLaneCount - 1
	count := 0
	for _, n := range notes {
		if n.Note < baseNote || n.Note > maxNote {
			count++
		}
	}
	return count
}

func (p *Processor) buildQualityReport(notes []engine.RawNote, result *ProcessResult, durationMs int64) shared.QualityReport {
	if len(notes) == 0 {
		return shared.QualityReport{Warnings: []string{"postprocess.empty_result"}}
	}

	baseNote := p.config.TargetBaseNote
	maxNote := baseNote + p.config.TargetLaneCount - 1
	total := len(notes)
	outRange := p.countOutOfRange(notes)

	minNote, maxN := notes[0].Note, notes[0].Note
	totalConf := 0.0
	for _, n := range notes {
		if n.Note < minNote {
			minNote = n.Note
		}
		if n.Note > maxN {
			maxN = n.Note
		}
		totalConf += n.Confidence
	}
	avgConf := totalConf / float64(total)

	playScore := 100.0
	if outRange > 0 {
		playScore -= float64(outRange) / float64(total) * 40
	}
	if avgConf < 0.7 {
		playScore -= (0.7 - avgConf) * 50
	}
	if result.LowConfidenceCount > 0 {
		playScore -= float64(result.LowConfidenceCount) / float64(result.DroppedCount+total) * 20
	}
	if playScore < 0 {
		playScore = 0
	}

	suggestTranspose := 0
	suggestOctave := 0
	if outRange > 0 {
		midNote := (minNote + maxN) / 2
		targetMid := (baseNote + maxNote) / 2
		suggestTranspose = targetMid - midNote
		suggestOctave = suggestTranspose / 12
	}

	warnings := []string{}
	if avgConf < 0.6 {
		warnings = append(warnings, "转录置信度较低，旋律识别可能不稳定")
	}
	if outRange > total/4 {
		warnings = append(warnings, fmt.Sprintf("%d/%d 音符超出 36 键范围，建议移调", outRange, total))
	}
	if result.DroppedCount > total/3 {
		warnings = append(warnings, "大量候选音符被过滤，可尝试降低最小置信度阈值")
	}

	// 使用分析阶段产出的真实 BPM/Key（若未提供则用默认值）
	bpm := p.config.EstimatedBPM
	bpmConf := p.config.BPMConfidence
	if bpm <= 0 {
		bpm = 120.0
	}
	keyEst := p.config.KeyEstimate
	if keyEst.Method == "" {
		keyEst = shared.KeyEstimate{Method: "pitch_class", Confidence: 0.3}
	}

	audioScore := p.config.AudioQualityScore
	if audioScore <= 0 {
		audioScore = 80
	}

	// 过密片段检测
	denseSegments := detectDenseSegments(notes, p.config.MaxPolyphony)

	return shared.QualityReport{
		OverallScore:            playScore,
		TranscriptionConfidence: avgConf,
		PlayabilityScore:        float64(total-outRange) / float64(total) * 100,
		AudioQualityScore:       audioScore,
		EstimatedBPM:            bpm,
		BPMConfidence:           bpmConf,
		KeyEstimate:             keyEst,
		NoteCount:               total,
		DroppedCandidateCount:   result.DroppedCount,
		LowConfidenceCount:      result.LowConfidenceCount,
		OutOfRangeCount:         outRange,
		ShortNoteCount:          result.DroppedCount,
		MaxPolyphony:            p.config.MaxPolyphony,
		DenseSegments:           denseSegments,
		SuggestedTranspose:      suggestTranspose,
		SuggestedOctaveShift:    suggestOctave,
		Warnings:                warnings,
	}
}

func (p *Processor) buildMelodySummary(notes []engine.RawNote, durationMs int64) shared.MelodySummary {
	total := len(notes)
	if total == 0 {
		return shared.MelodySummary{}
	}

	minNote, maxN := notes[0].Note, notes[0].Note
	totalVel := 0.0
	for _, n := range notes {
		if n.Note < minNote {
			minNote = n.Note
		}
		if n.Note > maxN {
			maxN = n.Note
		}
		totalVel += n.Velocity
	}

	dominant := "mid"
	if maxN <= 60 {
		dominant = "low"
	} else if minNote >= 72 {
		dominant = "high"
	}

	return shared.MelodySummary{
		NoteCount:        total,
		DurationMs:       durationMs,
		MinNote:          minNote,
		MaxNote:          maxN,
		AverageVelocity:  totalVel / float64(total),
		EstimatedBPM:     120.0,
		DominantRegister: dominant,
		PolyphonyRate:    0,
		PlayabilityScore: 80,
	}
}

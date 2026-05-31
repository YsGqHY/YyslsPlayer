package midi

import (
	"errors"
	"sort"
)

const defaultTempoMicrosecondsPerQuarter int64 = 500000

type TempoChange struct {
	Tick                   int64 `json:"tick"`
	MicrosecondsPerQuarter int64 `json:"microsecondsPerQuarter"`
}

type TempoSegment struct {
	Tick                   int64 `json:"tick"`
	TimeMicroseconds       int64 `json:"timeMicroseconds"`
	MicrosecondsPerQuarter int64 `json:"microsecondsPerQuarter"`
}

type TempoMap struct {
	ppq      int64
	segments []TempoSegment
}

func NewTempoMap(ppq int, changes []TempoChange) (TempoMap, error) {
	if ppq <= 0 {
		return TempoMap{}, errors.New("ppq must be positive")
	}

	normalized := normalizeTempoChanges(changes)
	segments := make([]TempoSegment, 0, len(normalized)+1)
	if len(normalized) == 0 || normalized[0].Tick > 0 {
		segments = append(segments, TempoSegment{
			Tick:                   0,
			MicrosecondsPerQuarter: defaultTempoMicrosecondsPerQuarter,
		})
	}
	segments = append(segments, normalized...)

	ppq64 := int64(ppq)
	for i := 1; i < len(segments); i++ {
		prev := segments[i-1]
		deltaTicks := segments[i].Tick - prev.Tick
		segments[i].TimeMicroseconds = prev.TimeMicroseconds + ticksToMicroseconds(deltaTicks, ppq64, prev.MicrosecondsPerQuarter)
	}

	return TempoMap{ppq: ppq64, segments: segments}, nil
}

func (m TempoMap) TickToMilliseconds(tick int64) int64 {
	return roundMicrosecondsToMilliseconds(m.TickToMicroseconds(tick))
}

func (m TempoMap) TickToMicroseconds(tick int64) int64 {
	if tick <= 0 || len(m.segments) == 0 {
		return 0
	}
	idx := sort.Search(len(m.segments), func(i int) bool {
		return m.segments[i].Tick > tick
	}) - 1
	if idx < 0 {
		idx = 0
	}
	seg := m.segments[idx]
	return seg.TimeMicroseconds + ticksToMicroseconds(tick-seg.Tick, m.ppq, seg.MicrosecondsPerQuarter)
}

func (m TempoMap) Segments() []TempoSegment {
	out := make([]TempoSegment, len(m.segments))
	copy(out, m.segments)
	return out
}

func normalizeTempoChanges(changes []TempoChange) []TempoSegment {
	valid := make([]TempoChange, 0, len(changes))
	for _, change := range changes {
		if change.Tick < 0 || change.MicrosecondsPerQuarter <= 0 {
			continue
		}
		valid = append(valid, change)
	}
	if len(valid) == 0 {
		return nil
	}

	sort.SliceStable(valid, func(i, j int) bool {
		return valid[i].Tick < valid[j].Tick
	})

	out := make([]TempoSegment, 0, len(valid))
	for _, change := range valid {
		seg := TempoSegment{
			Tick:                   change.Tick,
			MicrosecondsPerQuarter: change.MicrosecondsPerQuarter,
		}
		if len(out) > 0 && out[len(out)-1].Tick == change.Tick {
			out[len(out)-1] = seg
			continue
		}
		out = append(out, seg)
	}
	return out
}

func ticksToMicroseconds(ticks, ppq, microsecondsPerQuarter int64) int64 {
	if ticks <= 0 || ppq <= 0 || microsecondsPerQuarter <= 0 {
		return 0
	}
	return ticks * microsecondsPerQuarter / ppq
}

func roundMicrosecondsToMilliseconds(us int64) int64 {
	if us <= 0 {
		return 0
	}
	return (us + 500) / 1000
}

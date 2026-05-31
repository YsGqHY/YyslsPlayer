package midi

import (
	"bytes"
	"errors"
	"fmt"
	"sort"

	"YyslsPlayer/internal/storage"

	"gitlab.com/gomidi/midi/v2/smf"
)

var ErrMidiEmpty = errors.New("MIDI_EMPTY")

type normalizedScore struct {
	PPQ          int
	TrackCount   int
	ChannelCount int
	DurationMs   int64
	Events       []storage.MidiEvent
}

type activeNoteKey struct {
	track   int
	channel int
	note    int
}

type activeNote struct {
	track     int
	channel   int
	note      int
	velocity  int
	startTick int64
	startUS   int64
}

func parseNormalizedScore(data []byte) (normalizedScore, error) {
	smfData, err := smf.ReadFrom(bytes.NewReader(data))
	if err != nil {
		return normalizedScore{}, fmt.Errorf("%w: %v", ErrMidiUnsupportedFormat, err)
	}
	metricTicks, ok := smfData.TimeFormat.(smf.MetricTicks)
	if !ok {
		return normalizedScore{}, fmt.Errorf("%w: unsupported SMPTE time format", ErrMidiUnsupportedFormat)
	}
	ppq := int(metricTicks.Resolution())
	if ppq <= 0 {
		return normalizedScore{}, fmt.Errorf("%w: invalid PPQ", ErrMidiUnsupportedFormat)
	}

	tempoChanges := collectTempoChanges(smfData.Tracks)
	tempoMap, err := NewTempoMap(ppq, tempoChanges)
	if err != nil {
		return normalizedScore{}, err
	}
	events, channelSet, maxUS := normalizeTracks(smfData.Tracks, tempoMap)
	if len(events) == 0 {
		return normalizedScore{}, ErrMidiEmpty
	}

	return normalizedScore{
		PPQ:          ppq,
		TrackCount:   len(smfData.Tracks),
		ChannelCount: len(channelSet),
		DurationMs:   roundMicrosecondsToMilliseconds(maxUS),
		Events:       events,
	}, nil
}

func bpmToMicrosecondsPerQuarter(bpm float64) int64 {
	return int64((60000000.0 / bpm) + 0.5)
}

func collectTempoChanges(tracks []smf.Track) []TempoChange {
	changes := make([]TempoChange, 0)
	for _, track := range tracks {
		var absTick int64
		for _, ev := range track {
			absTick += int64(ev.Delta)
			var bpm float64
			if ev.Message.GetMetaTempo(&bpm) && bpm > 0 {
				changes = append(changes, TempoChange{
					Tick:                   absTick,
					MicrosecondsPerQuarter: bpmToMicrosecondsPerQuarter(bpm),
				})
			}
		}
	}
	return changes
}

func normalizeTracks(tracks []smf.Track, tempoMap TempoMap) ([]storage.MidiEvent, map[int]bool, int64) {
	events := make([]storage.MidiEvent, 0)
	channelSet := make(map[int]bool)
	var maxUS int64

	for trackIdx, track := range tracks {
		active := make(map[activeNoteKey][]activeNote)
		var absTick int64
		for _, ev := range track {
			absTick += int64(ev.Delta)
			var channel, note, velocity uint8
			if ev.Message.GetNoteStart(&channel, &note, &velocity) {
				channelSet[int(channel)] = true
				key := activeNoteKey{track: trackIdx, channel: int(channel), note: int(note)}
				active[key] = append(active[key], activeNote{
					track:     trackIdx,
					channel:   int(channel),
					note:      int(note),
					velocity:  int(velocity),
					startTick: absTick,
					startUS:   tempoMap.TickToMicroseconds(absTick),
				})
				continue
			}
			if ev.Message.GetNoteEnd(&channel, &note) {
				channelSet[int(channel)] = true
				key := activeNoteKey{track: trackIdx, channel: int(channel), note: int(note)}
				queue := active[key]
				if len(queue) == 0 {
					continue
				}
				started := queue[0]
				if len(queue) == 1 {
					delete(active, key)
				} else {
					active[key] = queue[1:]
				}
				endUS := tempoMap.TickToMicroseconds(absTick)
				durationUS := endUS - started.startUS
				if durationUS < 0 {
					durationUS = 0
				}
				startMs := roundMicrosecondsToMilliseconds(started.startUS)
				durationMs := roundMicrosecondsToMilliseconds(durationUS)
				events = append(events, storage.MidiEvent{
					Track:      started.track,
					Channel:    started.channel,
					Note:       started.note,
					Velocity:   started.velocity,
					StartMs:    startMs,
					DurationMs: durationMs,
				})
				if endUS > maxUS {
					maxUS = endUS
				}
			}
		}
	}

	sort.SliceStable(events, func(i, j int) bool {
		if events[i].StartMs != events[j].StartMs {
			return events[i].StartMs < events[j].StartMs
		}
		if events[i].Track != events[j].Track {
			return events[i].Track < events[j].Track
		}
		if events[i].Channel != events[j].Channel {
			return events[i].Channel < events[j].Channel
		}
		return events[i].Note < events[j].Note
	})
	return events, channelSet, maxUS
}

package midi

import (
	"sort"

	"YyslsPlayer/internal/storage"
)

type NoteRangeDTO struct {
	Min int `json:"min"`
	Max int `json:"max"`
}

type MappedRangeDTO struct {
	MinLane int `json:"minLane"`
	MaxLane int `json:"maxLane"`
}

type QualityReportDTO struct {
	NoteRange            NoteRangeDTO   `json:"noteRange"`
	MappedRange          MappedRangeDTO `json:"mappedRange"`
	TotalNotes           int            `json:"totalNotes"`
	PlayableNotes        int            `json:"playableNotes"`
	OutOfRangeCount      int            `json:"outOfRangeCount"`
	DroppedCount         int            `json:"droppedCount"`
	FoldedCount          int            `json:"foldedCount"`
	ClampedCount         int            `json:"clampedCount"`
	BlackKeyCount        int            `json:"blackKeyCount"`
	PlayableRatio        float64        `json:"playableRatio"`
	ChordDensity         int            `json:"chordDensity"`
	TrackCount           int            `json:"trackCount"`
	ChannelCount         int            `json:"channelCount"`
	SuggestedTranspose   int            `json:"suggestedTranspose"`
	SuggestedOctaveShift int            `json:"suggestedOctaveShift"`
	Warnings             []string       `json:"warnings"`
}

type timePoint struct {
	timeMs int64
	delta  int
}

func buildQualityReport(store *storage.Store, project storage.MidiProject) QualityReportDTO {
	return qualityReportFromEvents(project, store.ListEventsByProject(project.ID))
}

func buildQualityReportWithConfig(store *storage.Store, project storage.MidiProject, cfg MidiConfigDTO) QualityReportDTO {
	return qualityReportFromEventsWithConfig(project, store.ListEventsByProject(project.ID), cfg)
}

func qualityReportFromEvents(project storage.MidiProject, events []storage.MidiEvent) QualityReportDTO {
	report := QualityReportDTO{
		NoteRange:    NoteRangeDTO{Min: -1, Max: -1},
		MappedRange:  MappedRangeDTO{MinLane: -1, MaxLane: -1},
		TrackCount:   project.TrackCount,
		ChannelCount: project.ChannelCount,
		Warnings:     []string{},
	}
	if len(events) == 0 {
		return report
	}

	minNote := events[0].Note
	maxNote := events[0].Note
	tracks := make(map[int]bool)
	channels := make(map[int]bool)
	points := make([]timePoint, 0, len(events)*2)

	for _, event := range events {
		report.TotalNotes++
		if isBlackMidiNote(event.Note) {
			report.BlackKeyCount++
		}
		if event.Note < minNote {
			minNote = event.Note
		}
		if event.Note > maxNote {
			maxNote = event.Note
		}
		tracks[event.Track] = true
		channels[event.Channel] = true
		start := event.StartMs
		end := event.StartMs + event.DurationMs
		if end < start {
			end = start
		}
		points = append(points, timePoint{timeMs: start, delta: 1})
		points = append(points, timePoint{timeMs: end, delta: -1})
	}

	report.NoteRange = NoteRangeDTO{Min: minNote, Max: maxNote}
	report.TrackCount = len(tracks)
	report.ChannelCount = len(channels)
	report.ChordDensity = chordDensity(points)
	// M2 has not applied a 36-lane mapping yet; every parsed note is considered structurally playable here.
	report.PlayableNotes = report.TotalNotes
	if report.TotalNotes > 0 {
		report.PlayableRatio = 1
	}
	return report
}

func qualityReportFromEventsWithConfig(project storage.MidiProject, events []storage.MidiEvent, cfg MidiConfigDTO) QualityReportDTO {
	events = filterEventsByEnabledTracks(events, cfg.EnabledTracks)
	report := qualityReportFromEvents(project, events)
	if len(events) == 0 {
		return report
	}
	results := make([]LaneMappingDTO, 0, len(events))
	for _, event := range events {
		mapped, err := MapNoteWithPolicy(event.Note, cfg)
		if err != nil {
			results = append(results, mapped)
			continue
		}
		results = append(results, mapped)
	}
	applyMappingStatsToReport(&report, MappingStatsFromResults(results))
	report.SuggestedTranspose = suggestTranspose(report.NoteRange, cfg)
	report.SuggestedOctaveShift = suggestOctaveShift(report.SuggestedTranspose)
	report.Warnings = qualityWarnings(report)
	return report
}

func suggestTranspose(noteRange NoteRangeDTO, cfg MidiConfigDTO) int {
	if noteRange.Min < 0 || noteRange.Max < 0 {
		return 0
	}
	currentMin := noteRange.Min + cfg.Transpose + cfg.OctaveShift*12
	currentMax := noteRange.Max + cfg.Transpose + cfg.OctaveShift*12
	upper := cfg.BaseNote + MaxLane
	if currentMin >= cfg.BaseNote && currentMax <= upper {
		return 0
	}
	if currentMax > upper {
		return clampInt(upper-currentMax, MinTranspose, MaxTranspose)
	}
	return clampInt(cfg.BaseNote-currentMin, MinTranspose, MaxTranspose)
}

func suggestOctaveShift(transpose int) int {
	if transpose == 0 {
		return 0
	}
	shift := transpose / 12
	if transpose > 0 && transpose%12 >= 6 {
		shift++
	}
	if transpose < 0 && transpose%12 <= -6 {
		shift--
	}
	return clampInt(shift, MinOctaveShift, MaxOctaveShift)
}

func filterEventsByEnabledTracks(events []storage.MidiEvent, enabledTracks *[]int) []storage.MidiEvent {
	if enabledTracks == nil {
		return events
	}
	allowed := make(map[int]bool, len(*enabledTracks))
	for _, track := range *enabledTracks {
		allowed[track] = true
	}
	out := make([]storage.MidiEvent, 0, len(events))
	for _, event := range events {
		if allowed[event.Track] {
			out = append(out, event)
		}
	}
	return out
}

func qualityWarnings(report QualityReportDTO) []string {
	warnings := make([]string, 0, 3)
	if report.OutOfRangeCount > 0 {
		warnings = append(warnings, "out_of_range")
	}
	if report.DroppedCount > 0 {
		warnings = append(warnings, "dropped_notes")
	}
	if report.ChordDensity >= 5 {
		warnings = append(warnings, "high_chord_density")
	}
	return warnings
}

func clampInt(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func isBlackMidiNote(note int) bool {
	switch positiveMod(note, 12) {
	case 1, 3, 6, 8, 10:
		return true
	default:
		return false
	}
}

func positiveMod(value, mod int) int {
	out := value % mod
	if out < 0 {
		out += mod
	}
	return out
}

func chordDensity(points []timePoint) int {
	if len(points) == 0 {
		return 0
	}
	// In the same millisecond, releases are applied before presses so back-to-back notes do not inflate density.
	sortTimePoints(points)
	current := 0
	max := 0
	for _, point := range points {
		current += point.delta
		if current < 0 {
			current = 0
		}
		if current > max {
			max = current
		}
	}
	return max
}

func sortTimePoints(points []timePoint) {
	sort.SliceStable(points, func(i, j int) bool {
		return timePointLess(points[i], points[j])
	})
}

func timePointLess(a, b timePoint) bool {
	if a.timeMs != b.timeMs {
		return a.timeMs < b.timeMs
	}
	return a.delta < b.delta
}

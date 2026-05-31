package midi

import (
	"errors"
	"sort"

	"YyslsPlayer/internal/storage"
)

var ErrPlayPlanEmpty = errors.New("PLAYPLAN_EMPTY")

type PlayPlanDTO struct {
	ProjectID      uint             `json:"projectId"`
	ProfileID      uint             `json:"profileId"`
	DurationMs     int64            `json:"durationMs"`
	Speed          float64          `json:"speed"`
	BaseNote       int              `json:"baseNote"`
	ConfigSnapshot MidiConfigDTO    `json:"configSnapshot"`
	Frames         []KeyFrameDTO    `json:"frames"`
	Report         QualityReportDTO `json:"report"`
}

type KeyFrameDTO struct {
	TimeMs         int64         `json:"timeMs"`
	Action         string        `json:"action"`
	Lane           int           `json:"lane"`
	SourceNote     int           `json:"sourceNote"`
	NormalizedNote int           `json:"normalizedNote"`
	RawLane        int           `json:"rawLane"`
	Velocity       int           `json:"velocity"`
	Key            KeymapLaneDTO `json:"key"`
}

const (
	KeyActionPress   = "press"
	KeyActionRelease = "release"
)

func buildPlayPlan(store *storage.Store, projectID uint, profileID uint) (PlayPlanDTO, error) {
	if projectID == 0 {
		return PlayPlanDTO{}, errors.New("projectID required")
	}
	project, ok := store.GetProject(projectID)
	if !ok {
		return PlayPlanDTO{}, errors.New("project not found")
	}

	profile, err := loadPlayPlanProfile(store, project, profileID)
	if err != nil {
		return PlayPlanDTO{}, err
	}
	if profile.ProjectID != nil && *profile.ProjectID != project.ID {
		return PlayPlanDTO{}, errors.New("profile does not belong to project")
	}
	cfg, err := ValidateMidiConfig(MidiConfigFromProfile(profile))
	if err != nil {
		return PlayPlanDTO{}, err
	}
	mapping, err := LoadKey36Mapping(store, cfg.KeymapProfileID)
	if err != nil {
		return PlayPlanDTO{}, err
	}

	events := store.ListEventsByProject(projectID)
	if len(events) == 0 {
		return PlayPlanDTO{}, ErrPlayPlanEmpty
	}
	events = filterEventsByEnabledTracks(events, cfg.EnabledTracks)
	if len(events) == 0 {
		return PlayPlanDTO{}, ErrPlayPlanEmpty
	}

	frames, _, durationMs := buildKeyFrames(events, cfg, mapping)
	if len(frames) == 0 {
		return PlayPlanDTO{}, ErrPlayPlanEmpty
	}
	report := qualityReportFromEventsWithConfig(project, events, cfg)

	return PlayPlanDTO{
		ProjectID:      project.ID,
		ProfileID:      profile.ID,
		DurationMs:     durationMs,
		Speed:          cfg.Speed,
		BaseNote:       cfg.BaseNote,
		ConfigSnapshot: cfg,
		Frames:         frames,
		Report:         report,
	}, nil
}

func loadPlayPlanProfile(store *storage.Store, project storage.MidiProject, profileID uint) (storage.MidiProfile, error) {
	if profileID != 0 {
		if profile, ok := store.GetProfile(profileID); ok {
			return profile, nil
		}
		return storage.MidiProfile{}, errors.New("profile not found")
	}
	profiles := store.ListProjectProfiles(project.ID)
	return loadDefaultProfile(store, project.DefaultProfileID, profiles)
}

func buildKeyFrames(events []storage.MidiEvent, cfg MidiConfigDTO, mapping Key36Mapping) ([]KeyFrameDTO, []LaneMappingDTO, int64) {
	frames := make([]KeyFrameDTO, 0, len(events)*2)
	results := make([]LaneMappingDTO, 0, len(events))
	var durationMs int64
	laneReleaseUntil := make(map[int]int64)

	for _, event := range events {
		mapped, err := MapEventToLaneKey(event, cfg, mapping)
		if err != nil {
			results = append(results, mapped)
			continue
		}
		results = append(results, mapped)
		if mapped.Dropped {
			continue
		}

		pressAt := scaleTime(event.StartMs, cfg.Speed)
		if lastRelease, ok := laneReleaseUntil[mapped.Lane]; ok && pressAt < lastRelease+int64(cfg.ReleaseGapMs) {
			pressAt = lastRelease + int64(cfg.ReleaseGapMs)
		}
		noteDuration := event.DurationMs
		if noteDuration < int64(cfg.MinPressMs) {
			noteDuration = int64(cfg.MinPressMs)
		}
		durationMsScaled := scaleDuration(noteDuration, cfg.Speed)
		if durationMsScaled <= 0 {
			durationMsScaled = 1
		}
		releaseAt := pressAt + durationMsScaled
		if releaseAt <= pressAt {
			releaseAt = pressAt + 1
		}
		laneReleaseUntil[mapped.Lane] = releaseAt

		frames = append(frames,
			keyFrameFromMapping(pressAt, KeyActionPress, event, mapped),
			keyFrameFromMapping(releaseAt, KeyActionRelease, event, mapped),
		)
		if releaseAt > durationMs {
			durationMs = releaseAt
		}
	}

	sort.SliceStable(frames, func(i, j int) bool {
		if frames[i].TimeMs != frames[j].TimeMs {
			return frames[i].TimeMs < frames[j].TimeMs
		}
		if frames[i].Action != frames[j].Action {
			return frameActionOrder(frames[i].Action) < frameActionOrder(frames[j].Action)
		}
		if frames[i].Lane != frames[j].Lane {
			return frames[i].Lane < frames[j].Lane
		}
		return frames[i].SourceNote < frames[j].SourceNote
	})
	return frames, results, durationMs
}

func keyFrameFromMapping(timeMs int64, action string, event storage.MidiEvent, mapped LaneMappingDTO) KeyFrameDTO {
	return KeyFrameDTO{
		TimeMs:         timeMs,
		Action:         action,
		Lane:           mapped.Lane,
		SourceNote:     mapped.SourceNote,
		NormalizedNote: mapped.NormalizedNote,
		RawLane:        mapped.RawLane,
		Velocity:       event.Velocity,
		Key:            mapped.Key,
	}
}

func scaleTime(ms int64, speed float64) int64 {
	if ms <= 0 {
		return 0
	}
	return int64((float64(ms) / speed) + 0.5)
}

func scaleDuration(ms int64, speed float64) int64 {
	if ms <= 0 {
		return 0
	}
	return int64((float64(ms) / speed) + 0.5)
}

func frameActionOrder(action string) int {
	if action == KeyActionRelease {
		return 0
	}
	return 1
}

func applyMappingStatsToReport(report *QualityReportDTO, stats MappingStats) {
	report.MappedRange = MappedRangeDTO{MinLane: stats.MinLane, MaxLane: stats.MaxLane}
	report.TotalNotes = stats.TotalNotes
	report.PlayableNotes = stats.PlayableNotes
	report.OutOfRangeCount = stats.OutOfRangeCount
	report.DroppedCount = stats.DroppedCount
	report.FoldedCount = stats.FoldedCount
	report.ClampedCount = stats.ClampedCount
	if stats.TotalNotes > 0 {
		report.PlayableRatio = float64(stats.PlayableNotes) / float64(stats.TotalNotes)
	}
}

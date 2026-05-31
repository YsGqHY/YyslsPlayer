// Package midi exposes basic MIDI library persistence operations.
package midi

import (
	"context"
	"errors"
	"path/filepath"
	"strings"

	"YyslsPlayer/internal/storage"
)

const (
	defaultProjectListLimit = 100
	maxProjectListLimit     = 1000

	importStatusImported = "imported"
	importStatusSkipped  = "skipped"
	importStatusFailed   = "failed"

	importReasonDuplicateInLibrary = "duplicate_in_library"
	importReasonDuplicateInBatch   = "duplicate_in_batch"
)

type Service struct {
	holder *storage.Holder
}

func New(holder *storage.Holder) *Service {
	return &Service{holder: holder}
}

func (s *Service) store() *storage.Store {
	return s.holder.Current().Store
}

type ListProjectsRequest struct {
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
	Query  string `json:"query"`
}

type MidiProjectSummary struct {
	ID               uint   `json:"id"`
	DisplayName      string `json:"displayName"`
	FileName         string `json:"fileName"`
	SourcePath       string `json:"sourcePath"`
	FileHash         string `json:"fileHash"`
	PPQ              int    `json:"ppq"`
	TrackCount       int    `json:"trackCount"`
	ChannelCount     int    `json:"channelCount"`
	DurationMs       int64  `json:"durationMs"`
	NoteCount        int    `json:"noteCount"`
	DefaultProfileID *uint  `json:"defaultProfileId"`
	CreatedAt        int64  `json:"createdAt"`
	UpdatedAt        int64  `json:"updatedAt"`
}

type MidiProfileDTO struct {
	ID               uint    `json:"id"`
	ProjectID        *uint   `json:"projectId"`
	Name             string  `json:"name"`
	BaseNote         int     `json:"baseNote"`
	Transpose        int     `json:"transpose"`
	OctaveShift      int     `json:"octaveShift"`
	Speed            float64 `json:"speed"`
	OutOfRangePolicy string  `json:"outOfRangePolicy"`
	MinPressMs       int     `json:"minPressMs"`
	ReleaseGapMs     int     `json:"releaseGapMs"`
	KeymapProfileID  uint    `json:"keymapProfileId"`
	EnabledTracks    *[]int  `json:"enabledTracks"`
	CreatedAt        int64   `json:"createdAt"`
	UpdatedAt        int64   `json:"updatedAt"`
}

type KeymapLaneDTO struct {
	ID               uint   `json:"id"`
	ProfileID        uint   `json:"profileId"`
	ProfileName      string `json:"profileName"`
	Lane             int    `json:"lane"`
	Label            string `json:"label"`
	PitchClass       int    `json:"pitchClass"`
	IsBlackKey       bool   `json:"isBlackKey"`
	VirtualKey       int    `json:"virtualKey"`
	ScanCode         int    `json:"scanCode"`
	ModifierKeysJSON string `json:"modifierKeysJson"`
	DisplayOrder     int    `json:"displayOrder"`
	CreatedAt        int64  `json:"createdAt"`
	UpdatedAt        int64  `json:"updatedAt"`
}

type KeymapProfileDTO struct {
	ProfileID   uint            `json:"profileId"`
	ProfileName string          `json:"profileName"`
	Lanes       []KeymapLaneDTO `json:"lanes"`
}

type MidiProjectDetail struct {
	Project          MidiProjectSummary `json:"project"`
	DefaultProfile   MidiProfileDTO     `json:"defaultProfile"`
	Profiles         []MidiProfileDTO   `json:"profiles"`
	DefaultKeymap    KeymapProfileDTO   `json:"defaultKeymap"`
	QualityReport    QualityReportDTO   `json:"qualityReport"`
	EventCount       int64              `json:"eventCount"`
	ProfileCount     int64              `json:"profileCount"`
	PlayHistoryCount int64              `json:"playHistoryCount"`
}

type ImportBatchItem struct {
	Path        string `json:"path"`
	FileName    string `json:"fileName"`
	FileHash    string `json:"fileHash"`
	ProjectID   *uint  `json:"projectId"`
	DisplayName string `json:"displayName"`
	Status      string `json:"status"`
	Reason      string `json:"reason"`
	Error       string `json:"error"`
}

type ImportBatchResult struct {
	TotalCount             int               `json:"totalCount"`
	ImportedCount          int               `json:"importedCount"`
	SkippedCount           int               `json:"skippedCount"`
	FailedCount            int               `json:"failedCount"`
	FirstProjectID         *uint             `json:"firstProjectId"`
	LastProjectID          *uint             `json:"lastProjectId"`
	FirstImportedProjectID *uint             `json:"firstImportedProjectId"`
	LastImportedProjectID  *uint             `json:"lastImportedProjectId"`
	Items                  []ImportBatchItem `json:"items"`
}

func (s *Service) ImportFile(ctx context.Context, path string) (MidiProjectDetail, error) {
	item, err := s.importOne(ctx, path, nil)
	if err != nil {
		return MidiProjectDetail{}, err
	}
	if item.ProjectID == nil {
		return MidiProjectDetail{}, errors.New("import did not return a project")
	}
	return s.GetProject(ctx, *item.ProjectID)
}

func (s *Service) ImportFiles(ctx context.Context, paths []string) (ImportBatchResult, error) {
	if len(paths) == 0 {
		return ImportBatchResult{}, errors.New("paths required")
	}
	return s.importMany(ctx, paths), nil
}

func (s *Service) ImportDirectory(ctx context.Context, dir string) (ImportBatchResult, error) {
	paths, err := findMidiFilesInDirectory(dir)
	if err != nil {
		return ImportBatchResult{}, err
	}
	return s.importMany(ctx, paths), nil
}

func (s *Service) ListProjects(_ context.Context, req ListProjectsRequest) ([]MidiProjectSummary, error) {
	rows := s.store().ListProjects(storage.ProjectListOptions{Limit: normalizeLimit(req.Limit), Offset: req.Offset, Query: req.Query})
	out := make([]MidiProjectSummary, 0, len(rows))
	for _, row := range rows {
		out = append(out, projectSummary(row))
	}
	return out, nil
}

func (s *Service) GetProject(_ context.Context, projectID uint) (MidiProjectDetail, error) {
	if projectID == 0 {
		return MidiProjectDetail{}, errors.New("projectID required")
	}
	store := s.store()
	project, ok := store.GetProject(projectID)
	if !ok {
		return MidiProjectDetail{}, errors.New("project not found")
	}
	profiles := store.ListProjectProfiles(project.ID)
	defaultProfile, err := loadDefaultProfile(store, project.DefaultProfileID, profiles)
	if err != nil {
		return MidiProjectDetail{}, err
	}
	keymap, err := loadKeymapProfile(store, defaultProfile.KeymapProfileID)
	if err != nil {
		return MidiProjectDetail{}, err
	}
	cfg, err := ValidateMidiConfig(MidiConfigFromProfile(defaultProfile))
	if err != nil {
		return MidiProjectDetail{}, err
	}
	reportCfg := cfg
	reportCfg.Transpose = 0
	reportCfg.OctaveShift = 0
	report := buildQualityReportWithConfig(store, project, reportCfg)
	displayProfiles := profiles
	if globalDefault, err := loadGlobalDefaultProfile(store); err == nil {
		displayProfiles = append(displayProfiles, globalDefault)
	}
	return MidiProjectDetail{
		Project:          projectSummary(project),
		DefaultProfile:   profileDTO(defaultProfile),
		Profiles:         profileDTOs(displayProfiles),
		DefaultKeymap:    keymap,
		QualityReport:    report,
		EventCount:       store.CountEventsByProject(project.ID),
		ProfileCount:     int64(len(displayProfiles)),
		PlayHistoryCount: store.CountHistoryByProject(project.ID),
	}, nil
}

func (s *Service) BuildPlayPlan(_ context.Context, projectID uint, profileID uint) (PlayPlanDTO, error) {
	return buildPlayPlan(s.store(), projectID, profileID)
}

func (s *Service) UpdateProfile(_ context.Context, profile MidiProfileDTO) (MidiProfileDTO, error) {
	if profile.ID == 0 {
		return MidiProfileDTO{}, errors.New("profile id required")
	}
	validated, err := ValidateMidiProfileDTO(profile)
	if err != nil {
		return MidiProfileDTO{}, err
	}
	store := s.store()
	row, ok := store.GetProfile(validated.ID)
	if !ok {
		return MidiProfileDTO{}, errors.New("profile not found")
	}
	if row.ProjectID == nil && validated.ProjectID != nil {
		if _, ok := store.GetProject(*validated.ProjectID); !ok {
			return MidiProfileDTO{}, errors.New("project not found")
		}
		created := row
		created.ID = 0
		created.ProjectID = validated.ProjectID
		created.CreatedAt = 0
		created.UpdatedAt = 0
		applyProfileDTOToRow(&created, validated)
		saved, err := store.SaveProfile(created)
		if err != nil {
			return MidiProfileDTO{}, err
		}
		if err := store.UpdateProjectDefaultProfile(*validated.ProjectID, saved.ID); err != nil {
			return MidiProfileDTO{}, err
		}
		return profileDTO(saved), nil
	}
	if row.ProjectID != nil && validated.ProjectID != nil && *row.ProjectID != *validated.ProjectID {
		return MidiProfileDTO{}, errors.New("profile project mismatch")
	}
	applyProfileDTOToRow(&row, validated)
	saved, err := store.SaveProfile(row)
	if err != nil {
		return MidiProfileDTO{}, err
	}
	if saved.ProjectID != nil {
		if err := store.UpdateProjectDefaultProfile(*saved.ProjectID, saved.ID); err != nil {
			return MidiProfileDTO{}, err
		}
	}
	return profileDTO(saved), nil
}

func (s *Service) GetDefaultProfile(_ context.Context) (MidiProfileDTO, error) {
	profile, err := loadGlobalDefaultProfile(s.store())
	if err != nil {
		return MidiProfileDTO{}, err
	}
	return profileDTO(profile), nil
}

func (s *Service) UpdateDefaultProfile(_ context.Context, profile MidiProfileDTO) (MidiProfileDTO, error) {
	if profile.ProjectID != nil {
		return MidiProfileDTO{}, errors.New("default profile must not be project scoped")
	}
	validated, err := ValidateMidiProfileDTO(profile)
	if err != nil {
		return MidiProfileDTO{}, err
	}
	row, err := loadGlobalDefaultProfile(s.store())
	if err != nil {
		return MidiProfileDTO{}, err
	}
	if validated.ID != 0 && validated.ID != row.ID {
		return MidiProfileDTO{}, errors.New("default profile mismatch")
	}
	applyProfileDTOToRow(&row, validated)
	row.ProjectID = nil
	saved, err := s.store().SaveProfile(row)
	if err != nil {
		return MidiProfileDTO{}, err
	}
	return profileDTO(saved), nil
}

func (s *Service) ResetDefaultProfile(_ context.Context) (MidiProfileDTO, error) {
	row, err := loadGlobalDefaultProfile(s.store())
	if err != nil {
		return MidiProfileDTO{}, err
	}
	row.Name = storage.DefaultMidiProfileName
	row.BaseNote = storage.DefaultMidiBaseNote
	row.Transpose = storage.DefaultMidiTranspose
	row.OctaveShift = storage.DefaultMidiOctaveShift
	row.Speed = storage.DefaultMidiSpeed
	row.OutOfRangePolicy = storage.DefaultOutOfRangePolicy
	row.MinPressMs = storage.DefaultMinPressMs
	row.ReleaseGapMs = storage.DefaultReleaseGapMs
	row.KeymapProfileID = storage.DefaultKeymapProfileID
	row.EnabledTracksJSON = "null"
	row.ProjectID = nil
	saved, err := s.store().SaveProfile(row)
	if err != nil {
		return MidiProfileDTO{}, err
	}
	return profileDTO(saved), nil
}

func (s *Service) DeleteProject(_ context.Context, projectID uint) error {
	if projectID == 0 {
		return errors.New("projectID required")
	}
	return s.store().DeleteProject(projectID)
}

func (s *Service) GetDefaultKeymap(_ context.Context) (KeymapProfileDTO, error) {
	return loadKeymapProfile(s.store(), storage.DefaultKeymapProfileID)
}

func (s *Service) importMany(ctx context.Context, paths []string) ImportBatchResult {
	result := ImportBatchResult{TotalCount: len(paths), Items: make([]ImportBatchItem, 0, len(paths))}
	seenHashes := make(map[string]uint)
	for _, path := range paths {
		item, err := s.importOne(ctx, path, seenHashes)
		if err != nil {
			item = ImportBatchItem{Path: strings.TrimSpace(path), FileName: filepath.Base(strings.TrimSpace(path)), Status: importStatusFailed, Error: err.Error()}
		}
		result.addItem(item)
	}
	return result
}

func (s *Service) importOne(_ context.Context, path string, seenHashes map[string]uint) (ImportBatchItem, error) {
	file, err := readMidiFile(path)
	if err != nil {
		return ImportBatchItem{}, err
	}
	item := ImportBatchItem{Path: file.Path, FileName: file.Name, FileHash: file.FileHash}
	if seenHashes != nil {
		if projectID, ok := seenHashes[file.FileHash]; ok {
			item.Status = importStatusSkipped
			item.Reason = importReasonDuplicateInBatch
			if projectID != 0 {
				item.ProjectID = &projectID
			}
			return item, nil
		}
	}
	if existing, ok := s.store().GetProjectByHash(file.FileHash); ok {
		item.Status = importStatusSkipped
		item.Reason = importReasonDuplicateInLibrary
		item.ProjectID = &existing.ID
		item.DisplayName = existing.DisplayName
		if seenHashes != nil {
			seenHashes[file.FileHash] = existing.ID
		}
		return item, nil
	}
	project, err := importNewMidiProject(s.store(), file)
	if err != nil {
		return item, err
	}
	item.Status = importStatusImported
	item.ProjectID = &project.ID
	item.DisplayName = project.DisplayName
	if seenHashes != nil {
		seenHashes[file.FileHash] = project.ID
	}
	return item, nil
}

func importNewMidiProject(store *storage.Store, file midiFileData) (storage.MidiProject, error) {
	score, err := parseNormalizedScore(file.Bytes)
	if err != nil {
		return storage.MidiProject{}, err
	}
	project := storage.MidiProject{DisplayName: fileNameWithoutExt(file.Name), FileName: file.Name, SourcePath: &file.Path, FileHash: file.FileHash, PPQ: score.PPQ, TrackCount: score.TrackCount, ChannelCount: score.ChannelCount, DurationMs: score.DurationMs, NoteCount: len(score.Events)}
	return store.ImportProject(storage.ProjectImportData{Project: project, Events: score.Events})
}

func (r *ImportBatchResult) addItem(item ImportBatchItem) {
	r.Items = append(r.Items, item)
	switch item.Status {
	case importStatusImported:
		r.ImportedCount++
	case importStatusSkipped:
		r.SkippedCount++
	case importStatusFailed:
		r.FailedCount++
	}
	if item.ProjectID == nil {
		return
	}
	if r.FirstProjectID == nil {
		r.FirstProjectID = item.ProjectID
	}
	r.LastProjectID = item.ProjectID
	if item.Status == importStatusImported {
		if r.FirstImportedProjectID == nil {
			r.FirstImportedProjectID = item.ProjectID
		}
		r.LastImportedProjectID = item.ProjectID
	}
}

func fileNameWithoutExt(name string) string {
	base := filepath.Base(name)
	ext := filepath.Ext(base)
	if ext == "" {
		return base
	}
	return strings.TrimSuffix(base, ext)
}

func normalizeLimit(limit int) int {
	if limit <= 0 {
		return defaultProjectListLimit
	}
	if limit > maxProjectListLimit {
		return maxProjectListLimit
	}
	return limit
}

func loadDefaultProfile(store *storage.Store, profileID *uint, candidates []storage.MidiProfile) (storage.MidiProfile, error) {
	if profileID != nil {
		for _, row := range candidates {
			if row.ID == *profileID {
				return row, nil
			}
		}
		if row, ok := store.GetProfile(*profileID); ok {
			return row, nil
		}
		return storage.MidiProfile{}, errors.New("profile not found")
	}
	for _, row := range candidates {
		if row.ProjectID == nil {
			return row, nil
		}
	}
	return loadGlobalDefaultProfile(store)
}

func loadGlobalDefaultProfile(store *storage.Store) (storage.MidiProfile, error) {
	if row, ok := store.GetGlobalDefaultProfile(); ok {
		return row, nil
	}
	return storage.MidiProfile{}, errors.New("default profile not found")
}

func loadKeymapProfile(store *storage.Store, profileID uint) (KeymapProfileDTO, error) {
	if profileID == 0 {
		return KeymapProfileDTO{}, errors.New("keymap profile id required")
	}
	rows := store.ListKeymapProfile(profileID)
	if len(rows) == 0 {
		return KeymapProfileDTO{}, errors.New("keymap profile not found")
	}
	lanes := make([]KeymapLaneDTO, 0, len(rows))
	for _, row := range rows {
		lanes = append(lanes, keymapLaneDTO(row))
	}
	return KeymapProfileDTO{ProfileID: rows[0].ProfileID, ProfileName: rows[0].ProfileName, Lanes: lanes}, nil
}

func projectSummary(row storage.MidiProject) MidiProjectSummary {
	sourcePath := ""
	if row.SourcePath != nil {
		sourcePath = *row.SourcePath
	}
	return MidiProjectSummary{ID: row.ID, DisplayName: row.DisplayName, FileName: row.FileName, SourcePath: sourcePath, FileHash: row.FileHash, PPQ: row.PPQ, TrackCount: row.TrackCount, ChannelCount: row.ChannelCount, DurationMs: row.DurationMs, NoteCount: row.NoteCount, DefaultProfileID: row.DefaultProfileID, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
}

func profileDTO(row storage.MidiProfile) MidiProfileDTO {
	return MidiProfileDTO{ID: row.ID, ProjectID: row.ProjectID, Name: row.Name, BaseNote: row.BaseNote, Transpose: row.Transpose, OctaveShift: row.OctaveShift, Speed: row.Speed, OutOfRangePolicy: row.OutOfRangePolicy, MinPressMs: row.MinPressMs, ReleaseGapMs: row.ReleaseGapMs, KeymapProfileID: row.KeymapProfileID, EnabledTracks: decodeEnabledTracks(row.EnabledTracksJSON), CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
}

func profileDTOs(rows []storage.MidiProfile) []MidiProfileDTO {
	out := make([]MidiProfileDTO, 0, len(rows))
	for _, row := range rows {
		out = append(out, profileDTO(row))
	}
	return out
}

func applyProfileDTOToRow(row *storage.MidiProfile, profile MidiProfileDTO) {
	name := strings.TrimSpace(profile.Name)
	if name != "" {
		row.Name = name
	}
	row.BaseNote = profile.BaseNote
	row.Transpose = profile.Transpose
	row.OctaveShift = profile.OctaveShift
	row.Speed = profile.Speed
	row.OutOfRangePolicy = profile.OutOfRangePolicy
	row.MinPressMs = profile.MinPressMs
	row.ReleaseGapMs = profile.ReleaseGapMs
	row.KeymapProfileID = profile.KeymapProfileID
	row.EnabledTracksJSON = encodeEnabledTracks(profile.EnabledTracks)
}

func keymapLaneDTO(row storage.Keymap36) KeymapLaneDTO {
	return KeymapLaneDTO{ID: row.ID, ProfileID: row.ProfileID, ProfileName: row.ProfileName, Lane: row.Lane, Label: row.Label, PitchClass: row.PitchClass, IsBlackKey: row.IsBlackKey, VirtualKey: row.VirtualKey, ScanCode: row.ScanCode, ModifierKeysJSON: row.ModifierKeysJSON, DisplayOrder: row.DisplayOrder, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
}

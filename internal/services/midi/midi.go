// Package midi exposes basic MIDI library persistence operations.
package midi

import (
	"context"
	"errors"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

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

	projectBatchStatusDeleted = "deleted"
	projectBatchStatusFailed  = "failed"

	maxImportWorkers = 16
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
	FileSizeBytes    int64  `json:"fileSizeBytes"`
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

type ProjectBatchManageItem struct {
	ProjectID   uint   `json:"projectId"`
	DisplayName string `json:"displayName"`
	Status      string `json:"status"`
	Reason      string `json:"reason"`
	Error       string `json:"error"`
}

type ProjectBatchManageResult struct {
	TotalCount   int                      `json:"totalCount"`
	DeletedCount int                      `json:"deletedCount"`
	FailedCount  int                      `json:"failedCount"`
	Items        []ProjectBatchManageItem `json:"items"`
}

type preparedImport struct {
	Index int
	Path  string
	File  midiFileData
	Input storage.ProjectImportData
	Err   error
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

// ImportPaths 接收拖拽进来的原始路径集合（可能混含文件与文件夹）。
// 文件夹会递归展开为其中的 MIDI 文件，非 MIDI 文件按导入失败逐项上报，
// 最终复用 importMany 的去重与批量导入逻辑。
func (s *Service) ImportPaths(ctx context.Context, paths []string) (ImportBatchResult, error) {
	files, err := expandDroppedPaths(paths)
	if err != nil {
		return ImportBatchResult{}, err
	}
	if len(files) == 0 {
		return ImportBatchResult{}, ErrMidiFileNotFound
	}
	return s.importMany(ctx, files), nil
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

func (s *Service) DeleteProject(ctx context.Context, projectID uint) error {
	if projectID == 0 {
		return errors.New("projectID required")
	}
	result, err := s.DeleteProjects(ctx, []uint{projectID})
	if err != nil {
		return err
	}
	if result.DeletedCount != 1 {
		return errors.New("project not found")
	}
	return nil
}

func (s *Service) DeleteProjects(_ context.Context, projectIDs []uint) (ProjectBatchManageResult, error) {
	if len(projectIDs) == 0 {
		return ProjectBatchManageResult{}, errors.New("projectIDs required")
	}
	rows, err := s.store().DeleteProjectsBatch(projectIDs)
	if err != nil {
		return ProjectBatchManageResult{}, err
	}
	result := ProjectBatchManageResult{TotalCount: len(projectIDs), Items: make([]ProjectBatchManageItem, 0, len(rows))}
	for _, row := range rows {
		item := ProjectBatchManageItem{ProjectID: row.ProjectID}
		if row.Project.ID != 0 {
			item.DisplayName = row.Project.DisplayName
		}
		if row.Deleted {
			item.Status = projectBatchStatusDeleted
			result.DeletedCount++
		} else {
			item.Status = projectBatchStatusFailed
			item.Reason = row.Reason
			item.Error = row.Reason
			result.FailedCount++
		}
		result.Items = append(result.Items, item)
	}
	return result, nil
}

func (s *Service) GetDefaultKeymap(_ context.Context) (KeymapProfileDTO, error) {
	return loadKeymapProfile(s.store(), storage.DefaultKeymapProfileID)
}

func (s *Service) importMany(ctx context.Context, paths []string) ImportBatchResult {
	store := s.store()
	prepared := prepareImportsConcurrently(ctx, paths)
	items := make([]ImportBatchItem, len(paths))
	pending := make([]preparedImport, 0, len(paths))
	existingByHash := store.ProjectHashIndex()
	seenHashes := make(map[string]uint, len(paths))

	for i, prep := range prepared {
		if prep.Err != nil {
			trimmed := strings.TrimSpace(prep.Path)
			items[i] = ImportBatchItem{Path: trimmed, FileName: filepath.Base(trimmed), Status: importStatusFailed, Error: prep.Err.Error()}
			continue
		}

		item := ImportBatchItem{Path: prep.File.Path, FileName: prep.File.Name, FileHash: prep.File.FileHash}
		if projectID, ok := seenHashes[prep.File.FileHash]; ok {
			item.Status = importStatusSkipped
			item.Reason = importReasonDuplicateInBatch
			if projectID != 0 {
				item.ProjectID = &projectID
			}
			items[i] = item
			continue
		}
		if existing, ok := existingByHash[prep.File.FileHash]; ok {
			item.Status = importStatusSkipped
			item.Reason = importReasonDuplicateInLibrary
			item.ProjectID = &existing.ID
			item.DisplayName = existing.DisplayName
			seenHashes[prep.File.FileHash] = existing.ID
			items[i] = item
			continue
		}

		seenHashes[prep.File.FileHash] = 0
		pending = append(pending, prep)
		items[i] = item
	}

	if len(pending) > 0 {
		inputs := make([]storage.ProjectImportData, len(pending))
		pendingHashes := make(map[string]struct{}, len(pending))
		for i, prep := range pending {
			inputs[i] = prep.Input
			pendingHashes[prep.File.FileHash] = struct{}{}
		}
		batchResults, err := store.ImportProjectsBatch(inputs)
		if err != nil {
			for _, prep := range pending {
				items[prep.Index].Status = importStatusFailed
				items[prep.Index].Error = err.Error()
			}
			for i := range items {
				if items[i].Status == importStatusSkipped && items[i].Reason == importReasonDuplicateInBatch {
					if _, ok := pendingHashes[items[i].FileHash]; ok {
						items[i].Status = importStatusFailed
						items[i].Reason = ""
						items[i].Error = err.Error()
					}
				}
			}
		} else {
			for i, batchItem := range batchResults {
				prep := pending[i]
				item := items[prep.Index]
				project := batchItem.Project
				if project.ID != 0 {
					item.ProjectID = &project.ID
					item.DisplayName = project.DisplayName
					seenHashes[prep.File.FileHash] = project.ID
				}
				switch batchItem.Status {
				case storage.ProjectBatchImportStatusImported:
					item.Status = importStatusImported
				case storage.ProjectBatchImportStatusSkipped:
					item.Status = importStatusSkipped
					item.Reason = batchItem.Reason
					if item.Reason == "" {
						item.Reason = importReasonDuplicateInLibrary
					}
				default:
					item.Status = importStatusFailed
					item.Error = "unknown batch import status"
				}
				items[prep.Index] = item
			}
			for i := range items {
				if items[i].Status != importStatusSkipped || items[i].Reason != importReasonDuplicateInBatch || items[i].ProjectID != nil {
					continue
				}
				if projectID := seenHashes[items[i].FileHash]; projectID != 0 {
					items[i].ProjectID = &projectID
				}
			}
		}
	}

	result := ImportBatchResult{TotalCount: len(paths), Items: make([]ImportBatchItem, 0, len(paths))}
	for _, item := range items {
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
	input, err := buildProjectImportData(file)
	if err != nil {
		return storage.MidiProject{}, err
	}
	return store.ImportProject(input)
}

func prepareImportsConcurrently(ctx context.Context, paths []string) []preparedImport {
	if ctx == nil {
		ctx = context.Background()
	}
	results := make([]preparedImport, len(paths))
	if len(paths) == 0 {
		return results
	}
	workerCount := importWorkerCount(len(paths))
	jobs := make(chan int)
	var wg sync.WaitGroup
	wg.Add(workerCount)
	for worker := 0; worker < workerCount; worker++ {
		go func() {
			defer wg.Done()
			for idx := range jobs {
				path := paths[idx]
				prep := preparedImport{Index: idx, Path: path}
				if ctx.Err() != nil {
					prep.Err = ctx.Err()
					results[idx] = prep
					continue
				}
				file, err := readMidiFile(path)
				if err != nil {
					prep.Err = err
					results[idx] = prep
					continue
				}
				input, err := buildProjectImportData(file)
				prep.File = file
				prep.Input = input
				prep.Err = err
				results[idx] = prep
			}
		}()
	}
	for i := range paths {
		if ctx.Err() != nil {
			results[i] = preparedImport{Index: i, Path: paths[i], Err: ctx.Err()}
			continue
		}
		jobs <- i
	}
	close(jobs)
	wg.Wait()
	return results
}

func importWorkerCount(total int) int {
	if total <= 1 {
		return 1
	}
	workers := runtime.NumCPU() * 2
	if workers < 2 {
		workers = 2
	}
	if workers > maxImportWorkers {
		workers = maxImportWorkers
	}
	if workers > total {
		workers = total
	}
	return workers
}

func buildProjectImportData(file midiFileData) (storage.ProjectImportData, error) {
	score, err := parseNormalizedScore(file.Bytes)
	if err != nil {
		return storage.ProjectImportData{}, err
	}
	project := storage.MidiProject{
		DisplayName:   fileNameWithoutExt(file.Name),
		FileName:      file.Name,
		SourcePath:    &file.Path,
		FileHash:      file.FileHash,
		PPQ:           score.PPQ,
		TrackCount:    score.TrackCount,
		ChannelCount:  score.ChannelCount,
		DurationMs:    score.DurationMs,
		NoteCount:     len(score.Events),
		FileSizeBytes: file.Size,
	}
	return storage.ProjectImportData{Project: project, Events: score.Events}, nil
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
	return MidiProjectSummary{
		ID:               row.ID,
		DisplayName:      row.DisplayName,
		FileName:         row.FileName,
		SourcePath:       sourcePath,
		FileHash:         row.FileHash,
		PPQ:              row.PPQ,
		TrackCount:       row.TrackCount,
		ChannelCount:     row.ChannelCount,
		DurationMs:       row.DurationMs,
		NoteCount:        row.NoteCount,
		FileSizeBytes:    row.FileSizeBytes,
		DefaultProfileID: row.DefaultProfileID,
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
	}
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

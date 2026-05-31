package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"YyslsPlayer/internal/utils/filex"
)

const StoreVersion = 1

type Store struct {
	path string
	mu   sync.RWMutex
	data storeData
}

type storeData struct {
	Version        int             `json:"version"`
	NextIDs        nextIDs         `json:"nextIds"`
	Preferences    []Preference    `json:"preferences"`
	AppSettings    AppSettings     `json:"appSettings"`
	MidiProjects   []MidiProject   `json:"midiProjects"`
	MidiEvents     []MidiEvent     `json:"midiEvents"`
	MidiProfiles   []MidiProfile   `json:"midiProfiles"`
	Keymap36       []Keymap36      `json:"keymap36"`
	PlayHistory    []PlayHistory   `json:"playHistory"`
	HotkeyBindings []HotkeyBinding `json:"hotkeyBindings"`
}

type nextIDs struct {
	MidiProject uint `json:"midiProject"`
	MidiEvent   uint `json:"midiEvent"`
	MidiProfile uint `json:"midiProfile"`
	Keymap36    uint `json:"keymap36"`
	PlayHistory uint `json:"playHistory"`
}

type ProjectListOptions struct {
	Limit  int
	Offset int
	Query  string
}

type ProjectImportData struct {
	Project MidiProject
	Events  []MidiEvent
}

const (
	ProjectBatchImportStatusImported = "imported"
	ProjectBatchImportStatusSkipped  = "skipped"

	ProjectBatchImportReasonDuplicateInLibrary = "duplicate_in_library"
)

type ProjectBatchImportResult struct {
	Project MidiProject
	Status  string
	Reason  string
}

const (
	ProjectDeleteReasonNotFound  = "project_not_found"
	ProjectDeleteReasonInvalidID = "invalid_project_id"
	ProjectDeleteReasonDuplicate = "duplicate_project_id"
)

type ProjectDeleteResult struct {
	ProjectID uint
	Project   MidiProject
	Deleted   bool
	Reason    string
}

func OpenStore(path string) (*Store, error) {
	if path == "" {
		return nil, errors.New("storage: empty data path")
	}
	st := &Store{path: path}
	if err := st.load(); err != nil {
		return nil, err
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	st.ensureDefaultsLocked()
	if err := st.persistLocked(); err != nil {
		return nil, err
	}
	return st, nil
}

func (s *Store) Path() string { return s.path }

func (s *Store) Close() error { return nil }

func (s *Store) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ensureDir(dirOf(s.path)); err != nil {
		return fmt.Errorf("storage: ensure dir: %w", err)
	}
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			s.data = newStoreData()
			return nil
		}
		return fmt.Errorf("storage: read json: %w", err)
	}
	if len(data) == 0 {
		s.data = newStoreData()
		return nil
	}
	var decoded storeData
	if err := json.Unmarshal(data, &decoded); err != nil {
		return fmt.Errorf("storage: decode json: %w", err)
	}
	s.data = decoded
	if s.data.Version == 0 {
		s.data.Version = StoreVersion
	}
	s.repairNextIDsLocked()
	return nil
}

func newStoreData() storeData {
	return storeData{Version: StoreVersion, AppSettings: defaultAppSettings()}
}

func defaultAppSettings() AppSettings {
	return AppSettings{ID: 1, ThemeChoice: "system", LocaleChoice: "auto"}
}

func dirOf(path string) string {
	return filepath.Dir(path)
}

func (s *Store) persistLocked() error {
	s.data.Version = StoreVersion
	s.repairNextIDsLocked()
	data, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	return filex.WriteAtomic(s.path, data, 0o644)
}

func (s *Store) repairNextIDsLocked() {
	if min := maxProjectID(s.data.MidiProjects) + 1; s.data.NextIDs.MidiProject < min {
		s.data.NextIDs.MidiProject = min
	}
	if min := maxEventID(s.data.MidiEvents) + 1; s.data.NextIDs.MidiEvent < min {
		s.data.NextIDs.MidiEvent = min
	}
	if min := maxProfileID(s.data.MidiProfiles) + 1; s.data.NextIDs.MidiProfile < min {
		s.data.NextIDs.MidiProfile = min
	}
	if min := maxKeymapID(s.data.Keymap36) + 1; s.data.NextIDs.Keymap36 < min {
		s.data.NextIDs.Keymap36 = min
	}
	if min := maxHistoryID(s.data.PlayHistory) + 1; s.data.NextIDs.PlayHistory < min {
		s.data.NextIDs.PlayHistory = min
	}
}

func (s *Store) ensureDefaultsLocked() {
	if s.data.AppSettings.ID == 0 {
		s.data.AppSettings = defaultAppSettings()
	}
	if s.data.AppSettings.ThemeChoice == "" {
		s.data.AppSettings.ThemeChoice = "system"
	}
	if s.data.AppSettings.LocaleChoice == "" {
		s.data.AppSettings.LocaleChoice = "auto"
	}
	s.ensureDefaultKeymapLocked()
	s.ensureDefaultProfileLocked()
	s.repairNextIDsLocked()
}

func (s *Store) ensureDefaultProfileLocked() {
	for _, row := range s.data.MidiProfiles {
		if row.ProjectID == nil {
			return
		}
	}
	profile := defaultMidiProfile()
	if s.profileIDExistsLocked(DefaultMidiProfileID) {
		profile.ID = s.nextProfileIDLocked()
	}
	now := nowMillis()
	if profile.CreatedAt == 0 {
		profile.CreatedAt = now
	}
	profile.UpdatedAt = now
	s.data.MidiProfiles = append(s.data.MidiProfiles, profile)
}

func (s *Store) ensureDefaultKeymapLocked() {
	exists := map[int]bool{}
	for _, row := range s.data.Keymap36 {
		if row.ProfileID == DefaultKeymapProfileID {
			exists[row.Lane] = true
		}
	}
	now := nowMillis()
	for _, row := range defaultKeymap36Rows() {
		if exists[row.Lane] {
			continue
		}
		if row.ID == 0 || s.keymapIDExistsLocked(row.ID) {
			row.ID = s.nextKeymapIDLocked()
		}
		if row.CreatedAt == 0 {
			row.CreatedAt = now
		}
		row.UpdatedAt = now
		s.data.Keymap36 = append(s.data.Keymap36, row)
	}
}

func (s *Store) WithWrite(fn func(*storeData) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := fn(&s.data); err != nil {
		return err
	}
	s.ensureDefaultsLocked()
	return s.persistLocked()
}

func (s *Store) GetPreference(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, row := range s.data.Preferences {
		if row.Key == key {
			return row.Value, true
		}
	}
	return "", false
}

func (s *Store) SetPreference(key, value string) error {
	return s.WithWrite(func(d *storeData) error {
		now := nowMillis()
		for i := range d.Preferences {
			if d.Preferences[i].Key == key {
				d.Preferences[i].Value = value
				d.Preferences[i].UpdatedAt = now
				return nil
			}
		}
		d.Preferences = append(d.Preferences, Preference{Key: key, Value: value, UpdatedAt: now})
		return nil
	})
}

func (s *Store) ListPreferences() []Preference {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := append([]Preference(nil), s.data.Preferences...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

func (s *Store) DeletePreference(key string) error {
	return s.WithWrite(func(d *storeData) error {
		d.Preferences = filterSlice(d.Preferences, func(row Preference) bool { return row.Key != key })
		return nil
	})
}

func (s *Store) ClearPreferences() error {
	return s.WithWrite(func(d *storeData) error { d.Preferences = nil; return nil })
}

func (s *Store) GetAppSettings() AppSettings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data.AppSettings
}

func (s *Store) UpdateAppSettings(mut func(*AppSettings)) error {
	return s.WithWrite(func(d *storeData) error {
		mut(&d.AppSettings)
		d.AppSettings.ID = 1
		d.AppSettings.UpdatedAt = nowMillis()
		return nil
	})
}

func (s *Store) CountHotkeyBindings() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return int64(len(s.data.HotkeyBindings))
}

func (s *Store) ListHotkeyBindings() []HotkeyBinding {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]HotkeyBinding(nil), s.data.HotkeyBindings...)
}

func (s *Store) SaveHotkeyBinding(row HotkeyBinding) error {
	return s.WithWrite(func(d *storeData) error {
		now := nowMillis()
		row.UpdatedAt = now
		for i := range d.HotkeyBindings {
			if d.HotkeyBindings[i].ActionID == row.ActionID {
				d.HotkeyBindings[i] = row
				return nil
			}
		}
		d.HotkeyBindings = append(d.HotkeyBindings, row)
		return nil
	})
}

func (s *Store) GetHotkeyBinding(actionID string) (HotkeyBinding, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, row := range s.data.HotkeyBindings {
		if row.ActionID == actionID {
			return row, true
		}
	}
	return HotkeyBinding{}, false
}

func (s *Store) ClearHotkeyBindings() error {
	return s.WithWrite(func(d *storeData) error { d.HotkeyBindings = nil; return nil })
}

func (s *Store) ListProjects(opts ProjectListOptions) []MidiProject {
	s.mu.RLock()
	defer s.mu.RUnlock()
	query := strings.ToLower(strings.TrimSpace(opts.Query))
	rows := make([]MidiProject, 0, len(s.data.MidiProjects))
	for _, row := range s.data.MidiProjects {
		if query != "" && !strings.Contains(strings.ToLower(row.DisplayName), query) && !strings.Contains(strings.ToLower(row.FileName), query) && !strings.Contains(strings.ToLower(row.FileHash), query) {
			continue
		}
		rows = append(rows, row)
	}
	sortProjects(rows)
	if opts.Offset < 0 {
		opts.Offset = 0
	}
	if opts.Offset >= len(rows) {
		return []MidiProject{}
	}
	end := opts.Offset + opts.Limit
	if opts.Limit <= 0 || end > len(rows) {
		end = len(rows)
	}
	return append([]MidiProject(nil), rows[opts.Offset:end]...)
}

func (s *Store) GetProject(id uint) (MidiProject, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return findProject(s.data.MidiProjects, id)
}

func (s *Store) GetProjectByHash(hash string) (MidiProject, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, row := range s.data.MidiProjects {
		if row.FileHash == hash {
			return row, true
		}
	}
	return MidiProject{}, false
}

func (s *Store) ProjectHashIndex() map[string]MidiProject {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]MidiProject, len(s.data.MidiProjects))
	for _, row := range s.data.MidiProjects {
		if row.FileHash == "" {
			continue
		}
		out[row.FileHash] = row
	}
	return out
}

func (s *Store) ImportProject(input ProjectImportData) (MidiProject, error) {
	var imported MidiProject
	err := s.WithWrite(func(d *storeData) error {
		if _, ok := findProjectByHash(d.MidiProjects, input.Project.FileHash); ok {
			return errors.New("duplicate project")
		}
		now := nowMillis()
		project := input.Project
		project.ID = s.nextProjectIDLocked()
		if project.CreatedAt == 0 {
			project.CreatedAt = now
		}
		project.UpdatedAt = now
		d.MidiProjects = append(d.MidiProjects, project)
		for _, event := range input.Events {
			event.ID = s.nextEventIDLocked()
			event.ProjectID = project.ID
			d.MidiEvents = append(d.MidiEvents, event)
		}
		imported = project
		return nil
	})
	return imported, err
}

func (s *Store) ImportProjectsBatch(inputs []ProjectImportData) ([]ProjectBatchImportResult, error) {
	if len(inputs) == 0 {
		return []ProjectBatchImportResult{}, nil
	}
	results := make([]ProjectBatchImportResult, len(inputs))
	err := s.WithWrite(func(d *storeData) error {
		existingByHash := make(map[string]MidiProject, len(d.MidiProjects)+len(inputs))
		for _, row := range d.MidiProjects {
			if row.FileHash == "" {
				continue
			}
			existingByHash[row.FileHash] = row
		}

		now := nowMillis()
		for i, input := range inputs {
			if input.Project.FileHash != "" {
				if existing, ok := existingByHash[input.Project.FileHash]; ok {
					results[i] = ProjectBatchImportResult{Project: existing, Status: ProjectBatchImportStatusSkipped, Reason: ProjectBatchImportReasonDuplicateInLibrary}
					continue
				}
			}

			project := input.Project
			project.ID = s.nextProjectIDLocked()
			if project.CreatedAt == 0 {
				project.CreatedAt = now
			}
			project.UpdatedAt = now
			d.MidiProjects = append(d.MidiProjects, project)
			for _, event := range input.Events {
				event.ID = s.nextEventIDLocked()
				event.ProjectID = project.ID
				d.MidiEvents = append(d.MidiEvents, event)
			}
			if project.FileHash != "" {
				existingByHash[project.FileHash] = project
			}
			results[i] = ProjectBatchImportResult{Project: project, Status: ProjectBatchImportStatusImported}
		}
		return nil
	})
	return results, err
}

func (s *Store) ListProjectProfiles(projectID uint) []MidiProfile {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rows := make([]MidiProfile, 0)
	for _, row := range s.data.MidiProfiles {
		if row.ProjectID != nil && *row.ProjectID == projectID {
			rows = append(rows, row)
		}
	}
	sort.SliceStable(rows, func(i, j int) bool {
		return rows[i].ID < rows[j].ID
	})
	return rows
}

func (s *Store) GetProfile(id uint) (MidiProfile, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return findProfile(s.data.MidiProfiles, id)
}

func (s *Store) GetGlobalDefaultProfile() (MidiProfile, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, row := range s.data.MidiProfiles {
		if row.ProjectID == nil {
			return row, true
		}
	}
	return MidiProfile{}, false
}

func (s *Store) SaveProfile(row MidiProfile) (MidiProfile, error) {
	var saved MidiProfile
	err := s.WithWrite(func(d *storeData) error {
		now := nowMillis()
		if row.ID == 0 {
			row.ID = s.nextProfileIDLocked()
			row.CreatedAt = now
		} else if old, ok := findProfile(d.MidiProfiles, row.ID); ok && row.CreatedAt == 0 {
			row.CreatedAt = old.CreatedAt
		}
		row.UpdatedAt = now
		for i := range d.MidiProfiles {
			if d.MidiProfiles[i].ID == row.ID {
				d.MidiProfiles[i] = row
				saved = row
				return nil
			}
		}
		d.MidiProfiles = append(d.MidiProfiles, row)
		saved = row
		return nil
	})
	return saved, err
}

func (s *Store) UpdateProjectDefaultProfile(projectID uint, profileID uint) error {
	return s.WithWrite(func(d *storeData) error {
		for i := range d.MidiProjects {
			if d.MidiProjects[i].ID == projectID {
				d.MidiProjects[i].DefaultProfileID = &profileID
				d.MidiProjects[i].UpdatedAt = nowMillis()
				return nil
			}
		}
		return errors.New("project not found")
	})
}

func (s *Store) SaveProject(row MidiProject) (MidiProject, error) {
	var saved MidiProject
	err := s.WithWrite(func(d *storeData) error {
		now := nowMillis()
		if row.ID == 0 {
			row.ID = s.nextProjectIDLocked()
			row.CreatedAt = now
		} else if old, ok := findProject(d.MidiProjects, row.ID); ok && row.CreatedAt == 0 {
			row.CreatedAt = old.CreatedAt
		}
		row.UpdatedAt = now
		for i := range d.MidiProjects {
			if d.MidiProjects[i].ID == row.ID {
				d.MidiProjects[i] = row
				saved = row
				return nil
			}
		}
		d.MidiProjects = append(d.MidiProjects, row)
		saved = row
		return nil
	})
	return saved, err
}

func (s *Store) AddEvents(events []MidiEvent) error {
	return s.WithWrite(func(d *storeData) error {
		for _, event := range events {
			event.ID = s.nextEventIDLocked()
			d.MidiEvents = append(d.MidiEvents, event)
		}
		return nil
	})
}

func (s *Store) AddPlayHistory(row PlayHistory) (PlayHistory, error) {
	var saved PlayHistory
	err := s.WithWrite(func(d *storeData) error {
		if row.ID == 0 {
			row.ID = s.nextHistoryIDLocked()
		}
		d.PlayHistory = append(d.PlayHistory, row)
		saved = row
		return nil
	})
	return saved, err
}

func (s *Store) DeleteProject(projectID uint) error {
	results, err := s.DeleteProjectsBatch([]uint{projectID})
	if err != nil {
		return err
	}
	if len(results) == 0 || !results[0].Deleted {
		return errors.New("project not found")
	}
	return nil
}

func (s *Store) DeleteProjectsBatch(projectIDs []uint) ([]ProjectDeleteResult, error) {
	if len(projectIDs) == 0 {
		return []ProjectDeleteResult{}, nil
	}
	results := make([]ProjectDeleteResult, len(projectIDs))
	s.mu.Lock()
	defer s.mu.Unlock()

	projectByID := make(map[uint]MidiProject, len(s.data.MidiProjects))
	for _, row := range s.data.MidiProjects {
		projectByID[row.ID] = row
	}

	deleteSet := make(map[uint]struct{}, len(projectIDs))
	for i, projectID := range projectIDs {
		results[i].ProjectID = projectID
		if projectID == 0 {
			results[i].Reason = ProjectDeleteReasonInvalidID
			continue
		}
		if _, exists := deleteSet[projectID]; exists {
			results[i].Reason = ProjectDeleteReasonDuplicate
			continue
		}
		project, ok := projectByID[projectID]
		if !ok {
			results[i].Reason = ProjectDeleteReasonNotFound
			continue
		}
		deleteSet[projectID] = struct{}{}
		results[i].Project = project
		results[i].Deleted = true
	}

	if len(deleteSet) == 0 {
		return results, nil
	}
	s.data.MidiEvents = filterSlice(s.data.MidiEvents, func(row MidiEvent) bool { _, drop := deleteSet[row.ProjectID]; return !drop })
	s.data.PlayHistory = filterSlice(s.data.PlayHistory, func(row PlayHistory) bool { _, drop := deleteSet[row.ProjectID]; return !drop })
	s.data.MidiProfiles = filterSlice(s.data.MidiProfiles, func(row MidiProfile) bool {
		return row.ProjectID == nil || !containsProjectID(deleteSet, *row.ProjectID)
	})
	s.data.MidiProjects = filterSlice(s.data.MidiProjects, func(row MidiProject) bool { _, drop := deleteSet[row.ID]; return !drop })
	s.ensureDefaultsLocked()
	return results, s.persistLocked()
}

func (s *Store) ListEventsByProject(projectID uint) []MidiEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rows := make([]MidiEvent, 0)
	for _, row := range s.data.MidiEvents {
		if row.ProjectID == projectID {
			rows = append(rows, row)
		}
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].StartMs != rows[j].StartMs {
			return rows[i].StartMs < rows[j].StartMs
		}
		return rows[i].ID < rows[j].ID
	})
	return rows
}

func (s *Store) CountEventsByProject(projectID uint) int64 {
	return int64(len(s.ListEventsByProject(projectID)))
}

func (s *Store) CountProfilesByProject(projectID uint) int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var count int64
	for _, row := range s.data.MidiProfiles {
		if row.ProjectID != nil && *row.ProjectID == projectID {
			count++
		}
	}
	return count
}

func (s *Store) CountHistoryByProject(projectID uint) int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var count int64
	for _, row := range s.data.PlayHistory {
		if row.ProjectID == projectID {
			count++
		}
	}
	return count
}

func (s *Store) CountProjects() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return int64(len(s.data.MidiProjects))
}

func (s *Store) CountProjectProfiles(projectID uint) int64 {
	return s.CountProfilesByProject(projectID)
}

func (s *Store) CountGlobalDefaultProfiles() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var count int64
	for _, row := range s.data.MidiProfiles {
		if row.ProjectID == nil && row.KeymapProfileID == DefaultKeymapProfileID {
			count++
		}
	}
	return count
}

func (s *Store) CountKeymapRows(profileID uint) int64 {
	return int64(len(s.ListKeymapProfile(profileID)))
}

func (s *Store) ListKeymapProfile(profileID uint) []Keymap36 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rows := make([]Keymap36, 0)
	for _, row := range s.data.Keymap36 {
		if row.ProfileID == profileID {
			rows = append(rows, row)
		}
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].DisplayOrder != rows[j].DisplayOrder {
			return rows[i].DisplayOrder < rows[j].DisplayOrder
		}
		return rows[i].Lane < rows[j].Lane
	})
	return rows
}

func (s *Store) UpdateKeymapLane(profileID uint, lane int, mutate func(*Keymap36)) error {
	return s.WithWrite(func(d *storeData) error {
		for i := range d.Keymap36 {
			if d.Keymap36[i].ProfileID == profileID && d.Keymap36[i].Lane == lane {
				mutate(&d.Keymap36[i])
				d.Keymap36[i].UpdatedAt = nowMillis()
				return nil
			}
		}
		return errors.New("keymap lane not found")
	})
}

func (s *Store) ClearPlayHistory() error {
	return s.WithWrite(func(d *storeData) error { d.PlayHistory = nil; return nil })
}

func (s *Store) Usage() []TableUsage {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sizes := estimatedJSONSizes(s.data)
	return []TableUsage{
		{TableName: "preferences", RowCount: int64(len(s.data.Preferences)), SizeBytes: sizes["preferences"], Estimated: true},
		{TableName: "app_settings", RowCount: 1, SizeBytes: sizes["app_settings"], Estimated: true},
		{TableName: MidiProjectsTable, RowCount: int64(len(s.data.MidiProjects)), SizeBytes: sizes[MidiProjectsTable], Estimated: true},
		{TableName: MidiEventsTable, RowCount: int64(len(s.data.MidiEvents)), SizeBytes: sizes[MidiEventsTable], Estimated: true},
		{TableName: MidiProfilesTable, RowCount: int64(len(s.data.MidiProfiles)), SizeBytes: sizes[MidiProfilesTable], Estimated: true},
		{TableName: Keymap36Table, RowCount: int64(len(s.data.Keymap36)), SizeBytes: sizes[Keymap36Table], Estimated: true},
		{TableName: PlayHistoryTable, RowCount: int64(len(s.data.PlayHistory)), SizeBytes: sizes[PlayHistoryTable], Estimated: true},
		{TableName: HotkeyBindingsTable, RowCount: int64(len(s.data.HotkeyBindings)), SizeBytes: sizes[HotkeyBindingsTable], Estimated: true},
	}
}

func estimatedJSONSizes(d storeData) map[string]int64 {
	out := map[string]int64{}
	add := func(name string, v any) { data, _ := json.Marshal(v); out[name] = int64(len(data)) }
	add("preferences", d.Preferences)
	add("app_settings", d.AppSettings)
	add(MidiProjectsTable, d.MidiProjects)
	add(MidiEventsTable, d.MidiEvents)
	add(MidiProfilesTable, d.MidiProfiles)
	add(Keymap36Table, d.Keymap36)
	add(PlayHistoryTable, d.PlayHistory)
	add(HotkeyBindingsTable, d.HotkeyBindings)
	return out
}

func nowMillis() int64 { return time.Now().UnixMilli() }

func filterSlice[T any](in []T, keep func(T) bool) []T {
	out := in[:0]
	for _, row := range in {
		if keep(row) {
			out = append(out, row)
		}
	}
	return out
}

func containsProjectID(ids map[uint]struct{}, id uint) bool {
	_, ok := ids[id]
	return ok
}

func sortProjects(rows []MidiProject) {
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].UpdatedAt != rows[j].UpdatedAt {
			return rows[i].UpdatedAt > rows[j].UpdatedAt
		}
		return rows[i].ID > rows[j].ID
	})
}

func findProject(rows []MidiProject, id uint) (MidiProject, bool) {
	for _, row := range rows {
		if row.ID == id {
			return row, true
		}
	}
	return MidiProject{}, false
}

func findProjectByHash(rows []MidiProject, hash string) (MidiProject, bool) {
	for _, row := range rows {
		if row.FileHash == hash {
			return row, true
		}
	}
	return MidiProject{}, false
}

func findProfile(rows []MidiProfile, id uint) (MidiProfile, bool) {
	for _, row := range rows {
		if row.ID == id {
			return row, true
		}
	}
	return MidiProfile{}, false
}

func maxProjectID(rows []MidiProject) uint {
	var max uint
	for _, r := range rows {
		if r.ID > max {
			max = r.ID
		}
	}
	return max
}
func maxEventID(rows []MidiEvent) uint {
	var max uint
	for _, r := range rows {
		if r.ID > max {
			max = r.ID
		}
	}
	return max
}
func maxProfileID(rows []MidiProfile) uint {
	var max uint
	for _, r := range rows {
		if r.ID > max {
			max = r.ID
		}
	}
	return max
}
func maxKeymapID(rows []Keymap36) uint {
	var max uint
	for _, r := range rows {
		if r.ID > max {
			max = r.ID
		}
	}
	return max
}
func maxHistoryID(rows []PlayHistory) uint {
	var max uint
	for _, r := range rows {
		if r.ID > max {
			max = r.ID
		}
	}
	return max
}

func (s *Store) nextProjectIDLocked() uint {
	id := s.data.NextIDs.MidiProject
	s.data.NextIDs.MidiProject++
	if id == 0 {
		return s.nextProjectIDLocked()
	}
	return id
}
func (s *Store) nextEventIDLocked() uint {
	id := s.data.NextIDs.MidiEvent
	s.data.NextIDs.MidiEvent++
	if id == 0 {
		return s.nextEventIDLocked()
	}
	return id
}
func (s *Store) nextProfileIDLocked() uint {
	id := s.data.NextIDs.MidiProfile
	s.data.NextIDs.MidiProfile++
	if id == 0 {
		return s.nextProfileIDLocked()
	}
	return id
}
func (s *Store) nextKeymapIDLocked() uint {
	id := s.data.NextIDs.Keymap36
	s.data.NextIDs.Keymap36++
	if id == 0 {
		return s.nextKeymapIDLocked()
	}
	return id
}
func (s *Store) nextHistoryIDLocked() uint {
	id := s.data.NextIDs.PlayHistory
	s.data.NextIDs.PlayHistory++
	if id == 0 {
		return s.nextHistoryIDLocked()
	}
	return id
}

func (s *Store) profileIDExistsLocked(id uint) bool {
	_, ok := findProfile(s.data.MidiProfiles, id)
	return ok
}
func (s *Store) keymapIDExistsLocked(id uint) bool {
	for _, row := range s.data.Keymap36 {
		if row.ID == id {
			return true
		}
	}
	return false
}

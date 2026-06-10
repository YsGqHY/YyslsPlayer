package storage

import (
	"errors"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Store 是面向业务的持久化门面，背后由 GORM + SQLite 支撑。
// 业务 service 通过 holder.Current().Store 访问，方法签名与历史版本保持兼容。
type Store struct {
	gdb  *gorm.DB
	path string
	mu   sync.Mutex // 仅保护"读改写"复合操作（如默认状态补齐、批量删除）的原子性
}

// ProjectListOptions 控制曲库列表分页与搜索。
type ProjectListOptions struct {
	Limit  int
	Offset int
	Query  string
}

// ProjectImportData 是一次导入要写入的项目与其事件。
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

func nowMillis() int64 { return time.Now().UnixMilli() }

// Path 返回当前数据库文件路径。
func (s *Store) Path() string { return s.path }

// Close 为兼容旧接口保留；真正的连接关闭在 DB.Close 完成。
func (s *Store) Close() error { return nil }

// db 返回底层 GORM 句柄。
func (s *Store) db() *gorm.DB { return s.gdb }

// ===== 行为偏好 preferences =====

func (s *Store) GetPreference(key string) (string, bool) {
	var row Preference
	if err := s.db().Where("key = ?", key).Take(&row).Error; err != nil {
		return "", false
	}
	return row.Value, true
}

func (s *Store) SetPreference(key, value string) error {
	row := Preference{Key: key, Value: value, UpdatedAt: nowMillis()}
	return s.db().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at"}),
	}).Create(&row).Error
}

func (s *Store) ListPreferences() []Preference {
	var rows []Preference
	if err := s.db().Order("key ASC").Find(&rows).Error; err != nil {
		return []Preference{}
	}
	return rows
}

func (s *Store) DeletePreference(key string) error {
	return s.db().Where("key = ?", key).Delete(&Preference{}).Error
}

func (s *Store) ClearPreferences() error {
	return s.db().Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&Preference{}).Error
}

// ===== 应用设置 app_settings =====

func (s *Store) GetAppSettings() AppSettings {
	var row AppSettings
	if err := s.db().Where("id = ?", uint(1)).Take(&row).Error; err != nil {
		return defaultAppSettings()
	}
	return row
}

func (s *Store) UpdateAppSettings(mut func(*AppSettings)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	var row AppSettings
	err := s.db().Where("id = ?", uint(1)).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		row = defaultAppSettings()
	} else if err != nil {
		return err
	}
	mut(&row)
	row.ID = 1
	row.UpdatedAt = nowMillis()
	return s.db().Save(&row).Error
}

// ===== 全局快捷键 hotkey_bindings =====

func (s *Store) CountHotkeyBindings() int64 {
	var n int64
	s.db().Model(&HotkeyBinding{}).Count(&n)
	return n
}

func (s *Store) ListHotkeyBindings() []HotkeyBinding {
	var rows []HotkeyBinding
	if err := s.db().Find(&rows).Error; err != nil {
		return []HotkeyBinding{}
	}
	return rows
}

func (s *Store) SaveHotkeyBinding(row HotkeyBinding) error {
	row.UpdatedAt = nowMillis()
	return s.db().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "action_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"accelerator", "enabled", "updated_at"}),
	}).Create(&row).Error
}

func (s *Store) GetHotkeyBinding(actionID string) (HotkeyBinding, bool) {
	var row HotkeyBinding
	if err := s.db().Where("action_id = ?", actionID).Take(&row).Error; err != nil {
		return HotkeyBinding{}, false
	}
	return row, true
}

func (s *Store) ClearHotkeyBindings() error {
	return s.db().Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&HotkeyBinding{}).Error
}

// ===== 项目 midi_projects：查询 =====

func (s *Store) ListProjects(opts ProjectListOptions) []MidiProject {
	q := s.db().Model(&MidiProject{})
	if query := strings.TrimSpace(opts.Query); query != "" {
		like := "%" + strings.ToLower(query) + "%"
		q = q.Where(
			"LOWER(display_name) LIKE ? OR LOWER(file_name) LIKE ? OR LOWER(file_hash) LIKE ?",
			like, like, like,
		)
	}
	q = q.Order("updated_at DESC").Order("id DESC")
	if opts.Offset > 0 {
		q = q.Offset(opts.Offset)
	}
	if opts.Limit > 0 {
		q = q.Limit(opts.Limit)
	}
	var rows []MidiProject
	if err := q.Find(&rows).Error; err != nil {
		return []MidiProject{}
	}
	return rows
}

func (s *Store) GetProject(id uint) (MidiProject, bool) {
	var row MidiProject
	if err := s.db().Where("id = ?", id).Take(&row).Error; err != nil {
		return MidiProject{}, false
	}
	return row, true
}

func (s *Store) GetProjectByHash(hash string) (MidiProject, bool) {
	if hash == "" {
		return MidiProject{}, false
	}
	var row MidiProject
	if err := s.db().Where("file_hash = ?", hash).Take(&row).Error; err != nil {
		return MidiProject{}, false
	}
	return row, true
}

func (s *Store) ProjectHashIndex() map[string]MidiProject {
	var rows []MidiProject
	if err := s.db().Find(&rows).Error; err != nil {
		return map[string]MidiProject{}
	}
	out := make(map[string]MidiProject, len(rows))
	for _, row := range rows {
		if row.FileHash == "" {
			continue
		}
		out[row.FileHash] = row
	}
	return out
}

func (s *Store) CountProjects() int64 {
	var n int64
	s.db().Model(&MidiProject{}).Count(&n)
	return n
}

// ===== 项目导入：单个 / 批量 =====

func (s *Store) ImportProject(input ProjectImportData) (MidiProject, error) {
	var imported MidiProject
	err := s.db().Transaction(func(tx *gorm.DB) error {
		if input.Project.FileHash != "" {
			var count int64
			tx.Model(&MidiProject{}).Where("file_hash = ?", input.Project.FileHash).Count(&count)
			if count > 0 {
				return errors.New("duplicate project")
			}
		}
		project := input.Project
		project.ID = 0
		now := nowMillis()
		if project.CreatedAt == 0 {
			project.CreatedAt = now
		}
		project.UpdatedAt = now
		if err := tx.Create(&project).Error; err != nil {
			return err
		}
		if err := insertEventsForProject(tx, project.ID, input.Events); err != nil {
			return err
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
	err := s.db().Transaction(func(tx *gorm.DB) error {
		existingByHash := map[string]MidiProject{}
		var existing []MidiProject
		if err := tx.Find(&existing).Error; err != nil {
			return err
		}
		for _, row := range existing {
			if row.FileHash != "" {
				existingByHash[row.FileHash] = row
			}
		}

		now := nowMillis()
		for i, input := range inputs {
			if input.Project.FileHash != "" {
				if prev, ok := existingByHash[input.Project.FileHash]; ok {
					results[i] = ProjectBatchImportResult{Project: prev, Status: ProjectBatchImportStatusSkipped, Reason: ProjectBatchImportReasonDuplicateInLibrary}
					continue
				}
			}
			project := input.Project
			project.ID = 0
			if project.CreatedAt == 0 {
				project.CreatedAt = now
			}
			project.UpdatedAt = now
			if err := tx.Create(&project).Error; err != nil {
				return err
			}
			if err := insertEventsForProject(tx, project.ID, input.Events); err != nil {
				return err
			}
			if project.FileHash != "" {
				existingByHash[project.FileHash] = project
			}
			results[i] = ProjectBatchImportResult{Project: project, Status: ProjectBatchImportStatusImported}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return results, nil
}

// insertEventsForProject 批量写入某项目的标准化事件（重置 ID，绑定 ProjectID）。
func insertEventsForProject(tx *gorm.DB, projectID uint, events []MidiEvent) error {
	if len(events) == 0 {
		return nil
	}
	rows := make([]MidiEvent, len(events))
	for i, ev := range events {
		ev.ID = 0
		ev.ProjectID = projectID
		rows[i] = ev
	}
	return tx.CreateInBatches(rows, 500).Error
}

// ===== 配置档 midi_profiles =====

func (s *Store) ListProjectProfiles(projectID uint) []MidiProfile {
	var rows []MidiProfile
	if err := s.db().Where("project_id = ?", projectID).Order("id ASC").Find(&rows).Error; err != nil {
		return []MidiProfile{}
	}
	return rows
}

func (s *Store) GetProfile(id uint) (MidiProfile, bool) {
	var row MidiProfile
	if err := s.db().Where("id = ?", id).Take(&row).Error; err != nil {
		return MidiProfile{}, false
	}
	return row, true
}

func (s *Store) GetGlobalDefaultProfile() (MidiProfile, bool) {
	var row MidiProfile
	if err := s.db().Where("project_id IS NULL").Order("id ASC").Take(&row).Error; err != nil {
		return MidiProfile{}, false
	}
	return row, true
}

func (s *Store) SaveProfile(row MidiProfile) (MidiProfile, error) {
	now := nowMillis()
	if row.ID == 0 {
		row.CreatedAt = now
		row.UpdatedAt = now
		if err := s.db().Create(&row).Error; err != nil {
			return MidiProfile{}, err
		}
		return row, nil
	}
	if row.CreatedAt == 0 {
		var old MidiProfile
		if err := s.db().Where("id = ?", row.ID).Take(&old).Error; err == nil {
			row.CreatedAt = old.CreatedAt
		}
	}
	row.UpdatedAt = now
	if err := s.db().Save(&row).Error; err != nil {
		return MidiProfile{}, err
	}
	return row, nil
}

// ===== 项目默认档关联 + 保存项目 =====

func (s *Store) UpdateProjectDefaultProfile(projectID uint, profileID uint) error {
	res := s.db().Model(&MidiProject{}).
		Where("id = ?", projectID).
		Updates(map[string]any{"default_profile_id": profileID, "updated_at": nowMillis()})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errors.New("project not found")
	}
	return nil
}

func (s *Store) SaveProject(row MidiProject) (MidiProject, error) {
	now := nowMillis()
	if row.ID == 0 {
		row.CreatedAt = now
		row.UpdatedAt = now
		if err := s.db().Create(&row).Error; err != nil {
			return MidiProject{}, err
		}
		return row, nil
	}
	if row.CreatedAt == 0 {
		var old MidiProject
		if err := s.db().Where("id = ?", row.ID).Take(&old).Error; err == nil {
			row.CreatedAt = old.CreatedAt
		}
	}
	row.UpdatedAt = now
	if err := s.db().Save(&row).Error; err != nil {
		return MidiProject{}, err
	}
	return row, nil
}

// ===== 事件批量写入 + 播放历史 =====

func (s *Store) AddEvents(events []MidiEvent) error {
	if len(events) == 0 {
		return nil
	}
	rows := make([]MidiEvent, len(events))
	for i, ev := range events {
		ev.ID = 0
		rows[i] = ev
	}
	return s.db().CreateInBatches(rows, 500).Error
}

func (s *Store) AddPlayHistory(row PlayHistory) (PlayHistory, error) {
	row.ID = 0
	if err := s.db().Create(&row).Error; err != nil {
		return PlayHistory{}, err
	}
	return row, nil
}

// ===== 项目删除（单个 / 批量，级联清理） =====

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
	s.mu.Lock()
	defer s.mu.Unlock()

	results := make([]ProjectDeleteResult, len(projectIDs))
	err := s.db().Transaction(func(tx *gorm.DB) error {
		projectByID := map[uint]MidiProject{}
		var rows []MidiProject
		if err := tx.Find(&rows).Error; err != nil {
			return err
		}
		for _, row := range rows {
			projectByID[row.ID] = row
		}

		deleteSet := map[uint]struct{}{}
		for i, projectID := range projectIDs {
			results[i].ProjectID = projectID
			if projectID == 0 {
				results[i].Reason = ProjectDeleteReasonInvalidID
				continue
			}
			if _, dup := deleteSet[projectID]; dup {
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
			return nil
		}
		ids := make([]uint, 0, len(deleteSet))
		for id := range deleteSet {
			ids = append(ids, id)
		}
		if err := tx.Where("project_id IN ?", ids).Delete(&MidiEvent{}).Error; err != nil {
			return err
		}
		if err := tx.Where("project_id IN ?", ids).Delete(&PlayHistory{}).Error; err != nil {
			return err
		}
		if err := tx.Where("project_id IN ?", ids).Delete(&MidiProfile{}).Error; err != nil {
			return err
		}
		if err := tx.Where("id IN ?", ids).Delete(&MidiProject{}).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return results, nil
}

// ===== 事件查询 + 各类计数 =====

func (s *Store) ListEventsByProject(projectID uint) []MidiEvent {
	var rows []MidiEvent
	if err := s.db().Where("project_id = ?", projectID).
		Order("start_ms ASC").Order("id ASC").
		Find(&rows).Error; err != nil {
		return []MidiEvent{}
	}
	return rows
}

func (s *Store) CountEventsByProject(projectID uint) int64 {
	var n int64
	s.db().Model(&MidiEvent{}).Where("project_id = ?", projectID).Count(&n)
	return n
}

func (s *Store) CountProfilesByProject(projectID uint) int64 {
	var n int64
	s.db().Model(&MidiProfile{}).Where("project_id = ?", projectID).Count(&n)
	return n
}

func (s *Store) CountProjectProfiles(projectID uint) int64 {
	return s.CountProfilesByProject(projectID)
}

func (s *Store) CountHistoryByProject(projectID uint) int64 {
	var n int64
	s.db().Model(&PlayHistory{}).Where("project_id = ?", projectID).Count(&n)
	return n
}

func (s *Store) CountGlobalDefaultProfiles() int64 {
	var n int64
	s.db().Model(&MidiProfile{}).
		Where("project_id IS NULL AND keymap_profile_id = ?", DefaultKeymapProfileID).
		Count(&n)
	return n
}

func (s *Store) CountKeymapRows(profileID uint) int64 {
	var n int64
	s.db().Model(&Keymap36{}).Where("profile_id = ?", profileID).Count(&n)
	return n
}

// ===== 36 键映射 keymap_36 =====

func (s *Store) ListKeymapProfile(profileID uint) []Keymap36 {
	var rows []Keymap36
	if err := s.db().Where("profile_id = ?", profileID).
		Order("display_order ASC").Order("lane ASC").
		Find(&rows).Error; err != nil {
		return []Keymap36{}
	}
	return rows
}

func (s *Store) UpdateKeymapLane(profileID uint, lane int, mutate func(*Keymap36)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.db().Transaction(func(tx *gorm.DB) error {
		var row Keymap36
		err := tx.Where("profile_id = ? AND lane = ?", profileID, lane).Take(&row).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("keymap lane not found")
		}
		if err != nil {
			return err
		}
		mutate(&row)
		row.ProfileID = profileID
		row.Lane = lane
		row.UpdatedAt = nowMillis()
		return tx.Save(&row).Error
	})
}

// ===== 清空类 =====

func (s *Store) ClearPlayHistory() error {
	return s.db().Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&PlayHistory{}).Error
}

//go:build completion

package storage

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

// ===== 转录任务 CRUD =====

// CreateTranscriptionTask 写入一条新转录任务。
func (s *Store) CreateTranscriptionTask(task *TranscriptionTask) error {
	now := nowMillis()
	task.ID = 0
	task.CreatedAt = now
	task.UpdatedAt = now
	return s.db().Create(task).Error
}

// UpdateTranscriptionTask 更新转录任务（按 ID 匹配）。
// 只更新非零值字段，并自动维护 UpdatedAt。
func (s *Store) UpdateTranscriptionTask(task *TranscriptionTask) error {
	task.UpdatedAt = nowMillis()
	return s.db().Save(task).Error
}

// GetTranscriptionTask 按 ID 获取一条转录任务。
func (s *Store) GetTranscriptionTask(id uint) (TranscriptionTask, bool) {
	var row TranscriptionTask
	if err := s.db().Where("id = ?", id).Take(&row).Error; err != nil {
		return TranscriptionTask{}, false
	}
	return row, true
}

// ListTranscriptionTasks 分页列出转录任务，按创建时间降序。
func (s *Store) ListTranscriptionTasks(limit, offset int) []TranscriptionTask {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	var rows []TranscriptionTask
	if err := s.db().Order("created_at DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return []TranscriptionTask{}
	}
	return rows
}

// DeleteTranscriptionTask 删除转录任务并级联清理关联数据。
func (s *Store) DeleteTranscriptionTask(id uint) error {
	return s.db().Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("task_id = ?", id).Delete(&TranscriptionNote{}).Error; err != nil {
			return err
		}
		if err := tx.Where("task_id = ?", id).Delete(&TranscriptionAnalysis{}).Error; err != nil {
			return err
		}
		res := tx.Where("id = ?", id).Delete(&TranscriptionTask{})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return errors.New("task not found")
		}
		return nil
	})
}

// CountTranscriptionTasks 返回转录任务总数。
func (s *Store) CountTranscriptionTasks() int64 {
	var n int64
	s.db().Model(&TranscriptionTask{}).Count(&n)
	return n
}

// ===== 转录 Note 批量写入与查询 =====

// BatchCreateNotes 批量写入转录音符（一次性事务，避免逐条落盘）。
func (s *Store) BatchCreateNotes(notes []TranscriptionNote) error {
	if len(notes) == 0 {
		return nil
	}
	rows := make([]TranscriptionNote, len(notes))
	for i, n := range notes {
		n.ID = 0
		rows[i] = n
	}
	return s.db().CreateInBatches(rows, 500).Error
}

// ListNotesByTask 按 TaskID 获取所有关联音符。
func (s *Store) ListNotesByTask(taskID uint) []TranscriptionNote {
	var rows []TranscriptionNote
	if err := s.db().Where("task_id = ?", taskID).Order("start_ms ASC").Find(&rows).Error; err != nil {
		return []TranscriptionNote{}
	}
	return rows
}

// DeleteNotesByTask 删除某任务的所有关联音符。
func (s *Store) DeleteNotesByTask(taskID uint) error {
	return s.db().Where("task_id = ?", taskID).Delete(&TranscriptionNote{}).Error
}

// ===== 转录 Analysis =====

// SaveAnalysis 写入一条转录分析结果。
func (s *Store) SaveAnalysis(analysis *TranscriptionAnalysis) error {
	analysis.ID = 0
	analysis.CreatedAt = time.Now().UnixMilli()
	return s.db().Create(analysis).Error
}

// ListAnalysisByTask 按 TaskID 获取所有分析记录。
func (s *Store) ListAnalysisByTask(taskID uint) []TranscriptionAnalysis {
	var rows []TranscriptionAnalysis
	if err := s.db().Where("task_id = ?", taskID).Order("id ASC").Find(&rows).Error; err != nil {
		return []TranscriptionAnalysis{}
	}
	return rows
}

// DeleteAnalysisByTask 删除某任务的所有分析记录。
func (s *Store) DeleteAnalysisByTask(taskID uint) error {
	return s.db().Where("task_id = ?", taskID).Delete(&TranscriptionAnalysis{}).Error
}

// ===== 转录 Config =====

// GetTranscriptionConfig 返回默认转录配置（ID=1），不存在则返回零值。
func (s *Store) GetTranscriptionConfig() (TranscriptionConfig, error) {
	var row TranscriptionConfig
	err := s.db().Where("id = ?", uint(1)).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return defaultTranscriptionConfig(), nil
	}
	if err != nil {
		return TranscriptionConfig{}, err
	}
	return row, nil
}

// UpdateTranscriptionConfig 按 mut 回调更新默认转录配置。
func (s *Store) UpdateTranscriptionConfig(mut func(*TranscriptionConfig)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	var row TranscriptionConfig
	err := s.db().Where("id = ?", uint(1)).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		row = defaultTranscriptionConfig()
	} else if err != nil {
		return err
	}
	mut(&row)
	row.ID = 1
	row.UpdatedAt = nowMillis()
	return s.db().Save(&row).Error
}

// defaultTranscriptionConfig 返回转录配置默认值。
func defaultTranscriptionConfig() TranscriptionConfig {
	return TranscriptionConfig{
		ID:                  1,
		Mode:                "melody",
		MinConfidence:       0.55,
		MinDurationMs:       60,
		MergeGapMs:          40,
		Quantize:            "light",
		MaxPolyphony:        2,
		TargetBaseNote:      48,
		TargetLaneCount:     36,
		OutOfRangePolicy:    "drop",
		PreferMelodyRegister: true,
	}
}

// ===== 启动恢复：修复遗留 running / cancelling 状态 =====

// RecoverTranscriptionTasks 将遗留的 running / cancelling 任务标记为 failed。
// 应用启动时调用，确保不存在永久卡住的任务状态。
func (s *Store) RecoverTranscriptionTasks() error {
	now := nowMillis()
	msg := "task interrupted by application restart"
	return s.db().Transaction(func(tx *gorm.DB) error {
		// 恢复 running → failed
		if err := tx.Model(&TranscriptionTask{}).
			Where("status IN ?", []string{"running", "cancelling"}).
			Updates(map[string]any{
				"status":       "failed",
				"error_code":   "task.interrupted",
				"error_message": &msg,
				"finished_at":  &now,
				"updated_at":   now,
			}).Error; err != nil {
			return err
		}
		// queued 任务重置为新的 queued 状态（保持排队）
		// 不做特殊处理，下次调度会正常处理
		return nil
	})
}

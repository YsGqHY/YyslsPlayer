//go:build completion

package storage

import (
	"errors"

	"gorm.io/gorm"
)

// MacroDetail groups one macro profile with its ordered linear steps.
type MacroDetail struct {
	Profile MacroProfile
	Steps   []MacroStep
}

func (s *Store) ListMacroProfiles() []MacroProfile {
	var rows []MacroProfile
	if err := s.db().Order("updated_at DESC").Order("id DESC").Find(&rows).Error; err != nil {
		return []MacroProfile{}
	}
	return rows
}

func (s *Store) ListEnabledMacroProfiles() []MacroProfile {
	var rows []MacroProfile
	if err := s.db().Where("enabled = ?", true).
		Order("updated_at DESC").Order("id DESC").
		Find(&rows).Error; err != nil {
		return []MacroProfile{}
	}
	return rows
}

func (s *Store) GetMacroProfile(id uint) (MacroProfile, bool) {
	if id == 0 {
		return MacroProfile{}, false
	}
	var row MacroProfile
	if err := s.db().Where("id = ?", id).Take(&row).Error; err != nil {
		return MacroProfile{}, false
	}
	return row, true
}

func (s *Store) GetMacroDetail(id uint) (MacroDetail, bool) {
	profile, ok := s.GetMacroProfile(id)
	if !ok {
		return MacroDetail{}, false
	}
	return MacroDetail{Profile: profile, Steps: s.ListMacroSteps(id)}, true
}

func (s *Store) ListMacroSteps(macroID uint) []MacroStep {
	if macroID == 0 {
		return []MacroStep{}
	}
	var rows []MacroStep
	if err := s.db().Where("macro_id = ?", macroID).
		Order("order_index ASC").Order("id ASC").
		Find(&rows).Error; err != nil {
		return []MacroStep{}
	}
	return rows
}

func (s *Store) SaveMacroProfile(row MacroProfile) (MacroProfile, error) {
	now := nowMillis()
	if row.ID == 0 {
		row.CreatedAt = now
		row.UpdatedAt = now
		if err := s.db().Create(&row).Error; err != nil {
			return MacroProfile{}, err
		}
		return row, nil
	}
	var old MacroProfile
	err := s.db().Where("id = ?", row.ID).Take(&old).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return MacroProfile{}, gorm.ErrRecordNotFound
	}
	if err != nil {
		return MacroProfile{}, err
	}
	row.CreatedAt = old.CreatedAt
	row.UpdatedAt = now
	if err := s.db().Save(&row).Error; err != nil {
		return MacroProfile{}, err
	}
	return row, nil
}

func (s *Store) SaveMacroDetail(input MacroDetail) (MacroDetail, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var out MacroDetail
	err := s.db().Transaction(func(tx *gorm.DB) error {
		now := nowMillis()
		profile := input.Profile
		if profile.ID == 0 {
			profile.CreatedAt = now
			profile.UpdatedAt = now
			if err := tx.Create(&profile).Error; err != nil {
				return err
			}
		} else {
			var old MacroProfile
			err := tx.Where("id = ?", profile.ID).Take(&old).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return gorm.ErrRecordNotFound
			}
			if err != nil {
				return err
			}
			profile.CreatedAt = old.CreatedAt
			profile.UpdatedAt = now
			if err := tx.Save(&profile).Error; err != nil {
				return err
			}
			if err := tx.Where("macro_id = ?", profile.ID).Delete(&MacroStep{}).Error; err != nil {
				return err
			}
		}

		steps := make([]MacroStep, 0, len(input.Steps))
		for i, step := range input.Steps {
			step.ID = 0
			step.MacroID = profile.ID
			step.OrderIndex = i
			step.CreatedAt = now
			step.UpdatedAt = now
			steps = append(steps, step)
		}
		if len(steps) > 0 {
			if err := tx.CreateInBatches(&steps, 200).Error; err != nil {
				return err
			}
		}
		out = MacroDetail{Profile: profile, Steps: steps}
		return nil
	})
	return out, err
}

func (s *Store) DeleteMacro(id uint) error {
	if id == 0 {
		return gorm.ErrRecordNotFound
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.db().Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("macro_id = ?", id).Delete(&MacroStep{}).Error; err != nil {
			return err
		}
		res := tx.Where("id = ?", id).Delete(&MacroProfile{})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}

func (s *Store) UpdateMacroProfile(id uint, mutate func(*MacroProfile)) (MacroProfile, error) {
	if id == 0 {
		return MacroProfile{}, gorm.ErrRecordNotFound
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var row MacroProfile
	err := s.db().Where("id = ?", id).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return MacroProfile{}, gorm.ErrRecordNotFound
	}
	if err != nil {
		return MacroProfile{}, err
	}
	mutate(&row)
	row.UpdatedAt = nowMillis()
	if err := s.db().Save(&row).Error; err != nil {
		return MacroProfile{}, err
	}
	return row, nil
}

func (s *Store) CountMacroProfiles() int64 {
	var n int64
	s.db().Model(&MacroProfile{}).Count(&n)
	return n
}

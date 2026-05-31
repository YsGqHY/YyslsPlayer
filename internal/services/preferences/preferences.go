// Package preferences 提供"行为偏好"键值存储。
package preferences

import (
	"context"
	"errors"
	"strings"

	"YyslsPlayer/internal/storage"
)

type Service struct {
	holder *storage.Holder
}

func New(holder *storage.Holder) *Service {
	return &Service{holder: holder}
}

type Entry struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func (s *Service) store() *storage.Store {
	return s.holder.Current().Store
}

func (s *Service) Get(_ context.Context, key string) (string, error) {
	if strings.TrimSpace(key) == "" {
		return "", errors.New("key required")
	}
	value, _ := s.store().GetPreference(key)
	return value, nil
}

func (s *Service) Set(_ context.Context, key, value string) error {
	if strings.TrimSpace(key) == "" {
		return errors.New("key required")
	}
	return s.store().SetPreference(key, value)
}

func (s *Service) List(_ context.Context) ([]Entry, error) {
	rows := s.store().ListPreferences()
	out := make([]Entry, 0, len(rows))
	for _, r := range rows {
		out = append(out, Entry{Key: r.Key, Value: r.Value})
	}
	return out, nil
}

func (s *Service) Delete(_ context.Context, key string) error {
	return s.store().DeletePreference(key)
}

func (s *Service) Reset(_ context.Context) error {
	return s.store().ClearPreferences()
}

// Package appsettings 提供"应用设置"单行配置的读写。
package appsettings

import (
	"context"

	"YyslsPlayer/internal/storage"
)

type Service struct {
	holder *storage.Holder
}

func New(holder *storage.Holder) *Service {
	return &Service{holder: holder}
}

type Snapshot struct {
	ThemeChoice  string `json:"themeChoice"`
	CustomTheme  string `json:"customTheme"`
	LocaleChoice string `json:"localeChoice"`
}

func (s *Service) store() *storage.Store {
	return s.holder.Current().Store
}

func (s *Service) Get(_ context.Context) (Snapshot, error) {
	row := s.store().GetAppSettings()
	return Snapshot{ThemeChoice: row.ThemeChoice, CustomTheme: row.CustomTheme, LocaleChoice: row.LocaleChoice}, nil
}

func (s *Service) SetThemeChoice(_ context.Context, choice string) error {
	return s.store().UpdateAppSettings(func(row *storage.AppSettings) { row.ThemeChoice = choice })
}

func (s *Service) SetCustomTheme(_ context.Context, json string) error {
	return s.store().UpdateAppSettings(func(row *storage.AppSettings) { row.CustomTheme = json })
}

func (s *Service) ResetCustomTheme(ctx context.Context) error {
	return s.SetCustomTheme(ctx, "")
}

func (s *Service) SetLocaleChoice(_ context.Context, choice string) error {
	return s.store().UpdateAppSettings(func(row *storage.AppSettings) { row.LocaleChoice = choice })
}

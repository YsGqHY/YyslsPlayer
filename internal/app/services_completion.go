//go:build completion

package app

import (
	"YyslsPlayer/internal/services/hotkey"
	"YyslsPlayer/internal/services/keysim"
	"YyslsPlayer/internal/services/macro"
	"YyslsPlayer/internal/services/player"
	"YyslsPlayer/internal/services/transcription"
	"YyslsPlayer/internal/storage"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// TranscriptionLifecycle 转录服务的生命周期接口。
type TranscriptionLifecycle interface {
	SetApp(*application.App)
	AttachEmitter(func(name string, payload any))
	Start()
	Shutdown()
}

// MacroLifecycle 是 completion 宏服务的生命周期接口。
type MacroLifecycle interface {
	AttachEmitter(func(name string, payload any))
	Start()
	Stop()
}

// registerCompletionServices 将 completion 版本专属服务追加到 services 列表。
// lite 版本编译时此函数不存在（由 services_lite.go 提供空实现）。
func registerCompletionServices(services []application.Service, holder *storage.Holder, driver keysim.Driver, hotkeySvc *hotkey.Service, playerSvc *player.Service) ([]application.Service, TranscriptionLifecycle, MacroLifecycle) {
	transcriptionSvc := transcription.New(holder, nil)
	macroSvc := macro.New(holder, keysim.New(driver), hotkeySvc, playerSvc)
	return append(services,
		application.NewService(transcriptionSvc),
		application.NewService(macroSvc),
	), transcriptionSvc, macroSvc
}

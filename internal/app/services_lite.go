//go:build !completion

package app

import (
	"YyslsPlayer/internal/services/hotkey"
	"YyslsPlayer/internal/services/keysim"
	"YyslsPlayer/internal/services/player"
	"YyslsPlayer/internal/storage"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// TranscriptionLifecycle 转录服务的生命周期接口，completion 和 lite 版本都实现。
type TranscriptionLifecycle interface {
	SetApp(*application.App)
	AttachEmitter(func(name string, payload any))
	Start()
	Shutdown()
}

// MacroLifecycle 是 lite / completion 共享的宏生命周期接口。
type MacroLifecycle interface {
	AttachEmitter(func(name string, payload any))
	Start()
	Stop()
}

// registerCompletionServices 是 lite 版本的空实现。
func registerCompletionServices(services []application.Service, holder *storage.Holder, driver keysim.Driver, hotkeySvc *hotkey.Service, playerSvc *player.Service) ([]application.Service, TranscriptionLifecycle, MacroLifecycle) {
	return services, nil, nil
}

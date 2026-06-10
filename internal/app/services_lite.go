//go:build !completion

package app

import (
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

// registerCompletionServices 是 lite 版本的空实现。
func registerCompletionServices(services []application.Service, holder *storage.Holder) ([]application.Service, TranscriptionLifecycle) {
	return services, nil
}

//go:build completion

package app

import (
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

// registerCompletionServices 将 completion 版本专属服务追加到 services 列表。
// lite 版本编译时此函数不存在（由 services_lite.go 提供空实现）。
func registerCompletionServices(services []application.Service, holder *storage.Holder) ([]application.Service, TranscriptionLifecycle) {
	svc := transcription.New(holder, nil)
	return append(services,
		application.NewService(svc),
	), svc
}

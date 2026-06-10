//go:build completion

package events

import (
	"YyslsPlayer/internal/services/transcription"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// registerCompletionEvents 注册 completion 版本专属的 Wails 事件类型。
// lite 版本编译时此函数为空。
func registerCompletionEvents() {
	application.RegisterEvent[transcription.TranscriptionProgress](transcription.EventTaskProgress)
	application.RegisterEvent[transcription.TranscriptionResult](transcription.EventTaskCompleted)
	application.RegisterEvent[transcription.TranscriptionError](transcription.EventTaskFailed)
	application.RegisterEvent[transcription.TranscriptionProgress](transcription.EventTaskCancelled)
}

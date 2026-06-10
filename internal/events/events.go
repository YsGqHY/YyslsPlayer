package events

import (
	"YyslsPlayer/internal/services/hotkey"
	"YyslsPlayer/internal/services/midi"
	"YyslsPlayer/internal/services/player"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// Register 在 application 初始化前注册所有强类型事件，
// Wails 的 binding 生成器会据此为前端生成类型化 API。
func Register() {
	application.RegisterEvent[player.PlayerStateDTO](player.EventState)
	application.RegisterEvent[player.PlayerPositionDTO](player.EventPosition)
	application.RegisterEvent[player.PlayerErrorDTO](player.EventError)
	application.RegisterEvent[hotkey.TriggeredDTO](hotkey.EventTriggered)
	application.RegisterEvent[midi.FilesDroppedDTO](midi.EventFilesDropped)

	registerCompletionEvents()
}

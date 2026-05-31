//go:build production

package app

import (
	"log/slog"

	"YyslsPlayer/internal/utils/logx"
)

func defaultLogConfig() logx.Config {
	return logx.Config{
		Level:          slog.LevelInfo,
		ConsoleEnabled: false,
	}
}

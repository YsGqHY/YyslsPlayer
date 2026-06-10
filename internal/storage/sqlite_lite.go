//go:build !completion

package storage

import "gorm.io/gorm"

// repairCompletionColumns 是 lite 版本的 no-op。completion 专属表的修复见 sqlite_completion.go。
func repairCompletionColumns(db *gorm.DB) {}

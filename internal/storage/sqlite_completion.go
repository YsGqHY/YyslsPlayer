//go:build completion

package storage

import (
	"fmt"

	"gorm.io/gorm"
)

// repairCompletionColumns 删除 completion 版本专属表的旧列。
// 早期 TranscriptionTask 模型含 Audio{Path} 嵌入字段（列名 audio_path，NOT NULL），
// 重构为 SourcePath 后 AutoMigrate 不会删除旧列，导致插入新记录时 NOT NULL 约束失败。
// ALTER TABLE DROP COLUMN 在表/列不存在时静默失败，不影响正常启动。
func repairCompletionColumns(db *gorm.DB) {
	_ = db.Exec(fmt.Sprintf(
		"ALTER TABLE %s DROP COLUMN audio_path",
		TranscriptionTasksTable,
	)).Error
}

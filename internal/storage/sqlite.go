package storage

import (
	"fmt"

	glebarezsqlite "github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// pragmaSet 在打开连接后立即执行，决定 SQLite 的并发与性能基线。
//
// 关键项说明：
//   - journal_mode=WAL：读不阻塞写，多读 + 单写并行的核心
//   - synchronous=NORMAL：WAL 下安全，比 FULL 快数倍（电源故障下最多丢最近事务）
//   - busy_timeout=5000：写冲突时等 5s 而非立即 SQLITE_BUSY
//   - cache_size=-64000：64MB 缓存（负数表示 KB）
//   - temp_store=MEMORY：临时表 / 排序中间结果放内存
//   - foreign_keys=ON：开启外键约束（默认是关的，反直觉）
//   - wal_autocheckpoint=1000：每 1000 帧自动 checkpoint
var pragmaSet = []string{
	`PRAGMA journal_mode = WAL;`,
	`PRAGMA synchronous = NORMAL;`,
	`PRAGMA busy_timeout = 5000;`,
	`PRAGMA cache_size = -64000;`,
	`PRAGMA temp_store = MEMORY;`,
	`PRAGMA foreign_keys = ON;`,
	`PRAGMA wal_autocheckpoint = 1000;`,
}

// openGormSQLite 用纯 Go 的 modernc 驱动（glebarez 适配）打开 SQLite 数据库。
//
// 设计取舍：
//   - 纯 Go 驱动，CGO_ENABLED=0 即可跨平台构建，无需 C 编译器
//   - 单写连接（MaxOpenConns=1）+ WAL：把写串行交给连接池，避免 SQLITE_BUSY
//   - 关闭 GORM 默认 logger，业务日志统一走 logx
func openGormSQLite(path string) (*gorm.DB, error) {
	gdb, err := gorm.Open(glebarezsqlite.Open(path), &gorm.Config{
		Logger:                 logger.Default.LogMode(logger.Silent),
		SkipDefaultTransaction: true,
	})
	if err != nil {
		return nil, fmt.Errorf("storage: open sqlite: %w", err)
	}

	if err := applyPragmas(gdb); err != nil {
		if sqlDB, dbErr := gdb.DB(); dbErr == nil {
			_ = sqlDB.Close()
		}
		return nil, err
	}

	sqlDB, err := gdb.DB()
	if err != nil {
		return nil, fmt.Errorf("storage: access sql.DB: %w", err)
	}
	// 单连接：写串行靠连接池排队，PRAGMA（busy_timeout 等连接级设置）也只需配置一次。
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetConnMaxLifetime(0)

	return gdb, nil
}

// applyPragmas 在已打开的 *gorm.DB 上批量执行 PRAGMA。
// 注意：journal_mode 是连接级 → 数据库级粘性切换；后续新连接也会沿用 WAL 模式。
func applyPragmas(db *gorm.DB) error {
	for _, sql := range pragmaSet {
		if err := db.Exec(sql).Error; err != nil {
			return fmt.Errorf("storage: apply pragma %q: %w", sql, err)
		}
	}
	return nil
}

// migrateModels 跑 AutoMigrate 建表 / 加列 / 加索引（零迁移文件）。
func migrateModels(db *gorm.DB) error {
	models := make([]any, 0, len(AllModels))
	for _, desc := range AllModels {
		models = append(models, desc.Model)
	}
	if err := db.AutoMigrate(models...); err != nil {
		return fmt.Errorf("storage: auto migrate: %w", err)
	}
	return nil
}

// repairStaleColumns 删除 AutoMigrate 无法自动清理的旧列（通用表，lite/completion 共用）。
// GORM AutoMigrate 只加不删；当模型重构rename/删除字段时，旧列仍保留且NOT NULL约束
// 会导致新记录插入失败。此处处理已知的旧列迁移。
// completion 专属表的修复见 sqlite_completion.go。
func repairStaleColumns(db *gorm.DB) {
	// 当前无已知的通用旧列。
	_ = db
}

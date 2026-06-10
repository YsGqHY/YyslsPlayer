package storage

import (
	"errors"
	"fmt"
	"path/filepath"
	"sync"

	"gorm.io/gorm"
)

// DB 是上层使用的持久化句柄。
//
// Store 是面向业务的方法门面（背后是 GORM）；GORM 暴露给需要直接用 ORM 的
// 新业务（例如后续 1.1.0 转录领域）。Path 是当前数据库文件绝对路径。
type DB struct {
	Store *Store
	GORM  *gorm.DB
	Path  string
}

var openMu sync.Mutex

// Open 打开（或新建）位于 path 的 SQLite 数据库：
//  1. 确保目录存在
//  2. 用纯 Go modernc 驱动 + GORM 打开，应用 PRAGMA
//  3. AutoMigrate 建表
//  4. 补齐默认 MIDI profile / keymap / app settings
func Open(path string) (*DB, error) {
	openMu.Lock()
	defer openMu.Unlock()

	if path == "" {
		return nil, errors.New("storage: empty data path")
	}
	if err := ensureDir(filepath.Dir(path)); err != nil {
		return nil, fmt.Errorf("storage: ensure dir: %w", err)
	}

	gdb, err := openGormSQLite(path)
	if err != nil {
		return nil, err
	}
	if err := migrateModels(gdb); err != nil {
		closeGorm(gdb)
		return nil, err
	}
	repairStaleColumns(gdb)
	repairCompletionColumns(gdb)

	store := &Store{gdb: gdb, path: path}
	if err := store.ensureDefaults(); err != nil {
		closeGorm(gdb)
		return nil, err
	}

	return &DB{Store: store, GORM: gdb, Path: path}, nil
}

func MustOpen(path string) *DB {
	db, err := Open(path)
	if err != nil {
		panic(err)
	}
	return db
}

// Close 在关闭前对 WAL 做一次 TRUNCATE checkpoint，把 -wal/-shm 落回主库，
// 这样切换路径时复制单个 .db 文件即可拿到完整数据。
func (d *DB) Close() error {
	if d == nil || d.GORM == nil {
		return nil
	}
	d.GORM.Exec(`PRAGMA wal_checkpoint(TRUNCATE);`)
	return closeGorm(d.GORM)
}

func closeGorm(gdb *gorm.DB) error {
	if gdb == nil {
		return nil
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		return nil
	}
	return sqlDB.Close()
}

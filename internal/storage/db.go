package storage

import (
	"errors"
	"fmt"
	"path/filepath"
	"sync"
)

// DB 是上层使用的 JSON 存储句柄。
type DB struct {
	Store *Store
	Path  string
}

var openMu sync.Mutex

// Open 打开（或新建）位于 path 的 JSON 数据文件，并补齐默认状态。
func Open(path string) (*DB, error) {
	openMu.Lock()
	defer openMu.Unlock()

	if path == "" {
		return nil, errors.New("storage: empty data path")
	}
	if err := ensureDir(filepath.Dir(path)); err != nil {
		return nil, fmt.Errorf("storage: ensure dir: %w", err)
	}

	store, err := OpenStore(path)
	if err != nil {
		return nil, err
	}
	return &DB{Store: store, Path: path}, nil
}

func MustOpen(path string) *DB {
	db, err := Open(path)
	if err != nil {
		panic(err)
	}
	return db
}

func (d *DB) Close() error {
	if d == nil || d.Store == nil {
		return nil
	}
	return d.Store.Close()
}

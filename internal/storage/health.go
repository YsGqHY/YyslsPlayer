package storage

import (
	"errors"
	"os"
)

// IntegrityCheck 运行 SQLite 的 PRAGMA integrity_check。
// 结果为 "ok" 表示数据库结构完好；否则返回首条错误信息。
func IntegrityCheck(s *Store) error {
	if s == nil || s.gdb == nil {
		return nil
	}
	var result string
	if err := s.gdb.Raw(`PRAGMA integrity_check;`).Scan(&result).Error; err != nil {
		return err
	}
	if result != "" && result != "ok" {
		return errors.New("storage: integrity check failed: " + result)
	}
	return nil
}

// FileSize 返回数据库文件占用，用于设置页展示当前占用空间。
// 仅统计主库文件；WAL 在关闭时已 checkpoint 回主库。
func FileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	return info.Size(), nil
}

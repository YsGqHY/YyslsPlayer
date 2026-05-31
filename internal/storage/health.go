package storage

import "os"

// IntegrityCheck is kept for API compatibility. JSON decoding happens during Open.
func IntegrityCheck(_ *Store) error {
	return nil
}

// FileSize 返回 JSON 数据文件占用，用于设置页展示当前占用空间。
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

//go:build completion

package transcription

import (
	"fmt"
	"os"
	"path/filepath"

	"YyslsPlayer/internal/utils/filex"
)

// taskWorkDir 返回指定任务的工作目录绝对路径。
//
// 布局：<应用数据目录>/transcriptions/<task-id>/
func taskWorkDir(baseDataDir string, taskID uint) string {
	return filepath.Join(baseDataDir, "transcriptions", fmt.Sprintf("%d", taskID))
}

// ensureTaskWorkDir 确保任务工作目录存在。
func ensureTaskWorkDir(baseDataDir string, taskID uint) (string, error) {
	dir := taskWorkDir(baseDataDir, taskID)
	if err := filex.EnsureDir(dir); err != nil {
		return "", fmt.Errorf("create task work dir: %w", err)
	}
	return dir, nil
}

// cleanupTaskWorkDir 递归删除任务工作目录。
func cleanupTaskWorkDir(baseDataDir string, taskID uint) error {
	dir := taskWorkDir(baseDataDir, taskID)
	if err := os.RemoveAll(dir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("cleanup task work dir: %w", err)
	}
	return nil
}

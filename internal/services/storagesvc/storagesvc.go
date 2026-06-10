// Package storagesvc 暴露"数据存储"管理：路径查看、切换、统计。
//
// 关键约束：切换路径会改写整个进程持有的 *storage.DB 指针。
// 业务 service 通过 *storage.Holder 拿当前活跃存储，避免持有过期句柄。
package storagesvc

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"YyslsPlayer/internal/storage"
)

// Service 暴露给前端：路径管理 + 统计。
type Service struct {
	holder *storage.Holder
	cfg    *storage.ConfigManager
	mu     sync.Mutex // 切换路径串行
}

func New(holder *storage.Holder, cfg *storage.ConfigManager) *Service {
	return &Service{holder: holder, cfg: cfg}
}

// Stats 设置页用于展示当前路径与占用空间。
type Stats struct {
	Path        string `json:"path"`
	IsCustom    bool   `json:"isCustom"`
	DefaultPath string `json:"defaultPath"`
	SizeBytes   int64  `json:"sizeBytes"`
}

// TableInfo 单张表的展示元数据 + 实际占用。
// 元数据来自 storage.AllModels（labelKey / clearable），用量按行数占比估算得出。
type TableInfo struct {
	Name      string `json:"name"`      // 数据集合名（如 'preferences'）
	LabelKey  string `json:"labelKey"`  // i18n key 后缀（前端拼 settings.database.tables.<key>.label）
	Clearable bool   `json:"clearable"` // 是否允许 ClearTable
	RowCount  int64  `json:"rowCount"`
	SizeBytes int64  `json:"sizeBytes"`
	Estimated bool   `json:"estimated"` // SQLite 无每表精确字节数，按文件大小行数占比估算
}

// TableStats 整体快照。Total 是所有表的占用之和；
// 与 Stats.SizeBytes 不同 —— 后者是整个数据库文件，前者是各表按行数占比的估算和。
type TableStats struct {
	TotalBytes int64       `json:"totalBytes"`
	Tables     []TableInfo `json:"tables"`
}

// GetStats 拉取当前数据存储信息。
func (s *Service) GetStats(_ context.Context) (Stats, error) {
	def, err := storage.DefaultDBPath()
	if err != nil {
		return Stats{}, err
	}
	cfg := s.cfg.Get()
	cur := s.holder.Current().Path

	size, _ := storage.FileSize(cur) // 失败时给 0，不阻断 UI

	return Stats{
		Path:        cur,
		IsCustom:    cfg.DataPath != "",
		DefaultPath: def,
		SizeBytes:   size,
	}, nil
}

// GetTableStats 收集每类已注册数据的行数 + 估算占用。
//
// SQLite 没有便捷的每表精确字节统计；这里按各表行数占总文件大小的比例估算（前端会标注"估算"）。
func (s *Service) GetTableStats(ctx context.Context) (TableStats, error) {
	_ = ctx
	descs := storage.AllModels
	usage := s.holder.Current().Store.Usage()

	// 把 usage 与 descriptor 对齐，组装 TableInfo
	usageByName := make(map[string]storage.TableUsage, len(usage))
	for _, u := range usage {
		usageByName[u.TableName] = u
	}

	tables := make([]TableInfo, 0, len(descs))
	var total int64
	for _, d := range descs {
		u := usageByName[d.TableName]
		tables = append(tables, TableInfo{
			Name:      d.TableName,
			LabelKey:  d.LabelKey,
			Clearable: d.Clearable,
			RowCount:  u.RowCount,
			SizeBytes: u.SizeBytes,
			Estimated: u.Estimated,
		})
		total += u.SizeBytes
	}

	return TableStats{TotalBytes: total, Tables: tables}, nil
}

// ClearTable 清空指定数据集合。仅允许 storage.AllModels 中标记为 Clearable=true 的集合。
//
// 安全约束：表名必须在白名单（FindDescriptor），且表必须 clearable=true。
func (s *Service) ClearTable(ctx context.Context, tableName string) error {
	_ = ctx
	desc := storage.FindDescriptor(tableName)
	if desc == nil {
		return fmt.Errorf("unknown table: %s", tableName)
	}
	if !desc.Clearable {
		return fmt.Errorf("table %s is not clearable", tableName)
	}
	switch tableName {
	case "preferences":
		return s.holder.Current().Store.ClearPreferences()
	case storage.PlayHistoryTable:
		return s.holder.Current().Store.ClearPlayHistory()
	default:
		return fmt.Errorf("table %s is not clearable", tableName)
	}
}

// SetCustomPath 把 SQLite 数据库迁移到新位置。
//
// 流程（保证不丢数据）：
//  1. 校验目标目录可写
//  2. 关闭当前句柄（关闭时 WAL 已 checkpoint 回主库）
//  3. 复制数据库文件（含残留 -wal/-shm）到新位置
//  4. 在新位置打开并补齐默认状态
//  5. 通过 → 删旧文件 + 写 storage.json + 替换 holder
//  6. 任意步骤失败 → 删除新位置已写文件 + 重开旧数据文件 + 报错
func (s *Service) SetCustomPath(_ context.Context, newPath string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if newPath == "" {
		return errors.New("path required")
	}
	abs, err := filepath.Abs(newPath)
	if err != nil {
		return fmt.Errorf("abs path: %w", err)
	}
	current := s.holder.Current()
	if filepath.Clean(abs) == filepath.Clean(current.Path) {
		return nil // 无变化
	}

	if err := preflightWritable(abs); err != nil {
		return fmt.Errorf("target not writable: %w", err)
	}

	oldPath := current.Path
	if err := current.Close(); err != nil {
		return fmt.Errorf("close current: %w", err)
	}

	written, err := copyDataFile(oldPath, abs)
	if err != nil {
		s.tryReopenLegacy(oldPath)
		removeAll(written)
		return fmt.Errorf("copy data file: %w", err)
	}

	next, err := storage.Open(abs)
	if err != nil {
		s.tryReopenLegacy(oldPath)
		removeAll(written)
		return fmt.Errorf("open new: %w", err)
	}

	prev := s.holder.Swap(next)
	_ = prev

	if err := s.cfg.SetDataPath(abs); err != nil {
		_ = next.Close()
		removeAll(written)
		s.tryReopenLegacy(oldPath)
		return fmt.Errorf("persist config: %w", err)
	}

	_ = os.Remove(oldPath)
	_ = os.Remove(oldPath + "-wal")
	_ = os.Remove(oldPath + "-shm")
	return nil
}

// ResetPath 把数据文件迁回平台默认位置。
func (s *Service) ResetPath(ctx context.Context) error {
	def, err := storage.DefaultDBPath()
	if err != nil {
		return err
	}
	current := s.holder.Current()
	if filepath.Clean(def) == filepath.Clean(current.Path) {
		// 已经是默认位置：仅清空 storage.json 中的自定义路径配置
		return s.cfg.ResetDataPath()
	}
	if err := s.SetCustomPath(ctx, def); err != nil {
		return err
	}
	return s.cfg.ResetDataPath()
}

func (s *Service) tryReopenLegacy(oldPath string) {
	db, err := storage.Open(oldPath)
	if err != nil {
		panic(fmt.Errorf("storagesvc: failed to reopen data file at %s: %w", oldPath, err))
	}
	s.holder.Swap(db)
}

// preflightWritable 校验目标文件所在目录可写。
func preflightWritable(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	test := filepath.Join(dir, ".foundation-write-test")
	f, err := os.Create(test)
	if err != nil {
		return err
	}
	_ = f.Close()
	return os.Remove(test)
}

func copyDataFile(src, dst string) ([]string, error) {
	if err := copyFile(src, dst); err != nil {
		return nil, err
	}
	written := []string{dst}
	// SQLite WAL/SHM 边车文件：关闭前已 checkpoint(TRUNCATE) 回主库，
	// 正常情况下它们为空或不存在；仍尽力一并复制，避免极端时序下遗漏。
	for _, suffix := range []string{"-wal", "-shm"} {
		if _, err := os.Stat(src + suffix); err != nil {
			continue
		}
		if err := copyFile(src+suffix, dst+suffix); err != nil {
			removeAll(written)
			return nil, err
		}
		written = append(written, dst+suffix)
	}
	return written, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

func removeAll(paths []string) {
	for _, p := range paths {
		_ = os.Remove(p)
	}
}

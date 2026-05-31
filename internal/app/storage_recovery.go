package app

import (
	"fmt"

	"YyslsPlayer/internal/storage"
	"YyslsPlayer/internal/utils/logx"
)

func openStorageWithRecovery(cfgMgr *storage.ConfigManager) (*storage.DB, string, error) {
	dbPath, err := cfgMgr.EffectiveDBPath()
	if err != nil {
		return nil, "", fmt.Errorf("resolve db path: %w", err)
	}

	db, err := storage.Open(dbPath)
	if err == nil {
		return db, dbPath, nil
	}

	cfg := cfgMgr.Get()
	if cfg.DataPath == "" {
		return nil, "", fmt.Errorf("open db: %w", err)
	}

	logx.For("app").Warn("custom storage path failed; falling back to default", "path", dbPath, "error", err)
	defaultPath, defaultErr := storage.DefaultDBPath()
	if defaultErr != nil {
		return nil, "", fmt.Errorf("resolve default db path: %w", defaultErr)
	}
	db, defaultErr = storage.Open(defaultPath)
	if defaultErr != nil {
		return nil, "", fmt.Errorf("open default db after custom path failed (%s: %w): %w", dbPath, err, defaultErr)
	}
	if resetErr := cfgMgr.ResetDataPath(); resetErr != nil {
		_ = db.Close()
		return nil, "", fmt.Errorf("open db: %w; reset custom path: %w", err, resetErr)
	}

	logx.For("app").Info("default storage opened after custom path recovery", "path", defaultPath, "failedPath", dbPath)
	return db, defaultPath, nil
}

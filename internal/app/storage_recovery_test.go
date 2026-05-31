package app

import (
	"os"
	"path/filepath"
	"testing"

	"YyslsPlayer/internal/storage"
)

func TestOpenStorageWithRecoveryFallsBackFromInvalidCustomPath(t *testing.T) {
	appData := t.TempDir()
	t.Setenv("AppData", appData)
	t.Setenv("XDG_DATA_HOME", appData)
	t.Setenv("HOME", appData)

	customPath := filepath.Join(t.TempDir(), "legacy.db")
	if err := os.WriteFile(customPath, []byte("SQLite format 3\x00legacy"), 0o644); err != nil {
		t.Fatalf("write legacy db: %v", err)
	}

	cfgMgr, err := storage.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() failed: %v", err)
	}
	if err := cfgMgr.SetDataPath(customPath); err != nil {
		t.Fatalf("SetDataPath() failed: %v", err)
	}

	db, openedPath, err := openStorageWithRecovery(cfgMgr)
	if err != nil {
		t.Fatalf("openStorageWithRecovery() failed: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Fatalf("Close() failed: %v", err)
		}
	})

	defaultPath, err := storage.DefaultDBPath()
	if err != nil {
		t.Fatalf("DefaultDBPath() failed: %v", err)
	}
	if openedPath != defaultPath {
		t.Fatalf("opened path = %q, want default %q", openedPath, defaultPath)
	}
	if cfg := cfgMgr.Get(); cfg.DataPath != "" {
		t.Fatalf("DataPath = %q, want reset", cfg.DataPath)
	}
	if _, err := os.Stat(customPath); err != nil {
		t.Fatalf("legacy file should be left untouched: %v", err)
	}
}

package hotkey

import (
	"path/filepath"
	"testing"

	"YyslsPlayer/internal/services/keysim"
	"YyslsPlayer/internal/services/player"
	"YyslsPlayer/internal/storage"
)

func TestExternalBindingConflictsWithBuiltin(t *testing.T) {
	db := storage.MustOpen(filepath.Join(t.TempDir(), "test.db"))
	defer db.Close()
	svc := New(storage.NewHolder(db), player.New(keysim.New(keysim.NewStubDriver())))
	states := svc.SetExternalBindings("macro", []ExternalBinding{{TargetID: "macro:1", Accelerator: "F9", Enabled: true}})
	if len(states) != 1 {
		t.Fatalf("states = %d, want 1", len(states))
	}
	if states[0].ErrorCode != CodeAppConflict {
		t.Fatalf("errorCode = %q, want %q", states[0].ErrorCode, CodeAppConflict)
	}
}

func TestNormalizeAcceleratorExport(t *testing.T) {
	got, err := NormalizeAccelerator("ctrl+alt+f9")
	if err != nil {
		t.Fatalf("NormalizeAccelerator failed: %v", err)
	}
	if got != "Ctrl+Alt+F9" {
		t.Fatalf("NormalizeAccelerator = %q", got)
	}
}

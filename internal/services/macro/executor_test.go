//go:build completion

package macro

import (
	"context"
	"path/filepath"
	"testing"

	"YyslsPlayer/internal/services/keysim"
	"YyslsPlayer/internal/storage"
)

type noopMacroDriver struct{}

func (noopMacroDriver) Send(context.Context, keysim.KeyEvent, keysim.RunOptions) error {
	return nil
}

func newExecutorTestService(t *testing.T) *Service {
	t.Helper()
	db, err := storage.Open(filepath.Join(t.TempDir(), "macro-executor.db"))
	if err != nil {
		t.Fatalf("open test storage: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close test storage: %v", err)
		}
	})
	return New(storage.NewHolder(db), keysim.New(noopMacroDriver{}), nil, nil)
}

func saveInterruptTestMacro(t *testing.T, service *Service, name string, policy string) MacroDetailDTO {
	t.Helper()
	detail, err := service.SaveMacro(context.Background(), SaveMacroRequest{
		Name:            name,
		RepeatMode:      RepeatModeOnce,
		RepeatCount:     1,
		InterruptPolicy: policy,
		Steps: []MacroStepDTO{
			{Kind: StepDelay, WaitMs: 5_000},
		},
	})
	if err != nil {
		t.Fatalf("save macro %q: %v", name, err)
	}
	return detail
}

func TestRunMacroSecondTriggerStopsSameMacro(t *testing.T) {
	service := newExecutorTestService(t)
	detail := saveInterruptTestMacro(t, service, "stop on retrigger", InterruptStop)
	if detail.Profile.InterruptPolicy != InterruptStop {
		t.Fatalf("interrupt policy = %q, want %q", detail.Profile.InterruptPolicy, InterruptStop)
	}

	running, err := service.RunMacro(context.Background(), detail.Profile.ID)
	if err != nil {
		t.Fatalf("first RunMacro: %v", err)
	}
	if running.State != StateRunning {
		t.Fatalf("first state = %q, want %q", running.State, StateRunning)
	}

	stopped, err := service.RunMacro(context.Background(), detail.Profile.ID)
	if err != nil {
		t.Fatalf("second RunMacro: %v", err)
	}
	if stopped.State != StateStopped {
		t.Fatalf("second state = %q, want %q", stopped.State, StateStopped)
	}
	if stopped.MacroID != detail.Profile.ID {
		t.Fatalf("stopped macro id = %d, want %d", stopped.MacroID, detail.Profile.ID)
	}
	if stopped.StartedAt != running.StartedAt {
		t.Fatalf("macro restarted: startedAt changed from %d to %d", running.StartedAt, stopped.StartedAt)
	}

	service.mu.Lock()
	current := service.current
	service.mu.Unlock()
	if current != nil {
		t.Fatalf("current session still active for macro %d", current.macroID)
	}
}

func TestRunMacroInterruptStopDoesNotStartRequestedMacro(t *testing.T) {
	service := newExecutorTestService(t)
	active := saveInterruptTestMacro(t, service, "active", InterruptIgnore)
	stopper, err := service.SaveMacro(context.Background(), SaveMacroRequest{
		Name:            "stopper",
		RepeatMode:      RepeatModeOnce,
		RepeatCount:     1,
		InterruptPolicy: InterruptStop,
	})
	if err != nil {
		t.Fatalf("save zero-step stopper macro: %v", err)
	}

	running, err := service.RunMacro(context.Background(), active.Profile.ID)
	if err != nil {
		t.Fatalf("run active macro: %v", err)
	}
	stopped, err := service.RunMacro(context.Background(), stopper.Profile.ID)
	if err != nil {
		t.Fatalf("run stopper macro: %v", err)
	}
	if stopped.State != StateStopped || stopped.MacroID != active.Profile.ID {
		t.Fatalf("state after stop policy = %+v, want active macro stopped", stopped)
	}
	if stopped.StartedAt != running.StartedAt {
		t.Fatalf("requested macro started unexpectedly: startedAt changed from %d to %d", running.StartedAt, stopped.StartedAt)
	}
}

func TestNormalizeProfileAcceptsInterruptStop(t *testing.T) {
	profile := storage.MacroProfile{InterruptPolicy: InterruptStop}
	normalizeProfile(&profile)
	if profile.InterruptPolicy != InterruptStop {
		t.Fatalf("interrupt policy = %q, want %q", profile.InterruptPolicy, InterruptStop)
	}
}

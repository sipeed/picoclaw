package sandbox

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
)

func TestExpandHomePath(t *testing.T) {
	if got := expandHomePath(""); got != "" {
		t.Fatalf("expandHomePath(\"\") = %q, want empty", got)
	}
	if got := expandHomePath("abc"); got != "abc" {
		t.Fatalf("expandHomePath(abc) = %q", got)
	}
	if got := expandHomePath("~"); got == "" {
		t.Fatal("expandHomePath(~) should resolve to home")
	}
	if got := expandHomePath("~/x"); got == "" || got == "~/x" {
		t.Fatalf("expandHomePath(~/x) = %q, expected resolved path", got)
	}
}

func TestNewFromConfig_HostMode(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Sandbox.Mode = "off"

	sb := NewFromConfig(t.TempDir(), true, cfg)
	if _, ok := sb.(*HostSandbox); !ok {
		t.Fatalf("expected HostSandbox, got %T", sb)
	}
	if err := sb.Prune(context.Background()); err != nil {
		t.Fatalf("Prune() error: %v", err)
	}
}

func TestNewFromConfig_AllModeReturnsUnavailableWhenBlocked(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Sandbox.Mode = "all"
	cfg.Agents.Defaults.Sandbox.Docker.Network = "host"
	cfg.Agents.Defaults.Sandbox.Prune.IdleHours = 0
	cfg.Agents.Defaults.Sandbox.Prune.MaxAgeDays = 0

	sb := NewFromConfig(t.TempDir(), true, cfg)
	if _, ok := sb.(*unavailableSandbox); !ok {
		t.Fatalf("expected unavailableSandbox, got %T", sb)
	}
	if err := sb.Start(context.Background()); err == nil {
		t.Fatal("expected unavailable sandbox start error")
	}
}

func TestScopedSandboxManager_PruneLoopLifecycle(t *testing.T) {
	m := &scopedSandboxManager{
		mode:            "all",
		pruneIdleHours:  1,
		pruneMaxAgeDays: 0,
		scoped:          map[string]Sandbox{},
	}

	m.ensurePruneLoop()
	if m.loopStop == nil || m.loopDone == nil {
		t.Fatal("expected manager prune loop to start")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	m.stopPruneLoop(ctx)
	if m.loopStop != nil || m.loopDone != nil {
		t.Fatal("expected manager prune loop state reset after stop")
	}
}

func TestScopedSandboxManager_PruneOnceLoadRegistryError(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	stateDir := filepath.Join(home, ".picoclaw", "state", "sandbox")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatalf("mkdir state dir: %v", err)
	}
	regPath := filepath.Join(stateDir, "containers.json")
	if err := os.WriteFile(regPath, []byte("{not-json"), 0o644); err != nil {
		t.Fatalf("write invalid registry: %v", err)
	}

	m := &scopedSandboxManager{
		mode:            "all",
		pruneIdleHours:  1,
		pruneMaxAgeDays: 0,
		scoped:          map[string]Sandbox{},
	}
	if err := m.pruneOnce(context.Background()); err == nil {
		t.Fatal("expected pruneOnce() to return registry load error")
	}
}

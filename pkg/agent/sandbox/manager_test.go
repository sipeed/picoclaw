package sandbox

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/routing"
)

func TestNormalizeWorkspaceAccess(t *testing.T) {
	if got := normalizeWorkspaceAccess("ro"); got != "ro" {
		t.Fatalf("normalizeWorkspaceAccess(ro) = %q", got)
	}
	if got := normalizeWorkspaceAccess("RW"); got != "rw" {
		t.Fatalf("normalizeWorkspaceAccess(RW) = %q", got)
	}
	if got := normalizeWorkspaceAccess("invalid"); got != "none" {
		t.Fatalf("normalizeWorkspaceAccess(invalid) = %q", got)
	}
}

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
	// NewFromConfig always returns a HostSandbox regardless of mode config.
	sb := NewFromConfig(t.TempDir(), true, nil)
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
	cfg.Agents.Defaults.Sandbox.Prune.IdleHours = config.IntPtr(0)
	cfg.Agents.Defaults.Sandbox.Prune.MaxAgeDays = config.IntPtr(0)

	// NewFromConfigWithAgent is the manager factory; when Docker is unavailable
	// it should return an unavailableSandbox that implements Manager.
	mgr := NewFromConfigWithAgent(t.TempDir(), true, cfg, "test")
	if mgr == nil {
		t.Fatal("expected non-nil Manager when mode=all")
	}
	if _, ok := mgr.(*unavailableSandboxManager); !ok {
		t.Fatalf("expected unavailableSandbox, got %T", mgr)
	}
	// Resolve() must propagate the unavailability error.
	if _, err := mgr.Resolve(context.Background()); err == nil {
		t.Fatal("expected unavailable sandbox Resolve() to return error")
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
	stateDir := filepath.Join(home, ".picoclaw", "sandbox")
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

func TestScopedSandboxManager_ShouldSandbox_NonMain(t *testing.T) {
	m := &scopedSandboxManager{
		mode:    config.SandboxModeNonMain,
		agentID: "default",
	}

	if m.shouldSandbox(context.Background()) {
		t.Fatal("expected background context to map to main session (host path)")
	}

	if m.shouldSandbox(WithSessionKey(context.Background(), "main")) {
		t.Fatal("expected explicit main alias to remain host path")
	}

	mainKey := routing.BuildAgentMainSessionKey("default")
	if m.shouldSandbox(WithSessionKey(context.Background(), mainKey)) {
		t.Fatal("expected agent main session key to remain host path")
	}

	nonMainKey := "agent:default:direct:user-1"
	if !m.shouldSandbox(WithSessionKey(context.Background(), nonMainKey)) {
		t.Fatal("expected non-main session to use sandbox path")
	}
}

func TestHostOnlyManager(t *testing.T) {
	// hostOnlyManager just delegates to its inner host sandbox.
	workspace := t.TempDir()
	host := NewHostSandbox(workspace, false)
	mgr := &hostOnlyManager{host: host}

	ctx := context.Background()
	if err := mgr.Start(ctx); err != nil {
		t.Fatalf("Start() returned error: %v", err)
	}
	if err := mgr.Prune(ctx); err != nil {
		t.Fatalf("Prune() returned error: %v", err)
	}

	sb, err := mgr.Resolve(ctx)
	if err != nil {
		t.Fatalf("Resolve() returned error: %v", err)
	}
	if sb == nil {
		t.Fatal("Resolve() returned nil sandbox")
	}

	if fs := mgr.Fs(); fs == nil {
		t.Fatal("Fs() returned nil")
	}

	gotWs := mgr.GetWorkspace(ctx)
	if gotWs != host.GetWorkspace(ctx) {
		t.Fatalf("GetWorkspace() = %q, want %q", gotWs, host.GetWorkspace(ctx))
	}

	// Exec and ExecStream should also just delegate without panic.
	// Executing a simple command like "echo"
	req := ExecRequest{Command: "echo", Args: []string{"test"}}
	res, err := mgr.Exec(ctx, req)
	if err != nil {
		t.Fatalf("Exec() returned error: %v", err)
	}
	if res.ExitCode != 0 {
		t.Fatalf("Exec() returned non-zero exit code: %d", res.ExitCode)
	}

	streamRes, streamErr := mgr.ExecStream(ctx, req, func(e ExecEvent) error { return nil })
	if streamErr != nil {
		t.Fatalf("ExecStream() returned error: %v", streamErr)
	}
	if streamRes.ExitCode != 0 {
		t.Fatalf("ExecStream() returned non-zero exit code: %d", streamRes.ExitCode)
	}
}

func TestUnavailableSandboxManager(t *testing.T) {
	errReason := os.ErrPermission
	mgr := NewUnavailableSandboxManager(errReason)
	ctx := context.Background()

	if err := mgr.Start(ctx); err != errReason {
		t.Fatalf("Start() = %v, want %v", err, errReason)
	}

	// Prune is a no-op, shouldn't return error
	if err := mgr.Prune(ctx); err != nil {
		t.Fatalf("Prune() = %v, want nil", err)
	}

	_, err := mgr.Resolve(ctx)
	if err == nil {
		t.Fatal("Resolve() expected error, got nil")
	}

	if ws := mgr.GetWorkspace(ctx); ws != "" {
		t.Fatalf("GetWorkspace() = %q, want empty", ws)
	}

	req := ExecRequest{Command: "ls"}
	_, err = mgr.Exec(ctx, req)
	if err == nil {
		t.Fatal("Exec() expected error, got nil")
	}

	_, err = mgr.ExecStream(ctx, req, nil)
	if err == nil {
		t.Fatal("ExecStream() expected error, got nil")
	}

	fs := mgr.Fs()
	if fs == nil {
		t.Fatal("Fs() returned nil")
	}

	_, err = fs.ReadFile(ctx, "test.txt")
	if err == nil {
		t.Fatal("ReadFile() expected error, got nil")
	}

	err = fs.WriteFile(ctx, "test.txt", []byte("a"), false)
	if err == nil {
		t.Fatal("WriteFile() expected error, got nil")
	}

	_, err = fs.ReadDir(ctx, ".")
	if err == nil {
		t.Fatal("ReadDir() expected error, got nil")
	}
}

func TestScopedSandboxManager_Delegates(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	ws := filepath.Join(home, "default_ws")

	m := &scopedSandboxManager{
		mode:    config.SandboxModeNonMain,
		agentID: "agent-1",
		host:    NewHostSandbox(ws, false),
		scoped:  map[string]Sandbox{},
	}
	_ = m.host.Start(context.Background())
	m.fs = &managerFS{m: m}

	// 1. When ShouldSandbox is false (Context is main session), it delegates to HostSandbox.
	ctxMain := WithSessionKey(context.Background(), routing.BuildAgentMainSessionKey("agent-1"))

	if got := m.GetWorkspace(ctxMain); got != ws {
		t.Fatalf("GetWorkspace(main) = %q, want %q", got, ws)
	}

	req := ExecRequest{Command: "echo", Args: []string{"hello"}}
	res, err := m.Exec(ctxMain, req)
	if err != nil {
		t.Fatalf("Exec(main) error: %v", err)
	}
	if res.ExitCode != 0 {
		t.Fatalf("Exec(main) exit code: %d", res.ExitCode)
	}

	_, err = m.ExecStream(ctxMain, req, func(e ExecEvent) error { return nil })
	if err != nil {
		t.Fatalf("ExecStream(main) error: %v", err)
	}

	fs := m.Fs()
	testFile := "test_delegate.txt"
	err = fs.WriteFile(ctxMain, testFile, []byte("ok"), true)
	if err != nil {
		t.Fatalf("WriteFile(main) error: %v", err)
	}
	defer os.Remove(filepath.Join(ws, testFile))

	data, err := fs.ReadFile(ctxMain, testFile)
	if err != nil || string(data) != "ok" {
		t.Fatalf("ReadFile(main) error: %v, data: %q", err, string(data))
	}

	entries, err := fs.ReadDir(ctxMain, ".")
	if err != nil || len(entries) == 0 {
		t.Fatalf("ReadDir(main) error: %v, len: %d", err, len(entries))
	}

	sb, err := m.Resolve(ctxMain)
	if err != nil {
		t.Fatalf("Resolve(main) error: %v", err)
	}
	if sb != m.host {
		t.Fatal("Resolve(main) should return host sandbox")
	}
}

func TestScopedSandboxManager_ContainerDelegates(t *testing.T) {
	_, cleanup := skipIfNoDocker(t)
	defer cleanup()

	home := t.TempDir()
	t.Setenv("HOME", home)
	ws := filepath.Join(home, "container_ws")
	os.MkdirAll(ws, 0o755)

	mockContainer := NewHostSandbox(ws, false)
	_ = mockContainer.Start(context.Background())

	m := &scopedSandboxManager{
		mode:    config.SandboxModeAll,
		agentID: "agent-1",
		scoped:  map[string]Sandbox{},
	}
	m.fs = &managerFS{m: m}

	ctx := WithSessionKey(context.Background(), "test-session")
	scopeKey := m.scopeKeyFromContext(ctx)

	// Pre-inject the mock container to bypass actual docker creation
	m.scoped[scopeKey] = mockContainer

	// Now shouldSandbox(ctx) is true, so manager methods should delegate to mockContainer
	if got := m.GetWorkspace(ctx); got != ws {
		t.Fatalf("GetWorkspace(container) = %q, want %q", got, ws)
	}

	req := ExecRequest{Command: "echo", Args: []string{"hello"}}
	res, err := m.Exec(ctx, req)
	if err != nil {
		t.Fatalf("Exec(container) error: %v", err)
	}
	if res.ExitCode != 0 {
		t.Fatalf("Exec(container) exit code: %d", res.ExitCode)
	}

	_, err = m.ExecStream(ctx, req, func(e ExecEvent) error { return nil })
	if err != nil {
		t.Fatalf("ExecStream(container) error: %v", err)
	}

	fs := m.Fs()
	testFile := "test_container_delegate.txt"
	err = fs.WriteFile(ctx, testFile, []byte("ok"), true)
	if err != nil {
		t.Fatalf("WriteFile(container) error: %v", err)
	}
	defer os.Remove(filepath.Join(ws, testFile))

	data, err := fs.ReadFile(ctx, testFile)
	if err != nil || string(data) != "ok" {
		t.Fatalf("ReadFile(container) error: %v, data: %q", err, string(data))
	}

	entries, err := fs.ReadDir(ctx, ".")
	if err != nil || len(entries) == 0 {
		t.Fatalf("ReadDir(container) error: %v, len: %d", err, len(entries))
	}

	sb, err := m.Resolve(ctx)
	if err != nil {
		t.Fatalf("Resolve(container) error: %v", err)
	}
	if sb != mockContainer {
		t.Fatal("Resolve(container) should return the mock container sandbox")
	}
}

func TestScopedSandboxManager_ContainerCreationError(t *testing.T) {
	_, cleanup := skipIfNoDocker(t)
	defer cleanup()

	home := t.TempDir()
	t.Setenv("HOME", home)

	m := &scopedSandboxManager{
		mode:          config.SandboxModeAll,
		workspaceRoot: home,
		dockerCfg:     config.AgentSandboxDockerConfig{Image: "non-existent-image-12345"},
		scoped:        map[string]Sandbox{},
	}

	ctx := WithSessionKey(context.Background(), "error-session")

	// Fast-path misses, falls through to buildScopedContainerSandbox and Start()
	// Start() succeeds due to lazy evaluation, but Exec() should fail because
	// Docker is either unavailable or the image is missing
	sb, err := m.Resolve(ctx)
	if err != nil {
		t.Fatalf("expected Resolve() to succeed lazily, got error: %v", err)
	}

	res, err := sb.Exec(ctx, ExecRequest{Command: "echo"})
	if err == nil && res.ExitCode == 0 {
		t.Fatalf("expected Exec() to fail when container creation fails, but it succeeded: %#v", res)
	}
}

func TestScopedSandboxManager_PruneOnceFull(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	stateDir := filepath.Join(home, ".picoclaw", "sandbox")
	_ = os.MkdirAll(stateDir, 0o755)

	m := &scopedSandboxManager{ // this is correct initialization for the test
		mode:            config.SandboxModeAll,
		pruneIdleHours:  1,
		pruneMaxAgeDays: 0,
		scoped:          map[string]Sandbox{},
	}

	regPath := filepath.Join(stateDir, "containers.json")
	oldTime := time.Now().Add(-2 * time.Hour).UnixMilli()

	data := fmt.Sprintf(`{"entries": [{"container_name": "prune-me", "last_active_at": %d}]}`, oldTime)
	_ = os.WriteFile(regPath, []byte(data), 0o644)

	_ = m.pruneOnce(context.Background())

	b, _ := os.ReadFile(regPath)
	if strings.Contains(string(b), "prune-me") {
		t.Fatalf("expected prune-me to be removed from registry, got %s", string(b))
	}
}

package sandbox

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
)

func TestResolveContainerPath_Relative(t *testing.T) {
	got, err := resolveContainerPath("foo/bar.txt")
	if err != nil {
		t.Fatalf("resolveContainerPath returned error: %v", err)
	}
	if got != "/workspace/foo/bar.txt" {
		t.Fatalf("resolveContainerPath = %q, want %q", got, "/workspace/foo/bar.txt")
	}
}

func TestResolveContainerPath_AbsoluteInWorkspace(t *testing.T) {
	got, err := resolveContainerPath("/workspace/a/b.txt")
	if err != nil {
		t.Fatalf("resolveContainerPath returned error: %v", err)
	}
	if got != "/workspace/a/b.txt" {
		t.Fatalf("resolveContainerPath = %q, want %q", got, "/workspace/a/b.txt")
	}
}

func TestResolveContainerPath_RejectsEscape(t *testing.T) {
	_, err := resolveContainerPath("../../etc/passwd")
	if err == nil {
		t.Fatal("expected error for path traversal")
	}
	if !strings.Contains(err.Error(), "outside container workspace") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveContainerPath_RejectsAbsoluteOutsideWorkspace(t *testing.T) {
	_, err := resolveContainerPath("/etc/passwd")
	if err == nil {
		t.Fatal("expected error for absolute path outside workspace")
	}
	if !strings.Contains(err.Error(), "outside container workspace") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildExecCommand_DefaultShell(t *testing.T) {
	sb := NewContainerSandbox(ContainerSandboxConfig{})
	cmd, wd, err := sb.buildExecCommand(ExecRequest{
		Command: "echo hi",
	})
	if err != nil {
		t.Fatalf("buildExecCommand returned error: %v", err)
	}
	if wd != "/workspace" {
		t.Fatalf("working dir = %q, want /workspace", wd)
	}
	if len(cmd) != 3 || cmd[0] != "sh" || cmd[1] != "-lc" || cmd[2] != "echo hi" {
		t.Fatalf("unexpected command: %#v", cmd)
	}
}

func TestBuildExecCommand_WithArgs(t *testing.T) {
	sb := NewContainerSandbox(ContainerSandboxConfig{})
	cmd, wd, err := sb.buildExecCommand(ExecRequest{
		Command: "ls",
		Args:    []string{"-la", "/workspace"},
	})
	if err != nil {
		t.Fatalf("buildExecCommand returned error: %v", err)
	}
	if wd != "/workspace" {
		t.Fatalf("working dir = %q, want /workspace", wd)
	}
	if len(cmd) != 3 || cmd[0] != "ls" || cmd[1] != "-la" || cmd[2] != "/workspace" {
		t.Fatalf("unexpected command: %#v", cmd)
	}
}

func TestBuildExecCommand_WorkingDirUsesResolvedDirectory(t *testing.T) {
	sb := NewContainerSandbox(ContainerSandboxConfig{})
	cmd, wd, err := sb.buildExecCommand(ExecRequest{
		Command:    "cat foo.txt",
		WorkingDir: "subdir",
	})
	if err != nil {
		t.Fatalf("buildExecCommand returned error: %v", err)
	}
	if wd != "/workspace/subdir" {
		t.Fatalf("working dir = %q, want /workspace/subdir", wd)
	}
	if len(cmd) == 0 {
		t.Fatal("expected command to be populated")
	}
}

func TestContainerSandbox_FailClosedOnStartError(t *testing.T) {
	sb := NewContainerSandbox(ContainerSandboxConfig{})
	sb.startErr = errors.New("docker unavailable")

	_, err := sb.Exec(context.Background(), ExecRequest{Command: "echo hi"})
	if err == nil {
		t.Fatal("expected exec error")
	}
	if !strings.Contains(err.Error(), "docker unavailable") {
		t.Fatalf("unexpected exec error: %v", err)
	}

	_, err = sb.Fs().ReadFile(context.Background(), "a.txt")
	if err == nil {
		t.Fatal("expected fs read error")
	}
	if !strings.Contains(err.Error(), "docker unavailable") {
		t.Fatalf("unexpected fs read error: %v", err)
	}

	err = sb.Fs().WriteFile(context.Background(), "a.txt", []byte("x"), true)
	if err == nil {
		t.Fatal("expected fs write error")
	}
	if !strings.Contains(err.Error(), "docker unavailable") {
		t.Fatalf("unexpected fs write error: %v", err)
	}
}

func TestHostDirForContainerPath_Workspace(t *testing.T) {
	sb := NewContainerSandbox(ContainerSandboxConfig{Workspace: "/tmp/ws"})

	got, ok := sb.hostDirForContainerPath("/workspace/a/b")
	if !ok {
		t.Fatal("expected workspace path to resolve")
	}
	if got != "/tmp/ws/a/b" {
		t.Fatalf("host path = %q, want %q", got, "/tmp/ws/a/b")
	}
}

func TestHostDirForContainerPath_OutsideWorkspace(t *testing.T) {
	sb := NewContainerSandbox(ContainerSandboxConfig{Workspace: "/tmp/ws"})

	if got, ok := sb.hostDirForContainerPath("/etc"); ok || got != "" {
		t.Fatalf("expected outside path to be rejected, got (%q, %v)", got, ok)
	}
}

func TestHostDirForContainerPath_ReadOnlyWorkspace(t *testing.T) {
	sb := NewContainerSandbox(ContainerSandboxConfig{
		Workspace:       "/tmp/ws",
		WorkspaceAccess: "ro",
	})
	if got, ok := sb.hostDirForContainerPath("/workspace/a"); ok || got != "" {
		t.Fatalf("expected read-only workspace to disable host path mapping, got (%q, %v)", got, ok)
	}
}

func TestComputeContainerConfigHash_EnvOrderInsensitive(t *testing.T) {
	base := ContainerSandboxConfig{
		Image:     "img",
		Workspace: "/tmp/ws",
		Workdir:   "/workspace",
		Env: map[string]string{
			"A": "1",
			"B": "2",
		},
	}
	left := computeContainerConfigHash(base)
	base.Env = map[string]string{
		"B": "2",
		"A": "1",
	}
	right := computeContainerConfigHash(base)
	if left != right {
		t.Fatalf("expected hash to ignore env key order: %q != %q", left, right)
	}
}

func TestComputeContainerConfigHash_ArrayOrderSensitive(t *testing.T) {
	base := ContainerSandboxConfig{
		Image:     "img",
		Workspace: "/tmp/ws",
		Workdir:   "/workspace",
		DNS:       []string{"1.1.1.1", "8.8.8.8"},
	}
	left := computeContainerConfigHash(base)
	base.DNS = []string{"8.8.8.8", "1.1.1.1"}
	right := computeContainerConfigHash(base)
	if left == right {
		t.Fatal("expected hash to change when array order changes")
	}
}

func TestComputeContainerConfigHash_WorkspaceAccessAndRootAffectHash(t *testing.T) {
	base := ContainerSandboxConfig{
		Image:           "img",
		Workspace:       "/tmp/ws",
		Workdir:         "/workspace",
		WorkspaceAccess: "none",
		WorkspaceRoot:   "/tmp/sbx-a",
	}
	left := computeContainerConfigHash(base)
	base.WorkspaceAccess = "ro"
	if left == computeContainerConfigHash(base) {
		t.Fatal("expected hash to change when workspace_access changes")
	}
	base.WorkspaceAccess = "none"
	base.WorkspaceRoot = "/tmp/sbx-b"
	if left == computeContainerConfigHash(base) {
		t.Fatal("expected hash to change when workspace_root changes")
	}
}

func TestBuildDockerUlimit_NumberValue(t *testing.T) {
	value := int64(256)
	ul, ok := buildDockerUlimit("nproc", config.AgentSandboxDockerUlimitValue{Value: &value})
	if !ok || ul == nil {
		t.Fatal("expected ulimit to be built")
	}
	if ul.Soft != 256 || ul.Hard != 256 {
		t.Fatalf("unexpected ulimit values: soft=%d hard=%d", ul.Soft, ul.Hard)
	}
}

func TestContainerSandbox_Binds_WorkspaceAccessModes(t *testing.T) {
	root := t.TempDir()

	ro := NewContainerSandbox(ContainerSandboxConfig{
		Workspace:       filepath.Join(root, "ws-ro"),
		WorkspaceAccess: "ro",
		Workdir:         "/workspace",
	})
	roBinds := ro.binds()
	if len(roBinds) == 0 || !strings.HasSuffix(roBinds[0], ":/workspace:ro") {
		t.Fatalf("unexpected ro bind: %#v", roBinds)
	}

	rw := NewContainerSandbox(ContainerSandboxConfig{
		Workspace:       filepath.Join(root, "ws-rw"),
		WorkspaceAccess: "rw",
		Workdir:         "/workspace",
	})
	rwBinds := rw.binds()
	if len(rwBinds) == 0 || !strings.HasSuffix(rwBinds[0], ":/workspace:rw") {
		t.Fatalf("unexpected rw bind: %#v", rwBinds)
	}

	none := NewContainerSandbox(ContainerSandboxConfig{
		Workspace:       filepath.Join(root, "ws-none"),
		WorkspaceAccess: "none",
		Workdir:         "/workspace",
	})
	noneBinds := none.binds()
	if len(noneBinds) == 0 || !strings.HasSuffix(noneBinds[0], ":/workspace") {
		t.Fatalf("unexpected none bind: %#v", noneBinds)
	}
}

func TestContainerSandbox_RegistryPath_UsesSandboxStateDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	sb := NewContainerSandbox(ContainerSandboxConfig{
		Workspace:     "/tmp/ws",
		WorkspaceRoot: "/tmp/sbx",
	})
	want := filepath.Join(home, ".picoclaw", "state", "sandbox", "containers.json")
	if got := sb.registryPath(); got != want {
		t.Fatalf("registryPath = %q, want %q", got, want)
	}
}

func TestContainerSandbox_RegistryPath_UsesPicoClawHomeOverride(t *testing.T) {
	picoHome := t.TempDir()
	t.Setenv("PICOCLAW_HOME", picoHome)
	sb := NewContainerSandbox(ContainerSandboxConfig{})
	want := filepath.Join(picoHome, "state", "sandbox", "containers.json")
	if got := sb.registryPath(); got != want {
		t.Fatalf("registryPath = %q, want %q", got, want)
	}
}

func TestContainerSandbox_Start_BlockedSecurityConfig(t *testing.T) {
	sb := NewContainerSandbox(ContainerSandboxConfig{
		Network: "host",
	})
	err := sb.Start(context.Background())
	if err == nil {
		t.Fatal("expected start to fail for blocked network mode")
	}
	if !strings.Contains(err.Error(), "network mode") {
		t.Fatalf("unexpected error: %v", err)
	}
}

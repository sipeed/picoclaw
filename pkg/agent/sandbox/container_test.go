package sandbox

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
)

func TestResolveContainerPath_Relative(t *testing.T) {
	sb := NewContainerSandbox(ContainerSandboxConfig{Workdir: "/app"})
	if got := sb.GetWorkspace(context.Background()); got != "/app" {
		t.Errorf("GetWorkspace() = %q, want /app", got)
	}

	sb2 := NewContainerSandbox(ContainerSandboxConfig{})
	if got := sb2.GetWorkspace(context.Background()); got != "/workspace" {
		t.Errorf("GetWorkspace() default = %q, want /workspace", got)
	}

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

func TestParseTopLevelDirEntriesFromTar_KeepImmediateChildren(t *testing.T) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	writeHeader := func(hdr *tar.Header, content []byte) {
		t.Helper()
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("write header failed: %v", err)
		}
		if len(content) > 0 {
			if _, err := tw.Write(content); err != nil {
				t.Fatalf("write content failed: %v", err)
			}
		}
	}

	writeHeader(&tar.Header{Name: "it", Typeflag: tar.TypeDir, Mode: 0o755}, nil)
	writeHeader(&tar.Header{Name: "it/write.txt", Typeflag: tar.TypeReg, Mode: 0o644, Size: 1}, []byte("x"))
	writeHeader(&tar.Header{Name: "it/sub", Typeflag: tar.TypeDir, Mode: 0o755}, nil)
	writeHeader(&tar.Header{Name: "it/sub/nested.txt", Typeflag: tar.TypeReg, Mode: 0o644, Size: 1}, []byte("y"))
	if err := tw.Close(); err != nil {
		t.Fatalf("tar close failed: %v", err)
	}

	entries, err := parseTopLevelDirEntriesFromTar(tar.NewReader(&buf))
	if err != nil {
		t.Fatalf("parseTopLevelDirEntriesFromTar failed: %v", err)
	}

	got := make(map[string]struct{}, len(entries))
	for _, e := range entries {
		got[e.Name()] = struct{}{}
	}
	for _, want := range []string{"write.txt", "sub"} {
		if _, ok := got[want]; !ok {
			t.Fatalf("missing top-level entry %q in %+v", want, got)
		}
	}
	if _, ok := got["nested.txt"]; ok {
		t.Fatalf("nested file should not appear in top-level entries: %+v", got)
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
	if len(roBinds) == 0 || !strings.HasSuffix(roBinds[0], ":/workspace:ro,Z") {
		t.Fatalf("unexpected ro bind: %#v", roBinds)
	}

	rw := NewContainerSandbox(ContainerSandboxConfig{
		Workspace:       filepath.Join(root, "ws-rw"),
		WorkspaceAccess: "rw",
		Workdir:         "/workspace",
	})
	rwBinds := rw.binds()
	if len(rwBinds) == 0 || !strings.HasSuffix(rwBinds[0], ":/workspace:rw,Z") {
		t.Fatalf("unexpected rw bind: %#v", rwBinds)
	}

	none := NewContainerSandbox(ContainerSandboxConfig{
		Workspace:       filepath.Join(root, "ws-none"),
		WorkspaceAccess: "none",
		Workdir:         "/workspace",
		ContainerName:   "test-none",
	})
	noneBinds := none.binds()
	if len(noneBinds) == 0 || !strings.HasSuffix(noneBinds[0], ":/workspace:rw,Z") {
		t.Fatalf("expected none bind to isolated workspace, got: %#v", noneBinds)
	}
}

func TestContainerSandbox_RegistryPath_UsesSandboxStateDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	sb := NewContainerSandbox(ContainerSandboxConfig{
		Workspace:     "/tmp/ws",
		WorkspaceRoot: "/tmp/sbx",
	})
	want := filepath.Join(home, ".picoclaw", "sandbox", "containers.json")
	if got := sb.registryPath(); got != want {
		t.Fatalf("registryPath = %q, want %q", got, want)
	}
}

func TestContainerSandbox_RegistryPath_UsesPicoClawHomeOverride(t *testing.T) {
	picoHome := t.TempDir()
	t.Setenv("PICOCLAW_HOME", picoHome)
	sb := NewContainerSandbox(ContainerSandboxConfig{})
	want := filepath.Join(picoHome, "sandbox", "containers.json")
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

func TestContainerSandbox_StartCreatesWorkspaceBeforeDockerPing(t *testing.T) {
	workspace := filepath.Join(t.TempDir(), "workspace")
	workspaceRoot := filepath.Join(t.TempDir(), "sandbox-root")
	sb := NewContainerSandbox(ContainerSandboxConfig{
		Workspace:       workspace,
		WorkspaceRoot:   workspaceRoot,
		WorkspaceAccess: "none",
	})

	err := sb.Start(context.Background())
	if err == nil {
		_ = sb.Prune(context.Background())
		t.Skip("docker daemon available in this environment; skip unavailable-path assertion")
	}
	if !strings.Contains(err.Error(), "docker daemon unavailable") {
		t.Fatalf("Start() unexpected error: %v", err)
	}
	if _, stErr := os.Stat(workspace); stErr != nil {
		t.Fatalf("workspace should be created before docker ping: %v", stErr)
	}
	if _, stErr := os.Stat(workspaceRoot); stErr != nil {
		t.Fatalf("workspaceRoot should be created before docker ping: %v", stErr)
	}
}

func TestContainerSandbox_NoopPruneWithoutClient(t *testing.T) {
	sb := NewContainerSandbox(ContainerSandboxConfig{
		PruneIdleHours:  1,
		PruneMaxAgeDays: 0,
	})

	if err := sb.Prune(context.Background()); err != nil {
		t.Fatalf("Prune() with nil client should be noop, got: %v", err)
	}
}

func TestParseByteLimitAndHostConfig(t *testing.T) {
	if got, err := parseByteLimit("1024"); err != nil || got != 1024 {
		t.Fatalf("parseByteLimit numeric got (%d,%v), want (1024,nil)", got, err)
	}
	if got, err := parseByteLimit("1g"); err != nil || got <= 0 {
		t.Fatalf("parseByteLimit unit got (%d,%v), want positive", got, err)
	}
	if _, err := parseByteLimit("not-a-size"); err == nil {
		t.Fatal("expected parseByteLimit invalid input error")
	}

	soft := int64(256)
	hard := int64(512)
	sb := NewContainerSandbox(ContainerSandboxConfig{
		Workspace:       t.TempDir(),
		Workdir:         "/workspace",
		ReadOnlyRoot:    true,
		Network:         "none",
		CapDrop:         []string{"ALL"},
		Tmpfs:           []string{"/tmp:rw,noexec,nosuid", "  ", "/run"},
		PidsLimit:       123,
		Memory:          "1g",
		MemorySwap:      "2g",
		Cpus:            1.5,
		SeccompProfile:  "sec-profile.json",
		ApparmorProfile: "apparmor-profile",
		Ulimits: map[string]config.AgentSandboxDockerUlimitValue{
			"b": {Soft: &soft},
			"a": {Hard: &hard},
		},
	})
	hc, err := sb.hostConfig()
	if err != nil {
		t.Fatalf("hostConfig() error: %v", err)
	}
	if !hc.ReadonlyRootfs {
		t.Fatal("expected readonly rootfs")
	}
	if hc.Resources.PidsLimit == nil || *hc.Resources.PidsLimit != 123 {
		t.Fatalf("unexpected pids limit: %#v", hc.Resources.PidsLimit)
	}
	if hc.Memory <= 0 || hc.MemorySwap <= 0 || hc.NanoCPUs <= 0 {
		t.Fatalf(
			"expected memory/swap/cpu limits set, got mem=%d swap=%d cpu=%d",
			hc.Memory,
			hc.MemorySwap,
			hc.NanoCPUs,
		)
	}
	if len(hc.Tmpfs) != 2 || hc.Tmpfs["/run"] != "" {
		t.Fatalf("unexpected tmpfs map: %#v", hc.Tmpfs)
	}
	if got := strings.Join(
		hc.SecurityOpt,
		",",
	); !strings.Contains(got, "seccomp=sec-profile.json") ||
		!strings.Contains(got, "apparmor=apparmor-profile") {
		t.Fatalf("security options missing expected profiles: %v", hc.SecurityOpt)
	}
	if len(hc.Resources.Ulimits) != 2 {
		t.Fatalf("expected 2 ulimits, got %d", len(hc.Resources.Ulimits))
	}
	gotNames := []string{hc.Resources.Ulimits[0].Name, hc.Resources.Ulimits[1].Name}
	sorted := append([]string{}, gotNames...)
	sort.Strings(sorted)
	if gotNames[0] != sorted[0] || gotNames[1] != sorted[1] {
		t.Fatalf("expected deterministic sorted ulimits, got %v", gotNames)
	}
}

func TestHostConfigRejectsInvalidMemorySettings(t *testing.T) {
	sb := NewContainerSandbox(ContainerSandboxConfig{
		Memory: "bad",
	})
	if _, err := sb.hostConfig(); err == nil || !strings.Contains(err.Error(), "invalid docker.memory") {
		t.Fatalf("expected invalid docker.memory error, got %v", err)
	}

	sb = NewContainerSandbox(ContainerSandboxConfig{
		MemorySwap: "bad",
	})
	if _, err := sb.hostConfig(); err == nil || !strings.Contains(err.Error(), "invalid docker.memory_swap") {
		t.Fatalf("expected invalid docker.memory_swap error, got %v", err)
	}
}

func TestBuildDockerUlimitVariants(t *testing.T) {
	if _, ok := buildDockerUlimit(" ", config.AgentSandboxDockerUlimitValue{}); ok {
		t.Fatal("expected empty-name ulimit to be rejected")
	}
	if _, ok := buildDockerUlimit("nofile", config.AgentSandboxDockerUlimitValue{}); ok {
		t.Fatal("expected empty ulimit value to be rejected")
	}

	soft := int64(10)
	ul, ok := buildDockerUlimit("nofile", config.AgentSandboxDockerUlimitValue{Soft: &soft})
	if !ok || ul == nil || ul.Soft != 10 || ul.Hard != 10 {
		t.Fatalf("expected soft-only to mirror hard, got %#v ok=%v", ul, ok)
	}

	hard := int64(20)
	ul, ok = buildDockerUlimit("nofile", config.AgentSandboxDockerUlimitValue{Hard: &hard})
	if !ok || ul == nil || ul.Soft != 20 || ul.Hard != 20 {
		t.Fatalf("expected hard-only to mirror soft, got %#v ok=%v", ul, ok)
	}
}

func TestContainerHelpers(t *testing.T) {
	if got := shellEscape("a'b"); got != "'a'\"'\"'b'" {
		t.Fatalf("shellEscape() got %q", got)
	}
	if osTempDir() == "" {
		t.Fatal("osTempDir() should not be empty")
	}

	sb := NewContainerSandbox(ContainerSandboxConfig{SetupCommand: "   "})
	if err := sb.runSetupCommand(context.Background()); err != nil {
		t.Fatalf("runSetupCommand() empty command should be nil, got %v", err)
	}
}

func TestWaitExecDoneContextCancel(t *testing.T) {
	sb := NewContainerSandbox(ContainerSandboxConfig{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	code, err := sb.waitExecDone(ctx, "unused")
	if err == nil {
		t.Fatal("expected context cancellation error")
	}
	if code != 1 {
		t.Fatalf("unexpected exit code: %d", code)
	}
}

func TestContainerSandbox_StopWithoutClient(t *testing.T) {
	sb := NewContainerSandbox(ContainerSandboxConfig{
		PruneIdleHours:  1,
		PruneMaxAgeDays: 1,
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := sb.Prune(ctx); err != nil {
		t.Fatalf("Prune() error: %v", err)
	}
}

func TestNewContainerSandbox_SanitizesEnv(t *testing.T) {
	sb := NewContainerSandbox(ContainerSandboxConfig{
		Env: map[string]string{
			"LANG":           "C.UTF-8",
			"OPENAI_API_KEY": "secret",
			"SAFE_NAME":      "ok",
		},
	})

	got := sb.containerEnv()
	joined := strings.Join(got, "\n")
	if strings.Contains(joined, "OPENAI_API_KEY=") {
		t.Fatalf("sensitive env key should be filtered, got: %v", got)
	}
	if !strings.Contains(joined, "SAFE_NAME=ok") {
		t.Fatalf("safe env key should be preserved, got: %v", got)
	}
	if !strings.Contains(joined, "LANG=C.UTF-8") {
		t.Fatalf("LANG should be preserved or defaulted, got: %v", got)
	}
}

func TestSyncAgentWorkspace_SeedsFilesAndPreservesExisting(t *testing.T) {
	agentWs := t.TempDir()
	containerWs := t.TempDir()

	// Setup agent workspace with seed files
	agentAgentsFile := filepath.Join(agentWs, "AGENTS.md")
	if err := os.WriteFile(agentAgentsFile, []byte("agent content"), 0o644); err != nil {
		t.Fatalf("failed to create agent AGENTS.md: %v", err)
	}

	agentUserFile := filepath.Join(agentWs, "USER.md")
	if err := os.WriteFile(agentUserFile, []byte("user content"), 0o644); err != nil {
		t.Fatalf("failed to create agent USER.md: %v", err)
	}

	// Setup container workspace with PRE-EXISTING AGENTS.md (should not be overwritten)
	containerAgentsFile := filepath.Join(containerWs, "AGENTS.md")
	if err := os.WriteFile(containerAgentsFile, []byte("PRESERVED CONTENT"), 0o644); err != nil {
		t.Fatalf("failed to create container AGENTS.md: %v", err)
	}

	// Run Sync
	if err := syncAgentWorkspace(agentWs, containerWs); err != nil {
		t.Fatalf("syncAgentWorkspace failed: %v", err)
	}

	// Verify existing file was preserved
	content, err := os.ReadFile(containerAgentsFile)
	if err != nil {
		t.Fatalf("failed to read container AGENTS.md: %v", err)
	}
	if string(content) != "PRESERVED CONTENT" {
		t.Fatalf("existing file was overwritten. expected PRESERVED CONTENT, got: %s", string(content))
	}

	// Verify missing file was seeded
	content, err = os.ReadFile(filepath.Join(containerWs, "USER.md"))
	if err != nil {
		t.Fatalf("failed to read container USER.md: %v", err)
	}
	if string(content) != "user content" {
		t.Fatalf("missing file was not seeded correctly. got: %s", string(content))
	}

	// Verify non-existent seed files are handled cleanly (TOOLS.md, MEMORY.md)
	if _, err := os.Stat(filepath.Join(containerWs, "MEMORY.md")); !os.IsNotExist(err) {
		t.Fatalf("expected MEMORY.md to not exist, got: %v", err)
	}
}

func TestSyncAgentWorkspace_SyncsSkillsDirectory(t *testing.T) {
	agentWs := t.TempDir()
	containerWs := t.TempDir()

	// Setup agent workspace with skills
	agentSkillsDir := filepath.Join(agentWs, "skills")
	if err := os.MkdirAll(agentSkillsDir, 0o755); err != nil {
		t.Fatalf("failed to create agent skills dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentSkillsDir, "skill1.txt"), []byte("new skill"), 0o644); err != nil {
		t.Fatalf("failed to create skill1: %v", err)
	}

	// Setup container workspace with OLD skills directory that should be overwritten
	containerSkillsDir := filepath.Join(containerWs, "skills")
	if err := os.MkdirAll(containerSkillsDir, 0o755); err != nil {
		t.Fatalf("failed to create container skills dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(containerSkillsDir, "old-skill.txt"), []byte("old skill"), 0o644); err != nil {
		t.Fatalf("failed to create old skill: %v", err)
	}

	// Run Sync
	if err := syncAgentWorkspace(agentWs, containerWs); err != nil {
		t.Fatalf("syncAgentWorkspace failed: %v", err)
	}

	// Verify old skills are gone and new skills are present
	if _, err := os.Stat(filepath.Join(containerSkillsDir, "old-skill.txt")); !os.IsNotExist(err) {
		t.Fatalf("old skill file was not removed during sync")
	}

	content, err := os.ReadFile(filepath.Join(containerSkillsDir, "skill1.txt"))
	if err != nil {
		t.Fatalf("failed to read synced skill: %v", err)
	}
	if string(content) != "new skill" {
		t.Fatalf("skill file content mismatch. got: %s", string(content))
	}
}

func TestContainerSandbox_Resolve(t *testing.T) {
	c := NewContainerSandbox(ContainerSandboxConfig{})
	sb, err := c.Resolve(context.Background())
	if err != nil || sb != c {
		t.Fatal("expected Resolve to return self")
	}
}

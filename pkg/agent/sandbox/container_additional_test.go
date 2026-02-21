package sandbox

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
)

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
		t.Fatalf("expected memory/swap/cpu limits set, got mem=%d swap=%d cpu=%d", hc.Memory, hc.MemorySwap, hc.NanoCPUs)
	}
	if len(hc.Tmpfs) != 2 || hc.Tmpfs["/run"] != "" {
		t.Fatalf("unexpected tmpfs map: %#v", hc.Tmpfs)
	}
	if got := strings.Join(hc.SecurityOpt, ","); !strings.Contains(got, "seccomp=sec-profile.json") || !strings.Contains(got, "apparmor=apparmor-profile") {
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

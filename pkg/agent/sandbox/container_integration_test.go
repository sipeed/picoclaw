package sandbox

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

func TestContainerSandbox_Integration_ExecReadWrite(t *testing.T) {
	if os.Getenv("PICOCLAW_RUN_DOCKER_TESTS") != "1" {
		t.Skip("set PICOCLAW_RUN_DOCKER_TESTS=1 to run docker integration tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		t.Skipf("docker client unavailable: %v", err)
	}
	defer cli.Close()

	if _, err := cli.Ping(ctx); err != nil {
		t.Skipf("docker daemon unavailable: %v", err)
	}

	workspace := t.TempDir()
	containerName := fmt.Sprintf("picoclaw-test-%d", time.Now().UnixNano())
	image := strings.TrimSpace(os.Getenv("PICOCLAW_DOCKER_TEST_IMAGE"))
	if image == "" {
		image = "debian:bookworm-slim"
	}

	sb := NewContainerSandbox(ContainerSandboxConfig{
		Image:         image,
		ContainerName: containerName,
		Workspace:     workspace,
	})
	if err := sb.Start(ctx); err != nil {
		t.Fatalf("sandbox start failed: %v", err)
	}
	defer func() {
		_ = sb.Prune(context.Background())
		if sb.cli != nil {
			_ = sb.cli.ContainerRemove(context.Background(), containerName, container.RemoveOptions{Force: true})
		}
	}()

	content := []byte("hello from integration test")
	if err := sb.Fs().WriteFile(ctx, "it/write.txt", content, true); err != nil {
		t.Fatalf("write file failed: %v", err)
	}

	readBack, err := sb.Fs().ReadFile(ctx, "it/write.txt")
	if err != nil {
		t.Fatalf("read file failed: %v", err)
	}
	if string(readBack) != string(content) {
		t.Fatalf("read content mismatch: got %q want %q", string(readBack), string(content))
	}

	hostBytes, err := os.ReadFile(filepath.Join(workspace, "it", "write.txt"))
	if err != nil {
		t.Fatalf("host workspace read failed: %v", err)
	}
	if string(hostBytes) != string(content) {
		t.Fatalf("host content mismatch: got %q want %q", string(hostBytes), string(content))
	}

	execRes, err := sb.Exec(ctx, ExecRequest{
		Command: "cat /workspace/it/write.txt",
	})
	if err != nil {
		t.Fatalf("exec cat failed: %v", err)
	}
	if execRes.ExitCode != 0 {
		t.Fatalf("exec cat exit code = %d, stderr = %q", execRes.ExitCode, execRes.Stderr)
	}
	if strings.TrimSpace(execRes.Stdout) != string(content) {
		t.Fatalf("exec cat stdout mismatch: got %q want %q", strings.TrimSpace(execRes.Stdout), string(content))
	}

	pwdRes, err := sb.Exec(ctx, ExecRequest{
		Command:    "pwd",
		WorkingDir: "it/",
	})
	if err != nil {
		t.Fatalf("exec pwd failed: %v", err)
	}
	if pwdRes.ExitCode != 0 {
		t.Fatalf("exec pwd exit code = %d, stderr = %q", pwdRes.ExitCode, pwdRes.Stderr)
	}
	if strings.TrimSpace(pwdRes.Stdout) != "/workspace/it" {
		t.Fatalf("pwd mismatch: got %q want %q", strings.TrimSpace(pwdRes.Stdout), "/workspace/it")
	}
}

func TestContainerSandbox_Integration_WriteFileMkdirInContainerTmp(t *testing.T) {
	if os.Getenv("PICOCLAW_RUN_DOCKER_TESTS") != "1" {
		t.Skip("set PICOCLAW_RUN_DOCKER_TESTS=1 to run docker integration tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		t.Skipf("docker client unavailable: %v", err)
	}
	defer cli.Close()

	if _, err := cli.Ping(ctx); err != nil {
		t.Skipf("docker daemon unavailable: %v", err)
	}

	containerName := fmt.Sprintf("picoclaw-test-mkdir-%d", time.Now().UnixNano())
	image := strings.TrimSpace(os.Getenv("PICOCLAW_DOCKER_TEST_IMAGE"))
	if image == "" {
		image = "debian:bookworm-slim"
	}

	sb := NewContainerSandbox(ContainerSandboxConfig{
		Image:         image,
		ContainerName: containerName,
	})
	if err := sb.Start(ctx); err != nil {
		t.Fatalf("sandbox start failed: %v", err)
	}
	defer func() {
		_ = sb.Prune(context.Background())
		if sb.cli != nil {
			_ = sb.cli.ContainerRemove(context.Background(), containerName, container.RemoveOptions{Force: true})
		}
	}()

	content := []byte("mkdir path works")
	if err := sb.Fs().WriteFile(ctx, "/workspace/it_mkdir/nested/file.txt", content, true); err != nil {
		t.Fatalf("write with mkdir failed: %v", err)
	}

	out, err := sb.Exec(ctx, ExecRequest{
		Command: "cat /workspace/it_mkdir/nested/file.txt",
	})
	if err != nil {
		t.Fatalf("exec cat failed: %v", err)
	}
	if out.ExitCode != 0 {
		t.Fatalf("exec cat exit code = %d, stderr = %q", out.ExitCode, out.Stderr)
	}
	if strings.TrimSpace(out.Stdout) != string(content) {
		t.Fatalf("exec cat stdout mismatch: got %q want %q", strings.TrimSpace(out.Stdout), string(content))
	}
}

func TestContainerSandbox_Integration_SetupCommandSuccess(t *testing.T) {
	if os.Getenv("PICOCLAW_RUN_DOCKER_TESTS") != "1" {
		t.Skip("set PICOCLAW_RUN_DOCKER_TESTS=1 to run docker integration tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	containerName := fmt.Sprintf("picoclaw-test-setup-ok-%d", time.Now().UnixNano())
	image := strings.TrimSpace(os.Getenv("PICOCLAW_DOCKER_TEST_IMAGE"))
	if image == "" {
		image = "debian:bookworm-slim"
	}

	sb := NewContainerSandbox(ContainerSandboxConfig{
		Image:         image,
		ContainerName: containerName,
		Workspace:     t.TempDir(),
		SetupCommand:  "true",
	})
	if err := sb.Start(ctx); err != nil {
		t.Fatalf("sandbox start failed: %v", err)
	}
	defer func() {
		_ = sb.Prune(context.Background())
		if sb.cli != nil {
			_ = sb.cli.ContainerRemove(context.Background(), containerName, container.RemoveOptions{Force: true})
		}
	}()

	out, err := sb.Exec(ctx, ExecRequest{
		Command: "echo setup-ok",
	})
	if err != nil {
		t.Fatalf("exec after setup_command failed: %v", err)
	}
	if out.ExitCode != 0 {
		t.Fatalf("unexpected exit code=%d stderr=%q", out.ExitCode, out.Stderr)
	}
	if strings.TrimSpace(out.Stdout) != "setup-ok" {
		t.Fatalf("unexpected setup content: %q", out.Stdout)
	}
}

func TestContainerSandbox_Integration_SetupCommandFailureRemovesContainer(t *testing.T) {
	if os.Getenv("PICOCLAW_RUN_DOCKER_TESTS") != "1" {
		t.Skip("set PICOCLAW_RUN_DOCKER_TESTS=1 to run docker integration tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	containerName := fmt.Sprintf("picoclaw-test-setup-fail-%d", time.Now().UnixNano())
	image := strings.TrimSpace(os.Getenv("PICOCLAW_DOCKER_TEST_IMAGE"))
	if image == "" {
		image = "debian:bookworm-slim"
	}

	sb := NewContainerSandbox(ContainerSandboxConfig{
		Image:         image,
		ContainerName: containerName,
		Workspace:     t.TempDir(),
		SetupCommand:  "echo boom >&2; exit 7",
	})
	if err := sb.Start(ctx); err != nil {
		t.Fatalf("sandbox start failed: %v", err)
	}
	defer func() {
		_ = sb.Prune(context.Background())
		if sb.cli != nil {
			_ = sb.cli.ContainerRemove(context.Background(), containerName, container.RemoveOptions{Force: true})
		}
	}()

	_, err := sb.Exec(ctx, ExecRequest{Command: "echo never"})
	if err == nil || !strings.Contains(err.Error(), "setup_command failed") {
		t.Fatalf("expected setup_command failed error, got: %v", err)
	}

	_, inspectErr := sb.cli.ContainerInspect(ctx, containerName)
	if inspectErr == nil {
		t.Fatal("expected failed setup to remove container, but container still exists")
	}
}

func TestContainerSandbox_Integration_MaybePruneRemovesOldContainer(t *testing.T) {
	if os.Getenv("PICOCLAW_RUN_DOCKER_TESTS") != "1" {
		t.Skip("set PICOCLAW_RUN_DOCKER_TESTS=1 to run docker integration tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	root := t.TempDir()
	containerName := fmt.Sprintf("picoclaw-test-prune-%d", time.Now().UnixNano())
	image := strings.TrimSpace(os.Getenv("PICOCLAW_DOCKER_TEST_IMAGE"))
	if image == "" {
		image = "debian:bookworm-slim"
	}

	sb := NewContainerSandbox(ContainerSandboxConfig{
		Image:           image,
		ContainerName:   containerName,
		WorkspaceRoot:   root,
		WorkspaceAccess: "none",
		PruneIdleHours:  1,
		PruneMaxAgeDays: 0,
	})
	if err := sb.Start(ctx); err != nil {
		t.Fatalf("sandbox start failed: %v", err)
	}
	defer func() {
		_ = sb.Prune(context.Background())
		if sb.cli != nil {
			_ = sb.cli.ContainerRemove(context.Background(), containerName, container.RemoveOptions{Force: true})
		}
	}()

	if _, err := sb.Exec(ctx, ExecRequest{Command: "echo alive"}); err != nil {
		t.Fatalf("exec create failed: %v", err)
	}

	now := time.Now().UnixMilli()
	if err := upsertRegistryEntry(sb.registryPath(), registryEntry{
		ContainerName: containerName,
		Image:         image,
		ConfigHash:    sb.hash,
		CreatedAtMs:   now - int64(2*time.Hour/time.Millisecond),
		LastUsedAtMs:  now - int64(2*time.Hour/time.Millisecond),
	}); err != nil {
		t.Fatalf("upsert old registry entry failed: %v", err)
	}

	manager := &scopedSandboxManager{
		pruneIdleHours:  1,
		pruneMaxAgeDays: 0,
		scoped: map[string]Sandbox{
			"agent:main": sb,
		},
	}
	if err := manager.pruneOnce(ctx); err != nil {
		t.Fatalf("pruneOnce failed: %v", err)
	}

	if _, err := sb.cli.ContainerInspect(ctx, containerName); err == nil {
		t.Fatal("expected container to be removed by prune")
	}
	data, err := loadRegistry(sb.registryPath())
	if err != nil {
		t.Fatalf("loadRegistry failed: %v", err)
	}
	for _, e := range data.Entries {
		if e.ContainerName == containerName {
			t.Fatal("expected pruned container to be removed from registry")
		}
	}
}

func TestContainerSandbox_Integration_ExecTimeoutRespectsRequest(t *testing.T) {
	if os.Getenv("PICOCLAW_RUN_DOCKER_TESTS") != "1" {
		t.Skip("set PICOCLAW_RUN_DOCKER_TESTS=1 to run docker integration tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	containerName := fmt.Sprintf("picoclaw-test-timeout-%d", time.Now().UnixNano())
	image := strings.TrimSpace(os.Getenv("PICOCLAW_DOCKER_TEST_IMAGE"))
	if image == "" {
		image = "debian:bookworm-slim"
	}

	sb := NewContainerSandbox(ContainerSandboxConfig{
		Image:         image,
		ContainerName: containerName,
		Workspace:     t.TempDir(),
	})
	if err := sb.Start(ctx); err != nil {
		t.Fatalf("sandbox start failed: %v", err)
	}
	defer func() {
		_ = sb.Prune(context.Background())
		if sb.cli != nil {
			_ = sb.cli.ContainerRemove(context.Background(), containerName, container.RemoveOptions{Force: true})
		}
	}()

	start := time.Now()
	_, err := sb.Exec(ctx, ExecRequest{
		Command:   "sleep 3",
		TimeoutMs: 200,
	})
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if time.Since(start) > 2*time.Second {
		t.Fatalf("expected timeout to trigger early, took %v", time.Since(start))
	}
}

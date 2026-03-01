package sandbox

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/client"
)

// skipIfNoDocker checks if a Docker daemon is available and skips the test if not.
// It returns a functional client and a cleanup function if successful.
func skipIfNoDocker(t *testing.T) (*client.Client, func()) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		t.Skipf("Docker client setup failed: %v", err)
	}

	_, err = cli.Ping(ctx)
	if err != nil {
		cli.Close()
		t.Skip("Docker daemon unavailable (ping failed), skipping integration test")
	}

	return cli, func() { cli.Close() }
}

func getTestImage() string {
	image := strings.TrimSpace(os.Getenv("PICOCLAW_DOCKER_TEST_IMAGE"))
	if image == "" {
		image = "debian:bookworm-slim"
	}
	return image
}

func TestContainerSandbox_Integration_ExecReadWrite(t *testing.T) {
	_, cleanup := skipIfNoDocker(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	workspace := t.TempDir()
	containerName := fmt.Sprintf("picoclaw-test-%d", time.Now().UnixNano())
	image := getTestImage()

	sb := NewContainerSandbox(ContainerSandboxConfig{
		Image:         image,
		ContainerName: containerName,
		Workspace:     workspace,
		User:          fmt.Sprintf("%d:%d", os.Getuid(), os.Getgid()),
	})
	err := sb.Start(ctx)
	if err != nil {
		t.Fatalf("sandbox start failed: %v", err)
	}
	defer sb.Prune(ctx)

	// 1. Write file via FsBridge
	testData := []byte("hello from host")
	err = sb.Fs().WriteFile(ctx, "hello.txt", testData, false)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// 2. Read back via Exec (command line)
	res, err := sb.Exec(ctx, ExecRequest{
		Command: "cat hello.txt",
	})
	if err != nil {
		t.Fatalf("Exec failed: %v", err)
	}
	if strings.TrimSpace(res.Stdout) != string(testData) {
		t.Errorf("Exec output mismatch: got %q, want %q", res.Stdout, string(testData))
	}

	// 3. Write via Exec
	res, err = sb.Exec(ctx, ExecRequest{
		Command: "echo 'modified in container' > hello.txt",
	})
	if err != nil || res.ExitCode != 0 {
		t.Fatalf("Exec write failed: %v, exit=%d", err, res.ExitCode)
	}

	// 4. Read back via FsBridge
	readData, err := sb.Fs().ReadFile(ctx, "hello.txt")
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if strings.TrimSpace(string(readData)) != "modified in container" {
		t.Errorf("ReadFile output mismatch: got %q", string(readData))
	}

	// 5. Verify ReadDir
	entries, err := sb.Fs().ReadDir(ctx, ".")
	if err != nil {
		t.Fatalf("ReadDir failed: %v", err)
	}
	found := false
	for _, e := range entries {
		if e.Name() == "hello.txt" {
			found = true
			break
		}
	}
	if !found {
		t.Error("hello.txt not found in ReadDir")
	}
}

func TestContainerSandbox_Integration_WriteFileOwnership(t *testing.T) {
	_, cleanup := skipIfNoDocker(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	workspace := t.TempDir()
	containerName := fmt.Sprintf("picoclaw-test-ownership-%d", time.Now().UnixNano())
	image := getTestImage()

	// Use the current host user's UID:GID to test real-world mapping compatibility
	testUser := fmt.Sprintf("%d:%d", os.Getuid(), os.Getgid())

	sb := NewContainerSandbox(ContainerSandboxConfig{
		Image:         image,
		ContainerName: containerName,
		Workspace:     workspace,
		User:          testUser,
	})
	if err := sb.Start(ctx); err != nil {
		t.Fatalf("sandbox start failed: %v", err)
	}
	defer sb.Prune(ctx)

	// Write file via FsBridge with mkdir enabled
	testPath := "auth/test.txt"
	content := []byte("ownership verification")
	if err := sb.Fs().WriteFile(ctx, testPath, content, true); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Verify identity (UID:GID) and modification time via stat
	res, err := sb.Exec(ctx, ExecRequest{
		Command: fmt.Sprintf("stat -c '%%u:%%g %%Y' %s", testPath),
	})
	if err != nil || res.ExitCode != 0 {
		t.Fatalf("Stat failed: %v, stderr=%s", err, res.Stderr)
	}

	// Output format: "UID:GID UnixTimestamp"
	parts := strings.Fields(res.Stdout)
	if len(parts) < 2 {
		t.Fatalf("unexpected stat output: %q", res.Stdout)
	}

	// Check UID:GID
	if parts[0] != testUser {
		t.Errorf("ownership mismatch: got %q, want %q", parts[0], testUser)
	}

	// Check Timestamp (should not be 1970 Epoch)
	timestamp := parts[1]
	if strings.HasPrefix(timestamp, "0") || strings.HasPrefix(timestamp, "1 ") {
		// Specifically check for very small timestamps that indicate 1970
		t.Errorf("file created with suspicious 1970-era timestamp: %q", timestamp)
	}

	t.Logf("Stat verified: %s", res.Stdout)
}

func TestContainerSandbox_Integration_WriteFileMkdirInContainerTmp(t *testing.T) {
	_, cleanup := skipIfNoDocker(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	containerName := fmt.Sprintf("picoclaw-test-mkdir-%d", time.Now().UnixNano())
	image := getTestImage()
	workspace := t.TempDir()

	sb := NewContainerSandbox(ContainerSandboxConfig{
		Image:         image,
		ContainerName: containerName,
		Workspace:     workspace,
		User:          fmt.Sprintf("%d:%d", os.Getuid(), os.Getgid()),
	})
	err := sb.Start(ctx)
	if err != nil {
		t.Fatalf("sandbox start failed: %v", err)
	}
	defer sb.Prune(ctx)

	// Write to a directory that definitely doesn't exist in the container (under /workspace)
	// This forces FsBridge to use Exec fallback for mkdir -p.
	testPath := "/workspace/a/b/c/test.txt"
	content := []byte("mkdir test")
	err = sb.Fs().WriteFile(ctx, testPath, content, true)
	if err != nil {
		t.Fatalf("WriteFile with mkdir failed: %v", err)
	}

	// Verify it exists
	res, err := sb.Exec(ctx, ExecRequest{Command: "cat " + testPath})
	if err != nil || res.ExitCode != 0 {
		t.Fatalf("Verify cat failed: %v", err)
	}
	if strings.TrimSpace(res.Stdout) != string(content) {
		t.Errorf("cat mismatch: got %q", res.Stdout)
	}
}

func TestContainerSandbox_Integration_SetupCommandSuccess(t *testing.T) {
	_, cleanup := skipIfNoDocker(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	containerName := fmt.Sprintf("picoclaw-test-setup-ok-%d", time.Now().UnixNano())
	image := getTestImage()

	sb := NewContainerSandbox(ContainerSandboxConfig{
		Image:         image,
		ContainerName: containerName,
		SetupCommand:  "touch /tmp/setup_done",
	})
	if err := sb.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer sb.Prune(ctx)

	res, err := sb.Exec(ctx, ExecRequest{Command: "ls /tmp/setup_done"})
	if err != nil || res.ExitCode != 0 {
		t.Errorf("Setup command didn't run or fail to create file: %v", err)
	}
}

func TestContainerSandbox_Integration_SetupCommandFailureRemovesContainer(t *testing.T) {
	_, cleanup := skipIfNoDocker(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	containerName := fmt.Sprintf("picoclaw-test-setup-fail-%d", time.Now().UnixNano())
	image := getTestImage()

	sb := NewContainerSandbox(ContainerSandboxConfig{
		Image:         image,
		ContainerName: containerName,
		SetupCommand:  "false", // Force setup to fail
	})
	if err := sb.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Trigger container creation and setup command execution
	_, err := sb.Exec(ctx, ExecRequest{Command: "echo test"})
	if err == nil {
		t.Fatal("expected Exec to fail due to failing setup_command")
	}

	// Verify container was removed by the error handler in createAndStart
	cli, _ := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	defer cli.Close()
	_, err = cli.ContainerInspect(ctx, containerName)
	if !client.IsErrNotFound(err) {
		t.Errorf("expected container to be removed after failed setup, got err: %v", err)
	}
}

func TestContainerSandbox_Integration_MaybePruneRemovesOldContainer(t *testing.T) {
	_, cleanup := skipIfNoDocker(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	root := t.TempDir()
	containerName := fmt.Sprintf("picoclaw-test-prune-%d", time.Now().UnixNano())
	image := getTestImage()

	sb := NewContainerSandbox(ContainerSandboxConfig{
		Image:           image,
		ContainerName:   containerName,
		Workspace:       root,
		PruneMaxAgeDays: -1, // Force immediate prune eligibility
	})

	if err := sb.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Trigger container creation
	if _, err := sb.Exec(ctx, ExecRequest{Command: "echo test"}); err != nil {
		t.Fatalf("Exec failed: %v", err)
	}

	cli, _ := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	defer cli.Close()

	_, err := cli.ContainerInspect(ctx, containerName)
	if err != nil {
		t.Fatalf("container missing before prune: %v", err)
	}

	// Explicitly call Prune (scoped manager would normally do this in loop)
	if pruneErr := sb.Prune(ctx); pruneErr != nil {
		t.Fatalf("Prune failed: %v", pruneErr)
	}

	// Verify container is gone
	_, err = cli.ContainerInspect(ctx, containerName)
	if !client.IsErrNotFound(err) {
		t.Errorf("expected container gone after prune, got err: %v", err)
	}
}

func TestContainerSandbox_Integration_ExecTimeoutRespectsRequest(t *testing.T) {
	_, cleanup := skipIfNoDocker(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	containerName := fmt.Sprintf("picoclaw-test-timeout-%d", time.Now().UnixNano())
	image := getTestImage()

	sb := NewContainerSandbox(ContainerSandboxConfig{
		Image:         image,
		ContainerName: containerName,
	})
	if err := sb.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer sb.Prune(ctx)

	start := time.Now()
	_, err := sb.Exec(ctx, ExecRequest{
		Command:   "sleep 10",
		TimeoutMs: 100, // Very short timeout
	})
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected timeout error")
	}
	if elapsed > 2*time.Second {
		t.Errorf("Exec took too long to time out: %v", elapsed)
	}
}

func TestContainerSandbox_Integration_ExecTimeoutBreaksStdCopyBlock(t *testing.T) {
	_, cleanup := skipIfNoDocker(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	containerName := fmt.Sprintf("picoclaw-test-timeout-block-%d", time.Now().UnixNano())
	image := getTestImage()

	sb := NewContainerSandbox(ContainerSandboxConfig{
		Image:         image,
		ContainerName: containerName,
	})
	if err := sb.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer sb.Prune(ctx)

	// Ensure container is started so the timeout only applies to command execution
	if _, err := sb.Exec(ctx, ExecRequest{Command: "true"}); err != nil {
		t.Fatalf("warmup failed: %v", err)
	}

	// Simulate a command that hangs and might block output readers
	start := time.Now()
	_, err := sb.Exec(ctx, ExecRequest{
		Command:   "sleep 10", // Guaranteed to hang
		TimeoutMs: 500,
	})
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected timeout error for hanging command")
	}
	if elapsed > 3*time.Second {
		t.Errorf("Exec took too long to break block: %v", elapsed)
	}
}

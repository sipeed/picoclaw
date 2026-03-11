package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/agent/sandbox"
	"github.com/sipeed/picoclaw/pkg/config"
)

type stubSandbox struct {
	lastReq   sandbox.ExecRequest
	err       error
	res       *sandbox.ExecResult
	fs        sandbox.FsBridge
	workspace string
}

func (s *stubSandbox) Start(ctx context.Context) error { return nil }
func (s *stubSandbox) Prune(ctx context.Context) error { return nil }
func (s *stubSandbox) Resolve(ctx context.Context) (sandbox.Sandbox, error) {
	return s, nil
}

func (s *stubSandbox) GetWorkspace(ctx context.Context) string {
	return s.workspace
}

func (s *stubSandbox) Fs() sandbox.FsBridge {
	if s.fs != nil {
		return s.fs
	}
	return sandbox.NewHostSandbox("", false).Fs()
}

func (s *stubSandbox) Exec(ctx context.Context, req sandbox.ExecRequest) (*sandbox.ExecResult, error) {
	return sandboxAggregateFromStub(ctx, req, s.ExecStream)
}

func (s *stubSandbox) ExecStream(
	ctx context.Context,
	req sandbox.ExecRequest,
	onEvent func(sandbox.ExecEvent) error,
) (*sandbox.ExecResult, error) {
	s.lastReq = req
	if s.err != nil {
		return nil, s.err
	}
	if s.res != nil {
		if onEvent != nil {
			if s.res.Stdout != "" {
				if err := onEvent(
					sandbox.ExecEvent{Type: sandbox.ExecEventStdout, Chunk: []byte(s.res.Stdout)},
				); err != nil {
					return nil, err
				}
			}
			if s.res.Stderr != "" {
				if err := onEvent(
					sandbox.ExecEvent{Type: sandbox.ExecEventStderr, Chunk: []byte(s.res.Stderr)},
				); err != nil {
					return nil, err
				}
			}
			if err := onEvent(sandbox.ExecEvent{Type: sandbox.ExecEventExit, ExitCode: s.res.ExitCode}); err != nil {
				return nil, err
			}
		}
		return s.res, nil
	}
	if onEvent != nil {
		if err := onEvent(sandbox.ExecEvent{Type: sandbox.ExecEventStdout, Chunk: []byte("ok")}); err != nil {
			return nil, err
		}
		if err := onEvent(sandbox.ExecEvent{Type: sandbox.ExecEventExit, ExitCode: 0}); err != nil {
			return nil, err
		}
	}
	return &sandbox.ExecResult{Stdout: "ok", ExitCode: 0}, nil
}

func sandboxAggregateFromStub(
	ctx context.Context,
	req sandbox.ExecRequest,
	streamFn func(context.Context, sandbox.ExecRequest, func(sandbox.ExecEvent) error) (*sandbox.ExecResult, error),
) (*sandbox.ExecResult, error) {
	var stdout strings.Builder
	var stderr strings.Builder
	exitCode := 0
	res, err := streamFn(ctx, req, func(event sandbox.ExecEvent) error {
		switch event.Type {
		case sandbox.ExecEventStdout:
			_, _ = stdout.Write(event.Chunk)
		case sandbox.ExecEventStderr:
			_, _ = stderr.Write(event.Chunk)
		case sandbox.ExecEventExit:
			exitCode = event.ExitCode
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if res != nil {
		return res, nil
	}
	return &sandbox.ExecResult{Stdout: stdout.String(), Stderr: stderr.String(), ExitCode: exitCode}, nil
}

func mustNewExecTool(t *testing.T, workingDir string, restrict bool) *ExecTool {
	t.Helper()
	tool, err := NewExecTool(workingDir, restrict)
	if err != nil {
		t.Fatalf("unable to configure exec tool: %v", err)
	}
	return tool
}

func TestShellTool_Success(t *testing.T) {
	tool := mustNewExecTool(t, "", false)
	ctx := sandbox.WithSandbox(context.Background(), &stubSandbox{
		res: &sandbox.ExecResult{Stdout: "hello world", ExitCode: 0},
	})
	args := map[string]any{"command": "echo 'hello world'"}
	result := tool.Execute(ctx, args)
	if result.IsError {
		t.Errorf("Expected success, got error: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForUser, "hello world") {
		t.Errorf("Expected ForUser to contain 'hello world', got: %s", result.ForUser)
	}
}

func TestShellTool_Failure(t *testing.T) {
	tool := mustNewExecTool(t, "", false)
	ctx := sandbox.WithSandbox(context.Background(), &stubSandbox{
		res: &sandbox.ExecResult{Stderr: "error", ExitCode: 2},
	})
	args := map[string]any{"command": "ls /fail"}
	result := tool.Execute(ctx, args)
	if !result.IsError {
		t.Errorf("Expected error, got success")
	}
}

// TestShellTool_Timeout verifies command timeout handling
func TestShellTool_Timeout(t *testing.T) {
	tool := mustNewExecTool(t, "", false)
	tool.SetTimeout(100 * time.Millisecond)
	ctx := sandbox.WithSandbox(context.Background(), &stubSandbox{err: context.DeadlineExceeded})
	args := map[string]any{
		"command": "sleep 10",
	}

	result := tool.Execute(ctx, args)

	// Timeout should be marked as error
	if !result.IsError {
		t.Errorf("Expected error for timeout, got IsError=false")
	}

	// Should mention timeout
	if !strings.Contains(result.ForLLM, "timed out") && !strings.Contains(result.ForUser, "timed out") {
		t.Errorf("Expected timeout message, got ForLLM: %s, ForUser: %s", result.ForLLM, result.ForUser)
	}
}

// TestShellTool_WorkingDir verifies custom working directory
func TestShellTool_WorkingDir(t *testing.T) {
	tmpDir := t.TempDir()
	tool := mustNewExecTool(t, "", false)
	ctx := sandbox.WithSandbox(context.Background(), &stubSandbox{
		workspace: tmpDir,
		res:       &sandbox.ExecResult{Stdout: "test content", ExitCode: 0},
	})
	args := map[string]any{
		"command":     "cat test.txt",
		"working_dir": tmpDir,
	}

	result := tool.Execute(ctx, args)

	if result.IsError {
		t.Errorf("Expected success in custom working dir, got error: %s", result.ForLLM)
	}

	if !strings.Contains(result.ForUser, "test content") {
		t.Errorf("Expected output from custom dir, got: %s", result.ForUser)
	}
}

// TestShellTool_DangerousCommand verifies safety guard blocks dangerous commands
func TestShellTool_DangerousCommand(t *testing.T) {
	tool := mustNewExecTool(t, "", false)
	ctx := sandbox.WithSandbox(context.Background(), &stubSandbox{})
	args := map[string]any{
		"command": "rm -rf /",
	}

	result := tool.Execute(ctx, args)

	// Dangerous command should be blocked
	if !result.IsError {
		t.Errorf("Expected dangerous command to be blocked (IsError=true)")
	}

	if !strings.Contains(result.ForLLM, "blocked") && !strings.Contains(result.ForUser, "blocked") {
		t.Errorf("Expected 'blocked' message, got ForLLM: %s, ForUser: %s", result.ForLLM, result.ForUser)
	}
}

func TestShellTool_DangerousCommand_KillBlocked(t *testing.T) {
	tool := mustNewExecTool(t, "", false)
	ctx := sandbox.WithSandbox(context.Background(), &stubSandbox{})
	args := map[string]any{
		"command": "kill 12345",
	}

	result := tool.Execute(ctx, args)
	if !result.IsError {
		t.Errorf("Expected kill command to be blocked")
	}
	if !strings.Contains(result.ForLLM, "blocked") && !strings.Contains(result.ForUser, "blocked") {
		t.Errorf("Expected blocked message, got ForLLM: %s, ForUser: %s", result.ForLLM, result.ForUser)
	}
}

// TestShellTool_MissingCommand verifies error handling for missing command
func TestShellTool_MissingCommand(t *testing.T) {
	tool := mustNewExecTool(t, "", false)
	ctx := sandbox.WithSandbox(context.Background(), &stubSandbox{})
	args := map[string]any{}

	result := tool.Execute(ctx, args)

	// Should return error result
	if !result.IsError {
		t.Errorf("Expected error when command is missing")
	}
}

// TestShellTool_StderrCapture verifies stderr is captured and included
func TestShellTool_StderrCapture(t *testing.T) {
	tool := mustNewExecTool(t, "", false)
	ctx := sandbox.WithSandbox(context.Background(), &stubSandbox{
		res: &sandbox.ExecResult{Stdout: "stdout", Stderr: "stderr", ExitCode: 0},
	})
	args := map[string]any{
		"command": "sh -c 'echo stdout; echo stderr >&2'",
	}

	result := tool.Execute(ctx, args)

	// Both stdout and stderr should be in output
	if !strings.Contains(result.ForLLM, "stdout") {
		t.Errorf("Expected stdout in output, got: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "stderr") {
		t.Errorf("Expected stderr in output, got: %s", result.ForLLM)
	}
}

// TestShellTool_OutputTruncation verifies long output is truncated
func TestShellTool_OutputTruncation(t *testing.T) {
	tool := mustNewExecTool(t, "", false)
	ctx := sandbox.WithSandbox(context.Background(), &stubSandbox{
		res: &sandbox.ExecResult{Stdout: strings.Repeat("x", 20000), ExitCode: 0},
	})
	args := map[string]any{
		"command": "echo large-output",
	}

	result := tool.Execute(ctx, args)

	// Should have truncation message or be truncated
	if len(result.ForLLM) > 15000 {
		t.Errorf("Expected output to be truncated, got length: %d", len(result.ForLLM))
	}
}

func TestShellTool_WorkingDir_OutsideWorkspace(t *testing.T) {
	root := t.TempDir()
	workspace := filepath.Join(root, "workspace")
	outsideDir := filepath.Join(root, "outside")
	os.MkdirAll(workspace, 0o755)
	os.MkdirAll(outsideDir, 0o755)

	tool := mustNewExecTool(t, workspace, true)
	ctx := sandbox.WithSandbox(context.Background(), &stubSandbox{
		workspace: workspace,
		res:       &sandbox.ExecResult{ExitCode: 0},
	})
	result := tool.Execute(ctx, map[string]any{
		"command":     "pwd",
		"working_dir": outsideDir,
	})

	if !result.IsError || !strings.Contains(result.ForLLM, "blocked") {
		t.Fatalf("expected blocked error, got: %s", result.ForLLM)
	}
}

func TestShellTool_WorkingDir_SymlinkEscape(t *testing.T) {
	root := t.TempDir()
	workspace := filepath.Join(root, "workspace")
	secretDir := filepath.Join(root, "secret")
	os.MkdirAll(workspace, 0o755)
	os.MkdirAll(secretDir, 0o755)
	os.WriteFile(filepath.Join(secretDir, "secret.txt"), []byte("top secret"), 0o644)

	link := filepath.Join(workspace, "escape")
	if err := os.Symlink(secretDir, link); err != nil {
		t.Skip("symlinks not supported")
	}

	tool := mustNewExecTool(t, workspace, true)
	ctx := sandbox.WithSandbox(context.Background(), &stubSandbox{
		workspace: workspace,
	})
	result := tool.Execute(ctx, map[string]any{
		"command":     "cat secret.txt",
		"working_dir": link,
	})

	if !result.IsError || !strings.Contains(result.ForLLM, "blocked") {
		t.Fatalf("expected blocked error, got: %s", result.ForLLM)
	}
}

func TestShellTool_RestrictToWorkspace(t *testing.T) {
	tmpDir := t.TempDir()
	tool := mustNewExecTool(t, tmpDir, true)
	ctx := sandbox.WithSandbox(context.Background(), &stubSandbox{
		workspace: tmpDir,
	})
	args := map[string]any{"command": "cat ../../etc/passwd"}
	result := tool.Execute(ctx, args)
	if !result.IsError || !strings.Contains(result.ForLLM, "blocked") {
		t.Errorf("Expected path traversal to be blocked")
	}
}

func TestShellTool_SandboxMapsHostWorkingDirToRelative(t *testing.T) {
	workspace := t.TempDir()
	sb := &stubSandbox{workspace: workspace}
	tool := mustNewExecTool(t, workspace, true)

	ctx := sandbox.WithSandbox(context.Background(), sb)
	args := map[string]any{
		"command":     "echo test",
		"working_dir": filepath.Join(workspace, "subdir"),
	}
	result := tool.Execute(ctx, args)
	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.ForLLM)
	}
	if sb.lastReq.WorkingDir != "subdir" {
		t.Fatalf("sandbox working_dir = %q, want subdir", sb.lastReq.WorkingDir)
	}
}

func TestShellTool_SandboxAllowsAbsoluteWorkspaceWorkingDir(t *testing.T) {
	workspace := "/workspace"
	sb := &stubSandbox{workspace: workspace}
	tool := mustNewExecTool(t, workspace, true)

	ctx := sandbox.WithSandbox(context.Background(), sb)
	args := map[string]any{
		"command":     "echo test",
		"working_dir": "/workspace/subdir",
	}
	result := tool.Execute(ctx, args)
	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.ForLLM)
	}
	if sb.lastReq.WorkingDir != "subdir" && sb.lastReq.WorkingDir != "/workspace/subdir" {
		t.Fatalf("sandbox working_dir = %q, want subdir or /workspace/subdir", sb.lastReq.WorkingDir)
	}
}

func TestShellTool_SandboxBlocksAbsoluteNonWorkspaceWorkingDirWhenRestricted(t *testing.T) {
	workspace := t.TempDir()
	sb := &stubSandbox{workspace: workspace}
	tool := mustNewExecTool(t, workspace, true)

	ctx := sandbox.WithSandbox(context.Background(), sb)
	args := map[string]any{
		"command":     "echo test",
		"working_dir": "/tmp/logs",
	}
	result := tool.Execute(ctx, args)
	if !result.IsError || !strings.Contains(result.ForLLM, "blocked") {
		t.Fatalf("expected blocked error, got: %s", result.ForLLM)
	}
}

// TestShellTool_DevNullAllowed verifies that /dev/null redirections are not blocked (issue #964).
func TestShellTool_DevNullAllowed(t *testing.T) {
	tmpDir := t.TempDir()
	tool, err := NewExecTool(tmpDir, true)
	if err != nil {
		t.Fatalf("unable to configure exec tool: %s", err)
	}

	commands := []string{
		"echo hello 2>/dev/null",
		"echo hello >/dev/null",
		"echo hello > /dev/null",
		"echo hello 2> /dev/null",
		"echo hello >/dev/null 2>&1",
		"find " + tmpDir + " -name '*.go' 2>/dev/null",
	}

	for _, cmd := range commands {
		result := tool.Execute(context.Background(), map[string]any{"command": cmd})
		if result.IsError && strings.Contains(result.ForLLM, "blocked") {
			t.Errorf("command should not be blocked: %s\n  error: %s", cmd, result.ForLLM)
		}
	}
}

// TestShellTool_BlockDevices verifies that writes to block devices are blocked (issue #965).
func TestShellTool_BlockDevices(t *testing.T) {
	tool, err := NewExecTool("", false)
	if err != nil {
		t.Fatalf("unable to configure exec tool: %s", err)
	}

	blocked := []string{
		"echo x > /dev/sda",
		"echo x > /dev/hda",
		"echo x > /dev/vda",
		"echo x > /dev/xvda",
		"echo x > /dev/nvme0n1",
		"echo x > /dev/mmcblk0",
		"echo x > /dev/loop0",
		"echo x > /dev/dm-0",
		"echo x > /dev/md0",
		"echo x > /dev/sr0",
		"echo x > /dev/nbd0",
	}

	for _, cmd := range blocked {
		result := tool.Execute(context.Background(), map[string]any{"command": cmd})
		if !result.IsError {
			t.Errorf("expected block device write to be blocked: %s", cmd)
		}
	}
}

// TestShellTool_SafePathsInWorkspaceRestriction verifies that safe kernel pseudo-devices
// are allowed even when workspace restriction is active.
func TestShellTool_SafePathsInWorkspaceRestriction(t *testing.T) {
	tmpDir := t.TempDir()
	tool, err := NewExecTool(tmpDir, true)
	if err != nil {
		t.Fatalf("unable to configure exec tool: %s", err)
	}

	// These reference paths outside workspace but should be allowed via safePaths.
	commands := []string{
		"cat /dev/urandom | head -c 16 | od",
		"echo test > /dev/null",
		"dd if=/dev/zero bs=1 count=1",
	}

	for _, cmd := range commands {
		result := tool.Execute(context.Background(), map[string]any{"command": cmd})
		if result.IsError && strings.Contains(result.ForLLM, "path outside working dir") {
			t.Errorf("safe path should not be blocked by workspace check: %s\n  error: %s", cmd, result.ForLLM)
		}
	}
}

// TestShellTool_CustomAllowPatterns verifies that custom allow patterns exempt
// commands from deny pattern checks.
func TestShellTool_CustomAllowPatterns(t *testing.T) {
	cfg := &config.Config{
		Tools: config.ToolsConfig{
			Exec: config.ExecConfig{
				EnableDenyPatterns:  true,
				CustomAllowPatterns: []string{`\bgit\s+push\s+origin\b`},
			},
		},
	}

	tool, err := NewExecToolWithConfig("", false, cfg)
	if err != nil {
		t.Fatalf("unable to configure exec tool: %s", err)
	}

	// "git push origin main" should be allowed by custom allow pattern.
	result := tool.Execute(context.Background(), map[string]any{
		"command": "git push origin main",
	})
	if result.IsError && strings.Contains(result.ForLLM, "blocked") {
		t.Errorf("custom allow pattern should exempt 'git push origin main', got: %s", result.ForLLM)
	}

	// "git push upstream main" should still be blocked (does not match allow pattern).
	result = tool.Execute(context.Background(), map[string]any{
		"command": "git push upstream main",
	})
	if !result.IsError {
		t.Errorf("'git push upstream main' should still be blocked by deny pattern")
	}
}

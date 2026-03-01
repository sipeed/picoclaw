package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/agent/sandbox"
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

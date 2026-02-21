package sandbox

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

type HostSandbox struct {
	workspace string
	restrict  bool
	fs        FsBridge
}

func NewHostSandbox(workspace string, restrict bool) *HostSandbox {
	return &HostSandbox{
		workspace: workspace,
		restrict:  restrict,
		fs:        &hostFS{workspace: workspace, restrict: restrict},
	}
}

func (h *HostSandbox) Start(ctx context.Context) error {
	return nil
}

func (h *HostSandbox) Prune(ctx context.Context) error {
	return nil
}

func (h *HostSandbox) Fs() FsBridge {
	return h.fs
}

func (h *HostSandbox) Exec(ctx context.Context, req ExecRequest) (*ExecResult, error) {
	return aggregateExecStream(func(onEvent func(ExecEvent) error) (*ExecResult, error) {
		return h.ExecStream(ctx, req, onEvent)
	})
}

func (h *HostSandbox) ExecStream(ctx context.Context, req ExecRequest, onEvent func(ExecEvent) error) (*ExecResult, error) {
	if strings.TrimSpace(req.Command) == "" {
		return nil, fmt.Errorf("empty command")
	}

	cmdCtx := ctx
	cancel := func() {}
	if req.TimeoutMs > 0 {
		cmdCtx, cancel = context.WithTimeout(ctx, durationMs(req.TimeoutMs))
	}
	defer cancel()

	var cmd *exec.Cmd
	if len(req.Args) > 0 {
		cmd = exec.CommandContext(cmdCtx, req.Command, req.Args...)
	} else if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(cmdCtx, "powershell", "-NoProfile", "-NonInteractive", "-Command", req.Command)
	} else {
		cmd = exec.CommandContext(cmdCtx, "sh", "-c", req.Command)
	}

	if req.WorkingDir != "" {
		dir, err := h.resolvePath(req.WorkingDir)
		if err != nil {
			return nil, err
		}
		cmd.Dir = dir
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe setup failed: %w", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("stderr pipe setup failed: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	var stdout, stderr bytes.Buffer
	var callbackMu sync.Mutex
	emit := func(event ExecEvent) error {
		if onEvent == nil {
			return nil
		}
		callbackMu.Lock()
		defer callbackMu.Unlock()
		return onEvent(event)
	}

	readStream := func(r io.Reader, typ ExecEventType, dst *bytes.Buffer) error {
		buf := make([]byte, 4096)
		for {
			n, err := r.Read(buf)
			if n > 0 {
				chunk := append([]byte(nil), buf[:n]...)
				_, _ = dst.Write(chunk)
				if emitErr := emit(ExecEvent{Type: typ, Chunk: chunk}); emitErr != nil {
					return emitErr
				}
			}
			if err == io.EOF {
				return nil
			}
			if err != nil {
				return err
			}
		}
	}

	streamErrs := make(chan error, 2)
	go func() {
		streamErrs <- readStream(stdoutPipe, ExecEventStdout, &stdout)
	}()
	go func() {
		streamErrs <- readStream(stderrPipe, ExecEventStderr, &stderr)
	}()

	var streamErr error
	for i := 0; i < 2; i++ {
		if err := <-streamErrs; err != nil && streamErr == nil {
			streamErr = err
			cancel()
		}
	}

	waitErr := cmd.Wait()
	if streamErr != nil {
		return nil, streamErr
	}
	if cmdCtx.Err() != nil {
		return nil, cmdCtx.Err()
	}

	exitCode := 0
	if waitErr != nil {
		var ee *exec.ExitError
		if ok := asExitError(waitErr, &ee); ok {
			exitCode = ee.ExitCode()
		} else {
			return nil, waitErr
		}
	}
	if err := emit(ExecEvent{Type: ExecEventExit, ExitCode: exitCode}); err != nil {
		return nil, err
	}

	return &ExecResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode,
	}, nil
}

type hostFS struct {
	workspace string
	restrict  bool
}

func (h *hostFS) ReadFile(ctx context.Context, path string) ([]byte, error) {
	resolved, err := resolvePath(path, h.workspace, h.restrict)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(resolved)
}

func (h *hostFS) WriteFile(ctx context.Context, path string, data []byte, mkdir bool) error {
	resolved, err := resolvePath(path, h.workspace, h.restrict)
	if err != nil {
		return err
	}
	if mkdir {
		if err := os.MkdirAll(filepath.Dir(resolved), 0755); err != nil {
			return err
		}
	}
	return os.WriteFile(resolved, data, 0644)
}

func (h *HostSandbox) resolvePath(path string) (string, error) {
	return resolvePath(path, h.workspace, h.restrict)
}

func resolvePath(path, workspace string, restrict bool) (string, error) {
	if workspace == "" {
		return path, nil
	}

	absWorkspace, err := filepath.Abs(workspace)
	if err != nil {
		return "", fmt.Errorf("failed to resolve workspace path: %w", err)
	}

	var absPath string
	if filepath.IsAbs(path) {
		absPath = filepath.Clean(path)
	} else {
		absPath, err = filepath.Abs(filepath.Join(absWorkspace, path))
		if err != nil {
			return "", fmt.Errorf("failed to resolve file path: %w", err)
		}
	}

	if !restrict {
		return absPath, nil
	}

	rel, err := filepath.Rel(absWorkspace, absPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve relative path: %w", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("access denied: path is outside the workspace")
	}

	workspaceReal := absWorkspace
	if resolved, err := filepath.EvalSymlinks(absWorkspace); err == nil {
		workspaceReal = resolved
	}

	if resolved, err := filepath.EvalSymlinks(absPath); err == nil {
		relResolved, err := filepath.Rel(workspaceReal, resolved)
		if err != nil || relResolved == ".." || strings.HasPrefix(relResolved, ".."+string(os.PathSeparator)) {
			return "", fmt.Errorf("access denied: symlink resolves outside workspace")
		}
	}

	return absPath, nil
}

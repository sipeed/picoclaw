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
	// Initialize os.Root for restricted workspace mode to centrally mitigate TOCTOU (Time-of-Check-Time-of-Use) attacks.
	if h.restrict && h.workspace != "" {
		r, err := os.OpenRoot(h.workspace)
		if err != nil {
			return fmt.Errorf("failed to open workspace root: %w", err)
		}
		if hsFS, ok := h.fs.(*hostFS); ok {
			hsFS.root = r
		}
	}
	return nil
}

func (h *HostSandbox) Prune(ctx context.Context) error {
	// Clean up os.Root file descriptors securely.
	if h.restrict && h.workspace != "" {
		if hsFS, ok := h.fs.(*hostFS); ok && hsFS.root != nil {
			err := hsFS.root.Close()
			hsFS.root = nil
			return err
		}
	}
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

func (h *HostSandbox) ExecStream(
	ctx context.Context,
	req ExecRequest,
	onEvent func(ExecEvent) error,
) (*ExecResult, error) {
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
		dir, err := validatePath(req.WorkingDir, h.workspace, h.restrict)
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
	root      *os.Root // OS-level directory file descriptor to safely confine operations and prevent TOCTOU escapes.
}

func (h *hostFS) getSafeRelPath(path string) (string, error) {
	if !filepath.IsAbs(path) {
		return filepath.Clean(path), nil
	}
	if !isWithinWorkspace(path, h.workspace) {
		return "", ErrOutsideWorkspace
	}
	// Rel is safe because isWithinWorkspace returned true
	rel, _ := filepath.Rel(h.workspace, path)
	return rel, nil
}

func (h *hostFS) ReadFile(ctx context.Context, path string) ([]byte, error) {
	if !h.restrict || h.workspace == "" || h.root == nil {
		// Unrestricted mode continues to use traditional resolution
		resolved, err := validatePath(path, h.workspace, h.restrict)
		if err != nil {
			return nil, err
		}
		return os.ReadFile(resolved)
	}

	relPath, err := h.getSafeRelPath(path)
	if err != nil {
		return nil, err
	}

	// os.Root guarantees that the read operation strictly happens within the workspace directory,
	// effectively and atomically mitigating TOCTOU (Time-of-Check-Time-of-Use) vulnerabilities via symlinks.
	return h.root.ReadFile(relPath)
}

func (h *hostFS) WriteFile(ctx context.Context, path string, data []byte, mkdir bool) error {
	if !h.restrict || h.workspace == "" || h.root == nil {
		// Unrestricted mode continues to use traditional resolution
		resolved, err := validatePath(path, h.workspace, h.restrict)
		if err != nil {
			return err
		}
		if mkdir {
			if err := os.MkdirAll(filepath.Dir(resolved), 0o755); err != nil {
				return err
			}
		}
		return os.WriteFile(resolved, data, 0o644)
	}

	relPath, err := h.getSafeRelPath(path)
	if err != nil {
		return err
	}

	if mkdir {
		// MkdirAll natively resolves inside os.Root to avoid escapes.
		if err := h.root.MkdirAll(filepath.Dir(relPath), 0o755); err != nil {
			return err
		}
	}
	// Uses OS-level guarantees to restrict the file writing within the root descriptor.
	return h.root.WriteFile(relPath, data, 0o644)
}

// validatePath ensures the given path is within the workspace if restrict is true but does not ensure atomic TOCTOU protection.
// It is kept for setting string-based fields like cmd.Dir where os.Root cannot be directly mapped.
// The secure file operations boundary relies on os.Root implemented in FsBridge.
// validatePath ensures the given path is within the workspace if restrict is true.
func validatePath(path, workspace string, restrict bool) (string, error) {
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

	if restrict {
		if !isWithinWorkspace(absPath, absWorkspace) {
			return "", ErrOutsideWorkspace
		}

		var resolved string
		workspaceReal := absWorkspace
		if resolved, err = filepath.EvalSymlinks(absWorkspace); err == nil {
			workspaceReal = resolved
		}

		if resolved, err = filepath.EvalSymlinks(absPath); err == nil {
			if !isWithinWorkspace(resolved, workspaceReal) {
				return "", ErrOutsideWorkspace
			}
		} else if os.IsNotExist(err) {
			var parentResolved string
			if parentResolved, err = resolveExistingAncestor(filepath.Dir(absPath)); err == nil {
				if !isWithinWorkspace(parentResolved, workspaceReal) {
					return "", fmt.Errorf("access denied: symlink resolves outside workspace")
				}
			} else if !os.IsNotExist(err) {
				return "", fmt.Errorf("failed to resolve path: %w", err)
			}
		} else {
			return "", fmt.Errorf("failed to resolve path: %w", err)
		}
	}

	return absPath, nil
}

func resolveExistingAncestor(path string) (string, error) {
	for current := filepath.Clean(path); ; current = filepath.Dir(current) {
		if resolved, err := filepath.EvalSymlinks(current); err == nil {
			return resolved, nil
		} else if !os.IsNotExist(err) {
			return "", err
		}
		if filepath.Dir(current) == current {
			return "", os.ErrNotExist
		}
	}
}

func isWithinWorkspace(candidate, workspace string) bool {
	rel, err := filepath.Rel(filepath.Clean(workspace), filepath.Clean(candidate))
	return err == nil && filepath.IsLocal(rel)
}

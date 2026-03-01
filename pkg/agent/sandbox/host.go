package sandbox

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/fileutil"
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

func (h *HostSandbox) GetWorkspace(ctx context.Context) string {
	return h.workspace
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
		cmdCtx, cancel = context.WithTimeout(ctx, time.Duration(req.TimeoutMs)*time.Millisecond)
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
		dir, err := ValidatePath(req.WorkingDir, h.workspace, h.restrict)
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

	prepareCommandForTermination(cmd)
	cmd.Cancel = func() error {
		return terminateProcessTree(cmd)
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
		ee, ok := waitErr.(*exec.ExitError)
		if ok {
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
		resolved, err := ValidatePath(path, h.workspace, h.restrict)
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
		resolved, err := ValidatePath(path, h.workspace, h.restrict)
		if err != nil {
			return err
		}

		parent := filepath.Dir(resolved)
		if !mkdir {
			parentInfo, err := os.Stat(parent)
			if err != nil {
				return err
			}
			if !parentInfo.IsDir() {
				return fmt.Errorf("parent path is not a directory: %s", parent)
			}
		}

		return fileutil.WriteFileAtomic(resolved, data, 0o644)
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
	return writeFileAtomicInRoot(h.root, relPath, data)
}

func writeFileAtomicInRoot(root *os.Root, relPath string, data []byte) error {
	dir := filepath.Dir(relPath)
	tmpName := fmt.Sprintf(".tmp-%d-%d", os.Getpid(), time.Now().UnixNano())
	tmpRelPath := tmpName
	if dir != "." && dir != "/" {
		tmpRelPath = filepath.Join(dir, tmpName)
	}

	tmpFile, err := root.OpenFile(tmpRelPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		_ = root.Remove(tmpRelPath)
		return fmt.Errorf("failed to open temp file: %w", err)
	}

	if _, err := tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()
		_ = root.Remove(tmpRelPath)
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := tmpFile.Sync(); err != nil {
		_ = tmpFile.Close()
		_ = root.Remove(tmpRelPath)
		return fmt.Errorf("failed to sync temp file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		_ = root.Remove(tmpRelPath)
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	if err := root.Rename(tmpRelPath, relPath); err != nil {
		_ = root.Remove(tmpRelPath)
		return fmt.Errorf("failed to rename temp file over target: %w", err)
	}

	syncDir := "."
	if dir != "" && dir != "/" {
		syncDir = dir
	}
	if dirFile, err := root.Open(syncDir); err == nil {
		_ = dirFile.Sync()
		_ = dirFile.Close()
	}

	return nil
}

func (h *hostFS) ReadDir(ctx context.Context, path string) ([]os.DirEntry, error) {
	if !h.restrict || h.workspace == "" || h.root == nil {
		resolved, err := ValidatePath(path, h.workspace, h.restrict)
		if err != nil {
			return nil, err
		}
		return os.ReadDir(resolved)
	}

	relPath, err := h.getSafeRelPath(path)
	if err != nil {
		return nil, err
	}

	return fs.ReadDir(h.root.FS(), relPath)
}

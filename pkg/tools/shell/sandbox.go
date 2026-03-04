package shell

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"mvdan.cc/sh/v3/interp"
)

// safePaths are kernel pseudo-devices that are always safe to open,
// regardless of workspace restriction.
var safePaths = map[string]bool{
	"/dev/null":    true,
	"/dev/zero":    true,
	"/dev/random":  true,
	"/dev/urandom": true,
	"/dev/stdin":   true,
	"/dev/stdout":  true,
	"/dev/stderr":  true,
}

// isSafePath reports whether path is a platform-appropriate pseudo-device
// that should always be accessible regardless of sandbox restrictions.
func isSafePath(path string) bool {
	if safePaths[path] {
		return true
	}
	// On Windows, NUL (case-insensitive) is the equivalent of /dev/null.
	if runtime.GOOS == "windows" && strings.EqualFold(path, "NUL") {
		return true
	}
	return false
}

// SandboxedOpenHandler returns an interp.OpenHandlerFunc that restricts
// shell redirections (>, <, >>) to files within the workspace directory.
//
// NOTE: This only intercepts opens from the shell interpreter for
// redirections. External programs open files via their own syscalls
// and are NOT restricted by this handler.
func SandboxedOpenHandler(workspaceDir string) interp.OpenHandlerFunc {
	absWorkspace, err := filepath.Abs(workspaceDir)
	if err != nil {
		absWorkspace = workspaceDir
	}
	// Resolve workspace symlinks for accurate escape detection.
	absWorkspace, err = filepath.EvalSymlinks(absWorkspace)
	if err != nil {
		// Non-fatal: continue with absolute path. Realpath failures
		// are caught per-file when the sandbox is actually used.
	}

	return func(ctx context.Context, path string, flag int, perm os.FileMode) (io.ReadWriteCloser, error) {
		if isSafePath(path) {
			return os.OpenFile(path, flag, perm)
		}

		// Resolve relative paths against the interpreter's working directory,
		// not the process CWD. The interpreter tracks its own CWD via
		// interp.Dir() and internal cd commands without calling os.Chdir().
		var absPath string
		if filepath.IsAbs(path) {
			absPath = path
		} else {
			hctx := interp.HandlerCtx(ctx)
			absPath = filepath.Join(hctx.Dir, path)
		}
		// filepath.Join already returns a clean path; no extra Abs needed.

		// Resolve symlinks to prevent escape.
		// If the file doesn't exist yet, resolve the parent.
		var resolved string
		if _, err := os.Lstat(absPath); err == nil {
			resolved, err = filepath.EvalSymlinks(absPath)
			if err != nil {
				return nil, fmt.Errorf("sandbox: cannot resolve symlink %q: %w", path, err)
			}
		} else {
			parentDir := filepath.Dir(absPath)
			resolvedParent, err := filepath.EvalSymlinks(parentDir)
			if err != nil {
				return nil, fmt.Errorf("sandbox: cannot resolve parent dir %q: %w", parentDir, err)
			}
			resolved = filepath.Join(resolvedParent, filepath.Base(absPath))
		}

		if absWorkspace != "" {
			rel, err := filepath.Rel(absWorkspace, resolved)
			if err != nil || !filepath.IsLocal(rel) {
				return nil, fmt.Errorf("sandbox: path %q resolves outside workspace", path)
			}
		}

		return os.OpenFile(resolved, flag, perm)
	}
}

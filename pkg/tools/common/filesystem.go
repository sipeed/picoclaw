package common

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/fileutil"
)

// validatePath ensures the given path is within the workspace if restrict is true.
func ValidatePath(path, workspace string, restrict bool) (string, error) {
	if workspace == "" {
		return path, fmt.Errorf("workspace is not defined")
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
			return "", fmt.Errorf("access denied: path is outside the workspace")
		}

		var resolved string
		workspaceReal := absWorkspace
		if resolved, err = filepath.EvalSymlinks(absWorkspace); err == nil {
			workspaceReal = resolved
		}

		if resolved, err = filepath.EvalSymlinks(absPath); err == nil {
			if !isWithinWorkspace(resolved, workspaceReal) {
				return "", fmt.Errorf("access denied: symlink resolves outside workspace")
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

// FileSystem abstracts reading, writing, and listing files, allowing both
// unrestricted (host filesystem) and sandbox (os.Root) implementations to share the same polymorphic interface.
type FileSystem interface {
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte) error
	ReadDir(path string) ([]os.DirEntry, error)
}

// HostFs is an unrestricted fileReadWriter that operates directly on the host filesystem.
type HostFs struct{}

func (h *HostFs) ReadFile(path string) ([]byte, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to read file: file not found: %w", err)
		}
		if os.IsPermission(err) {
			return nil, fmt.Errorf("failed to read file: access denied: %w", err)
		}
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	return content, nil
}

func (h *HostFs) ReadDir(path string) ([]os.DirEntry, error) {
	return os.ReadDir(path)
}

func (h *HostFs) WriteFile(path string, data []byte) error {
	// Use unified atomic write utility with explicit sync for flash storage reliability.
	// Using 0o600 (owner read/write only) for secure default permissions.
	return fileutil.WriteFileAtomic(path, data, 0o600)
}

// SandboxFs is a sandboxed FileSystem that operates within a strictly defined workspace using os.Root.
type SandboxFs struct {
	Workspace string
}

func (r *SandboxFs) execute(path string, fn func(root *os.Root, relPath string) error) error {
	if r.Workspace == "" {
		return fmt.Errorf("workspace is not defined")
	}

	root, err := os.OpenRoot(r.Workspace)
	if err != nil {
		return fmt.Errorf("failed to open workspace: %w", err)
	}
	defer root.Close()

	relPath, err := getSafeRelPath(r.Workspace, path)
	if err != nil {
		return err
	}

	return fn(root, relPath)
}

func (r *SandboxFs) ReadFile(path string) ([]byte, error) {
	var content []byte
	err := r.execute(path, func(root *os.Root, relPath string) error {
		fileContent, err := root.ReadFile(relPath)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("failed to read file: file not found: %w", err)
			}
			// os.Root returns "escapes from parent" for paths outside the root
			if os.IsPermission(err) || strings.Contains(err.Error(), "escapes from parent") ||
				strings.Contains(err.Error(), "permission denied") {
				return fmt.Errorf("failed to read file: access denied: %w", err)
			}
			return fmt.Errorf("failed to read file: %w", err)
		}
		content = fileContent
		return nil
	})
	return content, err
}

func (r *SandboxFs) WriteFile(path string, data []byte) error {
	return r.execute(path, func(root *os.Root, relPath string) error {
		dir := filepath.Dir(relPath)
		if dir != "." && dir != "/" {
			if err := root.MkdirAll(dir, 0o755); err != nil {
				return fmt.Errorf("failed to create parent directories: %w", err)
			}
		}

		// Use atomic write pattern with explicit sync for flash storage reliability.
		// Using 0o600 (owner read/write only) for secure default permissions.
		tmpRelPath := fmt.Sprintf(".tmp-%d-%d", os.Getpid(), time.Now().UnixNano())

		tmpFile, err := root.OpenFile(tmpRelPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err != nil {
			root.Remove(tmpRelPath)
			return fmt.Errorf("failed to open temp file: %w", err)
		}

		if _, err := tmpFile.Write(data); err != nil {
			tmpFile.Close()
			root.Remove(tmpRelPath)
			return fmt.Errorf("failed to write temp file: %w", err)
		}

		// CRITICAL: Force sync to storage medium before rename.
		// This ensures data is physically written to disk, not just cached.
		if err := tmpFile.Sync(); err != nil {
			tmpFile.Close()
			root.Remove(tmpRelPath)
			return fmt.Errorf("failed to sync temp file: %w", err)
		}

		if err := tmpFile.Close(); err != nil {
			root.Remove(tmpRelPath)
			return fmt.Errorf("failed to close temp file: %w", err)
		}

		if err := root.Rename(tmpRelPath, relPath); err != nil {
			root.Remove(tmpRelPath)
			return fmt.Errorf("failed to rename temp file over target: %w", err)
		}

		// Sync directory to ensure rename is durable
		if dirFile, err := root.Open("."); err == nil {
			_ = dirFile.Sync()
			dirFile.Close()
		}

		return nil
	})
}

func (r *SandboxFs) ReadDir(path string) ([]os.DirEntry, error) {
	var entries []os.DirEntry
	err := r.execute(path, func(root *os.Root, relPath string) error {
		dirEntries, err := fs.ReadDir(root.FS(), relPath)
		if err != nil {
			return err
		}
		entries = dirEntries
		return nil
	})
	return entries, err
}

// Helper to get a safe relative path for os.Root usage
func getSafeRelPath(workspace, path string) (string, error) {
	if workspace == "" {
		return "", fmt.Errorf("workspace is not defined")
	}

	rel := filepath.Clean(path)
	if filepath.IsAbs(rel) {
		var err error
		rel, err = filepath.Rel(workspace, rel)
		if err != nil {
			return "", fmt.Errorf("failed to calculate relative path: %w", err)
		}
	}

	if !filepath.IsLocal(rel) {
		return "", fmt.Errorf("path escapes workspace: %s", path)
	}

	return rel, nil
}

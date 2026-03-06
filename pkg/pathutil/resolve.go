package pathutil

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ResolveWorkspacePath resolves path against root and returns an absolute path
// that is guaranteed to be a proper subdirectory of root.
//
// When root is set:
//   - Empty string and bare "." are rejected (must pick a subdirectory).
//   - Absolute paths are rejected (must use relative subdirectory names).
//   - ".." traversal is rejected before any path joining occurs.
//   - A valid relative path is joined to root and returned as an absolute path.
//   - A post-join boundary check confirms the result stays within root.
//
// When root is empty, the function falls back to filepath.Abs(path) for
// backward compatibility with callers that don't have a boundary configured.
func ResolveWorkspacePath(root, path string) (string, error) {
	if root == "" {
		// No boundary configured — resolve raw path as absolute.
		if path == "" {
			return "", fmt.Errorf("workspace path is empty")
		}
		if containsTraversal(path) {
			return "", fmt.Errorf("workspace path contains directory traversal")
		}
		return filepath.Abs(path)
	}

	// With a root boundary, path must be a non-empty relative subdirectory.
	if path == "" {
		return "", fmt.Errorf("workspace path must be a subdirectory of workspace_root, not root itself")
	}

	if isAbsoluteOrRooted(path) {
		return "", fmt.Errorf("workspace path must be relative, got absolute path")
	}

	// Check traversal before Clean — Clean("foo/..") normalises to "." and
	// would produce a misleading "not root itself" error instead.
	if containsTraversal(path) {
		return "", fmt.Errorf("workspace path contains directory traversal")
	}

	if filepath.Clean(path) == "." {
		return "", fmt.Errorf("workspace path must be a subdirectory of workspace_root, not root itself")
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("invalid workspace_root: %w", err)
	}

	resolved := filepath.Clean(filepath.Join(absRoot, path))

	// Belt-and-suspenders: confirm the resolved path is a proper subdirectory
	// of root even after Clean normalisation.
	if !strings.HasPrefix(resolved, absRoot+string(filepath.Separator)) {
		return "", fmt.Errorf("workspace path escapes workspace_root")
	}

	return resolved, nil
}

// containsTraversal detects ".." traversal in all four patterns:
//
//	bare ".."          — the path IS ".."
//	starts with "../"  — e.g. ../etc
//	ends with "/.."    — e.g. foo/..
//	contains "/../"    — e.g. foo/../bar
//
// Both forward-slash and backslash variants are checked so that this
// works correctly on Windows where either separator may appear.
func containsTraversal(path string) bool {
	if path == ".." {
		return true
	}

	for _, sep := range []string{"/", `\`} {
		if strings.HasPrefix(path, ".."+sep) {
			return true
		}
		if strings.HasSuffix(path, sep+"..") {
			return true
		}
		if strings.Contains(path, sep+".."+sep) {
			return true
		}
	}

	return false
}

// isAbsoluteOrRooted returns true if the path is absolute (e.g. C:\foo on
// Windows, /foo on Unix) or rooted with a leading separator. On Windows,
// filepath.IsAbs returns false for Unix-style "/foo" paths, but we still
// reject them since they are not relative subdirectory names.
func isAbsoluteOrRooted(path string) bool {
	if filepath.IsAbs(path) {
		return true
	}
	// Catch Unix-style rooted paths on Windows (e.g. "/etc/passwd").
	if len(path) > 0 && (path[0] == '/' || path[0] == '\\') {
		return true
	}
	return false
}

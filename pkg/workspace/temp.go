package workspace

import (
	"os"
	"path/filepath"
	"strings"
)

const TempDirName = "tmp"

// TempDir returns the standard workspace-local scratch directory. Agents should
// put transient scripts, drafts, and generated helper files here instead of
// scattering tmp_* files in the workspace root.
func TempDir(workspacePath string) string {
	workspacePath = strings.TrimSpace(workspacePath)
	if workspacePath == "" {
		return ""
	}
	return filepath.Join(workspacePath, TempDirName)
}

// EnsureTempDir creates the standard workspace-local scratch directory.
func EnsureTempDir(workspacePath string) (string, error) {
	dir := TempDir(workspacePath)
	if dir == "" {
		return "", nil
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return dir, nil
}

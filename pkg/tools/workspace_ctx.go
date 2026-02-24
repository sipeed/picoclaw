package tools

import (
	"context"
	"path/filepath"
	"strings"
)

type workspaceOverrideKey struct{}

// WithWorkspaceOverride returns a context carrying a workspace override path.
// Tools will resolve file operations against this path instead of the original workspace.
func WithWorkspaceOverride(ctx context.Context, workspace string) context.Context {
	return context.WithValue(ctx, workspaceOverrideKey{}, workspace)
}

// WorkspaceOverrideFromCtx extracts the workspace override from context, or "".
func WorkspaceOverrideFromCtx(ctx context.Context) string {
	if v, ok := ctx.Value(workspaceOverrideKey{}).(string); ok {
		return v
	}
	return ""
}

// resolveFS returns a fileSystem applying workspace override from context.
// Paths under "memory/" are excluded (always use original workspace).
// For sandboxFs: creates a temporary instance with the override workspace.
// For hostFs (unrestricted): returns as-is.
func resolveFS(ctx context.Context, fs fileSystem, path string) fileSystem {
	override := WorkspaceOverrideFromCtx(ctx)
	if override == "" {
		return fs
	}

	// memory/ paths always use original workspace
	if isMemoryPath(path) {
		return fs
	}

	// Only sandboxFs supports workspace override
	if sfs, ok := fs.(*sandboxFs); ok {
		if sfs.workspace == override {
			return fs
		}
		return &sandboxFs{workspace: override}
	}

	return fs
}

// isMemoryPath returns true for paths under the memory/ directory.
// Matches: "memory/MEMORY.md", "memory", "/workspace/memory/notes.md"
func isMemoryPath(path string) bool {
	p := filepath.ToSlash(filepath.Clean(path))

	// Relative path starting with memory/
	if strings.HasPrefix(p, "memory/") || p == "memory" {
		return true
	}

	// Absolute path containing /memory/ or ending with /memory
	if strings.Contains(p, "/memory/") || strings.HasSuffix(p, "/memory") {
		return true
	}

	return false
}

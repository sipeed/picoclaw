package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// validatePath ensures the given path is within one of the allowed paths if restrict is true.
// workspace can be a single path or multiple paths separated by '|'.
func validatePath(path string, workspace string, restrict bool) (string, error) {
	if workspace == "" {
		return path, nil
	}

	allowedPaths := strings.Split(workspace, "|")
	primaryWorkspace := allowedPaths[0]
	absPrimary, err := filepath.Abs(primaryWorkspace)
	if err != nil {
		return "", fmt.Errorf("failed to resolve primary workspace path: %w", err)
	}

	var absPath string
	if filepath.IsAbs(path) {
		absPath = filepath.Clean(path)
	} else {
		absPath, err = filepath.Abs(filepath.Join(absPrimary, path))
		if err != nil {
			return "", fmt.Errorf("failed to resolve file path: %w", err)
		}
	}

	if restrict {
		found := false
		for _, wp := range allowedPaths {
			absWP, _ := filepath.Abs(wp)
			if isWithinWorkspace(absPath, absWP) {
				found = true
				break
			}
		}

		if !found {
			return "", fmt.Errorf("access denied: path is outside the workspace")
		}

		// Symlink identification and deeper validation
		var resolved string
		if resolved, err = filepath.EvalSymlinks(absPath); err == nil {
			resFound := false
			for _, wp := range allowedPaths {
				absWP, _ := filepath.Abs(wp)
				resolvedWP := absWP
				if r, err := filepath.EvalSymlinks(absWP); err == nil {
					resolvedWP = r
				}
				if isWithinWorkspace(resolved, resolvedWP) {
					resFound = true
					break
				}
			}
			if !resFound {
				return "", fmt.Errorf("access denied: symlink resolves outside workspace")
			}
		} else if os.IsNotExist(err) {
			var parentResolved string
			if parentResolved, err = resolveExistingAncestor(filepath.Dir(absPath)); err == nil {
				resFound := false
				for _, wp := range allowedPaths {
					absWP, _ := filepath.Abs(wp)
					resolvedWP := absWP
					if r, err := filepath.EvalSymlinks(absWP); err == nil {
						resolvedWP = r
					}
					if isWithinWorkspace(parentResolved, resolvedWP) {
						resFound = true
						break
					}
				}
				if !resFound {
					return "", fmt.Errorf("access denied: symlink resolves outside workspace")
				}
			}
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
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}

type ReadFileTool struct {
	workspace string
	restrict  bool
}

func NewReadFileTool(workspace string, restrict bool) *ReadFileTool {
	return &ReadFileTool{workspace: workspace, restrict: restrict}
}

func (t *ReadFileTool) Name() string {
	return "read_file"
}

func (t *ReadFileTool) Description() string {
	return "Read the contents of a file"
}

func (t *ReadFileTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Path to the file to read",
			},
		},
		"required": []string{"path"},
	}
}

func (t *ReadFileTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	path, ok := args["path"].(string)
	if !ok {
		return ErrorResult("path is required")
	}

	resolvedPath, err := validatePath(path, t.workspace, t.restrict)
	if err != nil {
		return ErrorResult(err.Error())
	}

	content, err := os.ReadFile(resolvedPath)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to read file: %v", err))
	}

	return NewToolResult(string(content))
}

type WriteFileTool struct {
	workspace string
	restrict  bool
}

func NewWriteFileTool(workspace string, restrict bool) *WriteFileTool {
	return &WriteFileTool{workspace: workspace, restrict: restrict}
}

func (t *WriteFileTool) Name() string {
	return "write_file"
}

func (t *WriteFileTool) Description() string {
	return "Write content to a file"
}

func (t *WriteFileTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Path to the file to write",
			},
			"content": map[string]any{
				"type":        "string",
				"description": "Content to write to the file",
			},
		},
		"required": []string{"path", "content"},
	}
}

func (t *WriteFileTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	path, ok := args["path"].(string)
	if !ok {
		return ErrorResult("path is required")
	}

	content, ok := args["content"].(string)
	if !ok {
		return ErrorResult("content is required")
	}

	resolvedPath, err := validatePath(path, t.workspace, t.restrict)
	if err != nil {
		return ErrorResult(err.Error())
	}

	dir := filepath.Dir(resolvedPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return ErrorResult(fmt.Sprintf("failed to create directory: %v", err))
	}

	if err := os.WriteFile(resolvedPath, []byte(content), 0o644); err != nil {
		return ErrorResult(fmt.Sprintf("failed to write file: %v", err))
	}

	return SilentResult(fmt.Sprintf("File written: %s", path))
}

type ListDirTool struct {
	workspace string
	restrict  bool
}

func NewListDirTool(workspace string, restrict bool) *ListDirTool {
	return &ListDirTool{workspace: workspace, restrict: restrict}
}

func (t *ListDirTool) Name() string {
	return "list_dir"
}

func (t *ListDirTool) Description() string {
	return "List files and directories in a path"
}

func (t *ListDirTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Path to list",
			},
		},
		"required": []string{"path"},
	}
}

func (t *ListDirTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	path, ok := args["path"].(string)
	if !ok {
		path = "."
	}

	resolvedPath, err := validatePath(path, t.workspace, t.restrict)
	if err != nil {
		return ErrorResult(err.Error())
	}

	entries, err := os.ReadDir(resolvedPath)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to read directory: %v", err))
	}

	result := ""
	for _, entry := range entries {
		if entry.IsDir() {
			result += "DIR:  " + entry.Name() + "\n"
		} else {
			result += "FILE: " + entry.Name() + "\n"
		}
	}

	return NewToolResult(result)
}

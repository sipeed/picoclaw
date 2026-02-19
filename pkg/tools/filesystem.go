package tools

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

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

func (t *ReadFileTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Path to the file to read",
			},
		},
		"required": []string{"path"},
	}
}

func (t *ReadFileTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
	path, ok := args["path"].(string)
	if !ok {
		return ErrorResult("path is required")
	}

	// If restriction is disabled, fall back to standard os interactions (insecure but intended)
	if !t.restrict {
		content, err := os.ReadFile(path)
		if err != nil {
			return ErrorResult(fmt.Sprintf("failed to read file: %v", err))
		}
		return NewToolResult(string(content))
	}

	return executeInRoot(t.workspace, path, func(root *os.Root, relPath string) (*ToolResult, error) {
		f, err := root.Open(relPath)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, fmt.Errorf("failed to read file: file not found: %s", path)
			}
			return nil, fmt.Errorf("access denied or failed to open: %w", err)
		}
		defer f.Close()

		content, err := io.ReadAll(f)
		if err != nil {
			return nil, fmt.Errorf("failed to read file: %v", err)
		}
		return NewToolResult(string(content)), nil
	})
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

func (t *WriteFileTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Path to the file to write",
			},
			"content": map[string]interface{}{
				"type":        "string",
				"description": "Content to write to the file",
			},
		},
		"required": []string{"path", "content"},
	}
}

func (t *WriteFileTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
	path, ok := args["path"].(string)
	if !ok {
		return ErrorResult("path is required")
	}

	content, ok := args["content"].(string)
	if !ok {
		return ErrorResult("content is required")
	}

	if !t.restrict {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return ErrorResult(fmt.Sprintf("failed to create directory: %v", err))
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return ErrorResult(fmt.Sprintf("failed to write file: %v", err))
		}
		return SilentResult(fmt.Sprintf("File written: %s", path))
	}

	return executeInRoot(t.workspace, path, func(root *os.Root, relPath string) (*ToolResult, error) {
		// Ensure parent directory exists within root using recursive creation
		dir := filepath.Dir(relPath)
		if dir != "." && dir != "/" {
			if err := mkdirAllInRoot(root, dir); err != nil {
				return nil, fmt.Errorf("failed to create parent directories: %w", err)
			}
		}

		f, err := root.Create(relPath)
		if err != nil {
			return nil, fmt.Errorf("failed to create file: %w", err)
		}
		defer f.Close()

		_, err = f.WriteString(content)
		if err != nil {
			return nil, fmt.Errorf("failed to write file: %w", err)
		}
		return SilentResult(fmt.Sprintf("File written: %s", path)), nil
	})
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

func (t *ListDirTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Path to list",
			},
		},
		"required": []string{"path"},
	}
}

func (t *ListDirTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
	path, ok := args["path"].(string)
	if !ok {
		path = "."
	}

	if !t.restrict {
		entries, err := os.ReadDir(path)
		if err != nil {
			return ErrorResult(fmt.Sprintf("failed to read directory: %v", err))
		}
		return formatDirEntries(entries)
	}

	return executeInRoot(t.workspace, path, func(root *os.Root, relPath string) (*ToolResult, error) {
		f, err := root.Open(relPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open directory: %w", err)
		}
		defer f.Close()

		entries, err := f.ReadDir(-1)
		if err != nil {
			return nil, fmt.Errorf("failed to read directory: %w", err)
		}

		return formatDirEntries(entries), nil
	})
}

func formatDirEntries(entries []os.DirEntry) *ToolResult {
	var result strings.Builder
	for _, entry := range entries {
		if entry.IsDir() {
			result.WriteString("DIR:  " + entry.Name() + "\n")
		} else {
			result.WriteString("FILE: " + entry.Name() + "\n")
		}
	}
	return NewToolResult(result.String())
}

// Helper to get a safe relative path for os.Root usage
func getSafeRelPath(workspace, path string) (string, error) {
	if workspace == "" {
		return "", fmt.Errorf("workspace is empty and not defined")
	}

	path = filepath.Clean(path)

	// If absolute, make it relative to workspace
	// os.Root only accepts relative paths
	if filepath.IsAbs(path) {
		rel, err := filepath.Rel(workspace, path)
		if err != nil {
			return "", fmt.Errorf("failed to calculate relative path: %w", err)
		}
		path = rel
	}

	// Check for escape
	if path == ".." || strings.HasPrefix(path, "../") {
		return "", fmt.Errorf("path escapes workspace: %s", path)
	}

	return path, nil
}

// executeInRoot executes a function within the safety of os.Root
func executeInRoot(workspace string, path string, fn func(root *os.Root, relPath string) (*ToolResult, error)) *ToolResult {
	if workspace == "" {
		return ErrorResult("workspace is not defined")
	}

	// 1. Open the Root
	root, err := os.OpenRoot(workspace)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to open workspace root: %v", err))
	}
	defer root.Close()

	// 2. Calculate relative path
	relPath, err := getSafeRelPath(workspace, path)
	if err != nil {
		return ErrorResult(err.Error())
	}

	// 3. Execute the operation
	result, err := fn(root, relPath)
	if err != nil {
		return ErrorResult(err.Error())
	}

	return result
}

// mkdirAllInRoot mimics os.MkdirAll but within os.Root
func mkdirAllInRoot(root *os.Root, relPath string) error {
	relPath = filepath.Clean(relPath)
	if relPath == "." || relPath == "/" {
		return nil
	}

	dir := filepath.Dir(relPath)
	if dir != "." && dir != "/" {
		if err := mkdirAllInRoot(root, dir); err != nil {
			return err
		}
	}

	err := root.Mkdir(relPath, 0755)
	if err != nil && !os.IsExist(err) {
		return err
	}
	return nil
}

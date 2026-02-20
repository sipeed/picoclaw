package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
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

	if t.restrict {
		return executeInRoot(t.workspace, path, func(root *os.Root, relPath string) (*ToolResult, error) {
			content, err := (&rootRW{root: root}).Read(relPath)
			if err != nil {
				return nil, err
			}
			return NewToolResult(string(content)), nil
		})
	}

	content, err := (&hostRW{}).Read(path)
	if err != nil {
		return ErrorResult(err.Error())
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

	if t.restrict {
		return executeInRoot(t.workspace, path, func(root *os.Root, relPath string) (*ToolResult, error) {
			if err := (&rootRW{root: root}).Write(relPath, []byte(content)); err != nil {
				return nil, err
			}
			return SilentResult(fmt.Sprintf("File written: %s", path)), nil
		})
	}

	if err := (&hostRW{}).Write(path, []byte(content)); err != nil {
		return ErrorResult(err.Error())
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

// fileReadWriter abstracts reading and writing files, allowing both unrestricted
// (host filesystem) and sandbox (os.Root) implementations to share the same logic.
type fileReadWriter interface {
	Read(path string) ([]byte, error)
	Write(path string, data []byte) error
}

// hostRW is an unrestricted fileReadWriter that operates directly on the host filesystem.
type hostRW struct{}

func (h *hostRW) Read(path string) ([]byte, error) {
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

func (h *hostRW) Write(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create parent directories: %w", err)
	}

	tmpPath := fmt.Sprintf("%s.%d.tmp", path, time.Now().UnixNano())
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to replace original file: %w", err)
	}
	return nil
}

// rootRW is a sandboxed fileReadWriter that operates within an os.Root boundary.
// All paths passed to Read/Write must be relative to the root.
type rootRW struct {
	root *os.Root
}

func (r *rootRW) Read(path string) ([]byte, error) {
	content, err := r.root.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to read file: file not found: %w", err)
		}
		// os.Root returns "escapes from parent" for paths outside the root
		if os.IsPermission(err) || strings.Contains(err.Error(), "escapes from parent") || strings.Contains(err.Error(), "permission denied") {
			return nil, fmt.Errorf("failed to read file: access denied: %w", err)
		}
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	return content, nil
}

func (r *rootRW) Write(path string, data []byte) error {
	dir := filepath.Dir(path)
	if dir != "." && dir != "/" {
		// Use native root.MkdirAll which handles the "file exists at path" check internally.
		if err := r.root.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create parent directories: %w", err)
		}
	}

	tmpRelPath := fmt.Sprintf("%s.%d.tmp", path, time.Now().UnixNano())
	fw, err := r.root.Create(tmpRelPath)
	if err != nil {
		return fmt.Errorf("failed to create temp file for writing: %w", err)
	}

	if _, err := fw.Write(data); err != nil {
		fw.Close()
		r.root.Remove(tmpRelPath)
		return fmt.Errorf("failed to write to temp file: %w", err)
	}

	if err := fw.Close(); err != nil {
		r.root.Remove(tmpRelPath)
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	if err := r.root.Rename(tmpRelPath, path); err != nil {
		r.root.Remove(tmpRelPath)
		return fmt.Errorf("failed to rename temp file over target: %w", err)
	}
	return nil
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

	// Check for escape manually (defense-in-depth, as os.Root also rejects paths that escape the root)
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

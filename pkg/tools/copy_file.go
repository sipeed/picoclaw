package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// CopyFileTool copies a file from source to destination.
// Source is allowed from mediaDir or workspace (when restrict=true).
// Destination must be within workspace.
type CopyFileTool struct {
	workspace string
	mediaDir  string
	restrict  bool
}

func NewCopyFileTool(workspace, mediaDir string, restrict bool) *CopyFileTool {
	return &CopyFileTool{
		workspace: workspace,
		mediaDir:  mediaDir,
		restrict:  restrict,
	}
}

func (t *CopyFileTool) Name() string {
	return "copy_file"
}

func (t *CopyFileTool) Description() string {
	return "Copy a file from source to destination. Source can be a media file (from camera/screenshot) or a workspace file. Destination must be within the workspace."
}

func (t *CopyFileTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"source": map[string]interface{}{
				"type":        "string",
				"description": "Path to the source file to copy",
			},
			"destination": map[string]interface{}{
				"type":        "string",
				"description": "Path to the destination file (within workspace)",
			},
		},
		"required": []string{"source", "destination"},
	}
}

func (t *CopyFileTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
	source, ok := args["source"].(string)
	if !ok {
		return ErrorResult("source is required")
	}

	destination, ok := args["destination"].(string)
	if !ok {
		return ErrorResult("destination is required")
	}

	// Validate source path
	srcPath, err := t.validateSource(source)
	if err != nil {
		return ErrorResult(err.Error())
	}

	// Validate destination path (must be within workspace)
	dstPath, err := validatePath(destination, t.workspace, true)
	if err != nil {
		return ErrorResult(fmt.Sprintf("destination: %s", err.Error()))
	}

	// Read source file
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to read source file: %v", err))
	}

	// Create destination directory if needed
	if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
		return ErrorResult(fmt.Sprintf("failed to create directory: %v", err))
	}

	// Write destination file
	if err := os.WriteFile(dstPath, data, 0644); err != nil {
		return ErrorResult(fmt.Sprintf("failed to write destination file: %v", err))
	}

	return SilentResult(fmt.Sprintf("File copied: %s → %s", source, destination))
}

// validateSource checks that the source path is allowed.
// When restrict=true, source must be within workspace or mediaDir.
func (t *CopyFileTool) validateSource(source string) (string, error) {
	// First try workspace validation
	srcPath, err := validatePath(source, t.workspace, t.restrict)
	if err == nil {
		return srcPath, nil
	}

	// If restrict mode and workspace validation failed, check mediaDir
	if t.restrict && t.mediaDir != "" {
		absMediaDir, merr := filepath.Abs(t.mediaDir)
		if merr != nil {
			return "", fmt.Errorf("access denied: path is outside the workspace")
		}

		var absSource string
		if filepath.IsAbs(source) {
			absSource = filepath.Clean(source)
		} else {
			absSource = filepath.Clean(filepath.Join(absMediaDir, source))
		}

		if isWithinWorkspace(absSource, absMediaDir) {
			return absSource, nil
		}

		return "", fmt.Errorf("access denied: source path is outside workspace and media directory")
	}

	return "", err
}

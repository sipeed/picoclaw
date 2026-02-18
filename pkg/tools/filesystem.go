package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sipeed/picoclaw/pkg/security"
)

// validatePath ensures the given path is within the workspace if restrict is true.
// When pathMode is "off", only basic prefix check is performed (no symlink resolution).
// When pathMode is "block" or "approve", enhanced symlink resolution is used.
func validatePath(path, workspace string, restrict bool) (string, error) {
	return validatePathWithMode(path, workspace, restrict, security.ModeOff, nil, "", "")
}

// validatePathWithMode is the full-featured path validator with policy support.
func validatePathWithMode(path, workspace string, restrict bool, pathMode security.PolicyMode, pe *security.PolicyEngine, channel, chatID string) (string, error) {
	if workspace == "" {
		return path, nil
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
		useSymlinkResolution := !pathMode.IsOff()

		realWorkspace := absWorkspace
		if useSymlinkResolution {
			if resolved, err := filepath.EvalSymlinks(absWorkspace); err == nil {
				realWorkspace = resolved
			}
		}

		realPath := absPath
		if useSymlinkResolution {
			if resolved, err := filepath.EvalSymlinks(absPath); err == nil {
				realPath = resolved
			} else if os.IsNotExist(err) {
				if parentResolved, e2 := resolveExistingAncestor(filepath.Dir(absPath)); e2 == nil {
					realPath = filepath.Join(parentResolved, filepath.Base(absPath))
				}
			} else if resolved, err := filepath.EvalSymlinks(filepath.Dir(absPath)); err == nil {
				realPath = filepath.Join(resolved, filepath.Base(absPath))
			}
		}

		if !isWithinWorkspace(realPath, realWorkspace) {
			violation := fmt.Errorf("access denied: symlink resolves outside workspace")
			if !useSymlinkResolution {
				violation = fmt.Errorf("access denied: path is outside the workspace")
			}
			if pe != nil && pathMode == security.ModeApprove {
				ctx := context.Background()
				pErr := pe.Evaluate(ctx, pathMode, security.Violation{
					Category: "path_validation",
					Tool:     "filesystem",
					Action:   path,
					Reason:   violation.Error(),
				}, channel, chatID)
				if pErr != nil {
					return "", pErr
				}
			} else {
				return "", violation
			}
		}

		absPath = realPath
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

// PathPolicyOpts holds optional security policy settings for filesystem tools.
type PathPolicyOpts struct {
	PathMode     security.PolicyMode
	PolicyEngine *security.PolicyEngine
}

type ReadFileTool struct {
	workspace    string
	restrict     bool
	pathMode     security.PolicyMode
	policyEngine *security.PolicyEngine
	channel      string
	chatID       string
}

func NewReadFileTool(workspace string, restrict bool) *ReadFileTool {
	return &ReadFileTool{workspace: workspace, restrict: restrict}
}

func NewReadFileToolWithPolicy(workspace string, restrict bool, opts PathPolicyOpts) *ReadFileTool {
	return &ReadFileTool{workspace: workspace, restrict: restrict, pathMode: opts.PathMode, policyEngine: opts.PolicyEngine}
}

func (t *ReadFileTool) SetContext(channel, chatID string) {
	t.channel = channel
	t.chatID = chatID
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

	resolvedPath, err := validatePathWithMode(path, t.workspace, t.restrict, t.pathMode, t.policyEngine, t.channel, t.chatID)
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
	workspace    string
	restrict     bool
	pathMode     security.PolicyMode
	policyEngine *security.PolicyEngine
	channel      string
	chatID       string
}

func NewWriteFileTool(workspace string, restrict bool) *WriteFileTool {
	return &WriteFileTool{workspace: workspace, restrict: restrict}
}

func NewWriteFileToolWithPolicy(workspace string, restrict bool, opts PathPolicyOpts) *WriteFileTool {
	return &WriteFileTool{workspace: workspace, restrict: restrict, pathMode: opts.PathMode, policyEngine: opts.PolicyEngine}
}

func (t *WriteFileTool) SetContext(channel, chatID string) {
	t.channel = channel
	t.chatID = chatID
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

	resolvedPath, err := validatePathWithMode(path, t.workspace, t.restrict, t.pathMode, t.policyEngine, t.channel, t.chatID)
	if err != nil {
		return ErrorResult(err.Error())
	}

	dir := filepath.Dir(resolvedPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return ErrorResult(fmt.Sprintf("failed to create directory: %v", err))
	}

	if err := os.WriteFile(resolvedPath, []byte(content), 0600); err != nil {
		return ErrorResult(fmt.Sprintf("failed to write file: %v", err))
	}

	return SilentResult(fmt.Sprintf("File written: %s", path))
}

type ListDirTool struct {
	workspace    string
	restrict     bool
	pathMode     security.PolicyMode
	policyEngine *security.PolicyEngine
	channel      string
	chatID       string
}

func NewListDirTool(workspace string, restrict bool) *ListDirTool {
	return &ListDirTool{workspace: workspace, restrict: restrict}
}

func NewListDirToolWithPolicy(workspace string, restrict bool, opts PathPolicyOpts) *ListDirTool {
	return &ListDirTool{workspace: workspace, restrict: restrict, pathMode: opts.PathMode, policyEngine: opts.PolicyEngine}
}

func (t *ListDirTool) SetContext(channel, chatID string) {
	t.channel = channel
	t.chatID = chatID
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

	resolvedPath, err := validatePathWithMode(path, t.workspace, t.restrict, t.pathMode, t.policyEngine, t.channel, t.chatID)
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

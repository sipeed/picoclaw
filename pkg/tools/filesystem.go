package tools

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/sipeed/picoclaw/pkg/agent/sandbox"
	"github.com/sipeed/picoclaw/pkg/fileutil"
	"github.com/sipeed/picoclaw/pkg/logger"
)

const MaxReadFileSize = 64 * 1024 // 64KB limit to avoid context overflow

// validatePath ensures the given path is within the workspace if restrict is true.
func validatePath(path, workspace string, restrict bool) (string, error) {
	return sandbox.ValidatePath(path, workspace, restrict)
}

type ReadFileTool struct {
	allowPaths []*regexp.Regexp
	maxSize    int64
}

func NewReadFileTool(
	workspace string,
	restrict bool,
	maxReadFileSize int,
	allowPaths ...[]*regexp.Regexp,
) *ReadFileTool {
	maxSize := int64(maxReadFileSize)
	if maxSize <= 0 {
		maxSize = MaxReadFileSize
	}

	return &ReadFileTool{
		allowPaths: firstPatternSet(allowPaths),
		maxSize:    maxSize,
	}
}

func (t *ReadFileTool) Name() string {
	return "read_file"
}

func (t *ReadFileTool) Description() string {
	return "Read the contents of a file. Supports pagination via `offset` and `length`."
}

func (t *ReadFileTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Path to the file to read.",
			},
			"offset": map[string]any{
				"type":        "integer",
				"description": "Byte offset to start reading from.",
				"default":     0,
			},
			"length": map[string]any{
				"type":        "integer",
				"description": "Maximum number of bytes to read.",
				"default":     t.maxSize,
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

	// offset (optional, default 0)
	offset, err := getInt64Arg(args, "offset", 0)
	if err != nil {
		return ErrorResult(err.Error())
	}
	if offset < 0 {
		return ErrorResult("offset must be >= 0")
	}

	// length (optional, capped at MaxReadFileSize)
	length, err := getInt64Arg(args, "length", t.maxSize)
	if err != nil {
		return ErrorResult(err.Error())
	}
	if length <= 0 {
		return ErrorResult("length must be > 0")
	}
	if length > t.maxSize {
		length = t.maxSize
	}

	if content, handled, err := readAllowedHostPath(ctx, path, t.allowPaths); handled {
		if err != nil {
			return ErrorResult(fmt.Sprintf("failed to read file: %v", err))
		}
		return buildReadResult(path, content, offset, length)
	}

	sb := sandbox.FromContext(ctx)
	if sb == nil {
		return ErrorResult("sandbox environment unavailable")
	}

	content, err := sb.Fs().ReadFile(ctx, path)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to read file: %v", err))
	}
	return buildReadResult(path, content, offset, length)
}

func buildReadResult(path string, content []byte, offset, length int64) *ToolResult {
	totalSize := int64(len(content))
	if offset >= totalSize {
		return NewToolResult("[END OF FILE - no content at this offset]")
	}
	if length <= 0 {
		return ErrorResult("length must be > 0")
	}

	end := offset + length
	if end > totalSize {
		end = totalSize
	}

	data := content[offset:end]
	if len(data) == 0 {
		return NewToolResult("[END OF FILE - no content at this offset]")
	}

	// Build metadata header.
	// use filepath.Base(path) instead of the raw path to avoid leaking
	// internal filesystem structure into the LLM context.
	readEnd := offset + int64(len(data))
	readRange := fmt.Sprintf("bytes %d-%d", offset, readEnd-1)

	displayPath := filepath.Base(path)
	header := fmt.Sprintf(
		"[file: %s | total: %d bytes | read: %s]",
		displayPath, totalSize, readRange,
	)

	if readEnd < totalSize {
		header += fmt.Sprintf(
			"\n[TRUNCATED - file has more content. Call read_file again with offset=%d to continue.]",
			readEnd,
		)
	} else {
		header += "\n[END OF FILE - no further content.]"
	}

	logger.DebugCF("tool", "ReadFileTool execution completed successfully",
		map[string]any{
			"path":       path,
			"bytes_read": len(data),
			"has_more":   readEnd < totalSize,
		})

	return NewToolResult(header + "\n\n" + string(data))
}

type WriteFileTool struct {
	allowPaths []*regexp.Regexp
}

func NewWriteFileTool(workspace string, restrict bool, allowPaths ...[]*regexp.Regexp) *WriteFileTool {
	return &WriteFileTool{allowPaths: firstPatternSet(allowPaths)}
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

	if handled, err := writeAllowedHostPath(ctx, path, []byte(content), t.allowPaths); handled {
		if err != nil {
			return ErrorResult(fmt.Sprintf("failed to write file: %v", err))
		}
		return SilentResult(fmt.Sprintf("File written: %s", path))
	}

	sb := sandbox.FromContext(ctx)
	if sb == nil {
		return ErrorResult("sandbox environment unavailable")
	}

	if err := sb.Fs().WriteFile(ctx, path, []byte(content), true); err != nil {
		return ErrorResult(fmt.Sprintf("failed to write file: %v", err))
	}
	return SilentResult(fmt.Sprintf("File written: %s", path))
}

type ListDirTool struct {
	allowPaths []*regexp.Regexp
}

func NewListDirTool(workspace string, restrict bool, allowPaths ...[]*regexp.Regexp) *ListDirTool {
	return &ListDirTool{allowPaths: firstPatternSet(allowPaths)}
}

func (t *ListDirTool) Name() string {
	return "list_dir"
}

func (t *ListDirTool) Description() string {
	return "List files and directories in a directory"
}

func (t *ListDirTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Path to the directory to list",
			},
		},
	}
}

func (t *ListDirTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	path, _ := args["path"].(string)
	if path == "" {
		path = "."
	}

	if entries, handled, err := readAllowedHostDir(ctx, path, t.allowPaths); handled {
		if err != nil {
			return ErrorResult(fmt.Sprintf("failed to read directory: %v", err))
		}
		return formatDirEntries(entries)
	}

	sb := sandbox.FromContext(ctx)
	if sb == nil {
		return ErrorResult("sandbox environment unavailable")
	}

	entries, err := sb.Fs().ReadDir(ctx, path)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to read directory: %v", err))
	}

	return formatDirEntries(entries)
}

func formatDirEntries(entries []os.DirEntry) *ToolResult {
	var result strings.Builder
	for _, entry := range entries {
		if entry.IsDir() {
			result.WriteString(fmt.Sprintf("%s/ (dir)\n", entry.Name()))
		} else {
			result.WriteString(fmt.Sprintf("%s\n", entry.Name()))
		}
	}
	return NewToolResult(result.String())
}

func firstPatternSet(sets [][]*regexp.Regexp) []*regexp.Regexp {
	if len(sets) == 0 {
		return nil
	}
	return sets[0]
}

func hostSandboxFromContext(ctx context.Context) *sandbox.HostSandbox {
	sb := sandbox.FromContext(ctx)
	if sb == nil {
		return nil
	}
	host, _ := sb.(*sandbox.HostSandbox)
	return host
}

func matchesAnyPattern(path string, patterns []*regexp.Regexp) bool {
	if len(patterns) == 0 {
		return false
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}
	for _, pattern := range patterns {
		if pattern != nil && (pattern.MatchString(path) || pattern.MatchString(absPath)) {
			return true
		}
	}
	return false
}

func readAllowedHostPath(ctx context.Context, path string, patterns []*regexp.Regexp) ([]byte, bool, error) {
	if hostSandboxFromContext(ctx) == nil || !matchesAnyPattern(path, patterns) {
		return nil, false, nil
	}
	content, err := os.ReadFile(path)
	return content, true, err
}

func writeAllowedHostPath(ctx context.Context, path string, data []byte, patterns []*regexp.Regexp) (bool, error) {
	if hostSandboxFromContext(ctx) == nil || !matchesAnyPattern(path, patterns) {
		return false, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return true, err
	}
	return true, fileutil.WriteFileAtomic(path, data, 0o644)
}

func readAllowedHostDir(ctx context.Context, path string, patterns []*regexp.Regexp) ([]os.DirEntry, bool, error) {
	if hostSandboxFromContext(ctx) == nil || !matchesAnyPattern(path, patterns) {
		return nil, false, nil
	}
	entries, err := os.ReadDir(path)
	return entries, true, err
}

func getInt64Arg(args map[string]any, key string, defaultVal int64) (int64, error) {
	raw, exists := args[key]
	if !exists {
		return defaultVal, nil
	}

	switch v := raw.(type) {
	case float64:
		if v != math.Trunc(v) {
			return 0, fmt.Errorf("%s must be an integer, got float %v", key, v)
		}
		if v > math.MaxInt64 || v < math.MinInt64 {
			return 0, fmt.Errorf("%s value %v overflows int64", key, v)
		}
		return int64(v), nil
	case int:
		return int64(v), nil
	case int64:
		return v, nil
	case string:
		parsed, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid integer format for %s parameter: %w", key, err)
		}
		return parsed, nil
	default:
		return 0, fmt.Errorf("unsupported type %T for %s parameter", raw, key)
	}
}

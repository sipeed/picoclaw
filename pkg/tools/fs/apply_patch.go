package fstools

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"regexp"
	"strings"
)

// ApplyPatchTool applies a small Codex-style multi-file patch.
type ApplyPatchTool struct {
	fs fileSystem
}

func NewApplyPatchTool(workspace string, restrict bool, allowPaths ...[]*regexp.Regexp) *ApplyPatchTool {
	var patterns []*regexp.Regexp
	if len(allowPaths) > 0 {
		patterns = allowPaths[0]
	}
	return &ApplyPatchTool{fs: buildFs(workspace, restrict, patterns)}
}

func (t *ApplyPatchTool) Name() string {
	return "apply_patch"
}

func (t *ApplyPatchTool) Description() string {
	return "Apply a structured multi-file patch. Use the Codex patch format with *** Begin Patch, one or more *** Add File / *** Update File / *** Delete File sections, and *** End Patch. Prefer this over write_file for code edits spanning multiple files."
}

func (t *ApplyPatchTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"input": map[string]any{
				"type":        "string",
				"description": "Full patch text, including *** Begin Patch and *** End Patch.",
			},
		},
		"required": []string{"input"},
	}
}

func (t *ApplyPatchTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	input, ok := args["input"].(string)
	if !ok || strings.TrimSpace(input) == "" {
		return ErrorResult("input is required")
	}

	ops, err := parseApplyPatch(input)
	if err != nil {
		return ErrorResult(err.Error())
	}
	results, err := applyPatchOperations(t.fs, ops)
	if err != nil {
		return ErrorResult(err.Error())
	}

	return formatApplyPatchResult(results)
}

type patchOpKind int

const (
	patchOpAdd patchOpKind = iota
	patchOpUpdate
	patchOpDelete
)

type patchOperation struct {
	kind  patchOpKind
	path  string
	lines []string
}

type appliedPatchResult struct {
	path   string
	before []byte
	after  []byte
}

func parseApplyPatch(input string) ([]patchOperation, error) {
	lines := strings.Split(strings.ReplaceAll(input, "\r\n", "\n"), "\n")
	if len(lines) < 2 || strings.TrimSpace(lines[0]) != "*** Begin Patch" {
		return nil, fmt.Errorf("patch must start with *** Begin Patch")
	}

	var ops []patchOperation
	for i := 1; i < len(lines); {
		line := strings.TrimSpace(lines[i])
		i++

		if line == "" {
			continue
		}
		if line == "*** End Patch" {
			if len(ops) == 0 {
				return nil, fmt.Errorf("patch contains no operations")
			}
			return ops, nil
		}

		kind, path, err := parsePatchHeader(line)
		if err != nil {
			return nil, err
		}
		op := patchOperation{kind: kind, path: path}

		for i < len(lines) {
			next := strings.TrimSpace(lines[i])
			if strings.HasPrefix(next, "*** Add File: ") ||
				strings.HasPrefix(next, "*** Update File: ") ||
				strings.HasPrefix(next, "*** Delete File: ") ||
				next == "*** End Patch" {
				break
			}
			if strings.HasPrefix(next, "*** Move to: ") {
				return nil, fmt.Errorf("apply_patch does not support move operations yet")
			}
			op.lines = append(op.lines, lines[i])
			i++
		}

		ops = append(ops, op)
	}

	return nil, fmt.Errorf("patch must end with *** End Patch")
}

func parsePatchHeader(line string) (patchOpKind, string, error) {
	for _, candidate := range []struct {
		prefix string
		kind   patchOpKind
	}{
		{"*** Add File: ", patchOpAdd},
		{"*** Update File: ", patchOpUpdate},
		{"*** Delete File: ", patchOpDelete},
	} {
		if strings.HasPrefix(line, candidate.prefix) {
			path := strings.TrimSpace(strings.TrimPrefix(line, candidate.prefix))
			if path == "" {
				return candidate.kind, "", fmt.Errorf("patch operation path is required")
			}
			return candidate.kind, path, nil
		}
	}
	return patchOpAdd, "", fmt.Errorf("unsupported patch header: %s", line)
}

func applyPatchOperations(sysFs fileSystem, ops []patchOperation) ([]appliedPatchResult, error) {
	results := make([]appliedPatchResult, 0, len(ops))
	for _, op := range ops {
		result, err := applyPatchOperation(sysFs, op)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	return results, nil
}

func applyPatchOperation(sysFs fileSystem, op patchOperation) (appliedPatchResult, error) {
	switch op.kind {
	case patchOpAdd:
		after, err := patchAddedContent(op.lines)
		if err != nil {
			return appliedPatchResult{}, fmt.Errorf("%s: %w", op.path, err)
		}
		if _, err := sysFs.ReadFile(op.path); err == nil {
			return appliedPatchResult{}, fmt.Errorf("%s: file already exists", op.path)
		} else if !isNotExistError(err) {
			return appliedPatchResult{}, err
		}
		if err := sysFs.WriteFile(op.path, after); err != nil {
			return appliedPatchResult{}, err
		}
		return appliedPatchResult{path: op.path, after: after}, nil

	case patchOpDelete:
		before, err := sysFs.ReadFile(op.path)
		if err != nil {
			return appliedPatchResult{}, err
		}
		if err := sysFs.RemoveFile(op.path); err != nil {
			return appliedPatchResult{}, err
		}
		return appliedPatchResult{path: op.path, before: before}, nil

	case patchOpUpdate:
		before, err := sysFs.ReadFile(op.path)
		if err != nil {
			return appliedPatchResult{}, err
		}
		after, err := patchUpdatedContent(before, op.lines)
		if err != nil {
			return appliedPatchResult{}, fmt.Errorf("%s: %w", op.path, err)
		}
		if err := sysFs.WriteFile(op.path, after); err != nil {
			return appliedPatchResult{}, err
		}
		return appliedPatchResult{path: op.path, before: before, after: after}, nil
	default:
		return appliedPatchResult{}, fmt.Errorf("%s: unknown patch operation", op.path)
	}
}

func isNotExistError(err error) bool {
	return errors.Is(err, fs.ErrNotExist) ||
		(err != nil && strings.Contains(strings.ToLower(err.Error()), "file not found"))
}

func patchAddedContent(lines []string) ([]byte, error) {
	var out []string
	for _, line := range lines {
		if line == "" {
			out = append(out, "")
			continue
		}
		if !strings.HasPrefix(line, "+") {
			return nil, fmt.Errorf("add file lines must start with +")
		}
		out = append(out, strings.TrimPrefix(line, "+"))
	}
	return []byte(strings.Join(out, "\n") + "\n"), nil
}

func patchUpdatedContent(before []byte, lines []string) ([]byte, error) {
	blocks := splitPatchUpdateBlocks(lines)
	if len(blocks) == 0 {
		return nil, fmt.Errorf("update operation has no hunks")
	}

	content := string(before)
	for _, block := range blocks {
		oldText, newText, err := patchBlockTexts(block)
		if err != nil {
			return nil, err
		}
		if oldText == "" {
			return nil, fmt.Errorf("update hunk has no removable/context content")
		}
		count := strings.Count(content, oldText)
		if count == 0 {
			return nil, fmt.Errorf("hunk context not found")
		}
		if count > 1 {
			return nil, fmt.Errorf("hunk context appears %d times; add more context", count)
		}
		content = strings.Replace(content, oldText, newText, 1)
	}
	return []byte(content), nil
}

func splitPatchUpdateBlocks(lines []string) [][]string {
	var blocks [][]string
	var current []string
	seenHunk := false

	for _, line := range lines {
		if strings.HasPrefix(line, "@@") {
			if seenHunk && len(current) > 0 {
				blocks = append(blocks, current)
			}
			current = nil
			seenHunk = true
			continue
		}
		if !seenHunk && strings.TrimSpace(line) == "" {
			continue
		}
		seenHunk = true
		current = append(current, line)
	}
	if len(current) > 0 {
		blocks = append(blocks, current)
	}
	return blocks
}

func patchBlockTexts(lines []string) (string, string, error) {
	var oldLines []string
	var newLines []string

	for _, line := range lines {
		if line == `\ No newline at end of file` {
			continue
		}
		if line == "" {
			return "", "", fmt.Errorf("update hunk lines must start with space, -, or +")
		}

		prefix := line[0]
		text := line[1:] + "\n"
		switch prefix {
		case ' ':
			oldLines = append(oldLines, text)
			newLines = append(newLines, text)
		case '-':
			oldLines = append(oldLines, text)
		case '+':
			newLines = append(newLines, text)
		default:
			return "", "", fmt.Errorf("update hunk lines must start with space, -, or +")
		}
	}

	return strings.Join(oldLines, ""), strings.Join(newLines, ""), nil
}

func formatApplyPatchResult(results []appliedPatchResult) *ToolResult {
	var userParts []string
	var llmParts []string

	for _, result := range results {
		diff := DiffResult(result.path, result.before, result.after)
		if diff.ForUser != "" {
			userParts = append(userParts, diff.ForUser)
		}
		if diff.ForLLM != "" {
			llmParts = append(llmParts, diff.ForLLM)
		}
	}

	if len(llmParts) == 0 {
		llmParts = append(llmParts, fmt.Sprintf("Applied patch to %d file(s)", len(results)))
	}

	return &ToolResult{
		ForLLM:  strings.Join(llmParts, "\n\n"),
		ForUser: strings.Join(userParts, "\n\n"),
	}
}

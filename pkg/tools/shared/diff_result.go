package toolshared

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/pmezard/go-difflib/difflib"
)

const (
	noContentChangeDiffMessage = "(no content change)"
	noNewlineAtEOFMarker       = `\ No newline at end of file`
	diffPreviewSkippedMessage  = "[diff preview skipped: file too large for inline preview]"
	diffPreviewTruncatedNote   = "[diff preview truncated; call read_file for the full edited contents]"
	maxDiffInputBytes          = 64 * 1024
	maxDiffInputLines          = 2000
	maxUserDiffPreviewBytes    = 16 * 1024
)

// DiffResult creates a user-visible tool result containing a unified diff for
// a successful file edit. The diff is included for both the LLM and the user so
// the follow-up assistant response can reason about the resulting change set,
// including EOF newline transitions.
func DiffResult(path string, before, after []byte) *ToolResult {
	summary := fmt.Sprintf("File edited: %s", path)
	if exceedsDiffPreviewLimits(before, after) {
		return SilentResult(summary + "\n" + diffPreviewSkippedMessage)
	}

	diff, err := buildUnifiedDiff(path, before, after)
	if err != nil {
		return UserResult(fmt.Sprintf("%s\n[diff unavailable: %v]", summary, err))
	}

	userDiff, truncated := truncateDiffPreview(diff, maxUserDiffPreviewBytes)
	userContent := fmt.Sprintf("%s\n```diff\n%s\n```", summary, userDiff)
	if truncated {
		userContent += "\n" + diffPreviewTruncatedNote
	}

	llmContent := summary
	if diff == noContentChangeDiffMessage {
		llmContent = summary + "\n" + noContentChangeDiffMessage
	} else if truncated {
		llmContent = summary + "\n" + diffPreviewTruncatedNote
	}

	return &ToolResult{
		ForLLM:  llmContent,
		ForUser: userContent,
		Silent:  false,
		IsError: false,
		Async:   false,
	}
}

func buildUnifiedDiff(path string, before, after []byte) (string, error) {
	diff, err := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
		A:        splitDiffLinesPreservingEOF(before),
		B:        splitDiffLinesPreservingEOF(after),
		FromFile: "a/" + diffDisplayPath(path),
		ToFile:   "b/" + diffDisplayPath(path),
		Context:  3,
	})
	if err != nil {
		return "", err
	}

	diff = strings.TrimRight(diff, "\n")
	if diff == "" {
		return noContentChangeDiffMessage, nil
	}

	return diff, nil
}

func splitDiffLinesPreservingEOF(content []byte) []string {
	if len(content) == 0 {
		return nil
	}

	lines := make([]string, 0, bytes.Count(content, []byte{'\n'})+1)
	lineStart := 0
	for i, b := range content {
		if b != '\n' {
			continue
		}
		lines = append(lines, string(content[lineStart:i+1]))
		lineStart = i + 1
	}
	if lineStart < len(content) {
		lines = append(lines, string(content[lineStart:]))
	}

	if lacksTrailingNewline(content) {
		lines[len(lines)-1] += "\n"
		lines = append(lines, noNewlineAtEOFMarker+"\n")
	}

	return lines
}

func lacksTrailingNewline(content []byte) bool {
	return len(content) > 0 && !bytes.HasSuffix(content, []byte("\n"))
}

func exceedsDiffPreviewLimits(before, after []byte) bool {
	return len(before) > maxDiffInputBytes ||
		len(after) > maxDiffInputBytes ||
		countDiffLines(before) > maxDiffInputLines ||
		countDiffLines(after) > maxDiffInputLines
}

func countDiffLines(content []byte) int {
	if len(content) == 0 {
		return 0
	}

	lines := bytes.Count(content, []byte{'\n'})
	if !bytes.HasSuffix(content, []byte("\n")) {
		lines++
	}
	return lines
}

func truncateDiffPreview(diff string, maxBytes int) (string, bool) {
	if maxBytes <= 0 || len(diff) <= maxBytes {
		return diff, false
	}

	truncated := diff[:maxBytes]
	for len(truncated) > 0 && !utf8.ValidString(truncated) {
		truncated = truncated[:len(truncated)-1]
	}

	lastNewline := strings.LastIndexByte(truncated, '\n')
	if lastNewline > 0 {
		truncated = truncated[:lastNewline]
	}

	truncated = strings.TrimRight(truncated, "\n")
	if truncated == "" {
		truncated = diff[:maxBytes]
		for len(truncated) > 0 && !utf8.ValidString(truncated) {
			truncated = truncated[:len(truncated)-1]
		}
		truncated = strings.TrimRight(truncated, "\n")
	}

	return truncated, true
}

func diffDisplayPath(path string) string {
	displayPath := strings.TrimLeft(filepath.ToSlash(path), "/")
	if displayPath == "" {
		return "file"
	}
	return displayPath
}

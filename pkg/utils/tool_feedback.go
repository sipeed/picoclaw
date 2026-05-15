package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

const ToolFeedbackContinuationHint = "Continuing the current task."

func FormatArgsJSON(args map[string]any, prettyPrint, disableEscapeHTML bool) string {
	// Normalize nil to empty map for consistent output
	if args == nil {
		args = map[string]any{}
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	if prettyPrint {
		enc.SetIndent("", "  ")
	}
	if disableEscapeHTML {
		enc.SetEscapeHTML(false)
	}
	if err := enc.Encode(args); err != nil {
		// Fallback to fmt.Sprintf to preserve visibility of problematic args
		return fmt.Sprintf("%v", args)
	}
	return strings.TrimSpace(buf.String())
}

const ToolFeedbackStyleWorkingSummary = "working_summary"

// FormatToolFeedbackMessage renders a tool feedback message for chat channels.
// It keeps the tool name on the first line for animation and can include both
// a human explanation and the serialized tool arguments in the body.
func FormatToolFeedbackMessage(toolName, explanation, argsPreview string) string {
	toolName = strings.TrimSpace(toolName)
	explanation = strings.TrimSpace(explanation)
	argsPreview = strings.TrimSpace(argsPreview)

	bodyLines := make([]string, 0, 2)
	if explanation != "" {
		bodyLines = append(bodyLines, explanation)
	}
	if argsPreview != "" {
		bodyLines = append(bodyLines, "```json\n"+argsPreview+"\n```")
	}
	body := strings.Join(bodyLines, "\n")

	if toolName == "" {
		return body
	}
	if body == "" {
		return fmt.Sprintf("\U0001f527 `%s`", toolName)
	}

	return fmt.Sprintf("\U0001f527 `%s`\n%s", toolName, body)
}

// FormatToolFeedbackMessageWithStyle renders alternate tool feedback styles.
// The working_summary style intentionally omits explanations and raw arguments:
// progress UI should not leak internal paths, large JSON blobs, secrets, or
// model-drafted reasoning-like text.
func FormatToolFeedbackMessageWithStyle(style, toolName, explanation, argsPreview string) string {
	return FormatToolFeedbackMessageWithStyleAndTitle(style, "", toolName, explanation, argsPreview)
}

func FormatToolFeedbackMessageWithStyleAndTitle(
	style, title, toolName, explanation, argsPreview string,
) string {
	if strings.EqualFold(strings.TrimSpace(style), ToolFeedbackStyleWorkingSummary) {
		return FormatWorkingSummaryToolFeedbackMessageWithTitle(title, toolName, argsPreview)
	}
	return FormatToolFeedbackMessage(toolName, explanation, argsPreview)
}

func FormatWorkingSummaryToolFeedbackMessage(toolName, argsPreview string) string {
	return FormatWorkingSummaryToolFeedbackMessageWithTitle("", toolName, argsPreview)
}

func FormatWorkingSummaryToolFeedbackMessageWithTitle(title, toolName, argsPreview string) string {
	toolName = strings.TrimSpace(toolName)
	header := workingSummaryHeader(title)
	if toolName == "" {
		return header
	}
	line := fmt.Sprintf("• tool: `%s`", sanitizeToolFeedbackCodeSpan(toolName))
	if summary := summarizeToolFeedbackArgs(toolName, argsPreview); summary != "" {
		line += fmt.Sprintf(" — `%s`", sanitizeToolFeedbackCodeSpan(summary))
	}
	return header + "\n" + line
}

func workingSummaryHeader(title string) string {
	title = strings.TrimSpace(title)
	if title == "" {
		return "Working..."
	}
	return title + " working..."
}

func sanitizeToolFeedbackCodeSpan(text string) string {
	return strings.ReplaceAll(text, "`", "'")
}

func summarizeToolFeedbackArgs(toolName, argsPreview string) string {
	argsPreview = strings.TrimSpace(argsPreview)
	if argsPreview == "" {
		return ""
	}

	var args map[string]any
	if err := json.Unmarshal([]byte(argsPreview), &args); err != nil {
		return ""
	}

	normalizedToolName := strings.ToLower(strings.TrimSpace(toolName))
	if strings.Contains(normalizedToolName, "exec") {
		if command := firstStringArg(args, "command"); command != "" {
			return truncateToolFeedbackSummary(summarizeExecCommand(command))
		}
		return ""
	}
	if isFileToolFeedbackTool(normalizedToolName) {
		if summary := firstStringArg(args, "path", "file_path", "filepath"); summary != "" {
			return truncateToolFeedbackSummary(summarizeFilePath(summary))
		}
	}
	return ""
}

func isFileToolFeedbackTool(toolName string) bool {
	return strings.Contains(toolName, "read_file") ||
		strings.Contains(toolName, "write_file") ||
		strings.Contains(toolName, "edit_file") ||
		strings.Contains(toolName, "append_file") ||
		strings.Contains(toolName, "list_dir")
}

func firstStringArg(args map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := args[key]; ok {
			if s, ok := value.(string); ok {
				if normalized := normalizeToolFeedbackSummary(s); normalized != "" {
					return normalized
				}
			}
		}
	}
	return ""
}

func normalizeToolFeedbackSummary(text string) string {
	return redactToolFeedbackSecrets(strings.Join(strings.Fields(text), " "))
}

func summarizeFilePath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}

	cleaned := filepath.Clean(path)
	slashed := filepath.ToSlash(cleaned)
	if idx := strings.LastIndex(slashed, "/workspace/"); idx >= 0 {
		relative := strings.TrimPrefix(slashed[idx+len("/workspace/"):], "/")
		if relative != "" && relative != "." {
			return relative
		}
	}

	if !filepath.IsAbs(cleaned) {
		return slashed
	}
	return filepath.Base(cleaned)
}

func truncateToolFeedbackSummary(text string) string {
	const maxRunes = 96
	runes := []rune(text)
	if len(runes) <= maxRunes {
		return text
	}
	return strings.TrimSpace(string(runes[:maxRunes-3])) + "..."
}

func summarizeExecCommand(command string) string {
	fields := strings.Fields(command)
	for i := 0; i < len(fields); i++ {
		token := strings.Trim(fields[i], `"'`)
		if token == "" || isShellAssignment(token) {
			continue
		}
		switch token {
		case "env", "command", "time", "timeout", "sudo":
			continue
		case "bash", "sh", "zsh", "fish", "python", "python3", "node", "deno", "bun", "uv", "uvx", "npx":
			for j := i + 1; j < len(fields); j++ {
				next := strings.Trim(fields[j], `"'`)
				if next == "" || strings.HasPrefix(next, "-") || isShellAssignment(next) {
					continue
				}
				return filepath.Base(next)
			}
			return token
		default:
			return filepath.Base(token)
		}
	}
	return ""
}

func isShellAssignment(token string) bool {
	return strings.Contains(token, "=") && !strings.HasPrefix(token, "/") && !strings.HasPrefix(token, ".")
}

var (
	toolFeedbackSecretValuePatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\bsk-(?:proj|or-v1)?-[A-Za-z0-9_-]{16,}`),
		regexp.MustCompile(`\bAIza[A-Za-z0-9_-]{20,}`),
		regexp.MustCompile(`\bmat_[A-Za-z0-9_-]{16,}`),
		regexp.MustCompile(`\b\d{6,}:[A-Za-z0-9_-]{20,}`),
		regexp.MustCompile(`\b(?:ghp|github_pat|glpat|xox[baprs])_[A-Za-z0-9_-]{16,}`),
	}
	toolFeedbackSecretKVPattern   = regexp.MustCompile(`(?i)(\b(?:api[_-]?key|access[_-]?token|auth[_-]?token|token|secret|password|authorization)\b\s*[=:]\s*)("[^"]*"|'[^']*'|[^\s&]+)`)
	toolFeedbackSecretFlagPattern = regexp.MustCompile(`(?i)(--(?:api-key|access-token|auth-token|token|secret|password|authorization)(?:=|\s+))("[^"]*"|'[^']*'|[^\s&]+)`)
)

func redactToolFeedbackSecrets(text string) string {
	if text == "" {
		return ""
	}
	for _, pattern := range toolFeedbackSecretValuePatterns {
		text = pattern.ReplaceAllString(text, "[redacted]")
	}
	text = toolFeedbackSecretKVPattern.ReplaceAllString(text, "${1}[redacted]")
	text = toolFeedbackSecretFlagPattern.ReplaceAllString(text, "${1}[redacted]")
	return text
}

// FitToolFeedbackMessage keeps tool feedback within a single outbound message.
// It preserves the first line when possible and truncates the explanation body
// instead of letting the message be split into multiple chunks.
func FitToolFeedbackMessage(content string, maxLen int) string {
	content = strings.TrimSpace(content)
	if content == "" || maxLen <= 0 {
		return ""
	}
	if len([]rune(content)) <= maxLen {
		return content
	}

	firstLine, rest, hasRest := strings.Cut(content, "\n")
	firstLine = strings.TrimSpace(firstLine)
	rest = strings.TrimSpace(rest)

	if !hasRest || rest == "" {
		return Truncate(firstLine, maxLen)
	}

	if len([]rune(firstLine)) >= maxLen {
		return Truncate(firstLine, maxLen)
	}

	remaining := maxLen - len([]rune(firstLine)) - 1
	if remaining <= 0 {
		return Truncate(firstLine, maxLen)
	}

	return firstLine + "\n" + Truncate(rest, remaining)
}

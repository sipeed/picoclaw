package utils

import (
	"fmt"
	"strings"
)

const ToolFeedbackContinuationHint = "Continuing the current task."

// FormatToolFeedbackMessage renders the model-provided explanation for why a
// tool is being executed. When the model does not provide one, it keeps only
// the tool line and does not expose raw arguments or fallback text.
func FormatToolFeedbackMessage(toolName, explanation string) string {
	toolName = strings.TrimSpace(toolName)
	explanation = strings.TrimSpace(explanation)

	if toolName == "" {
		return explanation
	}
	if explanation == "" {
		return fmt.Sprintf("\U0001f527 `%s`", toolName)
	}

	return fmt.Sprintf("\U0001f527 `%s`\n%s", toolName, explanation)
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

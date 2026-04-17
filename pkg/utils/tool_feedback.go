package utils

import (
	"fmt"
	"strings"
)

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

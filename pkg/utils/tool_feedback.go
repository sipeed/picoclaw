package utils

import "fmt"

// FormatToolFeedbackMessage renders the tool name and arguments preview in the
// same markdown shape used by live tool feedback and session reconstruction.
func FormatToolFeedbackMessage(toolName, argsPreview string) string {
	return fmt.Sprintf("\U0001f527 `%s`\n```\n%s\n```", toolName, argsPreview)
}

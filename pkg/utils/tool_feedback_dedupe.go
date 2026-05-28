package utils

import (
	"strings"

	"github.com/sipeed/picoclaw/pkg/providers"
)

func normalizeToolFeedbackComparisonText(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	return strings.Join(strings.Fields(text), " ")
}

func ToolCallExplanationDuplicatesContent(content string, toolCalls []providers.ToolCall) bool {
	normalizedContent := normalizeToolFeedbackComparisonText(content)
	if normalizedContent == "" || len(toolCalls) == 0 {
		return false
	}

	for _, tc := range toolCalls {
		if tc.ExtraContent == nil {
			continue
		}
		explanation := normalizeToolFeedbackComparisonText(tc.ExtraContent.ToolFeedbackExplanation)
		if explanation == "" {
			continue
		}
		if explanation == normalizedContent {
			return true
		}
	}

	return false
}

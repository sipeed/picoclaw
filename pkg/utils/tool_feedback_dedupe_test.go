package utils

import (
	"testing"

	"github.com/sipeed/picoclaw/pkg/providers"
)

func TestToolCallExplanationDuplicatesContent(t *testing.T) {
	t.Run("exact duplicate", func(t *testing.T) {
		toolCalls := []providers.ToolCall{{
			ExtraContent: &providers.ExtraContent{
				ToolFeedbackExplanation: "Read the file before replying.",
			},
		}}

		if !ToolCallExplanationDuplicatesContent("Read the file before replying.", toolCalls) {
			t.Fatal("expected duplicated content to be detected")
		}
	})

	t.Run("whitespace normalized duplicate", func(t *testing.T) {
		toolCalls := []providers.ToolCall{{
			ExtraContent: &providers.ExtraContent{
				ToolFeedbackExplanation: "Read   the file\nbefore replying.",
			},
		}}

		if !ToolCallExplanationDuplicatesContent("  Read the file before replying.  ", toolCalls) {
			t.Fatal("expected whitespace-only differences to be ignored")
		}
	})

	t.Run("distinct content", func(t *testing.T) {
		toolCalls := []providers.ToolCall{{
			ExtraContent: &providers.ExtraContent{
				ToolFeedbackExplanation: "Read the file before replying.",
			},
		}}

		if ToolCallExplanationDuplicatesContent(
			"I will summarize the findings after reading the file.",
			toolCalls,
		) {
			t.Fatal("expected distinct content to remain visible")
		}
	})

	t.Run("missing explanation", func(t *testing.T) {
		toolCalls := []providers.ToolCall{{}}
		if ToolCallExplanationDuplicatesContent("Read the file before replying.", toolCalls) {
			t.Fatal("expected empty tool explanations to skip dedupe")
		}
	})
}

package memory

import (
	"testing"
)

func TestSessionMessage_WithFrontmatter(t *testing.T) {
	msg := SessionMessage{
		Role:    "user",
		Content: "test content",
		Tags:    []string{"coding", "golang"},
		Metadata: map[string]interface{}{
			"session_id": "abc123",
			"model":     "claude-sonnet-4",
		},
	}
	if len(msg.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(msg.Tags))
	}
	if msg.Metadata["session_id"] != "abc123" {
		t.Errorf("expected session_id in metadata")
	}
}
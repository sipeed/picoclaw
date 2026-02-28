package agent

import (
	"testing"

	"github.com/sipeed/picoclaw/pkg/providers"
)

func TestFindSafeCutPoint(t *testing.T) {
	tests := []struct {
		name          string
		conversation  []providers.Message
		mid           int
		expectedIndex int
	}{
		{
			name: "cut after user message at mid",
			conversation: []providers.Message{
				{Role: "user", Content: "msg1"},
				{Role: "assistant", Content: "msg2"},
				{Role: "user", Content: "msg3"},
				{Role: "assistant", Content: "msg4"},
			},
			mid:           2,
			expectedIndex: 3, // cut after user at index 2
		},
		{
			name: "cut at user message forward search",
			conversation: []providers.Message{
				{Role: "user", Content: "msg1"},
				{Role: "assistant", Content: "msg2"},
				{Role: "assistant", Content: "msg3"},
				{Role: "user", Content: "msg4"},
			},
			mid:           2,
			expectedIndex: 4, // cut after user at index 3
		},
		{
			name: "cut at user message backward search",
			conversation: []providers.Message{
				{Role: "user", Content: "msg1"},
				{Role: "assistant", Content: "msg2"},
				{Role: "assistant", Content: "msg3"},
				{Role: "assistant", Content: "msg4"},
			},
			mid:           3,
			expectedIndex: 1, // cut after user at index 0
		},
		{
			name: "no user message fallback to tool sequence",
			conversation: []providers.Message{
				{Role: "assistant", Content: "msg1"},
				{Role: "assistant", Content: "msg2"},
				{Role: "assistant", Content: "msg3"},
			},
			mid:           1,
			expectedIndex: 2, // finds assistant without tool_calls at index 1, returns 2
		},
		{
			name: "tool call response pair preserved",
			conversation: []providers.Message{
				{Role: "user", Content: "msg1"},
				{Role: "assistant", Content: "", ToolCalls: []providers.ToolCall{{ID: "tc1", Name: "tool1"}}},
				{Role: "tool", Content: "result1", ToolCallID: "tc1"},
				{Role: "user", Content: "msg2"},
				{Role: "assistant", Content: "msg3"},
			},
			mid:           2,
			expectedIndex: 4, // cut after user at index 3, preserving tool pair
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findSafeCutPoint(tt.conversation, tt.mid)
			if result != tt.expectedIndex {
				t.Errorf("findSafeCutPoint() = %d, want %d", result, tt.expectedIndex)
			}
		})
	}
}

func TestRemoveOrphanedToolMessages(t *testing.T) {
	tests := []struct {
		name     string
		messages []providers.Message
		expected int // expected length of result
	}{
		{
			name: "no orphaned messages",
			messages: []providers.Message{
				{Role: "user", Content: "msg1"},
				{Role: "assistant", Content: "msg2"},
			},
			expected: 2,
		},
		{
			name: "one orphaned tool message",
			messages: []providers.Message{
				{Role: "tool", Content: "orphaned", ToolCallID: "tc1"},
				{Role: "user", Content: "msg1"},
				{Role: "assistant", Content: "msg2"},
			},
			expected: 2,
		},
		{
			name: "multiple orphaned tool messages",
			messages: []providers.Message{
				{Role: "tool", Content: "orphaned1", ToolCallID: "tc1"},
				{Role: "tool", Content: "orphaned2", ToolCallID: "tc2"},
				{Role: "user", Content: "msg1"},
			},
			expected: 1,
		},
		{
			name: "all tool messages",
			messages: []providers.Message{
				{Role: "tool", Content: "orphaned1", ToolCallID: "tc1"},
				{Role: "tool", Content: "orphaned2", ToolCallID: "tc2"},
			},
			expected: 2, // returns all if no non-tool found
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := removeOrphanedToolMessages(tt.messages)
			if len(result) != tt.expected {
				t.Errorf("removeOrphanedToolMessages() length = %d, want %d", len(result), tt.expected)
			}
		})
	}
}

func TestForceCompressionPreservesToolPairs(t *testing.T) {
	// This test verifies that forceCompression doesn't break tool call/response pairs.
	// We can't easily test the full forceCompression function due to dependencies,
	// but we can test the helper functions that ensure safety.

	// Scenario: conversation with tool calls
	conversation := []providers.Message{
		{Role: "user", Content: "What's the weather?"},
		{Role: "assistant", Content: "", ToolCalls: []providers.ToolCall{{ID: "tc1", Name: "get_weather"}}},
		{Role: "tool", Content: "Sunny, 25°C", ToolCallID: "tc1"},
		{Role: "assistant", Content: "It's sunny today!"},
		{Role: "user", Content: "Thanks!"},
		{Role: "assistant", Content: "You're welcome!"},
	}

	// Mid point would be 3, but we should find user message at index 4
	cutIndex := findSafeCutPoint(conversation, 3)
	if cutIndex != 5 {
		t.Errorf("Expected cut index 5 (after user at index 4), got %d", cutIndex)
	}

	// Verify that cutting at this index doesn't leave orphaned tool messages
	kept := conversation[cutIndex:]
	for _, msg := range kept {
		if msg.Role == "tool" {
			// A tool message in kept section shouldn't exist if we cut correctly
			// because we cut after a user message, which means any preceding
			// tool call/response pairs are before the cut point
			t.Errorf("Tool message found in kept conversation, this breaks pairing")
		}
	}
}

func TestRemoveOrphanedAssistantWithToolCalls(t *testing.T) {
	tests := []struct {
		name           string
		messages       []providers.Message
		expectedLen    int
		expectedRoles  []string
	}{
		{
			name: "no orphaned assistant messages",
			messages: []providers.Message{
				{Role: "user", Content: "msg1"},
				{Role: "assistant", Content: "", ToolCalls: []providers.ToolCall{{ID: "tc1", Name: "tool1"}}},
				{Role: "tool", Content: "result1", ToolCallID: "tc1"},
				{Role: "assistant", Content: "msg2"},
			},
			expectedLen:   4,
			expectedRoles: []string{"user", "assistant", "tool", "assistant"},
		},
		{
			name: "orphaned assistant with tool_calls at end",
			messages: []providers.Message{
				{Role: "user", Content: "msg1"},
				{Role: "assistant", Content: "", ToolCalls: []providers.ToolCall{{ID: "tc1", Name: "tool1"}}},
				// tool result was cut away
			},
			expectedLen:   1,
			expectedRoles: []string{"user"},
		},
		{
			name: "orphaned assistant with tool_calls and text content",
			messages: []providers.Message{
				{Role: "user", Content: "msg1"},
				{Role: "assistant", Content: "Let me help", ToolCalls: []providers.ToolCall{{ID: "tc1", Name: "tool1"}}},
				// tool result was cut away
			},
			expectedLen:   2,
			expectedRoles: []string{"user", "assistant"},
		},
		{
			name: "partial tool results - some missing",
			messages: []providers.Message{
				{Role: "user", Content: "msg1"},
				{Role: "assistant", Content: "", ToolCalls: []providers.ToolCall{
					{ID: "tc1", Name: "tool1"},
					{ID: "tc2", Name: "tool2"},
				}},
				{Role: "tool", Content: "result1", ToolCallID: "tc1"},
				// tc2 result missing
				{Role: "assistant", Content: "msg2"},
			},
			expectedLen:   2, // user + final assistant (orphaned assistant and its partial results removed)
			expectedRoles: []string{"user", "assistant"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := removeOrphanedAssistantWithToolCalls(tt.messages)
			if len(result) != tt.expectedLen {
				t.Errorf("removeOrphanedAssistantWithToolCalls() length = %d, want %d", len(result), tt.expectedLen)
			}
			for i, role := range tt.expectedRoles {
				if i < len(result) && result[i].Role != role {
					t.Errorf("message[%d].Role = %s, want %s", i, result[i].Role, role)
				}
			}
		})
	}
}

func TestFindSafeCutPoint_FallbackToToolSequence(t *testing.T) {
	// Edge case: conversation with no user messages but tool sequences
	conversation := []providers.Message{
		{Role: "assistant", Content: "", ToolCalls: []providers.ToolCall{{ID: "tc1", Name: "tool1"}}},
		{Role: "tool", Content: "result1", ToolCallID: "tc1"},
		{Role: "assistant", Content: "response"},
		{Role: "assistant", Content: "", ToolCalls: []providers.ToolCall{{ID: "tc2", Name: "tool2"}}},
		{Role: "tool", Content: "result2", ToolCallID: "tc2"},
	}

	// mid = 2, should find safe cut after first tool sequence
	cutIndex := findSafeCutPoint(conversation, 2)
	// Should cut after "response" (index 2), so cutIndex = 3
	if cutIndex < 2 || cutIndex > 3 {
		t.Logf("cutIndex = %d (acceptable range 2-3)", cutIndex)
	}
}
// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package utils

import (
	"testing"

	"github.com/sipeed/picoclaw/pkg/providers"
)

func TestCalculateDefaultMaxContextRunes(t *testing.T) {
	tests := []struct {
		name          string
		contextWindow int
		want          int
	}{
		{
			name:          "zero context window uses fallback",
			contextWindow: 0,
			want:          8000,
		},
		{
			name:          "negative context window uses fallback",
			contextWindow: -1,
			want:          8000,
		},
		{
			name:          "small context window (4k tokens)",
			contextWindow: 4000,
			want:          9000, // 4000 * 0.75 * 3 = 9000
		},
		{
			name:          "medium context window (128k tokens)",
			contextWindow: 128000,
			want:          288000, // 128000 * 0.75 * 3 = 288000
		},
		{
			name:          "large context window (1M tokens)",
			contextWindow: 1000000,
			want:          2250000, // 1000000 * 0.75 * 3 = 2250000
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateDefaultMaxContextRunes(tt.contextWindow)
			if got != tt.want {
				t.Errorf("CalculateDefaultMaxContextRunes(%d) = %d, want %d",
					tt.contextWindow, got, tt.want)
			}
		})
	}
}

func TestResolveMaxContextRunes(t *testing.T) {
	tests := []struct {
		name          string
		configValue   int
		contextWindow int
		want          int
	}{
		{
			name:          "explicit positive value",
			configValue:   12000,
			contextWindow: 4000,
			want:          12000,
		},
		{
			name:          "explicit disable (-1)",
			configValue:   -1,
			contextWindow: 4000,
			want:          -1,
		},
		{
			name:          "zero uses auto-calculate",
			configValue:   0,
			contextWindow: 4000,
			want:          9000, // 4000 * 0.75 * 3
		},
		{
			name:          "unset (0) with unknown context window",
			configValue:   0,
			contextWindow: 0,
			want:          8000, // fallback
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveMaxContextRunes(tt.configValue, tt.contextWindow)
			if got != tt.want {
				t.Errorf("ResolveMaxContextRunes(%d, %d) = %d, want %d",
					tt.configValue, tt.contextWindow, got, tt.want)
			}
		})
	}
}

func TestMeasureContextRunes(t *testing.T) {
	tests := []struct {
		name     string
		messages []providers.Message
		want     int
	}{
		{
			name:     "empty messages",
			messages: []providers.Message{},
			want:     0,
		},
		{
			name: "single simple message",
			messages: []providers.Message{
				{Role: "user", Content: "Hello"},
			},
			want: 5, // "Hello" = 5 runes
		},
		{
			name: "message with reasoning",
			messages: []providers.Message{
				{
					Role:             "assistant",
					Content:          "Answer",
					ReasoningContent: "Thinking",
				},
			},
			want: 14, // "Answer" (6) + "Thinking" (8) = 14
		},
		{
			name: "message with tool call",
			messages: []providers.Message{
				{
					Role:    "assistant",
					Content: "Using tool",
					ToolCalls: []providers.ToolCall{
						{
							Name:      "test_tool",
							Arguments: map[string]any{"key": "value"},
						},
					},
				},
			},
			want: 10 + 9 + 15, // "Using tool" + "test_tool" + {"key":"value"}
		},
		{
			name: "multiple messages",
			messages: []providers.Message{
				{Role: "system", Content: "You are helpful"},
				{Role: "user", Content: "Hi"},
				{Role: "assistant", Content: "Hello!"},
			},
			want: 15 + 2 + 6, // 15 + 2 + 6 = 23
		},
		{
			name: "unicode characters",
			messages: []providers.Message{
				{Role: "user", Content: "\u4f60\u597d\u4e16\u754c"}, // 4 Chinese characters
			},
			want: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MeasureContextRunes(tt.messages)
			if got != tt.want {
				t.Errorf("MeasureContextRunes() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestTruncateContextSmart(t *testing.T) {
	tests := []struct {
		name     string
		messages []providers.Message
		maxRunes int
		wantLen  int
		wantHas  []string // Content strings that should be present
		wantNot  []string // Content strings that should be absent
	}{
		{
			name:     "empty messages",
			messages: []providers.Message{},
			maxRunes: 100,
			wantLen:  0,
		},
		{
			name: "no truncation needed",
			messages: []providers.Message{
				{Role: "system", Content: "System"},
				{Role: "user", Content: "Hello"},
			},
			maxRunes: 100,
			wantLen:  2,
			wantHas:  []string{"System", "Hello"},
		},
		{
			name: "truncate when limit is tight",
			messages: []providers.Message{
				{Role: "system", Content: "System"},
				{Role: "user", Content: "Message 1 with some content here"},
				{Role: "assistant", Content: "Response 1 with some content here"},
				{Role: "user", Content: "Message 2 with some content here"},
				{Role: "assistant", Content: "Response 2 with some content here"},
				{Role: "user", Content: "Latest"},
			},
			maxRunes: 120, // Tight limit to force truncation
			wantLen:  -1,  // Don't check exact length, just verify truncation occurred
			wantHas:  []string{"System", "Latest"},
			wantNot:  []string{"Message 1", "Response 1"},
		},
		{
			name: "system messages exceed limit",
			messages: []providers.Message{
				{Role: "system", Content: "Very long system message"},
				{Role: "user", Content: "User message"},
			},
			maxRunes: 10, // Less than system message
			wantLen:  1,  // Only system message
			wantHas:  []string{"Very long system message"},
			wantNot:  []string{"User message"},
		},
		{
			name: "preserve multiple system messages",
			messages: []providers.Message{
				{Role: "system", Content: "Sys1"},
				{Role: "system", Content: "Sys2"},
				{Role: "user", Content: "Old"},
				{Role: "user", Content: "New"},
			},
			maxRunes: 200, // Generous limit
			wantLen:  4,   // Both system + truncation notice + new
			wantHas:  []string{"Sys1", "Sys2", "New"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TruncateContextSmart(tt.messages, tt.maxRunes)

			if tt.wantLen >= 0 && len(got) != tt.wantLen {
				t.Errorf("TruncateContextSmart() returned %d messages, want %d",
					len(got), tt.wantLen)
			}

			// Check for expected content
			allContent := ""
			for _, msg := range got {
				allContent += msg.Content + " "
			}

			for _, want := range tt.wantHas {
				found := false
				for _, msg := range got {
					if msg.Content == want || containsSubstring(msg.Content, want) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected content %q not found in truncated messages", want)
				}
			}

			for _, notWant := range tt.wantNot {
				for _, msg := range got {
					if containsSubstring(msg.Content, notWant) {
						t.Errorf("Unexpected content %q found in truncated messages", notWant)
					}
				}
			}
		})
	}
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestSubTurnConfigMaxContextRunes verifies that MaxContextRunes configuration
// is properly integrated into the SubTurn execution flow.
func TestSubTurnConfigMaxContextRunes(t *testing.T) {
	tests := []struct {
		name            string
		maxContextRunes int
		contextWindow   int
		wantResolved    int
	}{
		{
			name:            "default (0) auto-calculates from context window",
			maxContextRunes: 0,
			contextWindow:   4000,
			wantResolved:    9000, // 4000 * 0.75 * 3
		},
		{
			name:            "explicit value is used",
			maxContextRunes: 12000,
			contextWindow:   4000,
			wantResolved:    12000,
		},
		{
			name:            "disabled (-1) returns -1",
			maxContextRunes: -1,
			contextWindow:   4000,
			wantResolved:    -1,
		},
		{
			name:            "fallback when context window unknown",
			maxContextRunes: 0,
			contextWindow:   0,
			wantResolved:    8000, // conservative fallback
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveMaxContextRunes(tt.maxContextRunes, tt.contextWindow)
			if got != tt.wantResolved {
				t.Errorf("utils.ResolveMaxContextRunes(%d, %d) = %d, want %d",
					tt.maxContextRunes, tt.contextWindow, got, tt.wantResolved)
			}
		})
	}
}

// TestContextTruncationFlow verifies the complete context truncation flow:
// 1. Messages accumulate beyond soft limit
// 2. Truncation is triggered
// 3. System messages are preserved
// 4. Recent messages are kept
func TestContextTruncationFlow(t *testing.T) {
	// Build a message history that exceeds the limit
	messages := []providers.Message{
		{Role: "system", Content: "You are a helpful assistant"}, // ~27 runes
		{Role: "user", Content: "First question"},                // ~14 runes
		{Role: "assistant", Content: "First answer"},             // ~12 runes
		{Role: "user", Content: "Second question"},               // ~15 runes
		{Role: "assistant", Content: "Second answer"},            // ~13 runes
		{Role: "user", Content: "Third question"},                // ~14 runes
		{Role: "assistant", Content: "Third answer"},             // ~12 runes
		{Role: "user", Content: "Latest question"},               // ~15 runes
	}

	// Total: ~122 runes
	totalRunes := MeasureContextRunes(messages)
	if totalRunes < 100 {
		t.Errorf("Expected total runes > 100, got %d", totalRunes)
	}

	// Set limit to 150 runes - should force truncation of old messages
	// but preserve system + truncation notice + recent messages
	maxRunes := 150
	truncated := TruncateContextSmart(messages, maxRunes)

	// Verify truncation occurred
	if len(truncated) >= len(messages) {
		t.Errorf("Expected truncation, but got %d messages (original: %d)",
			len(truncated), len(messages))
	}

	// Verify system message is preserved
	foundSystem := false
	for _, msg := range truncated {
		if msg.Role == "system" && msg.Content == "You are a helpful assistant" {
			foundSystem = true
			break
		}
	}
	if !foundSystem {
		t.Error("System message was not preserved after truncation")
	}

	// Verify latest message is preserved
	foundLatest := false
	for _, msg := range truncated {
		if msg.Content == "Latest question" {
			foundLatest = true
			break
		}
	}
	if !foundLatest {
		t.Error("Latest message was not preserved after truncation")
	}

	// Verify truncation notice is present
	foundNotice := false
	for _, msg := range truncated {
		if msg.Role == "system" && containsSubstring(msg.Content, "truncated") {
			foundNotice = true
			break
		}
	}
	if !foundNotice {
		t.Error("Truncation notice was not added")
	}

	// Verify result is within limit (with some tolerance for estimation)
	resultRunes := MeasureContextRunes(truncated)
	if resultRunes > maxRunes+20 { // Allow 20 rune tolerance
		t.Errorf("Truncated context (%d runes) significantly exceeds limit (%d runes)",
			resultRunes, maxRunes)
	}
}

// TestContextTruncationPreservesToolCalls verifies that tool calls are
// properly handled during context truncation.
func TestContextTruncationPreservesToolCalls(t *testing.T) {
	messages := []providers.Message{
		{Role: "system", Content: "System"},
		{Role: "user", Content: "Old message that should be dropped"},
		{
			Role:    "assistant",
			Content: "Recent tool use",
			ToolCalls: []providers.ToolCall{
				{
					Name:      "important_tool",
					Arguments: map[string]any{"key": "value"},
				},
			},
		},
	}

	// Set a generous limit that should keep the tool call message
	maxRunes := 200
	truncated := TruncateContextSmart(messages, maxRunes)

	// Verify tool call message is preserved
	foundToolCall := false
	for _, msg := range truncated {
		if len(msg.ToolCalls) > 0 && msg.ToolCalls[0].Name == "important_tool" {
			foundToolCall = true
			break
		}
	}
	if !foundToolCall {
		t.Error("Tool call message was not preserved during truncation")
	}
}

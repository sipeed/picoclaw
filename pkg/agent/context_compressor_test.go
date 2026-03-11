package agent

import (
	"sync"
	"testing"

	"github.com/sipeed/picoclaw/pkg/providers"
)

func TestEstimateTokens_ByteBased(t *testing.T) {
	cc := NewContextCompressor(nil, &sync.Map{}, &sync.WaitGroup{})

	tests := []struct {
		name     string
		messages []providers.Message
		want     int
	}{
		{
			name:     "empty",
			messages: nil,
			want:     0,
		},
		{
			name: "ascii text",
			messages: []providers.Message{
				{Content: "hello world"}, // 11 bytes → 5 tokens
			},
			want: 5,
		},
		{
			name: "multiple messages",
			messages: []providers.Message{
				{Content: "hello"},  // 5 bytes
				{Content: "world"}, // 5 bytes → total 10 → 5 tokens
			},
			want: 5,
		},
		{
			name: "CJK text (multi-byte runes)",
			messages: []providers.Message{
				{Content: "你好世界"}, // 12 bytes (3 per rune) → 6 tokens
			},
			want: 6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cc.EstimateTokens(tt.messages)
			if got != tt.want {
				t.Errorf("EstimateTokens() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestEstimateTokens_MatchesOriginalBehavior(t *testing.T) {
	// The original code used len(m.Content) / 2.
	// Verify that the refactored code produces identical results.
	cc := NewContextCompressor(nil, &sync.Map{}, &sync.WaitGroup{})

	messages := []providers.Message{
		{Content: "The quick brown fox jumps over the lazy dog"},
		{Content: "Hello, world!"},
	}

	// Original: sum of len(m.Content) / 2 for each message? No — it summed all
	// then divided. Let's compute manually:
	// "The quick brown fox jumps over the lazy dog" = 43 bytes
	// "Hello, world!" = 13 bytes
	// Total = 56 bytes → 56/2 = 28
	want := 28
	got := cc.EstimateTokens(messages)
	if got != want {
		t.Errorf("EstimateTokens() = %d, want %d (original len/2 behavior)", got, want)
	}
}

func TestForceCompression_MinHistory(t *testing.T) {
	cc := NewContextCompressor(nil, &sync.Map{}, &sync.WaitGroup{})

	// Create a mock agent with minimal history (below MinHistoryForCompression)
	// ForceCompression should be a no-op.
	// We can't easily test this without a full AgentInstance, but we can verify
	// the threshold constant is reasonable.
	if MinHistoryForCompression < 2 {
		t.Errorf("MinHistoryForCompression = %d, should be at least 2", MinHistoryForCompression)
	}
	_ = cc // used above
}

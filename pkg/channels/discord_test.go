package channels

import (
	"strings"
	"testing"
)

func TestSplitMessage(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		limit    int
		expected int
	}{
		{
			name:     "Empty string",
			content:  "",
			limit:    2000,
			expected: 0,
		},
		{
			name:     "Short message",
			content:  "Hello World",
			limit:    2000,
			expected: 1,
		},
		{
			name:     "Exact limit",
			content:  strings.Repeat("a", 2000),
			limit:    2000,
			expected: 1,
		},
		{
			name:     "Just over limit",
			content:  strings.Repeat("a", 2001),
			limit:    2000,
			expected: 2,
		},
		{
			name:     "Multi-byte characters",
			content:  strings.Repeat("你", 2005), // 2005 chars
			limit:    2000,
			expected: 2,
		},
		{
			name:     "Multiple chunks",
			content:  strings.Repeat("a", 6000),
			limit:    2000,
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunks := splitMessage(tt.content, tt.limit)
			if len(chunks) != tt.expected {
				t.Errorf("splitMessage() returned %d chunks, expected %d", len(chunks), tt.expected)
			}

			// Verify reconstruction
			reconstructed := strings.Join(chunks, "")
			if reconstructed != tt.content {
				t.Errorf("Reconstructed content does not match original. Got length %d, expected %d", len(reconstructed), len(tt.content))
			}

			// Verify each chunk size
			for i, chunk := range chunks {
				if len([]rune(chunk)) > tt.limit {
					t.Errorf("Chunk %d exceeds limit: %d > %d", i, len([]rune(chunk)), tt.limit)
				}
			}
		})
	}
}

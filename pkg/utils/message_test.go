package utils

import (
	"strings"
	"testing"
)

func TestSplitMessage(t *testing.T) {
	longText := strings.Repeat("a", 2500)
	longCode := "```go\n" + strings.Repeat("fmt.Println(\"hello\")\n", 100) + "```" // ~2100 chars

	tests := []struct {
		name         string
		content      string
		maxLen       int
		expectChunks int                                 // Check number of chunks
		checkContent func(t *testing.T, chunks []string) // Custom validation
	}{
		{
			name:         "Empty message",
			content:      "",
			maxLen:       2000,
			expectChunks: 0,
		},
		{
			name:         "Short message fits in one chunk",
			content:      "Hello world",
			maxLen:       2000,
			expectChunks: 1,
		},
		{
			name:         "MaxLen 0 (no split)",
			content:      "Hello world",
			maxLen:       0,
			expectChunks: 1,
		},
		{
			name:         "Simple split regular text",
			content:      longText,
			maxLen:       2000,
			expectChunks: 2,
			checkContent: func(t *testing.T, chunks []string) {
				if len(chunks[0]) > 2000 {
					t.Errorf("Chunk 0 too large: %d", len(chunks[0]))
				}
				if len(chunks[0])+len(chunks[1]) != len(longText) {
					t.Errorf("Total length mismatch. Got %d, want %d", len(chunks[0])+len(chunks[1]), len(longText))
				}
			},
		},
		{
			name: "Split at newline",
			content:      strings.Repeat("a", 1750) + "\n" + strings.Repeat("b", 300),
			maxLen:       2000,
			expectChunks: 2,
			checkContent: func(t *testing.T, chunks []string) {
				if len(chunks[0]) != 1750 {
					t.Errorf("Expected chunk 0 to be 1750 length (split at newline), got %d", len(chunks[0]))
				}
				if chunks[1] != strings.Repeat("b", 300) {
					t.Errorf("Chunk 1 content mismatch. Len: %d", len(chunks[1]))
				}
			},
		},
		{
			name:         "Long code block split",
			content:      "Prefix\n" + longCode,
			maxLen:       2000,
			expectChunks: 2,
			checkContent: func(t *testing.T, chunks []string) {
				if !strings.HasSuffix(chunks[0], "\n```") {
					t.Error("First chunk should end with injected closing fence")
				}
				if !strings.HasPrefix(chunks[1], "```go") {
					t.Error("Second chunk should start with injected code block header")
				}
			},
		},
		{
			name:         "Preserve Unicode characters",
			content:      strings.Repeat("\u4e16", 2500), // 2500 runes
			maxLen:       2000,
			expectChunks: 2,
			checkContent: func(t *testing.T, chunks []string) {
				// Each chunk should stay within max runes limit
				for i, chunk := range chunks {
					runeCount := len([]rune(chunk))
					if runeCount > 2000 {
						t.Errorf("Chunk %d has too many runes: %d", i, runeCount)
					}
				}
				// Verify total rune count
				totalRunes := 0
				for _, chunk := range chunks {
					totalRunes += len([]rune(chunk))
				}
				if totalRunes != 2500 {
					t.Errorf("Total rune count mismatch. Got %d, want 2500", totalRunes)
				}
			},
		},
		{
			name: "Prefer sentence boundary",
			// Content is: 1700 'a's, then ". ", then 500 'b's.
			// Effective limit with maxLen=2000 is 1800. 1700 is well within it.
			content:      strings.Repeat("a", 1700) + ". " + strings.Repeat("b", 500),
			maxLen:       2000,
			expectChunks: 2,
			checkContent: func(t *testing.T, chunks []string) {
				if len([]rune(chunks[0])) != 1701 { // 1700 'a's + '.'
					t.Errorf("Expected chunk 0 to be 1701 runes (split at period), got %d %q", len([]rune(chunks[0])), chunks[0])
				}
				if !strings.HasSuffix(chunks[0], ".") {
					t.Errorf("Chunk 0 should end with a period, got suffix: %q", chunks[0][len(chunks[0])-5:])
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := SplitMessage(tc.content, tc.maxLen)

			if tc.expectChunks == 0 {
				if len(got) != 0 {
					t.Errorf("Expected 0 chunks, got %d", len(got))
				}
				return
			}

			if len(got) != tc.expectChunks {
				t.Errorf("Expected %d chunks, got %d", tc.expectChunks, len(got))
				// Log sizes for debugging
				for i, c := range got {
					t.Logf("Chunk %d length: %d", i, len(c))
				}
				return // Stop further checks if count assumes specific split
			}

			if tc.checkContent != nil {
				tc.checkContent(t, got)
			}
		})
	}
}

func TestSplitMessage_CodeBlockIntegrity(t *testing.T) {
	// Focused test for the core requirement: splitting inside a code block preserves syntax highlighting

	// 60 chars total approximately
	content := "```go\npackage main\n\nfunc main() {\n\tprintln(\"Hello\")\n}\n```"
	maxLen := 40

	chunks := SplitMessage(content, maxLen)

	if len(chunks) != 2 {
		t.Fatalf("Expected 2 chunks, got %d: %q", len(chunks), chunks)
	}

	// First chunk must end with "\n```"
	if !strings.HasSuffix(chunks[0], "\n```") {
		t.Errorf("First chunk should end with closing fence. Got: %q", chunks[0])
	}

	// Second chunk must start with the header "```go"
	if !strings.HasPrefix(chunks[1], "```go") {
		t.Errorf("Second chunk should start with code block header. Got: %q", chunks[1])
	}

	// First chunk should contain meaningful content
	if len(chunks[0]) > 40 {
		t.Errorf("First chunk exceeded maxLen: length %d", len(chunks[0]))
	}
}

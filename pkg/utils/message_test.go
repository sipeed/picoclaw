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
			name:         "Simple split regular text",
			content:      longText,
			maxLen:       2000,
			expectChunks: 2,
			checkContent: func(t *testing.T, chunks []string) {
				if len([]rune(chunks[0])) > 2000 {
					t.Errorf("Chunk 0 too large: %d runes", len([]rune(chunks[0])))
				}
				if len([]rune(chunks[0]))+len([]rune(chunks[1])) != len([]rune(longText)) {
					t.Errorf(
						"Total rune length mismatch. Got %d, want %d",
						len([]rune(chunks[0]))+len([]rune(chunks[1])),
						len([]rune(longText)),
					)
				}
			},
		},
		{
			name: "Split at newline",
			// 1750 chars then newline, then more chars.
			// Dynamic buffer: 2000 / 10 = 200.
			// Effective limit: 2000 - 200 = 1800.
			// Split should happen at newline because it's at 1750 (< 1800).
			// Total length must > 2000 to trigger split. 1750 + 1 + 300 = 2051.
			content:      strings.Repeat("a", 1750) + "\n" + strings.Repeat("b", 300),
			maxLen:       2000,
			expectChunks: 2,
			checkContent: func(t *testing.T, chunks []string) {
				if len([]rune(chunks[0])) != 1750 {
					t.Errorf("Expected chunk 0 to be 1750 runes (split at newline), got %d", len([]rune(chunks[0])))
				}
				if chunks[1] != strings.Repeat("b", 300) {
					t.Errorf("Chunk 1 content mismatch. Len: %d", len([]rune(chunks[1])))
				}
			},
		},
		{
			name:         "Long code block split",
			content:      "Prefix\n" + longCode,
			maxLen:       2000,
			expectChunks: 2,
			checkContent: func(t *testing.T, chunks []string) {
				// Check that first chunk ends with closing fence
				if !strings.HasSuffix(chunks[0], "\n```") {
					t.Error("First chunk should end with injected closing fence")
				}
				// Check that second chunk starts with execution header
				if !strings.HasPrefix(chunks[1], "```go") {
					t.Error("Second chunk should start with injected code block header")
				}
			},
		},
		{
			name:         "Preserve Unicode characters (rune-aware)",
			content:      strings.Repeat("\u4e16", 2500), // 2500 runes, 7500 bytes
			maxLen:       2000,
			expectChunks: 2,
			checkContent: func(t *testing.T, chunks []string) {
				// Verify chunks contain valid unicode and don't split mid-rune
				for i, chunk := range chunks {
					runeCount := len([]rune(chunk))
					if runeCount > 2000 {
						t.Errorf("Chunk %d has %d runes, exceeds maxLen 2000", i, runeCount)
					}
					if !strings.Contains(chunk, "\u4e16") {
						t.Errorf("Chunk %d should contain unicode characters", i)
					}
				}
				// Verify total rune count is preserved
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
			name:         "Zero maxLen returns single chunk",
			content:      "Hello world",
			maxLen:       0,
			expectChunks: 1,
			checkContent: func(t *testing.T, chunks []string) {
				if chunks[0] != "Hello world" {
					t.Errorf("Expected original content, got %q", chunks[0])
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
	if len([]rune(chunks[0])) > 40 {
		t.Errorf("First chunk exceeded maxLen: length %d runes", len([]rune(chunks[0])))
	}
}

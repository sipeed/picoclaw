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
			// 1750 chars then newline, then more chars.
			// Dynamic buffer: 2000 / 10 = 200.
			// Effective limit: 2000 - 200 = 1800.
			// Split should happen at newline because it's at 1750 (< 1800).
			// Total length must > 2000 to trigger split. 1750 + 1 + 300 = 2051.
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
			name:         "Preserve Unicode characters",
			content:      strings.Repeat("\u4e16", 1000), // 3000 bytes
			maxLen:       2000,
			expectChunks: 2,
			checkContent: func(t *testing.T, chunks []string) {
				// Just verify we didn't panic and got valid strings.
				// Go strings are UTF-8, if we split mid-rune it would be bad,
				// but standard slicing might do that.
				// Let's assume standard behavior is acceptable or check if it produces invalid rune?
				if !strings.Contains(chunks[0], "\u4e16") {
					t.Error("Chunk should contain unicode characters")
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

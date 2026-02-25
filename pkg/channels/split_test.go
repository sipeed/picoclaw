package channels

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

// --- Helper function tests for index-based rune operations ---

func TestFindLastNewlineInRange(t *testing.T) {
	runes := []rune("aaa\nbbb\nccc")
	// Indices:        0123 4567 89 10

	tests := []struct {
		name         string
		start, end   int
		searchWindow int
		want         int
	}{
		{"finds last newline in full range", 0, 11, 200, 7},
		{"finds newline within search window", 0, 11, 4, 7},
		{"narrow window misses newline outside window", 4, 11, 3, 3}, // returns start-1 (not found)
		{"no newline in range", 0, 3, 200, -1},                       // start-1 = -1
		{"range limited to first segment", 0, 4, 200, 3},
		{"search window of 1 at newline", 0, 8, 1, 7},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := findLastNewlineInRange(runes, tc.start, tc.end, tc.searchWindow)
			if got != tc.want {
				t.Errorf("findLastNewlineInRange(runes, %d, %d, %d) = %d, want %d",
					tc.start, tc.end, tc.searchWindow, got, tc.want)
			}
		})
	}
}

func TestFindLastSpaceInRange(t *testing.T) {
	runes := []rune("abc def\tghi")
	// Indices:        0123 4567 89 10

	tests := []struct {
		name         string
		start, end   int
		searchWindow int
		want         int
	}{
		{"finds tab as last space/tab", 0, 11, 200, 7},
		{"finds space when tab out of window", 0, 7, 200, 3},
		{"no space in range", 0, 3, 200, -1},
		{"narrow window finds tab", 5, 11, 4, 7},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := findLastSpaceInRange(runes, tc.start, tc.end, tc.searchWindow)
			if got != tc.want {
				t.Errorf("findLastSpaceInRange(runes, %d, %d, %d) = %d, want %d",
					tc.start, tc.end, tc.searchWindow, got, tc.want)
			}
		})
	}
}

func TestFindNewlineFrom(t *testing.T) {
	runes := []rune("hello\nworld\n")

	tests := []struct {
		name string
		from int
		want int
	}{
		{"from start", 0, 5},
		{"from after first newline", 6, 11},
		{"from past all newlines", 12, -1},
		{"from newline itself", 5, 5},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := findNewlineFrom(runes, tc.from)
			if got != tc.want {
				t.Errorf("findNewlineFrom(runes, %d) = %d, want %d", tc.from, got, tc.want)
			}
		})
	}
}

func TestFindLastUnclosedCodeBlockInRange(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		start, end int
		want       int
	}{
		{
			name:    "no code blocks",
			content: "hello world",
			start:   0, end: 11,
			want: -1,
		},
		{
			name:    "complete code block",
			content: "```go\ncode\n```",
			start:   0, end: 14,
			want: -1,
		},
		{
			name:    "unclosed code block",
			content: "text\n```go\ncode here",
			start:   0, end: 20,
			want: 5,
		},
		{
			name:    "closed then unclosed",
			content: "```a\n```\n```b\ncode",
			start:   0, end: 17,
			want: 9,
		},
		{
			name:    "search within subrange",
			content: "```a\n```\n```b\ncode",
			start:   9, end: 17,
			want: 9,
		},
		{
			name:    "subrange with no code blocks",
			content: "```a\n```\nhello",
			start:   9, end: 14,
			want: -1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runes := []rune(tc.content)
			got := findLastUnclosedCodeBlockInRange(runes, tc.start, tc.end)
			if got != tc.want {
				t.Errorf("findLastUnclosedCodeBlockInRange(%q, %d, %d) = %d, want %d",
					tc.content, tc.start, tc.end, got, tc.want)
			}
		})
	}
}

func TestFindNextClosingCodeBlockInRange(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		startIdx int
		end      int
		want     int
	}{
		{
			name:     "finds closing fence",
			content:  "code\n```\nmore",
			startIdx: 0, end: 13,
			want: 8, // position after ```
		},
		{
			name:     "no closing fence",
			content:  "just code here",
			startIdx: 0, end: 14,
			want: -1,
		},
		{
			name:     "fence at start of search",
			content:  "```end",
			startIdx: 0, end: 6,
			want: 3,
		},
		{
			name:     "fence outside range",
			content:  "code\n```",
			startIdx: 0, end: 4,
			want: -1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runes := []rune(tc.content)
			got := findNextClosingCodeBlockInRange(runes, tc.startIdx, tc.end)
			if got != tc.want {
				t.Errorf("findNextClosingCodeBlockInRange(%q, %d, %d) = %d, want %d",
					tc.content, tc.startIdx, tc.end, got, tc.want)
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

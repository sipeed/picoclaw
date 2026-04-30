package slackwebhook

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConvertMarkdownToMrkdwn(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "bold double asterisk",
			input:    "This is **bold** text",
			expected: "This is *bold* text",
		},
		{
			name:     "italic single asterisk",
			input:    "This is *italic* text",
			expected: "This is _italic_ text",
		},
		{
			name:     "italic underscore",
			input:    "This is _italic_ text",
			expected: "This is _italic_ text",
		},
		{
			name:     "strikethrough",
			input:    "This is ~~struck~~ text",
			expected: "This is ~struck~ text",
		},
		{
			name:     "inline code unchanged",
			input:    "Use `code` here",
			expected: "Use `code` here",
		},
		{
			name:     "link conversion",
			input:    "Click [here](https://example.com) now",
			expected: "Click <https://example.com|here> now",
		},
		{
			name:     "header to bold",
			input:    "# Header One",
			expected: "*Header One*",
		},
		{
			name:     "header level 2",
			input:    "## Header Two",
			expected: "*Header Two*",
		},
		{
			name:     "bullet list",
			input:    "- item one\n- item two",
			expected: "• item one\n• item two",
		},
		{
			name:     "mixed formatting",
			input:    "**bold** and *italic* and [link](http://x.com)",
			expected: "*bold* and _italic_ and <http://x.com|link>",
		},
		{
			name:     "code block unchanged",
			input:    "```\ncode here\n```",
			expected: "```\ncode here\n```",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertMarkdownToMrkdwn(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSplitContentWithTables(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedCount  int
		expectedTables int
	}{
		{
			name:           "no table",
			input:          "Just some text",
			expectedCount:  1,
			expectedTables: 0,
		},
		{
			name:           "simple table",
			input:          "| A | B |\n|---|---|\n| 1 | 2 |",
			expectedCount:  1,
			expectedTables: 1,
		},
		{
			name:           "text before table",
			input:          "Intro text\n\n| A | B |\n|---|---|\n| 1 | 2 |",
			expectedCount:  2,
			expectedTables: 1,
		},
		{
			name:           "text before and after table",
			input:          "Before\n\n| A | B |\n|---|---|\n| 1 | 2 |\n\nAfter",
			expectedCount:  3,
			expectedTables: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			segments := splitContentWithTables(tt.input)
			assert.Equal(t, tt.expectedCount, len(segments))
			tableCount := 0
			for _, seg := range segments {
				if seg.isTable {
					tableCount++
				}
			}
			assert.Equal(t, tt.expectedTables, tableCount)
		})
	}
}

func TestRenderTable(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		expectCode bool
	}{
		{
			name:       "narrow table renders as text",
			input:      "| A | B |\n|---|---|\n| 1 | 2 |",
			expectCode: false,
		},
		{
			name:       "wide table renders as code block",
			input:      "| This is a very long column header | Another extremely long column header here |\n|---|---|\n| Some long value content here | More long value content |",
			expectCode: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := renderTable(tt.input)
			if tt.expectCode {
				assert.Contains(t, result, "```")
			} else {
				assert.NotContains(t, result, "```")
				assert.Contains(t, result, "*") // Bold headers
			}
		})
	}
}

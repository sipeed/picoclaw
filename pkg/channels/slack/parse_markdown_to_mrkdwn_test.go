package slack

import (
	"strings"
	"testing"

	goslack "github.com/slack-go/slack"
)

func Test_convertMessage_text(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected string
		hasTable bool
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},

		// Bold
		{
			name:     "bold double asterisk",
			input:    "this is **bold** text",
			expected: "this is *bold* text",
		},
		{
			name:     "bold double underscore",
			input:    "this is __bold__ text",
			expected: "this is *bold* text",
		},
		{
			name:     "multiple bold segments",
			input:    "**one** and **two**",
			expected: "*one* and *two*",
		},

		// Italic (passthrough)
		{
			name:     "italic unchanged",
			input:    "this is _italic_ text",
			expected: "this is _italic_ text",
		},

		// Strikethrough
		{
			name:     "strikethrough",
			input:    "this is ~~deleted~~ text",
			expected: "this is ~deleted~ text",
		},

		// Headings
		{
			name:     "h1 heading",
			input:    "# Main Title",
			expected: "*Main Title*",
		},
		{
			name:     "h3 heading",
			input:    "### Sub Heading",
			expected: "*Sub Heading*",
		},
		{
			name:     "heading with inline bold collapses",
			input:    "## **Bold Heading**",
			expected: "*Bold Heading*",
		},

		// Links
		{
			name:     "markdown link",
			input:    "visit [Google](https://google.com) today",
			expected: "visit <https://google.com|Google> today",
		},
		{
			name:     "link with special chars in text",
			input:    "[click & go](https://example.com/path?a=1&b=2)",
			expected: "<https://example.com/path?a=1&b=2|click & go>",
		},

		// Images
		{
			name:     "image to link",
			input:    "![screenshot](https://example.com/img.png)",
			expected: "<https://example.com/img.png|screenshot>",
		},
		{
			name:     "image before link",
			input:    "![img](https://a.com/1.png) and [link](https://b.com)",
			expected: "<https://a.com/1.png|img> and <https://b.com|link>",
		},

		// Code blocks (passthrough)
		{
			name:     "fenced code block preserved",
			input:    "```go\nfmt.Println(\"**not bold**\")\n```",
			expected: "```go\nfmt.Println(\"**not bold**\")\n```",
		},
		{
			name:     "inline code preserved",
			input:    "use `**bold**` syntax",
			expected: "use `**bold**` syntax",
		},

		// Blockquotes (passthrough)
		{
			name:     "blockquote unchanged",
			input:    "> this is a quote",
			expected: "> this is a quote",
		},

		// Horizontal rules
		{
			name:     "horizontal rule removed",
			input:    "above\n---\nbelow",
			expected: "above\n\nbelow",
		},
		{
			name:     "asterisk hr removed",
			input:    "above\n***\nbelow",
			expected: "above\n\nbelow",
		},

		// Unordered lists
		{
			name:     "dash list to bullet",
			input:    "- item one\n- item two",
			expected: "\u2022 item one\n\u2022 item two",
		},
		{
			name:     "asterisk list to bullet",
			input:    "* item one\n* item two",
			expected: "\u2022 item one\n\u2022 item two",
		},
		{
			name:     "nested list indentation",
			input:    "- top\n  - nested",
			expected: "\u2022 top\n  \u2022 nested",
		},

		// Ordered lists (passthrough)
		{
			name:     "ordered list unchanged",
			input:    "1. first\n2. second",
			expected: "1. first\n2. second",
		},

		// Tables → extracted as Block Kit table, removed from text
		{
			name:     "table extracted from text",
			input:    "| A | B |\n| --- | --- |\n| 1 | 2 |",
			expected: "",
			hasTable: true,
		},
		{
			name:     "table with surrounding text",
			input:    "Here is a table:\n\n| A | B |\n| --- | --- |\n| 1 | 2 |\n\nEnd.",
			expected: "Here is a table:\n\nEnd.",
			hasTable: true,
		},

		// Edge cases
		{
			name:     "single asterisk not converted",
			input:    "this * is not * bold",
			expected: "this * is not * bold",
		},
		{
			name:     "bold inside code block not converted",
			input:    "```\n**bold** inside code\n```",
			expected: "```\n**bold** inside code\n```",
		},
		{
			name:     "link inside inline code not converted",
			input:    "use `[text](url)` for links",
			expected: "use `[text](url)` for links",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			msg := convertMessage(tc.input)
			if msg.text != tc.expected {
				t.Errorf("\ninput:    %q\nexpected: %q\nactual:   %q", tc.input, tc.expected, msg.text)
			}
			if tc.hasTable && msg.table == nil {
				t.Error("expected table block, got nil")
			}
			if !tc.hasTable && msg.table != nil {
				t.Error("expected no table block, got one")
			}
		})
	}
}

func Test_convertMessage_table(t *testing.T) {
	cases := []struct {
		name        string
		input       string
		wantRows    int
		wantCols    int
		wantHeader  []string
		wantDataRow []string
	}{
		{
			name:        "simple two column",
			input:       "| Name | Age |\n| --- | --- |\n| Alice | 30 |",
			wantRows:    2,
			wantCols:    2,
			wantHeader:  []string{"Name", "Age"},
			wantDataRow: []string{"Alice", "30"},
		},
		{
			name:        "three columns",
			input:       "| A | B | C |\n| --- | --- | --- |\n| 1 | 2 | 3 |",
			wantRows:    2,
			wantCols:    3,
			wantHeader:  []string{"A", "B", "C"},
			wantDataRow: []string{"1", "2", "3"},
		},
		{
			name:        "bold stripped from cells",
			input:       "| Feature | **Status** |\n| --- | --- |\n| **Auth** | Done |",
			wantRows:    2,
			wantCols:    2,
			wantHeader:  []string{"Feature", "Status"},
			wantDataRow: []string{"Auth", "Done"},
		},
		{
			name:        "strikethrough stripped from cells",
			input:       "| Item | Status |\n| --- | --- |\n| ~~old~~ | removed |",
			wantRows:    2,
			wantCols:    2,
			wantHeader:  []string{"Item", "Status"},
			wantDataRow: []string{"old", "removed"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			msg := convertMessage(tc.input)
			if msg.table == nil {
				t.Fatal("expected table block, got nil")
			}

			if len(msg.table.Rows) != tc.wantRows {
				t.Fatalf("rows: got %d, want %d", len(msg.table.Rows), tc.wantRows)
			}

			if len(msg.table.Rows[0]) != tc.wantCols {
				t.Fatalf("cols: got %d, want %d", len(msg.table.Rows[0]), tc.wantCols)
			}

			for i, want := range tc.wantHeader {
				got := cellText(msg.table.Rows[0][i])
				if got != want {
					t.Errorf("header[%d]: got %q, want %q", i, got, want)
				}
			}

			if tc.wantDataRow != nil && len(msg.table.Rows) > 1 {
				for i, want := range tc.wantDataRow {
					got := cellText(msg.table.Rows[1][i])
					if got != want {
						t.Errorf("data[0][%d]: got %q, want %q", i, got, want)
					}
				}
			}
		})
	}
}

func Test_convertMessage_multipleTablesSecondFallsBack(t *testing.T) {
	input := "| A | B |\n| - | - |\n| 1 | 2 |\n\ntext\n\n| C | D |\n| - | - |\n| 3 | 4 |"
	msg := convertMessage(input)

	if msg.table == nil {
		t.Fatal("expected first table as Block Kit table")
	}

	if !strings.Contains(msg.text, "```\nC | D") {
		t.Errorf("expected second table as code block in text, got: %q", msg.text)
	}
}

func Test_splitTableRow(t *testing.T) {
	cases := []struct {
		input    string
		expected []string
	}{
		{"| A | B |", []string{"A", "B"}},
		{"| one | two | three |", []string{"one", "two", "three"}},
		{"|  spaced  |  cells  |", []string{"spaced", "cells"}},
		{"| **Bold** | ~~strike~~ |", []string{"Bold", "strike"}},
		{"| `code` | __under__ |", []string{"code", "under"}},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			actual := splitTableRow(tc.input)
			if len(actual) != len(tc.expected) {
				t.Fatalf("len: got %d, want %d", len(actual), len(tc.expected))
			}
			for i := range tc.expected {
				if actual[i] != tc.expected[i] {
					t.Errorf("[%d]: got %q, want %q", i, actual[i], tc.expected[i])
				}
			}
		})
	}
}

// cellText extracts the text string from a RichTextBlock table cell.
func cellText(cell *goslack.RichTextBlock) string {
	for _, elem := range cell.Elements {
		section, ok := elem.(*goslack.RichTextSection)
		if !ok {
			continue
		}
		for _, se := range section.Elements {
			textElem, ok := se.(*goslack.RichTextSectionTextElement)
			if !ok {
				continue
			}
			return textElem.Text
		}
	}
	return ""
}

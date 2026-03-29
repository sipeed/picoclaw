package slack

import "testing"

func Test_markdownToSlackMrkdwn(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected string
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

		// Tables
		{
			name: "simple table",
			input: "| Name | Age |\n| --- | --- |\n| Alice | 30 |\n| Bob | 25 |",
			expected: "```\nName | Age\nAlice | 30\nBob | 25\n```",
		},
		{
			name: "table with surrounding text",
			input: "Here is a table:\n\n| A | B |\n| --- | --- |\n| 1 | 2 |\n\nEnd.",
			expected: "Here is a table:\n\n```\nA | B\n1 | 2\n```\n\nEnd.",
		},

		// Mixed content
		{
			name: "realistic LLM output",
			input: "## Summary\n\nHere are the **key points**:\n\n- First item with `code`\n- Second item with [a link](https://example.com)\n- ~~Removed~~ item\n\n### Details\n\n| Feature | Status |\n| --- | --- |\n| Auth | Done |\n| API | WIP |\n\n> Note: check the docs.",
			expected: "*Summary*\n\nHere are the *key points*:\n\n\u2022 First item with `code`\n\u2022 Second item with <https://example.com|a link>\n\u2022 ~Removed~ item\n\n*Details*\n\n```\nFeature | Status\nAuth | Done\nAPI | WIP\n```\n\n> Note: check the docs.",
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
			actual := markdownToSlackMrkdwn(tc.input)
			if actual != tc.expected {
				t.Errorf("\ninput:    %q\nexpected: %q\nactual:   %q", tc.input, tc.expected, actual)
			}
		})
	}
}

func Test_convertTables(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no table",
			input:    "just some text",
			expected: "just some text",
		},
		{
			name:     "simple two column",
			input:    "| A | B |\n| --- | --- |\n| 1 | 2 |",
			expected: "```\nA | B\n1 | 2\n```",
		},
		{
			name:     "three columns with alignment markers",
			input:    "| Left | Center | Right |\n| :--- | :---: | ---: |\n| a | b | c |",
			expected: "```\nLeft | Center | Right\na | b | c\n```",
		},
		{
			name:     "multiple tables separated by text",
			input:    "| A | B |\n| - | - |\n| 1 | 2 |\n\ntext\n\n| C | D |\n| - | - |\n| 3 | 4 |",
			expected: "```\nA | B\n1 | 2\n```\n\ntext\n\n```\nC | D\n3 | 4\n```",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual := convertTables(tc.input)
			if actual != tc.expected {
				t.Errorf("\ninput:    %q\nexpected: %q\nactual:   %q", tc.input, tc.expected, actual)
			}
		})
	}
}

func Test_formatTableRow(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"| A | B |", "A | B"},
		{"| one | two | three |", "one | two | three"},
		{"|  spaced  |  cells  |", "spaced | cells"},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			actual := formatTableRow(tc.input)
			if actual != tc.expected {
				t.Errorf("formatTableRow(%q) = %q, want %q", tc.input, actual, tc.expected)
			}
		})
	}
}

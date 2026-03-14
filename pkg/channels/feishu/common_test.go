package feishu

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
)

func TestExtractJSONStringField(t *testing.T) {
	tests := []struct {
		name    string
		content string
		field   string
		want    string
	}{
		{
			name:    "valid field",
			content: `{"image_key": "img_v2_xxx"}`,
			field:   "image_key",
			want:    "img_v2_xxx",
		},
		{
			name:    "missing field",
			content: `{"image_key": "img_v2_xxx"}`,
			field:   "file_key",
			want:    "",
		},
		{
			name:    "invalid JSON",
			content: `not json at all`,
			field:   "image_key",
			want:    "",
		},
		{
			name:    "empty content",
			content: "",
			field:   "image_key",
			want:    "",
		},
		{
			name:    "non-string field value",
			content: `{"count": 42}`,
			field:   "count",
			want:    "",
		},
		{
			name:    "empty string value",
			content: `{"image_key": ""}`,
			field:   "image_key",
			want:    "",
		},
		{
			name:    "multiple fields",
			content: `{"file_key": "file_xxx", "file_name": "test.pdf"}`,
			field:   "file_name",
			want:    "test.pdf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractJSONStringField(tt.content, tt.field)
			if got != tt.want {
				t.Errorf("extractJSONStringField(%q, %q) = %q, want %q", tt.content, tt.field, got, tt.want)
			}
		})
	}
}

func TestExtractImageKey(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "normal",
			content: `{"image_key": "img_v2_abc123"}`,
			want:    "img_v2_abc123",
		},
		{
			name:    "missing key",
			content: `{"file_key": "file_xxx"}`,
			want:    "",
		},
		{
			name:    "malformed JSON",
			content: `{broken`,
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractImageKey(tt.content)
			if got != tt.want {
				t.Errorf("extractImageKey(%q) = %q, want %q", tt.content, got, tt.want)
			}
		})
	}
}

func TestExtractFileKey(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "normal",
			content: `{"file_key": "file_v2_abc123", "file_name": "test.doc"}`,
			want:    "file_v2_abc123",
		},
		{
			name:    "missing key",
			content: `{"image_key": "img_xxx"}`,
			want:    "",
		},
		{
			name:    "malformed JSON",
			content: `not json`,
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractFileKey(tt.content)
			if got != tt.want {
				t.Errorf("extractFileKey(%q) = %q, want %q", tt.content, got, tt.want)
			}
		})
	}
}

func TestExtractFileName(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "normal",
			content: `{"file_key": "file_xxx", "file_name": "report.pdf"}`,
			want:    "report.pdf",
		},
		{
			name:    "missing name",
			content: `{"file_key": "file_xxx"}`,
			want:    "",
		},
		{
			name:    "malformed JSON",
			content: `{bad`,
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractFileName(tt.content)
			if got != tt.want {
				t.Errorf("extractFileName(%q) = %q, want %q", tt.content, got, tt.want)
			}
		})
	}
}

func TestBuildMarkdownCard(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{
			name:    "normal content",
			content: "Hello **world**",
		},
		{
			name:    "empty content",
			content: "",
		},
		{
			name:    "special characters",
			content: `Code: "foo" & <bar> 'baz'`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := buildMarkdownCard(tt.content)
			if err != nil {
				t.Fatalf("buildMarkdownCard(%q) unexpected error: %v", tt.content, err)
			}

			// Verify valid JSON
			var parsed map[string]any
			if err := json.Unmarshal([]byte(result), &parsed); err != nil {
				t.Fatalf("buildMarkdownCard(%q) produced invalid JSON: %v", tt.content, err)
			}

			// Verify schema
			if parsed["schema"] != "2.0" {
				t.Errorf("schema = %v, want %q", parsed["schema"], "2.0")
			}

			// Verify body.elements[0].content == input
			body, ok := parsed["body"].(map[string]any)
			if !ok {
				t.Fatal("missing body in card JSON")
			}
			elements, ok := body["elements"].([]any)
			if !ok || len(elements) == 0 {
				t.Fatal("missing or empty elements in card JSON")
			}
			elem, ok := elements[0].(map[string]any)
			if !ok {
				t.Fatal("first element is not an object")
			}
			if elem["tag"] != "markdown" {
				t.Errorf("tag = %v, want %q", elem["tag"], "markdown")
			}
			if elem["content"] != tt.content {
				t.Errorf("content = %v, want %q", elem["content"], tt.content)
			}
		})
	}
}

func TestStripMentionPlaceholders(t *testing.T) {
	strPtr := func(s string) *string { return &s }

	tests := []struct {
		name     string
		content  string
		mentions []*larkim.MentionEvent
		want     string
	}{
		{
			name:     "no mentions",
			content:  "Hello world",
			mentions: nil,
			want:     "Hello world",
		},
		{
			name:    "single mention",
			content: "@_user_1 hello",
			mentions: []*larkim.MentionEvent{
				{Key: strPtr("@_user_1")},
			},
			want: "hello",
		},
		{
			name:    "multiple mentions",
			content: "@_user_1 @_user_2 hey",
			mentions: []*larkim.MentionEvent{
				{Key: strPtr("@_user_1")},
				{Key: strPtr("@_user_2")},
			},
			want: "hey",
		},
		{
			name:     "empty content",
			content:  "",
			mentions: []*larkim.MentionEvent{{Key: strPtr("@_user_1")}},
			want:     "",
		},
		{
			name:     "empty mentions slice",
			content:  "@_user_1 test",
			mentions: []*larkim.MentionEvent{},
			want:     "@_user_1 test",
		},
		{
			name:    "mention with nil key",
			content: "@_user_1 test",
			mentions: []*larkim.MentionEvent{
				{Key: nil},
			},
			want: "test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripMentionPlaceholders(tt.content, tt.mentions)
			if got != tt.want {
				t.Errorf("stripMentionPlaceholders(%q, ...) = %q, want %q", tt.content, got, tt.want)
			}
		})
	}
}

func TestSplitContentByTableCount(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		wantParts []string // Expected content of each part
	}{
		{
			name:    "no tables - single part",
			content: "Just some text without tables",
			wantParts: []string{
				"Just some text without tables",
			},
		},
		{
			name: "single table - single part",
			content: `| Col1 | Col2 |
|------|------|
| Data | Data |`,
			wantParts: []string{
				`| Col1 | Col2 |
|------|------|
| Data | Data |`,
			},
		},
		{
			name:      "exactly 5 tables - single part",
			content:   generateTableContent(1, 5),
			wantParts: []string{generateTableContent(1, 5)},
		},
		{
			name:      "6 tables - split into 2 parts",
			content:   generateTableContent(1, 6),
			wantParts: []string{generateTableContent(1, 5), generateTableContent(6, 6)},
		},
		{
			name: "text before and between tables",
			content: `Intro text

| T1C1 | T1C2 |
|------|------|
| Data | Data |

Middle text

| T2C1 | T2C2 |
|------|------|
| Data | Data |

| T3C1 | T3C2 |
|------|------|
| Data | Data |

| T4C1 | T4C2 |
|------|------|
| Data | Data |

| T5C1 | T5C2 |
|------|------|
| Data | Data |

| T6C1 | T6C2 |
|------|------|
| Data | Data |

Outro text`,
			wantParts: []string{
				`Intro text

| T1C1 | T1C2 |
|------|------|
| Data | Data |

Middle text

| T2C1 | T2C2 |
|------|------|
| Data | Data |

| T3C1 | T3C2 |
|------|------|
| Data | Data |

| T4C1 | T4C2 |
|------|------|
| Data | Data |

| T5C1 | T5C2 |
|------|------|
| Data | Data |`,
				`| T6C1 | T6C2 |
|------|------|
| Data | Data |

Outro text`,
			},
		},
		{
			name:    "10 tables - split into 2 parts",
			content: generateTableContent(1, 10),
			wantParts: []string{
				generateTableContent(1, 5),
				generateTableContent(6, 10),
			},
		},
		{
			name:    "11 tables - split into 3 parts",
			content: generateTableContent(1, 11),
			wantParts: []string{
				generateTableContent(1, 5),
				generateTableContent(6, 10),
				generateTableContent(11, 11),
			},
		},
		{
			name: "6 tables - verify tables are not truncated at split boundary",
			content: `| T1C1 | T1C2 |
|------|------|
| D1 | D1 |

| T2C1 | T2C2 |
|------|------|
| D2 | D2 |

| T3C1 | T3C2 |
|------|------|
| D3 | D3 |

| T4C1 | T4C2 |
|------|------|
| D4 | D4 |

| T5C1 | T5C2 |
|------|------|
| D5 | D5 |

| T6C1 | T6C2 |
|------|------|
| D6 | D6 |`,
			wantParts: []string{
				`| T1C1 | T1C2 |
|------|------|
| D1 | D1 |

| T2C1 | T2C2 |
|------|------|
| D2 | D2 |

| T3C1 | T3C2 |
|------|------|
| D3 | D3 |

| T4C1 | T4C2 |
|------|------|
| D4 | D4 |

| T5C1 | T5C2 |
|------|------|
| D5 | D5 |`,
				`| T6C1 | T6C2 |
|------|------|
| D6 | D6 |`,
			},
		},
		{
			name: "consecutive tables without blank line between them",
			content: `| T1C1 | T1C2 |
|------|------|
| D1 | D1 |
| T2C1 | T2C2 |
|------|------|
| D2 | D2 |
| T3C1 | T3C2 |
|------|------|
| D3 | D3 |
| T4C1 | T4C2 |
|------|------|
| D4 | D4 |
| T5C1 | T5C2 |
|------|------|
| D5 | D5 |
| T6C1 | T6C2 |
|------|------|
| D6 | D6 |`,
			wantParts: []string{
				`| T1C1 | T1C2 |
|------|------|
| D1 | D1 |
| T2C1 | T2C2 |
|------|------|
| D2 | D2 |
| T3C1 | T3C2 |
|------|------|
| D3 | D3 |
| T4C1 | T4C2 |
|------|------|
| D4 | D4 |
| T5C1 | T5C2 |
|------|------|
| D5 | D5 |`,
				`| T6C1 | T6C2 |
|------|------|
| D6 | D6 |`,
			},
		},
		{
			name: "content ends with table (no trailing newline or blank line)",
			content: `| T1C1 | T1C2 |
|------|------|
| D1 | D1 |

| T2C1 | T2C2 |
|------|------|
| D2 | D2 |

| T3C1 | T3C2 |
|------|------|
| D3 | D3 |

| T4C1 | T4C2 |
|------|------|
| D4 | D4 |

| T5C1 | T5C2 |
|------|------|
| D5 | D5 |

| T6C1 | T6C2 |
|------|------|
| D6 | D6 |`,
			wantParts: []string{
				`| T1C1 | T1C2 |
|------|------|
| D1 | D1 |

| T2C1 | T2C2 |
|------|------|
| D2 | D2 |

| T3C1 | T3C2 |
|------|------|
| D3 | D3 |

| T4C1 | T4C2 |
|------|------|
| D4 | D4 |

| T5C1 | T5C2 |
|------|------|
| D5 | D5 |`,
				`| T6C1 | T6C2 |
|------|------|
| D6 | D6 |`,
			},
		},
		{
			name: "table followed by text without blank line",
			content: `| T1C1 | T1C2 |
|------|------|
| D1 | D1 |
Some text immediately after table

| T2C1 | T2C2 |
|------|------|
| D2 | D2 |

| T3C1 | T3C2 |
|------|------|
| D3 | D3 |

| T4C1 | T4C2 |
|------|------|
| D4 | D4 |

| T5C1 | T5C2 |
|------|------|
| D5 | D5 |

| T6C1 | T6C2 |
|------|------|
| D6 | D6 |`,
			wantParts: []string{
				`| T1C1 | T1C2 |
|------|------|
| D1 | D1 |
Some text immediately after table

| T2C1 | T2C2 |
|------|------|
| D2 | D2 |

| T3C1 | T3C2 |
|------|------|
| D3 | D3 |

| T4C1 | T4C2 |
|------|------|
| D4 | D4 |

| T5C1 | T5C2 |
|------|------|
| D5 | D5 |`,
				`| T6C1 | T6C2 |
|------|------|
| D6 | D6 |`,
			},
		},
		{
			name: "text before 6th table goes to part 1",
			content: `| T1 | T2 |
|----|----|
| D1 | D1 |

| T1 | T2 |
|----|----|
| D2 | D2 |

| T1 | T2 |
|----|----|
| D3 | D3 |

| T1 | T2 |
|----|----|
| D4 | D4 |

| T1 | T2 |
|----|----|
| D5 | D5 |

Intro text for table 6

| T1 | T2 |
|----|----|
| D6 | D6 |`,
			wantParts: []string{
				`| T1 | T2 |
|----|----|
| D1 | D1 |

| T1 | T2 |
|----|----|
| D2 | D2 |

| T1 | T2 |
|----|----|
| D3 | D3 |

| T1 | T2 |
|----|----|
| D4 | D4 |

| T1 | T2 |
|----|----|
| D5 | D5 |

Intro text for table 6`,
				`| T1 | T2 |
|----|----|
| D6 | D6 |`,
			},
		},
		{
			name: "single row table (header only, no data rows)",
			content: `| T1C1 | T1C2 |
|------|------|

| T2C1 | T2C2 |
|------|------|

| T3C1 | T3C2 |
|------|------|

| T4C1 | T4C2 |
|------|------|

| T5C1 | T5C2 |
|------|------|

| T6C1 | T6C2 |
|------|------|`,
			wantParts: []string{
				`| T1C1 | T1C2 |
|------|------|

| T2C1 | T2C2 |
|------|------|

| T3C1 | T3C2 |
|------|------|

| T4C1 | T4C2 |
|------|------|

| T5C1 | T5C2 |
|------|------|`,
				`| T6C1 | T6C2 |
|------|------|`,
			},
		},
		{
			name: "table with alignment colons in separator",
			content: `| Left | Center | Right |
|:-----|:------:|------:|
| L1 | C1 | R1 |

| Left | Center | Right |
|:-----|:------:|------:|
| L2 | C2 | R2 |

| Left | Center | Right |
|:-----|:------:|------:|
| L3 | C3 | R3 |

| Left | Center | Right |
|:-----|:------:|------:|
| L4 | C4 | R4 |

| Left | Center | Right |
|:-----|:------:|------:|
| L5 | C5 | R5 |

| Left | Center | Right |
|:-----|:------:|------:|
| L6 | C6 | R6 |`,
			wantParts: []string{
				`| Left | Center | Right |
|:-----|:------:|------:|
| L1 | C1 | R1 |

| Left | Center | Right |
|:-----|:------:|------:|
| L2 | C2 | R2 |

| Left | Center | Right |
|:-----|:------:|------:|
| L3 | C3 | R3 |

| Left | Center | Right |
|:-----|:------:|------:|
| L4 | C4 | R4 |

| Left | Center | Right |
|:-----|:------:|------:|
| L5 | C5 | R5 |`,
				`| Left | Center | Right |
|:-----|:------:|------:|
| L6 | C6 | R6 |`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitContentByTableCount(tt.content)

			// Verify number of parts
			if len(got) != len(tt.wantParts) {
				t.Errorf("splitContentByTableCount() returned %d parts, want %d", len(got), len(tt.wantParts))
			}

			// Verify content matches exactly
			for i, expected := range tt.wantParts {
				if got[i] != expected {
					t.Errorf("Part %d mismatch:\ngot:\n%q\nwant:\n%q", i+1, got[i], expected)
				}
			}
		})
	}
}

// Helper function to generate content with tables from start to end (inclusive)
func generateTableContent(start, end int) string {
	var sb strings.Builder
	for i := start; i <= end; i++ {
		sb.WriteString(fmt.Sprintf("| T%dC1 | T%dC2 |\n", i, i))
		sb.WriteString("|------|------|\n")
		sb.WriteString("| Data | Data |")
		if i < end {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

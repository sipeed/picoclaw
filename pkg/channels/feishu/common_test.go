package feishu

import (
	"encoding/json"
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

func TestExtractPostImageKeys(t *testing.T) {
	tests := []struct {
		name       string
		rawContent string
		want       []string
	}{
		{
			name:       "text and image",
			rawContent: `{"title":"","content":[[{"tag":"img","image_key":"img_v3_abc","width":100,"height":100}],[{"tag":"text","text":"hello","style":[]}]]}`,
			want:       []string{"img_v3_abc"},
		},
		{
			name:       "multiple images",
			rawContent: `{"title":"","content":[[{"tag":"img","image_key":"img_1","width":100,"height":100},{"tag":"img","image_key":"img_2","width":100,"height":100}]]}`,
			want:       []string{"img_1", "img_2"},
		},
		{
			name:       "no images",
			rawContent: `{"title":"","content":[[{"tag":"text","text":"hello","style":[]}]]}`,
			want:       nil,
		},
		{
			name:       "invalid JSON",
			rawContent: `not json`,
			want:       nil,
		},
		{
			name:       "empty content",
			rawContent: `{"title":"","content":[]}`,
			want:       nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractPostImageKeys(tt.rawContent)
			if len(got) != len(tt.want) {
				t.Errorf("extractPostImageKeys() = %v, want %v", got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("extractPostImageKeys()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestExtractPostText(t *testing.T) {
	tests := []struct {
		name       string
		rawContent string
		want       string
	}{
		{
			name:       "text and image",
			rawContent: `{"title":"","content":[[{"tag":"img","image_key":"img_v3_abc","width":100,"height":100}],[{"tag":"text","text":"图里有啥","style":[]}]]}`,
			want:       "图里有啥",
		},
		{
			name:       "with title",
			rawContent: `{"title":"My Post","content":[[{"tag":"text","text":"body text","style":[]}]]}`,
			want:       "My Post\nbody text",
		},
		{
			name:       "only images returns empty",
			rawContent: `{"title":"","content":[[{"tag":"img","image_key":"img_v3_abc","width":100,"height":100}]]}`,
			want:       "",
		},
		{
			name:       "invalid JSON returns raw",
			rawContent: `not json`,
			want:       "not json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractPostText(tt.rawContent)
			if got != tt.want {
				t.Errorf("extractPostText(%q) = %q, want %q", tt.rawContent, got, tt.want)
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

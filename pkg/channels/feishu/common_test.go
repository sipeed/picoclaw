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

func TestExtractPostImageKeys(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []string
	}{
		{
			name:    "empty content",
			content: "",
			want:    nil,
		},
		{
			name:    "invalid JSON",
			content: "not json",
			want:    nil,
		},
		{
			name:    "post with no images",
			content: `{"zh_cn":{"title":"Title","content":[[{"tag":"text","text":"hello"}]]}}`,
			want:    nil,
		},
		{
			name:    "post with one image",
			content: `{"zh_cn":{"title":"","content":[[{"tag":"img","image_key":"img_v3_001"}]]}}`,
			want:    []string{"img_v3_001"},
		},
		{
			name:    "post with multiple images",
			content: `{"zh_cn":{"title":"","content":[[{"tag":"text","text":"see"},{"tag":"img","image_key":"img_001"}],[{"tag":"img","image_key":"img_002"}]]}}`,
			want:    []string{"img_001", "img_002"},
		},
		{
			name:    "post with text and image mixed in row",
			content: `{"zh_cn":{"title":"","content":[[{"tag":"text","text":"hi"},{"tag":"img","image_key":"img_mix"}]]}}`,
			want:    []string{"img_mix"},
		},
		{
			name:    "en_us locale",
			content: `{"en_us":{"title":"","content":[[{"tag":"img","image_key":"img_en"}]]}}`,
			want:    []string{"img_en"},
		},
		{
			name:    "multiple locales with distinct images",
			content: `{"zh_cn":{"title":"","content":[[{"tag":"img","image_key":"img_zh"}]]},"en_us":{"title":"","content":[[{"tag":"img","image_key":"img_en"}]]}}`,
			want:    []string{"img_zh", "img_en"},
		},
		{
			name:    "duplicate image_key across locales is deduplicated",
			content: `{"zh_cn":{"title":"","content":[[{"tag":"img","image_key":"img_same"}]]},"en_us":{"title":"","content":[[{"tag":"img","image_key":"img_same"}]]}}`,
			want:    []string{"img_same"},
		},
		{
			name:    "image with empty image_key",
			content: `{"zh_cn":{"title":"","content":[[{"tag":"img","image_key":""}]]}}`,
			want:    nil,
		},
		{
			name:    "flat format without locale wrapper",
			content: `{"title":"","content":[[{"tag":"img","image_key":"img_v3_flat","width":1826,"height":338}],[{"tag":"text","text":" check this image","style":[]}]]}`,
			want:    []string{"img_v3_flat"},
		},
		{
			name:    "flat format multiple images",
			content: `{"title":"","content":[[{"tag":"img","image_key":"img_flat_1"}],[{"tag":"img","image_key":"img_flat_2"},{"tag":"text","text":"desc"}]]}`,
			want:    []string{"img_flat_1", "img_flat_2"},
		},
		{
			name:    "flat format no images",
			content: `{"title":"Test","content":[[{"tag":"text","text":"just text"}]]}`,
			want:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractPostImageKeys(tt.content)
			if len(got) != len(tt.want) {
				t.Errorf("extractPostImageKeys() = %v, want %v", got, tt.want)
				return
			}
			// Use set comparison to avoid map iteration order dependency
			gotSet := make(map[string]bool, len(got))
			for _, v := range got {
				gotSet[v] = true
			}
			for _, v := range tt.want {
				if !gotSet[v] {
					t.Errorf("extractPostImageKeys() missing expected key %q; got %v", v, got)
				}
			}
		})
	}
}

func TestExtractCardImageKeys(t *testing.T) {
	tests := []struct {
		name             string
		content          string
		wantFeishuKeys   []string
		wantExternalURLs []string
	}{
		{
			name:             "empty content",
			content:          "",
			wantFeishuKeys:   nil,
			wantExternalURLs: nil,
		},
		{
			name:             "invalid JSON",
			content:          "not json",
			wantFeishuKeys:   nil,
			wantExternalURLs: nil,
		},
		{
			name:             "card with no images",
			content:          `{"schema":"2.0","body":{"elements":[{"tag":"markdown","content":"text"}]}}`,
			wantFeishuKeys:   nil,
			wantExternalURLs: nil,
		},
		{
			name:             "single image with img_key",
			content:          `{"elements":[{"tag":"img","img_key":"img_abc123"}]}`,
			wantFeishuKeys:   []string{"img_abc123"},
			wantExternalURLs: nil,
		},
		{
			name:             "single image with src as Feishu key",
			content:          `{"elements":[{"tag":"img","src":"img_xyz789"}]}`,
			wantFeishuKeys:   []string{"img_xyz789"},
			wantExternalURLs: nil,
		},
		{
			name:             "multiple images",
			content:          `{"elements":[{"tag":"img","img_key":"img_1"},{"tag":"div","text":{"content":"text"}},{"tag":"img","img_key":"img_2"}]}`,
			wantFeishuKeys:   []string{"img_1", "img_2"},
			wantExternalURLs: nil,
		},
		{
			name:             "nested image in columns",
			content:          `{"elements":[{"tag":"div","columns":[{"tag":"img","img_key":"img_col1"},{"tag":"img","img_key":"img_col2"}]}]}`,
			wantFeishuKeys:   []string{"img_col1", "img_col2"},
			wantExternalURLs: nil,
		},
		{
			name:             "image in action",
			content:          `{"elements":[{"tag":"action","actions":[{"tag":"img","img_key":"img_action"}]}]}`,
			wantFeishuKeys:   []string{"img_action"},
			wantExternalURLs: nil,
		},
		{
			name:             "icon element",
			content:          `{"elements":[{"tag":"icon","icon_key":"icon_123"}]}`,
			wantFeishuKeys:   []string{"icon_123"},
			wantExternalURLs: nil,
		},
		{
			name:             "complex card with text and images",
			content:          `{"header":{"title":{"content":"Title"}},"elements":[{"tag":"div","text":{"content":"Description"}},{"tag":"img","img_key":"img_main"}]}`,
			wantFeishuKeys:   []string{"img_main"},
			wantExternalURLs: nil,
		},
		{
			name:             "external URL in src",
			content:          `{"elements":[{"tag":"img","src":"https://example.com/image.png"}]}`,
			wantFeishuKeys:   nil,
			wantExternalURLs: []string{"https://example.com/image.png"},
		},
		{
			name:             "mixed Feishu keys and external URLs",
			content:          `{"elements":[{"tag":"img","img_key":"img_feishu"},{"tag":"img","src":"https://cdn.example.com/external.jpg"},{"tag":"img","src":"img_another"}]}`,
			wantFeishuKeys:   []string{"img_feishu", "img_another"},
			wantExternalURLs: []string{"https://cdn.example.com/external.jpg"},
		},
		{
			name:             "multiple external URLs",
			content:          `{"elements":[{"tag":"img","src":"https://a.com/1.png"},{"tag":"img","src":"http://b.com/2.jpg"}]}`,
			wantFeishuKeys:   nil,
			wantExternalURLs: []string{"https://a.com/1.png", "http://b.com/2.jpg"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotFeishuKeys, gotExternalURLs := extractCardImageKeys(tt.content)

			// Compare Feishu keys
			if len(gotFeishuKeys) != len(tt.wantFeishuKeys) {
				t.Errorf("extractCardImageKeys() feishuKeys = %v, want %v", gotFeishuKeys, tt.wantFeishuKeys)
				return
			}
			for i, v := range gotFeishuKeys {
				if v != tt.wantFeishuKeys[i] {
					t.Errorf("extractCardImageKeys() feishuKeys[%d] = %q, want %q", i, v, tt.wantFeishuKeys[i])
				}
			}

			// Compare external URLs
			if len(gotExternalURLs) != len(tt.wantExternalURLs) {
				t.Errorf("extractCardImageKeys() externalURLs = %v, want %v", gotExternalURLs, tt.wantExternalURLs)
				return
			}
			for i, v := range gotExternalURLs {
				if v != tt.wantExternalURLs[i] {
					t.Errorf("extractCardImageKeys() externalURLs[%d] = %q, want %q", i, v, tt.wantExternalURLs[i])
				}
			}
		})
	}
}

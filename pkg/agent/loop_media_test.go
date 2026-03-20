package agent

import (
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
)

func TestInjectImageDescriptions_ReplacesTag(t *testing.T) {
	content := "Check this out [image: photo]"
	result := injectImageDescriptions(content, []string{"a cat sitting on a table"})
	want := "Check this out [image: a cat sitting on a table]"
	if result != want {
		t.Errorf("got %q, want %q", result, want)
	}
}

func TestInjectImageDescriptions_MultipleImages(t *testing.T) {
	content := "Here are two photos [image: photo] and [image: photo]"
	result := injectImageDescriptions(content, []string{"a dog", "a cat"})
	want := "Here are two photos [image: a dog] and [image: a cat]"
	if result != want {
		t.Errorf("got %q, want %q", result, want)
	}
}

func TestInjectImageDescriptions_NoTag(t *testing.T) {
	content := "Hello"
	result := injectImageDescriptions(content, []string{"a sunset"})
	want := "Hello [image: a sunset]"
	if result != want {
		t.Errorf("got %q, want %q", result, want)
	}
}

func TestInjectImageDescriptions_EmptyContent(t *testing.T) {
	result := injectImageDescriptions("", []string{"a chart"})
	want := "[image: a chart]"
	if result != want {
		t.Errorf("got %q, want %q", result, want)
	}
}

func TestDescribeImages_NoImageCandidates(t *testing.T) {
	// When no ImageCandidates are configured, messages should pass through unchanged.
	messages := []providers.Message{
		{
			Role:    "user",
			Content: "[image: photo]",
			Media:   []string{"data:image/jpeg;base64,/9j/4AAQ"},
		},
	}
	agent := &AgentInstance{
		ImageCandidates: nil,
	}

	// With no candidates, describeImagesInMessages should still work
	// but return "description unavailable" for each image.
	al := &AgentLoop{}
	result := al.describeImagesInMessages(t.Context(), messages, agent, "", "")

	// The data URL should be removed from Media
	if len(result[0].Media) != 0 {
		t.Errorf("expected empty media, got %v", result[0].Media)
	}
	// Content should have the placeholder
	if result[0].Content != "[image: description unavailable]" {
		t.Errorf("content = %q", result[0].Content)
	}
}

func TestDescribeImages_SkippedInPlanMode(t *testing.T) {
	// isPlanPreExecution should return true for interviewing/review
	if !isPlanPreExecution("interviewing") {
		t.Error("interviewing should be pre-execution")
	}
	if !isPlanPreExecution("review") {
		t.Error("review should be pre-execution")
	}
	if isPlanPreExecution("executing") {
		t.Error("executing should not be pre-execution")
	}
	if isPlanPreExecution("") {
		t.Error("empty should not be pre-execution")
	}
}

func TestResolveImageModel_FallsToPlanModel(t *testing.T) {
	// When ImageModel is empty, should fall back to PlanModel
	model := resolveImageModel(nil, &config.AgentDefaults{
		PlanModel: "openai/gpt-5.4",
	})
	if model != "openai/gpt-5.4" {
		t.Errorf("got %q, want %q", model, "openai/gpt-5.4")
	}

	// When ImageModel is set, should use it
	model = resolveImageModel(nil, &config.AgentDefaults{
		ImageModel: "openai/gpt-5.4-nano",
		PlanModel:  "openai/gpt-5.4",
	})
	if model != "openai/gpt-5.4-nano" {
		t.Errorf("got %q, want %q", model, "openai/gpt-5.4-nano")
	}
}

func TestExtractPreview_Short(t *testing.T) {
	text := "Hello world"
	result := extractPreview(text, 500)
	if result != "Hello world" {
		t.Errorf("got %q", result)
	}
}

func TestExtractPreview_Truncated(t *testing.T) {
	text := strings.Repeat("a", 600)
	result := extractPreview(text, 500)
	if len([]rune(result)) != 503 { // 500 + "..."
		t.Errorf("len = %d, want 503", len([]rune(result)))
	}
	if !strings.HasSuffix(result, "...") {
		t.Error("should end with ...")
	}
}

func TestFormatDocumentTag(t *testing.T) {
	tag := formatDocumentTag("preview text", "/path/to/doc.md", 18)
	if !strings.Contains(tag, "preview text") {
		t.Error("should contain preview")
	}
	if !strings.Contains(tag, "/path/to/doc.md") {
		t.Error("should contain file path")
	}
	if !strings.Contains(tag, "18 pages") {
		t.Error("should contain page count")
	}
	if !strings.Contains(tag, "read_file") {
		t.Error("should contain read_file hint")
	}
}

func TestFormatDocumentTag_UnknownPages(t *testing.T) {
	tag := formatDocumentTag("preview", "/path.md", 0)
	if !strings.Contains(tag, "? pages") {
		t.Error("should show ? for unknown page count")
	}
}

func TestReplacePDFTags_NoPDF(t *testing.T) {
	al := &AgentLoop{}
	content := "Check this out [file:/path/to/audio.mp3]"
	result := al.replacePDFTags(t.Context(), content, &config.OCRConfig{Command: "echo"}, "", "", false, "auto")
	if result != content {
		t.Errorf("non-PDF should be unchanged, got %q", result)
	}
}

func TestReplacePDFTags_NoTags(t *testing.T) {
	al := &AgentLoop{}
	content := "Hello world"
	result := al.replacePDFTags(t.Context(), content, &config.OCRConfig{Command: "echo"}, "", "", false, "auto")
	if result != content {
		t.Errorf("no tags should be unchanged, got %q", result)
	}
}

func TestProcessPDFs_NilOCR(t *testing.T) {
	// When OCR is not configured, messages should pass through unchanged
	messages := []providers.Message{
		{Role: "user", Content: "Check [file:/tmp/doc.pdf]"},
	}
	al := &AgentLoop{}
	result := al.processPDFsInMessages(t.Context(), messages, nil, "", "")
	if result[0].Content != messages[0].Content {
		t.Errorf("content should be unchanged when OCR config is nil")
	}
}

func TestDetectReadingOrder(t *testing.T) {
	tests := []struct {
		content    string
		cfgDefault string
		want       string
	}{
		{"\u3053\u306e\u6587\u66f8\u3092\u8aad\u3093\u3067", "", "auto"},
		{"\u7e26\u66f8\u304d\u3067\u8aad\u3093\u3067", "", "right2left"},
		{"\u6a2a\u66f8\u304d\u306ePDF", "", "top2bottom"},
		{"vertical layout", "", "right2left"},
		{"horizontal doc", "", "top2bottom"},
		{"right2left please", "", "right2left"},
		{"top2bottom mode", "", "top2bottom"},
		{"left2right table", "", "left2right"},
		{"\u305f\u3066\u304c\u304d\u306e\u6587\u66f8", "", "right2left"},
		{"\u3088\u3053\u304c\u304d\u306e\u6587\u66f8", "", "top2bottom"},
		{"plain message", "right2left", "right2left"},
		{"plain message", "", "auto"},
		{"\u7e26\u66f8\u304d", "top2bottom", "right2left"}, // message keyword overrides config default
	}
	for _, tt := range tests {
		if got := detectReadingOrder(tt.content, tt.cfgDefault); got != tt.want {
			t.Errorf("detectReadingOrder(%q, %q) = %q, want %q", tt.content, tt.cfgDefault, got, tt.want)
		}
	}
}

func TestWantFigures(t *testing.T) {
	tests := []struct {
		content string
		want    bool
	}{
		{"check this pdf", false},
		{"extract with figures please", true},
		{"Figures included", true},
		{"figure mode", true},
		{"with images", true},
		{"", false},
	}
	for _, tt := range tests {
		if got := wantFigures(tt.content); got != tt.want {
			t.Errorf("wantFigures(%q) = %v, want %v", tt.content, got, tt.want)
		}
	}
}

package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/providers"
)

func TestPreLLM_parseResponse(t *testing.T) {
	p := &Analyser{}

	tests := []struct {
		name       string
		input      string
		wantIntent string
		wantTags   []string
		wantCot    string
	}{
		{
			name:       "valid JSON with cot_prompt",
			input:      `{"intent":"question","tags":["golang","testing"],"cot_prompt":"1. Understand the question\n2. Research the answer"}`,
			wantIntent: "question",
			wantTags:   []string{"golang", "testing"},
			wantCot:    "1. Understand the question\n2. Research the answer",
		},
		{
			name:       "JSON with markdown fences",
			input:      "```json\n{\"intent\":\"task\",\"tags\":[\"deploy\"],\"cot_prompt\":\"1. Plan\\n2. Execute\"}\n```",
			wantIntent: "task",
			wantTags:   []string{"deploy"},
			wantCot:    "1. Plan\n2. Execute",
		},
		{
			name:       "empty cot_prompt for chat",
			input:      `{"intent":"chat","tags":[],"cot_prompt":""}`,
			wantIntent: "chat",
			wantTags:   []string{},
			wantCot:    "",
		},
		{
			name:       "invalid JSON",
			input:      "this is not json",
			wantIntent: "",
			wantTags:   nil,
			wantCot:    "",
		},
		{
			name:       "tags trimmed and lowered",
			input:      `{"intent":"code","tags":["  GoLang  ","  API  "],"cot_prompt":"think"}`,
			wantIntent: "code",
			wantTags:   []string{"golang", "api"},
			wantCot:    "think",
		},
		{
			name:       "tags limited to 5",
			input:      `{"intent":"search","tags":["a","b","c","d","e","f","g"],"cot_prompt":"search"}`,
			wantIntent: "search",
			wantTags:   []string{"a", "b", "c", "d", "e"},
			wantCot:    "search",
		},
		{
			name:       "missing cot_prompt field",
			input:      `{"intent":"question","tags":["golang"]}`,
			wantIntent: "question",
			wantTags:   []string{"golang"},
			wantCot:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.parseResponse(tt.input)
			if result.Intent != tt.wantIntent {
				t.Errorf("intent = %q, want %q", result.Intent, tt.wantIntent)
			}
			if result.CotPrompt != tt.wantCot {
				t.Errorf("cot_prompt = %q, want %q", result.CotPrompt, tt.wantCot)
			}

			if tt.wantTags == nil {
				if result.Tags != nil {
					t.Errorf("tags = %v, want nil", result.Tags)
				}
				return
			}

			if len(result.Tags) != len(tt.wantTags) {
				t.Errorf("tags len = %d, want %d (tags=%v)", len(result.Tags), len(tt.wantTags), result.Tags)
				return
			}
			for i, tag := range result.Tags {
				if tag != tt.wantTags[i] {
					t.Errorf("tag[%d] = %q, want %q", i, tag, tt.wantTags[i])
				}
			}
		})
	}
}

func TestPreLLM_Analyse_NoProvider(t *testing.T) {
	p := &Analyser{} // no provider, no model
	result := p.Analyse(context.Background(), "hello", nil, nil)
	if result.Intent != "" || len(result.Tags) != 0 {
		t.Errorf("expected empty result with no provider, got %+v", result)
	}
}

func TestPreLLM_Analyse_NoTags(t *testing.T) {
	dir := t.TempDir()
	ms := NewMemoryStore(dir)
	defer ms.Close()

	cotReg := NewCotRegistry(dir)
	mp := &mockLLMProvider{
		response: `{"intent":"chat","tags":[],"cot_prompt":""}`,
	}
	p := NewAnalyser(mp, "test-model", cotReg)

	result := p.Analyse(context.Background(), "hello there", ms, nil)
	if result.Intent != "chat" {
		t.Errorf("expected intent 'chat', got %q", result.Intent)
	}
	if result.CotPrompt != "" {
		t.Errorf("expected empty cot_prompt for chat, got %q", result.CotPrompt)
	}
}

func TestPreLLM_Analyse_WithMemory(t *testing.T) {
	dir := t.TempDir()
	ms := NewMemoryStore(dir)
	defer ms.Close()

	// Seed memory.
	ms.AddEntry("Go is great for concurrency", []string{"golang", "concurrency"})
	ms.AddEntry("Kubernetes cluster setup notes", []string{"k8s", "devops"})
	ms.AddEntry("Go testing best practices", []string{"golang", "testing"})

	cotReg := NewCotRegistry(dir)
	mp := &mockLLMProvider{
		response: `{"intent":"question","tags":["golang"],"cot_prompt":"1. Check Go docs\n2. Write example code\n3. Verify with tests"}`,
	}
	p := NewAnalyser(mp, "test-model", cotReg)

	result := p.Analyse(context.Background(), "How do I test Go code?", ms, nil)

	if result.Intent != "question" {
		t.Errorf("intent = %q, want %q", result.Intent, "question")
	}
	if result.CotPrompt == "" {
		t.Error("expected non-empty CotPrompt")
	}
	if !strings.Contains(result.CotPrompt, "Go docs") {
		t.Error("CotPrompt should contain the LLM-generated strategy")
	}
	if len(result.Tags) != 1 || result.Tags[0] != "golang" {
		t.Errorf("tags = %v, want [golang]", result.Tags)
	}
	if result.MemoryContext == "" {
		t.Error("expected non-empty MemoryContext with matching tags")
	}
	if !contains(result.MemoryContext, "Go is great for concurrency") {
		t.Error("MemoryContext missing 'Go is great for concurrency'")
	}
	if !contains(result.MemoryContext, "Go testing best practices") {
		t.Error("MemoryContext missing 'Go testing best practices'")
	}
	if contains(result.MemoryContext, "Kubernetes") {
		t.Error("MemoryContext should not contain 'Kubernetes' entry")
	}

	// Verify usage was recorded with tags.
	records, _ := ms.GetRecentCotUsage(1)
	if len(records) == 0 {
		t.Fatal("expected usage record to be recorded")
	}
	if len(records[0].Tags) != 1 || records[0].Tags[0] != "golang" {
		t.Errorf("recorded tags = %v, want [golang]", records[0].Tags)
	}
}

func TestSearchByAnyTag(t *testing.T) {
	dir := t.TempDir()
	ms := NewMemoryStore(dir)
	defer ms.Close()

	ms.AddEntry("Go concurrency", []string{"golang", "concurrency"})
	ms.AddEntry("K8s setup", []string{"k8s", "devops"})
	ms.AddEntry("Go testing", []string{"golang", "testing"})
	ms.AddEntry("Python ML", []string{"python", "ml"})

	entries, err := ms.SearchByAnyTag([]string{"golang", "k8s"})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Errorf("got %d entries, want 3", len(entries))
	}

	entries, err = ms.SearchByAnyTag([]string{"python"})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Errorf("got %d entries, want 1", len(entries))
	}

	entries, err = ms.SearchByAnyTag([]string{"nonexistent"})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("got %d entries, want 0", len(entries))
	}
}

func TestFormatMemoryEntries(t *testing.T) {
	entries := []MemoryEntry{
		{ID: 1, Content: "Test content 1", Tags: []string{"tag1", "tag2"}},
		{ID: 2, Content: "Test content 2", Tags: []string{"tag3"}},
		{ID: 3, Content: "No tags entry", Tags: nil},
	}

	result := formatMemoryEntries(entries)
	if result == "" {
		t.Fatal("expected non-empty result")
	}
	if !contains(result, "Relevant Memories") {
		t.Error("missing header")
	}
	if !contains(result, "#1") {
		t.Error("missing entry #1")
	}
	if !contains(result, "[tag1, tag2]") {
		t.Error("missing tags for entry #1")
	}
}

func TestFormatMemoryEntries_Empty(t *testing.T) {
	result := formatMemoryEntries(nil)
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

// --- Helpers ---

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

func TestPreLLM_MemoryDBPath(t *testing.T) {
	dir := t.TempDir()
	ms := NewMemoryStore(dir)
	defer ms.Close()

	dbPath := filepath.Join(dir, "memory.db")
	if _, err := os.Stat(dbPath); err != nil {
		t.Errorf("memory.db not created: %v", err)
	}
}

// mockLLMProvider returns a configurable response for pre-LLM testing.
type mockLLMProvider struct {
	response string
}

func (m *mockLLMProvider) Chat(
	_ context.Context,
	_ []providers.Message,
	_ []providers.ToolDefinition,
	_ string,
	_ map[string]any,
) (*providers.LLMResponse, error) {
	return &providers.LLMResponse{
		Content:   m.response,
		ToolCalls: []providers.ToolCall{},
	}, nil
}

func (m *mockLLMProvider) GetDefaultModel() string {
	return "mock-pre-llm"
}

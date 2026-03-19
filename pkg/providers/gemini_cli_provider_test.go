package providers

import (
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
)

// --- Compile-time interface check ---

var _ LLMProvider = (*GeminiCliProvider)(nil)

// --- Constructor tests ---

func TestNewGeminiCliProvider(t *testing.T) {
	p := NewGeminiCliProvider("/test/workspace")
	if p == nil {
		t.Fatal("NewGeminiCliProvider returned nil")
	}
	if p.workspace != "/test/workspace" {
		t.Errorf("workspace = %q, want %q", p.workspace, "/test/workspace")
	}
	if p.command != "gemini" {
		t.Errorf("command = %q, want %q", p.command, "gemini")
	}
}

// --- GetDefaultModel tests ---

func TestGeminiCliProvider_GetDefaultModel(t *testing.T) {
	p := NewGeminiCliProvider("/workspace")
	if got := p.GetDefaultModel(); got != "gemini-cli" {
		t.Errorf("GetDefaultModel() = %q, want %q", got, "gemini-cli")
	}
}

// --- buildPrompt tests ---

func TestGeminiCliProvider_BuildPrompt_SingleUser(t *testing.T) {
	p := NewGeminiCliProvider("/workspace")
	messages := []Message{
		{Role: "user", Content: "Hello"},
	}
	got := p.buildPrompt(messages, nil)
	// Single user message with no system or tools should be simplified (no prefix)
	want := "Hello"
	if got != want {
		t.Errorf("buildPrompt() = %q, want %q", got, want)
	}
}

func TestGeminiCliProvider_BuildPrompt_WithSystem(t *testing.T) {
	p := NewGeminiCliProvider("/workspace")
	messages := []Message{
		{Role: "system", Content: "You are a helpful assistant."},
		{Role: "user", Content: "What is Go?"},
	}
	got := p.buildPrompt(messages, nil)
	if !strings.Contains(got, "## System Instructions") {
		t.Errorf("buildPrompt() missing ## System Instructions header, got %q", got)
	}
	if !strings.Contains(got, "You are a helpful assistant.") {
		t.Errorf("buildPrompt() missing system message content, got %q", got)
	}
	if !strings.Contains(got, "## Task") {
		t.Errorf("buildPrompt() missing ## Task header, got %q", got)
	}
	if !strings.Contains(got, "What is Go?") {
		t.Errorf("buildPrompt() missing user message content, got %q", got)
	}
}

func TestGeminiCliProvider_BuildPrompt_WithTools(t *testing.T) {
	p := NewGeminiCliProvider("/workspace")
	messages := []Message{
		{Role: "user", Content: "What is the weather?"},
	}
	tools := []ToolDefinition{
		{
			Type: "function",
			Function: ToolFunctionDefinition{
				Name:        "get_weather",
				Description: "Get weather for a location",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"location": map[string]any{"type": "string"},
					},
				},
			},
		},
	}
	got := p.buildPrompt(messages, tools)
	if !strings.Contains(got, "get_weather") {
		t.Errorf("buildPrompt() missing tool definition, got %q", got)
	}
	if !strings.Contains(got, "Available Tools") {
		t.Errorf("buildPrompt() missing Available Tools header, got %q", got)
	}
	if !strings.Contains(got, "What is the weather?") {
		t.Errorf("buildPrompt() missing user message, got %q", got)
	}
}

// --- parseGeminiCliResponse tests ---

func TestGeminiCliProvider_ParseResponse_Basic(t *testing.T) {
	p := NewGeminiCliProvider("/workspace")
	output := `{
		"session_id": "abc123",
		"response": "Hello! How can I assist you?",
		"stats": {
			"models": {
				"gemini-2.5-flash-lite": {
					"tokens": {
						"input": 2634,
						"candidates": 29,
						"total": 2735
					}
				},
				"gemini-3-flash-preview": {
					"tokens": {
						"input": 16921,
						"candidates": 14,
						"total": 16935
					}
				}
			}
		}
	}`

	resp, err := p.parseGeminiCliResponse(output)
	if err != nil {
		t.Fatalf("parseGeminiCliResponse() error = %v", err)
	}
	if resp.Content != "Hello! How can I assist you?" {
		t.Errorf("Content = %q, want %q", resp.Content, "Hello! How can I assist you?")
	}
	if resp.FinishReason != "stop" {
		t.Errorf("FinishReason = %q, want %q", resp.FinishReason, "stop")
	}
	if len(resp.ToolCalls) != 0 {
		t.Errorf("ToolCalls len = %d, want 0", len(resp.ToolCalls))
	}
	if resp.Usage == nil {
		t.Fatal("Usage should not be nil")
	}
	// Summed input: 2634 + 16921 = 19555
	if resp.Usage.PromptTokens != 19555 {
		t.Errorf("PromptTokens = %d, want 19555", resp.Usage.PromptTokens)
	}
	// Summed candidates: 29 + 14 = 43
	if resp.Usage.CompletionTokens != 43 {
		t.Errorf("CompletionTokens = %d, want 43", resp.Usage.CompletionTokens)
	}
	// Summed total: 2735 + 16935 = 19670
	if resp.Usage.TotalTokens != 19670 {
		t.Errorf("TotalTokens = %d, want 19670", resp.Usage.TotalTokens)
	}
}

func TestGeminiCliProvider_ParseResponse_WithToolCalls(t *testing.T) {
	p := NewGeminiCliProvider("/workspace")
	output := `{"session_id":"s1","response":"Checking weather.\n{\"tool_calls\":[{\"id\":\"call_1\",\"type\":\"function\",\"function\":{\"name\":\"get_weather\",\"arguments\":\"{\\\"location\\\":\\\"NYC\\\"}\"}}]}","stats":{}}`

	resp, err := p.parseGeminiCliResponse(output)
	if err != nil {
		t.Fatalf("parseGeminiCliResponse() error = %v", err)
	}
	if resp.FinishReason != "tool_calls" {
		t.Errorf("FinishReason = %q, want %q", resp.FinishReason, "tool_calls")
	}
	if len(resp.ToolCalls) != 1 {
		t.Fatalf("ToolCalls len = %d, want 1", len(resp.ToolCalls))
	}
	if resp.ToolCalls[0].Name != "get_weather" {
		t.Errorf("ToolCalls[0].Name = %q, want %q", resp.ToolCalls[0].Name, "get_weather")
	}
	if resp.ToolCalls[0].Arguments["location"] != "NYC" {
		t.Errorf("ToolCalls[0].Arguments[location] = %v, want NYC", resp.ToolCalls[0].Arguments["location"])
	}
	if strings.Contains(resp.Content, "tool_calls") {
		t.Errorf("Content should not contain tool_calls JSON, got %q", resp.Content)
	}
}

func TestGeminiCliProvider_ParseResponse_InvalidJSON(t *testing.T) {
	p := NewGeminiCliProvider("/workspace")
	_, err := p.parseGeminiCliResponse("not valid json")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "failed to parse gemini cli response") {
		t.Errorf("error = %q, want to contain 'failed to parse gemini cli response'", err.Error())
	}
}

func TestGeminiCliProvider_ParseResponse_NoStats(t *testing.T) {
	p := NewGeminiCliProvider("/workspace")
	output := `{"session_id":"s","response":"hello"}`

	resp, err := p.parseGeminiCliResponse(output)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if resp.Content != "hello" {
		t.Errorf("Content = %q, want %q", resp.Content, "hello")
	}
	if resp.Usage != nil {
		t.Errorf("Usage should be nil when no stats, got %+v", resp.Usage)
	}
}

// --- Factory tests ---

func TestCreateProvider_GeminiCli(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.ModelList = []config.ModelConfig{
		{ModelName: "gemini-cli", Model: "gemini-cli/gemini-cli", Workspace: "/test/ws"},
	}
	cfg.Agents.Defaults.Model = "gemini-cli"

	provider, modelID, err := CreateProvider(cfg)
	if err != nil {
		t.Fatalf("CreateProvider(gemini-cli) error = %v", err)
	}

	geminiProvider, ok := provider.(*GeminiCliProvider)
	if !ok {
		t.Fatalf("CreateProvider(gemini-cli) returned %T, want *GeminiCliProvider", provider)
	}
	if geminiProvider.workspace != "/test/ws" {
		t.Errorf("workspace = %q, want %q", geminiProvider.workspace, "/test/ws")
	}
	// modelID should be the part after the slash
	if modelID != "gemini-cli" {
		t.Errorf("modelID = %q, want %q", modelID, "gemini-cli")
	}
}

func TestCreateProvider_GeminiCliWithModel(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.ModelList = []config.ModelConfig{
		{ModelName: "gemini-flash", Model: "gemini-cli/gemini-2.5-flash", Workspace: "/ws"},
	}
	cfg.Agents.Defaults.Model = "gemini-flash"

	provider, modelID, err := CreateProvider(cfg)
	if err != nil {
		t.Fatalf("CreateProvider(gemini-cli/gemini-2.5-flash) error = %v", err)
	}
	if _, ok := provider.(*GeminiCliProvider); !ok {
		t.Fatalf("CreateProvider returned %T, want *GeminiCliProvider", provider)
	}
	// modelID should carry through the actual model name
	if modelID != "gemini-2.5-flash" {
		t.Errorf("modelID = %q, want %q", modelID, "gemini-2.5-flash")
	}
}

func TestCreateProvider_GeminiCliDefaultWorkspace(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Workspace = "" // clear default so the "." fallback is exercised
	cfg.ModelList = []config.ModelConfig{
		{ModelName: "gemini-cli", Model: "gemini-cli/gemini-cli"},
	}
	cfg.Agents.Defaults.Model = "gemini-cli"

	provider, _, err := CreateProvider(cfg)
	if err != nil {
		t.Fatalf("CreateProvider error = %v", err)
	}
	geminiProvider, ok := provider.(*GeminiCliProvider)
	if !ok {
		t.Fatalf("returned %T, want *GeminiCliProvider", provider)
	}
	if geminiProvider.workspace != "." {
		t.Errorf("workspace = %q, want %q (default)", geminiProvider.workspace, ".")
	}
}

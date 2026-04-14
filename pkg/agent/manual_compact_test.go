package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/providers"
)

type promptCapturingProvider struct {
	response string
	prompts  []string
}

func (p *promptCapturingProvider) Chat(
	_ context.Context,
	messages []providers.Message,
	_ []providers.ToolDefinition,
	_ string,
	_ map[string]any,
) (*providers.LLMResponse, error) {
	if len(messages) > 0 {
		p.prompts = append(p.prompts, messages[0].Content)
	}
	return &providers.LLMResponse{
		Content:   p.response,
		ToolCalls: []providers.ToolCall{},
	}, nil
}

func (p *promptCapturingProvider) GetDefaultModel() string {
	return "prompt-capturing-model"
}

func TestLegacyCompact_Manual_UsesInstructionsAndCompactsSynchronously(t *testing.T) {
	cfg := testConfig(t)
	provider := &promptCapturingProvider{response: "manual compact summary"}
	al := NewAgentLoop(cfg, bus.NewMessageBus(), provider)

	defaultAgent := al.registry.GetDefaultAgent()
	if defaultAgent == nil {
		t.Fatal("expected default agent")
	}

	history := []providers.Message{
		{Role: "user", Content: "question one"},
		{Role: "assistant", Content: "answer one"},
		{Role: "user", Content: "question two"},
		{Role: "assistant", Content: "answer two"},
		{Role: "user", Content: "question three"},
		{Role: "assistant", Content: "answer three"},
	}
	defaultAgent.Sessions.SetHistory("session-manual", history)

	err := al.contextManager.Compact(context.Background(), &CompactRequest{
		SessionKey:   "session-manual",
		Reason:       ContextCompressReasonSummarize,
		Instructions: "focus on decisions",
		Manual:       true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	newHistory := defaultAgent.Sessions.GetHistory("session-manual")
	if len(newHistory) >= len(history) {
		t.Fatalf("expected compacted history, got %d messages (was %d)", len(newHistory), len(history))
	}

	summary := defaultAgent.Sessions.GetSummary("session-manual")
	if !strings.Contains(summary, "manual compact summary") {
		t.Fatalf("summary=%q, want manual summary content", summary)
	}
	if len(provider.prompts) == 0 || !strings.Contains(provider.prompts[0], "focus on decisions") {
		t.Fatalf("expected compaction prompt to include manual instructions, got %v", provider.prompts)
	}
	if got := al.getCompactionCount("session-manual"); got != 1 {
		t.Fatalf("compaction count=%d, want 1", got)
	}
}

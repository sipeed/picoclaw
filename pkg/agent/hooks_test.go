package agent

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/tools"
)

func newHookTestLoop(
	t *testing.T,
	provider providers.LLMProvider,
) (*AgentLoop, *AgentInstance, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "agent-hooks-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         tmpDir,
				Model:             "test-model",
				MaxTokens:         4096,
				MaxToolIterations: 10,
			},
		},
	}

	al := NewAgentLoop(cfg, bus.NewMessageBus(), provider)
	agent := al.registry.GetDefaultAgent()
	if agent == nil {
		t.Fatal("expected default agent")
	}

	return al, agent, func() {
		al.Close()
		_ = os.RemoveAll(tmpDir)
	}
}

func TestHookManager_SortsInProcessBeforeProcess(t *testing.T) {
	hm := NewHookManager(nil)
	defer hm.Close()

	if err := hm.Mount(HookRegistration{
		Name:     "process",
		Priority: -10,
		Source:   HookSourceProcess,
		Hook:     struct{}{},
	}); err != nil {
		t.Fatalf("mount process hook: %v", err)
	}
	if err := hm.Mount(HookRegistration{
		Name:     "in-process",
		Priority: 100,
		Source:   HookSourceInProcess,
		Hook:     struct{}{},
	}); err != nil {
		t.Fatalf("mount in-process hook: %v", err)
	}

	ordered := hm.snapshotHooks()
	if len(ordered) != 2 {
		t.Fatalf("expected 2 hooks, got %d", len(ordered))
	}
	if ordered[0].Name != "in-process" {
		t.Fatalf("expected in-process hook first, got %q", ordered[0].Name)
	}
	if ordered[1].Name != "process" {
		t.Fatalf("expected process hook second, got %q", ordered[1].Name)
	}
}

type llmHookTestProvider struct {
	mu        sync.Mutex
	lastModel string
}

func (p *llmHookTestProvider) Chat(
	ctx context.Context,
	messages []providers.Message,
	tools []providers.ToolDefinition,
	model string,
	opts map[string]any,
) (*providers.LLMResponse, error) {
	p.mu.Lock()
	p.lastModel = model
	p.mu.Unlock()

	return &providers.LLMResponse{
		Content: "provider content",
	}, nil
}

func (p *llmHookTestProvider) GetDefaultModel() string {
	return "llm-hook-provider"
}

type llmObserverHook struct {
	eventCh chan Event
}

func (h *llmObserverHook) OnEvent(ctx context.Context, evt Event) error {
	if evt.Kind == EventKindTurnEnd {
		select {
		case h.eventCh <- evt:
		default:
		}
	}
	return nil
}

func (h *llmObserverHook) BeforeLLM(
	ctx context.Context,
	req *LLMHookRequest,
) (*LLMHookRequest, HookDecision, error) {
	next := req.Clone()
	next.Model = "hook-model"
	return next, HookDecision{Action: HookActionModify}, nil
}

func (h *llmObserverHook) AfterLLM(
	ctx context.Context,
	resp *LLMHookResponse,
) (*LLMHookResponse, HookDecision, error) {
	next := resp.Clone()
	next.Response.Content = "hooked content"
	return next, HookDecision{Action: HookActionModify}, nil
}

func TestAgentLoop_Hooks_ObserverAndLLMInterceptor(t *testing.T) {
	provider := &llmHookTestProvider{}
	al, agent, cleanup := newHookTestLoop(t, provider)
	defer cleanup()

	hook := &llmObserverHook{eventCh: make(chan Event, 1)}
	if err := al.MountHook(NamedHook("llm-observer", hook)); err != nil {
		t.Fatalf("MountHook failed: %v", err)
	}

	resp, err := al.runAgentLoop(context.Background(), agent, processOptions{
		SessionKey:      "session-1",
		Channel:         "cli",
		ChatID:          "direct",
		UserMessage:     "hello",
		DefaultResponse: defaultResponse,
		EnableSummary:   false,
		SendResponse:    false,
	})
	if err != nil {
		t.Fatalf("runAgentLoop failed: %v", err)
	}
	if resp != "hooked content" {
		t.Fatalf("expected hooked content, got %q", resp)
	}

	provider.mu.Lock()
	lastModel := provider.lastModel
	provider.mu.Unlock()
	if lastModel != "hook-model" {
		t.Fatalf("expected model hook-model, got %q", lastModel)
	}

	select {
	case evt := <-hook.eventCh:
		if evt.Kind != EventKindTurnEnd {
			t.Fatalf("expected turn end event, got %v", evt.Kind)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for hook observer event")
	}
}

type toolHookProvider struct {
	mu    sync.Mutex
	calls int
}

func (p *toolHookProvider) Chat(
	ctx context.Context,
	messages []providers.Message,
	tools []providers.ToolDefinition,
	model string,
	opts map[string]any,
) (*providers.LLMResponse, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.calls++
	if p.calls == 1 {
		return &providers.LLMResponse{
			ToolCalls: []providers.ToolCall{
				{
					ID:        "call-1",
					Name:      "echo_text",
					Arguments: map[string]any{"text": "original"},
				},
			},
		}, nil
	}

	last := messages[len(messages)-1]
	return &providers.LLMResponse{
		Content: last.Content,
	}, nil
}

func (p *toolHookProvider) GetDefaultModel() string {
	return "tool-hook-provider"
}

type echoTextTool struct{}

func (t *echoTextTool) Name() string {
	return "echo_text"
}

func (t *echoTextTool) Description() string {
	return "echo a text argument"
}

func (t *echoTextTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"text": map[string]any{
				"type": "string",
			},
		},
	}
}

func (t *echoTextTool) Execute(ctx context.Context, args map[string]any) *tools.ToolResult {
	text, _ := args["text"].(string)
	return tools.SilentResult(text)
}

type toolRewriteHook struct{}

func (h *toolRewriteHook) BeforeTool(
	ctx context.Context,
	call *ToolCallHookRequest,
) (*ToolCallHookRequest, HookDecision, error) {
	next := call.Clone()
	next.Arguments["text"] = "modified"
	return next, HookDecision{Action: HookActionModify}, nil
}

func (h *toolRewriteHook) AfterTool(
	ctx context.Context,
	result *ToolResultHookResponse,
) (*ToolResultHookResponse, HookDecision, error) {
	next := result.Clone()
	next.Result.ForLLM = "after:" + next.Result.ForLLM
	return next, HookDecision{Action: HookActionModify}, nil
}

func TestAgentLoop_Hooks_ToolInterceptorCanRewrite(t *testing.T) {
	provider := &toolHookProvider{}
	al, agent, cleanup := newHookTestLoop(t, provider)
	defer cleanup()

	al.RegisterTool(&echoTextTool{})
	if err := al.MountHook(NamedHook("tool-rewrite", &toolRewriteHook{})); err != nil {
		t.Fatalf("MountHook failed: %v", err)
	}

	resp, err := al.runAgentLoop(context.Background(), agent, processOptions{
		SessionKey:      "session-1",
		Channel:         "cli",
		ChatID:          "direct",
		UserMessage:     "run tool",
		DefaultResponse: defaultResponse,
		EnableSummary:   false,
		SendResponse:    false,
	})
	if err != nil {
		t.Fatalf("runAgentLoop failed: %v", err)
	}
	if resp != "after:modified" {
		t.Fatalf("expected rewritten tool result, got %q", resp)
	}
}

type denyApprovalHook struct{}

func (h *denyApprovalHook) ApproveTool(ctx context.Context, req *ToolApprovalRequest) (ApprovalDecision, error) {
	return ApprovalDecision{
		Approved: false,
		Reason:   "blocked",
	}, nil
}

func TestAgentLoop_Hooks_ToolApproverCanDeny(t *testing.T) {
	provider := &toolHookProvider{}
	al, agent, cleanup := newHookTestLoop(t, provider)
	defer cleanup()

	al.RegisterTool(&echoTextTool{})
	if err := al.MountHook(NamedHook("deny-approval", &denyApprovalHook{})); err != nil {
		t.Fatalf("MountHook failed: %v", err)
	}

	sub := al.SubscribeEvents(16)
	defer al.UnsubscribeEvents(sub.ID)

	resp, err := al.runAgentLoop(context.Background(), agent, processOptions{
		SessionKey:      "session-1",
		Channel:         "cli",
		ChatID:          "direct",
		UserMessage:     "run tool",
		DefaultResponse: defaultResponse,
		EnableSummary:   false,
		SendResponse:    false,
	})
	if err != nil {
		t.Fatalf("runAgentLoop failed: %v", err)
	}
	expected := "Tool execution denied by approval hook: blocked"
	if resp != expected {
		t.Fatalf("expected %q, got %q", expected, resp)
	}

	events := collectEventStream(sub.C)
	skippedEvt, ok := findEvent(events, EventKindToolExecSkipped)
	if !ok {
		t.Fatal("expected tool skipped event")
	}
	payload, ok := skippedEvt.Payload.(ToolExecSkippedPayload)
	if !ok {
		t.Fatalf("expected ToolExecSkippedPayload, got %T", skippedEvt.Payload)
	}
	if payload.Reason != expected {
		t.Fatalf("expected skipped reason %q, got %q", expected, payload.Reason)
	}
}

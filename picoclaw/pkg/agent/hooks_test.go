package agent

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/routing"
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
				ModelName:         "test-model",
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

// respondHook is a test hook for testing HookActionRespond functionality
type respondHook struct {
	respondTools map[string]bool // tool names to respond to
}

func (h *respondHook) BeforeTool(
	ctx context.Context,
	call *ToolCallHookRequest,
) (*ToolCallHookRequest, HookDecision, error) {
	if h.respondTools[call.Tool] {
		next := call.Clone()
		next.HookResult = &tools.ToolResult{
			ForLLM:  "hook-responded: " + call.Tool,
			ForUser: "",
			Silent:  false,
			IsError: false,
		}
		return next, HookDecision{Action: HookActionRespond}, nil
	}
	return call, HookDecision{Action: HookActionContinue}, nil
}

func (h *respondHook) AfterTool(
	ctx context.Context,
	result *ToolResultHookResponse,
) (*ToolResultHookResponse, HookDecision, error) {
	// Should not be called since respond skips tool execution
	return result, HookDecision{Action: HookActionContinue}, nil
}

func TestAgentLoop_Hooks_ToolRespondAction(t *testing.T) {
	provider := &toolHookProvider{}
	al, agent, cleanup := newHookTestLoop(t, provider)
	defer cleanup()

	al.RegisterTool(&echoTextTool{})
	if err := al.MountHook(NamedHook("respond-hook", &respondHook{
		respondTools: map[string]bool{"echo_text": true},
	})); err != nil {
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

	// Verify response comes from hook, not tool
	expected := "hook-responded: echo_text"
	if resp != expected {
		t.Fatalf("expected %q, got %q", expected, resp)
	}

	// Verify event stream has ToolExecEnd, not actual tool execution
	events := collectEventStream(sub.C)
	endEvt, ok := findEvent(events, EventKindToolExecEnd)
	if !ok {
		t.Fatal("expected tool exec end event")
	}
	payload, ok := endEvt.Payload.(ToolExecEndPayload)
	if !ok {
		t.Fatalf("expected ToolExecEndPayload, got %T", endEvt.Payload)
	}
	if payload.Tool != "echo_text" {
		t.Fatalf("expected tool echo_text, got %q", payload.Tool)
	}
	if payload.ForLLMLen != len(expected) {
		t.Fatalf("expected ForLLMLen %d, got %d", len(expected), payload.ForLLMLen)
	}
}

// denyToolHook tests HookActionDenyTool functionality
type denyToolHook struct {
	denyTools map[string]bool
}

func (h *denyToolHook) BeforeTool(
	ctx context.Context,
	call *ToolCallHookRequest,
) (*ToolCallHookRequest, HookDecision, error) {
	if h.denyTools[call.Tool] {
		return call, HookDecision{Action: HookActionDenyTool, Reason: "tool denied by hook"}, nil
	}
	return call, HookDecision{Action: HookActionContinue}, nil
}

func (h *denyToolHook) AfterTool(
	ctx context.Context,
	result *ToolResultHookResponse,
) (*ToolResultHookResponse, HookDecision, error) {
	return result, HookDecision{Action: HookActionContinue}, nil
}

func TestAgentLoop_Hooks_ToolDenyAction(t *testing.T) {
	provider := &toolHookProvider{}
	al, agent, cleanup := newHookTestLoop(t, provider)
	defer cleanup()

	al.RegisterTool(&echoTextTool{})
	if err := al.MountHook(NamedHook("deny-hook", &denyToolHook{
		denyTools: map[string]bool{"echo_text": true},
	})); err != nil {
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

	expected := "Tool execution denied by hook: tool denied by hook"
	if resp != expected {
		t.Fatalf("expected %q, got %q", expected, resp)
	}
}

func TestHookManager_BeforeTool_RespondAction(t *testing.T) {
	hm := NewHookManager(nil)
	defer hm.Close()

	hook := &respondHook{
		respondTools: map[string]bool{"test_tool": true},
	}
	if err := hm.Mount(NamedHook("respond-test", hook)); err != nil {
		t.Fatalf("mount hook: %v", err)
	}

	req := &ToolCallHookRequest{
		Tool:      "test_tool",
		Arguments: map[string]any{"arg": "value"},
	}
	result, decision := hm.BeforeTool(context.Background(), req)

	if decision.Action != HookActionRespond {
		t.Fatalf("expected action %q, got %q", HookActionRespond, decision.Action)
	}

	if result.HookResult == nil {
		t.Fatal("expected HookResult to be set")
	}
	if result.HookResult.ForLLM != "hook-responded: test_tool" {
		t.Fatalf("unexpected HookResult.ForLLM: %q", result.HookResult.ForLLM)
	}
}

type respondWithMediaHook struct {
	respondTools    map[string]bool
	media           []string
	responseHandled bool
	forLLM          string
}

func (h *respondWithMediaHook) BeforeTool(
	ctx context.Context,
	call *ToolCallHookRequest,
) (*ToolCallHookRequest, HookDecision, error) {
	if h.respondTools[call.Tool] {
		next := call.Clone()
		next.HookResult = &tools.ToolResult{
			ForLLM:          h.forLLM,
			ForUser:         "media result",
			Media:           h.media,
			ResponseHandled: h.responseHandled,
			Silent:          false,
			IsError:         false,
		}
		return next, HookDecision{Action: HookActionRespond}, nil
	}
	return call, HookDecision{Action: HookActionContinue}, nil
}

func (h *respondWithMediaHook) AfterTool(
	ctx context.Context,
	result *ToolResultHookResponse,
) (*ToolResultHookResponse, HookDecision, error) {
	return result, HookDecision{Action: HookActionContinue}, nil
}

type errorMediaChannel struct {
	fakeChannel
	sendErr error
}

func (f *errorMediaChannel) SendMedia(ctx context.Context, msg bus.OutboundMediaMessage) ([]string, error) {
	return nil, f.sendErr
}

func TestAgentLoop_HookRespond_MediaError(t *testing.T) {
	provider := &multiToolProvider{
		toolCalls: []providers.ToolCall{
			{ID: "call-1", Name: "media_tool", Arguments: map[string]any{}},
		},
		finalContent: "done",
	}
	al, agent, cleanup := newHookTestLoop(t, provider)
	defer cleanup()

	hook := &respondWithMediaHook{
		respondTools:    map[string]bool{"media_tool": true},
		media:           []string{"media://test/image.png"},
		responseHandled: true,
		forLLM:          "media sent successfully",
	}
	if err := al.MountHook(NamedHook("media-hook", hook)); err != nil {
		t.Fatalf("MountHook failed: %v", err)
	}

	al.channelManager = newStartedTestChannelManager(t, al.bus, al.mediaStore, "discord", &errorMediaChannel{
		sendErr: errors.New("channel unavailable"),
	})

	sub := al.SubscribeEvents(16)
	defer al.UnsubscribeEvents(sub.ID)

	_, err := al.runAgentLoop(context.Background(), agent, processOptions{
		SessionKey:      "session-media-err",
		Channel:         "discord",
		ChatID:          "chat1",
		UserMessage:     "send media",
		DefaultResponse: defaultResponse,
		EnableSummary:   false,
		SendResponse:    false,
	})
	if err != nil {
		t.Fatalf("runAgentLoop failed: %v", err)
	}

	events := collectEventStream(sub.C)
	endEvt, ok := findEvent(events, EventKindToolExecEnd)
	if !ok {
		t.Fatal("expected ToolExecEnd event")
	}
	payload, ok := endEvt.Payload.(ToolExecEndPayload)
	if !ok {
		t.Fatalf("expected ToolExecEndPayload, got %T", endEvt.Payload)
	}

	if !payload.IsError {
		t.Fatal("expected IsError=true when SendMedia fails")
	}

	if payload.ForLLMLen < 30 {
		t.Fatalf("expected ForLLM to contain error message, got ForLLMLen=%d", payload.ForLLMLen)
	}
}

func TestAgentLoop_HookRespond_BusFallback(t *testing.T) {
	provider := &multiToolProvider{
		toolCalls: []providers.ToolCall{
			{ID: "call-1", Name: "media_tool", Arguments: map[string]any{}},
		},
		finalContent: "done",
	}
	al, agent, cleanup := newHookTestLoop(t, provider)
	defer cleanup()

	hook := &respondWithMediaHook{
		respondTools:    map[string]bool{"media_tool": true},
		media:           []string{"media://test/image.png"},
		responseHandled: true,
		forLLM:          "media queued",
	}
	if err := al.MountHook(NamedHook("media-hook", hook)); err != nil {
		t.Fatalf("MountHook failed: %v", err)
	}

	sub := al.SubscribeEvents(16)
	defer al.UnsubscribeEvents(sub.ID)

	resp, err := al.runAgentLoop(context.Background(), agent, processOptions{
		SessionKey:      "session-bus-fallback",
		Channel:         "cli",
		ChatID:          "chat1",
		UserMessage:     "send media",
		DefaultResponse: defaultResponse,
		EnableSummary:   false,
		SendResponse:    false,
	})
	if err != nil {
		t.Fatalf("runAgentLoop failed: %v", err)
	}

	events := collectEventStream(sub.C)
	endEvt, ok := findEvent(events, EventKindToolExecEnd)
	if !ok {
		t.Fatal("expected ToolExecEnd event")
	}
	payload, ok := endEvt.Payload.(ToolExecEndPayload)
	if !ok {
		t.Fatalf("expected ToolExecEndPayload, got %T", endEvt.Payload)
	}

	if payload.IsError {
		t.Fatal("expected IsError=false for bus fallback (media queued, not delivered)")
	}

	if resp != "done" {
		t.Fatalf("expected response 'done', got %q", resp)
	}
}

type multiToolProvider struct {
	mu           sync.Mutex
	callCount    int
	toolCalls    []providers.ToolCall
	finalContent string
}

func (p *multiToolProvider) Chat(
	ctx context.Context,
	messages []providers.Message,
	tools []providers.ToolDefinition,
	model string,
	opts map[string]any,
) (*providers.LLMResponse, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.callCount++
	if p.callCount == 1 && len(p.toolCalls) > 0 {
		return &providers.LLMResponse{
			ToolCalls: p.toolCalls,
		}, nil
	}

	return &providers.LLMResponse{
		Content: p.finalContent,
	}, nil
}

func (p *multiToolProvider) GetDefaultModel() string {
	return "multi-tool-provider"
}

func TestAgentLoop_HookRespond_InterruptSkipsRemaining(t *testing.T) {
	provider := &multiToolProvider{
		toolCalls: []providers.ToolCall{
			{ID: "call-1", Name: "tool_one", Arguments: map[string]any{}},
			{ID: "call-2", Name: "tool_two", Arguments: map[string]any{}},
			{ID: "call-3", Name: "tool_three", Arguments: map[string]any{}},
		},
		finalContent: "done",
	}
	al, _, cleanup := newHookTestLoop(t, provider)
	defer cleanup()

	tool1ExecCh := make(chan struct{}, 1)
	al.RegisterTool(&slowTool{name: "tool_two", duration: 100 * time.Millisecond, execCh: tool1ExecCh})
	al.RegisterTool(&slowTool{name: "tool_three", duration: 100 * time.Millisecond})

	hook := &respondHook{
		respondTools: map[string]bool{"tool_one": true},
	}
	if err := al.MountHook(NamedHook("respond-hook", hook)); err != nil {
		t.Fatalf("MountHook failed: %v", err)
	}

	sub := al.SubscribeEvents(32)
	defer al.UnsubscribeEvents(sub.ID)

	sessionKey := routing.BuildAgentMainSessionKey(routing.DefaultAgentID)

	type result struct {
		resp string
		err  error
	}
	resultCh := make(chan result, 1)
	go func() {
		resp, err := al.ProcessDirectWithChannel(
			context.Background(),
			"run tools",
			sessionKey,
			"cli",
			"chat1",
		)
		resultCh <- result{resp: resp, err: err}
	}()

	time.Sleep(50 * time.Millisecond)

	if err := al.InterruptGraceful("stop now"); err != nil {
		t.Fatalf("InterruptGraceful failed: %v", err)
	}

	select {
	case r := <-resultCh:
		if r.err != nil {
			t.Fatalf("unexpected error: %v", r.err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for result")
	}

	events := collectEventStream(sub.C)

	skippedEvts := filterEvents(events, EventKindToolExecSkipped)
	if len(skippedEvts) < 1 {
		t.Fatal("expected at least one ToolExecSkipped event after interrupt")
	}

	for _, evt := range skippedEvts {
		payload, ok := evt.Payload.(ToolExecSkippedPayload)
		if !ok {
			t.Fatalf("expected ToolExecSkippedPayload, got %T", evt.Payload)
		}
		if payload.Reason != "graceful interrupt requested" {
			t.Fatalf("expected skip reason 'graceful interrupt requested', got %q", payload.Reason)
		}
	}
}

func TestAgentLoop_HookRespond_SteeringSkipsRemaining(t *testing.T) {
	provider := &multiToolProvider{
		toolCalls: []providers.ToolCall{
			{ID: "call-1", Name: "tool_one", Arguments: map[string]any{}},
			{ID: "call-2", Name: "tool_two", Arguments: map[string]any{}},
			{ID: "call-3", Name: "tool_three", Arguments: map[string]any{}},
		},
		finalContent: "done",
	}
	al, _, cleanup := newHookTestLoop(t, provider)
	defer cleanup()

	al.RegisterTool(&slowTool{name: "tool_two", duration: 100 * time.Millisecond})
	al.RegisterTool(&slowTool{name: "tool_three", duration: 100 * time.Millisecond})

	hook := &respondHook{
		respondTools: map[string]bool{"tool_one": true},
	}
	if err := al.MountHook(NamedHook("respond-hook", hook)); err != nil {
		t.Fatalf("MountHook failed: %v", err)
	}

	sub := al.SubscribeEvents(32)
	defer al.UnsubscribeEvents(sub.ID)

	sessionKey := routing.BuildAgentMainSessionKey(routing.DefaultAgentID)

	type result struct {
		resp string
		err  error
	}
	resultCh := make(chan result, 1)
	go func() {
		resp, err := al.ProcessDirectWithChannel(
			context.Background(),
			"run tools",
			sessionKey,
			"cli",
			"chat1",
		)
		resultCh <- result{resp: resp, err: err}
	}()

	time.Sleep(50 * time.Millisecond)

	al.Steer(providers.Message{Role: "user", Content: "change direction"})

	select {
	case r := <-resultCh:
		if r.err != nil {
			t.Fatalf("unexpected error: %v", r.err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for result")
	}

	events := collectEventStream(sub.C)

	skippedEvts := filterEvents(events, EventKindToolExecSkipped)
	if len(skippedEvts) < 1 {
		t.Fatal("expected at least one ToolExecSkipped event after steering")
	}

	for _, evt := range skippedEvts {
		payload, ok := evt.Payload.(ToolExecSkippedPayload)
		if !ok {
			t.Fatalf("expected ToolExecSkippedPayload, got %T", evt.Payload)
		}
		if payload.Reason != "queued user steering message" {
			t.Fatalf("expected skip reason 'queued user steering message', got %q", payload.Reason)
		}
	}
}

func filterEvents(events []Event, kind EventKind) []Event {
	var result []Event
	for _, evt := range events {
		if evt.Kind == kind {
			result = append(result, evt)
		}
	}
	return result
}

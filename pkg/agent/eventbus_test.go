package agent

import (
	"context"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/routing"
	"github.com/sipeed/picoclaw/pkg/session"
	"github.com/sipeed/picoclaw/pkg/tools"
)

func TestEventBus_SubscribeEmitUnsubscribeClose(t *testing.T) {
	eventBus := NewEventBus()
	sub := eventBus.Subscribe(1)

	eventBus.Emit(Event{
		Kind: EventKindTurnStart,
		Meta: EventMeta{TurnID: "turn-1"},
	})

	select {
	case evt := <-sub.C:
		if evt.Kind != EventKindTurnStart {
			t.Fatalf("expected %v, got %v", EventKindTurnStart, evt.Kind)
		}
		if evt.Meta.TurnID != "turn-1" {
			t.Fatalf("expected turn id turn-1, got %q", evt.Meta.TurnID)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}

	eventBus.Unsubscribe(sub.ID)
	if _, ok := <-sub.C; ok {
		t.Fatal("expected subscriber channel to be closed after unsubscribe")
	}

	eventBus.Close()
	closedSub := eventBus.Subscribe(1)
	if _, ok := <-closedSub.C; ok {
		t.Fatal("expected closed bus to return a closed subscriber channel")
	}
}

func TestEventBus_DropsWhenSubscriberIsFull(t *testing.T) {
	eventBus := NewEventBus()
	sub := eventBus.Subscribe(1)
	defer eventBus.Unsubscribe(sub.ID)

	start := time.Now()
	for i := 0; i < 1000; i++ {
		eventBus.Emit(Event{Kind: EventKindLLMRequest})
	}

	if elapsed := time.Since(start); elapsed > 100*time.Millisecond {
		t.Fatalf("Emit took too long with a blocked subscriber: %s", elapsed)
	}

	if got := eventBus.Dropped(EventKindLLMRequest); got != 999 {
		t.Fatalf("expected 999 dropped events, got %d", got)
	}
}

type scriptedToolProvider struct {
	calls int
}

func (m *scriptedToolProvider) Chat(
	ctx context.Context,
	messages []providers.Message,
	toolDefs []providers.ToolDefinition,
	model string,
	opts map[string]any,
) (*providers.LLMResponse, error) {
	m.calls++
	if m.calls == 1 {
		return &providers.LLMResponse{
			ToolCalls: []providers.ToolCall{
				{
					ID:        "call-1",
					Name:      "mock_custom",
					Arguments: map[string]any{"task": "ping"},
				},
			},
		}, nil
	}

	return &providers.LLMResponse{
		Content: "done",
	}, nil
}

func (m *scriptedToolProvider) GetDefaultModel() string {
	return "scripted-tool-model"
}

func TestAgentLoop_EmitsMinimalTurnEvents(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-eventbus-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

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

	msgBus := bus.NewMessageBus()
	provider := &scriptedToolProvider{}
	al := NewAgentLoop(cfg, msgBus, provider)
	al.RegisterTool(&mockCustomTool{})
	defaultAgent := al.registry.GetDefaultAgent()
	if defaultAgent == nil {
		t.Fatal("expected default agent")
	}

	sub := al.SubscribeEvents(16)
	defer al.UnsubscribeEvents(sub.ID)

	response, err := al.runAgentLoop(context.Background(), defaultAgent, processOptions{
		SessionKey:      "session-1",
		Channel:         "cli",
		ChatID:          "direct",
		UserMessage:     "run tool",
		DefaultResponse: defaultResponse,
		EnableSummary:   false,
		SendResponse:    false,
		InboundContext: &bus.InboundContext{
			Channel:  "cli",
			ChatID:   "direct",
			ChatType: "direct",
			SenderID: "tester",
		},
		RouteResult: &routing.ResolvedRoute{
			AgentID:   "main",
			Channel:   "cli",
			AccountID: routing.DefaultAccountID,
			SessionPolicy: routing.SessionPolicy{
				Dimensions: []string{"sender"},
			},
			MatchedBy: "default",
		},
		SessionScope: &session.SessionScope{
			Version:    session.ScopeVersionV1,
			AgentID:    "main",
			Channel:    "cli",
			Account:    routing.DefaultAccountID,
			Dimensions: []string{"sender"},
			Values: map[string]string{
				"sender": "tester",
			},
		},
	})
	if err != nil {
		t.Fatalf("runAgentLoop failed: %v", err)
	}
	if response != "done" {
		t.Fatalf("expected final response 'done', got %q", response)
	}

	events := collectEventStream(sub.C)
	if len(events) != 8 {
		t.Fatalf("expected 8 events, got %d", len(events))
	}

	kinds := make([]EventKind, 0, len(events))
	for _, evt := range events {
		kinds = append(kinds, evt.Kind)
	}

	expectedKinds := []EventKind{
		EventKindTurnStart,
		EventKindLLMRequest,
		EventKindLLMResponse,
		EventKindToolExecStart,
		EventKindToolExecEnd,
		EventKindLLMRequest,
		EventKindLLMResponse,
		EventKindTurnEnd,
	}
	if !slices.Equal(kinds, expectedKinds) {
		t.Fatalf("unexpected event sequence: got %v want %v", kinds, expectedKinds)
	}

	turnID := events[0].Meta.TurnID
	for i, evt := range events {
		if evt.Meta.TurnID != turnID {
			t.Fatalf("event %d has mismatched turn id %q, want %q", i, evt.Meta.TurnID, turnID)
		}
		if evt.Meta.SessionKey != "session-1" {
			t.Fatalf("event %d has session key %q, want session-1", i, evt.Meta.SessionKey)
		}
		if evt.Context == nil || evt.Context.Inbound == nil {
			t.Fatalf("event %d missing inbound turn context", i)
		}
		if evt.Context.Inbound.Channel != "cli" || evt.Context.Inbound.SenderID != "tester" {
			t.Fatalf("event %d inbound context = %+v", i, evt.Context.Inbound)
		}
		if evt.Context.Route == nil || evt.Context.Route.AgentID != "main" {
			t.Fatalf("event %d missing route context: %+v", i, evt.Context.Route)
		}
		if evt.Context.Scope == nil || evt.Context.Scope.Values["sender"] != "tester" {
			t.Fatalf("event %d missing session scope: %+v", i, evt.Context.Scope)
		}
	}

	startPayload, ok := events[0].Payload.(TurnStartPayload)
	if !ok {
		t.Fatalf("expected TurnStartPayload, got %T", events[0].Payload)
	}
	if startPayload.UserMessage != "run tool" {
		t.Fatalf("expected user message 'run tool', got %q", startPayload.UserMessage)
	}

	toolStartPayload, ok := events[3].Payload.(ToolExecStartPayload)
	if !ok {
		t.Fatalf("expected ToolExecStartPayload, got %T", events[3].Payload)
	}
	if toolStartPayload.Tool != "mock_custom" {
		t.Fatalf("expected tool name mock_custom, got %q", toolStartPayload.Tool)
	}

	toolEndPayload, ok := events[4].Payload.(ToolExecEndPayload)
	if !ok {
		t.Fatalf("expected ToolExecEndPayload, got %T", events[4].Payload)
	}
	if toolEndPayload.Tool != "mock_custom" {
		t.Fatalf("expected tool end payload for mock_custom, got %q", toolEndPayload.Tool)
	}
	if toolEndPayload.IsError {
		t.Fatal("expected mock_custom tool to succeed")
	}

	turnEndPayload, ok := events[len(events)-1].Payload.(TurnEndPayload)
	if !ok {
		t.Fatalf("expected TurnEndPayload, got %T", events[len(events)-1].Payload)
	}
	if turnEndPayload.Status != TurnEndStatusCompleted {
		t.Fatalf("expected completed turn, got %q", turnEndPayload.Status)
	}
	if turnEndPayload.Iterations != 2 {
		t.Fatalf("expected 2 iterations, got %d", turnEndPayload.Iterations)
	}
}

func TestAgentLoop_EmitsSteeringAndSkippedToolEvents(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-eventbus-steering-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

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

	tool1ExecCh := make(chan struct{})
	tool1 := &slowTool{name: "tool_one", duration: 50 * time.Millisecond, execCh: tool1ExecCh}
	tool2 := &slowTool{name: "tool_two", duration: 50 * time.Millisecond}

	provider := &toolCallProvider{
		toolCalls: []providers.ToolCall{
			{
				ID:   "call_1",
				Type: "function",
				Name: "tool_one",
				Function: &providers.FunctionCall{
					Name:      "tool_one",
					Arguments: "{}",
				},
				Arguments: map[string]any{},
			},
			{
				ID:   "call_2",
				Type: "function",
				Name: "tool_two",
				Function: &providers.FunctionCall{
					Name:      "tool_two",
					Arguments: "{}",
				},
				Arguments: map[string]any{},
			},
		},
		finalResp: "steered response",
	}

	msgBus := bus.NewMessageBus()
	al := NewAgentLoop(cfg, msgBus, provider)
	al.RegisterTool(tool1)
	al.RegisterTool(tool2)

	sub := al.SubscribeEvents(32)
	defer al.UnsubscribeEvents(sub.ID)

	resultCh := make(chan string, 1)
	go func() {
		resp, _ := al.ProcessDirectWithChannel(context.Background(), "do something", "test-session", "test", "chat1")
		resultCh <- resp
	}()

	select {
	case <-tool1ExecCh:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for tool_one to start")
	}

	if err := al.Steer(providers.Message{Role: "user", Content: "change course"}); err != nil {
		t.Fatalf("Steer failed: %v", err)
	}

	select {
	case resp := <-resultCh:
		if resp != "steered response" {
			t.Fatalf("expected steered response, got %q", resp)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for steered response")
	}

	events := collectEventStream(sub.C)
	steeringEvt, ok := findEvent(events, EventKindSteeringInjected)
	if !ok {
		t.Fatal("expected steering injected event")
	}
	steeringPayload, ok := steeringEvt.Payload.(SteeringInjectedPayload)
	if !ok {
		t.Fatalf("expected SteeringInjectedPayload, got %T", steeringEvt.Payload)
	}
	if steeringPayload.Count != 1 {
		t.Fatalf("expected 1 steering message, got %d", steeringPayload.Count)
	}

	skippedEvt, ok := findEvent(events, EventKindToolExecSkipped)
	if !ok {
		t.Fatal("expected skipped tool event")
	}
	skippedPayload, ok := skippedEvt.Payload.(ToolExecSkippedPayload)
	if !ok {
		t.Fatalf("expected ToolExecSkippedPayload, got %T", skippedEvt.Payload)
	}
	if skippedPayload.Tool != "tool_two" {
		t.Fatalf("expected skipped tool_two, got %q", skippedPayload.Tool)
	}

	interruptEvt, ok := findEvent(events, EventKindInterruptReceived)
	if !ok {
		t.Fatal("expected interrupt received event")
	}
	interruptPayload, ok := interruptEvt.Payload.(InterruptReceivedPayload)
	if !ok {
		t.Fatalf("expected InterruptReceivedPayload, got %T", interruptEvt.Payload)
	}
	if interruptPayload.Role != "user" {
		t.Fatalf("expected interrupt role user, got %q", interruptPayload.Role)
	}
	if interruptPayload.Kind != InterruptKindSteering {
		t.Fatalf("expected steering interrupt kind, got %q", interruptPayload.Kind)
	}
	if interruptPayload.ContentLen != len("change course") {
		t.Fatalf("expected interrupt content len %d, got %d", len("change course"), interruptPayload.ContentLen)
	}
}

func TestAgentLoop_EmitsContextCompressEventOnRetry(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-eventbus-compress-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

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

	contextErr := stringError("InvalidParameter: Total tokens of image and text exceed max message tokens")
	provider := &failFirstMockProvider{
		failures:    1,
		failError:   contextErr,
		successResp: "Recovered from context error",
	}
	msgBus := bus.NewMessageBus()
	al := NewAgentLoop(cfg, msgBus, provider)
	defaultAgent := al.registry.GetDefaultAgent()
	if defaultAgent == nil {
		t.Fatal("expected default agent")
	}

	defaultAgent.Sessions.SetHistory("session-1", []providers.Message{
		{Role: "user", Content: "Old message 1"},
		{Role: "assistant", Content: "Old response 1"},
		{Role: "user", Content: "Old message 2"},
		{Role: "assistant", Content: "Old response 2"},
		{Role: "user", Content: "Trigger message"},
	})

	sub := al.SubscribeEvents(16)
	defer al.UnsubscribeEvents(sub.ID)

	resp, err := al.runAgentLoop(context.Background(), defaultAgent, processOptions{
		SessionKey:      "session-1",
		Channel:         "cli",
		ChatID:          "direct",
		UserMessage:     "Trigger message",
		DefaultResponse: defaultResponse,
		EnableSummary:   false,
		SendResponse:    false,
	})
	if err != nil {
		t.Fatalf("runAgentLoop failed: %v", err)
	}
	if resp != "Recovered from context error" {
		t.Fatalf("expected retry success, got %q", resp)
	}

	events := collectEventStream(sub.C)
	retryEvt, ok := findEvent(events, EventKindLLMRetry)
	if !ok {
		t.Fatal("expected llm retry event")
	}
	retryPayload, ok := retryEvt.Payload.(LLMRetryPayload)
	if !ok {
		t.Fatalf("expected LLMRetryPayload, got %T", retryEvt.Payload)
	}
	if retryPayload.Reason != "context_limit" {
		t.Fatalf("expected context_limit retry reason, got %q", retryPayload.Reason)
	}
	if retryPayload.Attempt != 1 {
		t.Fatalf("expected retry attempt 1, got %d", retryPayload.Attempt)
	}

	compressEvt, ok := findEvent(events, EventKindContextCompress)
	if !ok {
		t.Fatal("expected context compress event")
	}
	payload, ok := compressEvt.Payload.(ContextCompressPayload)
	if !ok {
		t.Fatalf("expected ContextCompressPayload, got %T", compressEvt.Payload)
	}
	if payload.Reason != ContextCompressReasonRetry {
		t.Fatalf("expected retry compress reason, got %q", payload.Reason)
	}
	if payload.DroppedMessages == 0 {
		t.Fatal("expected dropped messages to be recorded")
	}
}

func TestAgentLoop_EmitsSessionSummarizeEvent(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-eventbus-summary-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:                 tmpDir,
				ModelName:                 "test-model",
				MaxTokens:                 4096,
				MaxToolIterations:         10,
				ContextWindow:             8000,
				SummarizeMessageThreshold: 2,
				SummarizeTokenPercent:     75,
			},
		},
	}

	msgBus := bus.NewMessageBus()
	al := NewAgentLoop(cfg, msgBus, &simpleMockProvider{response: "summary text"})
	defaultAgent := al.registry.GetDefaultAgent()
	if defaultAgent == nil {
		t.Fatal("expected default agent")
	}

	defaultAgent.Sessions.SetHistory("session-1", []providers.Message{
		{Role: "user", Content: "Question one"},
		{Role: "assistant", Content: "Answer one"},
		{Role: "user", Content: "Question two"},
		{Role: "assistant", Content: "Answer two"},
		{Role: "user", Content: "Question three"},
		{Role: "assistant", Content: "Answer three"},
	})

	sub := al.SubscribeEvents(16)
	defer al.UnsubscribeEvents(sub.ID)

	lcm := &legacyContextManager{al: al}
	lcm.summarizeSession(defaultAgent, "session-1")

	events := collectEventStream(sub.C)
	summaryEvt, ok := findEvent(events, EventKindSessionSummarize)
	if !ok {
		t.Fatal("expected session summarize event")
	}
	payload, ok := summaryEvt.Payload.(SessionSummarizePayload)
	if !ok {
		t.Fatalf("expected SessionSummarizePayload, got %T", summaryEvt.Payload)
	}
	if payload.SummaryLen == 0 {
		t.Fatal("expected non-empty summary length")
	}
}

func TestAgentLoop_EmitsFollowUpQueuedEvent(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-eventbus-followup-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

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

	provider := &toolCallProvider{
		toolCalls: []providers.ToolCall{
			{
				ID:   "call_async_1",
				Type: "function",
				Name: "async_followup",
				Function: &providers.FunctionCall{
					Name:      "async_followup",
					Arguments: "{}",
				},
				Arguments: map[string]any{},
			},
		},
		finalResp: "async launched",
	}

	msgBus := bus.NewMessageBus()
	al := NewAgentLoop(cfg, msgBus, provider)
	doneCh := make(chan struct{})
	al.RegisterTool(&asyncFollowUpTool{
		name:          "async_followup",
		followUpText:  "background result",
		completionSig: doneCh,
	})
	defaultAgent := al.registry.GetDefaultAgent()
	if defaultAgent == nil {
		t.Fatal("expected default agent")
	}

	sub := al.SubscribeEvents(32)
	defer al.UnsubscribeEvents(sub.ID)

	resp, err := al.runAgentLoop(context.Background(), defaultAgent, processOptions{
		SessionKey:      "session-1",
		Channel:         "cli",
		ChatID:          "direct",
		UserMessage:     "run async tool",
		DefaultResponse: defaultResponse,
		EnableSummary:   false,
		SendResponse:    false,
	})
	if err != nil {
		t.Fatalf("runAgentLoop failed: %v", err)
	}
	if resp != "async launched" {
		t.Fatalf("expected final response 'async launched', got %q", resp)
	}

	select {
	case <-doneCh:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for async tool completion")
	}

	followUpEvt := waitForEvent(t, sub.C, 2*time.Second, func(evt Event) bool {
		return evt.Kind == EventKindFollowUpQueued
	})
	payload, ok := followUpEvt.Payload.(FollowUpQueuedPayload)
	if !ok {
		t.Fatalf("expected FollowUpQueuedPayload, got %T", followUpEvt.Payload)
	}
	if payload.SourceTool != "async_followup" {
		t.Fatalf("expected source tool async_followup, got %q", payload.SourceTool)
	}
	if payload.ContentLen != len("background result") {
		t.Fatalf("expected content len %d, got %d", len("background result"), payload.ContentLen)
	}
	if followUpEvt.Meta.SessionKey != "session-1" {
		t.Fatalf("expected session key session-1, got %q", followUpEvt.Meta.SessionKey)
	}
	if followUpEvt.Meta.TurnID == "" {
		t.Fatal("expected follow-up event to include turn id")
	}
}

func collectEventStream(ch <-chan Event) []Event {
	var events []Event
	for {
		select {
		case evt, ok := <-ch:
			if !ok {
				return events
			}
			events = append(events, evt)
		default:
			return events
		}
	}
}

func waitForEvent(t *testing.T, ch <-chan Event, timeout time.Duration, match func(Event) bool) Event {
	t.Helper()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for {
		select {
		case evt, ok := <-ch:
			if !ok {
				t.Fatal("event stream closed before expected event arrived")
			}
			if match(evt) {
				return evt
			}
		case <-timer.C:
			t.Fatal("timed out waiting for expected event")
		}
	}
}

func findEvent(events []Event, kind EventKind) (Event, bool) {
	for _, evt := range events {
		if evt.Kind == kind {
			return evt, true
		}
	}
	return Event{}, false
}

type stringError string

func (e stringError) Error() string {
	return string(e)
}

type asyncFollowUpTool struct {
	name          string
	followUpText  string
	completionSig chan struct{}
}

func (t *asyncFollowUpTool) Name() string {
	return t.name
}

func (t *asyncFollowUpTool) Description() string {
	return "async follow-up tool for testing"
}

func (t *asyncFollowUpTool) Parameters() map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
}

func (t *asyncFollowUpTool) Execute(ctx context.Context, args map[string]any) *tools.ToolResult {
	return tools.AsyncResult("async follow-up scheduled")
}

func (t *asyncFollowUpTool) ExecuteAsync(
	ctx context.Context,
	args map[string]any,
	cb tools.AsyncCallback,
) *tools.ToolResult {
	go func() {
		cb(ctx, &tools.ToolResult{ForLLM: t.followUpText})
		if t.completionSig != nil {
			close(t.completionSig)
		}
	}()
	return tools.AsyncResult("async follow-up scheduled")
}

var (
	_ tools.Tool          = (*mockCustomTool)(nil)
	_ tools.AsyncExecutor = (*asyncFollowUpTool)(nil)
)

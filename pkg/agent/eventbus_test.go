package agent

import (
	"context"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	runtimeevents "github.com/sipeed/picoclaw/pkg/events"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/routing"
	"github.com/sipeed/picoclaw/pkg/session"
	"github.com/sipeed/picoclaw/pkg/tools"
)

func TestAgentLoop_PublishesRuntimeEvents(t *testing.T) {
	runtimeBus := runtimeevents.NewBus()
	al := &AgentLoop{
		runtimeEvents: runtimeBus,
	}
	defer func() {
		if err := runtimeBus.Close(); err != nil {
			t.Errorf("runtime bus close failed: %v", err)
		}
	}()

	runtimeSub, runtimeCh, err := al.RuntimeEvents().OfKind(runtimeevents.KindAgentToolExecStart).SubscribeChan(
		context.Background(),
		runtimeevents.SubscribeOptions{Name: "runtime", Buffer: 1},
	)
	if err != nil {
		t.Fatalf("SubscribeChan failed: %v", err)
	}
	defer func() {
		if err := runtimeSub.Close(); err != nil {
			t.Errorf("runtime subscription close failed: %v", err)
		}
	}()

	al.emitEvent(
		runtimeevents.KindAgentToolExecStart,
		HookMeta{
			AgentID:      "main",
			TurnID:       "turn-1",
			ParentTurnID: "parent-turn",
			SessionKey:   "session-1",
			Iteration:    2,
			TracePath:    "trace/root",
			Source:       "pipeline_execute",
			turnContext: &TurnContext{
				Inbound: &bus.InboundContext{
					Channel:   "cli",
					Account:   "default",
					ChatID:    "direct",
					ChatType:  "direct",
					SenderID:  "tester",
					MessageID: "msg-1",
					TopicID:   "topic-1",
				},
			},
		},
		ToolExecStartPayload{Tool: "mock_custom", Arguments: map[string]any{"task": "ping"}},
	)

	runtimeEvt := receiveRuntimeEvent(t, runtimeCh)
	if runtimeEvt.Kind != runtimeevents.KindAgentToolExecStart {
		t.Fatalf("runtime kind = %q, want %q", runtimeEvt.Kind, runtimeevents.KindAgentToolExecStart)
	}
	if runtimeEvt.Source != (runtimeevents.Source{Component: "agent", Name: "main"}) {
		t.Fatalf("runtime source = %+v", runtimeEvt.Source)
	}
	if runtimeEvt.Scope.AgentID != "main" ||
		runtimeEvt.Scope.SessionKey != "session-1" ||
		runtimeEvt.Scope.TurnID != "turn-1" ||
		runtimeEvt.Scope.Channel != "cli" ||
		runtimeEvt.Scope.Account != "default" ||
		runtimeEvt.Scope.ChatID != "direct" ||
		runtimeEvt.Scope.TopicID != "topic-1" ||
		runtimeEvt.Scope.ChatType != "direct" ||
		runtimeEvt.Scope.SenderID != "tester" ||
		runtimeEvt.Scope.MessageID != "msg-1" {
		t.Fatalf("runtime scope = %+v", runtimeEvt.Scope)
	}
	if runtimeEvt.Correlation.TraceID != "trace/root" ||
		runtimeEvt.Correlation.ParentTurnID != "parent-turn" {
		t.Fatalf("runtime correlation = %+v", runtimeEvt.Correlation)
	}
	if runtimeEvt.Attrs["agent_source"] != "pipeline_execute" || runtimeEvt.Attrs["iteration"] != 2 {
		t.Fatalf("runtime attrs = %+v", runtimeEvt.Attrs)
	}
	payload, ok := runtimeEvt.Payload.(ToolExecStartPayload)
	if !ok {
		t.Fatalf("runtime payload = %T, want ToolExecStartPayload", runtimeEvt.Payload)
	}
	if payload.Tool != "mock_custom" {
		t.Fatalf("runtime payload tool = %q, want mock_custom", payload.Tool)
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

	expectedKinds := []runtimeevents.Kind{
		runtimeevents.KindAgentTurnStart,
		runtimeevents.KindAgentLLMRequest,
		runtimeevents.KindAgentLLMResponse,
		runtimeevents.KindAgentToolExecStart,
		runtimeevents.KindAgentToolExecEnd,
		runtimeevents.KindAgentLLMRequest,
		runtimeevents.KindAgentLLMResponse,
		runtimeevents.KindAgentTurnEnd,
	}
	runtimeCh, closeRuntimeEvents := subscribeRuntimeEventsForTest(t, al, 16, expectedKinds...)
	defer closeRuntimeEvents()

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

	events := collectRuntimeEventStream(runtimeCh)
	if len(events) != 8 {
		t.Fatalf("expected 8 events, got %d", len(events))
	}

	kinds := make([]runtimeevents.Kind, 0, len(events))
	for _, evt := range events {
		kinds = append(kinds, evt.Kind)
	}

	if !slices.Equal(kinds, expectedKinds) {
		t.Fatalf("unexpected event sequence: got %v want %v", kinds, expectedKinds)
	}

	turnID := events[0].Scope.TurnID
	if turnID == "" {
		t.Fatal("expected runtime events to include turn id")
	}
	for i, evt := range events {
		if evt.Scope.TurnID != turnID {
			t.Fatalf("event %d has mismatched turn id %q, want %q", i, evt.Scope.TurnID, turnID)
		}
		if evt.Scope.SessionKey != "session-1" {
			t.Fatalf("event %d has session key %q, want session-1", i, evt.Scope.SessionKey)
		}
		if evt.Scope.Channel != "cli" || evt.Scope.ChatID != "direct" || evt.Scope.SenderID != "tester" {
			t.Fatalf("event %d scope = %+v", i, evt.Scope)
		}
		if evt.Scope.AgentID != "main" {
			t.Fatalf("event %d has agent id %q, want main", i, evt.Scope.AgentID)
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

	runtimeCh, closeRuntimeEvents := subscribeRuntimeEventsForTest(
		t,
		al,
		32,
		runtimeevents.KindAgentSteeringInjected,
		runtimeevents.KindAgentToolExecSkipped,
		runtimeevents.KindAgentInterruptReceived,
	)
	defer closeRuntimeEvents()

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

	events := collectRuntimeEventStream(runtimeCh)
	steeringEvt, ok := findRuntimeEvent(events, runtimeevents.KindAgentSteeringInjected)
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

	skippedEvt, ok := findRuntimeEvent(events, runtimeevents.KindAgentToolExecSkipped)
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

	interruptEvt, ok := findRuntimeEvent(events, runtimeevents.KindAgentInterruptReceived)
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

	runtimeCh, closeRuntimeEvents := subscribeRuntimeEventsForTest(
		t,
		al,
		16,
		runtimeevents.KindAgentLLMRetry,
		runtimeevents.KindAgentContextCompress,
	)
	defer closeRuntimeEvents()

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

	events := collectRuntimeEventStream(runtimeCh)
	retryEvt, ok := findRuntimeEvent(events, runtimeevents.KindAgentLLMRetry)
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

	compressEvt, ok := findRuntimeEvent(events, runtimeevents.KindAgentContextCompress)
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

	runtimeCh, closeRuntimeEvents := subscribeRuntimeEventsForTest(
		t,
		al,
		16,
		runtimeevents.KindAgentSessionSummarize,
	)
	defer closeRuntimeEvents()

	lcm := &legacyContextManager{al: al}
	lcm.summarizeSession(defaultAgent, "session-1")

	events := collectRuntimeEventStream(runtimeCh)
	summaryEvt, ok := findRuntimeEvent(events, runtimeevents.KindAgentSessionSummarize)
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

	runtimeCh, closeRuntimeEvents := subscribeRuntimeEventsForTest(
		t,
		al,
		32,
		runtimeevents.KindAgentFollowUpQueued,
	)
	defer closeRuntimeEvents()

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

	followUpEvt := waitForRuntimeEvent(t, runtimeCh, 2*time.Second, func(evt runtimeevents.Event) bool {
		return evt.Kind == runtimeevents.KindAgentFollowUpQueued
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
	if followUpEvt.Scope.SessionKey != "session-1" {
		t.Fatalf("expected session key session-1, got %q", followUpEvt.Scope.SessionKey)
	}
	if followUpEvt.Scope.TurnID == "" {
		t.Fatal("expected follow-up event to include turn id")
	}
}

func receiveRuntimeEvent(t *testing.T, ch <-chan runtimeevents.Event) runtimeevents.Event {
	t.Helper()

	select {
	case evt, ok := <-ch:
		if !ok {
			t.Fatal("runtime event stream closed before expected event arrived")
		}
		return evt
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for runtime event")
		return runtimeevents.Event{}
	}
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

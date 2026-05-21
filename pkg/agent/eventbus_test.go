package agent

import (
	"context"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	runtimeevents "github.com/sipeed/picoclaw/pkg/events"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/routing"
	"github.com/sipeed/picoclaw/pkg/session"
	taskregistry "github.com/sipeed/picoclaw/pkg/tasks"
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
	forUserText   string
	completionSig chan struct{}
	deliveryMode  tools.AsyncDeliveryMode
	taskID        string
}

func (t *asyncFollowUpTool) Name() string {
	return t.name
}

func (t *asyncFollowUpTool) Description() string {
	return "async follow-up tool for testing"
}

func (t *asyncFollowUpTool) Parameters() map[string]any {
	return map[string]any{
		"type":                 "object",
		"properties":           map[string]any{},
		"additionalProperties": true,
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
		res := &tools.ToolResult{ForLLM: t.followUpText, ForUser: t.forUserText}
		if t.deliveryMode != "" {
			res.WithAsyncDelivery(t.deliveryMode)
		}
		if t.taskID != "" {
			res.WithAsyncTaskID(t.taskID)
		}
		cb(ctx, res)
		if t.completionSig != nil {
			close(t.completionSig)
		}
	}()
	return tools.AsyncResult("async follow-up scheduled")
}

func waitForInboundMessage(t *testing.T, ch <-chan bus.InboundMessage, timeout time.Duration, match func(bus.InboundMessage) bool) bus.InboundMessage {
	t.Helper()
	deadline := time.After(timeout)
	for {
		select {
		case msg := <-ch:
			if match == nil || match(msg) {
				return msg
			}
		case <-deadline:
			t.Fatal("timeout waiting for inbound message")
		}
	}
}

func waitForOutboundMessage(t *testing.T, ch <-chan bus.OutboundMessage, timeout time.Duration, match func(bus.OutboundMessage) bool) bus.OutboundMessage {
	t.Helper()
	deadline := time.After(timeout)
	for {
		select {
		case msg := <-ch:
			if match == nil || match(msg) {
				return msg
			}
		case <-deadline:
			t.Fatal("timeout waiting for outbound message")
		}
	}
}

func waitForToolCallProviderCalls(t *testing.T, provider *toolCallProvider, timeout time.Duration, minCalls int) {
	t.Helper()
	deadline := time.After(timeout)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		provider.mu.Lock()
		calls := provider.calls
		provider.mu.Unlock()
		if calls >= minCalls {
			return
		}
		select {
		case <-ticker.C:
		case <-deadline:
			t.Fatalf("timeout waiting for provider calls >= %d, got %d", minCalls, calls)
		}
	}
}

func assertNoSyntheticAsyncCompletionInbound(t *testing.T, msgBus *bus.MessageBus) {
	t.Helper()
	select {
	case inbound := <-msgBus.InboundChan():
		if isAsyncCompletionSystemMessage(inbound) {
			t.Fatalf("unexpected synthetic async completion inbound: %+v", inbound)
		}
	case <-time.After(100 * time.Millisecond):
	}
}

func upsertAsyncTaskForTest(t *testing.T, al *AgentLoop, workspace, taskID string) {
	t.Helper()
	registry := al.taskRegistryForWorkspace(workspace)
	if registry == nil {
		t.Fatal("expected task registry")
	}
	if err := registry.Upsert(taskregistry.Record{
		TaskID:         taskID,
		Runtime:        taskregistry.RuntimeSubagent,
		TaskKind:       "spawn",
		Task:           "async test task",
		Status:         taskregistry.StatusRunning,
		DeliveryStatus: taskregistry.DeliveryPending,
	}); err != nil {
		t.Fatalf("upsert task: %v", err)
	}
}

func assertTaskDeliveryStatusForTest(
	t *testing.T,
	al *AgentLoop,
	workspace string,
	taskID string,
	want taskregistry.DeliveryStatus,
) {
	t.Helper()
	registry := al.taskRegistryForWorkspace(workspace)
	if registry == nil {
		t.Fatal("expected task registry")
	}
	rec, ok := registry.Get(taskID)
	if !ok {
		t.Fatalf("task %q not found", taskID)
	}
	if rec.DeliveryStatus != want {
		t.Fatalf("DeliveryStatus = %q, want %q", rec.DeliveryStatus, want)
	}
}

func TestAgentLoop_TaskRegistryReconcilesPendingTerminalDelivery(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace: tmpDir,
				ModelName: "test-model",
				MaxTokens: 4096,
			},
		},
	}
	registry := taskregistry.NewRegistry(taskregistry.WorkspaceStorePath(tmpDir))
	if err := registry.Upsert(taskregistry.Record{
		TaskID:         "stale-completion",
		Runtime:        taskregistry.RuntimeSubagent,
		TaskKind:       "spawn",
		Task:           "stale task",
		Status:         taskregistry.StatusSucceeded,
		DeliveryStatus: taskregistry.DeliveryPending,
	}); err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}

	al := NewAgentLoop(cfg, bus.NewMessageBus(), &simpleMockProvider{response: "ok"})
	rec, ok := al.taskRegistryForWorkspace(tmpDir).Get("stale-completion")
	if !ok {
		t.Fatal("expected stale task")
	}
	if rec.DeliveryStatus != taskregistry.DeliveryParentMissing {
		t.Fatalf("DeliveryStatus = %q, want %q", rec.DeliveryStatus, taskregistry.DeliveryParentMissing)
	}
	if !strings.Contains(rec.Error, "restart/reload") {
		t.Fatalf("Error = %q, want restart/reload marker", rec.Error)
	}
}

var (
	_ tools.Tool          = (*mockCustomTool)(nil)
	_ tools.AsyncExecutor = (*asyncFollowUpTool)(nil)
)

func TestAgentLoop_AsyncToolUserOnly_DoesNotEmitFollowUpQueued(t *testing.T) {
	tmpDir := t.TempDir()

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
				Name: "async_followup_user_only",
				Function: &providers.FunctionCall{
					Name:      "async_followup_user_only",
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
		name:          "async_followup_user_only",
		followUpText:  "background result",
		completionSig: doneCh,
		deliveryMode:  tools.AsyncDeliveryUserOnly,
	})
	defaultAgent := al.registry.GetDefaultAgent()
	if defaultAgent == nil {
		t.Fatal("expected default agent")
	}

	runtimeCh, closeRuntimeEvents := subscribeRuntimeEventsForTest(
		t,
		al,
		8,
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

	select {
	case evt := <-runtimeCh:
		t.Fatalf("unexpected follow-up queued event: %+v", evt)
	case <-time.After(200 * time.Millisecond):
	}
}

func TestAgentLoop_EmitsAsyncCompletionEvent(t *testing.T) {
	tmpDir := t.TempDir()

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
				Name: "async_completion_event",
				Function: &providers.FunctionCall{
					Name:      "async_completion_event",
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
		name:          "async_completion_event",
		followUpText:  "background result",
		completionSig: doneCh,
		deliveryMode:  tools.AsyncDeliveryUserOnly,
		taskID:        "subagent-42",
	})
	defaultAgent := al.registry.GetDefaultAgent()
	if defaultAgent == nil {
		t.Fatal("expected default agent")
	}

	runtimeCh, closeRuntimeEvents := subscribeRuntimeEventsForTest(
		t,
		al,
		8,
		runtimeevents.KindAgentAsyncCompletion,
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

	evt := waitForRuntimeEvent(t, runtimeCh, 2*time.Second, func(evt runtimeevents.Event) bool {
		return evt.Kind == runtimeevents.KindAgentAsyncCompletion
	})
	payload, ok := evt.Payload.(AsyncCompletionPayload)
	if !ok {
		t.Fatalf("expected AsyncCompletionPayload, got %T", evt.Payload)
	}
	if payload.SourceTool != "async_completion_event" {
		t.Fatalf("SourceTool = %q, want async_completion_event", payload.SourceTool)
	}
	if payload.DeliveryMode != string(tools.AsyncDeliveryUserOnly) {
		t.Fatalf("DeliveryMode = %q, want %q", payload.DeliveryMode, tools.AsyncDeliveryUserOnly)
	}
	if payload.ContentLen != len("background result") {
		t.Fatalf("ContentLen = %d, want %d", payload.ContentLen, len("background result"))
	}
	if payload.WillUser {
		t.Fatal("expected user delivery to be false because the result has no ForUser text")
	}
	if payload.WillParent {
		t.Fatal("expected parent delivery to be false")
	}
	if payload.CompletionID == "" {
		t.Fatal("expected completion id")
	}
	if payload.TaskID != "subagent-42" {
		t.Fatalf("TaskID = %q, want subagent-42", payload.TaskID)
	}
}

func TestAsyncToolResultDeliveryRouting(t *testing.T) {
	tests := []struct {
		name              string
		result            *tools.ToolResult
		wantPublishToUser bool
		wantQueueParent   bool
	}{
		{
			name:              "nil",
			result:            nil,
			wantPublishToUser: false,
			wantQueueParent:   false,
		},
		{
			name: "default_routes_to_user_and_parent",
			result: &tools.ToolResult{
				ForLLM:  "parent text",
				ForUser: "user text",
			},
			wantPublishToUser: true,
			wantQueueParent:   true,
		},
		{
			name: "user_only_routes_to_user_without_parent",
			result: (&tools.ToolResult{
				ForLLM:  "parent text",
				ForUser: "user text",
			}).WithAsyncDelivery(tools.AsyncDeliveryUserOnly),
			wantPublishToUser: true,
			wantQueueParent:   false,
		},
		{
			name: "parent_only_routes_to_parent_without_user",
			result: (&tools.ToolResult{
				ForLLM:  "parent text",
				ForUser: "user text",
			}).WithAsyncDelivery(tools.AsyncDeliveryParentOnly),
			wantPublishToUser: false,
			wantQueueParent:   true,
		},
		{
			name: "user_and_parent_routes_to_both",
			result: (&tools.ToolResult{
				ForLLM:  "parent text",
				ForUser: "user text",
			}).WithAsyncDelivery(tools.AsyncDeliveryUserAndParent),
			wantPublishToUser: true,
			wantQueueParent:   true,
		},
		{
			name: "silent_user_only_routes_nowhere",
			result: (&tools.ToolResult{
				ForLLM:  "parent text",
				ForUser: "user text",
				Silent:  true,
			}).WithAsyncDelivery(tools.AsyncDeliveryUserOnly),
			wantPublishToUser: false,
			wantQueueParent:   false,
		},
		{
			name: "empty_parent_content_does_not_queue",
			result: (&tools.ToolResult{
				ForUser: "user text",
			}).WithAsyncDelivery(tools.AsyncDeliveryParentOnly),
			wantPublishToUser: false,
			wantQueueParent:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldPublishAsyncToolResultToUser(tt.result); got != tt.wantPublishToUser {
				t.Fatalf("shouldPublishAsyncToolResultToUser() = %v, want %v", got, tt.wantPublishToUser)
			}
			if got := shouldQueueAsyncToolResultForParent(tt.result); got != tt.wantQueueParent {
				t.Fatalf("shouldQueueAsyncToolResultForParent() = %v, want %v", got, tt.wantQueueParent)
			}
		})
	}
}

func TestAgentLoop_AsyncUserOnlyAckSuppressesDefaultFinalResponse(t *testing.T) {
	tmpDir := t.TempDir()
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
				ID:        "call_spawn_1",
				Type:      "function",
				Name:      "spawn",
				Arguments: map[string]any{"task": "send video to user"},
			},
		},
		finalResp: "",
	}
	msgBus := bus.NewMessageBus()
	al := NewAgentLoop(cfg, msgBus, provider)
	doneCh := make(chan struct{})
	al.RegisterTool(&asyncFollowUpTool{
		name:          "spawn",
		followUpText:  "background result",
		completionSig: doneCh,
		deliveryMode:  tools.AsyncDeliveryUserOnly,
	})

	defaultAgent := al.registry.GetDefaultAgent()
	if defaultAgent == nil {
		t.Fatal("expected default agent")
	}
	resp, err := al.runAgentLoop(context.Background(), defaultAgent, processOptions{
		Dispatch: DispatchRequest{
			SessionKey:  "session-1",
			UserMessage: "send video",
			InboundContext: &bus.InboundContext{
				Channel:  "telegram",
				ChatID:   "chat-1",
				SenderID: "user-1",
			},
		},
		DefaultResponse: defaultResponse,
		SendResponse:    true,
	})
	if err != nil {
		t.Fatalf("runAgentLoop failed: %v", err)
	}
	if resp != "" {
		t.Fatalf("response = %q, want empty", resp)
	}
	if provider.calls != 1 {
		t.Fatalf("provider calls = %d, want 1", provider.calls)
	}
	select {
	case <-doneCh:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for async completion")
	}
	select {
	case outbound := <-msgBus.OutboundChan():
		t.Fatalf("unexpected outbound final response: %#v", outbound)
	default:
	}
}

func TestAgentLoop_AsyncParentOnlyQueuesFollowUpWithoutUserDelivery(t *testing.T) {
	tmpDir := t.TempDir()
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
				ID:        "call_delegate_1",
				Type:      "function",
				Name:      "delegate",
				Arguments: map[string]any{"agent_id": "media", "task": "download and summarize"},
			},
		},
		finalResp: "parent final response",
	}
	msgBus := bus.NewMessageBus()
	al := NewAgentLoop(cfg, msgBus, provider)
	parentTaskID := "async-parent-task"
	upsertAsyncTaskForTest(t, al, tmpDir, parentTaskID)
	doneCh := make(chan struct{})
	al.RegisterTool(&asyncFollowUpTool{
		name:          "delegate",
		followUpText:  "parent-only completion",
		forUserText:   "do not send this directly",
		completionSig: doneCh,
		deliveryMode:  tools.AsyncDeliveryParentOnly,
		taskID:        parentTaskID,
	})

	defaultAgent := al.registry.GetDefaultAgent()
	if defaultAgent == nil {
		t.Fatal("expected default agent")
	}
	resp, err := al.runAgentLoop(context.Background(), defaultAgent, processOptions{
		Dispatch: DispatchRequest{
			SessionKey:  "session-1",
			UserMessage: "run delegate",
			InboundContext: &bus.InboundContext{
				Channel:  "telegram",
				ChatID:   "chat-1",
				TopicID:  "topic-1",
				SenderID: "user-1",
			},
		},
		DefaultResponse: defaultResponse,
		SendResponse:    true,
	})
	if err != nil {
		t.Fatalf("runAgentLoop failed: %v", err)
	}
	if resp != "parent final response" {
		t.Fatalf("response = %q, want parent final response", resp)
	}
	select {
	case <-doneCh:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for async completion")
	}
	waitForToolCallProviderCalls(t, provider, 2*time.Second, 3)
	assertNoSyntheticAsyncCompletionInbound(t, msgBus)
	assertTaskDeliveryStatusForTest(t, al, tmpDir, parentTaskID, taskregistry.DeliverySessionQueued)
	select {
	case outbound := <-msgBus.OutboundChan():
		if strings.Contains(outbound.Content, "do not send this directly") {
			t.Fatalf("unexpected direct user delivery: %#v", outbound)
		}
	case <-time.After(200 * time.Millisecond):
	}
}

func TestAgentLoop_AsyncUserAndParentPublishesUserAndQueuesFollowUp(t *testing.T) {
	tmpDir := t.TempDir()
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
				ID:        "call_spawn_1",
				Type:      "function",
				Name:      "spawn",
				Arguments: map[string]any{"task": "download media", "delivery_mode": string(tools.AsyncDeliveryUserAndParent)},
			},
		},
		finalResp: "parent final response",
	}
	msgBus := bus.NewMessageBus()
	al := NewAgentLoop(cfg, msgBus, provider)
	userAndParentTaskID := "async-user-parent-task"
	upsertAsyncTaskForTest(t, al, tmpDir, userAndParentTaskID)
	doneCh := make(chan struct{})
	al.RegisterTool(&asyncFollowUpTool{
		name:          "spawn",
		followUpText:  "parent completion",
		forUserText:   "user completion",
		completionSig: doneCh,
		deliveryMode:  tools.AsyncDeliveryUserAndParent,
		taskID:        userAndParentTaskID,
	})

	defaultAgent := al.registry.GetDefaultAgent()
	if defaultAgent == nil {
		t.Fatal("expected default agent")
	}
	resp, err := al.runAgentLoop(context.Background(), defaultAgent, processOptions{
		Dispatch: DispatchRequest{
			SessionKey:  "session-1",
			UserMessage: "run spawn",
			InboundContext: &bus.InboundContext{
				Channel:  "telegram",
				ChatID:   "chat-1",
				TopicID:  "topic-1",
				SenderID: "user-1",
			},
		},
		DefaultResponse: defaultResponse,
		SendResponse:    true,
	})
	if err != nil {
		t.Fatalf("runAgentLoop failed: %v", err)
	}
	if resp != "parent final response" {
		t.Fatalf("response = %q, want parent final response", resp)
	}
	select {
	case <-doneCh:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for async completion")
	}
	outbound := waitForOutboundMessage(t, msgBus.OutboundChan(), 2*time.Second, func(msg bus.OutboundMessage) bool {
		return msg.Content == "user completion"
	})
	if outbound.Context.TopicID != "topic-1" {
		t.Fatalf("outbound topic_id = %q, want topic-1", outbound.Context.TopicID)
	}
	waitForToolCallProviderCalls(t, provider, 2*time.Second, 3)
	assertNoSyntheticAsyncCompletionInbound(t, msgBus)
	assertTaskDeliveryStatusForTest(t, al, tmpDir, userAndParentTaskID, taskregistry.DeliveryDelivered)
}

type captureMessagesProvider struct {
	response string
	calls    int
	messages []providers.Message
}

func (p *captureMessagesProvider) Chat(
	ctx context.Context,
	messages []providers.Message,
	tools []providers.ToolDefinition,
	model string,
	opts map[string]any,
) (*providers.LLMResponse, error) {
	p.calls++
	p.messages = append([]providers.Message(nil), messages...)
	return &providers.LLMResponse{Content: p.response}, nil
}

func (p *captureMessagesProvider) GetDefaultModel() string {
	return "capture-messages"
}

func TestProcessAsyncCompletionMessageUsesNoHistory(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace: tmpDir,
				ModelName: "test-model",
				MaxTokens: 4096,
			},
		},
	}
	provider := &captureMessagesProvider{response: "completion summary"}
	msgBus := bus.NewMessageBus()
	al := NewAgentLoop(cfg, msgBus, provider)
	defaultAgent := al.registry.GetDefaultAgent()
	if defaultAgent == nil {
		t.Fatal("expected default agent")
	}

	sessionKey := session.BuildMainSessionKey(defaultAgent.ID)
	defaultAgent.Sessions.AddMessage(sessionKey, "user", "old user message")
	defaultAgent.Sessions.AddMessage(sessionKey, "assistant", "[System: async:spawn] old background result")

	msg := bus.InboundMessage{
		Channel:  "system",
		ChatID:   "telegram:chat-1",
		SenderID: "async:spawn",
		Context: bus.InboundContext{
			Channel:  "system",
			ChatID:   "telegram:chat-1",
			ChatType: "direct",
			SenderID: "async:spawn",
			Raw: systemFollowUpAsyncCompletionRaw(&bus.InboundContext{
				Channel:  "telegram",
				ChatID:   "chat-1",
				ChatType: "direct",
			}, "telegram", "chat-1", "completion-1"),
		},
		Content: asyncCompletionPrompt("spawn", "fresh background result"),
	}

	got, err := al.processSystemMessage(context.Background(), msg)
	if err != nil {
		t.Fatalf("processSystemMessage failed: %v", err)
	}
	if got != "completion summary" {
		t.Fatalf("response = %q, want completion summary", got)
	}

	var prompt strings.Builder
	for _, message := range provider.messages {
		prompt.WriteString(message.Content)
		prompt.WriteByte('\n')
	}
	promptText := prompt.String()
	if !strings.Contains(promptText, "fresh background result") {
		t.Fatalf("prompt did not include fresh async completion result: %q", promptText)
	}
	if strings.Contains(promptText, "old background result") || strings.Contains(promptText, "old user message") {
		t.Fatalf("async completion prompt leaked session history: %q", promptText)
	}

	history := defaultAgent.Sessions.GetHistory(sessionKey)
	if len(history) != 2 {
		t.Fatalf("history length = %d, want 2 existing messages only", len(history))
	}
}

func TestProcessAsyncCompletionMessageSkipsDuplicateCompletionID(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace: tmpDir,
				ModelName: "test-model",
				MaxTokens: 4096,
			},
		},
	}
	provider := &captureMessagesProvider{response: "completion summary"}
	msgBus := bus.NewMessageBus()
	al := NewAgentLoop(cfg, msgBus, provider)

	msg := bus.InboundMessage{
		Channel:  "system",
		ChatID:   "telegram:chat-1",
		SenderID: "async:spawn",
		Context: bus.InboundContext{
			Channel:  "system",
			ChatID:   "telegram:chat-1",
			ChatType: "direct",
			SenderID: "async:spawn",
			Raw: systemFollowUpAsyncCompletionRaw(&bus.InboundContext{
				Channel:  "telegram",
				ChatID:   "chat-1",
				ChatType: "direct",
			}, "telegram", "chat-1", "same-completion"),
		},
		Content: asyncCompletionPrompt("spawn", "fresh background result"),
	}

	if _, err := al.processSystemMessage(context.Background(), msg); err != nil {
		t.Fatalf("first processSystemMessage failed: %v", err)
	}
	if _, err := al.processSystemMessage(context.Background(), msg); err != nil {
		t.Fatalf("second processSystemMessage failed: %v", err)
	}
	if provider.calls != 1 {
		t.Fatalf("provider calls = %d, want 1", provider.calls)
	}
}

func TestProcessAsyncCompletionTypedPathDoesNotPublishSystemInbound(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace: tmpDir,
				ModelName: "test-model",
				MaxTokens: 4096,
			},
		},
	}
	provider := &captureMessagesProvider{response: "typed completion summary"}
	msgBus := bus.NewMessageBus()
	al := NewAgentLoop(cfg, msgBus, provider)

	got, err := al.processAsyncCompletion(context.Background(), AsyncCompletionInput{
		SourceTool:   "spawn",
		CompletionID: "typed-completion-1",
		Content:      asyncCompletionPrompt("spawn", "typed background result"),
		Origin: bus.InboundContext{
			Channel:  "telegram",
			ChatID:   "chat-1",
			ChatType: "direct",
			SenderID: "async:spawn",
		},
		SenderID: "async:spawn",
	})
	if err != nil {
		t.Fatalf("processAsyncCompletion failed: %v", err)
	}
	if got != "typed completion summary" {
		t.Fatalf("response = %q, want typed completion summary", got)
	}
	if provider.calls != 1 {
		t.Fatalf("provider calls = %d, want 1", provider.calls)
	}
	assertNoSyntheticAsyncCompletionInbound(t, msgBus)
}

func TestProcessAsyncCompletionMessageDoesNotSendDefaultFallback(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace: tmpDir,
				ModelName: "test-model",
				MaxTokens: 4096,
			},
		},
	}
	provider := &captureMessagesProvider{response: ""}
	msgBus := bus.NewMessageBus()
	al := NewAgentLoop(cfg, msgBus, provider)

	msg := bus.InboundMessage{
		Channel:  "system",
		ChatID:   "telegram:chat-1",
		SenderID: "async:spawn",
		Context: bus.InboundContext{
			Channel:  "system",
			ChatID:   "telegram:chat-1",
			ChatType: "direct",
			SenderID: "async:spawn",
			Raw: systemFollowUpAsyncCompletionRaw(&bus.InboundContext{
				Channel:  "telegram",
				ChatID:   "chat-1",
				ChatType: "direct",
			}, "telegram", "chat-1", "completion-empty"),
		},
		Content: asyncCompletionPrompt("spawn", "fresh background result"),
	}

	got, err := al.processSystemMessage(context.Background(), msg)
	if err != nil {
		t.Fatalf("processSystemMessage failed: %v", err)
	}
	if got != "" {
		t.Fatalf("response = %q, want empty", got)
	}
	select {
	case outbound := <-msgBus.OutboundChan():
		t.Fatalf("unexpected outbound message: %#v", outbound)
	default:
	}
}

package agent

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	taskregistry "github.com/sipeed/picoclaw/pkg/tasks"
	"github.com/sipeed/picoclaw/pkg/tools"
)

func TestDecideAsyncToolResultDelivery(t *testing.T) {
	tests := []struct {
		name string
		in   *tools.ToolResult
		want AsyncDeliveryDecision
	}{
		{
			name: "nil defaults without routing",
			want: AsyncDeliveryDecision{
				DeliveryMode: tools.AsyncDeliveryUserAndParent,
			},
		},
		{
			name: "default routes user text and parent content",
			in: &tools.ToolResult{
				ForLLM:      "parent text",
				ForUser:     "user text",
				AsyncTaskID: "subagent-9",
			},
			want: AsyncDeliveryDecision{
				TaskID:        "subagent-9",
				DeliveryMode:  tools.AsyncDeliveryUserAndParent,
				PublishToUser: true,
				QueueParent:   true,
				ContentLen:    len("parent text"),
				ForUserLen:    len("user text"),
			},
		},
		{
			name: "user only suppresses parent",
			in: (&tools.ToolResult{
				ForLLM:  "parent text",
				ForUser: "user text",
			}).WithAsyncDelivery(tools.AsyncDeliveryUserOnly),
			want: AsyncDeliveryDecision{
				DeliveryMode:  tools.AsyncDeliveryUserOnly,
				PublishToUser: true,
				QueueParent:   false,
				ParentHandled: true,
				ContentLen:    len("parent text"),
				ForUserLen:    len("user text"),
			},
		},
		{
			name: "parent only suppresses user",
			in: (&tools.ToolResult{
				ForLLM:  "parent text",
				ForUser: "user text",
			}).WithAsyncDelivery(tools.AsyncDeliveryParentOnly),
			want: AsyncDeliveryDecision{
				DeliveryMode:  tools.AsyncDeliveryParentOnly,
				PublishToUser: false,
				QueueParent:   true,
				ContentLen:    len("parent text"),
				ForUserLen:    len("user text"),
			},
		},
		{
			name: "silent suppresses user but not parent",
			in: (&tools.ToolResult{
				ForLLM:  "parent text",
				ForUser: "user text",
				Silent:  true,
			}).WithAsyncDelivery(tools.AsyncDeliveryUserAndParent),
			want: AsyncDeliveryDecision{
				DeliveryMode:  tools.AsyncDeliveryUserAndParent,
				PublishToUser: false,
				QueueParent:   true,
				ContentLen:    len("parent text"),
				ForUserLen:    len("user text"),
			},
		},
		{
			name: "media counts direct and completion media",
			in: (&tools.ToolResult{
				ForLLM: "parent text",
				Media:  []string{"media://direct"},
				Completion: &tools.CompletionResult{
					Media: []tools.CompletionMedia{
						{Ref: "media://completion-1"},
						{Ref: "media://completion-2"},
					},
				},
			}).WithAsyncDelivery(tools.AsyncDeliveryParentOnly),
			want: AsyncDeliveryDecision{
				DeliveryMode: tools.AsyncDeliveryParentOnly,
				QueueParent:  true,
				ContentLen:   -1,
				MediaCount:   3,
			},
		},
		{
			name: "user only media publishes without user text",
			in: (&tools.ToolResult{
				ForLLM: "internal media result",
				Completion: &tools.CompletionResult{
					Media: []tools.CompletionMedia{{Ref: "media://completion-video"}},
				},
			}).WithAsyncDelivery(tools.AsyncDeliveryUserOnly),
			want: AsyncDeliveryDecision{
				DeliveryMode:  tools.AsyncDeliveryUserOnly,
				PublishToUser: true,
				QueueParent:   false,
				ParentHandled: true,
				ContentLen:    -1,
				ForUserLen:    0,
				MediaCount:    1,
			},
		},
		{
			name: "error is surfaced in decision",
			in: (&tools.ToolResult{
				ForLLM:  "failed",
				ForUser: "failed",
				IsError: true,
			}).WithAsyncDelivery(tools.AsyncDeliveryUserOnly),
			want: AsyncDeliveryDecision{
				DeliveryMode:  tools.AsyncDeliveryUserOnly,
				PublishToUser: true,
				ContentLen:    len("failed"),
				ForUserLen:    len("failed"),
				IsError:       true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.want.ContentLen == -1 {
				tt.want.ContentLen = len(tt.in.ContentForLLM())
			}
			got := decideAsyncToolResultDelivery(tt.in)
			if got != tt.want {
				t.Fatalf("decision = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestDeliverAsyncToolCompletion_UserOnlyUpdatesDelivered(t *testing.T) {
	al, msgBus, ts, workspace := newDeliveryCoordinatorTestRuntime(t, "ok")
	taskID := "coordinator-user-only"
	upsertAsyncTaskForTest(t, al, workspace, taskID)
	result := (&tools.ToolResult{
		ForLLM:      "internal",
		ForUser:     "user done",
		AsyncTaskID: taskID,
	}).WithAsyncDelivery(tools.AsyncDeliveryUserOnly)

	al.deliverAsyncToolCompletion(AsyncDeliveryRequest{
		TurnState:    ts,
		ToolName:     "spawn",
		CompletionID: "completion-user-only",
		Result:       result,
		Decision:     decideAsyncToolResultDelivery(result),
	})

	outbound := waitForOutboundMessage(t, msgBus.OutboundChan(), 2*time.Second, func(msg bus.OutboundMessage) bool {
		return msg.Content == "user done"
	})
	if outbound.Context.TopicID != "topic-1" {
		t.Fatalf("TopicID = %q, want topic-1", outbound.Context.TopicID)
	}
	assertTaskDeliveryStatusForTest(t, al, workspace, taskID, taskregistry.DeliveryDelivered)
	rec, _ := al.taskRegistryForWorkspace(workspace).Get(taskID)
	if rec.LastCompletionID != "completion-user-only" {
		t.Fatalf("LastCompletionID = %q, want completion-user-only", rec.LastCompletionID)
	}
	if rec.DeliveredAt == 0 {
		t.Fatal("DeliveredAt was not set")
	}
	events := al.taskRegistryForWorkspace(workspace).ListEvents(taskID)
	assertTaskEventForTest(t, events, taskregistry.EventTaskDeliveryDecision, map[string]string{
		"completion_id": "completion-user-only",
		"source_tool":   "spawn",
		"mode":          string(tools.AsyncDeliveryUserOnly),
		"will_user":     "true",
		"will_parent":   "false",
	})
}

func TestDeliverAsyncToolCompletion_ParentOnlyUpdatesSessionQueued(t *testing.T) {
	al, msgBus, ts, workspace := newDeliveryCoordinatorTestRuntime(t, "parent synthesized")
	taskID := "coordinator-parent-only"
	upsertAsyncTaskForTest(t, al, workspace, taskID)
	result := (&tools.ToolResult{
		ForLLM:      "parent data",
		ForUser:     "do not send",
		AsyncTaskID: taskID,
	}).WithAsyncDelivery(tools.AsyncDeliveryParentOnly)

	al.deliverAsyncToolCompletion(AsyncDeliveryRequest{
		TurnState:    ts,
		ToolName:     "delegate",
		CompletionID: "completion-parent-only",
		Result:       result,
		Decision:     decideAsyncToolResultDelivery(result),
	})

	waitForOutboundMessage(t, msgBus.OutboundChan(), 2*time.Second, func(msg bus.OutboundMessage) bool {
		return msg.Content == "parent synthesized"
	})
	assertTaskDeliveryStatusForTest(t, al, workspace, taskID, taskregistry.DeliverySessionQueued)
	assertNoSyntheticAsyncCompletionInbound(t, msgBus)
}

func TestDeliverAsyncToolCompletion_UserAndParentDeliversBothOnce(t *testing.T) {
	al, msgBus, ts, workspace := newDeliveryCoordinatorTestRuntime(t, "parent synthesized")
	taskID := "coordinator-user-and-parent"
	upsertAsyncTaskForTest(t, al, workspace, taskID)
	result := (&tools.ToolResult{
		ForLLM:      "parent data",
		ForUser:     "user visible",
		AsyncTaskID: taskID,
	}).WithAsyncDelivery(tools.AsyncDeliveryUserAndParent)
	req := AsyncDeliveryRequest{
		TurnState:    ts,
		ToolName:     "spawn",
		CompletionID: "same-user-and-parent-completion",
		Result:       result,
		Decision:     decideAsyncToolResultDelivery(result),
	}

	al.deliverAsyncToolCompletion(req)
	waitForOutboundMessage(t, msgBus.OutboundChan(), 2*time.Second, func(msg bus.OutboundMessage) bool {
		return msg.Content == "user visible"
	})
	waitForOutboundMessage(t, msgBus.OutboundChan(), 2*time.Second, func(msg bus.OutboundMessage) bool {
		return msg.Content == "parent synthesized"
	})
	assertTaskDeliveryStatusForTest(t, al, workspace, taskID, taskregistry.DeliveryDelivered)
	assertNoSyntheticAsyncCompletionInbound(t, msgBus)

	reloaded, reloadedBus, reloadedTS, _ := newDeliveryCoordinatorTestRuntimeWithWorkspace(
		t,
		workspace,
		"parent duplicate",
	)
	req.TurnState = reloadedTS
	reloaded.deliverAsyncToolCompletion(req)
	assertNoOutboundMessage(t, reloadedBus, "duplicate user_and_parent delivery")
	assertTaskDeliveryStatusForTest(t, reloaded, workspace, taskID, taskregistry.DeliveryDelivered)
}

func TestDeliverAsyncToolCompletion_SkipsDuplicateUserDelivery(t *testing.T) {
	al, msgBus, ts, workspace := newDeliveryCoordinatorTestRuntime(t, "ok")
	taskID := "coordinator-duplicate-user"
	upsertAsyncTaskForTest(t, al, workspace, taskID)
	result := (&tools.ToolResult{
		ForLLM:      "internal",
		ForUser:     "user once",
		AsyncTaskID: taskID,
	}).WithAsyncDelivery(tools.AsyncDeliveryUserOnly)
	req := AsyncDeliveryRequest{
		TurnState:    ts,
		ToolName:     "spawn",
		CompletionID: "same-completion",
		Result:       result,
		Decision:     decideAsyncToolResultDelivery(result),
	}

	al.deliverAsyncToolCompletion(req)
	waitForOutboundMessage(t, msgBus.OutboundChan(), 2*time.Second, func(msg bus.OutboundMessage) bool {
		return msg.Content == "user once"
	})
	al.deliverAsyncToolCompletion(req)
	assertNoOutboundMessage(t, msgBus, "duplicate user delivery")
}

func TestDeliverAsyncToolCompletion_SkipsDuplicateParentDeliveryAfterReload(t *testing.T) {
	al, msgBus, ts, workspace := newDeliveryCoordinatorTestRuntime(t, "parent once")
	taskID := "coordinator-duplicate-parent"
	upsertAsyncTaskForTest(t, al, workspace, taskID)
	result := (&tools.ToolResult{
		ForLLM:      "parent data",
		ForUser:     "do not send",
		AsyncTaskID: taskID,
	}).WithAsyncDelivery(tools.AsyncDeliveryParentOnly)
	req := AsyncDeliveryRequest{
		TurnState:    ts,
		ToolName:     "delegate",
		CompletionID: "same-parent-completion",
		Result:       result,
		Decision:     decideAsyncToolResultDelivery(result),
	}

	al.deliverAsyncToolCompletion(req)
	waitForOutboundMessage(t, msgBus.OutboundChan(), 2*time.Second, func(msg bus.OutboundMessage) bool {
		return msg.Content == "parent once"
	})

	reloaded, reloadedBus, reloadedTS, _ := newDeliveryCoordinatorTestRuntimeWithWorkspace(
		t,
		workspace,
		"parent duplicate",
	)
	req.TurnState = reloadedTS
	reloaded.deliverAsyncToolCompletion(req)
	assertNoOutboundMessage(t, reloadedBus, "duplicate parent delivery")
	assertTaskDeliveryStatusForTest(t, reloaded, workspace, taskID, taskregistry.DeliverySessionQueued)
}

func TestDeliverAsyncToolCompletion_SkipsDuplicateMediaAfterReload(t *testing.T) {
	al, msgBus, ts, workspace := newDeliveryCoordinatorTestRuntime(t, "ok")
	taskID := "coordinator-duplicate-media"
	upsertAsyncTaskForTest(t, al, workspace, taskID)
	result := (&tools.ToolResult{
		ForLLM:      "internal media result",
		AsyncTaskID: taskID,
		Completion: &tools.CompletionResult{
			Text: "https://example.com/reel",
			Media: []tools.CompletionMedia{{
				Ref:         "media://video-1",
				Type:        "video",
				Filename:    "source.mp4",
				ContentType: "video/mp4",
			}},
		},
	}).WithAsyncDelivery(tools.AsyncDeliveryUserOnly)
	req := AsyncDeliveryRequest{
		TurnState:    ts,
		ToolName:     "spawn",
		CompletionID: "same-media-completion",
		Result:       result,
		Decision:     decideAsyncToolResultDelivery(result),
	}

	al.deliverAsyncToolCompletion(req)
	media := waitForOutboundMediaMessage(t, msgBus.OutboundMediaChan(), 2*time.Second)
	if len(media.Parts) != 1 || media.Parts[0].Ref != "media://video-1" {
		t.Fatalf("media parts = %+v, want media://video-1", media.Parts)
	}
	assertTaskDeliveryStatusForTest(t, al, workspace, taskID, taskregistry.DeliveryDelivered)

	reloaded, reloadedBus, reloadedTS, _ := newDeliveryCoordinatorTestRuntimeWithWorkspace(
		t,
		workspace,
		"duplicate should not synthesize",
	)
	req.TurnState = reloadedTS
	reloaded.deliverAsyncToolCompletion(req)
	assertNoOutboundMediaMessage(t, reloadedBus, "duplicate media delivery after reload")
	assertNoOutboundMessage(t, reloadedBus, "duplicate media parent synthesis after reload")
	assertTaskDeliveryStatusForTest(t, reloaded, workspace, taskID, taskregistry.DeliveryDelivered)
}

func TestDeliverAsyncToolCompletion_MediaDeliveryFailureRecordsFailed(t *testing.T) {
	al, _, ts, workspace := newDeliveryCoordinatorTestRuntime(t, "ok")
	taskID := "coordinator-media-failed"
	upsertAsyncTaskForTest(t, al, workspace, taskID)
	result := (&tools.ToolResult{
		ForLLM:      "internal media result",
		AsyncTaskID: taskID,
		Completion: &tools.CompletionResult{
			Media: []tools.CompletionMedia{{
				Ref:         "media://video-fail",
				Type:        "video",
				Filename:    "source.mp4",
				ContentType: "video/mp4",
			}},
		},
	}).WithAsyncDelivery(tools.AsyncDeliveryUserOnly)

	al.bus = failingMessageBus{}
	al.deliverAsyncToolCompletion(AsyncDeliveryRequest{
		TurnState:    ts,
		ToolName:     "spawn",
		CompletionID: "failed-media-completion",
		Result:       result,
		Decision:     decideAsyncToolResultDelivery(result),
	})

	rec, ok := al.taskRegistryForWorkspace(workspace).Get(taskID)
	if !ok {
		t.Fatal("expected task")
	}
	if rec.DeliveryStatus != taskregistry.DeliveryFailed {
		t.Fatalf("DeliveryStatus = %q, want failed", rec.DeliveryStatus)
	}
	if rec.LastCompletionID != "failed-media-completion" {
		t.Fatalf("LastCompletionID = %q, want failed-media-completion", rec.LastCompletionID)
	}
	if !strings.Contains(rec.DeliveryError, "publish failed") {
		t.Fatalf("DeliveryError = %q, want publish failed", rec.DeliveryError)
	}
}

func newDeliveryCoordinatorTestRuntime(
	t *testing.T,
	response string,
) (*AgentLoop, *bus.MessageBus, *turnState, string) {
	t.Helper()
	workspace := t.TempDir()
	return newDeliveryCoordinatorTestRuntimeWithWorkspace(t, workspace, response)
}

func newDeliveryCoordinatorTestRuntimeWithWorkspace(
	t *testing.T,
	workspace string,
	response string,
) (*AgentLoop, *bus.MessageBus, *turnState, string) {
	t.Helper()
	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace: workspace,
				ModelName: "test-model",
				MaxTokens: 4096,
			},
		},
	}
	msgBus := bus.NewMessageBus()
	al := NewAgentLoop(cfg, msgBus, &simpleMockProvider{response: response})
	agent := al.registry.GetDefaultAgent()
	if agent == nil {
		t.Fatal("expected default agent")
	}
	inbound := &bus.InboundContext{
		Channel:  "telegram",
		ChatID:   "chat-1",
		ChatType: "direct",
		TopicID:  "topic-1",
		SenderID: "user-1",
	}
	ts := &turnState{
		agent:      agent,
		agentID:    agent.ID,
		workspace:  workspace,
		channel:    "telegram",
		chatID:     "chat-1",
		sessionKey: "session-1",
		opts: processOptions{
			Dispatch: DispatchRequest{
				SessionKey:     "session-1",
				InboundContext: inbound,
			},
		},
		scope: al.newTurnEventScope(agent.ID, "session-1", &TurnContext{Inbound: inbound}),
	}
	return al, msgBus, ts, workspace
}

func assertNoOutboundMessage(t *testing.T, msgBus *bus.MessageBus, context string) {
	t.Helper()
	select {
	case msg := <-msgBus.OutboundChan():
		t.Fatalf("unexpected outbound during %s: %+v", context, msg)
	case <-time.After(150 * time.Millisecond):
	}
}

func waitForOutboundMediaMessage(
	t *testing.T,
	ch <-chan bus.OutboundMediaMessage,
	timeout time.Duration,
) bus.OutboundMediaMessage {
	t.Helper()
	select {
	case msg := <-ch:
		return msg
	case <-time.After(timeout):
		t.Fatal("timeout waiting for outbound media message")
	}
	return bus.OutboundMediaMessage{}
}

func assertNoOutboundMediaMessage(t *testing.T, msgBus *bus.MessageBus, context string) {
	t.Helper()
	select {
	case msg := <-msgBus.OutboundMediaChan():
		t.Fatalf("unexpected outbound media during %s: %+v", context, msg)
	case <-time.After(150 * time.Millisecond):
	}
}

func assertTaskEventForTest(
	t *testing.T,
	events []taskregistry.TaskEvent,
	eventType taskregistry.EventType,
	payload map[string]string,
) {
	t.Helper()
	for _, evt := range events {
		if evt.Type != eventType {
			continue
		}
		for key, want := range payload {
			if got := evt.Payload[key]; got != want {
				t.Fatalf("event %s payload[%s] = %q, want %q; event=%+v", eventType, key, got, want, evt)
			}
		}
		return
	}
	t.Fatalf("event %s not found in %+v", eventType, events)
}

type failingMessageBus struct{}

func (failingMessageBus) PublishInbound(context.Context, bus.InboundMessage) error {
	return errors.New("publish failed")
}

func (failingMessageBus) PublishObserved(context.Context, bus.ObservedMessage) error {
	return errors.New("publish failed")
}

func (failingMessageBus) PublishOutbound(context.Context, bus.OutboundMessage) error {
	return errors.New("publish failed")
}

func (failingMessageBus) PublishOutboundMedia(context.Context, bus.OutboundMediaMessage) error {
	return errors.New("publish failed")
}

func (failingMessageBus) GetStreamer(context.Context, string, string, string) (bus.Streamer, bool) {
	return nil, false
}

func (failingMessageBus) InboundChan() <-chan bus.InboundMessage {
	return nil
}

func (failingMessageBus) ObservedChan() <-chan bus.ObservedMessage {
	return nil
}

func TestDeliverAsyncToolCompletion_FailedDeliveryRecordsCompletionError(t *testing.T) {
	al, _, ts, workspace := newDeliveryCoordinatorTestRuntime(t, "ok")
	taskID := "coordinator-failed"
	upsertAsyncTaskForTest(t, al, workspace, taskID)
	result := (&tools.ToolResult{
		ForLLM:      "internal",
		ForUser:     "user fail",
		AsyncTaskID: taskID,
	}).WithAsyncDelivery(tools.AsyncDeliveryUserOnly)

	al.bus = failingMessageBus{}
	al.deliverAsyncToolCompletion(AsyncDeliveryRequest{
		TurnState:    ts,
		ToolName:     "spawn",
		CompletionID: "failed-completion",
		Result:       result,
		Decision:     decideAsyncToolResultDelivery(result),
	})

	rec, ok := al.taskRegistryForWorkspace(workspace).Get(taskID)
	if !ok {
		t.Fatal("expected task")
	}
	if rec.DeliveryStatus != taskregistry.DeliveryFailed {
		t.Fatalf("DeliveryStatus = %q, want failed", rec.DeliveryStatus)
	}
	if rec.LastCompletionID != "failed-completion" {
		t.Fatalf("LastCompletionID = %q, want failed-completion", rec.LastCompletionID)
	}
	if !strings.Contains(rec.DeliveryError, "publish failed") {
		t.Fatalf("DeliveryError = %q, want publish failed", rec.DeliveryError)
	}
}

func TestDeliverAsyncToolCompletion_ErrorDeliveryUpdatesTaskStatus(t *testing.T) {
	al, msgBus, ts, workspace := newDeliveryCoordinatorTestRuntime(t, "ok")
	taskID := "coordinator-error-delivered"
	upsertAsyncTaskForTest(t, al, workspace, taskID)
	result := (&tools.ToolResult{
		ForLLM:      "internal error",
		ForUser:     "user error",
		AsyncTaskID: taskID,
		IsError:     true,
	}).WithAsyncDelivery(tools.AsyncDeliveryUserOnly)

	al.deliverAsyncToolCompletion(AsyncDeliveryRequest{
		TurnState:    ts,
		ToolName:     "spawn",
		CompletionID: "error-completion",
		Result:       result,
		Decision:     decideAsyncToolResultDelivery(result),
	})

	waitForOutboundMessage(t, msgBus.OutboundChan(), 2*time.Second, func(msg bus.OutboundMessage) bool {
		return msg.Content == "user error"
	})
	assertTaskDeliveryStatusForTest(t, al, workspace, taskID, taskregistry.DeliveryDelivered)
	rec, _ := al.taskRegistryForWorkspace(workspace).Get(taskID)
	if rec.LastCompletionID != "error-completion" {
		t.Fatalf("LastCompletionID = %q, want error-completion", rec.LastCompletionID)
	}
}

package agent

import (
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

func newDeliveryCoordinatorTestRuntime(
	t *testing.T,
	response string,
) (*AgentLoop, *bus.MessageBus, *turnState, string) {
	t.Helper()
	workspace := t.TempDir()
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

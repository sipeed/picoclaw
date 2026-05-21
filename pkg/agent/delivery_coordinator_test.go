package agent

import (
	"testing"

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

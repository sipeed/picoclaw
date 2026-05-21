package tools

import (
	"context"
	"strings"
	"testing"
	"time"

	taskregistry "github.com/sipeed/picoclaw/pkg/tasks"
)

func TestTaskStatusTool_ListsVisibleRecords(t *testing.T) {
	registry := taskregistry.NewRegistry(taskregistry.WorkspaceStorePath(t.TempDir()))
	now := time.Now().UnixMilli()
	if err := registry.Upsert(taskregistry.Record{
		TaskID:         "delegate-1",
		Runtime:        taskregistry.RuntimeDelegate,
		TaskKind:       "delegate",
		Channel:        "telegram",
		ChatID:         "chat-1",
		TopicID:        "topic-1",
		AgentID:        "media",
		Task:           "download reel",
		Status:         taskregistry.StatusSucceeded,
		DeliveryStatus: taskregistry.DeliverySessionQueued,
		DeliveryMode:   string(AsyncDeliveryParentOnly),
		CreatedAt:      now,
		StartedAt:      now,
		EndedAt:        now,
	}); err != nil {
		t.Fatalf("Upsert(delegate) error = %v", err)
	}
	if err := registry.Upsert(taskregistry.Record{
		TaskID:         "delegate-other",
		Runtime:        taskregistry.RuntimeDelegate,
		TaskKind:       "delegate",
		Channel:        "telegram",
		ChatID:         "chat-2",
		Task:           "other chat",
		Status:         taskregistry.StatusSucceeded,
		DeliveryStatus: taskregistry.DeliverySessionQueued,
		CreatedAt:      now,
	}); err != nil {
		t.Fatalf("Upsert(other) error = %v", err)
	}

	tool := NewTaskStatusTool(registry)
	result := tool.Execute(WithToolContext(context.Background(), "telegram", "chat-1"), nil)

	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.ForLLM)
	}
	for _, want := range []string{
		"Task status report (1 total)",
		"delegate-1",
		"delegate/delegate",
		"Agent: media",
		"Scope: telegram/chat-1 topic=topic-1",
	} {
		if !strings.Contains(result.ForLLM, want) {
			t.Fatalf("result missing %q:\n%s", want, result.ForLLM)
		}
	}
	if strings.Contains(result.ForLLM, "delegate-other") {
		t.Fatalf("result leaked other chat task:\n%s", result.ForLLM)
	}
}

func TestTaskStatusTool_TaskID(t *testing.T) {
	registry := taskregistry.NewRegistry(taskregistry.WorkspaceStorePath(t.TempDir()))
	if err := registry.Upsert(taskregistry.Record{
		TaskID:         "subagent-1",
		Runtime:        taskregistry.RuntimeSubagent,
		TaskKind:       "spawn",
		Task:           "background work",
		Status:         taskregistry.StatusRunning,
		DeliveryStatus: taskregistry.DeliveryPending,
	}); err != nil {
		t.Fatalf("Upsert(subagent) error = %v", err)
	}

	tool := NewTaskStatusTool(registry)
	result := tool.Execute(context.Background(), map[string]any{"task_id": "subagent-1"})

	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "Task subagent-1 [subagent/spawn]") {
		t.Fatalf("unexpected result:\n%s", result.ForLLM)
	}
}

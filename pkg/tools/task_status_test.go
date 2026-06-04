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
		BoardID:        "board-1",
		ParentTaskID:   "root-1",
		StepID:         "download",
		StepTitle:      "Download media",
		Owner:          "media",
		DependsOn:      []string{"root-1"},
		BlockedBy:      []string{"caption"},
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
		Deliverable: &taskregistry.DeliverablePayload{
			Text: "video downloaded",
			Artifacts: []taskregistry.DeliverableItem{{
				Ref:  "media://video",
				Kind: "video",
			}},
		},
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
		"Board: board_id=board-1 parent=root-1 owner=media",
		"Step: Download media (download)",
		"Depends on: root-1",
		"Blocked by: caption",
		"Deliverable: text=true artifacts=1 report=true",
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

func TestTaskStatusTool_TaskIDIncludesEvents(t *testing.T) {
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
	if err := registry.Update("subagent-1", func(rec *taskregistry.Record) {
		rec.Status = taskregistry.StatusSucceeded
		rec.DeliveryStatus = taskregistry.DeliveryDelivered
	}); err != nil {
		t.Fatalf("Update(subagent) error = %v", err)
	}

	tool := NewTaskStatusTool(registry)
	result := tool.Execute(context.Background(), map[string]any{
		"task_id":        "subagent-1",
		"include_events": true,
	})

	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.ForLLM)
	}
	for _, want := range []string{
		"Events:",
		"#1 task.upserted",
		"#2 task.status_changed",
		"#3 task.delivery_changed",
		"payload={from=\"running\", to=\"succeeded\"}",
	} {
		if !strings.Contains(result.ForLLM, want) {
			t.Fatalf("result missing %q:\n%s", want, result.ForLLM)
		}
	}
}

func TestTaskStatusTool_ListsSpawnAndDelegateRecords(t *testing.T) {
	registry := taskregistry.NewRegistry(taskregistry.WorkspaceStorePath(t.TempDir()))
	now := time.Now().UnixMilli()
	records := []taskregistry.Record{
		{
			TaskID:         "subagent-1",
			Runtime:        taskregistry.RuntimeSubagent,
			TaskKind:       "spawn",
			Channel:        "telegram",
			ChatID:         "chat-1",
			TopicID:        "topic-1",
			AgentID:        "research",
			Task:           "background research",
			Status:         taskregistry.StatusRunning,
			DeliveryStatus: taskregistry.DeliveryPending,
			CreatedAt:      now,
		},
		{
			TaskID:         "delegate-1",
			Runtime:        taskregistry.RuntimeDelegate,
			TaskKind:       "delegate",
			Channel:        "telegram",
			ChatID:         "chat-1",
			TopicID:        "topic-1",
			AgentID:        "media",
			Task:           "download media",
			Status:         taskregistry.StatusSucceeded,
			DeliveryStatus: taskregistry.DeliverySessionQueued,
			CreatedAt:      now + 1,
		},
	}
	for _, rec := range records {
		if err := registry.Upsert(rec); err != nil {
			t.Fatalf("Upsert(%s) error = %v", rec.TaskID, err)
		}
	}

	tool := NewTaskStatusTool(registry)
	ctx := WithToolTopicID(WithToolContext(context.Background(), "telegram", "chat-1"), "topic-1")
	result := tool.Execute(ctx, nil)

	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.ForLLM)
	}
	for _, want := range []string{
		"Task status report (2 total)",
		"subagent-1",
		"subagent/spawn",
		"delegate-1",
		"delegate/delegate",
	} {
		if !strings.Contains(result.ForLLM, want) {
			t.Fatalf("result missing %q:\n%s", want, result.ForLLM)
		}
	}
}

func TestTaskStatusTool_TaskKindFilter(t *testing.T) {
	registry := taskregistry.NewRegistry(taskregistry.WorkspaceStorePath(t.TempDir()))
	for _, rec := range []taskregistry.Record{
		{
			TaskID:         "subagent-1",
			Runtime:        taskregistry.RuntimeSubagent,
			TaskKind:       "spawn",
			Channel:        "telegram",
			ChatID:         "chat-1",
			Status:         taskregistry.StatusRunning,
			DeliveryStatus: taskregistry.DeliveryPending,
		},
		{
			TaskID:         "delegate-1",
			Runtime:        taskregistry.RuntimeDelegate,
			TaskKind:       "delegate",
			Channel:        "telegram",
			ChatID:         "chat-1",
			Status:         taskregistry.StatusSucceeded,
			DeliveryStatus: taskregistry.DeliverySessionQueued,
		},
	} {
		if err := registry.Upsert(rec); err != nil {
			t.Fatalf("Upsert(%s) error = %v", rec.TaskID, err)
		}
	}

	tool := NewTaskStatusTool(registry)
	result := tool.Execute(WithToolContext(context.Background(), "telegram", "chat-1"), map[string]any{
		"task_kind": "delegate",
	})

	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "delegate-1") {
		t.Fatalf("expected delegate record:\n%s", result.ForLLM)
	}
	if strings.Contains(result.ForLLM, "subagent-1") {
		t.Fatalf("spawn record leaked through delegate filter:\n%s", result.ForLLM)
	}
}

func TestTaskStatusTool_BoardIDFilter(t *testing.T) {
	registry := taskregistry.NewRegistry(taskregistry.WorkspaceStorePath(t.TempDir()))
	for _, rec := range []taskregistry.Record{
		{
			TaskID:         "board-a-step-1",
			Runtime:        taskregistry.RuntimeDelegate,
			TaskKind:       "delegate",
			BoardID:        "board-a",
			Task:           "step 1",
			Status:         taskregistry.StatusSucceeded,
			DeliveryStatus: taskregistry.DeliverySessionQueued,
		},
		{
			TaskID:         "board-b-step-1",
			Runtime:        taskregistry.RuntimeDelegate,
			TaskKind:       "delegate",
			BoardID:        "board-b",
			Task:           "step 1",
			Status:         taskregistry.StatusSucceeded,
			DeliveryStatus: taskregistry.DeliverySessionQueued,
		},
	} {
		if err := registry.Upsert(rec); err != nil {
			t.Fatalf("Upsert(%s) error = %v", rec.TaskID, err)
		}
	}

	tool := NewTaskStatusTool(registry)
	result := tool.Execute(context.Background(), map[string]any{"board_id": "board-a"})

	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "board-a-step-1") {
		t.Fatalf("expected board-a record:\n%s", result.ForLLM)
	}
	if strings.Contains(result.ForLLM, "board-b-step-1") {
		t.Fatalf("board-b record leaked through board filter:\n%s", result.ForLLM)
	}
}

func TestTaskStatusTool_TopicScoping(t *testing.T) {
	registry := taskregistry.NewRegistry(taskregistry.WorkspaceStorePath(t.TempDir()))
	for _, rec := range []taskregistry.Record{
		{
			TaskID:         "delegate-topic-1",
			Runtime:        taskregistry.RuntimeDelegate,
			TaskKind:       "delegate",
			Channel:        "telegram",
			ChatID:         "chat-1",
			TopicID:        "topic-1",
			Status:         taskregistry.StatusSucceeded,
			DeliveryStatus: taskregistry.DeliverySessionQueued,
		},
		{
			TaskID:         "delegate-topic-2",
			Runtime:        taskregistry.RuntimeDelegate,
			TaskKind:       "delegate",
			Channel:        "telegram",
			ChatID:         "chat-1",
			TopicID:        "topic-2",
			Status:         taskregistry.StatusSucceeded,
			DeliveryStatus: taskregistry.DeliverySessionQueued,
		},
	} {
		if err := registry.Upsert(rec); err != nil {
			t.Fatalf("Upsert(%s) error = %v", rec.TaskID, err)
		}
	}

	tool := NewTaskStatusTool(registry)
	ctx := WithToolTopicID(WithToolContext(context.Background(), "telegram", "chat-1"), "topic-1")
	result := tool.Execute(ctx, nil)

	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "delegate-topic-1") {
		t.Fatalf("expected topic-1 record:\n%s", result.ForLLM)
	}
	if strings.Contains(result.ForLLM, "delegate-topic-2") {
		t.Fatalf("topic-2 record leaked into topic-1 status:\n%s", result.ForLLM)
	}
}

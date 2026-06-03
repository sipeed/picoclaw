package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	taskregistry "github.com/sipeed/picoclaw/pkg/tasks"
)

func TestTaskBoardTool_CreateAddStepAndList(t *testing.T) {
	registry := taskregistry.NewRegistry(taskregistry.WorkspaceStorePath(t.TempDir()))
	tool := NewTaskBoardTool(registry)
	ctx := WithToolTopicID(WithToolContext(context.Background(), "telegram", "chat-1"), "topic-1")
	ctx = WithToolSessionContext(ctx, "main", "session-1", nil)

	create := tool.Execute(ctx, map[string]any{
		"action":      "create",
		"board_id":    "workflow-1",
		"title":       "Instagram recipe workflow",
		"description": "Download, extract, polish",
	})
	if create.IsError {
		t.Fatalf("create failed: %s", create.ForLLM)
	}

	add := tool.Execute(ctx, map[string]any{
		"action":     "add_step",
		"board_id":   "workflow-1",
		"step_id":    "polish-translation",
		"step_title": "Polish translation",
		"owner":      "deep-research",
		"task":       "Improve the Russian recipe text",
		"depends_on": []any{"media-extract"},
	})
	if add.IsError {
		t.Fatalf("add_step failed: %s", add.ForLLM)
	}

	records := registry.ListBoard("workflow-1")
	if len(records) != 2 {
		t.Fatalf("ListBoard count = %d, want 2: %+v", len(records), records)
	}
	if records[0].Status != taskregistry.StatusPlanned || records[1].Status != taskregistry.StatusPlanned {
		t.Fatalf("records should be planned: %+v", records)
	}

	list := tool.Execute(ctx, map[string]any{
		"action":   "list",
		"board_id": "workflow-1",
	})
	if list.IsError {
		t.Fatalf("list failed: %s", list.ForLLM)
	}
	var payload struct {
		BoardID string `json:"board_id"`
		Count   int    `json:"count"`
		Counts  map[string]int
		Steps   []struct {
			TaskID    string   `json:"task_id"`
			Status    string   `json:"status"`
			StepID    string   `json:"step_id"`
			DependsOn []string `json:"depends_on"`
		} `json:"steps"`
	}
	if err := json.Unmarshal([]byte(list.ForLLM), &payload); err != nil {
		t.Fatalf("list JSON error = %v\n%s", err, list.ForLLM)
	}
	if payload.BoardID != "workflow-1" || payload.Count != 2 || payload.Counts["planned"] != 2 {
		t.Fatalf("unexpected list payload: %+v", payload)
	}
	if payload.Steps[1].StepID != "polish-translation" || payload.Steps[1].DependsOn[0] != "media-extract" {
		t.Fatalf("unexpected step payload: %+v", payload.Steps)
	}
}

func TestTaskBoardTool_UpdateStepStatus(t *testing.T) {
	registry := taskregistry.NewRegistry(taskregistry.WorkspaceStorePath(t.TempDir()))
	tool := NewTaskBoardTool(registry)
	ctx := WithToolTopicID(WithToolContext(context.Background(), "telegram", "chat-1"), "topic-1")
	ctx = WithToolSessionContext(ctx, "coding", "session-1", nil)

	if result := tool.Execute(ctx, map[string]any{
		"action":   "create",
		"board_id": "workflow-1",
		"title":    "Architecture diagrams",
	}); result.IsError {
		t.Fatalf("create failed: %s", result.ForLLM)
	}
	if result := tool.Execute(ctx, map[string]any{
		"action":     "add_step",
		"board_id":   "workflow-1",
		"step_id":    "diagram-generation",
		"step_title": "Generate diagrams",
	}); result.IsError {
		t.Fatalf("add_step failed: %s", result.ForLLM)
	}

	update := tool.Execute(ctx, map[string]any{
		"action":   "update",
		"board_id": "workflow-1",
		"step_id":  "diagram-generation",
		"status":   "succeeded",
		"summary":  "Created 6 Excalidraw diagrams.",
	})
	if update.IsError {
		t.Fatalf("update failed: %s", update.ForLLM)
	}

	rec, ok := registry.Get("board:workflow-1:step:diagram-generation")
	if !ok {
		t.Fatal("updated step missing from registry")
	}
	if rec.Status != taskregistry.StatusSucceeded {
		t.Fatalf("status = %q, want succeeded", rec.Status)
	}
	if rec.DeliveryStatus != taskregistry.DeliveryNotApplicable {
		t.Fatalf("delivery = %q, want not_applicable", rec.DeliveryStatus)
	}
	if rec.StartedAt == 0 || rec.EndedAt == 0 || rec.LastEventAt == 0 {
		t.Fatalf("timestamps not populated: %+v", rec)
	}
	if rec.TerminalSummary != "Created 6 Excalidraw diagrams." {
		t.Fatalf("terminal summary = %q", rec.TerminalSummary)
	}

	list := tool.Execute(ctx, map[string]any{
		"action":   "list",
		"board_id": "workflow-1",
	})
	if list.IsError {
		t.Fatalf("list failed: %s", list.ForLLM)
	}
	if !strings.Contains(list.ForLLM, `"succeeded": 1`) ||
		!strings.Contains(list.ForLLM, `"summary": "Created 6 Excalidraw diagrams."`) {
		t.Fatalf("list did not show updated status/summary:\n%s", list.ForLLM)
	}
}

func TestTaskBoardTool_ResultsReturnsDeliverablesForVisibleBoard(t *testing.T) {
	registry := taskregistry.NewRegistry(taskregistry.WorkspaceStorePath(t.TempDir()))
	now := time.Now().UnixMilli()
	if err := registry.Upsert(taskregistry.Record{
		TaskID:         "board:workflow-1:step:media-extract",
		Runtime:        taskregistry.RuntimeTool,
		TaskKind:       "task_board_step",
		BoardID:        "workflow-1",
		StepID:         "media-extract",
		StepTitle:      "Extract media",
		Channel:        "telegram",
		ChatID:         "chat-1",
		Status:         taskregistry.StatusPlanned,
		DeliveryStatus: taskregistry.DeliveryNotApplicable,
		Task:           "planned media extract",
		CreatedAt:      now,
	}); err != nil {
		t.Fatalf("Upsert(planned) error = %v", err)
	}
	if err := registry.Upsert(taskregistry.Record{
		TaskID:         "delegate-1",
		Runtime:        taskregistry.RuntimeDelegate,
		TaskKind:       "delegate",
		BoardID:        "workflow-1",
		StepID:         "media-extract",
		StepTitle:      "Extract media",
		Channel:        "telegram",
		ChatID:         "chat-1",
		AgentID:        "media",
		Task:           "download reel",
		Status:         taskregistry.StatusSucceeded,
		DeliveryStatus: taskregistry.DeliverySessionQueued,
		CreatedAt:      now + 1,
		EndedAt:        now + 2,
		Deliverable: &taskregistry.DeliverablePayload{
			Text: "caption text",
			Artifacts: []taskregistry.DeliverableItem{{
				Ref:  "/tmp/video.mp4",
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
		BoardID:        "workflow-1",
		Channel:        "telegram",
		ChatID:         "chat-2",
		Task:           "other chat",
		Status:         taskregistry.StatusSucceeded,
		DeliveryStatus: taskregistry.DeliverySessionQueued,
		Deliverable:    &taskregistry.DeliverablePayload{Text: "secret"},
	}); err != nil {
		t.Fatalf("Upsert(other) error = %v", err)
	}

	tool := NewTaskBoardTool(registry)
	result := tool.Execute(WithToolContext(context.Background(), "telegram", "chat-1"), map[string]any{
		"action":   "results",
		"board_id": "workflow-1",
	})
	if result.IsError {
		t.Fatalf("results failed: %s", result.ForLLM)
	}
	if strings.Contains(result.ForLLM, "delegate-other") || strings.Contains(result.ForLLM, "secret") {
		t.Fatalf("results leaked other chat record:\n%s", result.ForLLM)
	}
	if strings.Contains(result.ForLLM, "board:workflow-1:step:media-extract") {
		t.Fatalf("results should not include planned placeholder without payload:\n%s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, `"task_id": "delegate-1"`) ||
		!strings.Contains(result.ForLLM, `"text": "caption text"`) ||
		!strings.Contains(result.ForLLM, `"artifacts"`) {
		t.Fatalf("results missing deliverable:\n%s", result.ForLLM)
	}
}

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
		BoardID string         `json:"board_id"`
		Count   int            `json:"count"`
		Counts  map[string]int `json:"counts"`
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

func TestTaskBoardTool_CreateWithTaskPacket(t *testing.T) {
	registry := taskregistry.NewRegistry(taskregistry.WorkspaceStorePath(t.TempDir()))
	tool := NewTaskBoardTool(registry)
	ctx := WithToolSessionContext(
		WithToolContext(context.Background(), "telegram", "chat-1"),
		"main",
		"session-1",
		nil,
	)

	create := tool.Execute(ctx, map[string]any{
		"action":   "create",
		"board_id": "workflow-media-1",
		"title":    "Instagram recipe workflow",
		"task_packet": map[string]any{
			"kind":      "media",
			"objective": "Download the reel and produce a Russian recipe.",
			"scope":     "One Instagram Reel URL.",
			"acceptance_criteria": []any{
				"video artifact is available",
				"recipe is translated to Russian",
			},
			"verification_plan": []any{
				"inspect media deliverable",
				"check final recipe text",
			},
			"resources": []any{
				map[string]any{
					"type":        "url",
					"uri":         "https://example.test/reel",
					"description": "source reel",
				},
			},
			"reporting": map[string]any{
				"audience": "user",
				"format":   "short",
			},
			"recovery": map[string]any{
				"retry_policy": "safe",
				"escalation":   "ask_user",
			},
			"media": map[string]any{
				"expected_artifacts": []any{"video", "caption"},
				"send_media":         true,
			},
		},
	})
	if create.IsError {
		t.Fatalf("create failed: %s", create.ForLLM)
	}

	rec, ok := registry.Get("board:workflow-media-1")
	if !ok {
		t.Fatal("board root missing")
	}
	if rec.TaskPacket == nil {
		t.Fatal("task packet missing from registry record")
	}
	if rec.TaskPacket.Kind != "media" {
		t.Fatalf("kind = %q, want media", rec.TaskPacket.Kind)
	}
	if rec.TaskPacket.Objective != "Download the reel and produce a Russian recipe." {
		t.Fatalf("objective = %q", rec.TaskPacket.Objective)
	}
	if len(rec.TaskPacket.AcceptanceCriteria) != 2 ||
		rec.TaskPacket.AcceptanceCriteria[0] != "video artifact is available" {
		t.Fatalf("acceptance criteria = %#v", rec.TaskPacket.AcceptanceCriteria)
	}
	if len(rec.TaskPacket.Resources) != 1 || rec.TaskPacket.Resources[0].URI != "https://example.test/reel" {
		t.Fatalf("resources = %#v", rec.TaskPacket.Resources)
	}
	if rec.TaskPacket.Media["send_media"] != true {
		t.Fatalf("media block = %#v", rec.TaskPacket.Media)
	}

	list := tool.Execute(ctx, map[string]any{
		"action":   "list",
		"board_id": "workflow-media-1",
	})
	if list.IsError {
		t.Fatalf("list failed: %s", list.ForLLM)
	}
	if !strings.Contains(list.ForLLM, `"task_packet"`) ||
		!strings.Contains(list.ForLLM, `"objective": "Download the reel and produce a Russian recipe."`) {
		t.Fatalf("list did not include task packet:\n%s", list.ForLLM)
	}
}

func TestTaskBoardTool_CreateRejectsTaskPacketWithoutObjective(t *testing.T) {
	registry := taskregistry.NewRegistry(taskregistry.WorkspaceStorePath(t.TempDir()))
	tool := NewTaskBoardTool(registry)

	result := tool.Execute(context.Background(), map[string]any{
		"action": "create",
		"title":  "Broken workflow",
		"task_packet": map[string]any{
			"kind":  "media",
			"scope": "missing objective",
		},
	})
	if !result.IsError {
		t.Fatalf("expected error, got: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "task_packet.objective required") {
		t.Fatalf("unexpected error: %s", result.ForLLM)
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
		TaskID:         "delegate-2",
		Runtime:        taskregistry.RuntimeDelegate,
		TaskKind:       "delegate",
		BoardID:        "workflow-1",
		StepID:         "media-extract",
		StepTitle:      "Extract media",
		Channel:        "telegram",
		ChatID:         "chat-1",
		AgentID:        "media",
		Task:           "download reel retry",
		Status:         taskregistry.StatusFailed,
		DeliveryStatus: taskregistry.DeliveryFailed,
		CreatedAt:      now + 3,
		EndedAt:        now + 4,
		Error:          "transient media backend failure",
	}); err != nil {
		t.Fatalf("Upsert(delegate failure) error = %v", err)
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

	var payload struct {
		StepResults []struct {
			StepID                 string                           `json:"step_id"`
			LatestTaskID           string                           `json:"latest_task_id"`
			LatestStatus           string                           `json:"latest_status"`
			LatestSuccessfulTaskID string                           `json:"latest_successful_task_id"`
			LatestFailureTaskID    string                           `json:"latest_failure_task_id"`
			LatestFailureStatus    string                           `json:"latest_failure_status"`
			LatestFailureError     string                           `json:"latest_failure_error"`
			HasResult              bool                             `json:"has_result"`
			Deliverable            *taskregistry.DeliverablePayload `json:"deliverable"`
			LatestSuccessful       *taskregistry.DeliverablePayload `json:"latest_successful_deliverable"`
			LegacyCompletion       *taskregistry.CompletionPayload  `json:"legacy_completion"`
		} `json:"step_results"`
	}
	if err := json.Unmarshal([]byte(result.ForLLM), &payload); err != nil {
		t.Fatalf("results JSON error = %v\n%s", err, result.ForLLM)
	}
	if len(payload.StepResults) != 1 {
		t.Fatalf("step_results len = %d, want 1: %+v\n%s", len(payload.StepResults), payload.StepResults, result.ForLLM)
	}
	step := payload.StepResults[0]
	if step.StepID != "media-extract" ||
		step.LatestTaskID != "delegate-2" ||
		step.LatestStatus != "failed" ||
		step.LatestSuccessfulTaskID != "delegate-1" ||
		step.LatestFailureTaskID != "delegate-2" ||
		step.LatestFailureStatus != "failed" ||
		step.LatestFailureError != "transient media backend failure" {
		t.Fatalf("unexpected step result metadata: %+v\n%s", step, result.ForLLM)
	}
	if step.HasResult || step.Deliverable != nil {
		t.Fatalf("latest failed run should not expose stale top-level result: %+v\n%s", step, result.ForLLM)
	}
	if step.LatestSuccessful == nil || step.LatestSuccessful.Text != "caption text" ||
		len(step.LatestSuccessful.Artifacts) != 1 {
		t.Fatalf("unexpected latest successful deliverable: %+v\n%s", step.LatestSuccessful, result.ForLLM)
	}
	if step.LegacyCompletion != nil {
		t.Fatalf("unexpected legacy completion: %+v", step.LegacyCompletion)
	}
}

func TestTaskBoardTool_ListIncludesEffectiveStatusAndFreshness(t *testing.T) {
	registry := taskregistry.NewRegistry(taskregistry.WorkspaceStorePath(t.TempDir()))
	tool := NewTaskBoardTool(registry)
	now := time.Now().UnixMilli()
	old := time.Now().Add(-10 * time.Minute).UnixMilli()

	records := []taskregistry.Record{
		{
			TaskID:         "board:workflow-1:step:media-extract",
			Runtime:        taskregistry.RuntimeTool,
			TaskKind:       "task_board_step",
			BoardID:        "workflow-1",
			StepID:         "media-extract",
			StepTitle:      "Extract media",
			Owner:          "media",
			Task:           "planned media extract",
			Status:         taskregistry.StatusPlanned,
			DeliveryStatus: taskregistry.DeliveryNotApplicable,
			CreatedAt:      now - 10,
			LastEventAt:    now - 10,
		},
		{
			TaskID:         "delegate-media-1",
			Runtime:        taskregistry.RuntimeDelegate,
			TaskKind:       "delegate",
			BoardID:        "workflow-1",
			StepID:         "media-extract",
			StepTitle:      "Extract media",
			AgentID:        "media",
			Task:           "download reel",
			Status:         taskregistry.StatusSucceeded,
			DeliveryStatus: taskregistry.DeliverySessionQueued,
			CreatedAt:      now - 9,
			LastEventAt:    now - 8,
			EndedAt:        now - 8,
			Deliverable: &taskregistry.DeliverablePayload{
				Text: "caption text",
			},
		},
		{
			TaskID:         "board:workflow-1:step:polish-translation",
			Runtime:        taskregistry.RuntimeTool,
			TaskKind:       "task_board_step",
			BoardID:        "workflow-1",
			StepID:         "polish-translation",
			StepTitle:      "Polish translation",
			Owner:          "research",
			Task:           "planned polish",
			Status:         taskregistry.StatusPlanned,
			DeliveryStatus: taskregistry.DeliveryNotApplicable,
			CreatedAt:      old - 10,
			LastEventAt:    old - 10,
		},
		{
			TaskID:         "delegate-polish-1",
			Runtime:        taskregistry.RuntimeDelegate,
			TaskKind:       "delegate",
			BoardID:        "workflow-1",
			StepID:         "polish-translation",
			StepTitle:      "Polish translation",
			AgentID:        "research",
			Task:           "polish recipe",
			Status:         taskregistry.StatusRunning,
			DeliveryStatus: taskregistry.DeliveryPending,
			CreatedAt:      old,
			StartedAt:      old,
			LastEventAt:    old,
		},
	}
	for _, rec := range records {
		if err := registry.Upsert(rec); err != nil {
			t.Fatalf("Upsert(%s) error = %v", rec.TaskID, err)
		}
	}

	result := tool.Execute(WithToolContext(context.Background(), "telegram", "chat-1"), map[string]any{
		"action":   "list",
		"board_id": "workflow-1",
	})
	if result.IsError {
		t.Fatalf("list failed: %s", result.ForLLM)
	}

	var payload struct {
		OverallStatus   string         `json:"overall_status"`
		EffectiveCounts map[string]int `json:"effective_counts"`
		EffectiveSteps  []struct {
			StepID              string `json:"step_id"`
			EffectiveStatus     string `json:"effective_status"`
			Freshness           string `json:"freshness"`
			LatestRunTaskID     string `json:"latest_run_task_id"`
			LastEventAgeSeconds int64  `json:"last_event_age_seconds"`
			Deliverable         bool   `json:"deliverable"`
		} `json:"effective_steps"`
	}
	if err := json.Unmarshal([]byte(result.ForLLM), &payload); err != nil {
		t.Fatalf("list JSON error = %v\n%s", err, result.ForLLM)
	}
	if payload.OverallStatus != "stalled" {
		t.Fatalf("overall_status = %q, want stalled\n%s", payload.OverallStatus, result.ForLLM)
	}
	if payload.EffectiveCounts["succeeded"] != 1 || payload.EffectiveCounts["running"] != 1 {
		t.Fatalf("effective counts = %#v", payload.EffectiveCounts)
	}
	if len(payload.EffectiveSteps) != 2 {
		t.Fatalf("effective steps = %#v", payload.EffectiveSteps)
	}
	mediaStep := payload.EffectiveSteps[0]
	if mediaStep.StepID != "media-extract" ||
		mediaStep.EffectiveStatus != "succeeded" ||
		mediaStep.Freshness != "finished" ||
		mediaStep.LatestRunTaskID != "delegate-media-1" ||
		!mediaStep.Deliverable {
		t.Fatalf("unexpected media effective step: %#v\n%s", mediaStep, result.ForLLM)
	}
	polishStep := payload.EffectiveSteps[1]
	if polishStep.StepID != "polish-translation" ||
		polishStep.EffectiveStatus != "running" ||
		polishStep.Freshness != "stalled" ||
		polishStep.LatestRunTaskID != "delegate-polish-1" ||
		polishStep.LastEventAgeSeconds < 300 {
		t.Fatalf("unexpected polish effective step: %#v\n%s", polishStep, result.ForLLM)
	}
}

func TestTaskBoardTool_ReadyResolvesDependencies(t *testing.T) {
	registry := taskregistry.NewRegistry(taskregistry.WorkspaceStorePath(t.TempDir()))
	tool := NewTaskBoardTool(registry)
	ctx := WithToolContext(context.Background(), "telegram", "chat-1")
	now := time.Now().UnixMilli()

	records := []taskregistry.Record{
		{
			TaskID:         "board:workflow-1:step:extract",
			Runtime:        taskregistry.RuntimeTool,
			TaskKind:       "task_board_step",
			BoardID:        "workflow-1",
			StepID:         "extract",
			StepTitle:      "Extract media",
			Owner:          "media",
			Channel:        "telegram",
			ChatID:         "chat-1",
			Status:         taskregistry.StatusPlanned,
			DeliveryStatus: taskregistry.DeliveryNotApplicable,
			Task:           "extract",
			CreatedAt:      now,
		},
		{
			TaskID:         "board:workflow-1:step:polish",
			Runtime:        taskregistry.RuntimeTool,
			TaskKind:       "task_board_step",
			BoardID:        "workflow-1",
			StepID:         "polish",
			StepTitle:      "Polish translation",
			Owner:          "research",
			DependsOn:      []string{"extract"},
			Channel:        "telegram",
			ChatID:         "chat-1",
			Status:         taskregistry.StatusPlanned,
			DeliveryStatus: taskregistry.DeliveryNotApplicable,
			Task:           "polish",
			CreatedAt:      now + 1,
		},
		{
			TaskID:         "board:workflow-1:step:publish",
			Runtime:        taskregistry.RuntimeTool,
			TaskKind:       "task_board_step",
			BoardID:        "workflow-1",
			StepID:         "publish",
			StepTitle:      "Publish final",
			DependsOn:      []string{"polish"},
			Channel:        "telegram",
			ChatID:         "chat-1",
			Status:         taskregistry.StatusPlanned,
			DeliveryStatus: taskregistry.DeliveryNotApplicable,
			Task:           "publish",
			CreatedAt:      now + 2,
		},
		{
			TaskID:         "board:workflow-1:step:notify",
			Runtime:        taskregistry.RuntimeTool,
			TaskKind:       "task_board_step",
			BoardID:        "workflow-1",
			StepID:         "notify",
			StepTitle:      "Notify user",
			BlockedBy:      []string{"manual-review"},
			Channel:        "telegram",
			ChatID:         "chat-1",
			Status:         taskregistry.StatusPlanned,
			DeliveryStatus: taskregistry.DeliveryNotApplicable,
			Task:           "notify",
			CreatedAt:      now + 3,
		},
		{
			TaskID:         "board:workflow-1:step:archive",
			Runtime:        taskregistry.RuntimeTool,
			TaskKind:       "task_board_step",
			BoardID:        "workflow-1",
			StepID:         "archive",
			StepTitle:      "Archive outputs",
			DependsOn:      []string{"missing-step"},
			Channel:        "telegram",
			ChatID:         "chat-1",
			Status:         taskregistry.StatusPlanned,
			DeliveryStatus: taskregistry.DeliveryNotApplicable,
			Task:           "archive",
			CreatedAt:      now + 4,
		},
		{
			TaskID:         "board:workflow-1:step:inconsistent-done",
			Runtime:        taskregistry.RuntimeTool,
			TaskKind:       "task_board_step",
			BoardID:        "workflow-1",
			StepID:         "inconsistent-done",
			StepTitle:      "Inconsistent completed step",
			DependsOn:      []string{"missing-required-step"},
			Channel:        "telegram",
			ChatID:         "chat-1",
			Status:         taskregistry.StatusPlanned,
			DeliveryStatus: taskregistry.DeliveryNotApplicable,
			Task:           "inconsistent",
			CreatedAt:      now + 5,
		},
		{
			TaskID:         "delegate-extract-1",
			Runtime:        taskregistry.RuntimeDelegate,
			TaskKind:       "delegate",
			BoardID:        "workflow-1",
			StepID:         "extract",
			StepTitle:      "Extract media",
			AgentID:        "media",
			Channel:        "telegram",
			ChatID:         "chat-1",
			Status:         taskregistry.StatusSucceeded,
			DeliveryStatus: taskregistry.DeliverySessionQueued,
			Task:           "extract run",
			CreatedAt:      now + 6,
			EndedAt:        now + 7,
			LastEventAt:    now + 7,
		},
		{
			TaskID:         "delegate-inconsistent-1",
			Runtime:        taskregistry.RuntimeDelegate,
			TaskKind:       "delegate",
			BoardID:        "workflow-1",
			StepID:         "inconsistent-done",
			StepTitle:      "Inconsistent completed step",
			AgentID:        "main",
			Channel:        "telegram",
			ChatID:         "chat-1",
			Status:         taskregistry.StatusSucceeded,
			DeliveryStatus: taskregistry.DeliverySessionQueued,
			Task:           "inconsistent run",
			CreatedAt:      now + 8,
			EndedAt:        now + 9,
			LastEventAt:    now + 9,
		},
		{
			TaskID:         "delegate-publish-1",
			Runtime:        taskregistry.RuntimeDelegate,
			TaskKind:       "delegate",
			BoardID:        "workflow-1",
			StepID:         "publish",
			StepTitle:      "Publish final",
			AgentID:        "main",
			Channel:        "telegram",
			ChatID:         "chat-1",
			Status:         taskregistry.StatusRunning,
			DeliveryStatus: taskregistry.DeliveryPending,
			Task:           "publish run",
			CreatedAt:      now + 10,
			StartedAt:      now + 10,
			LastEventAt:    now + 10,
		},
	}
	for _, rec := range records {
		if err := registry.Upsert(rec); err != nil {
			t.Fatalf("Upsert(%s) error = %v", rec.TaskID, err)
		}
	}

	result := tool.Execute(ctx, map[string]any{
		"action":   "ready",
		"board_id": "workflow-1",
	})
	if result.IsError {
		t.Fatalf("ready failed: %s", result.ForLLM)
	}

	var payload struct {
		Counts     map[string]int `json:"counts"`
		ReadySteps []struct {
			StepID    string   `json:"step_id"`
			Reason    string   `json:"reason"`
			DependsOn []string `json:"depends_on"`
		} `json:"ready_steps"`
		WaitingSteps []struct {
			StepID      string   `json:"step_id"`
			MissingDeps []string `json:"missing_dependencies"`
		} `json:"waiting_steps"`
		ActiveSteps []struct {
			StepID          string `json:"step_id"`
			LatestRunTaskID string `json:"latest_run_task_id"`
		} `json:"active_steps"`
		DoneSteps []struct {
			StepID string `json:"step_id"`
		} `json:"done_steps"`
		BlockedSteps []struct {
			StepID      string   `json:"step_id"`
			BlockedBy   []string `json:"blocked_by"`
			MissingDeps []string `json:"missing_dependencies"`
		} `json:"blocked_steps"`
	}
	if err := json.Unmarshal([]byte(result.ForLLM), &payload); err != nil {
		t.Fatalf("ready JSON error = %v\n%s", err, result.ForLLM)
	}
	if payload.Counts["ready"] != 1 ||
		payload.Counts["waiting"] != 1 ||
		payload.Counts["active"] != 1 ||
		payload.Counts["done"] != 1 ||
		payload.Counts["blocked"] != 2 {
		t.Fatalf("unexpected counts: %#v\n%s", payload.Counts, result.ForLLM)
	}
	if len(payload.ReadySteps) != 1 ||
		payload.ReadySteps[0].StepID != "polish" ||
		payload.ReadySteps[0].Reason != "ready" ||
		payload.ReadySteps[0].DependsOn[0] != "extract" {
		t.Fatalf("unexpected ready steps: %#v\n%s", payload.ReadySteps, result.ForLLM)
	}
	if len(payload.WaitingSteps) != 1 ||
		payload.WaitingSteps[0].StepID != "archive" ||
		payload.WaitingSteps[0].MissingDeps[0] != "missing-step" {
		t.Fatalf("unexpected waiting steps: %#v\n%s", payload.WaitingSteps, result.ForLLM)
	}
	if len(payload.ActiveSteps) != 1 ||
		payload.ActiveSteps[0].StepID != "publish" ||
		payload.ActiveSteps[0].LatestRunTaskID != "delegate-publish-1" {
		t.Fatalf("unexpected active steps: %#v\n%s", payload.ActiveSteps, result.ForLLM)
	}
	if len(payload.DoneSteps) != 1 || payload.DoneSteps[0].StepID != "extract" {
		t.Fatalf("unexpected done steps: %#v\n%s", payload.DoneSteps, result.ForLLM)
	}
	if len(payload.BlockedSteps) != 2 {
		t.Fatalf("unexpected blocked steps: %#v\n%s", payload.BlockedSteps, result.ForLLM)
	}
	blockedByStep := map[string]struct {
		BlockedBy   []string
		MissingDeps []string
	}{}
	for _, step := range payload.BlockedSteps {
		blockedByStep[step.StepID] = struct {
			BlockedBy   []string
			MissingDeps []string
		}{BlockedBy: step.BlockedBy, MissingDeps: step.MissingDeps}
	}
	if blockedByStep["notify"].BlockedBy[0] != "manual-review" {
		t.Fatalf("notify should be explicitly blocked: %#v\n%s", payload.BlockedSteps, result.ForLLM)
	}
	if blockedByStep["inconsistent-done"].MissingDeps[0] != "missing-required-step" {
		t.Fatalf(
			"succeeded step with missing dependency should be blocked: %#v\n%s",
			payload.BlockedSteps,
			result.ForLLM,
		)
	}
}

func TestTaskBoardTool_NextReturnsDryRunExecutionPlan(t *testing.T) {
	registry := taskregistry.NewRegistry(taskregistry.WorkspaceStorePath(t.TempDir()))
	tool := NewTaskBoardTool(registry)
	ctx := WithToolContext(context.Background(), "telegram", "chat-1")
	now := time.Now().UnixMilli()

	records := []taskregistry.Record{
		{
			TaskID:         "board:workflow-1:step:media-extract",
			Runtime:        taskregistry.RuntimeTool,
			TaskKind:       "task_board_step",
			BoardID:        "workflow-1",
			StepID:         "media-extract",
			StepTitle:      "Extract media",
			Owner:          "media",
			Channel:        "telegram",
			ChatID:         "chat-1",
			Status:         taskregistry.StatusPlanned,
			DeliveryStatus: taskregistry.DeliveryNotApplicable,
			Task:           "Download the reel and extract the caption.",
			CreatedAt:      now,
			LastEventAt:    now,
		},
		{
			TaskID:         "board:workflow-1:step:local-summary",
			Runtime:        taskregistry.RuntimeTool,
			TaskKind:       "task_board_step",
			BoardID:        "workflow-1",
			StepID:         "local-summary",
			StepTitle:      "Write final summary",
			Channel:        "telegram",
			ChatID:         "chat-1",
			Status:         taskregistry.StatusPlanned,
			DeliveryStatus: taskregistry.DeliveryNotApplicable,
			Task:           "Write the final response in the parent conversation.",
			CreatedAt:      now + 1,
			LastEventAt:    now + 1,
		},
	}
	for _, rec := range records {
		if err := registry.Upsert(rec); err != nil {
			t.Fatalf("Upsert(%s) error = %v", rec.TaskID, err)
		}
	}

	result := tool.Execute(ctx, map[string]any{
		"action":   "next",
		"board_id": "workflow-1",
	})
	if result.IsError {
		t.Fatalf("next failed: %s", result.ForLLM)
	}

	type planStep struct {
		StepID          string         `json:"step_id"`
		RecommendedTool string         `json:"recommended_tool"`
		DelegateArgs    map[string]any `json:"delegate_args"`
		SpawnArgs       map[string]any `json:"spawn_args"`
		UpdateArgs      map[string]any `json:"update_args"`
	}
	var payload struct {
		Action    string     `json:"action"`
		CanRun    bool       `json:"can_run"`
		PlanCount int        `json:"plan_count"`
		Plan      []planStep `json:"plan"`
	}
	if err := json.Unmarshal([]byte(result.ForLLM), &payload); err != nil {
		t.Fatalf("next JSON error = %v\n%s", err, result.ForLLM)
	}
	if payload.Action != "next" || !payload.CanRun || payload.PlanCount != 2 || len(payload.Plan) != 2 {
		t.Fatalf("unexpected next payload: %+v\n%s", payload, result.ForLLM)
	}

	planByStep := make(map[string]planStep, len(payload.Plan))
	for _, step := range payload.Plan {
		planByStep[step.StepID] = step
	}

	media := planByStep["media-extract"]
	if media.StepID != "media-extract" ||
		media.RecommendedTool != "delegate" ||
		media.DelegateArgs["agent_id"] != "media" ||
		media.DelegateArgs["board_id"] != "workflow-1" ||
		media.DelegateArgs["step_id"] != "media-extract" ||
		media.SpawnArgs["agent_id"] != "media" {
		t.Fatalf("unexpected media plan: %+v\n%s", media, result.ForLLM)
	}

	local := planByStep["local-summary"]
	if local.StepID != "local-summary" ||
		local.RecommendedTool != "task_board.update" ||
		local.UpdateArgs["status"] != "running" ||
		local.UpdateArgs["board_id"] != "workflow-1" ||
		local.UpdateArgs["step_id"] != "local-summary" {
		t.Fatalf("unexpected local plan: %+v\n%s", local, result.ForLLM)
	}
}

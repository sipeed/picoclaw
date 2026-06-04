package tools

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	taskregistry "github.com/sipeed/picoclaw/pkg/tasks"
)

// delegateMockSpawner records the config and returns a canned result.
type delegateMockSpawner struct {
	lastCfg SubTurnConfig
	result  *ToolResult
	err     error
}

func (m *delegateMockSpawner) SpawnSubTurn(_ context.Context, cfg SubTurnConfig) (*ToolResult, error) {
	m.lastCfg = cfg
	if m.err != nil {
		return nil, m.err
	}
	if m.result != nil {
		return m.result, nil
	}
	return &ToolResult{
		ForLLM:  "completed: " + cfg.SystemPrompt,
		ForUser: "completed",
	}, nil
}

func TestDelegateTool_Name(t *testing.T) {
	tool := NewDelegateTool()
	if tool.Name() != "delegate" {
		t.Errorf("Name() = %q, want %q", tool.Name(), "delegate")
	}
}

func TestDelegateTool_Parameters(t *testing.T) {
	tool := NewDelegateTool()
	params := tool.Parameters()

	props, ok := params["properties"].(map[string]any)
	if !ok {
		t.Fatal("properties should be a map")
	}
	_, hasAgentID := props["agent_id"]
	if !hasAgentID {
		t.Error("agent_id parameter should exist")
	}
	_, hasTask := props["task"]
	if !hasTask {
		t.Error("task parameter should exist")
	}

	required, ok := params["required"].([]string)
	if !ok {
		t.Fatal("required should be a string array")
	}
	if len(required) != 2 {
		t.Fatalf("required should have 2 entries, got %d", len(required))
	}
}

func TestDelegateTool_Execute_Success(t *testing.T) {
	spawner := &delegateMockSpawner{}
	tool := NewDelegateTool()
	tool.SetSpawner(spawner)

	result := tool.Execute(context.Background(), map[string]any{
		"agent_id": "researcher",
		"task":     "summarize the logs",
	})

	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, `[Response from agent "researcher"]`) {
		t.Errorf("result should contain attribution, got: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "summarize the logs") {
		t.Errorf("result should contain task output, got: %s", result.ForLLM)
	}

	// Verify spawner received correct config
	if spawner.lastCfg.TargetAgentID != "researcher" {
		t.Errorf("TargetAgentID = %q, want %q", spawner.lastCfg.TargetAgentID, "researcher")
	}
	if spawner.lastCfg.Async {
		t.Error("delegate should be synchronous (Async=false)")
	}
	if spawner.lastCfg.SystemPrompt != "summarize the logs" {
		t.Errorf("SystemPrompt = %q, want %q", spawner.lastCfg.SystemPrompt, "summarize the logs")
	}
	if spawner.lastCfg.DeliveryMode != AsyncDeliveryParentOnly {
		t.Errorf("DeliveryMode = %q, want %q", spawner.lastCfg.DeliveryMode, AsyncDeliveryParentOnly)
	}
}

func TestDelegateTool_Execute_PassesTimeoutSeconds(t *testing.T) {
	spawner := &delegateMockSpawner{}
	tool := NewDelegateTool()
	tool.SetSpawner(spawner)

	result := tool.Execute(context.Background(), map[string]any{
		"agent_id":        "researcher",
		"task":            "summarize the logs",
		"timeout_seconds": 2.5,
	})

	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.ForLLM)
	}
	if spawner.lastCfg.Timeout != 2500*time.Millisecond {
		t.Fatalf("Timeout = %v, want 2.5s", spawner.lastCfg.Timeout)
	}
}

func TestDelegateTool_Execute_RecordsTaskRegistry(t *testing.T) {
	registry := taskregistry.NewRegistry(taskregistry.WorkspaceStorePath(t.TempDir()))
	spawner := &delegateMockSpawner{}
	tool := NewDelegateTool()
	tool.SetSpawner(spawner)
	tool.SetTaskRegistry(registry)

	ctx := WithToolContext(context.Background(), "telegram", "chat-1")
	ctx = WithToolTopicID(ctx, "topic-1")
	ctx = WithToolSessionContext(ctx, "main", "session-1", nil)
	result := tool.Execute(ctx, map[string]any{
		"agent_id": "media",
		"task":     "download reel",
	})

	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.ForLLM)
	}
	records := registry.List()
	if len(records) != 1 {
		t.Fatalf("registry records = %d, want 1: %#v", len(records), records)
	}
	rec := records[0]
	if !strings.HasPrefix(rec.TaskID, "delegate-") {
		t.Fatalf("TaskID = %q, want delegate-*", rec.TaskID)
	}
	if rec.Runtime != taskregistry.RuntimeDelegate {
		t.Fatalf("Runtime = %q, want %q", rec.Runtime, taskregistry.RuntimeDelegate)
	}
	if rec.TaskKind != "delegate" {
		t.Fatalf("TaskKind = %q, want delegate", rec.TaskKind)
	}
	if rec.Status != taskregistry.StatusSucceeded {
		t.Fatalf("Status = %q, want succeeded", rec.Status)
	}
	if rec.DeliveryStatus != taskregistry.DeliverySessionQueued {
		t.Fatalf("DeliveryStatus = %q, want session_queued", rec.DeliveryStatus)
	}
	if rec.AgentID != "media" || rec.Channel != "telegram" || rec.ChatID != "chat-1" || rec.TopicID != "topic-1" {
		t.Fatalf("unexpected routing fields: %+v", rec)
	}
	if rec.RequesterSessionKey != "session-1" || rec.OwnerKey != "main" {
		t.Fatalf("unexpected owner fields: %+v", rec)
	}
	if rec.Deliverable != nil {
		t.Fatalf("unexpected deliverable for plain result: %+v", rec.Deliverable)
	}
}

func TestDelegateTool_Execute_RecordsBoardMetadata(t *testing.T) {
	registry := taskregistry.NewRegistry(taskregistry.WorkspaceStorePath(t.TempDir()))
	tool := NewDelegateTool()
	tool.SetSpawner(&delegateMockSpawner{})
	tool.SetTaskRegistry(registry)

	for _, step := range []struct {
		id    string
		title string
		agent string
		deps  []any
	}{
		{id: "download-media", title: "Download media", agent: "media"},
		{id: "translate-recipe", title: "Translate recipe", agent: "research", deps: []any{"download-media"}},
	} {
		result := tool.Execute(context.Background(), map[string]any{
			"agent_id":       step.agent,
			"task":           step.title,
			"board_id":       "instagram-recipe-1",
			"parent_task_id": "workflow-root-1",
			"step_id":        step.id,
			"step_title":     step.title,
			"depends_on":     step.deps,
		})
		if result.IsError {
			t.Fatalf("delegate step %q failed: %s", step.id, result.ForLLM)
		}
	}

	records := registry.ListBoard("instagram-recipe-1")
	if len(records) != 2 {
		t.Fatalf("ListBoard count = %d, want 2: %+v", len(records), records)
	}
	if records[0].StepID != "download-media" || records[0].StepTitle != "Download media" {
		t.Fatalf("first board step = %+v, want download-media", records[0])
	}
	if records[1].ParentTaskID != "workflow-root-1" || records[1].Owner != "research" {
		t.Fatalf("second board ownership = %+v, want parent root and owner research", records[1])
	}
	if len(records[1].DependsOn) != 1 || records[1].DependsOn[0] != "download-media" {
		t.Fatalf("second board dependencies = %+v, want download-media", records[1].DependsOn)
	}
}

func TestDelegateTool_Execute_RecordsDeliverableFromCompletion(t *testing.T) {
	registry := taskregistry.NewRegistry(taskregistry.WorkspaceStorePath(t.TempDir()))
	spawner := &delegateMockSpawner{
		result: (&ToolResult{
			ForLLM: "child finished",
			Completion: &CompletionResult{
				Text: "recipe text",
				Media: []CompletionMedia{{
					Ref:         "media://video",
					Type:        "video",
					Filename:    "source.mp4",
					ContentType: "video/mp4",
				}},
			},
		}),
	}
	tool := NewDelegateTool()
	tool.SetSpawner(spawner)
	tool.SetTaskRegistry(registry)

	result := tool.Execute(context.Background(), map[string]any{
		"agent_id": "media",
		"task":     "download reel",
	})

	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.ForLLM)
	}
	records := registry.List()
	if len(records) != 1 {
		t.Fatalf("registry records = %d, want 1: %#v", len(records), records)
	}
	rec := records[0]
	if rec.Completion != nil {
		t.Fatalf("Completion = %+v, want nil when Deliverable is present", rec.Completion)
	}
	if rec.Deliverable == nil || rec.Deliverable.Text != "recipe text" {
		t.Fatalf("Deliverable = %+v, want recipe text", rec.Deliverable)
	}
	if len(rec.Deliverable.Artifacts) != 1 || rec.Deliverable.Artifacts[0].Ref != "media://video" {
		t.Fatalf("Deliverable artifacts = %+v, want media://video", rec.Deliverable.Artifacts)
	}
}

func TestDelegateTool_Execute_RecordsDeliverableArtifactFromLabeledPath(t *testing.T) {
	registry := taskregistry.NewRegistry(taskregistry.WorkspaceStorePath(t.TempDir()))
	spawner := &delegateMockSpawner{
		result: (&ToolResult{
			ForLLM: "child finished",
			Completion: &CompletionResult{
				Text: "- sendable_file_path: `/tmp/picoclaw/source.mp4`\n- russian_recipe_translation: `recipe`",
			},
		}),
	}
	tool := NewDelegateTool()
	tool.SetSpawner(spawner)
	tool.SetTaskRegistry(registry)

	result := tool.Execute(context.Background(), map[string]any{
		"agent_id": "media",
		"task":     "download reel",
	})

	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.ForLLM)
	}
	records := registry.List()
	if len(records) != 1 {
		t.Fatalf("registry records = %d, want 1: %#v", len(records), records)
	}
	deliverable := records[0].Deliverable
	if deliverable == nil {
		t.Fatal("expected deliverable")
	}
	if len(deliverable.Artifacts) != 1 {
		t.Fatalf("artifact count = %d, want 1: %+v", len(deliverable.Artifacts), deliverable)
	}
	artifact := deliverable.Artifacts[0]
	if artifact.Ref != "file:/tmp/picoclaw/source.mp4" {
		t.Fatalf("artifact ref = %q, want file:/tmp/picoclaw/source.mp4", artifact.Ref)
	}
	if artifact.Kind != "video" {
		t.Fatalf("artifact kind = %q, want video", artifact.Kind)
	}
	if artifact.Filename != "source.mp4" {
		t.Fatalf("artifact filename = %q, want source.mp4", artifact.Filename)
	}
	if artifact.ContentType != "video/mp4" {
		t.Fatalf("artifact content type = %q, want video/mp4", artifact.ContentType)
	}
}

func TestCompletionPayloadForLegacyStorageKeepsCompletionOnlyRecords(t *testing.T) {
	completion := &taskregistry.CompletionPayload{Text: "legacy-only result"}
	if got := completionPayloadForLegacyStorage(completion, nil); got != completion {
		t.Fatalf("completion-only storage = %+v, want original completion", got)
	}
	deliverable := &taskregistry.DeliverablePayload{Text: "typed result"}
	if got := completionPayloadForLegacyStorage(completion, deliverable); got != nil {
		t.Fatalf("deliverable-backed storage completion = %+v, want nil", got)
	}
}

func TestDelegateTool_Execute_EmptyAgentID(t *testing.T) {
	tests := []struct {
		name string
		args map[string]any
	}{
		{"missing", map[string]any{"task": "test"}},
		{"empty string", map[string]any{"agent_id": "", "task": "test"}},
		{"whitespace only", map[string]any{"agent_id": "  ", "task": "test"}},
		{"wrong type", map[string]any{"agent_id": 123, "task": "test"}},
	}

	tool := NewDelegateTool()
	tool.SetSpawner(&delegateMockSpawner{})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tool.Execute(context.Background(), tt.args)
			if !result.IsError {
				t.Error("expected error for invalid agent_id")
			}
			if !strings.Contains(result.ForLLM, "agent_id is required") {
				t.Errorf("error should mention agent_id, got: %s", result.ForLLM)
			}
		})
	}
}

func TestDelegateTool_Execute_EmptyTask(t *testing.T) {
	tests := []struct {
		name string
		args map[string]any
	}{
		{"missing", map[string]any{"agent_id": "a"}},
		{"empty string", map[string]any{"agent_id": "a", "task": ""}},
		{"whitespace only", map[string]any{"agent_id": "a", "task": "\t\n"}},
	}

	tool := NewDelegateTool()
	tool.SetSpawner(&delegateMockSpawner{})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tool.Execute(context.Background(), tt.args)
			if !result.IsError {
				t.Error("expected error for invalid task")
			}
			if !strings.Contains(result.ForLLM, "task is required") {
				t.Errorf("error should mention task, got: %s", result.ForLLM)
			}
		})
	}
}

func TestDelegateTool_Execute_PermissionDenied(t *testing.T) {
	tool := NewDelegateTool()
	tool.SetSpawner(&delegateMockSpawner{})
	tool.SetAllowlistChecker(func(targetAgentID string) bool {
		return targetAgentID == "allowed-agent"
	})

	result := tool.Execute(context.Background(), map[string]any{
		"agent_id": "forbidden-agent",
		"task":     "test",
	})

	if !result.IsError {
		t.Error("expected error for denied agent")
	}
	if !strings.Contains(result.ForLLM, "not allowed to delegate") {
		t.Errorf("error should mention permission, got: %s", result.ForLLM)
	}
}

func TestDelegateTool_Execute_PermissionAllowed(t *testing.T) {
	tool := NewDelegateTool()
	tool.SetSpawner(&delegateMockSpawner{})
	tool.SetAllowlistChecker(func(targetAgentID string) bool {
		return targetAgentID == "allowed-agent"
	})

	result := tool.Execute(context.Background(), map[string]any{
		"agent_id": "allowed-agent",
		"task":     "test",
	})

	if result.IsError {
		t.Errorf("expected success for allowed agent, got error: %s", result.ForLLM)
	}
}

func TestDelegateTool_Execute_NoSpawner(t *testing.T) {
	tool := NewDelegateTool()

	result := tool.Execute(context.Background(), map[string]any{
		"agent_id": "a",
		"task":     "test",
	})

	if !result.IsError {
		t.Error("expected error when spawner is nil")
	}
	if !strings.Contains(result.ForLLM, "not configured") {
		t.Errorf("error should mention not configured, got: %s", result.ForLLM)
	}
}

func TestDelegateTool_Execute_SpawnerError(t *testing.T) {
	spawner := &delegateMockSpawner{
		err: fmt.Errorf("context deadline exceeded"),
	}
	tool := NewDelegateTool()
	tool.SetSpawner(spawner)

	result := tool.Execute(context.Background(), map[string]any{
		"agent_id": "researcher",
		"task":     "test",
	})

	if !result.IsError {
		t.Error("expected error when spawner fails")
	}
	if !strings.Contains(result.ForLLM, "delegation to agent") {
		t.Errorf("error should mention delegation failure, got: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "context deadline exceeded") {
		t.Errorf("error should propagate cause, got: %s", result.ForLLM)
	}
}

func TestDelegateTool_Execute_NoAllowlistCheck(t *testing.T) {
	// When no allowlist checker is set, all agents are allowed
	tool := NewDelegateTool()
	tool.SetSpawner(&delegateMockSpawner{})

	result := tool.Execute(context.Background(), map[string]any{
		"agent_id": "any-agent",
		"task":     "test",
	})

	if result.IsError {
		t.Errorf("expected success without allowlist, got error: %s", result.ForLLM)
	}
}

func TestDelegateTool_Execute_UserOnlyMarksHandled(t *testing.T) {
	spawner := &delegateMockSpawner{}
	tool := NewDelegateTool()
	tool.SetSpawner(spawner)

	result := tool.Execute(context.Background(), map[string]any{
		"agent_id":      "media",
		"task":          "deliver this to the user",
		"delivery_mode": string(AsyncDeliveryUserOnly),
	})

	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.ForLLM)
	}
	if !result.ResponseHandled {
		t.Fatal("expected delegate user_only result to be marked ResponseHandled")
	}
	if !result.Silent {
		t.Fatal("expected delegate user_only result to be silent in parent")
	}
	if spawner.lastCfg.DeliveryMode != AsyncDeliveryUserOnly {
		t.Fatalf("DeliveryMode = %q, want %q", spawner.lastCfg.DeliveryMode, AsyncDeliveryUserOnly)
	}
}

func TestDelegateTool_Execute_InvalidDeliveryMode(t *testing.T) {
	tool := NewDelegateTool()
	tool.SetSpawner(&delegateMockSpawner{})

	result := tool.Execute(context.Background(), map[string]any{
		"agent_id":      "media",
		"task":          "test",
		"delivery_mode": "wrong",
	})

	if !result.IsError {
		t.Fatal("expected invalid delivery_mode to error")
	}
	if !strings.Contains(result.ForLLM, "delivery_mode") {
		t.Fatalf("expected delivery_mode error, got %q", result.ForLLM)
	}
}

func TestDelegateTool_Execute_NilResult(t *testing.T) {
	tool := NewDelegateTool()
	tool.SetSpawner(&nilResultSpawner{})

	result := tool.Execute(context.Background(), map[string]any{
		"agent_id": "researcher",
		"task":     "test",
	})

	if !result.IsError {
		t.Error("expected error for nil result")
	}
	if !strings.Contains(result.ForLLM, "returned no result") {
		t.Errorf("error should mention no result, got: %s", result.ForLLM)
	}
}

func TestDelegateTool_Execute_SelfDelegation(t *testing.T) {
	tool := NewDelegateTool()
	tool.SetSpawner(&delegateMockSpawner{})
	tool.SetSelfAgentID("alpha")

	result := tool.Execute(context.Background(), map[string]any{
		"agent_id": "alpha",
		"task":     "test",
	})

	if !result.IsError {
		t.Error("expected error for self-delegation")
	}
	if !strings.Contains(result.ForLLM, "cannot delegate to self") {
		t.Errorf("error should mention self-delegation, got: %s", result.ForLLM)
	}
}

func TestDelegateTool_Execute_SelfDelegation_Normalized(t *testing.T) {
	tool := NewDelegateTool()
	tool.SetSpawner(&delegateMockSpawner{})
	tool.SetSelfAgentID("alpha") // stored normalized

	// Case-insensitive and whitespace variants should still be caught
	variants := []string{"ALPHA", " Alpha ", "  alpha  "}
	for _, v := range variants {
		t.Run(v, func(t *testing.T) {
			result := tool.Execute(context.Background(), map[string]any{
				"agent_id": v,
				"task":     "test",
			})
			if !result.IsError {
				t.Errorf("agent_id=%q should be caught as self-delegation", v)
			}
		})
	}
}

// nilResultSpawner always returns (nil, nil).
type nilResultSpawner struct{}

func (m *nilResultSpawner) SpawnSubTurn(_ context.Context, _ SubTurnConfig) (*ToolResult, error) {
	return nil, nil
}

package tools

import (
	"context"
	"testing"
	"time"
)

func TestSubmitPlanTool_Execute_Approved(t *testing.T) {
	outCh := make(chan ContainerMessage, 4)

	inCh := make(chan string, 1)

	tool := NewSubmitPlanTool("subagent-1", "conductor:main", "subagent:subagent-1", outCh, inCh, nil)

	var gotGoal string

	var gotSteps []string

	tool.SetPlanCallback(func(goal string, steps []string) {
		gotGoal = goal

		gotSteps = steps
	})

	if tool.Name() != "submit_plan" {
		t.Errorf("Name() = %q, want %q", tool.Name(), "submit_plan")
	}

	// Simulate conductor approving in background.

	go func() {
		msg := <-outCh

		if msg.Type != "plan_review" {
			t.Errorf("msg.Type = %q, want %q", msg.Type, "plan_review")
		}

		inCh <- "approved"
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	defer cancel()

	result := tool.Execute(ctx, map[string]any{
		"goal": "Add auth",

		"steps": []any{"Add middleware", "Add JWT"},
	})

	if result.IsError {
		t.Fatalf("unexpected error: %s", result.ForLLM)
	}

	if gotGoal != "Add auth" {
		t.Errorf("setPlan goal = %q, want %q", gotGoal, "Add auth")
	}

	if len(gotSteps) != 2 {
		t.Errorf("setPlan steps count = %d, want 2", len(gotSteps))
	}
}

func TestSubmitPlanTool_Execute_Rejected(t *testing.T) {
	outCh := make(chan ContainerMessage, 4)

	inCh := make(chan string, 1)

	tool := NewSubmitPlanTool("subagent-1", "conductor:main", "subagent:subagent-1", outCh, inCh, nil)

	var planSet bool

	tool.SetPlanCallback(func(goal string, steps []string) {
		planSet = true
	})

	go func() {
		<-outCh

		inCh <- "rejected: needs more detail on step 2"
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	defer cancel()

	result := tool.Execute(ctx, map[string]any{
		"goal": "Add auth",

		"steps": []any{"Add middleware"},
	})

	if result.IsError {
		t.Fatalf("unexpected error: %s", result.ForLLM)
	}

	if planSet {
		t.Error("setPlan should NOT be called on rejection")
	}

	if result.ForLLM == "" {
		t.Error("ForLLM should contain rejection message")
	}
}

func TestSubmitPlanTool_MissingParams(t *testing.T) {
	tool := NewSubmitPlanTool("subagent-1", "", "", nil, nil, nil)

	result := tool.Execute(context.Background(), map[string]any{})

	if !result.IsError {
		t.Error("expected error for missing goal")
	}

	result = tool.Execute(context.Background(), map[string]any{"goal": "test"})

	if !result.IsError {
		t.Error("expected error for missing steps")
	}
}

func TestSubmitPlanTool_ContextCanceled(t *testing.T) {
	outCh := make(chan ContainerMessage) // unbuffered

	inCh := make(chan string)

	tool := NewSubmitPlanTool("subagent-1", "", "", outCh, inCh, nil)

	ctx, cancel := context.WithCancel(context.Background())

	cancel()

	result := tool.Execute(ctx, map[string]any{
		"goal": "test",

		"steps": []any{"step1"},
	})

	if !result.IsError {
		t.Error("expected error on canceled context")
	}
}

func TestAnswerSubagentTool_Execute(t *testing.T) {
	mgr := &SubagentManager{
		tasks: map[string]*SubagentTask{
			"subagent-1": {
				ID: "subagent-1",

				inCh: make(chan string, 1),
			},
		},
	}

	tool := NewAnswerSubagentTool(mgr)

	if tool.Name() != "answer_subagent" {
		t.Errorf("Name() = %q, want %q", tool.Name(), "answer_subagent")
	}

	result := tool.Execute(context.Background(), map[string]any{
		"task_id": "subagent-1",

		"answer": "Use port 8080",
	})

	if result.IsError {
		t.Fatalf("unexpected error: %s", result.ForLLM)
	}

	// Verify the answer was sent.

	answer := <-mgr.tasks["subagent-1"].inCh

	if answer != "Use port 8080" {
		t.Errorf("answer = %q, want %q", answer, "Use port 8080")
	}
}

func TestAnswerSubagentTool_MissingParams(t *testing.T) {
	tool := NewAnswerSubagentTool(nil)

	result := tool.Execute(context.Background(), map[string]any{})

	if !result.IsError {
		t.Error("expected error for missing task_id")
	}

	result = tool.Execute(context.Background(), map[string]any{"task_id": "x"})

	if !result.IsError {
		t.Error("expected error for missing answer")
	}
}

func TestReviewSubagentPlanTool_Execute(t *testing.T) {
	mgr := &SubagentManager{
		tasks: map[string]*SubagentTask{
			"subagent-1": {
				ID: "subagent-1",

				inCh: make(chan string, 1),
			},
		},
	}

	tool := NewReviewSubagentPlanTool(mgr)

	if tool.Name() != "review_subagent_plan" {
		t.Errorf("Name() = %q, want %q", tool.Name(), "review_subagent_plan")
	}

	result := tool.Execute(context.Background(), map[string]any{
		"task_id": "subagent-1",

		"decision": "approved",
	})

	if result.IsError {
		t.Fatalf("unexpected error: %s", result.ForLLM)
	}

	decision := <-mgr.tasks["subagent-1"].inCh

	if decision != "approved" {
		t.Errorf("decision = %q, want %q", decision, "approved")
	}
}

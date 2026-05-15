package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestUpdatePlanToolParameters(t *testing.T) {
	tool := NewUpdatePlanTool()

	if got := tool.Name(); got != "update_plan" {
		t.Fatalf("Name() = %q, want update_plan", got)
	}
	description := tool.Description()
	for _, want := range []string{"non-trivial multi-step work", "exactly one step in_progress", "avoid repeating the whole plan"} {
		if !strings.Contains(description, want) {
			t.Fatalf("Description() missing %q: %s", want, description)
		}
	}
	params := tool.Parameters()
	if params["type"] != "object" {
		t.Fatalf("Parameters type = %v, want object", params["type"])
	}
	if params["properties"] == nil {
		t.Fatal("Parameters should include properties")
	}
}

func TestUpdatePlanToolExecute(t *testing.T) {
	tool := NewUpdatePlanTool()

	result := tool.Execute(context.Background(), map[string]any{
		"explanation": "Starting implementation.",
		"plan": []any{
			map[string]any{"step": "Inspect existing tools", "status": "completed"},
			map[string]any{"step": "Add update_plan", "status": "in_progress"},
			map[string]any{"step": "Run tests", "status": "pending"},
		},
	})
	if result == nil {
		t.Fatal("Execute returned nil")
	}
	if result.IsError {
		t.Fatalf("Execute returned error: %s", result.ForLLM)
	}
	if !result.Silent {
		t.Fatal("update_plan should be silent")
	}

	var response updatePlanResponse
	if err := json.Unmarshal([]byte(result.ForLLM), &response); err != nil {
		t.Fatalf("result is not JSON: %v\n%s", err, result.ForLLM)
	}
	if response.Status != "updated" {
		t.Fatalf("status = %q, want updated", response.Status)
	}
	if response.Explanation != "Starting implementation." {
		t.Fatalf("explanation = %q", response.Explanation)
	}
	if len(response.Plan) != 3 {
		t.Fatalf("plan length = %d, want 3", len(response.Plan))
	}
	if response.Plan[1].Status != "in_progress" {
		t.Fatalf("second step status = %q, want in_progress", response.Plan[1].Status)
	}
}

func TestUpdatePlanToolRejectsMultipleInProgress(t *testing.T) {
	tool := NewUpdatePlanTool()

	result := tool.Execute(context.Background(), map[string]any{
		"plan": []any{
			map[string]any{"step": "One", "status": "in_progress"},
			map[string]any{"step": "Two", "status": "in_progress"},
		},
	})
	if result == nil || !result.IsError {
		t.Fatalf("Execute should reject multiple in_progress steps, got %#v", result)
	}
	if !strings.Contains(result.ForLLM, "at most one in_progress") {
		t.Fatalf("unexpected error: %s", result.ForLLM)
	}
}

func TestUpdatePlanToolRejectsInvalidStatus(t *testing.T) {
	tool := NewUpdatePlanTool()

	result := tool.Execute(context.Background(), map[string]any{
		"plan": []any{
			map[string]any{"step": "One", "status": "blocked"},
		},
	})
	if result == nil || !result.IsError {
		t.Fatalf("Execute should reject invalid status, got %#v", result)
	}
	if !strings.Contains(result.ForLLM, "must be one of") {
		t.Fatalf("unexpected error: %s", result.ForLLM)
	}
}

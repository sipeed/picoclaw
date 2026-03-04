package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// SubmitPlanTool allows a subagent to submit a plan for conductor review.

// The subagent blocks until the conductor approves or rejects.

type SubmitPlanTool struct {
	taskID string

	conductorKey string

	subagentKey string

	outCh chan<- ContainerMessage

	inCh <-chan string

	recorder SessionRecorder

	setPlan func(goal string, steps []string) // callback to record plan on task
}

func NewSubmitPlanTool(
	taskID, conductorKey, subagentKey string,

	outCh chan<- ContainerMessage,

	inCh <-chan string,

	recorder SessionRecorder,
) *SubmitPlanTool {
	return &SubmitPlanTool{
		taskID: taskID,

		conductorKey: conductorKey,

		subagentKey: subagentKey,

		outCh: outCh,

		inCh: inCh,

		recorder: recorder,
	}
}

// SetPlanCallback sets the function called when a plan is approved to record

// the goal and steps on the parent SubagentTask.

func (t *SubmitPlanTool) SetPlanCallback(fn func(goal string, steps []string)) {
	t.setPlan = fn
}

func (t *SubmitPlanTool) Name() string { return "submit_plan" }

func (t *SubmitPlanTool) Description() string {
	return "Submit your execution plan for conductor review. Blocks until the conductor approves or rejects. On rejection, revise and resubmit."
}

func (t *SubmitPlanTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",

		"properties": map[string]any{
			"goal": map[string]any{
				"type": "string",

				"description": "The goal of the plan",
			},

			"steps": map[string]any{
				"type": "array",

				"description": "Ordered list of steps to execute",

				"items": map[string]any{"type": "string"},
			},
		},

		"required": []string{"goal", "steps"},
	}
}

func (t *SubmitPlanTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	goal, _ := args["goal"].(string)

	if goal == "" {
		return ErrorResult("required parameter \"goal\" (string) is missing")
	}

	stepsRaw, _ := args["steps"]

	var steps []string

	switch v := stepsRaw.(type) {
	case []any:

		steps = make([]string, 0, len(v))

		for _, s := range v {
			if str, ok := s.(string); ok {
				steps = append(steps, str)
			}
		}

	case []string:

		steps = v
	}

	if len(steps) == 0 {
		return ErrorResult("required parameter \"steps\" (array of strings) is missing or empty")
	}

	// Build plan text for recording and display.

	var b strings.Builder

	b.WriteString("Goal: ")

	b.WriteString(goal)

	b.WriteByte('\n')

	for i, step := range steps {
		fmt.Fprintf(&b, "%d. %s\n", i+1, step)
	}

	planText := b.String()

	// Record in session DAG.

	if t.recorder != nil {
		_ = t.recorder.RecordPlanSubmit(t.conductorKey, t.subagentKey, t.taskID, planText)
	}

	// Encode plan as JSON for the conductor.

	planJSON, _ := json.Marshal(map[string]any{"goal": goal, "steps": steps})

	// Send plan_review to conductor.

	select {
	case t.outCh <- ContainerMessage{Type: "plan_review", Content: string(planJSON), TaskID: t.taskID}:

	case <-ctx.Done():

		return ErrorResult(fmt.Sprintf("context canceled while submitting plan: %v", ctx.Err()))
	}

	// Wait for conductor's decision.

	select {
	case decision := <-t.inCh:

		if strings.HasPrefix(decision, "approved") {
			if t.setPlan != nil {
				t.setPlan(goal, steps)
			}

			return &ToolResult{
				ForLLM: "Plan approved by conductor. Proceed with execution.",

				ForUser: "Plan approved",
			}
		}

		return &ToolResult{
			ForLLM: fmt.Sprintf("Plan rejected by conductor: %s\nRevise your plan and resubmit.", decision),

			ForUser: fmt.Sprintf("Plan rejected: %s", decision),
		}

	case <-ctx.Done():

		return ErrorResult(fmt.Sprintf("context canceled while waiting for review: %v", ctx.Err()))
	}
}

package tools

import (
	"context"
	"fmt"
)

// AnswerSubagentTool allows the conductor to answer a subagent's question.

type AnswerSubagentTool struct {
	manager *SubagentManager
}

func NewAnswerSubagentTool(manager *SubagentManager) *AnswerSubagentTool {
	return &AnswerSubagentTool{manager: manager}
}

func (t *AnswerSubagentTool) Name() string { return "answer_subagent" }

func (t *AnswerSubagentTool) Description() string {
	return "Answer a subagent's question or escalation. The subagent is blocked waiting for your response."
}

func (t *AnswerSubagentTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",

		"properties": map[string]any{
			"task_id": map[string]any{
				"type": "string",

				"description": "The task ID of the subagent (e.g. subagent-1)",
			},

			"answer": map[string]any{
				"type": "string",

				"description": "Your answer to the subagent's question",
			},
		},

		"required": []string{"task_id", "answer"},
	}
}

func (t *AnswerSubagentTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	taskID, _ := args["task_id"].(string)

	if taskID == "" {
		return ErrorResult("required parameter \"task_id\" (string) is missing")
	}

	answer, _ := args["answer"].(string)

	if answer == "" {
		return ErrorResult("required parameter \"answer\" (string) is missing")
	}

	if t.manager == nil {
		return ErrorResult("subagent manager not available")
	}

	if err := t.manager.AnswerQuestion(taskID, answer); err != nil {
		return ErrorResult(fmt.Sprintf("failed to answer subagent: %v", err))
	}

	return &ToolResult{
		ForLLM: fmt.Sprintf("Answer sent to %s.", taskID),

		ForUser: fmt.Sprintf("Answered %s", taskID),
	}
}

// ReviewSubagentPlanTool allows the conductor to approve/reject a subagent's plan.

type ReviewSubagentPlanTool struct {
	manager *SubagentManager
}

func NewReviewSubagentPlanTool(manager *SubagentManager) *ReviewSubagentPlanTool {
	return &ReviewSubagentPlanTool{manager: manager}
}

func (t *ReviewSubagentPlanTool) Name() string { return "review_subagent_plan" }

func (t *ReviewSubagentPlanTool) Description() string {
	return "Approve or reject a subagent's execution plan. Use decision 'approved' to approve, or provide rejection feedback."
}

func (t *ReviewSubagentPlanTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",

		"properties": map[string]any{
			"task_id": map[string]any{
				"type": "string",

				"description": "The task ID of the subagent (e.g. subagent-1)",
			},

			"decision": map[string]any{
				"type": "string",

				"description": "Decision: 'approved' to approve, or rejection feedback text",
			},
		},

		"required": []string{"task_id", "decision"},
	}
}

func (t *ReviewSubagentPlanTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	taskID, _ := args["task_id"].(string)

	if taskID == "" {
		return ErrorResult("required parameter \"task_id\" (string) is missing")
	}

	decision, _ := args["decision"].(string)

	if decision == "" {
		return ErrorResult("required parameter \"decision\" (string) is missing")
	}

	if t.manager == nil {
		return ErrorResult("subagent manager not available")
	}

	if err := t.manager.AnswerQuestion(taskID, decision); err != nil {
		return ErrorResult(fmt.Sprintf("failed to send review decision: %v", err))
	}

	return &ToolResult{
		ForLLM: fmt.Sprintf("Review decision '%s' sent to %s.", decision, taskID),

		ForUser: fmt.Sprintf("Reviewed %s: %s", taskID, decision),
	}
}

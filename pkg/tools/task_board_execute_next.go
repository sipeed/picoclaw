package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	taskregistry "github.com/sipeed/picoclaw/pkg/tasks"
)

// TaskBoardExecuteNextTool executes one delegate-backed ready step from a board.
//
// It intentionally does not auto-run spawn/manual steps. Spawn requires async
// callback ownership, and manual steps require parent/local work outside this
// tool. Keeping this first executor delegate-only preserves delivery semantics.
type TaskBoardExecuteNextTool struct {
	registry *taskregistry.Registry
	tools    *ToolRegistry
}

func NewTaskBoardExecuteNextTool(registry *taskregistry.Registry, tools *ToolRegistry) *TaskBoardExecuteNextTool {
	return &TaskBoardExecuteNextTool{registry: registry, tools: tools}
}

func (t *TaskBoardExecuteNextTool) Name() string {
	return "task_board_execute_next"
}

func (t *TaskBoardExecuteNextTool) Description() string {
	return "Execute one delegate-backed ready step from a task_board next plan. " +
		"This is conservative: it does not execute spawn or manual steps; use task_board next for those plans."
}

func (t *TaskBoardExecuteNextTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"board_id": map[string]any{
				"type":        "string",
				"description": "Workflow/task-board ID to execute one ready step from.",
			},
			"step_id": map[string]any{
				"type":        "string",
				"description": "Optional step_id to execute. If omitted, the first delegate-backed ready step from task_board next is used.",
			},
		},
		"required": []string{"board_id"},
	}
}

func (t *TaskBoardExecuteNextTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	if t == nil || t.registry == nil || t.tools == nil {
		return ErrorResult("task_board_execute_next not configured")
	}
	boardID, err := requiredStringArg(args, "board_id", "board_id")
	if err != nil {
		return ErrorResult(err.Error())
	}
	stepID, err := optionalStringArg(args, "step_id")
	if err != nil {
		return ErrorResult(err.Error())
	}

	records := visibleTaskBoardRecordsForToolContext(ctx, t.registry, boardID)
	plan := buildTaskBoardNextView(boardID, records, time.Now())
	selected, ok := selectTaskBoardExecutableStep(plan.Plan, stepID)
	if !ok {
		return taskBoardJSONResult(map[string]any{
			"action":            "execute_next",
			"board_id":          boardID,
			"executed":          false,
			"error":             "no delegate-backed ready step found",
			"next_plan":         plan.Plan,
			"active":            plan.ActiveSteps,
			"waiting":           plan.WaitingSteps,
			"blocked":           plan.BlockedSteps,
			"requested_step_id": stepID,
		})
	}
	if selected.RecommendedTool != "delegate" || selected.DelegateArgs == nil {
		return taskBoardJSONResult(map[string]any{
			"action":           "execute_next",
			"board_id":         boardID,
			"step_id":          selected.StepID,
			"executed":         false,
			"recommended_tool": selected.RecommendedTool,
			"error":            "selected step is not delegate-backed; use task_board next and execute it explicitly",
			"plan":             selected,
		})
	}
	if _, ok := t.tools.Get("delegate"); !ok {
		return ErrorResult("task_board_execute_next requires delegate tool")
	}

	result := t.tools.ExecuteWithContext(
		ctx,
		"delegate",
		selected.DelegateArgs,
		ToolChannel(ctx),
		ToolChatID(ctx),
		nil,
	)
	if result == nil {
		return ErrorResult("delegate returned nil result")
	}
	payload := map[string]any{
		"action":           "execute_next",
		"board_id":         boardID,
		"step_id":          selected.StepID,
		"executed":         !result.IsError,
		"recommended_tool": selected.RecommendedTool,
		"delegate_args":    selected.DelegateArgs,
		"result":           result.ContentForLLM(),
	}
	if result.IsError {
		payload["error"] = result.ContentForLLM()
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to encode task board execution result: %v", err)).WithError(err)
	}
	if result.IsError {
		delegateErr := result.Err
		if delegateErr == nil {
			delegateErr = fmt.Errorf("delegate step failed")
		}
		return ErrorResult(string(data)).WithError(delegateErr)
	}
	return NewToolResult(string(data))
}

func selectTaskBoardExecutableStep(plan []taskBoardNextStepView, stepID string) (taskBoardNextStepView, bool) {
	for _, step := range plan {
		if stepID != "" && step.StepID != stepID {
			continue
		}
		if step.RecommendedTool == "delegate" && step.DelegateArgs != nil {
			return step, true
		}
		if stepID != "" {
			return step, true
		}
	}
	return taskBoardNextStepView{}, false
}

func visibleTaskBoardRecordsForToolContext(
	ctx context.Context,
	registry *taskregistry.Registry,
	boardID string,
) []taskregistry.Record {
	channel := ToolChannel(ctx)
	chatID := ToolChatID(ctx)
	topicID := ToolTopicID(ctx)
	records := registry.ListBoard(boardID)
	filtered := make([]taskregistry.Record, 0, len(records))
	for _, rec := range records {
		if taskRecordVisibleToCaller(rec, channel, chatID, topicID) {
			filtered = append(filtered, rec)
		}
	}
	return filtered
}

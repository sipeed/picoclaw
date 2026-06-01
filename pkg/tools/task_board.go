package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	taskregistry "github.com/sipeed/picoclaw/pkg/tasks"
)

// TaskBoardTool records and inspects durable multi-step workflow boards.
//
// It is intentionally a metadata/results primitive, not a scheduler: agents
// still execute work by calling delegate/spawn with the same board_id/step_id.
type TaskBoardTool struct {
	registry *taskregistry.Registry
}

func NewTaskBoardTool(registry *taskregistry.Registry) *TaskBoardTool {
	return &TaskBoardTool{registry: registry}
}

func (t *TaskBoardTool) Name() string {
	return "task_board"
}

func (t *TaskBoardTool) Description() string {
	return "Create and inspect durable task boards for composite workflows. " +
		"Use this when a task has multiple child steps that should share one board_id. " +
		"This tool records planned board steps and reads completed deliverables; it does not execute steps. " +
		"Execute steps with delegate/spawn using the same board_id and step_id, then use task_board results or list."
}

func (t *TaskBoardTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type":        "string",
				"enum":        []string{"create", "add_step", "list", "results"},
				"description": "Board operation. create returns a board_id; add_step records planned metadata; list/results inspect visible board records.",
			},
			"board_id": map[string]any{
				"type":        "string",
				"description": "Workflow/task-board ID. Optional for create; required for add_step/list/results.",
			},
			"title": map[string]any{
				"type":        "string",
				"description": "Readable board title for create.",
			},
			"description": map[string]any{
				"type":        "string",
				"description": "Optional board description for create.",
			},
			"step_id": map[string]any{
				"type":        "string",
				"description": "Stable step ID for add_step, for example media-extract.",
			},
			"step_title": map[string]any{
				"type":        "string",
				"description": "Readable step title for add_step.",
			},
			"owner": map[string]any{
				"type":        "string",
				"description": "Optional logical owner or target agent for this planned step.",
			},
			"task": map[string]any{
				"type":        "string",
				"description": "Optional task/instructions text for add_step.",
			},
			"depends_on": map[string]any{
				"type":        "array",
				"description": "Optional step/task IDs this step depends on.",
				"items": map[string]any{
					"type": "string",
				},
			},
			"blocked_by": map[string]any{
				"type":        "array",
				"description": "Optional step/task IDs currently blocking this step.",
				"items": map[string]any{
					"type": "string",
				},
			},
		},
		"required": []string{"action"},
	}
}

func (t *TaskBoardTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	if t == nil || t.registry == nil {
		return ErrorResult("task registry not configured")
	}
	action, err := requiredStringArg(args, "action", "action")
	if err != nil {
		return ErrorResult(err.Error())
	}
	switch action {
	case "create":
		return t.create(ctx, args)
	case "add_step":
		return t.addStep(ctx, args)
	case "list":
		return t.list(ctx, args, false)
	case "results":
		return t.list(ctx, args, true)
	default:
		return ErrorResult(`action must be one of: create, add_step, list, results`)
	}
}

func (t *TaskBoardTool) create(ctx context.Context, args map[string]any) *ToolResult {
	title, err := requiredStringArg(args, "title", "title")
	if err != nil {
		return ErrorResult(err.Error())
	}
	boardID, err := optionalStringArg(args, "board_id")
	if err != nil {
		return ErrorResult(err.Error())
	}
	if boardID == "" {
		boardID = generateTaskBoardID(title)
	}
	description, err := optionalStringArg(args, "description")
	if err != nil {
		return ErrorResult(err.Error())
	}
	now := time.Now().UnixMilli()
	rec := taskregistry.Record{
		TaskID:         boardRootTaskID(boardID),
		Runtime:        taskregistry.RuntimeTool,
		TaskKind:       "task_board",
		BoardID:        boardID,
		StepID:         "board-root",
		StepTitle:      title,
		Owner:          ToolAgentID(ctx),
		Channel:        ToolChannel(ctx),
		ChatID:         ToolChatID(ctx),
		TopicID:        ToolTopicID(ctx),
		AgentID:        ToolAgentID(ctx),
		Task:           firstNonEmpty(description, title),
		Status:         taskregistry.StatusPlanned,
		DeliveryStatus: taskregistry.DeliveryNotApplicable,
		NotifyPolicy:   taskregistry.NotifySilent,
		CreatedAt:      now,
		LastEventAt:    now,
	}
	if err := t.registry.Upsert(rec); err != nil {
		return ErrorResult(fmt.Sprintf("failed to create task board: %v", err)).WithError(err)
	}
	return taskBoardJSONResult(map[string]any{
		"action":   "create",
		"board_id": boardID,
		"task_id":  rec.TaskID,
		"title":    title,
		"status":   string(rec.Status),
	})
}

func (t *TaskBoardTool) addStep(ctx context.Context, args map[string]any) *ToolResult {
	boardID, err := requiredStringArg(args, "board_id", "board_id")
	if err != nil {
		return ErrorResult(err.Error())
	}
	stepID, err := requiredStringArg(args, "step_id", "step_id")
	if err != nil {
		return ErrorResult(err.Error())
	}
	stepTitle, err := requiredStringArg(args, "step_title", "step_title")
	if err != nil {
		return ErrorResult(err.Error())
	}
	task, err := optionalStringArg(args, "task")
	if err != nil {
		return ErrorResult(err.Error())
	}
	owner, err := optionalStringArg(args, "owner")
	if err != nil {
		return ErrorResult(err.Error())
	}
	dependsOn, err := optionalStringListArg(args, "depends_on")
	if err != nil {
		return ErrorResult(err.Error())
	}
	blockedBy, err := optionalStringListArg(args, "blocked_by")
	if err != nil {
		return ErrorResult(err.Error())
	}
	now := time.Now().UnixMilli()
	rec := taskregistry.Record{
		TaskID:         boardStepTaskID(boardID, stepID),
		Runtime:        taskregistry.RuntimeTool,
		TaskKind:       "task_board_step",
		BoardID:        boardID,
		ParentTaskID:   boardRootTaskID(boardID),
		StepID:         stepID,
		StepTitle:      stepTitle,
		Owner:          owner,
		DependsOn:      dependsOn,
		BlockedBy:      blockedBy,
		Channel:        ToolChannel(ctx),
		ChatID:         ToolChatID(ctx),
		TopicID:        ToolTopicID(ctx),
		AgentID:        ToolAgentID(ctx),
		Task:           firstNonEmpty(task, stepTitle),
		Status:         taskregistry.StatusPlanned,
		DeliveryStatus: taskregistry.DeliveryNotApplicable,
		NotifyPolicy:   taskregistry.NotifySilent,
		CreatedAt:      now,
		LastEventAt:    now,
	}
	if rec.Owner == "" {
		rec.Owner = rec.AgentID
	}
	if err := t.registry.Upsert(rec); err != nil {
		return ErrorResult(fmt.Sprintf("failed to add task board step: %v", err)).WithError(err)
	}
	return taskBoardJSONResult(map[string]any{
		"action":     "add_step",
		"board_id":   boardID,
		"task_id":    rec.TaskID,
		"step_id":    stepID,
		"step_title": stepTitle,
		"status":     string(rec.Status),
		"depends_on": dependsOn,
	})
}

func (t *TaskBoardTool) list(ctx context.Context, args map[string]any, resultsOnly bool) *ToolResult {
	boardID, err := requiredStringArg(args, "board_id", "board_id")
	if err != nil {
		return ErrorResult(err.Error())
	}
	channel := ToolChannel(ctx)
	chatID := ToolChatID(ctx)
	topicID := ToolTopicID(ctx)

	records := t.registry.ListBoard(boardID)
	filtered := make([]taskregistry.Record, 0, len(records))
	for _, rec := range records {
		if !taskRecordVisibleToCaller(rec, channel, chatID, topicID) {
			continue
		}
		if resultsOnly && rec.Deliverable == nil && rec.Completion == nil && strings.TrimSpace(rec.TerminalSummary) == "" {
			continue
		}
		filtered = append(filtered, rec)
	}
	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].StepID != filtered[j].StepID {
			return filtered[i].StepID < filtered[j].StepID
		}
		if filtered[i].CreatedAt != filtered[j].CreatedAt {
			return filtered[i].CreatedAt < filtered[j].CreatedAt
		}
		return filtered[i].TaskID < filtered[j].TaskID
	})

	counts := map[string]int{}
	steps := make([]taskBoardRecordView, 0, len(filtered))
	for _, rec := range filtered {
		counts[string(rec.Status)]++
		steps = append(steps, taskBoardView(rec, resultsOnly))
	}
	return taskBoardJSONResult(map[string]any{
		"action":       map[bool]string{true: "results", false: "list"}[resultsOnly],
		"board_id":     boardID,
		"count":        len(filtered),
		"counts":       counts,
		"results_only": resultsOnly,
		"steps":        steps,
	})
}

type taskBoardRecordView struct {
	TaskID         string                           `json:"task_id"`
	TaskKind       string                           `json:"task_kind,omitempty"`
	Status         string                           `json:"status"`
	DeliveryStatus string                           `json:"delivery_status,omitempty"`
	AgentID        string                           `json:"agent_id,omitempty"`
	Owner          string                           `json:"owner,omitempty"`
	StepID         string                           `json:"step_id,omitempty"`
	StepTitle      string                           `json:"step_title,omitempty"`
	DependsOn      []string                         `json:"depends_on,omitempty"`
	BlockedBy      []string                         `json:"blocked_by,omitempty"`
	Task           string                           `json:"task,omitempty"`
	CreatedAt      string                           `json:"created_at,omitempty"`
	EndedAt        string                           `json:"ended_at,omitempty"`
	Error          string                           `json:"error,omitempty"`
	Summary        string                           `json:"summary,omitempty"`
	Deliverable    *taskregistry.DeliverablePayload `json:"deliverable,omitempty"`
	Completion     *taskregistry.CompletionPayload  `json:"legacy_completion,omitempty"`
}

func taskBoardView(rec taskregistry.Record, includePayloads bool) taskBoardRecordView {
	view := taskBoardRecordView{
		TaskID:         rec.TaskID,
		TaskKind:       rec.TaskKind,
		Status:         string(rec.Status),
		DeliveryStatus: string(rec.DeliveryStatus),
		AgentID:        rec.AgentID,
		Owner:          rec.Owner,
		StepID:         rec.StepID,
		StepTitle:      rec.StepTitle,
		DependsOn:      rec.DependsOn,
		BlockedBy:      rec.BlockedBy,
		Task:           rec.Task,
		Error:          rec.Error,
		Summary:        rec.TerminalSummary,
	}
	if rec.CreatedAt > 0 {
		view.CreatedAt = time.UnixMilli(rec.CreatedAt).Format(time.RFC3339)
	}
	if rec.EndedAt > 0 {
		view.EndedAt = time.UnixMilli(rec.EndedAt).Format(time.RFC3339)
	}
	if includePayloads {
		view.Deliverable = rec.Deliverable
		view.Completion = rec.Completion
	}
	return view
}

func taskBoardJSONResult(value any) *ToolResult {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to encode task board result: %v", err)).WithError(err)
	}
	return NewToolResult(string(data))
}

func generateTaskBoardID(title string) string {
	base := slugifyTaskBoardID(title)
	if base == "" {
		base = "workflow"
	}
	return fmt.Sprintf("%s-%d", base, time.Now().UnixMilli())
}

func boardRootTaskID(boardID string) string {
	return "board:" + strings.TrimSpace(boardID)
}

func boardStepTaskID(boardID, stepID string) string {
	return "board:" + strings.TrimSpace(boardID) + ":step:" + strings.TrimSpace(stepID)
}

func slugifyTaskBoardID(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		ok := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if ok {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

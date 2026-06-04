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
// still execute work by calling delegate/spawn with the same board_id/step_id,
// or by marking a local/manual step with action=update.
type TaskBoardTool struct {
	registry *taskregistry.Registry
}

const taskBoardStalledAfter = 5 * time.Minute

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
		"Execute steps with delegate/spawn using the same board_id and step_id, or update local/manual steps when finished, then use task_board next/ready/results/list."
}

func (t *TaskBoardTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type":        "string",
				"enum":        []string{"create", "add_step", "update", "list", "results", "ready", "next"},
				"description": "Board operation. create returns a board_id; add_step records planned metadata; update changes a planned/local step status; list/results inspect visible board records; ready resolves dependency state; next returns a dry-run execution plan. ready/next do not execute steps.",
			},
			"board_id": map[string]any{
				"type":        "string",
				"description": "Workflow/task-board ID. Optional for create; required for add_step/list/results/ready/next.",
			},
			"title": map[string]any{
				"type":        "string",
				"description": "Readable board title for create.",
			},
			"description": map[string]any{
				"type":        "string",
				"description": "Optional board description for create.",
			},
			"task_packet": map[string]any{
				"type":        "object",
				"description": "Optional typed workflow contract for action=create. Use for serious/composite workflows. Requires objective when provided. Keep generic fields at top level and put domain-specific details under coding/media/research/nutrition.",
				"properties": map[string]any{
					"kind": map[string]any{
						"type":        "string",
						"description": "Workflow kind, for example generic, coding, media, research, nutrition.",
					},
					"objective": map[string]any{
						"type":        "string",
						"description": "Concise outcome this board must achieve.",
					},
					"scope": map[string]any{
						"type":        "string",
						"description": "Boundaries for the work.",
					},
					"acceptance_criteria": map[string]any{
						"type":  "array",
						"items": map[string]any{"type": "string"},
					},
					"verification_plan": map[string]any{
						"type":  "array",
						"items": map[string]any{"type": "string"},
					},
					"resources": map[string]any{
						"type": "array",
						"items": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"type":        map[string]any{"type": "string"},
								"uri":         map[string]any{"type": "string"},
								"description": map[string]any{"type": "string"},
								"metadata":    map[string]any{"type": "object"},
							},
						},
					},
					"constraints": map[string]any{
						"type":  "array",
						"items": map[string]any{"type": "string"},
					},
					"reporting": map[string]any{"type": "object"},
					"recovery":  map[string]any{"type": "object"},
					"coding":    map[string]any{"type": "object"},
					"media":     map[string]any{"type": "object"},
					"research":  map[string]any{"type": "object"},
					"nutrition": map[string]any{"type": "object"},
					"extra":     map[string]any{"type": "object"},
				},
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
			"status": map[string]any{
				"type": "string",
				"enum": []string{
					"planned",
					"queued",
					"running",
					"succeeded",
					"failed",
					"timed_out",
					"canceled",
					"lost",
				},
				"description": "New status for action=update.",
			},
			"summary": map[string]any{
				"type":        "string",
				"description": "Optional progress or terminal summary for action=update.",
			},
			"error": map[string]any{
				"type":        "string",
				"description": "Optional error text for failed/timed_out/canceled/lost updates.",
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
	case "update":
		return t.update(ctx, args)
	case "list":
		return t.list(ctx, args, false)
	case "results":
		return t.list(ctx, args, true)
	case "ready":
		return t.ready(ctx, args)
	case "next":
		return t.next(ctx, args)
	default:
		return ErrorResult(`action must be one of: create, add_step, update, list, results, ready, next`)
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
	taskPacket, err := optionalTaskPacketArg(args, "task_packet")
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
		TaskPacket:     taskPacket,
	}
	if err := t.registry.Upsert(rec); err != nil {
		return ErrorResult(fmt.Sprintf("failed to create task board: %v", err)).WithError(err)
	}
	return taskBoardJSONResult(map[string]any{
		"action":      "create",
		"board_id":    boardID,
		"task_id":     rec.TaskID,
		"title":       title,
		"status":      string(rec.Status),
		"task_packet": taskPacket,
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

func (t *TaskBoardTool) update(ctx context.Context, args map[string]any) *ToolResult {
	boardID, err := requiredStringArg(args, "board_id", "board_id")
	if err != nil {
		return ErrorResult(err.Error())
	}
	stepID, err := requiredStringArg(args, "step_id", "step_id")
	if err != nil {
		return ErrorResult(err.Error())
	}
	statusArg, err := requiredStringArg(args, "status", "status")
	if err != nil {
		return ErrorResult(err.Error())
	}
	status, err := parseTaskBoardStatus(statusArg)
	if err != nil {
		return ErrorResult(err.Error())
	}
	summary, err := optionalStringArg(args, "summary")
	if err != nil {
		return ErrorResult(err.Error())
	}
	errorText, err := optionalStringArg(args, "error")
	if err != nil {
		return ErrorResult(err.Error())
	}

	taskID := boardStepTaskID(boardID, stepID)
	now := time.Now().UnixMilli()
	if err := t.registry.Update(taskID, func(rec *taskregistry.Record) {
		rec.Status = status
		rec.LastEventAt = now
		if rec.StartedAt == 0 && (status == taskregistry.StatusRunning || isTaskBoardTerminalStatus(status)) {
			rec.StartedAt = now
		}
		if isTaskBoardTerminalStatus(status) {
			rec.EndedAt = now
			rec.DeliveryStatus = taskregistry.DeliveryNotApplicable
			rec.TerminalSummary = summary
		} else {
			rec.ProgressSummary = summary
		}
		rec.Error = errorText
		if len(args) > 0 {
			if blockedBy, err := optionalStringListArg(args, "blocked_by"); err == nil {
				rec.BlockedBy = blockedBy
			}
		}
	}); err != nil {
		return ErrorResult(fmt.Sprintf("failed to update task board step: %v", err)).WithError(err)
	}

	return taskBoardJSONResult(map[string]any{
		"action":   "update",
		"board_id": boardID,
		"task_id":  taskID,
		"step_id":  stepID,
		"status":   string(status),
		"summary":  summary,
		"error":    errorText,
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
		if resultsOnly && rec.Deliverable == nil && rec.Completion == nil &&
			strings.TrimSpace(rec.TerminalSummary) == "" && strings.TrimSpace(rec.Error) == "" {
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
	payload := map[string]any{
		"action":       map[bool]string{true: "results", false: "list"}[resultsOnly],
		"board_id":     boardID,
		"count":        len(filtered),
		"counts":       counts,
		"results_only": resultsOnly,
		"steps":        steps,
	}
	if resultsOnly {
		payload["step_results"] = buildTaskBoardStepResults(filtered)
	} else {
		effective := buildTaskBoardEffectiveView(filtered, time.Now(), taskBoardStalledAfter)
		payload["overall_status"] = effective.OverallStatus
		payload["effective_counts"] = effective.Counts
		payload["effective_steps"] = effective.Steps
	}
	return taskBoardJSONResult(payload)
}

func (t *TaskBoardTool) ready(ctx context.Context, args map[string]any) *ToolResult {
	boardID, err := requiredStringArg(args, "board_id", "board_id")
	if err != nil {
		return ErrorResult(err.Error())
	}
	records := t.visibleBoardRecords(ctx, boardID)
	return taskBoardJSONResult(buildTaskBoardReadyView(boardID, records, time.Now()))
}

func (t *TaskBoardTool) next(ctx context.Context, args map[string]any) *ToolResult {
	boardID, err := requiredStringArg(args, "board_id", "board_id")
	if err != nil {
		return ErrorResult(err.Error())
	}
	records := t.visibleBoardRecords(ctx, boardID)
	return taskBoardJSONResult(buildTaskBoardNextView(boardID, records, time.Now()))
}

func (t *TaskBoardTool) visibleBoardRecords(ctx context.Context, boardID string) []taskregistry.Record {
	channel := ToolChannel(ctx)
	chatID := ToolChatID(ctx)
	topicID := ToolTopicID(ctx)
	records := t.registry.ListBoard(boardID)
	filtered := make([]taskregistry.Record, 0, len(records))
	for _, rec := range records {
		if taskRecordVisibleToCaller(rec, channel, chatID, topicID) {
			filtered = append(filtered, rec)
		}
	}
	return filtered
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
	TaskPacket     *taskregistry.TaskPacketPayload  `json:"task_packet,omitempty"`
}

type taskBoardEffectiveView struct {
	OverallStatus string                       `json:"overall_status"`
	Counts        map[string]int               `json:"effective_counts"`
	Steps         []taskBoardEffectiveStepView `json:"effective_steps"`
}

type taskBoardEffectiveStepView struct {
	StepID              string `json:"step_id"`
	StepTitle           string `json:"step_title,omitempty"`
	Owner               string `json:"owner,omitempty"`
	EffectiveStatus     string `json:"effective_status"`
	Freshness           string `json:"freshness"`
	LatestTaskID        string `json:"latest_task_id,omitempty"`
	LatestRunTaskID     string `json:"latest_run_task_id,omitempty"`
	LatestRunStatus     string `json:"latest_run_status,omitempty"`
	LastEventAgeSeconds int64  `json:"last_event_age_seconds,omitempty"`
	Deliverable         bool   `json:"deliverable,omitempty"`
	Blocked             bool   `json:"blocked,omitempty"`
}

type taskBoardStepResultView struct {
	StepID                           string                           `json:"step_id"`
	StepTitle                        string                           `json:"step_title,omitempty"`
	Owner                            string                           `json:"owner,omitempty"`
	LatestTaskID                     string                           `json:"latest_task_id,omitempty"`
	LatestStatus                     string                           `json:"latest_status,omitempty"`
	LatestSuccessfulTaskID           string                           `json:"latest_successful_task_id,omitempty"`
	LatestSuccessfulEndedAt          string                           `json:"latest_successful_ended_at,omitempty"`
	LatestSuccessfulDeliverable      *taskregistry.DeliverablePayload `json:"latest_successful_deliverable,omitempty"`
	LatestSuccessfulLegacyCompletion *taskregistry.CompletionPayload  `json:"latest_successful_legacy_completion,omitempty"`
	LatestSuccessfulTerminalSummary  string                           `json:"latest_successful_terminal_summary,omitempty"`
	LatestFailureTaskID              string                           `json:"latest_failure_task_id,omitempty"`
	LatestFailureStatus              string                           `json:"latest_failure_status,omitempty"`
	LatestFailureError               string                           `json:"latest_failure_error,omitempty"`
	Deliverable                      *taskregistry.DeliverablePayload `json:"deliverable,omitempty"`
	LegacyCompletion                 *taskregistry.CompletionPayload  `json:"legacy_completion,omitempty"`
	TerminalSummary                  string                           `json:"terminal_summary,omitempty"`
	HasResult                        bool                             `json:"has_result"`
}

type taskBoardReadyView struct {
	Action       string                   `json:"action"`
	BoardID      string                   `json:"board_id"`
	Counts       map[string]int           `json:"counts"`
	ReadySteps   []taskBoardReadyStepView `json:"ready_steps"`
	WaitingSteps []taskBoardReadyStepView `json:"waiting_steps"`
	ActiveSteps  []taskBoardReadyStepView `json:"active_steps"`
	DoneSteps    []taskBoardReadyStepView `json:"done_steps"`
	BlockedSteps []taskBoardReadyStepView `json:"blocked_steps"`
}

type taskBoardReadyStepView struct {
	StepID          string   `json:"step_id"`
	StepTitle       string   `json:"step_title,omitempty"`
	Owner           string   `json:"owner,omitempty"`
	Status          string   `json:"status"`
	Freshness       string   `json:"freshness,omitempty"`
	LatestTaskID    string   `json:"latest_task_id,omitempty"`
	LatestRunTaskID string   `json:"latest_run_task_id,omitempty"`
	DependsOn       []string `json:"depends_on,omitempty"`
	BlockedBy       []string `json:"blocked_by,omitempty"`
	WaitingOn       []string `json:"waiting_on,omitempty"`
	MissingDeps     []string `json:"missing_dependencies,omitempty"`
	FailedDeps      []string `json:"failed_dependencies,omitempty"`
	Reason          string   `json:"reason"`
}

type taskBoardNextView struct {
	Action       string                   `json:"action"`
	BoardID      string                   `json:"board_id"`
	CanRun       bool                     `json:"can_run"`
	PlanCount    int                      `json:"plan_count"`
	Plan         []taskBoardNextStepView  `json:"plan"`
	BlockedSteps []taskBoardReadyStepView `json:"blocked_steps,omitempty"`
	WaitingSteps []taskBoardReadyStepView `json:"waiting_steps,omitempty"`
	ActiveSteps  []taskBoardReadyStepView `json:"active_steps,omitempty"`
}

type taskBoardNextStepView struct {
	StepID          string         `json:"step_id"`
	StepTitle       string         `json:"step_title,omitempty"`
	Owner           string         `json:"owner,omitempty"`
	Task            string         `json:"task,omitempty"`
	RecommendedTool string         `json:"recommended_tool"`
	Reason          string         `json:"reason"`
	DelegateArgs    map[string]any `json:"delegate_args,omitempty"`
	SpawnArgs       map[string]any `json:"spawn_args,omitempty"`
	UpdateArgs      map[string]any `json:"update_args,omitempty"`
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
		TaskPacket:     rec.TaskPacket,
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

func buildTaskBoardStepResults(records []taskregistry.Record) []taskBoardStepResultView {
	byStep := make(map[string][]taskregistry.Record)
	for _, rec := range records {
		if rec.StepID == "" || rec.StepID == "board-root" || rec.TaskKind == "task_board" ||
			rec.TaskKind == "task_board_step" {
			continue
		}
		byStep[rec.StepID] = append(byStep[rec.StepID], rec)
	}
	stepIDs := make([]string, 0, len(byStep))
	for stepID := range byStep {
		stepIDs = append(stepIDs, stepID)
	}
	sort.Strings(stepIDs)

	out := make([]taskBoardStepResultView, 0, len(stepIDs))
	for _, stepID := range stepIDs {
		out = append(out, taskBoardStepResultFromRecords(stepID, byStep[stepID]))
	}
	return out
}

func taskBoardStepResultFromRecords(stepID string, records []taskregistry.Record) taskBoardStepResultView {
	sort.Slice(records, func(i, j int) bool {
		if taskBoardRecordEventTime(records[i]) != taskBoardRecordEventTime(records[j]) {
			return taskBoardRecordEventTime(records[i]) < taskBoardRecordEventTime(records[j])
		}
		return records[i].TaskID < records[j].TaskID
	})

	latest := records[len(records)-1]
	view := taskBoardStepResultView{
		StepID:       stepID,
		StepTitle:    latest.StepTitle,
		Owner:        firstNonEmpty(latest.Owner, latest.AgentID),
		LatestTaskID: latest.TaskID,
		LatestStatus: string(latest.Status),
	}
	if latest.Status == taskregistry.StatusSucceeded && recordHasDeliverable(latest) {
		view.Deliverable = latest.Deliverable
		view.LegacyCompletion = latest.Completion
		view.TerminalSummary = latest.TerminalSummary
		view.HasResult = true
	}
	for i := len(records) - 1; i >= 0; i-- {
		rec := records[i]
		if view.LatestFailureTaskID == "" && isTaskBoardFailureStatus(rec.Status) {
			view.LatestFailureTaskID = rec.TaskID
			view.LatestFailureStatus = string(rec.Status)
			view.LatestFailureError = rec.Error
		}
		if view.LatestSuccessfulTaskID != "" ||
			rec.Status != taskregistry.StatusSucceeded ||
			!recordHasDeliverable(rec) {
			continue
		}
		view.StepTitle = firstNonEmpty(rec.StepTitle, view.StepTitle)
		view.Owner = firstNonEmpty(rec.Owner, rec.AgentID, view.Owner)
		view.LatestSuccessfulTaskID = rec.TaskID
		view.LatestSuccessfulEndedAt = formatTaskBoardTime(rec.EndedAt)
		view.LatestSuccessfulDeliverable = rec.Deliverable
		view.LatestSuccessfulLegacyCompletion = rec.Completion
		view.LatestSuccessfulTerminalSummary = rec.TerminalSummary
	}
	return view
}

func buildTaskBoardReadyView(boardID string, records []taskregistry.Record, now time.Time) taskBoardReadyView {
	byStep := taskBoardRecordsByStep(records, true)
	stepIDs := sortedTaskBoardStepIDs(byStep)
	effectiveByStep := make(map[string]taskBoardEffectiveStepView, len(stepIDs))
	metaByStep := make(map[string]taskregistry.Record, len(stepIDs))
	for _, stepID := range stepIDs {
		effectiveByStep[stepID] = effectiveStepFromRecords(stepID, byStep[stepID], now, taskBoardStalledAfter)
		metaByStep[stepID] = latestTaskBoardStepMetadata(byStep[stepID])
	}

	view := taskBoardReadyView{
		Action:  "ready",
		BoardID: boardID,
		Counts:  map[string]int{},
	}
	for _, stepID := range stepIDs {
		step := taskBoardReadyStep(stepID, effectiveByStep, metaByStep)
		switch step.Reason {
		case "ready":
			view.ReadySteps = append(view.ReadySteps, step)
		case "done":
			view.DoneSteps = append(view.DoneSteps, step)
		case "active":
			view.ActiveSteps = append(view.ActiveSteps, step)
		case "blocked":
			view.BlockedSteps = append(view.BlockedSteps, step)
		default:
			view.WaitingSteps = append(view.WaitingSteps, step)
		}
		view.Counts[step.Reason]++
	}
	return view
}

func buildTaskBoardNextView(boardID string, records []taskregistry.Record, now time.Time) taskBoardNextView {
	ready := buildTaskBoardReadyView(boardID, records, now)
	byStep := taskBoardRecordsByStep(records, true)
	metaByStep := make(map[string]taskregistry.Record, len(byStep))
	for stepID, stepRecords := range byStep {
		metaByStep[stepID] = latestTaskBoardStepMetadata(stepRecords)
	}

	view := taskBoardNextView{
		Action:       "next",
		BoardID:      boardID,
		CanRun:       len(ready.ReadySteps) > 0,
		BlockedSteps: ready.BlockedSteps,
		WaitingSteps: ready.WaitingSteps,
		ActiveSteps:  ready.ActiveSteps,
	}
	for _, step := range ready.ReadySteps {
		view.Plan = append(view.Plan, taskBoardNextStep(boardID, step, metaByStep[step.StepID]))
	}
	view.PlanCount = len(view.Plan)
	return view
}

func taskBoardNextStep(
	boardID string,
	step taskBoardReadyStepView,
	meta taskregistry.Record,
) taskBoardNextStepView {
	owner := firstNonEmpty(step.Owner, meta.Owner, meta.AgentID)
	task := firstNonEmpty(meta.Task, step.StepTitle, step.StepID)
	out := taskBoardNextStepView{
		StepID:    step.StepID,
		StepTitle: step.StepTitle,
		Owner:     owner,
		Task:      task,
		Reason:    "step dependencies are satisfied and no active run is recorded",
	}
	baseArgs := map[string]any{
		"board_id":   boardID,
		"step_id":    step.StepID,
		"step_title": step.StepTitle,
		"task":       task,
	}
	if owner != "" {
		out.RecommendedTool = "delegate"
		out.DelegateArgs = copyStringAnyMap(baseArgs)
		out.DelegateArgs["agent_id"] = owner
		out.SpawnArgs = copyStringAnyMap(baseArgs)
		out.SpawnArgs["agent_id"] = owner
		out.SpawnArgs["label"] = firstNonEmpty(step.StepTitle, step.StepID)
		if len(step.DependsOn) > 0 {
			out.DelegateArgs["depends_on"] = append([]string(nil), step.DependsOn...)
			out.SpawnArgs["depends_on"] = append([]string(nil), step.DependsOn...)
		}
		return out
	}
	out.RecommendedTool = "task_board.update"
	out.UpdateArgs = map[string]any{
		"action":   "update",
		"board_id": boardID,
		"step_id":  step.StepID,
		"status":   string(taskregistry.StatusRunning),
		"summary":  "manual/local step started",
	}
	return out
}

func taskBoardReadyStep(
	stepID string,
	effectiveByStep map[string]taskBoardEffectiveStepView,
	metaByStep map[string]taskregistry.Record,
) taskBoardReadyStepView {
	effective := effectiveByStep[stepID]
	meta := metaByStep[stepID]
	step := taskBoardReadyStepView{
		StepID:          stepID,
		StepTitle:       firstNonEmpty(effective.StepTitle, meta.StepTitle),
		Owner:           firstNonEmpty(effective.Owner, meta.Owner, meta.AgentID),
		Status:          effective.EffectiveStatus,
		Freshness:       effective.Freshness,
		LatestTaskID:    effective.LatestTaskID,
		LatestRunTaskID: effective.LatestRunTaskID,
		DependsOn:       append([]string(nil), meta.DependsOn...),
		BlockedBy:       append([]string(nil), meta.BlockedBy...),
	}
	for _, dep := range step.DependsOn {
		depStep, ok := effectiveByStep[dep]
		if !ok {
			step.MissingDeps = append(step.MissingDeps, dep)
			continue
		}
		switch depStep.EffectiveStatus {
		case string(taskregistry.StatusSucceeded):
		case string(taskregistry.StatusFailed),
			string(taskregistry.StatusTimedOut),
			string(taskregistry.StatusCancelled),
			string(taskregistry.StatusLost),
			"blocked":
			step.FailedDeps = append(step.FailedDeps, dep)
		default:
			step.WaitingOn = append(step.WaitingOn, dep)
		}
	}
	step.Reason = taskBoardReadyReason(step)
	return step
}

func taskBoardReadyReason(step taskBoardReadyStepView) string {
	if len(step.BlockedBy) > 0 || len(step.FailedDeps) > 0 ||
		taskBoardReadyStatusIsFailure(step.Status) {
		return "blocked"
	}
	if step.Status == string(taskregistry.StatusSucceeded) {
		if len(step.MissingDeps) > 0 || len(step.WaitingOn) > 0 {
			return "blocked"
		}
		return "done"
	}
	if step.Status == string(taskregistry.StatusRunning) || step.Status == string(taskregistry.StatusQueued) {
		return "active"
	}
	if len(step.WaitingOn) > 0 || len(step.MissingDeps) > 0 {
		return "waiting"
	}
	return "ready"
}

func taskBoardReadyStatusIsFailure(status string) bool {
	switch status {
	case string(taskregistry.StatusFailed),
		string(taskregistry.StatusTimedOut),
		string(taskregistry.StatusCancelled),
		string(taskregistry.StatusLost),
		"blocked":
		return true
	default:
		return false
	}
}

func buildTaskBoardEffectiveView(
	records []taskregistry.Record,
	now time.Time,
	stalledAfter time.Duration,
) taskBoardEffectiveView {
	byStep := taskBoardRecordsByStep(records, false)
	stepIDs := sortedTaskBoardStepIDs(byStep)

	counts := map[string]int{}
	steps := make([]taskBoardEffectiveStepView, 0, len(stepIDs))
	for _, stepID := range stepIDs {
		step := effectiveStepFromRecords(stepID, byStep[stepID], now, stalledAfter)
		counts[step.EffectiveStatus]++
		steps = append(steps, step)
	}

	return taskBoardEffectiveView{
		OverallStatus: effectiveBoardOverallStatus(steps),
		Counts:        counts,
		Steps:         steps,
	}
}

func taskBoardRecordsByStep(
	records []taskregistry.Record,
	includeOnlyPlannedStepsAndRuns bool,
) map[string][]taskregistry.Record {
	byStep := make(map[string][]taskregistry.Record)
	for _, rec := range records {
		if rec.StepID == "" || rec.StepID == "board-root" || rec.TaskKind == "task_board" {
			continue
		}
		if includeOnlyPlannedStepsAndRuns && rec.TaskKind == "" {
			continue
		}
		byStep[rec.StepID] = append(byStep[rec.StepID], rec)
	}
	return byStep
}

func sortedTaskBoardStepIDs(byStep map[string][]taskregistry.Record) []string {
	stepIDs := make([]string, 0, len(byStep))
	for stepID := range byStep {
		stepIDs = append(stepIDs, stepID)
	}
	sort.Strings(stepIDs)
	return stepIDs
}

func latestTaskBoardStepMetadata(records []taskregistry.Record) taskregistry.Record {
	sorted := append([]taskregistry.Record(nil), records...)
	sort.Slice(sorted, func(i, j int) bool {
		if taskBoardRecordEventTime(sorted[i]) != taskBoardRecordEventTime(sorted[j]) {
			return taskBoardRecordEventTime(sorted[i]) < taskBoardRecordEventTime(sorted[j])
		}
		return sorted[i].TaskID < sorted[j].TaskID
	})
	for i := len(sorted) - 1; i >= 0; i-- {
		if sorted[i].TaskKind == "task_board_step" {
			return sorted[i]
		}
	}
	if len(sorted) == 0 {
		return taskregistry.Record{}
	}
	return sorted[len(sorted)-1]
}

func effectiveStepFromRecords(
	stepID string,
	records []taskregistry.Record,
	now time.Time,
	stalledAfter time.Duration,
) taskBoardEffectiveStepView {
	sort.Slice(records, func(i, j int) bool {
		if taskBoardRecordEventTime(records[i]) != taskBoardRecordEventTime(records[j]) {
			return taskBoardRecordEventTime(records[i]) < taskBoardRecordEventTime(records[j])
		}
		return records[i].TaskID < records[j].TaskID
	})

	latest := records[len(records)-1]
	latestRun := latestNonBoardStepRecord(records)
	statusSource := latest
	if latestRun != nil {
		statusSource = *latestRun
	}

	status := string(statusSource.Status)
	freshness, ageSeconds := taskBoardFreshness(statusSource, now, stalledAfter)
	view := taskBoardEffectiveStepView{
		StepID:              stepID,
		StepTitle:           firstNonEmpty(statusSource.StepTitle, latest.StepTitle),
		Owner:               firstNonEmpty(statusSource.Owner, statusSource.AgentID, latest.Owner, latest.AgentID),
		EffectiveStatus:     status,
		Freshness:           freshness,
		LatestTaskID:        latest.TaskID,
		LastEventAgeSeconds: ageSeconds,
		Deliverable:         recordHasDeliverable(statusSource),
		Blocked:             len(statusSource.BlockedBy) > 0,
	}
	if latestRun != nil {
		view.LatestRunTaskID = latestRun.TaskID
		view.LatestRunStatus = string(latestRun.Status)
	}
	if view.Blocked && view.EffectiveStatus == string(taskregistry.StatusPlanned) {
		view.EffectiveStatus = "blocked"
	}
	return view
}

func latestNonBoardStepRecord(records []taskregistry.Record) *taskregistry.Record {
	for i := len(records) - 1; i >= 0; i-- {
		if records[i].TaskKind != "task_board_step" {
			return &records[i]
		}
	}
	return nil
}

func taskBoardFreshness(
	rec taskregistry.Record,
	now time.Time,
	stalledAfter time.Duration,
) (string, int64) {
	ref := taskBoardRecordEventTime(rec)
	var ageSeconds int64
	if ref > 0 {
		ageSeconds = int64(now.Sub(time.UnixMilli(ref)).Seconds())
		if ageSeconds < 0 {
			ageSeconds = 0
		}
	}
	switch rec.Status {
	case taskregistry.StatusSucceeded,
		taskregistry.StatusFailed,
		taskregistry.StatusTimedOut,
		taskregistry.StatusCancelled:
		return "finished", ageSeconds
	case taskregistry.StatusLost:
		return "lost", ageSeconds
	case taskregistry.StatusRunning:
		if ref > 0 && now.Sub(time.UnixMilli(ref)) > stalledAfter {
			return "stalled", ageSeconds
		}
		return "healthy", ageSeconds
	case taskregistry.StatusQueued:
		if ref > 0 && now.Sub(time.UnixMilli(ref)) > stalledAfter {
			return "stalled", ageSeconds
		}
		return "healthy", ageSeconds
	default:
		return "unknown", ageSeconds
	}
}

func effectiveBoardOverallStatus(steps []taskBoardEffectiveStepView) string {
	if len(steps) == 0 {
		return "empty"
	}
	allSucceeded := true
	hasActive := false
	hasPlanned := false
	hasBlocked := false
	hasStalled := false
	for _, step := range steps {
		switch step.EffectiveStatus {
		case string(taskregistry.StatusFailed),
			string(taskregistry.StatusTimedOut),
			string(taskregistry.StatusCancelled),
			string(taskregistry.StatusLost):
			return step.EffectiveStatus
		case "blocked":
			hasBlocked = true
			allSucceeded = false
		case string(taskregistry.StatusRunning), string(taskregistry.StatusQueued):
			hasActive = true
			allSucceeded = false
		case string(taskregistry.StatusPlanned):
			hasPlanned = true
			allSucceeded = false
		case string(taskregistry.StatusSucceeded):
		default:
			allSucceeded = false
		}
		if step.Freshness == "stalled" {
			hasStalled = true
		}
	}
	if hasStalled {
		return "stalled"
	}
	if hasBlocked {
		return "blocked"
	}
	if hasActive {
		return string(taskregistry.StatusRunning)
	}
	if allSucceeded {
		return string(taskregistry.StatusSucceeded)
	}
	if hasPlanned {
		return string(taskregistry.StatusPlanned)
	}
	return "unknown"
}

func taskBoardRecordEventTime(rec taskregistry.Record) int64 {
	switch {
	case rec.LastEventAt > 0:
		return rec.LastEventAt
	case rec.EndedAt > 0:
		return rec.EndedAt
	case rec.StartedAt > 0:
		return rec.StartedAt
	default:
		return rec.CreatedAt
	}
}

func recordHasDeliverable(rec taskregistry.Record) bool {
	if rec.Deliverable != nil {
		return strings.TrimSpace(rec.Deliverable.Text) != "" || len(rec.Deliverable.Artifacts) > 0
	}
	if rec.Completion != nil {
		return strings.TrimSpace(rec.Completion.Text) != "" || len(rec.Completion.Media) > 0
	}
	return strings.TrimSpace(rec.TerminalSummary) != ""
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

func copyStringAnyMap(values map[string]any) map[string]any {
	out := make(map[string]any, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}

func parseTaskBoardStatus(value string) (taskregistry.Status, error) {
	switch taskregistry.Status(strings.TrimSpace(value)) {
	case taskregistry.StatusPlanned:
		return taskregistry.StatusPlanned, nil
	case taskregistry.StatusQueued:
		return taskregistry.StatusQueued, nil
	case taskregistry.StatusRunning:
		return taskregistry.StatusRunning, nil
	case taskregistry.StatusSucceeded:
		return taskregistry.StatusSucceeded, nil
	case taskregistry.StatusFailed:
		return taskregistry.StatusFailed, nil
	case taskregistry.StatusTimedOut:
		return taskregistry.StatusTimedOut, nil
	case "canceled":
		return taskregistry.StatusCancelled, nil
	case taskregistry.StatusCancelled:
		return taskregistry.StatusCancelled, nil
	case taskregistry.StatusLost:
		return taskregistry.StatusLost, nil
	default:
		return "", fmt.Errorf(
			"status must be one of: planned, queued, running, succeeded, failed, timed_out, canceled, lost",
		)
	}
}

func isTaskBoardTerminalStatus(status taskregistry.Status) bool {
	switch status {
	case taskregistry.StatusSucceeded,
		taskregistry.StatusFailed,
		taskregistry.StatusTimedOut,
		taskregistry.StatusCancelled,
		taskregistry.StatusLost:
		return true
	default:
		return false
	}
}

func isTaskBoardFailureStatus(status taskregistry.Status) bool {
	switch status {
	case taskregistry.StatusFailed,
		taskregistry.StatusTimedOut,
		taskregistry.StatusCancelled,
		taskregistry.StatusLost:
		return true
	default:
		return false
	}
}

func formatTaskBoardTime(value int64) string {
	if value <= 0 {
		return ""
	}
	return time.UnixMilli(value).Format(time.RFC3339)
}

func optionalTaskPacketArg(args map[string]any, key string) (*taskregistry.TaskPacketPayload, error) {
	raw, ok := args[key]
	if !ok || raw == nil {
		return nil, nil
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("%s must be an object: %w", key, err)
	}
	var packet taskregistry.TaskPacketPayload
	if err := json.Unmarshal(data, &packet); err != nil {
		return nil, fmt.Errorf("%s must match the task packet schema: %w", key, err)
	}
	packet.Kind = strings.TrimSpace(packet.Kind)
	packet.Objective = strings.TrimSpace(packet.Objective)
	packet.Scope = strings.TrimSpace(packet.Scope)
	packet.AcceptanceCriteria = cleanStringList(packet.AcceptanceCriteria)
	packet.VerificationPlan = cleanStringList(packet.VerificationPlan)
	packet.Constraints = cleanStringList(packet.Constraints)
	if packet.Objective == "" {
		return nil, fmt.Errorf("%s.objective required when task_packet is provided", key)
	}
	packet.Resources = cleanTaskPacketResources(packet.Resources)
	return &packet, nil
}

func cleanTaskPacketResources(resources []taskregistry.TaskPacketResource) []taskregistry.TaskPacketResource {
	out := make([]taskregistry.TaskPacketResource, 0, len(resources))
	for _, resource := range resources {
		resource.Type = strings.TrimSpace(resource.Type)
		resource.URI = strings.TrimSpace(resource.URI)
		resource.Description = strings.TrimSpace(resource.Description)
		if resource.Type == "" && resource.URI == "" && resource.Description == "" && len(resource.Metadata) == 0 {
			continue
		}
		out = append(out, resource)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

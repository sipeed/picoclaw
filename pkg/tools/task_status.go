package tools

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	taskregistry "github.com/sipeed/picoclaw/pkg/tasks"
)

const taskStatusActiveStaleAfter = 30 * time.Minute

// TaskStatusTool reports durable runtime task/run records across spawn,
// delegate, cron, and future background runtimes.
type TaskStatusTool struct {
	registry *taskregistry.Registry
}

func NewTaskStatusTool(registry *taskregistry.Registry) *TaskStatusTool {
	return &TaskStatusTool{registry: registry}
}

func (t *TaskStatusTool) Name() string {
	return "task_status"
}

func (t *TaskStatusTool) Description() string {
	return "Get durable runtime task status for spawn/delegate/cron/subtask runs. " +
		"Prefer this for general task history, completed task checks, and after service restarts. " +
		"Use this instead of spawn_status when the task may have used delegate or another child-run mechanism. " +
		"Results are scoped to the current conversation's channel/chat when available."
}

func (t *TaskStatusTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"task_id": map[string]any{
				"type":        "string",
				"description": "Optional durable task ID, e.g. subagent-1 or delegate-...",
			},
			"task_kind": map[string]any{
				"type":        "string",
				"description": "Optional task kind filter, e.g. spawn or delegate.",
			},
			"board_id": map[string]any{
				"type":        "string",
				"description": "Optional task board/workflow ID. Use this to inspect all steps belonging to one durable workflow.",
			},
			"include_events": map[string]any{
				"type":        "boolean",
				"description": "When task_id is provided, include the typed task event stream for that task.",
			},
		},
		"required": []string{},
	}
}

func (t *TaskStatusTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	if t == nil || t.registry == nil {
		return ErrorResult("task registry not configured")
	}
	if _, err := t.registry.MarkStaleActiveLost(
		taskStatusActiveStaleAfter,
		"active task did not report progress before task_status stale timeout",
	); err != nil {
		return ErrorResult(fmt.Sprintf("failed to reconcile stale active tasks: %v", err)).WithError(err)
	}
	taskID, err := optionalTaskStatusStringArg(args, "task_id")
	if err != nil {
		return ErrorResult(err.Error())
	}
	taskKind, err := optionalTaskStatusStringArg(args, "task_kind")
	if err != nil {
		return ErrorResult(err.Error())
	}
	boardID, err := optionalTaskStatusStringArg(args, "board_id")
	if err != nil {
		return ErrorResult(err.Error())
	}
	includeEvents, err := optionalTaskStatusBoolArg(args, "include_events")
	if err != nil {
		return ErrorResult(err.Error())
	}
	callerChannel := ToolChannel(ctx)
	callerChatID := ToolChatID(ctx)
	callerTopicID := ToolTopicID(ctx)

	if taskID != "" {
		rec, ok := t.registry.Get(taskID)
		if !ok || !taskRecordVisibleToCaller(rec, callerChannel, callerChatID, callerTopicID) {
			return ErrorResult(fmt.Sprintf("No task found with task ID: %s", taskID))
		}
		out := formatTaskRecord(rec)
		if includeEvents {
			out = out + "\n" + formatTaskEvents(t.registry.ListEvents(taskID))
		}
		return NewToolResult(out)
	}

	records := t.registry.List()
	filtered := make([]taskregistry.Record, 0, len(records))
	for _, rec := range records {
		if taskKind != "" && rec.TaskKind != taskKind {
			continue
		}
		if boardID != "" && rec.BoardID != boardID {
			continue
		}
		if !taskRecordVisibleToCaller(rec, callerChannel, callerChatID, callerTopicID) {
			continue
		}
		filtered = append(filtered, rec)
	}
	if len(filtered) == 0 {
		if boardID != "" {
			return NewToolResult(fmt.Sprintf("No visible tasks found for board_id %q.", boardID))
		}
		if taskKind != "" {
			return NewToolResult(fmt.Sprintf("No visible tasks found for task_kind %q.", taskKind))
		}
		return NewToolResult("No visible durable tasks are registered for this conversation.")
	}

	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].CreatedAt != filtered[j].CreatedAt {
			return filtered[i].CreatedAt < filtered[j].CreatedAt
		}
		return filtered[i].TaskID < filtered[j].TaskID
	})

	counts := map[taskregistry.Status]int{}
	for _, rec := range filtered {
		counts[rec.Status]++
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Task status report (%d total):\n", len(filtered)))
	for _, status := range []taskregistry.Status{
		taskregistry.StatusPlanned,
		taskregistry.StatusQueued,
		taskregistry.StatusRunning,
		taskregistry.StatusSucceeded,
		taskregistry.StatusFailed,
		taskregistry.StatusTimedOut,
		taskregistry.StatusCancelled,
		taskregistry.StatusLost,
	} {
		if n := counts[status]; n > 0 {
			sb.WriteString(fmt.Sprintf("  %-10s %d\n", status+":", n))
		}
	}
	sb.WriteString("\n")
	for _, rec := range filtered {
		sb.WriteString(formatTaskRecord(rec))
		sb.WriteString("\n")
	}
	return NewToolResult(strings.TrimSpace(sb.String()))
}

func optionalTaskStatusBoolArg(args map[string]any, key string) (bool, error) {
	raw, ok := args[key]
	if !ok || raw == nil {
		return false, nil
	}
	value, ok := raw.(bool)
	if !ok {
		return false, fmt.Errorf("%s must be a boolean", key)
	}
	return value, nil
}

func optionalTaskStatusStringArg(args map[string]any, key string) (string, error) {
	raw, ok := args[key]
	if !ok || raw == nil {
		return "", nil
	}
	value, ok := raw.(string)
	if !ok {
		return "", fmt.Errorf("%s must be a string", key)
	}
	return strings.TrimSpace(value), nil
}

func taskRecordVisibleToCaller(rec taskregistry.Record, channel, chatID, topicID string) bool {
	if channel != "" && rec.Channel != "" && rec.Channel != channel {
		return false
	}
	if chatID != "" && rec.ChatID != "" && rec.ChatID != chatID {
		return false
	}
	if topicID != "" && rec.TopicID != "" && rec.TopicID != topicID {
		return false
	}
	return true
}

func formatTaskRecord(rec taskregistry.Record) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Task %s [%s/%s]\n", rec.TaskID, rec.Runtime, rec.TaskKind))
	sb.WriteString(fmt.Sprintf("  Status: %s\n", rec.Status))
	sb.WriteString(fmt.Sprintf("  Delivery: %s", rec.DeliveryStatus))
	if rec.DeliveryMode != "" {
		sb.WriteString(fmt.Sprintf(" (%s)", rec.DeliveryMode))
	}
	sb.WriteString("\n")
	if rec.AgentID != "" {
		sb.WriteString(fmt.Sprintf("  Agent: %s\n", rec.AgentID))
	}
	if rec.BoardID != "" || rec.ParentTaskID != "" || rec.Owner != "" {
		sb.WriteString("  Board:")
		if rec.BoardID != "" {
			sb.WriteString(fmt.Sprintf(" board_id=%s", rec.BoardID))
		}
		if rec.ParentTaskID != "" {
			sb.WriteString(fmt.Sprintf(" parent=%s", rec.ParentTaskID))
		}
		if rec.Owner != "" {
			sb.WriteString(fmt.Sprintf(" owner=%s", rec.Owner))
		}
		sb.WriteString("\n")
	}
	if rec.StepTitle != "" {
		sb.WriteString(fmt.Sprintf("  Step: %s", rec.StepTitle))
		if rec.StepID != "" && rec.StepID != rec.TaskID {
			sb.WriteString(fmt.Sprintf(" (%s)", rec.StepID))
		}
		sb.WriteString("\n")
	}
	if len(rec.DependsOn) > 0 {
		sb.WriteString(fmt.Sprintf("  Depends on: %s\n", strings.Join(rec.DependsOn, ", ")))
	}
	if len(rec.BlockedBy) > 0 {
		sb.WriteString(fmt.Sprintf("  Blocked by: %s\n", strings.Join(rec.BlockedBy, ", ")))
	}
	if rec.Channel != "" || rec.ChatID != "" || rec.TopicID != "" {
		sb.WriteString(fmt.Sprintf("  Scope: %s/%s", rec.Channel, rec.ChatID))
		if rec.TopicID != "" {
			sb.WriteString(fmt.Sprintf(" topic=%s", rec.TopicID))
		}
		sb.WriteString("\n")
	}
	if rec.CreatedAt > 0 {
		sb.WriteString(fmt.Sprintf("  Created: %s\n", formatTaskTime(rec.CreatedAt)))
	}
	if rec.EndedAt > 0 {
		sb.WriteString(fmt.Sprintf("  Ended: %s\n", formatTaskTime(rec.EndedAt)))
	}
	if rec.Task != "" {
		sb.WriteString(fmt.Sprintf("  Task: %s\n", truncateTaskText(rec.Task, 240)))
	}
	if rec.TerminalSummary != "" {
		sb.WriteString(fmt.Sprintf("  Result: %s\n", truncateTaskText(rec.TerminalSummary, 500)))
	}
	if rec.Error != "" {
		sb.WriteString(fmt.Sprintf("  Error: %s\n", truncateTaskText(rec.Error, 500)))
	}
	if rec.Deliverable != nil {
		sb.WriteString(
			fmt.Sprintf(
				"  Deliverable: text=%t artifacts=%d report=%t\n",
				rec.Deliverable.Text != "",
				len(rec.Deliverable.Artifacts),
				rec.Deliverable.Report != nil,
			),
		)
	}
	if rec.Completion != nil && rec.Deliverable == nil {
		sb.WriteString(
			fmt.Sprintf(
				"  Legacy completion: text=%t media=%d\n",
				rec.Completion.Text != "",
				len(rec.Completion.Media),
			),
		)
	}
	return strings.TrimRight(sb.String(), "\n")
}

func formatTaskEvents(events []taskregistry.TaskEvent) string {
	if len(events) == 0 {
		return "Events: none"
	}
	var sb strings.Builder
	sb.WriteString("Events:\n")
	for _, evt := range events {
		sb.WriteString(fmt.Sprintf(
			"  #%d %s status=%s delivery=%s at=%s",
			evt.Seq,
			evt.Type,
			evt.Status,
			evt.DeliveryStatus,
			formatTaskTime(evt.EmittedAt),
		))
		if evt.Fingerprint != "" {
			sb.WriteString(fmt.Sprintf(" fingerprint=%s", truncateTaskText(evt.Fingerprint, 12)))
		}
		if len(evt.Payload) > 0 {
			sb.WriteString(fmt.Sprintf(" payload=%s", formatTaskEventPayload(evt.Payload)))
		}
		sb.WriteString("\n")
	}
	return strings.TrimRight(sb.String(), "\n")
}

func formatTaskEventPayload(payload map[string]string) string {
	if len(payload) == 0 {
		return "{}"
	}
	keys := make([]string, 0, len(payload))
	for key := range payload {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%q", key, payload[key]))
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

func formatTaskTime(ms int64) string {
	return time.UnixMilli(ms).Format(time.RFC3339)
}

func truncateTaskText(s string, limit int) string {
	s = strings.TrimSpace(s)
	if limit <= 0 || len(s) <= limit {
		return s
	}
	return s[:limit] + "..."
}

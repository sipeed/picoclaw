package tools

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	taskregistry "github.com/sipeed/picoclaw/pkg/tasks"
)

// TaskStatusTool reports durable runtime task/run records across spawn,
// delegate, and future background runtimes.
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
	return "Get durable runtime task status for spawn/delegate/subtask runs. " +
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
		},
		"required": []string{},
	}
}

func (t *TaskStatusTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	if t == nil || t.registry == nil {
		return ErrorResult("task registry not configured")
	}
	taskID, err := optionalTaskStatusStringArg(args, "task_id")
	if err != nil {
		return ErrorResult(err.Error())
	}
	taskKind, err := optionalTaskStatusStringArg(args, "task_kind")
	if err != nil {
		return ErrorResult(err.Error())
	}
	callerChannel := ToolChannel(ctx)
	callerChatID := ToolChatID(ctx)

	if taskID != "" {
		rec, ok := t.registry.Get(taskID)
		if !ok || !taskRecordVisibleToCaller(rec, callerChannel, callerChatID) {
			return ErrorResult(fmt.Sprintf("No task found with task ID: %s", taskID))
		}
		return NewToolResult(formatTaskRecord(rec))
	}

	records := t.registry.List()
	filtered := make([]taskregistry.Record, 0, len(records))
	for _, rec := range records {
		if taskKind != "" && rec.TaskKind != taskKind {
			continue
		}
		if !taskRecordVisibleToCaller(rec, callerChannel, callerChatID) {
			continue
		}
		filtered = append(filtered, rec)
	}
	if len(filtered) == 0 {
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

func taskRecordVisibleToCaller(rec taskregistry.Record, channel, chatID string) bool {
	if channel != "" && rec.Channel != "" && rec.Channel != channel {
		return false
	}
	if chatID != "" && rec.ChatID != "" && rec.ChatID != chatID {
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
	if rec.Completion != nil {
		sb.WriteString(fmt.Sprintf("  Completion: text=%t media=%d\n", rec.Completion.Text != "", len(rec.Completion.Media)))
	}
	return strings.TrimRight(sb.String(), "\n")
}

func formatTaskTime(ms int64) string {
	return time.UnixMilli(ms).Format(time.RFC3339)
}

func truncateTaskText(s string, max int) string {
	s = strings.TrimSpace(s)
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

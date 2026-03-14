package tools

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
)

// SpawnStatusTool reports the status of subagents that were spawned via the
// spawn tool. It can query a specific task by ID, or list every known task with
// a summary count broken-down by status.
type SpawnStatusTool struct {
	manager *SubagentManager
}

// NewSpawnStatusTool creates a SpawnStatusTool backed by the given manager.
func NewSpawnStatusTool(manager *SubagentManager) *SpawnStatusTool {
	return &SpawnStatusTool{manager: manager}
}

func (t *SpawnStatusTool) Name() string {
	return "spawn_status"
}

func (t *SpawnStatusTool) Description() string {
	return "Get the status of spawned subagents. " +
		"Returns a list of all subagents and their current state " +
		"(running, completed, failed, or canceled), or retrieves details " +
		"for a specific subagent task when task_id is provided."
}

func (t *SpawnStatusTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"task_id": map[string]any{
				"type": "string",
				"description": "Optional task ID (e.g. \"subagent-1\") to inspect a specific " +
					"subagent. When omitted, all known subagents are listed.",
			},
		},
		"required": []string{},
	}
}

func (t *SpawnStatusTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	if t.manager == nil {
		return ErrorResult("Subagent manager not configured")
	}

	taskID, _ := args["task_id"].(string)
	taskID = strings.TrimSpace(taskID)

	if taskID != "" {
		task, ok := t.manager.GetTask(taskID)
		if !ok {
			return ErrorResult(fmt.Sprintf("No subagent found with task ID: %s", taskID))
		}
		return NewToolResult(spawnStatusFormatTask(task))
	}

	tasks := t.manager.ListTasks()
	if len(tasks) == 0 {
		return NewToolResult("No subagents have been spawned yet.")
	}

	// Deterministic ordering: sort by ID string (e.g. "subagent-1" < "subagent-2").
	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].ID < tasks[j].ID
	})

	counts := map[string]int{}
	for _, task := range tasks {
		counts[task.Status]++
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Subagent status report (%d total):\n", len(tasks)))
	for _, status := range []string{"running", "completed", "failed", "canceled"} {
		if n := counts[status]; n > 0 {
			label := strings.ToUpper(status[:1]) + status[1:] + ":"
			sb.WriteString(fmt.Sprintf("  %-10s %d\n", label, n))
		}
	}
	sb.WriteString("\n")

	for _, task := range tasks {
		sb.WriteString(spawnStatusFormatTask(task))
		sb.WriteString("\n\n")
	}

	return NewToolResult(strings.TrimRight(sb.String(), "\n"))
}

// spawnStatusFormatTask renders a single SubagentTask as a human-readable block.
func spawnStatusFormatTask(task *SubagentTask) string {
	var sb strings.Builder

	header := fmt.Sprintf("[%s] status=%s", task.ID, task.Status)
	if task.Label != "" {
		header += fmt.Sprintf("  label=%q", task.Label)
	}
	if task.AgentID != "" {
		header += fmt.Sprintf("  agent=%s", task.AgentID)
	}
	if task.Created > 0 {
		created := time.UnixMilli(task.Created).UTC().Format("2006-01-02 15:04:05 UTC")
		header += fmt.Sprintf("  created=%s", created)
	}
	sb.WriteString(header)

	if task.Task != "" {
		sb.WriteString(fmt.Sprintf("\n  task:   %s", task.Task))
	}
	if task.Result != "" {
		result := task.Result
		const maxResultLen = 300
		if len(result) > maxResultLen {
			result = result[:maxResultLen] + "…"
		}
		sb.WriteString(fmt.Sprintf("\n  result: %s", result))
	}

	return sb.String()
}

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
	return "Get the status of spawned subagents associated with the current tool context " +
		"(for example, the current channel or chat). Returns a list of all such subagents " +
		"and their current state (running, completed, failed, or canceled), or retrieves " +
		"details for a specific subagent task when task_id is provided. If the tool is " +
		"invoked without any associated context, the result will be empty."
}

func (t *SpawnStatusTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"task_id": map[string]any{
				"type": "string",
				"description": "Optional task ID (e.g. \"subagent-1\") to inspect a specific " +
					"subagent. When omitted, all known subagents in the current tool context " +
					"are listed; if the tool has no associated context, the result will be empty.",
			},
		},
		"required": []string{},
	}
}

func (t *SpawnStatusTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	if t.manager == nil {
		return ErrorResult("Subagent manager not configured")
	}

	// Derive the calling conversation's identity so we can scope results to the
	// current chat only — preventing cross-conversation task leakage in
	// multi-user deployments.
	callerChannel := ToolChannel(ctx)
	callerChatID := ToolChatID(ctx)

	taskID, _ := args["task_id"].(string)
	taskID = strings.TrimSpace(taskID)

	if taskID != "" {
		task, ok := t.manager.GetTask(taskID)
		if !ok {
			return ErrorResult(fmt.Sprintf("No subagent found with task ID: %s", taskID))
		}

		// Snapshot before formatting to avoid racing with the subagent goroutine.
		taskCopy := *task

		// Restrict lookup to tasks that belong to this conversation.
		if callerChannel != "" && taskCopy.OriginChannel != "" && taskCopy.OriginChannel != callerChannel {
			return ErrorResult(fmt.Sprintf("No subagent found with task ID: %s", taskID))
		}
		if callerChatID != "" && taskCopy.OriginChatID != "" && taskCopy.OriginChatID != callerChatID {
			return ErrorResult(fmt.Sprintf("No subagent found with task ID: %s", taskID))
		}

		return NewToolResult(spawnStatusFormatTask(&taskCopy))
	}

	origTasks := t.manager.ListTasks()
	if len(origTasks) == 0 {
		return NewToolResult("No subagents have been spawned yet.")
	}

	// Snapshot each task to avoid reading concurrently mutated state via shared
	// pointers once the manager lock is released.
	tasks := make([]*SubagentTask, 0, len(origTasks))
	for _, task := range origTasks {
		if task == nil {
			continue
		}
		cpy := *task

		// Filter to tasks that originate from the current conversation only.
		if callerChannel != "" && cpy.OriginChannel != "" && cpy.OriginChannel != callerChannel {
			continue
		}
		if callerChatID != "" && cpy.OriginChatID != "" && cpy.OriginChatID != callerChatID {
			continue
		}

		tasks = append(tasks, &cpy)
	}

	if len(tasks) == 0 {
		return NewToolResult("No subagents have been spawned yet.")
	}

	// Deterministic ordering: sort by ID string (e.g. "subagent-1" < "subagent-2").
	sort.Slice(tasks, func(i, j int) bool {
		if tasks[i] == nil || tasks[j] == nil {
			return false
		}
		return tasks[i].ID < tasks[j].ID
	})

	counts := map[string]int{}
	for _, task := range tasks {
		if task == nil {
			continue
		}
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
		if task == nil {
			continue
		}
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
		runes := []rune(result)
		if len(runes) > maxResultLen {
			result = string(runes[:maxResultLen]) + "…"
		}
		sb.WriteString(fmt.Sprintf("\n  result: %s", result))
	}

	return sb.String()
}

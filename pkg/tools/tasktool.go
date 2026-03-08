package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/session"
)

type TaskTool struct {
	taskManager     *session.TaskManager
	sendPlaceholder func(ctx context.Context, channel, chatID, content string) (string, error)
	editMessage     func(ctx context.Context, channel, chatID, messageID, content string) error
	icons           config.TaskToolIconsConfig
}

func NewTaskTool(taskManager *session.TaskManager, icons config.TaskToolIconsConfig) *TaskTool {

	return &TaskTool{
		taskManager: taskManager,
		icons:       icons,
	}
}

func (t *TaskTool) Name() string {
	return "tasktool"
}

func (t *TaskTool) ExecuteSequentially() bool {
	return true
}

func (t *TaskTool) Description() string {
	return "Manage planning mode tasks. Use action='create_plan' to start a new plan with a list of tasks. Use action='update_task' to update the status of an existing task and return the current plan state.\n\n" +
		"CRITICAL INSTRUCTIONS:\n" +
		"- If you determine a user request is complex and requires planning, use 'create_plan' to define a checklist. Wait for the user to accept the plan. Once accepted, execute the plan and update the status of each step using 'update_task'.\n" +
		"- Use ONLY `tasktool` for storing and updating tasks. Do NOT save tasks into files.\n" +
		"- Do not duplicate the plan text into the chat. `tasktool` already sends the plan automatically. Only write messages when completing a task or if there are questions/problems."
}

func (t *TaskTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type":        "string",
				"description": "Action to perform: 'create_plan', 'update_task', 'list_plan', or 'resend_plan'",
				"enum":        []string{"create_plan", "update_task", "list_plan", "resend_plan"},
			},
			"tasks": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"id": map[string]any{
							"type":        "string",
							"description": "Unique identifier for the task (e.g. 'task_1')",
						},
						"description": map[string]any{
							"type":        "string",
							"description": "Description of the task to be completed",
						},
					},
					"required": []string{"id", "description"},
				},
				"description": "List of tasks (only used for 'create_plan')",
			},
			"task_id": map[string]any{
				"type":        "string",
				"description": "ID of the task to update (only used for 'update_task')",
			},
			"status": map[string]any{
				"type":        "string",
				"description": "New status for the task (only used for 'update_task')",
				"enum":        []string{"pending", "in_progress", "completed", "failed"},
			},
			"result": map[string]any{
				"type":        "string",
				"description": "Optional brief result or note about the task update (only used for 'update_task')",
			},
		},
		"required": []string{"action"},
	}
}

func (t *TaskTool) SetCallbacks(
	sendPlaceholder func(ctx context.Context, channel, chatID, content string) (string, error),
	editMessage func(ctx context.Context, channel, chatID, messageID, content string) error,
) {
	t.sendPlaceholder = sendPlaceholder
	t.editMessage = editMessage
}

func (t *TaskTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	if t.taskManager == nil {
		return &ToolResult{ForLLM: "tasktool: task manager not configured", IsError: true}
	}

	channel := ToolChannel(ctx)
	chatID := ToolChatID(ctx)
	// Use the agent-scoped session key injected by the agent loop so that task
	// plans are stored under the same key used for conversation history.
	// Fall back to channel:chatID only when running outside of an agent loop
	// (e.g. unit tests or direct tool invocations).
	sessionKey := ToolSessionKey(ctx)
	if sessionKey == "" {
		sessionKey = fmt.Sprintf("%s:%s", channel, chatID)
	}

	action, ok := args["action"].(string)
	if !ok {
		return &ToolResult{ForLLM: "tasktool: action is required", IsError: true}
	}

	switch action {
	case "create_plan":
		return t.handleCreatePlan(ctx, sessionKey, channel, chatID, args)
	case "update_task":
		return t.handleUpdateTask(ctx, sessionKey, channel, chatID, args)
	case "list_plan":
		return t.handleListPlan(sessionKey)
	case "resend_plan":
		return t.handleResendPlan(ctx, sessionKey, channel, chatID)
	default:
		return &ToolResult{ForLLM: fmt.Sprintf("tasktool: unknown action '%s'", action), IsError: true}
	}
}

func (t *TaskTool) handleCreatePlan(ctx context.Context, sessionKey, channel, chatID string, args map[string]any) *ToolResult {
	tasksRaw, ok := args["tasks"].([]interface{})
	if !ok || len(tasksRaw) == 0 {
		return &ToolResult{ForLLM: "tasktool: tasks array is required and cannot be empty for 'create_plan'", IsError: true}
	}

	var parsedTasks []session.Task
	for i, raw := range tasksRaw {
		taskMap, ok := raw.(map[string]interface{})
		if !ok {
			return &ToolResult{ForLLM: fmt.Sprintf("tasktool: invalid task at index %d", i), IsError: true}
		}

		id, ok := taskMap["id"].(string)
		if !ok || id == "" {
			return &ToolResult{ForLLM: fmt.Sprintf("tasktool: missing id for task at index %d", i), IsError: true}
		}

		desc, ok := taskMap["description"].(string)
		if !ok || desc == "" {
			return &ToolResult{ForLLM: fmt.Sprintf("tasktool: missing description for task at index %d", i), IsError: true}
		}

		parsedTasks = append(parsedTasks, session.Task{
			ID:          id,
			Description: desc,
			Status:      session.TaskStatusPending,
		})
	}

	st := t.taskManager.CreatePlan(sessionKey, parsedTasks)

	content := t.formatPlanMessage(st.Tasks)
	summary := fmt.Sprintf("Plan created with %d tasks.\nTasks: %s", len(parsedTasks), mustMarshalJSON(parsedTasks))

	delivered := false
	var deliveryErr error
	if t.sendPlaceholder != nil {
		msgID, err := t.sendPlaceholder(ctx, channel, chatID, content)
		if err == nil {
			delivered = true
			if msgID != "" {
				t.taskManager.SetMessageID(sessionKey, msgID)
			}
		} else {
			deliveryErr = err
		}
	}

	return t.newPlanResult(summary, content, delivered, deliveryErr)
}

func (t *TaskTool) handleListPlan(sessionKey string) *ToolResult {
	st := t.taskManager.Get(sessionKey)
	if st == nil || len(st.Tasks) == 0 {
		return &ToolResult{
			ForLLM: "No active plan found for this session.",
			Silent: true,
		}
	}

	content := t.formatPlanMessage(st.Tasks)
	summary := fmt.Sprintf("Current plan state:\n%s\n\nRaw JSON:\n%s", content, mustMarshalJSON(st.Tasks))

	// list_plan does not send anything on its own, so always expose the plan to
	// direct callers through ForUser as well.
	return &ToolResult{
		ForLLM:  summary,
		ForUser: content,
		Silent:  false,
	}
}

func (t *TaskTool) handleResendPlan(ctx context.Context, sessionKey, channel, chatID string) *ToolResult {
	st := t.taskManager.GetOrCreate(sessionKey)
	if len(st.Tasks) == 0 {
		return &ToolResult{
			ForLLM:  "No active plan found for this session to resend.",
			IsError: true,
		}
	}

	content := t.formatPlanMessage(st.Tasks)

	delivered := false
	var deliveryErr error
	if t.sendPlaceholder != nil {
		msgID, err := t.sendPlaceholder(ctx, channel, chatID, content)
		if err == nil {
			delivered = true
			if msgID != "" {
				t.taskManager.SetMessageID(sessionKey, msgID)
			}
		} else {
			deliveryErr = err
		}
	}

	summary := fmt.Sprintf("Plan content prepared for resend.\nTasks: %s", mustMarshalJSON(st.Tasks))
	if delivered {
		summary = fmt.Sprintf("Plan successfully resent as a new message.\nTasks: %s", mustMarshalJSON(st.Tasks))
	}

	return t.newPlanResult(summary, content, delivered, deliveryErr)
}

func (t *TaskTool) handleUpdateTask(ctx context.Context, sessionKey, channel, chatID string, args map[string]any) *ToolResult {
	taskID, _ := args["task_id"].(string)
	if taskID == "" {
		return &ToolResult{ForLLM: "tasktool: task_id is required for 'update_task'", IsError: true}
	}

	statusStr, _ := args["status"].(string)
	if statusStr == "" {
		return &ToolResult{ForLLM: "tasktool: status is required for 'update_task'", IsError: true}
	}

	result, _ := args["result"].(string)

	st, err := t.taskManager.UpdateTask(sessionKey, taskID, session.TaskStatus(statusStr), result)
	if err != nil {
		return &ToolResult{ForLLM: fmt.Sprintf("tasktool: %v", err), IsError: true}
	}

	content := t.formatPlanMessage(st.Tasks)
	summary := fmt.Sprintf("Task '%s' updated to '%s'. Current plan:\n%s", taskID, statusStr, mustMarshalJSON(st.Tasks))

	delivered := false
	var deliveryErr error
	if t.editMessage != nil && st.MessageID != "" {
		if err := t.editMessage(ctx, channel, chatID, st.MessageID, content); err != nil {
			logger.WarnCF("tasktool", "Failed to edit task message", map[string]any{
				"channel":    channel,
				"chat_id":    chatID,
				"message_id": st.MessageID,
				"error":      err.Error(),
			})
			deliveryErr = err
		} else {
			delivered = true
		}
	}

	if !delivered && t.sendPlaceholder != nil {
		// Fallback: send new progress message if we didn't have one
		msgID, err := t.sendPlaceholder(ctx, channel, chatID, content)
		if err == nil {
			delivered = true
			deliveryErr = nil
			if msgID != "" {
				t.taskManager.SetMessageID(sessionKey, msgID)
			}
		} else if deliveryErr == nil {
			deliveryErr = err
		}
	}

	return t.newPlanResult(summary, content, delivered, deliveryErr)
}

func (t *TaskTool) formatPlanMessage(tasks []session.Task) string {
	var sb strings.Builder
	sb.WriteString("📋 **Execution Plan**:\n\n")

	for _, task := range tasks {
		var icon string
		switch task.Status {
		case session.TaskStatusPending:
			icon = t.icons.Pending
		case session.TaskStatusInProgress:
			icon = t.icons.InProgress
		case session.TaskStatusCompleted:
			icon = t.icons.Completed
		case session.TaskStatusFailed:
			icon = t.icons.Failed
		default:
			icon = t.icons.Pending
		}

		// Primitive markdown-to-html regex parser across multiple lines.
		// For the description and result, we replace lone underscores to prevent similar italic bugs.
		safeDesc := strings.ReplaceAll(task.Description, "_", "\\_")

		sb.WriteString(fmt.Sprintf("%s %s\n", icon, safeDesc))
		if task.Result != "" {
			safeResult := strings.ReplaceAll(task.Result, "_", "\\_")
			sb.WriteString(fmt.Sprintf("    **Result**: %s\n", safeResult))
		}
	}

	return sb.String()
}

func (t *TaskTool) newPlanResult(summary, content string, delivered bool, deliveryErr error) *ToolResult {
	if delivered {
		return &ToolResult{
			ForLLM: summary,
			Silent: true,
		}
	}

	forLLM := summary
	if deliveryErr != nil {
		forLLM = fmt.Sprintf("%s\nAutomatic delivery failed (%v). Respond to the user with the following plan content:\n\n%s", summary, deliveryErr, content)
	} else {
		forLLM = fmt.Sprintf("%s\nAutomatic delivery is unavailable in this context. Respond to the user with the following plan content:\n\n%s", summary, content)
	}

	return &ToolResult{
		ForLLM:  forLLM,
		ForUser: content,
		Silent:  false,
	}
}

func mustMarshalJSON(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		return "[]"
	}
	return string(data)
}

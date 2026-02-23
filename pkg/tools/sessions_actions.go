package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	defaultSessionsSendTimeoutSeconds = 30
	maxSessionsSendTimeoutSeconds     = 3600
)

// SessionsSendExecutor executes a message inside a target session key.
type SessionsSendExecutor interface {
	ProcessSessionMessage(ctx context.Context, content, sessionKey, channel, chatID string) (string, error)
}

type SessionsSendTool struct {
	executor SessionsSendExecutor
	channel  string
	chatID   string
}

func NewSessionsSendTool(executor SessionsSendExecutor) *SessionsSendTool {
	return &SessionsSendTool{
		executor: executor,
		channel:  "system",
		chatID:   "sessions-send",
	}
}

func (t *SessionsSendTool) Name() string {
	return "sessions_send"
}

func (t *SessionsSendTool) Description() string {
	return "Send a message into another session and return the target session's assistant reply."
}

func (t *SessionsSendTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"session_key": map[string]any{
				"type":        "string",
				"description": "Target session key (from sessions_list)",
			},
			"message": map[string]any{
				"type":        "string",
				"description": "Message to send into the target session",
			},
			"timeout_seconds": map[string]any{
				"type":        "integer",
				"description": "Max wait time for target response (default 30, max 3600, 0 = no timeout)",
				"minimum":     0.0,
				"maximum":     3600.0,
			},
		},
		"required": []string{"session_key", "message"},
	}
}

func (t *SessionsSendTool) SetContext(channel, chatID string) {
	if strings.TrimSpace(channel) != "" {
		t.channel = channel
	}
	if strings.TrimSpace(chatID) != "" {
		t.chatID = chatID
	}
}

func (t *SessionsSendTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	if t.executor == nil {
		return ErrorResult("sessions executor not configured")
	}

	sessionKey, ok := getStringArg(args, "session_key")
	if !ok || strings.TrimSpace(sessionKey) == "" {
		return ErrorResult("session_key is required")
	}
	message, ok := getStringArg(args, "message")
	if !ok || strings.TrimSpace(message) == "" {
		return ErrorResult("message is required")
	}

	timeoutSeconds, err := parseOptionalIntArg(
		args,
		"timeout_seconds",
		defaultSessionsSendTimeoutSeconds,
		0,
		maxSessionsSendTimeoutSeconds,
	)
	if err != nil {
		return ErrorResult(err.Error())
	}

	execCtx := ctx
	cancel := func() {}
	if timeoutSeconds > 0 {
		execCtx, cancel = context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
	}
	defer cancel()

	reply, execErr := t.executor.ProcessSessionMessage(
		execCtx,
		message,
		strings.TrimSpace(sessionKey),
		"system",
		"sessions-send",
	)

	payload := map[string]any{
		"session_key": strings.TrimSpace(sessionKey),
	}

	switch {
	case execErr == nil:
		payload["status"] = "ok"
		payload["reply"] = reply
	case errors.Is(execErr, context.DeadlineExceeded) || errors.Is(execCtx.Err(), context.DeadlineExceeded):
		payload["status"] = "timeout"
		payload["error"] = execErr.Error()
	default:
		payload["status"] = "error"
		payload["error"] = execErr.Error()
	}

	data, marshalErr := json.MarshalIndent(payload, "", "  ")
	if marshalErr != nil {
		return ErrorResult(fmt.Sprintf("failed to encode sessions_send payload: %v", marshalErr))
	}
	if execErr != nil && payload["status"] == "error" {
		return ErrorResult(string(data))
	}
	return SilentResult(string(data))
}

type SessionsSpawnTool struct {
	manager        *SubagentManager
	originChannel  string
	originChatID   string
	allowlistCheck func(targetAgentID string) bool
}

func NewSessionsSpawnTool(manager *SubagentManager) *SessionsSpawnTool {
	return &SessionsSpawnTool{
		manager:       manager,
		originChannel: "cli",
		originChatID:  "direct",
	}
}

func (t *SessionsSpawnTool) Name() string {
	return "sessions_spawn"
}

func (t *SessionsSpawnTool) Description() string {
	return "Spawn a background subagent task and return task metadata."
}

func (t *SessionsSpawnTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"task": map[string]any{
				"type":        "string",
				"description": "Task content for the spawned subagent",
			},
			"label": map[string]any{
				"type":        "string",
				"description": "Optional short label for tracking",
			},
			"agent_id": map[string]any{
				"type":        "string",
				"description": "Optional target agent id",
			},
		},
		"required": []string{"task"},
	}
}

func (t *SessionsSpawnTool) SetContext(channel, chatID string) {
	t.originChannel = channel
	t.originChatID = chatID
}

func (t *SessionsSpawnTool) SetAllowlistChecker(check func(targetAgentID string) bool) {
	t.allowlistCheck = check
}

func (t *SessionsSpawnTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	if t.manager == nil {
		return ErrorResult("Subagent manager not configured")
	}

	task, ok := getStringArg(args, "task")
	if !ok || strings.TrimSpace(task) == "" {
		return ErrorResult("task is required")
	}

	label, _ := getStringArg(args, "label")
	label = strings.TrimSpace(label)
	agentID, _ := getStringArg(args, "agent_id")
	agentID = strings.TrimSpace(agentID)

	if agentID != "" && t.allowlistCheck != nil && !t.allowlistCheck(agentID) {
		return ErrorResult(fmt.Sprintf("not allowed to spawn agent '%s'", agentID))
	}

	taskInfo, err := t.manager.SpawnTask(
		ctx,
		strings.TrimSpace(task),
		label,
		agentID,
		t.originChannel,
		t.originChatID,
		nil,
	)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to spawn subagent: %v", err))
	}

	payload := map[string]any{
		"status":   "accepted",
		"task_id":  taskInfo.ID,
		"task":     taskInfo.Task,
		"label":    taskInfo.Label,
		"agent_id": taskInfo.AgentID,
	}
	data, marshalErr := json.MarshalIndent(payload, "", "  ")
	if marshalErr != nil {
		return ErrorResult(fmt.Sprintf("failed to encode sessions_spawn payload: %v", marshalErr))
	}
	return SilentResult(string(data))
}

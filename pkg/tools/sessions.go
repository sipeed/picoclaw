package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/session"
)

const (
	defaultSessionsListLimit   = 50
	maxSessionsListLimit       = 200
	defaultSessionsHistorySize = 200
	maxSessionsHistorySize     = 1000
	maxSessionPreviewMessages  = 50
	maxMessagePreviewChars     = 500
)

type SessionsListTool struct {
	sessions *session.SessionManager
}

func NewSessionsListTool(sm *session.SessionManager) *SessionsListTool {
	return &SessionsListTool{sessions: sm}
}

func (t *SessionsListTool) Name() string {
	return "sessions_list"
}

func (t *SessionsListTool) ParallelPolicy() ToolParallelPolicy {
	return ToolParallelReadOnly
}

func (t *SessionsListTool) Description() string {
	return "List known conversation sessions with metadata. Useful for debugging, navigation, and context inspection."
}

func (t *SessionsListTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"kinds": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "string",
				},
				"description": "Optional session kind filter (e.g. main, direct, group, cron, subagent)",
			},
			"limit": map[string]any{
				"type":        "integer",
				"description": "Maximum number of sessions to return (default 50, max 200)",
				"minimum":     1.0,
				"maximum":     200.0,
			},
			"active_minutes": map[string]any{
				"type":        "integer",
				"description": "Only include sessions updated within N minutes",
				"minimum":     1.0,
			},
			"message_limit": map[string]any{
				"type":        "integer",
				"description": "Include up to N recent preview messages per session (default 0, max 50)",
				"minimum":     0.0,
				"maximum":     50.0,
			},
			"include_tools": map[string]any{
				"type":        "boolean",
				"description": "When message_limit > 0, include tool messages in preview (default false)",
			},
		},
	}
}

func (t *SessionsListTool) Execute(_ context.Context, args map[string]any) *ToolResult {
	if t.sessions == nil {
		return ErrorResult("session manager not configured")
	}

	limit, err := parseIntArg(args, "limit", defaultSessionsListLimit, 1, maxSessionsListLimit)
	if err != nil {
		return ErrorResult(err.Error())
	}

	activeMinutes, err := parseOptionalIntArg(args, "active_minutes", 0, 0, 365*24*60)
	if err != nil {
		return ErrorResult(err.Error())
	}

	messageLimit, err := parseIntArg(args, "message_limit", 0, 0, maxSessionPreviewMessages)
	if err != nil {
		return ErrorResult(err.Error())
	}

	includeTools, err := parseBoolArg(args, "include_tools", false)
	if err != nil {
		return ErrorResult(err.Error())
	}

	kinds, err := parseStringSliceArg(args, "kinds")
	if err != nil {
		return ErrorResult(err.Error())
	}

	allowedKinds := make(map[string]struct{}, len(kinds))
	for _, k := range kinds {
		allowedKinds[strings.ToLower(strings.TrimSpace(k))] = struct{}{}
	}

	now := time.Now()
	snapshots := t.sessions.ListSessionSnapshots()
	outSessions := make([]sessionListItem, 0, min(limit, len(snapshots)))

	for _, s := range snapshots {
		kind := classifySessionKind(s.Key)

		if len(allowedKinds) > 0 {
			if _, ok := allowedKinds[kind]; !ok {
				continue
			}
		}

		if activeMinutes > 0 && now.Sub(s.Updated) > time.Duration(activeMinutes)*time.Minute {
			continue
		}

		item := sessionListItem{
			Key:          s.Key,
			Kind:         kind,
			CreatedAt:    s.Created.Format(time.RFC3339),
			UpdatedAt:    s.Updated.Format(time.RFC3339),
			MessageCount: len(s.Messages),
		}
		if strings.TrimSpace(s.Summary) != "" {
			item.Summary = s.Summary
		}
		if messageLimit > 0 {
			item.Messages = tailMessages(s.Messages, messageLimit, includeTools)
		}

		outSessions = append(outSessions, item)
		if len(outSessions) >= limit {
			break
		}
	}

	payload := sessionsListOutput{
		Count:    len(outSessions),
		Sessions: outSessions,
	}

	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to encode sessions list: %v", err))
	}
	return SilentResult(string(data))
}

type SessionsHistoryTool struct {
	sessions *session.SessionManager
}

func NewSessionsHistoryTool(sm *session.SessionManager) *SessionsHistoryTool {
	return &SessionsHistoryTool{sessions: sm}
}

func (t *SessionsHistoryTool) Name() string {
	return "sessions_history"
}

func (t *SessionsHistoryTool) ParallelPolicy() ToolParallelPolicy {
	return ToolParallelReadOnly
}

func (t *SessionsHistoryTool) Description() string {
	return "Get full or partial message history for one session key."
}

func (t *SessionsHistoryTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"session_key": map[string]any{
				"type":        "string",
				"description": "Session key returned by sessions_list",
			},
			"limit": map[string]any{
				"type":        "integer",
				"description": "Maximum messages to return from the tail (default 200, max 1000)",
				"minimum":     1.0,
				"maximum":     1000.0,
			},
			"include_tools": map[string]any{
				"type":        "boolean",
				"description": "Include tool-role messages (default false)",
			},
		},
		"required": []string{"session_key"},
	}
}

func (t *SessionsHistoryTool) Execute(_ context.Context, args map[string]any) *ToolResult {
	if t.sessions == nil {
		return ErrorResult("session manager not configured")
	}

	key, ok := getStringArg(args, "session_key")
	if !ok {
		return ErrorResult("session_key is required")
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return ErrorResult("session_key is required")
	}

	limit, err := parseIntArg(args, "limit", defaultSessionsHistorySize, 1, maxSessionsHistorySize)
	if err != nil {
		return ErrorResult(err.Error())
	}

	includeTools, err := parseBoolArg(args, "include_tools", false)
	if err != nil {
		return ErrorResult(err.Error())
	}

	snapshot, ok := t.sessions.GetSessionSnapshot(key)
	if !ok {
		return ErrorResult(fmt.Sprintf("session %q not found", key))
	}

	messages := tailMessages(snapshot.Messages, limit, includeTools)
	payload := sessionHistoryOutput{
		SessionKey:   snapshot.Key,
		Kind:         classifySessionKind(snapshot.Key),
		CreatedAt:    snapshot.Created.Format(time.RFC3339),
		UpdatedAt:    snapshot.Updated.Format(time.RFC3339),
		MessageCount: len(messages),
		Messages:     messages,
	}
	if strings.TrimSpace(snapshot.Summary) != "" {
		payload.Summary = snapshot.Summary
	}

	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to encode session history: %v", err))
	}
	return SilentResult(string(data))
}

type sessionListItem struct {
	Key          string              `json:"key"`
	Kind         string              `json:"kind"`
	CreatedAt    string              `json:"created_at"`
	UpdatedAt    string              `json:"updated_at"`
	MessageCount int                 `json:"message_count"`
	Summary      string              `json:"summary,omitempty"`
	Messages     []providers.Message `json:"messages,omitempty"`
}

type sessionsListOutput struct {
	Count    int               `json:"count"`
	Sessions []sessionListItem `json:"sessions"`
}

type sessionHistoryOutput struct {
	SessionKey   string              `json:"session_key"`
	Kind         string              `json:"kind"`
	CreatedAt    string              `json:"created_at"`
	UpdatedAt    string              `json:"updated_at"`
	MessageCount int                 `json:"message_count"`
	Summary      string              `json:"summary,omitempty"`
	Messages     []providers.Message `json:"messages"`
}

func classifySessionKind(key string) string {
	k := strings.ToLower(strings.TrimSpace(key))
	switch {
	case k == "":
		return "other"
	case strings.HasPrefix(k, "agent:") && strings.HasSuffix(k, ":main"):
		return "main"
	case strings.Contains(k, ":subagent:"):
		return "subagent"
	case strings.Contains(k, ":group:"):
		return "group"
	case strings.Contains(k, ":channel:"):
		return "channel"
	case strings.Contains(k, ":direct:"):
		return "direct"
	case strings.HasPrefix(k, "cron:") || strings.HasPrefix(k, "cron-"):
		return "cron"
	case strings.HasPrefix(k, "hook:"):
		return "hook"
	case strings.HasPrefix(k, "node-"):
		return "node"
	case k == "heartbeat":
		return "heartbeat"
	default:
		return "other"
	}
}

func tailMessages(messages []providers.Message, limit int, includeTools bool) []providers.Message {
	if limit <= 0 || len(messages) == 0 {
		return nil
	}

	selected := make([]providers.Message, 0, min(limit, len(messages)))
	for i := len(messages) - 1; i >= 0 && len(selected) < limit; i-- {
		msg := messages[i]
		if !includeTools && msg.Role == "tool" {
			continue
		}
		if len(msg.Content) > maxMessagePreviewChars {
			msg.Content = msg.Content[:maxMessagePreviewChars] + "...(truncated)"
		}
		selected = append(selected, msg)
	}

	// Reverse back to chronological order.
	for i, j := 0, len(selected)-1; i < j; i, j = i+1, j-1 {
		selected[i], selected[j] = selected[j], selected[i]
	}
	return selected
}

func parseIntArg(args map[string]any, key string, defaultVal, minVal, maxVal int) (int, error) {
	val, exists := args[key]
	if !exists {
		return defaultVal, nil
	}

	n, err := toInt(val)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer", key)
	}
	if n < minVal || n > maxVal {
		return 0, fmt.Errorf("%s must be between %d and %d", key, minVal, maxVal)
	}
	return n, nil
}

func parseOptionalIntArg(args map[string]any, key string, defaultVal, minVal, maxVal int) (int, error) {
	val, exists := args[key]
	if !exists {
		return defaultVal, nil
	}

	n, err := toInt(val)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer", key)
	}
	if n < minVal || n > maxVal {
		return 0, fmt.Errorf("%s must be between %d and %d", key, minVal, maxVal)
	}
	return n, nil
}

func parseBoolArg(args map[string]any, key string, defaultVal bool) (bool, error) {
	val, exists := args[key]
	if !exists {
		return defaultVal, nil
	}
	b, ok := val.(bool)
	if !ok {
		return false, fmt.Errorf("%s must be a boolean", key)
	}
	return b, nil
}

func parseStringSliceArg(args map[string]any, key string) ([]string, error) {
	val, exists := args[key]
	if !exists {
		return nil, nil
	}

	switch v := val.(type) {
	case []string:
		out := make([]string, 0, len(v))
		for _, s := range v {
			if trimmed := strings.TrimSpace(s); trimmed != "" {
				out = append(out, trimmed)
			}
		}
		return out, nil
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			s, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("%s must be an array of strings", key)
			}
			if trimmed := strings.TrimSpace(s); trimmed != "" {
				out = append(out, trimmed)
			}
		}
		return out, nil
	default:
		return nil, fmt.Errorf("%s must be an array of strings", key)
	}
}

func getStringArg(args map[string]any, key string) (string, bool) {
	v, ok := args[key]
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

func toInt(v any) (int, error) {
	switch n := v.(type) {
	case int:
		return n, nil
	case int8:
		return int(n), nil
	case int16:
		return int(n), nil
	case int32:
		return int(n), nil
	case int64:
		return int(n), nil
	case uint:
		return int(n), nil
	case uint8:
		return int(n), nil
	case uint16:
		return int(n), nil
	case uint32:
		return int(n), nil
	case uint64:
		return int(n), nil
	case float64:
		if n != math.Trunc(n) {
			return 0, fmt.Errorf("not an integer")
		}
		return int(n), nil
	case float32:
		if float64(n) != math.Trunc(float64(n)) {
			return 0, fmt.Errorf("not an integer")
		}
		return int(n), nil
	default:
		return 0, fmt.Errorf("not an integer")
	}
}

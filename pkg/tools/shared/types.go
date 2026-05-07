package toolshared

import "context"

type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

type ToolCall struct {
	ID        string         `json:"id"`
	Type      string         `json:"type"`
	Function  *FunctionCall  `json:"function,omitempty"`
	Name      string         `json:"name,omitempty"`
	Arguments map[string]any `json:"arguments,omitempty"`
}

type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type LLMResponse struct {
	Content      string     `json:"content"`
	ToolCalls    []ToolCall `json:"tool_calls,omitempty"`
	FinishReason string     `json:"finish_reason"`
	Usage        *UsageInfo `json:"usage,omitempty"`
}

type UsageInfo struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type LLMProvider interface {
	Chat(
		ctx context.Context,
		messages []Message,
		tools []ToolDefinition,
		model string,
		options map[string]any,
	) (*LLMResponse, error)
	GetDefaultModel() string
}

type ToolDefinition struct {
	Type     string                 `json:"type"`
	Function ToolFunctionDefinition `json:"function"`
}

type ToolFunctionDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

type ExecRequest struct {
	Action     string            `json:"action"`
	Command    string            `json:"command,omitempty"`
	PTY        bool              `json:"pty,omitempty"`
	Background bool              `json:"background,omitempty"`
	Timeout    int               `json:"timeout,omitempty"`
	Env        map[string]string `json:"env,omitempty"`
	Cwd        string            `json:"cwd,omitempty"`
	SessionID  string            `json:"sessionId,omitempty"`
	Data       string            `json:"data,omitempty"`
}

type ExecResponse struct {
	SessionID string        `json:"sessionId,omitempty"`
	Status    string        `json:"status,omitempty"`
	ExitCode  int           `json:"exitCode,omitempty"`
	Output    string        `json:"output,omitempty"`
	Error     string        `json:"error,omitempty"`
	Sessions  []SessionInfo `json:"sessions,omitempty"`
}

type SessionInfo struct {
	ID        string `json:"id"`
	Command   string `json:"command"`
	Status    string `json:"status"`
	PID       int    `json:"pid"`
	StartedAt int64  `json:"startedAt"`
}

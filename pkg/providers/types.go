package providers

import (
	"context"
	"encoding/json"
)

type ToolCall struct {
	ID           string                 `json:"id"`
	Type         string                 `json:"type,omitempty"`
	Function     *FunctionCall          `json:"function,omitempty"`
	ExtraContent map[string]interface{} `json:"extra_content,omitempty"`
	Name         string                 `json:"name,omitempty"`
	Arguments    map[string]interface{} `json:"arguments,omitempty"`
}

type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type LLMResponse struct {
	Content             string          `json:"content"`
	ToolCalls           []ToolCall      `json:"tool_calls,omitempty"`
	FinishReason        string          `json:"finish_reason"`
	Usage               *UsageInfo      `json:"usage,omitempty"`
	RawAssistantMessage json.RawMessage `json:"-"`
}

type UsageInfo struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type ContentPart struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *ImageURL `json:"image_url,omitempty"`
}

type ImageURL struct {
	URL string `json:"url"`
}

type Message struct {
	Role         string          `json:"role"`
	Content      string          `json:"content"`
	ContentParts []ContentPart   `json:"content_parts,omitempty"`
	ToolCalls    []ToolCall      `json:"tool_calls,omitempty"`
	ToolCallID   string          `json:"tool_call_id,omitempty"`
	RawAPIMessage json.RawMessage `json:"raw_api_message,omitempty"`
}

type LLMProvider interface {
	Chat(ctx context.Context, messages []Message, tools []ToolDefinition, model string, options map[string]interface{}) (*LLMResponse, error)
	GetDefaultModel() string
}

type ToolDefinition struct {
	Type     string                 `json:"type"`
	Function ToolFunctionDefinition `json:"function"`
}

type ToolFunctionDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

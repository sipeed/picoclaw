package protocoltypes

import (
	"encoding/json"
	"strings"
)


type ToolCall struct {
	ID               string         `json:"id"`
	Type             string         `json:"type,omitempty"`
	Function         *FunctionCall  `json:"function,omitempty"`
	Name             string         `json:"-"`
	Arguments        map[string]any `json:"-"`
	ThoughtSignature string         `json:"-"` // Internal use only
	ExtraContent     *ExtraContent  `json:"extra_content,omitempty"`
}

type ExtraContent struct {
	Google *GoogleExtra `json:"google,omitempty"`
}

type GoogleExtra struct {
	ThoughtSignature string `json:"thought_signature,omitempty"`
}

type FunctionCall struct {
	Name             string         `json:"name"`
	Arguments        map[string]any `json:"-"`
	ThoughtSignature string         `json:"thought_signature,omitempty"`
}

func (f *FunctionCall) UnmarshalJSON(data []byte) error {
	var wire struct {
		Name             string `json:"name"`
		Arguments        any    `json:"arguments"`
		ThoughtSignature string `json:"thought_signature,omitempty"`
	}
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}

	f.Name = wire.Name
	f.ThoughtSignature = wire.ThoughtSignature
	f.Arguments = decodeFunctionArguments(wire.Arguments)
	return nil
}

func (f FunctionCall) MarshalJSON() ([]byte, error) {
	args := "{}"
	if len(f.Arguments) > 0 {
		payload, err := json.Marshal(f.Arguments)
		if err != nil {
			return nil, err
		}
		args = string(payload)
	}

	wire := struct {
		Name             string `json:"name"`
		Arguments        string `json:"arguments"`
		ThoughtSignature string `json:"thought_signature,omitempty"`
	}{
		Name:             f.Name,
		Arguments:        args,
		ThoughtSignature: f.ThoughtSignature,
	}
	return json.Marshal(wire)
}

func decodeFunctionArguments(raw any) map[string]any {
	switch v := raw.(type) {
	case nil:
		return map[string]any{}
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return map[string]any{}
		}
		var parsed map[string]any
		if err := json.Unmarshal([]byte(trimmed), &parsed); err != nil || parsed == nil {
			return map[string]any{"raw": v}
		}
		return parsed
	case map[string]any:
		if v == nil {
			return map[string]any{}
		}
		return v
	default:
		payload, err := json.Marshal(v)
		if err != nil {
			return map[string]any{}
		}
		return map[string]any{"raw": string(payload)}
	}
}

type LLMResponse struct {
	Content          string            `json:"content"`
	ReasoningContent string            `json:"reasoning_content,omitempty"`
	ToolCalls        []ToolCall        `json:"tool_calls,omitempty"`
	FinishReason     string            `json:"finish_reason"`
	Usage            *UsageInfo        `json:"usage,omitempty"`
	Reasoning        string            `json:"reasoning"`
	ReasoningDetails []ReasoningDetail `json:"reasoning_details"`
}

type ReasoningDetail struct {
	Format string `json:"format"`
	Index  int    `json:"index"`
	Type   string `json:"type"`
	Text   string `json:"text"`
}

type UsageInfo struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// CacheControl marks a content block for LLM-side prefix caching.
// Currently only "ephemeral" is supported (used by Anthropic).
type CacheControl struct {
	Type string `json:"type"` // "ephemeral"
}

// ContentBlock represents a structured segment of a system message.
// Adapters that understand SystemParts can use these blocks to set
// per-block cache control (e.g. Anthropic's cache_control: ephemeral).
type ContentBlock struct {
	Type         string        `json:"type"` // "text"
	Text         string        `json:"text"`
	CacheControl *CacheControl `json:"cache_control,omitempty"`
}

type Message struct {
	Role             string         `json:"role"`
	Content          string         `json:"content"`
	Media            []string       `json:"media,omitempty"`
	ReasoningContent string         `json:"reasoning_content,omitempty"`
	SystemParts      []ContentBlock `json:"system_parts,omitempty"` // structured system blocks for cache-aware adapters
	ToolCalls        []ToolCall     `json:"tool_calls,omitempty"`
	ToolCallID       string         `json:"tool_call_id,omitempty"`
}

type ToolDefinition struct {
	Type     string                 `json:"type"`
	Function ToolFunctionDefinition `json:"function"`
}

type ToolFunctionDefinition struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

func (t *ToolFunctionDefinition) ParametersMap() map[string]any {
	if t == nil || len(t.Parameters) == 0 {
		return nil
	}
	var params map[string]any
	if err := json.Unmarshal(t.Parameters, &params); err != nil {
		return nil
	}
	return params
}

func (t *ToolFunctionDefinition) SetParametersMap(params map[string]any) error {
	if len(params) == 0 {
		t.Parameters = json.RawMessage(`{}`)
		return nil
	}
	payload, err := json.Marshal(params)
	if err != nil {
		return err
	}
	t.Parameters = json.RawMessage(payload)
	return nil
}

func MustMarshalParameters(params map[string]any) json.RawMessage {
	if len(params) == 0 {
		return json.RawMessage(`{}`)
	}
	payload, err := json.Marshal(params)
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return json.RawMessage(payload)
}

// StreamEvent represents a single chunk from an SSE streaming response.
type StreamEvent struct {
	ContentDelta   string
	ReasoningDelta string // incremental reasoning/thinking content
	ToolCallDeltas []StreamToolCallDelta
	FinishReason   string     // set only on the final event
	Usage          *UsageInfo // set only on the final event
	Err            error      // non-nil when the stream encountered an error
}

// StreamToolCallDelta carries an incremental piece of a streaming tool call.
type StreamToolCallDelta struct {
	Index          int
	ID             string // set on the first chunk for this tool call
	Name           string // set on the first chunk for this tool call
	ArgumentsDelta string // JSON fragment (incremental)
}

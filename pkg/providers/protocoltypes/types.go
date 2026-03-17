package protocoltypes

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
	Name             string `json:"name"`
	Arguments        string `json:"arguments"`
	ThoughtSignature string `json:"thought_signature,omitempty"`
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

// MessageSender carries author identity for a user message.
// Stored alongside the message in history so the LLM can address
// participants by name in multi-user conversations.
type MessageSender struct {
	Username  string `json:"username,omitempty"`   // e.g. "@alice" (platform handle)
	FirstName string `json:"first_name,omitempty"` // given name
	LastName  string `json:"last_name,omitempty"`  // family name
}

type Message struct {
	Role             string         `json:"role"`
	Content          string         `json:"content"`
	Media            []string       `json:"media,omitempty"`
	ReasoningContent string         `json:"reasoning_content,omitempty"`
	SystemParts      []ContentBlock `json:"system_parts,omitempty"` // structured system blocks for cache-aware adapters
	ToolCalls        []ToolCall     `json:"tool_calls,omitempty"`
	ToolCallID       string         `json:"tool_call_id,omitempty"`
	MessageID        string         `json:"message_id,omitempty"`          // Platform message ID
	ReplyToMessageID string         `json:"reply_to_message_id,omitempty"` // Parent message ID (for threading)
	Sender           *MessageSender `json:"sender,omitempty"`              // Author identity (user messages only)
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

package tracer

// Message is a single role/content pair in an LLM conversation.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Tool is an available tool definition sent to the LLM.
type Tool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Parameters  string `json:"parameters"`
}

// LLMCall holds data about one iteration of an LLM request/response.
type LLMCall struct {
	Iteration       int       `json:"iteration"`
	Model           string    `json:"model"`
	MessagesCount   int       `json:"messages_count"`
	ToolsCount      int       `json:"tools_count"`
	MaxTokens       int       `json:"max_tokens"`
	SystemPromptLen int       `json:"system_prompt_len,omitempty"`
	Temperature     string    `json:"temperature,omitempty"`
	ContentLen      *int      `json:"content_len"`
	ToolCallsCount  *int      `json:"tool_calls_count"`
	HasReasoning    *bool     `json:"has_reasoning"`
	Timestamp       string    `json:"timestamp"`
	Messages        []Message `json:"messages"`
	Tools           []Tool    `json:"tools"`
}

// ToolExec holds data about a single tool execution within a turn.
type ToolExec struct {
	Tool       string `json:"tool"`
	ArgsCount  int    `json:"args_count"`
	DurationMs *int   `json:"duration_ms"`
	IsError    *bool  `json:"is_error"`
	ForLLMLen  *int   `json:"for_llm_len"`
	Timestamp  string `json:"timestamp"`
}

// Turn represents one complete user→agent interaction.
type Turn struct {
	TurnID     string     `json:"turn_id"`
	Timestamp  string     `json:"timestamp"`
	Channel    string     `json:"channel"`
	ChatID     string     `json:"chat_id"`
	SenderID   string     `json:"sender_id"`
	SessionKey string     `json:"session_key"`
	UserLen    int        `json:"user_len"`
	Status     string     `json:"status"`
	DurationMs *int       `json:"duration_ms"`
	Iterations int        `json:"iterations"`
	LLMCalls   []LLMCall  `json:"llm_calls"`
	ToolExecs  []ToolExec `json:"tool_execs"`
}

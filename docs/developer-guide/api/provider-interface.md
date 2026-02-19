# LLMProvider Interface Reference

This document provides detailed reference for the LLMProvider interface and related types.

## Core Interface

### LLMProvider

The interface that all LLM providers must implement.

```go
type LLMProvider interface {
    Chat(ctx context.Context, messages []Message, tools []ToolDefinition,
         model string, options map[string]interface{}) (*LLMResponse, error)
    GetDefaultModel() string
}
```

### Methods

#### Chat

Sends a chat request to the LLM.

```go
Chat(ctx context.Context, messages []Message, tools []ToolDefinition,
     model string, options map[string]interface{}) (*LLMResponse, error)
```

**Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| ctx | context.Context | Context for cancellation and timeout |
| messages | []Message | Conversation history |
| tools | []ToolDefinition | Available tools for function calling |
| model | string | Model identifier (may include provider prefix) |
| options | map[string]interface{} | Additional options (max_tokens, temperature, etc.) |

**Returns:**

| Type | Description |
|------|-------------|
| *LLMResponse | LLM response with content and tool calls |
| error | Error if request failed |

**Common Options:**

| Key | Type | Description |
|-----|------|-------------|
| max_tokens | int | Maximum tokens in response |
| temperature | float64 | Sampling temperature (0-2) |
| top_p | float64 | Nucleus sampling parameter |
| stop | []string | Stop sequences |

#### GetDefaultModel

Returns the default model for this provider.

```go
GetDefaultModel() string
```

**Returns:** Model identifier string (e.g., "openrouter/anthropic/claude-opus-4-5")

## Message Types

### Message

Represents a message in the conversation.

```go
type Message struct {
    Role       string     `json:"role"`
    Content    string     `json:"content"`
    ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
    ToolCallID string     `json:"tool_call_id,omitempty"`
}
```

**Fields:**

| Field | Type | Description |
|-------|------|-------------|
| Role | string | "system", "user", "assistant", or "tool" |
| Content | string | Message text content |
| ToolCalls | []ToolCall | Tool calls from assistant |
| ToolCallID | string | ID for tool response messages |

**Roles:**

- `system`: System instructions
- `user`: User message
- `assistant`: LLM response
- `tool`: Tool execution result

### ToolCall

Represents a tool call request from the LLM.

```go
type ToolCall struct {
    ID        string                 `json:"id"`
    Type      string                 `json:"type,omitempty"`
    Function  *FunctionCall          `json:"function,omitempty"`
    Name      string                 `json:"name,omitempty"`
    Arguments map[string]interface{} `json:"arguments,omitempty"`
}
```

**Fields:**

| Field | Type | Description |
|-------|------|-------------|
| ID | string | Unique identifier for the tool call |
| Type | string | Usually "function" |
| Function | *FunctionCall | Function call details (for API format) |
| Name | string | Function/tool name |
| Arguments | map[string]interface{} | Parsed arguments |

### FunctionCall

Detailed function call information.

```go
type FunctionCall struct {
    Name      string `json:"name"`
    Arguments string `json:"arguments"` // JSON string
}
```

**Fields:**

| Field | Type | Description |
|-------|------|-------------|
| Name | string | Function name |
| Arguments | string | Arguments as JSON string |

### LLMResponse

Response from the LLM.

```go
type LLMResponse struct {
    Content      string     `json:"content"`
    ToolCalls    []ToolCall `json:"tool_calls,omitempty"`
    FinishReason string     `json:"finish_reason"`
    Usage        *UsageInfo `json:"usage,omitempty"`
}
```

**Fields:**

| Field | Type | Description |
|-------|------|-------------|
| Content | string | Text content of response |
| ToolCalls | []ToolCall | Requested tool calls |
| FinishReason | string | Why generation stopped |
| Usage | *UsageInfo | Token usage information |

**Finish Reasons:**

- `stop`: Natural end of response
- `tool_calls`: LLM requested tool execution
- `length`: Max tokens reached
- `content_filter`: Content policy triggered

### UsageInfo

Token usage statistics.

```go
type UsageInfo struct {
    PromptTokens     int `json:"prompt_tokens"`
    CompletionTokens int `json:"completion_tokens"`
    TotalTokens      int `json:"total_tokens"`
}
```

**Fields:**

| Field | Type | Description |
|-------|------|-------------|
| PromptTokens | int | Tokens in the prompt |
| CompletionTokens | int | Tokens in the response |
| TotalTokens | int | Total tokens used |

## Tool Definition Types

### ToolDefinition

Defines a tool for the LLM.

```go
type ToolDefinition struct {
    Type     string                 `json:"type"`
    Function ToolFunctionDefinition `json:"function"`
}
```

### ToolFunctionDefinition

Describes a tool function.

```go
type ToolFunctionDefinition struct {
    Name        string                 `json:"name"`
    Description string                 `json:"description"`
    Parameters  map[string]interface{} `json:"parameters"`
}
```

## Error Handling

### FailoverError

Error with classification for fallback support.

```go
type FailoverError struct {
    Reason   FailoverReason
    Provider string
    Model    string
    Status   int
    Wrapped  error
}
```

**Fields:**

| Field | Type | Description |
|-------|------|-------------|
| Reason | FailoverReason | Classification of the error |
| Provider | string | Provider identifier |
| Model | string | Model being used |
| Status | int | HTTP status code |
| Wrapped | error | Underlying error |

### FailoverReason

Classification of LLM errors.

```go
type FailoverReason string

const (
    FailoverAuth       FailoverReason = "auth"
    FailoverRateLimit  FailoverReason = "rate_limit"
    FailoverBilling    FailoverReason = "billing"
    FailoverTimeout    FailoverReason = "timeout"
    FailoverFormat     FailoverReason = "format"
    FailoverOverloaded FailoverReason = "overloaded"
    FailoverUnknown    FailoverReason = "unknown"
)
```

**Reasons:**

| Reason | Description | Retriable |
|--------|-------------|-----------|
| auth | Authentication error | No |
| rate_limit | Rate limited | Yes |
| billing | Billing/quota issue | No |
| timeout | Request timeout | Yes |
| format | Invalid request format | No |
| overloaded | Service overloaded | Yes |
| unknown | Unknown error | Yes |

### Methods

#### Error() string

Returns formatted error message.

```go
func (e *FailoverError) Error() string
// "failover(auth): provider=openrouter model=gpt-4 status=401: ..."
```

#### Unwrap() error

Returns the underlying error.

```go
func (e *FailoverError) Unwrap() error
```

#### IsRetriable() bool

Returns true if the error should trigger fallback.

```go
func (e *FailoverError) IsRetriable() bool
// Returns false for Format errors, true otherwise
```

## Fallback Chain

### ModelConfig

Configuration for primary and fallback models.

```go
type ModelConfig struct {
    Primary   string
    Fallbacks []string
}
```

### FallbackChain

Handles fallback between model candidates.

```go
type FallbackChain struct {
    // contains filtered or unexported fields
}
```

#### NewFallbackChain(cooldown *CooldownTracker) *FallbackChain

Creates a new fallback chain.

```go
cooldown := providers.NewCooldownTracker()
fallback := providers.NewFallbackChain(cooldown)
```

#### Execute

Executes with fallback logic.

```go
func (fc *FallbackChain) Execute(ctx context.Context,
    candidates []ModelCandidate,
    fn func(ctx context.Context, provider, model string) (*LLMResponse, error),
) (*FallbackResult, error)
```

### CooldownTracker

Tracks provider cooldown periods.

```go
type CooldownTracker struct {
    // contains filtered or unexported fields
}
```

#### NewCooldownTracker() *CooldownTracker

Creates a new tracker.

```go
tracker := providers.NewCooldownTracker()
```

#### SetCooldown(provider string, duration time.Duration)

Sets a cooldown for a provider.

```go
tracker.SetCooldown("openrouter", 30*time.Second)
```

#### IsOnCooldown(provider string) bool

Checks if provider is on cooldown.

```go
if tracker.IsOnCooldown("openrouter") {
    // Skip this provider
}
```

## Model Reference

### ModelRef

Parses and manipulates model references.

Format: `provider/model` or just `model`

```go
// Parse model reference
ref := providers.ParseModelRef("openrouter/anthropic/claude-opus-4-5")

ref.Provider  // "openrouter"
ref.Model     // "anthropic/claude-opus-4-5"
ref.Full      // "openrouter/anthropic/claude-opus-4-5"
```

## Usage Example

### Basic Chat

```go
provider := openai_compat.NewProvider(apiKey, apiBase, "")

messages := []providers.Message{
    {Role: "system", Content: "You are a helpful assistant."},
    {Role: "user", Content: "Hello!"},
}

response, err := provider.Chat(ctx, messages, nil, "gpt-4", map[string]interface{}{
    "max_tokens":  1024,
    "temperature": 0.7,
})
if err != nil {
    return err
}

fmt.Println(response.Content)
```

### With Tool Calling

```go
tools := []providers.ToolDefinition{
    {
        Type: "function",
        Function: providers.ToolFunctionDefinition{
            Name:        "get_weather",
            Description: "Get current weather",
            Parameters: map[string]interface{}{
                "type": "object",
                "properties": map[string]interface{}{
                    "location": map[string]interface{}{
                        "type": "string",
                    },
                },
                "required": []string{"location"},
            },
        },
    },
}

response, err := provider.Chat(ctx, messages, tools, "gpt-4", options)

// Handle tool calls
for _, tc := range response.ToolCalls {
    fmt.Printf("Tool: %s, Args: %v\n", tc.Name, tc.Arguments)
}
```

### With Fallback

```go
candidates := []providers.ModelCandidate{
    {Provider: "openrouter", Model: "anthropic/claude-opus-4-5"},
    {Provider: "openrouter", Model: "anthropic/claude-sonnet-4"},
    {Provider: "groq", Model: "llama-3-70b"},
}

result, err := fallback.Execute(ctx, candidates,
    func(ctx context.Context, provider, model string) (*providers.LLMResponse, error) {
        return providerClient.Chat(ctx, messages, tools, model, options)
    },
)

if err != nil {
    return err
}

fmt.Printf("Succeeded with %s/%s\n", result.Provider, result.Model)
fmt.Println(result.Response.Content)
```

## Best Practices

1. **Always Use Context**: Pass context for cancellation
2. **Handle Timeouts**: Set appropriate timeouts
3. **Classify Errors**: Use FailoverError for fallback support
4. **Normalize Models**: Handle provider prefixes
5. **Validate Responses**: Check for empty or malformed responses
6. **Log Requests**: Log for debugging (with redacted API keys)

## See Also

- [Creating LLM Providers](../extending/creating-providers.md)
- [Provider Implementations](https://github.com/sipeed/picoclaw/tree/main/pkg/providers)
- [OpenAI-Compatible Provider](https://github.com/sipeed/picoclaw/tree/main/pkg/providers/openai_compat)

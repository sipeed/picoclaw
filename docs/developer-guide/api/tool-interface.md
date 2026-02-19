# Tool Interface Reference

This document provides detailed reference for the Tool interface and related types.

## Core Interface

### Tool

The base interface that all tools must implement.

```go
type Tool interface {
    Name() string
    Description() string
    Parameters() map[string]interface{}
    Execute(ctx context.Context, args map[string]interface{}) *ToolResult
}
```

#### Methods

##### Name() string

Returns the unique identifier for the tool.

- **Returns**: Tool name (e.g., "files_read", "exec", "message")
- **Usage**: Used for tool lookup and LLM function calling

```go
func (t *MyTool) Name() string {
    return "my-tool"
}
```

##### Description() string

Returns a human-readable description of what the tool does.

- **Returns**: Description text
- **Usage**: Helps the LLM understand when to use this tool

```go
func (t *MyTool) Description() string {
    return "Performs a specific action. Use this when you need to..."
}
```

##### Parameters() map[string]interface{}

Returns the JSON Schema for the tool's parameters.

- **Returns**: JSON Schema object
- **Usage**: Defines the structure of arguments for LLM function calling

```go
func (t *MyTool) Parameters() map[string]interface{} {
    return map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "path": map[string]interface{}{
                "type":        "string",
                "description": "File path to read",
            },
            "encoding": map[string]interface{}{
                "type":        "string",
                "default":     "utf-8",
                "description": "File encoding",
            },
        },
        "required": []string{"path"},
    }
}
```

##### Execute(ctx context.Context, args map[string]interface{}) *ToolResult

Executes the tool with the provided arguments.

- **Parameters**:
  - `ctx`: Context for cancellation and timeout
  - `args`: Arguments matching the Parameters schema
- **Returns**: ToolResult with execution outcome

```go
func (t *MyTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
    path, ok := args["path"].(string)
    if !ok {
        return ErrorResult("path is required")
    }

    // Do work...
    content, err := os.ReadFile(path)
    if err != nil {
        return ErrorResult(fmt.Sprintf("failed to read: %v", err))
    }

    return UserResult(string(content))
}
```

## Optional Interfaces

### ContextualTool

For tools that need channel/chat context.

```go
type ContextualTool interface {
    Tool
    SetContext(channel, chatID string)
}
```

#### SetContext(channel, chatID string)

Called before Execute to provide message context.

- **Parameters**:
  - `channel`: Platform identifier (e.g., "telegram")
  - `chatID`: Chat/conversation identifier

```go
type MessageTool struct {
    defaultChannel string
    defaultChatID  string
}

func (t *MessageTool) SetContext(channel, chatID string) {
    t.defaultChannel = channel
    t.defaultChatID = chatID
}
```

### AsyncTool

For tools that execute asynchronously.

```go
type AsyncCallback func(ctx context.Context, result *ToolResult)

type AsyncTool interface {
    Tool
    SetCallback(cb AsyncCallback)
}
```

#### SetCallback(cb AsyncCallback)

Registers a callback for async completion notification.

- **Parameters**:
  - `cb`: Callback function to invoke when complete

```go
type SpawnTool struct {
    callback AsyncCallback
}

func (t *SpawnTool) SetCallback(cb AsyncCallback) {
    t.callback = cb
}

func (t *SpawnTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
    go func() {
        // Do async work
        result := doWork(args)

        // Notify completion
        if t.callback != nil {
            t.callback(ctx, result)
        }
    }()

    return AsyncResult("Task started")
}
```

## ToolResult

The return type for tool execution.

```go
type ToolResult struct {
    ForLLM  string `json:"for_llm"`
    ForUser string `json:"for_user,omitempty"`
    Silent  bool   `json:"silent"`
    IsError bool   `json:"is_error"`
    Async   bool   `json:"async"`
    Err     error  `json:"-"`
}
```

### Fields

| Field | Type | Description |
|-------|------|-------------|
| ForLLM | string | Content sent to LLM (always required) |
| ForUser | string | Content sent directly to user |
| Silent | bool | If true, no message sent to user |
| IsError | bool | Indicates execution failure |
| Async | bool | Indicates async operation |
| Err | error | Underlying error (not JSON serialized) |

### Constructor Functions

#### NewToolResult(forLLM string) *ToolResult

Creates a basic result with LLM content.

```go
result := NewToolResult("Operation completed")
```

#### SilentResult(forLLM string) *ToolResult

Creates a silent result (no user message).

```go
result := SilentResult("File saved")
```

#### UserResult(content string) *ToolResult

Creates a result with same content for LLM and user.

```go
result := UserResult("Found 42 items")
```

#### ErrorResult(message string) *ToolResult

Creates an error result.

```go
result := ErrorResult("Failed to process: reason")
```

#### AsyncResult(forLLM string) *ToolResult

Creates an async result for background operations.

```go
result := AsyncResult("Task started, will notify on completion")
```

### Methods

#### WithError(err error) *ToolResult

Attaches an underlying error for logging.

```go
result := ErrorResult("Failed").WithError(err)
```

## ToolRegistry

Manages registered tools.

```go
type ToolRegistry struct {
    tools map[string]Tool
    mu    sync.RWMutex
}
```

### Methods

#### NewToolRegistry() *ToolRegistry

Creates a new empty registry.

```go
registry := tools.NewToolRegistry()
```

#### Register(tool Tool)

Registers a tool.

```go
registry.Register(tools.NewMessageTool())
```

#### Get(name string) (Tool, bool)

Retrieves a tool by name.

```go
tool, ok := registry.Get("message")
```

#### Execute(ctx context.Context, name string, args map[string]interface{}) *ToolResult

Executes a tool by name.

```go
result := registry.Execute(ctx, "message", map[string]interface{}{
    "content": "Hello",
})
```

#### ExecuteWithContext(ctx context.Context, name string, args map[string]interface{}, channel, chatID string, asyncCallback AsyncCallback) *ToolResult

Executes with context and optional async callback.

```go
result := registry.ExecuteWithContext(ctx, "spawn", args, "telegram", "123", callback)
```

#### GetDefinitions() []map[string]interface{}

Returns all tool definitions in JSON Schema format.

```go
defs := registry.GetDefinitions()
```

#### ToProviderDefs() []providers.ToolDefinition

Returns definitions in provider-compatible format.

```go
providerDefs := registry.ToProviderDefs()
```

#### List() []string

Returns all registered tool names.

```go
names := registry.List() // ["message", "exec", "files_read", ...]
```

#### Count() int

Returns the number of registered tools.

```go
count := registry.Count()
```

#### GetSummaries() []string

Returns human-readable summaries.

```go
summaries := registry.GetSummaries()
// ["- `message` - Send a message to user", ...]
```

## Helper Functions

### ToolToSchema(tool Tool) map[string]interface{}

Converts a tool to JSON Schema format.

```go
schema := tools.ToolToSchema(myTool)
// {
//   "type": "function",
//   "function": {
//     "name": "my-tool",
//     "description": "...",
//     "parameters": {...}
//   }
// }
```

## Parameter Schema Guide

### Basic Types

```go
// String
"name": map[string]interface{}{
    "type":        "string",
    "description": "User name",
}

// Integer
"count": map[string]interface{}{
    "type":        "integer",
    "description": "Number of items",
    "minimum":     1,
    "maximum":     100,
}

// Number (float)
"ratio": map[string]interface{}{
    "type":        "number",
    "description": "Ratio value",
    "minimum":     0.0,
    "maximum":     1.0,
}

// Boolean
"enabled": map[string]interface{}{
    "type":        "boolean",
    "description": "Enable feature",
    "default":     false,
}
```

### Enumerations

```go
"mode": map[string]interface{}{
    "type":        "string",
    "enum":        []string{"fast", "normal", "thorough"},
    "description": "Processing mode",
}
```

### Arrays

```go
"tags": map[string]interface{}{
    "type":        "array",
    "items":       map[string]interface{}{"type": "string"},
    "description": "List of tags",
}
```

### Nested Objects

```go
"options": map[string]interface{}{
    "type":        "object",
    "properties": map[string]interface{}{
        "timeout": map[string]interface{}{
            "type":        "integer",
            "description": "Timeout in seconds",
        },
        "retries": map[string]interface{}{
            "type":        "integer",
            "description": "Number of retries",
        },
    },
}
```

### Required Fields

```go
map[string]interface{}{
    "type": "object",
    "properties": map[string]interface{}{
        "path": map[string]interface{}{...},
        "mode": map[string]interface{}{...},
    },
    "required": []string{"path"},  // mode is optional
}
```

## Best Practices

1. **Clear Names**: Use lowercase with underscores
2. **Good Descriptions**: Explain when and how to use the tool
3. **Validate Inputs**: Check all arguments before processing
4. **Meaningful Errors**: Provide actionable error messages
5. **Use Context**: Support cancellation via context
6. **Choose Right Result Type**: Match result type to use case
7. **Handle Panics**: Recover in Execute to prevent crashes

## See Also

- [Creating Custom Tools](../extending/creating-tools.md)
- [Tool Implementations](https://github.com/sipeed/picoclaw/tree/main/pkg/tools)

# API Reference

This directory contains reference documentation for PicoClaw's core APIs.

## Overview

PicoClaw provides several key APIs for extending and integrating with the system:

| API | Description | Documentation |
|-----|-------------|---------------|
| Tool Interface | Create tools for agent capabilities | [tool-interface.md](tool-interface.md) |
| Provider Interface | Implement LLM provider integrations | [provider-interface.md](provider-interface.md) |
| Message Bus | Async message passing system | [message-bus.md](message-bus.md) |
| Session Manager | Conversation history management | [session-api.md](session-api.md) |

## Core Types

### Message Types

```go
// Inbound message from external platform
type InboundMessage struct {
    Channel    string
    SenderID   string
    ChatID     string
    Content    string
    Media      []string
    SessionKey string
    Metadata   map[string]string
}

// Outbound message to external platform
type OutboundMessage struct {
    Channel string
    ChatID  string
    Content string
}
```

### LLM Types

```go
// Message for LLM conversation
type Message struct {
    Role       string
    Content    string
    ToolCalls  []ToolCall
    ToolCallID string
}

// LLM response
type LLMResponse struct {
    Content      string
    ToolCalls    []ToolCall
    FinishReason string
    Usage        *UsageInfo
}

// Tool definition for LLM
type ToolDefinition struct {
    Type     string
    Function ToolFunctionDefinition
}
```

### Tool Types

```go
// Tool result from execution
type ToolResult struct {
    ForLLM  string
    ForUser string
    Silent  bool
    IsError bool
    Async   bool
    Err     error
}
```

## Package Overview

### pkg/tools

Tool system for agent capabilities.

Key types:
- `Tool` - Base tool interface
- `ContextualTool` - Tools with channel context
- `AsyncTool` - Asynchronous tools
- `ToolResult` - Execution results
- `ToolRegistry` - Tool management

### pkg/providers

LLM provider implementations.

Key types:
- `LLMProvider` - Provider interface
- `Message` - Conversation message
- `LLMResponse` - LLM response
- `FailoverError` - Error with classification
- `FallbackChain` - Model fallback logic

### pkg/bus

Message bus for async communication.

Key types:
- `MessageBus` - Bus implementation
- `InboundMessage` - Incoming messages
- `OutboundMessage` - Outgoing messages

### pkg/session

Session and history management.

Key types:
- `Session` - Conversation session
- `SessionManager` - Session management

### pkg/channels

Platform integrations.

Key types:
- `Channel` - Channel interface
- `BaseChannel` - Common functionality

### pkg/skills

Skill loading system.

Key types:
- `SkillsLoader` - Skill discovery
- `SkillInfo` - Skill metadata

## Quick Reference

### Implementing a Tool

```go
type MyTool struct{}

func (t *MyTool) Name() string { return "my-tool" }
func (t *MyTool) Description() string { return "Does something" }
func (t *MyTool) Parameters() map[string]interface{} {
    return map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "input": map[string]interface{}{"type": "string"},
        },
        "required": []string{"input"},
    }
}
func (t *MyTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
    return UserResult("Result")
}
```

### Implementing a Provider

```go
type MyProvider struct{}

func (p *MyProvider) Chat(ctx context.Context, messages []Message,
    tools []ToolDefinition, model string,
    options map[string]interface{}) (*LLMResponse, error) {
    // Implementation
}

func (p *MyProvider) GetDefaultModel() string {
    return "my-model"
}
```

### Using the Message Bus

```go
bus := bus.NewMessageBus()

// Publish inbound
bus.PublishInbound(bus.InboundMessage{
    Channel: "telegram",
    ChatID:  "123",
    Content: "Hello",
})

// Consume inbound
msg, ok := bus.ConsumeInbound(ctx)

// Publish outbound
bus.PublishOutbound(bus.OutboundMessage{
    Channel: "telegram",
    ChatID:  "123",
    Content: "Hi there!",
})
```

### Using Session Manager

```go
sm := session.NewSessionManager(storagePath)

// Get or create session
session := sm.GetOrCreate("telegram:123")

// Add message
sm.AddMessage("telegram:123", "user", "Hello")

// Get history
history := sm.GetHistory("telegram:123")

// Save
sm.Save("telegram:123")
```

## See Also

- [Creating Tools](../extending/creating-tools.md)
- [Creating Providers](../extending/creating-providers.md)
- [Creating Channels](../extending/creating-channels.md)
- [Architecture Overview](../architecture.md)

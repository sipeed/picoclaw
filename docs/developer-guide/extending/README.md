# Extending PicoClaw

This directory contains guides for extending PicoClaw with custom functionality.

## Overview

PicoClaw is designed to be extensible. The main extension points are:

1. **Tools** - Add new capabilities for the agent to use
2. **LLM Providers** - Add support for new LLM APIs
3. **Channels** - Add integrations with new chat platforms
4. **Skills** - Create reusable prompt templates

## Extension Points

### [Creating Custom Tools](creating-tools.md)

Tools are the primary way agents interact with the world. Create custom tools to:

- Integrate with external APIs
- Add domain-specific functionality
- Interact with local systems

Example use cases:
- Database queries
- File format conversion
- API integrations
- Custom calculations

### [Creating LLM Providers](creating-providers.md)

Providers handle communication with LLM APIs. Create new providers to:

- Support new LLM services
- Implement custom authentication
- Add specialized request/response handling

Example use cases:
- New AI service integrations
- Local LLM support
- Custom API gateways

### [Creating Channel Integrations](creating-channels.md)

Channels connect PicoClaw to chat platforms. Create new channels to:

- Support new messaging platforms
- Add custom communication protocols
- Integrate with internal systems

Example use cases:
- Team chat platforms
- Custom bots
- Internal communication systems

### [Creating Skills](creating-skills.md)

Skills are reusable prompt templates. Create skills to:

- Package domain expertise
- Create reusable workflows
- Share capabilities across agents

Example use cases:
- Code review assistance
- Document analysis
- Domain-specific knowledge

## Architecture for Extensions

```
┌─────────────────────────────────────────────────────────────┐
│                        Agent Loop                            │
│                                                              │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │    Tools     │  │   Provider   │  │   Channels   │      │
│  │  Registry    │  │   Factory    │  │   Manager    │      │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘      │
│         │                 │                 │               │
│         ▼                 ▼                 ▼               │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │ Custom Tool  │  │   Custom     │  │   Custom     │      │
│  │  (Yours)     │  │  Provider    │  │   Channel    │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
└─────────────────────────────────────────────────────────────┘
```

## Getting Started

Choose an extension type based on what you want to accomplish:

| Goal | Extension Type | Guide |
|------|---------------|-------|
| Add new capabilities | Tool | [Creating Tools](creating-tools.md) |
| Use new LLM service | Provider | [Creating Providers](creating-providers.md) |
| Connect new platform | Channel | [Creating Channels](creating-channels.md) |
| Share prompts | Skill | [Creating Skills](creating-skills.md) |

## Quick Reference

### Tool Interface

```go
type Tool interface {
    Name() string
    Description() string
    Parameters() map[string]interface{}
    Execute(ctx context.Context, args map[string]interface{}) *ToolResult
}
```

### Provider Interface

```go
type LLMProvider interface {
    Chat(ctx context.Context, messages []Message, tools []ToolDefinition,
         model string, options map[string]interface{}) (*LLMResponse, error)
    GetDefaultModel() string
}
```

### Channel Interface

```go
type Channel interface {
    Name() string
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    Send(ctx context.Context, msg bus.OutboundMessage) error
    IsRunning() bool
    IsAllowed(senderID string) bool
}
```

## Development Workflow

1. **Plan** - Define what you want to build
2. **Implement** - Create the extension following the guide
3. **Test** - Write tests for your extension
4. **Integrate** - Register your extension with PicoClaw
5. **Document** - Add documentation for users

## Contributing Extensions

If you create a useful extension, consider contributing it back:

1. Fork the repository
2. Add your extension
3. Add tests
4. Submit a pull request

See [Contributing Guidelines](../contributing.md) for details.

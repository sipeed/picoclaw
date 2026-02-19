# Code Style Guide

This document describes the code style conventions used in PicoClaw.

## Overview

PicoClaw follows standard Go conventions with some project-specific guidelines. When in doubt, refer to:

- [Effective Go](https://golang.org/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)

## Formatting

### Automatic Formatting

Use `gofmt` or `go fmt`:

```bash
# Format all code
make fmt

# Or manually
go fmt ./pkg/...
```

### General Rules

- Use tabs for indentation (Go standard)
- No trailing whitespace
- Files end with a newline

## Naming Conventions

### Packages

- Short, lowercase names
- No underscores or mixedCaps
- Single word preferred

```go
// Good
package agent
package tools
package bus

// Bad
package agentLoop
package tool_registry
```

### Types

- Exported types: PascalCase
- Private types: camelCase
- Interface names: typically end with "er" for single-method interfaces

```go
// Good
type Tool interface { ... }
type LLMProvider interface { ... }
type toolRegistry struct { ... }

// Bad
type tool_interface interface { ... }
type ToolRegistry struct { ... }  // if private
```

### Functions and Methods

- Exported: PascalCase
- Private: camelCase
- Constructors: `NewTypeName()`

```go
// Good
func NewToolRegistry() *ToolRegistry
func (r *ToolRegistry) Register(tool Tool)
func (r *toolRegistry) getTool(name string) Tool

// Bad
func new_ToolRegistry() *ToolRegistry
func (r *ToolRegistry) register_tool(tool Tool)
```

### Variables

- Short names for short scope
- Descriptive names for longer scope
- camelCase for local variables

```go
// Good - short scope
for i, msg := range messages {
    fmt.Println(i, msg)
}

// Good - longer scope
sessionManager := session.NewManager(storage)

// Bad
messageIndex := 0  // too long for short scope
```

### Constants

- camelCase or PascalCase depending on export

```go
const (
    maxIterations  = 10
    DefaultModel   = "gpt-4"
)
```

## Error Handling

### Error Wrapping

Use `fmt.Errorf` with `%w` for wrapping:

```go
// Good
if err != nil {
    return fmt.Errorf("failed to process message: %w", err)
}

// Bad
if err != nil {
    return fmt.Errorf("failed to process message: %v", err)
}
```

### Error Returns

Return errors, don't panic:

```go
// Good
func (t *Tool) Execute(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
    if err := validate(args); err != nil {
        return nil, fmt.Errorf("validation failed: %w", err)
    }
    // ...
}

// Bad
func (t *Tool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
    if err := validate(args); err != nil {
        panic(err)  // Don't panic in library code
    }
    // ...
}
```

### Custom Errors

Define custom error types when appropriate:

```go
type FailoverError struct {
    Reason   FailoverReason
    Provider string
    Model    string
    Status   int
    Wrapped  error
}

func (e *FailoverError) Error() string {
    return fmt.Sprintf("failover(%s): provider=%s model=%s status=%d: %v",
        e.Reason, e.Provider, e.Model, e.Status, e.Wrapped)
}

func (e *FailoverError) Unwrap() error {
    return e.Wrapped
}
```

## Logging

### Structured Logging

Use the project's logger package:

```go
// Good
logger.InfoCF("agent", "Processing message",
    map[string]interface{}{
        "channel": msg.Channel,
        "chat_id": msg.ChatID,
        "sender":  msg.SenderID,
    })

logger.ErrorCF("tool", "Tool execution failed",
    map[string]interface{}{
        "tool":  name,
        "error": err.Error(),
    })

// Bad
log.Printf("Processing message from %s", msg.Channel)
```

### Log Levels

- `DebugCF`: Detailed information for debugging
- `InfoCF`: General operational information
- `WarnCF`: Warning conditions
- `ErrorCF`: Error conditions

### Component Names

Use consistent component names:

- `"agent"` - Agent loop operations
- `"tool"` - Tool execution
- `"channel"` - Channel operations
- `"provider"` - LLM provider operations
- `"session"` - Session management

## Comments

### Package Comments

Document packages with a package comment:

```go
// Package agent provides the core agent loop for processing messages.
// It handles message routing, LLM interaction, and tool execution.
package agent
```

### Function Comments

Document exported functions:

```go
// Execute processes the tool call with the given arguments.
// It returns a ToolResult containing the output for both the LLM and user.
//
// The args map must contain the parameters defined by Parameters().
// If a required parameter is missing, an error result is returned.
func (t *MessageTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
```

### Inline Comments

Use sparingly and only when necessary:

```go
// Check for async tools and inject callback if needed
if asyncTool, ok := tool.(AsyncTool); ok && asyncCallback != nil {
    asyncTool.SetCallback(asyncCallback)
}
```

### TODO Comments

```go
// TODO(username): Add support for streaming responses
// TODO: Consider caching provider responses
```

## Code Organization

### File Organization

Order within a file:

1. Package declaration
2. Imports
3. Constants
4. Variables
5. Types
6. Functions (exported first, then private)

### Import Organization

Group imports:

```go
import (
    // Standard library
    "context"
    "fmt"

    // External packages
    "github.com/something/external"

    // Internal packages
    "github.com/sipeed/picoclaw/pkg/tools"
)
```

### Function Length

Keep functions focused and reasonably sized:

- Single responsibility
- Break up long functions
- Extract helper functions when needed

## Interfaces

### Define Interfaces Where Used

Define interfaces in the package that uses them:

```go
// In pkg/agent/loop.go
type LLMProvider interface {
    Chat(ctx context.Context, messages []providers.Message, ...) (*LLMResponse, error)
}
```

### Small Interfaces

Prefer small, focused interfaces:

```go
// Good
type Tool interface {
    Name() string
    Description() string
    Parameters() map[string]interface{}
    Execute(ctx context.Context, args map[string]interface{}) *ToolResult
}

type ContextualTool interface {
    Tool
    SetContext(channel, chatID string)
}

// Bad - too large
type MegaInterface interface {
    Name() string
    Description() string
    Parameters() map[string]interface{}
    Execute(ctx context.Context, args map[string]interface{}) *ToolResult
    SetContext(channel, chatID string)
    SetCallback(cb AsyncCallback)
    Start() error
    Stop() error
}
```

## Testing

### Test File Naming

- Test files: `*_test.go`
- Integration tests: `*_integration_test.go`

### Test Function Naming

```go
func TestFunctionName(t *testing.T) {}
func TestFunctionName_Scenario(t *testing.T) {}
func TestType_Method(t *testing.T) {}
```

### Table-Driven Tests

```go
func TestValidateInput(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        wantErr bool
    }{
        {"valid", "hello", false},
        {"empty", "", true},
        {"whitespace", "   ", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := validateInput(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("validateInput() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

## Concurrency

### Context for Cancellation

Always accept context:

```go
func (p *Provider) Chat(ctx context.Context, messages []Message, ...) (*Response, error) {
    req, _ := http.NewRequestWithContext(ctx, "POST", url, body)
    // ...
}
```

### Goroutine Safety

Document thread-safety:

```go
// ToolRegistry is safe for concurrent use.
type ToolRegistry struct {
    tools map[string]Tool
    mu    sync.RWMutex
}
```

### Wait for Goroutines

```go
var wg sync.WaitGroup
for _, item := range items {
    wg.Add(1)
    go func(i Item) {
        defer wg.Done()
        process(i)
}
wg.Wait()
```

## JSON Configuration

### Field Tags

Use snake_case for JSON field names:

```go
type Config struct {
    APIKey     string `json:"api_key"`
    MaxRetries int    `json:"max_retries"`
    ModelName  string `json:"model_name"`
}
```

## Avoid These Patterns

### Global Variables

Avoid global state; use dependency injection:

```go
// Bad
var globalConfig *Config

func Process() {
    apiKey := globalConfig.APIKey
}

// Good
type Processor struct {
    config *Config
}

func NewProcessor(config *Config) *Processor {
    return &Processor{config: config}
}

func (p *Processor) Process() {
    apiKey := p.config.APIKey
}
```

### Interface Pollution

Don't define interfaces prematurely:

```go
// Bad - interface for single implementation
type StringProcessor interface {
    Process(s string) string
}

type UpperCaseProcessor struct{}

// Good - define interface when you have multiple implementations
// or when mocking for tests
```

## Summary

1. Run `make fmt` before committing
2. Follow standard Go conventions
3. Use structured logging
4. Wrap errors with context
5. Write tests for new code
6. Document exported functions
7. Keep interfaces small

For questions about style not covered here, refer to [Effective Go](https://golang.org/doc/effective_go).

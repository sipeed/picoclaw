# Creating Custom Tools

This guide explains how to create custom tools for PicoClaw.

## Overview

Tools are the primary way agents interact with the world. Each tool implements the `Tool` interface and can be invoked by the LLM during message processing.

## Tool Interface

All tools must implement this interface:

```go
type Tool interface {
    Name() string
    Description() string
    Parameters() map[string]interface{}
    Execute(ctx context.Context, args map[string]interface{}) *ToolResult
}
```

### Optional Interfaces

Tools can also implement optional interfaces for additional functionality:

```go
// ContextualTool receives channel/chat context
type ContextualTool interface {
    Tool
    SetContext(channel, chatID string)
}

// AsyncTool supports asynchronous execution
type AsyncTool interface {
    Tool
    SetCallback(cb AsyncCallback)
}
```

## Creating a Basic Tool

### Step 1: Define the Tool Struct

```go
package tools

import (
    "context"
    "fmt"
)

// CalculatorTool performs arithmetic operations
type CalculatorTool struct {
    // Add any configuration fields here
}

func NewCalculatorTool() *CalculatorTool {
    return &CalculatorTool{}
}
```

### Step 2: Implement Required Methods

```go
// Name returns the tool's identifier
func (t *CalculatorTool) Name() string {
    return "calculator"
}

// Description explains what the tool does
func (t *CalculatorTool) Description() string {
    return "Perform arithmetic calculations. Use this tool for mathematical operations."
}

// Parameters defines the JSON schema for arguments
func (t *CalculatorTool) Parameters() map[string]interface{} {
    return map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "operation": map[string]interface{}{
                "type":        "string",
                "enum":        []string{"add", "subtract", "multiply", "divide"},
                "description": "The arithmetic operation to perform",
            },
            "a": map[string]interface{}{
                "type":        "number",
                "description": "First operand",
            },
            "b": map[string]interface{}{
                "type":        "number",
                "description": "Second operand",
            },
        },
        "required": []string{"operation", "a", "b"},
    }
}
```

### Step 3: Implement Execute

```go
// Execute performs the tool operation
func (t *CalculatorTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
    // Extract arguments with type checking
    operation, ok := args["operation"].(string)
    if !ok {
        return ErrorResult("operation must be a string")
    }

    a, ok := toFloat64(args["a"])
    if !ok {
        return ErrorResult("a must be a number")
    }

    b, ok := toFloat64(args["b"])
    if !ok {
        return ErrorResult("b must be a number")
    }

    // Perform calculation
    var result float64
    switch operation {
    case "add":
        result = a + b
    case "subtract":
        result = a - b
    case "multiply":
        result = a * b
    case "divide":
        if b == 0 {
            return ErrorResult("division by zero")
        }
        result = a / b
    default:
        return ErrorResult(fmt.Sprintf("unknown operation: %s", operation))
    }

    // Return result
    return UserResult(fmt.Sprintf("%.2f", result))
}

// Helper function to convert interface{} to float64
func toFloat64(v interface{}) (float64, bool) {
    switch val := v.(type) {
    case float64:
        return val, true
    case float32:
        return float64(val), true
    case int:
        return float64(val), true
    case int64:
        return float64(val), true
    default:
        return 0, false
    }
}
```

## Tool Results

PicoClaw provides several result types:

```go
// Basic result - content for LLM only
result := NewToolResult("Operation completed")

// Silent result - no user message, LLM only
result := SilentResult("File saved successfully")

// User result - same content for LLM and user
result := UserResult("Found 42 files")

// Error result
result := ErrorResult("Operation failed: reason")

// Async result
result := AsyncResult("Task started, will report back")
```

### Advanced Result Usage

```go
// Different content for LLM and user
result := &ToolResult{
    ForLLM:  "Technical details here...",
    ForUser: "User-friendly summary here",
    Silent:  false,
}

// Error with underlying error
result := ErrorResult("Failed to connect").WithError(err)
```

## Contextual Tools

Tools that need channel/chat context implement `ContextualTool`:

```go
type NotificationTool struct {
    defaultChannel string
    defaultChatID  string
}

func (t *NotificationTool) SetContext(channel, chatID string) {
    t.defaultChannel = channel
    t.defaultChatID = chatID
}

func (t *NotificationTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
    // Use t.defaultChannel and t.defaultChatID
    return UserResult(fmt.Sprintf("Notification sent to %s:%s", t.defaultChannel, t.defaultChatID))
}
```

## Async Tools

For long-running operations, implement `AsyncTool`:

```go
type LongRunningTool struct {
    callback AsyncCallback
}

func (t *LongRunningTool) SetCallback(cb AsyncCallback) {
    t.callback = cb
}

func (t *LongRunningTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
    // Start async work
    go func() {
        // Do long-running work
        time.Sleep(10 * time.Second)
        result := UserResult("Long operation completed")

        // Notify completion
        if t.callback != nil {
            t.callback(ctx, result)
        }
    }()

    // Return immediately
    return AsyncResult("Operation started, will complete in 10 seconds")
}
```

## Registering Tools

### Method 1: In AgentLoop

```go
// Register with all agents
agentLoop.RegisterTool(myTool)
```

### Method 2: In Tool Registry

```go
// Create registry
registry := tools.NewToolRegistry()

// Register tools
registry.Register(tools.NewCalculatorTool())
registry.Register(tools.NewMessageTool())

// Use in agent
agent.Tools = registry
```

## Complete Example: Database Query Tool

```go
package tools

import (
    "context"
    "database/sql"
    "encoding/json"
    "fmt"

    _ "github.com/mattn/go-sqlite3"
)

type DatabaseTool struct {
    db *sql.DB
}

func NewDatabaseTool(dbPath string) (*DatabaseTool, error) {
    db, err := sql.Open("sqlite3", dbPath)
    if err != nil {
        return nil, err
    }
    return &DatabaseTool{db: db}, nil
}

func (t *DatabaseTool) Name() string {
    return "database"
}

func (t *DatabaseTool) Description() string {
    return "Execute SQL queries on the database. Use for data retrieval and analysis."
}

func (t *DatabaseTool) Parameters() map[string]interface{} {
    return map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "query": map[string]interface{}{
                "type":        "string",
                "description": "SQL query to execute (SELECT only)",
            },
        },
        "required": []string{"query"},
    }
}

func (t *DatabaseTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
    query, ok := args["query"].(string)
    if !ok {
        return ErrorResult("query must be a string")
    }

    // Security: Only allow SELECT queries
    if !isSelectQuery(query) {
        return ErrorResult("only SELECT queries are allowed")
    }

    // Execute query
    rows, err := t.db.QueryContext(ctx, query)
    if err != nil {
        return ErrorResult(fmt.Sprintf("query failed: %v", err))
    }
    defer rows.Close()

    // Get column names
    columns, err := rows.Columns()
    if err != nil {
        return ErrorResult(fmt.Sprintf("failed to get columns: %v", err))
    }

    // Collect results
    var results []map[string]interface{}
    for rows.Next() {
        values := make([]interface{}, len(columns))
        valuePtrs := make([]interface{}, len(columns))
        for i := range values {
            valuePtrs[i] = &values[i]
        }

        if err := rows.Scan(valuePtrs...); err != nil {
            return ErrorResult(fmt.Sprintf("scan failed: %v", err))
        }

        row := make(map[string]interface{})
        for i, col := range columns {
            row[col] = values[i]
        }
        results = append(results, row)
    }

    // Format result
    jsonResult, _ := json.MarshalIndent(results, "", "  ")
    return UserResult(fmt.Sprintf("Found %d rows:\n%s", len(results), string(jsonResult)))
}

func isSelectQuery(query string) bool {
    // Simple check - in production, use proper SQL parsing
    return len(query) > 6 && query[:6] == "SELECT"
}
```

## Testing Tools

```go
package tools

import (
    "context"
    "testing"
)

func TestCalculatorTool(t *testing.T) {
    tool := NewCalculatorTool()

    // Test Name
    if tool.Name() != "calculator" {
        t.Errorf("expected name 'calculator', got %s", tool.Name())
    }

    // Test addition
    result := tool.Execute(context.Background(), map[string]interface{}{
        "operation": "add",
        "a":         5,
        "b":         3,
    })

    if result.IsError {
        t.Errorf("unexpected error: %s", result.ForLLM)
    }
    if result.ForLLM != "8.00" {
        t.Errorf("expected '8.00', got %s", result.ForLLM)
    }

    // Test division by zero
    result = tool.Execute(context.Background(), map[string]interface{}{
        "operation": "divide",
        "a":         5,
        "b":         0,
    })

    if !result.IsError {
        t.Error("expected error for division by zero")
    }
}
```

## Best Practices

1. **Clear Names**: Use descriptive, lowercase names with underscores
2. **Good Descriptions**: Explain what the tool does and when to use it
3. **Validate Inputs**: Check types and values before processing
4. **Meaningful Errors**: Return helpful error messages
5. **Security First**: Validate and sanitize all inputs
6. **Use Context**: Support cancellation for long operations
7. **Appropriate Results**: Choose the right result type
8. **Test Thoroughly**: Write tests for all scenarios

## Parameter Schema

The `Parameters()` method returns a JSON Schema object:

```go
func (t *MyTool) Parameters() map[string]interface{} {
    return map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "required_param": map[string]interface{}{
                "type":        "string",
                "description": "Description of the parameter",
            },
            "optional_param": map[string]interface{}{
                "type":        "integer",
                "description": "Optional parameter",
                "default":     10,
            },
            "enum_param": map[string]interface{}{
                "type":        "string",
                "enum":        []string{"option1", "option2"},
                "description": "Parameter with limited options",
            },
            "array_param": map[string]interface{}{
                "type":        "array",
                "items":       map[string]interface{}{"type": "string"},
                "description": "Array of strings",
            },
        },
        "required": []string{"required_param"},
    }
}
```

## See Also

- [Tool Interface Reference](../api/tool-interface.md)
- [Tool Result Types](../api/tool-interface.md#toolresult)
- [Existing Tools](https://github.com/sipeed/picoclaw/tree/main/pkg/tools)

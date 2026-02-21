# Running Tests

This guide explains how to run and write tests for PicoClaw.

## Running Tests

### Run All Tests

Run the complete test suite:

```bash
make test
```

Or using Go directly:

```bash
go test ./pkg/...
```

### Run Tests for a Package

Test a specific package:

```bash
go test ./pkg/agent/... -v
go test ./pkg/providers/... -v
go test ./pkg/tools/... -v
```

### Run Specific Tests

Run tests matching a pattern:

```bash
go test ./pkg/providers/... -v -run TestFallbackChain
go test ./pkg/tools/... -v -run TestMessageTool
```

### Run with Verbose Output

```bash
go test ./pkg/... -v
```

### Run with Coverage

Generate coverage reports:

```bash
# Coverage for all packages
go test ./pkg/... -cover

# Detailed coverage report
go test ./pkg/... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

### Run Integration Tests

Integration tests require external APIs and are tagged separately:

```bash
go test ./pkg/... -v -tags=integration
```

Integration tests are located in files with the build tag:

```go
//go:build integration
// +build integration
```

## Test Organization

### Directory Structure

```
pkg/
├── agent/
│   ├── loop.go
│   ├── loop_test.go       # Unit tests
│   └── loop_integration_test.go  # Integration tests
├── providers/
│   ├── fallback.go
│   ├── fallback_test.go
│   └── openai_compat/
│       ├── provider.go
│       └── provider_test.go
└── tools/
    ├── message.go
    ├── message_test.go
    └── ...
```

### Test File Naming

- Unit tests: `*_test.go`
- Integration tests: `*_integration_test.go` (with build tag)

## Writing Tests

### Basic Test

```go
package tools

import (
    "context"
    "testing"
)

func TestMessageTool(t *testing.T) {
    tool := NewMessageTool()

    // Test Name()
    if tool.Name() != "message" {
        t.Errorf("expected name 'message', got %s", tool.Name())
    }

    // Test Description()
    if tool.Description() == "" {
        t.Error("description should not be empty")
    }
}
```

### Table-Driven Tests

```go
func TestToolResult(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
        isError  bool
    }{
        {
            name:     "success",
            input:    "valid input",
            expected: "valid input",
            isError:  false,
        },
        {
            name:     "error case",
            input:    "",
            expected: "input is required",
            isError:  true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := processInput(tt.input)
            if result.ForLLM != tt.expected {
                t.Errorf("expected %q, got %q", tt.expected, result.ForLLM)
            }
            if result.IsError != tt.isError {
                t.Errorf("expected isError=%v, got %v", tt.isError, result.IsError)
            }
        })
    }
}
```

### Testing Tool Execution

```go
func TestMessageToolExecute(t *testing.T) {
    tool := NewMessageTool()

    // Set up mock callback
    var sentMessage string
    tool.SetSendCallback(func(channel, chatID, content string) error {
        sentMessage = content
        return nil
    })

    // Set context
    tool.SetContext("telegram", "123456")

    // Execute
    result := tool.Execute(context.Background(), map[string]interface{}{
        "content": "Hello, world!",
    })

    // Verify
    if result.IsError {
        t.Errorf("unexpected error: %s", result.ForLLM)
    }
    if sentMessage != "Hello, world!" {
        t.Errorf("expected 'Hello, world!', got %q", sentMessage)
    }
}
```

### Testing with Context

```go
func TestWithContext(t *testing.T) {
    ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
    defer cancel()

    result := longRunningOperation(ctx)
    if result == nil {
        t.Error("expected result")
    }
}
```

### Testing Error Cases

```go
func TestErrorHandling(t *testing.T) {
    tool := NewMessageTool()

    // Execute without setting context
    result := tool.Execute(context.Background(), map[string]interface{}{
        "content": "Hello",
    })

    if !result.IsError {
        t.Error("expected error when context not set")
    }
}
```

### Mocking

Create mock implementations for testing:

```go
// Mock provider for testing
type MockProvider struct {
    response *providers.LLMResponse
    err      error
}

func (m *MockProvider) Chat(ctx context.Context, messages []providers.Message,
    tools []providers.ToolDefinition, model string,
    options map[string]interface{}) (*providers.LLMResponse, error) {
    return m.response, m.err
}

func (m *MockProvider) GetDefaultModel() string {
    return "mock-model"
}

func TestWithMockProvider(t *testing.T) {
    mock := &MockProvider{
        response: &providers.LLMResponse{
            Content: "Hello!",
        },
    }

    // Use mock in tests
    // ...
}
```

## Test Utilities

### Helper Functions

```go
// Create a test message bus
func newTestBus() *bus.MessageBus {
    return bus.NewMessageBus()
}

// Create test config
func newTestConfig() *config.Config {
    return &config.Config{
        Agents: config.AgentsConfig{
            Defaults: config.AgentDefaults{
                Model: "test-model",
            },
        },
    }
}
```

### Setup and Teardown

```go
func TestWithSetup(t *testing.T) {
    // Setup
    tmpDir, err := os.MkdirTemp("", "picoclaw-test")
    if err != nil {
        t.Fatal(err)
    }
    defer os.RemoveAll(tmpDir) // Teardown

    // Test using tmpDir
    // ...
}
```

## Benchmark Tests

### Writing Benchmarks

```go
func BenchmarkToolExecute(b *testing.B) {
    tool := NewMessageTool()
    tool.SetContext("telegram", "123456")
    tool.SetSendCallback(func(channel, chatID, content string) error {
        return nil
    })

    args := map[string]interface{}{
        "content": "test message",
    }

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        tool.Execute(context.Background(), args)
    }
}
```

### Running Benchmarks

```bash
go test ./pkg/tools/... -bench=.
go test ./pkg/tools/... -bench=. -benchmem
```

## Continuous Integration

Tests are run automatically in CI. Ensure all tests pass before submitting PRs:

```bash
# Run full check
make check
```

This runs:
1. `make deps` - Download dependencies
2. `make fmt` - Format code
3. `make vet` - Run linter
4. `make test` - Run tests

## Linting

Run the Go linter:

```bash
make vet
```

Or directly:

```bash
go vet ./pkg/...
```

## Common Issues

### Test Fails with "context deadline exceeded"

The test may be timing out. Increase the timeout or check for blocking operations:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
```

### Test Fails with "no such file or directory"

Tests may be looking for files in the wrong location. Use absolute paths or create temp directories:

```go
tmpDir := t.TempDir() // Automatically cleaned up
```

### Race Condition Detected

Run tests with race detection:

```bash
go test ./pkg/... -race
```

Fix race conditions by adding proper synchronization.

## Best Practices

1. **Write tests for new features** - Aim for good coverage
2. **Use table-driven tests** - Test multiple cases efficiently
3. **Test error paths** - Don't just test happy paths
4. **Use mocks for external dependencies** - Isolate unit tests
5. **Keep tests fast** - Use short timeouts and mock slow operations
6. **Clean up resources** - Use `t.Cleanup()` or defer
7. **Use meaningful test names** - Describe what's being tested

## Test Coverage Goals

- Aim for >70% coverage on core packages
- 100% coverage on critical paths (tool execution, message routing)
- Integration tests for end-to-end scenarios

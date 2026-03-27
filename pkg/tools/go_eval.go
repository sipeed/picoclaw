package tools

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
)

type threadSafeBuffer struct {
	b  []byte
	mu sync.Mutex
}

func (b *threadSafeBuffer) Write(p []byte) (n int, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.b = append(b.b, p...)
	return len(p), nil
}

func (b *threadSafeBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return string(b.b)
}

func (b *threadSafeBuffer) Len() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.b)
}

type GoEvalTool struct {
	workspace string
	timeout   time.Duration
	Bindings  map[string]reflect.Value
}

func NewGoEvalTool(workspace string) *GoEvalTool {
	return &GoEvalTool{
		workspace: workspace,
		timeout:   60 * time.Second, // Default timeout
	}
}

func (t *GoEvalTool) SetBindings(bindings map[string]reflect.Value) {
	t.Bindings = bindings
}

func (t *GoEvalTool) Name() string {
	return "go_eval"
}

func (t *GoEvalTool) Description() string {
	return "Executes Go code dynamically using Yaegi interpreter. Provide valid Go source code. The code will be interpreted and executed safely without requiring the Go toolchain. Useful for complex logic or tasks that require writing a Go script."
}

func (t *GoEvalTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"code": map[string]any{
				"type":        "string",
				"description": "Valid Go source code to execute. It does not need to be a complete package with main func, scripts can just be valid Go statements or functions.",
			},
		},
		"required": []string{"code"},
	}
}

func (t *GoEvalTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	code, ok := args["code"].(string)
	if !ok || code == "" {
		return ErrorResult("code is required")
	}

	cmdCtx, cancel := context.WithTimeout(ctx, t.timeout)
	defer cancel()

	var stdout, stderr threadSafeBuffer

	i := interp.New(interp.Options{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{},
	})

	if err := i.Use(stdlib.Symbols); err != nil {
		return &ToolResult{
			ForLLM:  fmt.Sprintf("Failed to initialize standard library symbols: %v", err),
			ForUser: "Execution environment setup failed.",
			IsError: true,
		}
	}

	if t.Bindings != nil && len(t.Bindings) > 0 {
		exports := interp.Exports{
			"jane/env/env": t.Bindings,
			"jane/env":     t.Bindings,
		}
		if err := i.Use(exports); err != nil {
			return &ToolResult{
				ForLLM:  fmt.Sprintf("Failed to initialize injected bindings: %v", err),
				ForUser: "Execution environment setup failed.",
				IsError: true,
			}
		}
	}

	// Channel to capture execution result
	done := make(chan error, 1)
	go func() {
		_, err := i.EvalWithContext(cmdCtx, code)
		done <- err
	}()

	var err error
	select {
	case <-cmdCtx.Done():
		err = cmdCtx.Err()
	case err = <-done:
	}

	output := stdout.String()
	if stderr.Len() > 0 {
		if output != "" {
			output += "\nSTDERR:\n"
		}
		output += stderr.String()
	}

	if err != nil {
		if err == context.DeadlineExceeded {
			return &ToolResult{
				ForLLM:  fmt.Sprintf("Execution timed out after %v.\nOutput so far:\n%s", t.timeout, output),
				ForUser: "Execution timed out.",
				IsError: true,
			}
		}

		return &ToolResult{
			ForLLM:  fmt.Sprintf("Execution failed: %v\nOutput:\n%s", err, output),
			ForUser: "Execution failed.",
			IsError: true,
		}
	}

	if output == "" {
		output = "(no output)"
	}

	// Truncate output if necessary
	maxLen := 10000
	if len(output) > maxLen {
		output = output[:maxLen] + fmt.Sprintf("\n... (truncated, %d more chars)", len(output)-maxLen)
	}

	return &ToolResult{
		ForLLM:  output,
		ForUser: output,
	}
}

func (t *GoEvalTool) RequiresApproval() bool {
	return false
}

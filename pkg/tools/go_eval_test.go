package tools

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestGoEvalTool(t *testing.T) {
	tool := NewGoEvalTool("/tmp/test_workspace")

	ctx := context.Background()

	t.Run("execute simple go code", func(t *testing.T) {
		args := map[string]any{
			"code": `
				import "fmt"
				func main() {
					fmt.Println("hello from yaegi")
				}
			`,
		}

		result := tool.Execute(ctx, args)
		if result.IsError {
			t.Fatalf("Expected no error, got: %s", result.ForLLM)
		}

		if !strings.Contains(result.ForLLM, "hello from yaegi") {
			t.Errorf("Expected output to contain 'hello from yaegi', got %q", result.ForLLM)
		}
	})

	t.Run("execution timeout", func(t *testing.T) {
		tool.timeout = 100 * time.Millisecond
		defer func() { tool.timeout = 60 * time.Second }()

		args := map[string]any{
			"code": `
				import "time"
				func init() {
					time.Sleep(1 * time.Second)
				}
			`,
		}

		result := tool.Execute(ctx, args)
		if !result.IsError {
			t.Fatal("Expected timeout error, got success")
		}
		if !strings.Contains(result.ForLLM, "timed out") {
			t.Errorf("Expected timeout message, got %q", result.ForLLM)
		}
	})
}

var TestWorkspace = "mock_workspace"
var DoMockTask = func() string {
	return "mock task completed"
}

func TestGoEvalToolWithBindings(t *testing.T) {
	tool := NewGoEvalTool("/tmp/test_workspace")

	bindings := map[string]reflect.Value{
		"Workspace": reflect.ValueOf(&TestWorkspace).Elem(),
		"DoTask":    reflect.ValueOf(&DoMockTask).Elem(),
	}

	tool.SetBindings(bindings)

	ctx := context.Background()

	t.Run("execute with bindings", func(t *testing.T) {
		args := map[string]any{
			"code": `
				import "jane/env"
				import "fmt"

				func init() {
					fmt.Println("workspace:", env.Workspace)
					fmt.Println("task:", env.DoTask())
				}
			`,
		}

		result := tool.Execute(ctx, args)
		if result.IsError {
			t.Fatalf("Expected no error, got: %s", result.ForLLM)
		}

		if !strings.Contains(result.ForLLM, "workspace: mock_workspace") {
			t.Errorf("Expected output to contain 'workspace: mock_workspace', got %q", result.ForLLM)
		}

		if !strings.Contains(result.ForLLM, "task: mock task completed") {
			t.Errorf("Expected output to contain 'task: mock task completed', got %q", result.ForLLM)
		}
	})
}

package tools

import (
	"context"
	"testing"
)

func TestVerifyTool(t *testing.T) {
	// Create a temp workspace or just use current dir for simple tests
	tool := NewVerifyTool(".", false)

	t.Run("SuccessCommand", func(t *testing.T) {
		ctx := context.Background()
		args := map[string]interface{}{
			"command": "echo 'ok'",
			"label":   "Check OK",
		}

		result := tool.Execute(ctx, args)
		if result.IsError {
			t.Errorf("Expected success, got error: %v", result.Err)
		}
		if result.Err != nil {
			t.Errorf("Expected nil Err, got: %v", result.Err)
		}
	})

	t.Run("FailureCommand", func(t *testing.T) {
		ctx := context.Background()
		args := map[string]interface{}{
			"command": "exit 1",
			"label":   "Fail Check",
		}

		result := tool.Execute(ctx, args)
		if result.Err == nil {
			t.Error("Expected error for failing command, got nil")
		}
	})

	t.Run("MissingCommand", func(t *testing.T) {
		ctx := context.Background()
		args := map[string]interface{}{}

		result := tool.Execute(ctx, args)
		if !result.IsError {
			t.Error("Expected error for missing command")
		}
	})
}

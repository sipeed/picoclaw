package tools

import (
	"context"
	"errors"
	"testing"
)

func TestPushoverTool_Execute_Success(t *testing.T) {
	tool := NewPushoverTool()

	var sentMessage string
	tool.SetPushoverCallback(func(message string) error {
		sentMessage = message
		return nil
	})

	ctx := context.Background()
	args := map[string]interface{}{
		"message": "Test notification",
	}

	result := tool.Execute(ctx, args)

	if sentMessage != "Test notification" {
		t.Errorf("Expected message 'Test notification', got '%s'", sentMessage)
	}

	if result.IsError {
		t.Error("Expected IsError=false for successful send")
	}

	if result.ForLLM != "Push notification sent: Test notification" {
		t.Errorf("Unexpected ForLLM: %s", result.ForLLM)
	}
}

func TestPushoverTool_Execute_MissingMessage(t *testing.T) {
	tool := NewPushoverTool()
	tool.SetPushoverCallback(func(message string) error {
		return nil
	})

	ctx := context.Background()
	args := map[string]interface{}{}

	result := tool.Execute(ctx, args)

	if !result.IsError {
		t.Error("Expected IsError=true for missing message")
	}

	if result.ForLLM != "message is required" {
		t.Errorf("Expected 'message is required', got '%s'", result.ForLLM)
	}
}

func TestPushoverTool_Execute_NilCallback(t *testing.T) {
	tool := NewPushoverTool()

	ctx := context.Background()
	args := map[string]interface{}{
		"message": "Test notification",
	}

	result := tool.Execute(ctx, args)

	if !result.IsError {
		t.Error("Expected IsError=true for nil callback")
	}

	if result.ForLLM != "Pushover not configured" {
		t.Errorf("Expected 'Pushover not configured', got '%s'", result.ForLLM)
	}
}

func TestPushoverTool_Execute_CallbackError(t *testing.T) {
	tool := NewPushoverTool()
	expectedErr := errors.New("pushover API error")
	tool.SetPushoverCallback(func(message string) error {
		return expectedErr
	})

	ctx := context.Background()
	args := map[string]interface{}{
		"message": "Test notification",
	}

	result := tool.Execute(ctx, args)

	if !result.IsError {
		t.Error("Expected IsError=true for callback error")
	}

	if result.Err != expectedErr {
		t.Errorf("Expected Err to be '%v', got '%v'", expectedErr, result.Err)
	}
}

func TestPushoverTool_Name(t *testing.T) {
	tool := NewPushoverTool()
	if tool.Name() != "pushover" {
		t.Errorf("Expected name 'pushover', got '%s'", tool.Name())
	}
}

func TestPushoverTool_Description(t *testing.T) {
	tool := NewPushoverTool()
	if tool.Description() == "" {
		t.Error("Expected non-empty description")
	}
}

func TestPushoverTool_Parameters(t *testing.T) {
	tool := NewPushoverTool()
	params := tool.Parameters()

	if params["type"] != "object" {
		t.Errorf("Expected type 'object', got '%v'", params["type"])
	}

	props, ok := params["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected properties to be a map")
	}

	if _, exists := props["message"]; !exists {
		t.Error("Expected 'message' property to exist")
	}
}

package tools

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestNewToolResult(t *testing.T) {
	result := NewToolResult("test content")

	if result.ForLLM != "test content" {
		t.Errorf("Expected ForLLM 'test content', got '%s'", result.ForLLM)
	}
	if result.Silent {
		t.Error("Expected Silent to be false")
	}
	if result.IsError {
		t.Error("Expected IsError to be false")
	}
	if result.Async {
		t.Error("Expected Async to be false")
	}
}

func TestSilentResult(t *testing.T) {
	result := SilentResult("silent operation")

	if result.ForLLM != "silent operation" {
		t.Errorf("Expected ForLLM 'silent operation', got '%s'", result.ForLLM)
	}
	if !result.Silent {
		t.Error("Expected Silent to be true")
	}
	if result.IsError {
		t.Error("Expected IsError to be false")
	}
	if result.Async {
		t.Error("Expected Async to be false")
	}
}

func TestAsyncResult(t *testing.T) {
	result := AsyncResult("async task started")

	if result.ForLLM != "async task started" {
		t.Errorf("Expected ForLLM 'async task started', got '%s'", result.ForLLM)
	}
	if result.Silent {
		t.Error("Expected Silent to be false")
	}
	if result.IsError {
		t.Error("Expected IsError to be false")
	}
	if !result.Async {
		t.Error("Expected Async to be true")
	}
}

func TestErrorResult(t *testing.T) {
	result := ErrorResult("operation failed")

	if result.ForLLM != "operation failed" {
		t.Errorf("Expected ForLLM 'operation failed', got '%s'", result.ForLLM)
	}
	if result.Silent {
		t.Error("Expected Silent to be false")
	}
	if !result.IsError {
		t.Error("Expected IsError to be true")
	}
	if result.Async {
		t.Error("Expected Async to be false")
	}
}

func TestUserResult(t *testing.T) {
	content := "user visible message"
	result := UserResult(content)

	if result.ForLLM != content {
		t.Errorf("Expected ForLLM '%s', got '%s'", content, result.ForLLM)
	}
	if result.ForUser != content {
		t.Errorf("Expected ForUser '%s', got '%s'", content, result.ForUser)
	}
	if result.Silent {
		t.Error("Expected Silent to be false")
	}
	if result.IsError {
		t.Error("Expected IsError to be false")
	}
	if result.Async {
		t.Error("Expected Async to be false")
	}
}

func TestToolResultJSONSerialization(t *testing.T) {
	tests := []struct {
		name   string
		result *ToolResult
	}{
		{
			name:   "basic result",
			result: NewToolResult("basic content"),
		},
		{
			name:   "silent result",
			result: SilentResult("silent content"),
		},
		{
			name:   "async result",
			result: AsyncResult("async content"),
		},
		{
			name:   "error result",
			result: ErrorResult("error content"),
		},
		{
			name:   "user result",
			result: UserResult("user content"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal to JSON
			data, err := json.Marshal(tt.result)
			if err != nil {
				t.Fatalf("Failed to marshal: %v", err)
			}

			// Unmarshal back
			var decoded ToolResult
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			// Verify fields match (Err should be excluded)
			if decoded.ForLLM != tt.result.ForLLM {
				t.Errorf("ForLLM mismatch: got '%s', want '%s'", decoded.ForLLM, tt.result.ForLLM)
			}
			if decoded.ForUser != tt.result.ForUser {
				t.Errorf("ForUser mismatch: got '%s', want '%s'", decoded.ForUser, tt.result.ForUser)
			}
			if decoded.Silent != tt.result.Silent {
				t.Errorf("Silent mismatch: got %v, want %v", decoded.Silent, tt.result.Silent)
			}
			if decoded.IsError != tt.result.IsError {
				t.Errorf("IsError mismatch: got %v, want %v", decoded.IsError, tt.result.IsError)
			}
			if decoded.Async != tt.result.Async {
				t.Errorf("Async mismatch: got %v, want %v", decoded.Async, tt.result.Async)
			}
		})
	}
}

func TestToolResultWithErrors(t *testing.T) {
	err := errors.New("underlying error")
	result := ErrorResult("error message").WithError(err)

	if result.Err == nil {
		t.Error("Expected Err to be set")
	}
	if result.Err.Error() != "underlying error" {
		t.Errorf("Expected Err message 'underlying error', got '%s'", result.Err.Error())
	}

	// Verify Err is not serialized
	data, marshalErr := json.Marshal(result)
	if marshalErr != nil {
		t.Fatalf("Failed to marshal: %v", marshalErr)
	}

	var decoded ToolResult
	if unmarshalErr := json.Unmarshal(data, &decoded); unmarshalErr != nil {
		t.Fatalf("Failed to unmarshal: %v", unmarshalErr)
	}

	if decoded.Err != nil {
		t.Error("Expected Err to be nil after JSON round-trip (should not be serialized)")
	}
}

func TestToolResultJSONStructure(t *testing.T) {
	result := UserResult("test content")

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Verify JSON structure
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// Check expected keys exist
	if _, ok := parsed["for_llm"]; !ok {
		t.Error("Expected 'for_llm' key in JSON")
	}
	if _, ok := parsed["for_user"]; !ok {
		t.Error("Expected 'for_user' key in JSON")
	}
	if _, ok := parsed["silent"]; !ok {
		t.Error("Expected 'silent' key in JSON")
	}
	if _, ok := parsed["is_error"]; !ok {
		t.Error("Expected 'is_error' key in JSON")
	}
	if _, ok := parsed["async"]; !ok {
		t.Error("Expected 'async' key in JSON")
	}

	// Check that 'err' is NOT present (it should have json:"-" tag)
	if _, ok := parsed["err"]; ok {
		t.Error("Expected 'err' key to be excluded from JSON")
	}

	// Verify values
	if parsed["for_llm"] != "test content" {
		t.Errorf("Expected for_llm 'test content', got %v", parsed["for_llm"])
	}
	if parsed["silent"] != false {
		t.Errorf("Expected silent false, got %v", parsed["silent"])
	}
}

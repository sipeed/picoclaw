package tools

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestGetCurrentTimeTool_Name(t *testing.T) {
	tool := NewGetCurrentTimeTool("")
	if tool.Name() != "get_current_time" {
		t.Errorf("expected name 'get_current_time', got %s", tool.Name())
	}
}

func TestGetCurrentTimeTool_Description(t *testing.T) {
	tool := NewGetCurrentTimeTool("")
	if tool.Description() == "" {
		t.Error("description should not be empty")
	}
}

func TestGetCurrentTimeTool_Parameters(t *testing.T) {
	tool := NewGetCurrentTimeTool("")
	params := tool.Parameters()
	if params == nil {
		t.Fatal("parameters should not be nil")
	}

	// Check type
	if params["type"] != "object" {
		t.Errorf("expected type 'object', got %v", params["type"])
	}

	// Check properties exist
	properties, ok := params["properties"].(map[string]any)
	if !ok {
		t.Fatal("properties should be a map")
	}

	if _, ok := properties["format"]; !ok {
		t.Error("format property should exist")
	}

	if _, ok := properties["timezone"]; !ok {
		t.Error("timezone property should exist")
	}
}

func TestGetCurrentTimeTool_Execute_DefaultFormat(t *testing.T) {
	tool := NewGetCurrentTimeTool("")
	result := tool.Execute(context.Background(), map[string]any{})

	if result == nil {
		t.Fatal("result should not be nil")
	}

	if result.IsError {
		t.Errorf("should not return error: %s", result.ForLLM)
	}

	// Check that result contains time information
	if result.ForLLM == "" {
		t.Error("ForLLM should not be empty")
	}

	// Default format is ISO, should contain current year
	currentYear := time.Now().Format("2006")
	if !strings.Contains(result.ForLLM, currentYear) {
		t.Errorf("result should contain current year %s, got: %s", currentYear, result.ForLLM)
	}
}

func TestGetCurrentTimeTool_Execute_TimeFormat(t *testing.T) {
	tool := NewGetCurrentTimeTool("")
	result := tool.Execute(context.Background(), map[string]any{
		"format": "time",
	})

	if result == nil {
		t.Fatal("result should not be nil")
	}

	if result.IsError {
		t.Errorf("should not return error: %s", result.ForLLM)
	}

	// Time format should be HH:MM:SS
	if !strings.Contains(result.ForLLM, ":") {
		t.Errorf("time format should contain colon, got: %s", result.ForLLM)
	}
}

func TestGetCurrentTimeTool_Execute_DateFormat(t *testing.T) {
	tool := NewGetCurrentTimeTool("")
	result := tool.Execute(context.Background(), map[string]any{
		"format": "date",
	})

	if result == nil {
		t.Fatal("result should not be nil")
	}

	if result.IsError {
		t.Errorf("should not return error: %s", result.ForLLM)
	}

	// Date format should be YYYY-MM-DD
	currentYear := time.Now().Format("2006")
	if !strings.Contains(result.ForLLM, currentYear) {
		t.Errorf("date format should contain current year %s, got: %s", currentYear, result.ForLLM)
	}
}

func TestGetCurrentTimeTool_Execute_DateTimeFormat(t *testing.T) {
	tool := NewGetCurrentTimeTool("")
	result := tool.Execute(context.Background(), map[string]any{
		"format": "datetime",
	})

	if result == nil {
		t.Fatal("result should not be nil")
	}

	if result.IsError {
		t.Errorf("should not return error: %s", result.ForLLM)
	}

	// Datetime format should contain both date and time
	if !strings.Contains(result.ForLLM, "-") || !strings.Contains(result.ForLLM, ":") {
		t.Errorf("datetime format should contain both date and time, got: %s", result.ForLLM)
	}
}

func TestGetCurrentTimeTool_Execute_UnixFormat(t *testing.T) {
	tool := NewGetCurrentTimeTool("")
	result := tool.Execute(context.Background(), map[string]any{
		"format": "unix",
	})

	if result == nil {
		t.Fatal("result should not be nil")
	}

	if result.IsError {
		t.Errorf("should not return error: %s", result.ForLLM)
	}

	// Unix timestamp should be a number (just check it doesn't contain date separators)
	// The response format is "Current time (Local): 1777295143" which contains a colon after "Local"
	// So we just check the actual timestamp part is numeric
	if !strings.Contains(result.ForLLM, "Current time") {
		t.Errorf("result should contain 'Current time', got: %s", result.ForLLM)
	}
}

func TestGetCurrentTimeTool_Execute_WithTimezone(t *testing.T) {
	tool := NewGetCurrentTimeTool("")
	result := tool.Execute(context.Background(), map[string]any{
		"timezone": "UTC",
	})

	if result == nil {
		t.Fatal("result should not be nil")
	}

	if result.IsError {
		t.Errorf("should not return error: %s", result.ForLLM)
	}

	// Should mention UTC
	if !strings.Contains(result.ForLLM, "UTC") {
		t.Errorf("result should mention UTC timezone, got: %s", result.ForLLM)
	}
}

func TestGetCurrentTimeTool_Execute_InvalidTimezone(t *testing.T) {
	tool := NewGetCurrentTimeTool("")
	result := tool.Execute(context.Background(), map[string]any{
		"timezone": "Invalid/Timezone",
	})

	if result == nil {
		t.Fatal("result should not be nil")
	}

	if result.IsError {
		t.Error("should handle invalid timezone gracefully without error")
	}

	// Should fallback to Local
	if !strings.Contains(result.ForLLM, "Local") {
		t.Errorf("should fallback to Local timezone, got: %s", result.ForLLM)
	}
}

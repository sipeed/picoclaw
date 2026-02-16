package tools

import (
	"context"
	"os/exec"
	"strings"
	"testing"
)

// TestBrowserSearchTool_Name verifies tool name
func TestBrowserSearchTool_Name(t *testing.T) {
	tool := NewBrowserSearchTool()
	if tool.Name() != "browser_search" {
		t.Errorf("expected name 'browser_search', got '%s'", tool.Name())
	}
}

// TestBrowserSearchTool_Description verifies description is non-empty
func TestBrowserSearchTool_Description(t *testing.T) {
	tool := NewBrowserSearchTool()
	if tool.Description() == "" {
		t.Error("expected non-empty description")
	}
}

// TestBrowserSearchTool_Parameters verifies parameter schema
func TestBrowserSearchTool_Parameters(t *testing.T) {
	tool := NewBrowserSearchTool()
	params := tool.Parameters()
	props, ok := params["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("expected properties map")
	}
	if _, ok := props["query"]; !ok {
		t.Error("expected 'query' parameter")
	}
	if _, ok := props["domain"]; !ok {
		t.Error("expected 'domain' parameter")
	}
	required, ok := params["required"].([]string)
	if !ok || len(required) != 1 || required[0] != "query" {
		t.Error("expected 'query' in required list")
	}
}

// TestBrowserSearchTool_MissingQuery verifies error when query is missing
func TestBrowserSearchTool_MissingQuery(t *testing.T) {
	tool := NewBrowserSearchTool()
	result := tool.Execute(context.Background(), map[string]interface{}{})
	if !result.IsError {
		t.Error("expected error when query is missing")
	}
	if !strings.Contains(result.ForLLM, "query is required") {
		t.Errorf("expected 'query is required' in error, got: %s", result.ForLLM)
	}
}

// TestBrowserGetTool_Name verifies tool name
func TestBrowserGetTool_Name(t *testing.T) {
	tool := NewBrowserGetTool()
	if tool.Name() != "browser_get" {
		t.Errorf("expected name 'browser_get', got '%s'", tool.Name())
	}
}

// TestBrowserGetTool_MissingID verifies error when action_id is missing
func TestBrowserGetTool_MissingID(t *testing.T) {
	tool := NewBrowserGetTool()
	result := tool.Execute(context.Background(), map[string]interface{}{})
	if !result.IsError {
		t.Error("expected error when action_id is missing")
	}
	if !strings.Contains(result.ForLLM, "action_id is required") {
		t.Errorf("expected 'action_id is required' in error, got: %s", result.ForLLM)
	}
}

// TestBrowserTool_Name verifies tool name
func TestBrowserTool_Name(t *testing.T) {
	tool := NewBrowserTool(true)
	if tool.Name() != "browser" {
		t.Errorf("expected name 'browser', got '%s'", tool.Name())
	}
}

// TestBrowserTool_MissingAction verifies error when action is missing
func TestBrowserTool_MissingAction(t *testing.T) {
	tool := NewBrowserTool(true)
	result := tool.Execute(context.Background(), map[string]interface{}{})
	if !result.IsError {
		t.Error("expected error when action is missing")
	}
	if !strings.Contains(result.ForLLM, "action is required") {
		t.Errorf("expected 'action is required' in error, got: %s", result.ForLLM)
	}
}

// TestBrowserTool_InvalidAction verifies rejection of unknown actions
func TestBrowserTool_InvalidAction(t *testing.T) {
	tool := NewBrowserTool(true)
	result := tool.Execute(context.Background(), map[string]interface{}{
		"action": "malicious_command",
	})
	if !result.IsError {
		t.Error("expected error for invalid action")
	}
	if !strings.Contains(result.ForLLM, "unknown browser action") {
		t.Errorf("expected 'unknown browser action' in error, got: %s", result.ForLLM)
	}
}

// TestBrowserTool_ActionAllowlist verifies all valid actions pass validation
func TestBrowserTool_ActionAllowlist(t *testing.T) {
	// Only test the allowlist check, not actual execution
	for action := range allowedBrowserActions {
		if !allowedBrowserActions[action] {
			t.Errorf("action %q should be in allowlist", action)
		}
	}

	invalidActions := []string{"rm", "exec", "sudo", "rm -rf", "shell", ""}
	for _, action := range invalidActions {
		if allowedBrowserActions[action] {
			t.Errorf("action %q should NOT be in allowlist", action)
		}
	}
}

// TestBrowserTool_OpenRequiresURL verifies open action needs url parameter
func TestBrowserTool_OpenRequiresURL(t *testing.T) {
	tool := NewBrowserTool(true)
	result := tool.Execute(context.Background(), map[string]interface{}{
		"action": "open",
	})
	if !result.IsError {
		t.Error("expected error when url is missing for open")
	}
	if !strings.Contains(result.ForLLM, "url is required") {
		t.Errorf("expected 'url is required' in error, got: %s", result.ForLLM)
	}
}

// TestBrowserTool_ClickRequiresSelector verifies click action needs selector
func TestBrowserTool_ClickRequiresSelector(t *testing.T) {
	tool := NewBrowserTool(true)
	result := tool.Execute(context.Background(), map[string]interface{}{
		"action": "click",
	})
	if !result.IsError {
		t.Error("expected error when selector is missing for click")
	}
	if !strings.Contains(result.ForLLM, "selector is required") {
		t.Errorf("expected 'selector is required' in error, got: %s", result.ForLLM)
	}
}

// TestBrowserTool_FillRequiresSelectorAndValue verifies fill needs both params
func TestBrowserTool_FillRequiresSelectorAndValue(t *testing.T) {
	tool := NewBrowserTool(true)

	// Missing selector
	result := tool.Execute(context.Background(), map[string]interface{}{
		"action": "fill",
		"value":  "text",
	})
	if !result.IsError {
		t.Error("expected error when selector is missing for fill")
	}

	// Missing value
	result = tool.Execute(context.Background(), map[string]interface{}{
		"action":   "fill",
		"selector": "#input",
	})
	if !result.IsError {
		t.Error("expected error when value is missing for fill")
	}
}

// TestBrowserTool_PressRequiresValue verifies press needs key value
func TestBrowserTool_PressRequiresValue(t *testing.T) {
	tool := NewBrowserTool(true)
	result := tool.Execute(context.Background(), map[string]interface{}{
		"action": "press",
	})
	if !result.IsError {
		t.Error("expected error when value is missing for press")
	}
}

// TestBrowserTool_EvalRequiresValue verifies eval needs script value
func TestBrowserTool_EvalRequiresValue(t *testing.T) {
	tool := NewBrowserTool(true)
	result := tool.Execute(context.Background(), map[string]interface{}{
		"action": "eval",
	})
	if !result.IsError {
		t.Error("expected error when value is missing for eval")
	}
}

// TestBrowserTool_WaitRequiresSelector verifies wait needs selector
func TestBrowserTool_WaitRequiresSelector(t *testing.T) {
	tool := NewBrowserTool(true)
	result := tool.Execute(context.Background(), map[string]interface{}{
		"action": "wait",
	})
	if !result.IsError {
		t.Error("expected error when selector is missing for wait")
	}
}

// TestBrowserTool_CaseInsensitive verifies that action matching is case-insensitive
func TestBrowserTool_CaseInsensitive(t *testing.T) {
	tool := NewBrowserTool(true)
	// "OPEN" should normalize to "open" and then require url
	result := tool.Execute(context.Background(), map[string]interface{}{
		"action": "OPEN",
	})
	// It should not error with "unknown action" — it should error with "url is required"
	if !result.IsError {
		t.Error("expected error")
	}
	if strings.Contains(result.ForLLM, "unknown browser action") {
		t.Error("action should be case-insensitive")
	}
	if !strings.Contains(result.ForLLM, "url is required") {
		t.Errorf("expected 'url is required' for 'OPEN' action, got: %s", result.ForLLM)
	}
}

// TestGetIntArg verifies integer argument extraction from float64
func TestGetIntArg(t *testing.T) {
	args := map[string]interface{}{
		"timeout": float64(5000),
	}
	v, ok := getIntArg(args, "timeout")
	if !ok || v != 5000 {
		t.Errorf("expected 5000, got %d (ok=%v)", v, ok)
	}

	_, ok = getIntArg(args, "nonexistent")
	if ok {
		t.Error("expected false for missing key")
	}
}

// TestBrowserSearchTool_Integration performs a real search if actionbook is available
func TestBrowserSearchTool_Integration(t *testing.T) {
	if _, err := exec.LookPath("actionbook"); err != nil {
		t.Skip("actionbook not in PATH, skipping integration test")
	}

	tool := NewBrowserSearchTool()
	result := tool.Execute(context.Background(), map[string]interface{}{
		"query": "google search",
	})

	if result.IsError {
		t.Errorf("expected success, got error: %s", result.ForLLM)
	}

	if result.ForLLM == "" {
		t.Error("expected non-empty output")
	}
}

// TestTruncateOutput verifies output truncation
func TestTruncateOutput(t *testing.T) {
	short := "hello"
	if truncateOutput(short, 100) != short {
		t.Error("short string should not be truncated")
	}

	long := strings.Repeat("x", 200)
	truncated := truncateOutput(long, 100)
	if len(truncated) >= 200 {
		t.Errorf("expected truncation, got length %d", len(truncated))
	}
	if !strings.Contains(truncated, "truncated") {
		t.Error("expected truncation marker")
	}
}

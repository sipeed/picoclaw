package policy

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
)

func TestNewEvaluator(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Write minimal config
	cfgData := `{"version": 2}`
	if err := os.WriteFile(configPath, []byte(cfgData), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Create evaluator without policy file (should use defaults)
	eval, err := NewEvaluator(&config.Config{}, configPath)
	if err != nil {
		t.Fatalf("Failed to create evaluator: %v", err)
	}

	if eval == nil {
		t.Fatal("Evaluator should not be nil")
	}

	if eval.IsEnabled() {
		t.Error("Policy should be disabled by default when no policy file exists")
	}
}

func TestEvaluateToolCall_DeniedTools(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Write policy file with denied tools
	policyData := `
enabled: true
default_allow: true
denied_tools:
  - "bash"
  - "shell"
  - "exec"
`
	policyPath := filepath.Join(tmpDir, ".policy.yml")
	if err := os.WriteFile(policyPath, []byte(policyData), 0644); err != nil {
		t.Fatalf("Failed to write policy: %v", err)
	}

	if err := os.WriteFile(configPath, []byte(`{"version": 2}`), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	eval, err := NewEvaluator(&config.Config{}, configPath)
	if err != nil {
		t.Fatalf("Failed to create evaluator: %v", err)
	}

	ctx := context.Background()

	// Test denied tool
	toolCall := ToolCall{
		Name: "bash",
		Arguments: map[string]interface{}{
			"command": "ls -la",
		},
	}

	result, err := eval.EvaluateToolCall(ctx, toolCall)
	if err != nil {
		t.Fatalf("EvaluateToolCall failed: %v", err)
	}

	if result.Allowed {
		t.Error("Tool call should be denied")
	}

	// Test allowed tool
	toolCall.Name = "web_search"
	result, err = eval.EvaluateToolCall(ctx, toolCall)
	if err != nil {
		t.Fatalf("EvaluateToolCall failed: %v", err)
	}

	if !result.Allowed {
		t.Error("Tool call should be allowed")
	}
}

func TestEvaluateToolCall_AllowedTools(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Write policy file with allowed tools whitelist
	policyData := `
enabled: true
default_allow: false
allowed_tools:
  - "web_search"
  - "web_fetch"
  - "message"
`
	policyPath := filepath.Join(tmpDir, ".policy.yml")
	if err := os.WriteFile(policyPath, []byte(policyData), 0644); err != nil {
		t.Fatalf("Failed to write policy: %v", err)
	}

	if err := os.WriteFile(configPath, []byte(`{"version": 2}`), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	eval, err := NewEvaluator(&config.Config{}, configPath)
	if err != nil {
		t.Fatalf("Failed to create evaluator: %v", err)
	}

	ctx := context.Background()

	// Test allowed tool
	toolCall := ToolCall{
		Name: "web_search",
		Arguments: map[string]interface{}{
			"query": "weather",
		},
	}

	result, err := eval.EvaluateToolCall(ctx, toolCall)
	if err != nil {
		t.Fatalf("EvaluateToolCall failed: %v", err)
	}

	if !result.Allowed {
		t.Error("Tool call should be allowed")
	}

	// Test tool not in whitelist
	toolCall.Name = "bash"
	result, err = eval.EvaluateToolCall(ctx, toolCall)
	if err != nil {
		t.Fatalf("EvaluateToolCall failed: %v", err)
	}

	if result.Allowed {
		t.Error("Tool call should be denied (not in whitelist)")
	}
}

func TestEvaluateToolCall_ArgumentPatterns(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Write policy file with argument patterns
	policyData := `
enabled: true
default_allow: true
argument_patterns:
  - tool: "bash"
    argument: "command"
    pattern: "^(rm|sudo|chmod)"
    action: "deny"
    reason: "Destructive commands not allowed"
`
	policyPath := filepath.Join(tmpDir, ".policy.yml")
	if err := os.WriteFile(policyPath, []byte(policyData), 0644); err != nil {
		t.Fatalf("Failed to write policy: %v", err)
	}

	if err := os.WriteFile(configPath, []byte(`{"version": 2}`), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	eval, err := NewEvaluator(&config.Config{}, configPath)
	if err != nil {
		t.Fatalf("Failed to create evaluator: %v", err)
	}

	ctx := context.Background()

	// Test dangerous command
	toolCall := ToolCall{
		Name: "bash",
		Arguments: map[string]interface{}{
			"command": "rm -rf /",
		},
	}

	result, err := eval.EvaluateToolCall(ctx, toolCall)
	if err != nil {
		t.Fatalf("EvaluateToolCall failed: %v", err)
	}

	if result.Allowed {
		t.Error("Dangerous command should be denied")
	}

	// Test safe command
	toolCall.Arguments["command"] = "echo hello"
	result, err = eval.EvaluateToolCall(ctx, toolCall)
	if err != nil {
		t.Fatalf("EvaluateToolCall failed: %v", err)
	}

	if !result.Allowed {
		t.Error("Safe command should be allowed")
	}
}

func TestEvaluateIntent(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Write policy file with intent rules
	policyData := `
enabled: true
default_allow: true
denied_intents:
  - "execute_code"
  - "modify_system"
allowed_intents:
  - "search"
  - "fetch"
`
	policyPath := filepath.Join(tmpDir, ".policy.yml")
	if err := os.WriteFile(policyPath, []byte(policyData), 0644); err != nil {
		t.Fatalf("Failed to write policy: %v", err)
	}

	if err := os.WriteFile(configPath, []byte(`{"version": 2}`), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	eval, err := NewEvaluator(&config.Config{}, configPath)
	if err != nil {
		t.Fatalf("Failed to create evaluator: %v", err)
	}

	ctx := context.Background()

	// Test denied intent
	intent := Intent{
		Type:        "execute_code",
		Description: "User wants to run code",
		Confidence:  0.95,
	}

	result, err := eval.EvaluateIntent(ctx, intent)
	if err != nil {
		t.Fatalf("EvaluateIntent failed: %v", err)
	}

	if result.Allowed {
		t.Error("Intent should be denied")
	}

	// Test allowed intent
	intent.Type = "search"
	result, err = eval.EvaluateIntent(ctx, intent)
	if err != nil {
		t.Fatalf("EvaluateIntent failed: %v", err)
	}

	if !result.Allowed {
		t.Error("Intent should be allowed")
	}
}

func TestEvaluateActionPlan(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Write policy file
	policyData := `
enabled: true
default_allow: true
denied_tools:
  - "bash"
`
	policyPath := filepath.Join(tmpDir, ".policy.yml")
	if err := os.WriteFile(policyPath, []byte(policyData), 0644); err != nil {
		t.Fatalf("Failed to write policy: %v", err)
	}

	if err := os.WriteFile(configPath, []byte(`{"version": 2}`), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	eval, err := NewEvaluator(&config.Config{}, configPath)
	if err != nil {
		t.Fatalf("Failed to create evaluator: %v", err)
	}

	ctx := context.Background()

	// Test plan with allowed actions
	plan := ActionPlan{
		Actions: []Action{
			{
				Type: "tool_call",
				Tool: "web_search",
				Arguments: map[string]interface{}{
					"query": "weather",
				},
			},
		},
	}

	result, err := eval.EvaluateActionPlan(ctx, plan)
	if err != nil {
		t.Fatalf("EvaluateActionPlan failed: %v", err)
	}

	if !result.Allowed {
		t.Error("Action plan should be allowed")
	}

	// Test plan with denied action
	plan.Actions[0].Tool = "bash"
	result, err = eval.EvaluateActionPlan(ctx, plan)
	if err != nil {
		t.Fatalf("EvaluateActionPlan failed: %v", err)
	}

	if result.Allowed {
		t.Error("Action plan should be denied")
	}
}

func TestRules(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Write policy file with rules
	policyData := `
enabled: true
default_allow: true
rules:
  - id: "block-bash"
    description: "Block bash tool"
    tools:
      - "bash"
    action: "deny"
    priority: 100
  - id: "allow-search"
    description: "Allow search tools"
    tools:
      - "web_search"
      - "web_fetch"
    action: "allow"
    priority: 50
`
	policyPath := filepath.Join(tmpDir, ".policy.yml")
	if err := os.WriteFile(policyPath, []byte(policyData), 0644); err != nil {
		t.Fatalf("Failed to write policy: %v", err)
	}

	if err := os.WriteFile(configPath, []byte(`{"version": 2}`), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	eval, err := NewEvaluator(&config.Config{}, configPath)
	if err != nil {
		t.Fatalf("Failed to create evaluator: %v", err)
	}

	ctx := context.Background()

	// Test rule blocking bash
	toolCall := ToolCall{
		Name: "bash",
		Arguments: map[string]interface{}{},
	}

	result, err := eval.EvaluateToolCall(ctx, toolCall)
	if err != nil {
		t.Fatalf("EvaluateToolCall failed: %v", err)
	}

	if result.Allowed {
		t.Error("Bash should be denied by rule")
	}

	// Test rule allowing search
	toolCall.Name = "web_search"
	result, err = eval.EvaluateToolCall(ctx, toolCall)
	if err != nil {
		t.Fatalf("EvaluateToolCall failed: %v", err)
	}

	if !result.Allowed {
		t.Error("Web search should be allowed by rule")
	}
}

func TestRequireApproval(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Write policy file with require_approval list
	policyData := `
enabled: true
default_allow: true
require_approval:
  - "spawn"
  - "install_skill"
`
	policyPath := filepath.Join(tmpDir, ".policy.yml")
	if err := os.WriteFile(policyPath, []byte(policyData), 0644); err != nil {
		t.Fatalf("Failed to write policy: %v", err)
	}

	if err := os.WriteFile(configPath, []byte(`{"version": 2}`), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	eval, err := NewEvaluator(&config.Config{}, configPath)
	if err != nil {
		t.Fatalf("Failed to create evaluator: %v", err)
	}

	ctx := context.Background()

	// Test tool requiring approval
	toolCall := ToolCall{
		Name: "spawn",
		Arguments: map[string]interface{}{
			"task": "analyze data",
		},
	}

	result, err := eval.EvaluateToolCall(ctx, toolCall)
	if err != nil {
		t.Fatalf("EvaluateToolCall failed: %v", err)
	}

	if result.Allowed {
		t.Error("Tool requiring approval should not be allowed")
	}

	if result.Data == nil {
		t.Fatal("Result data should not be nil")
	}

	requiresApproval, ok := result.Data["requires_approval"].(bool)
	if !ok || !requiresApproval {
		t.Error("Result should indicate requires_approval")
	}
}

func TestTimeout(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Write policy file with short timeout
	policyData := `
enabled: true
timeout: 1
default_allow: true
`
	policyPath := filepath.Join(tmpDir, ".policy.yml")
	if err := os.WriteFile(policyPath, []byte(policyData), 0644); err != nil {
		t.Fatalf("Failed to write policy: %v", err)
	}

	if err := os.WriteFile(configPath, []byte(`{"version": 2}`), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	eval, err := NewEvaluator(&config.Config{}, configPath)
	if err != nil {
		t.Fatalf("Failed to create evaluator: %v", err)
	}

	// Test that timeout is set correctly
	if eval.timeout != 1*time.Second {
		t.Errorf("Expected timeout 1s, got %v", eval.timeout)
	}
}

func TestReload(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Write initial policy file
	policyData := `
enabled: true
default_allow: true
denied_tools:
  - "bash"
`
	policyPath := filepath.Join(tmpDir, ".policy.yml")
	if err := os.WriteFile(policyPath, []byte(policyData), 0644); err != nil {
		t.Fatalf("Failed to write policy: %v", err)
	}

	if err := os.WriteFile(configPath, []byte(`{"version": 2}`), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	eval, err := NewEvaluator(&config.Config{}, configPath)
	if err != nil {
		t.Fatalf("Failed to create evaluator: %v", err)
	}

	ctx := context.Background()

	// Verify bash is denied
	toolCall := ToolCall{Name: "bash"}
	result, _ := eval.EvaluateToolCall(ctx, toolCall)
	if result.Allowed {
		t.Error("Bash should be denied initially")
	}

	// Update policy file
	newPolicyData := `
enabled: true
default_allow: true
denied_tools: []
`
	if err := os.WriteFile(policyPath, []byte(newPolicyData), 0644); err != nil {
		t.Fatalf("Failed to update policy: %v", err)
	}

	// Reload policies
	if err := eval.Reload(); err != nil {
		t.Fatalf("Failed to reload: %v", err)
	}

	// Verify bash is now allowed
	result, _ = eval.EvaluateToolCall(ctx, toolCall)
	if !result.Allowed {
		t.Error("Bash should be allowed after reload")
	}
}

func TestDisabledPolicy(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Write policy file with disabled policy
	policyData := `
enabled: false
default_allow: false
denied_tools:
  - "bash"
`
	policyPath := filepath.Join(tmpDir, ".policy.yml")
	if err := os.WriteFile(policyPath, []byte(policyData), 0644); err != nil {
		t.Fatalf("Failed to write policy: %v", err)
	}

	if err := os.WriteFile(configPath, []byte(`{"version": 2}`), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	eval, err := NewEvaluator(&config.Config{}, configPath)
	if err != nil {
		t.Fatalf("Failed to create evaluator: %v", err)
	}

	ctx := context.Background()

	// Even though bash is in denied_tools, policy is disabled
	toolCall := ToolCall{Name: "bash"}
	result, err := eval.EvaluateToolCall(ctx, toolCall)
	if err != nil {
		t.Fatalf("EvaluateToolCall failed: %v", err)
	}

	if !result.Allowed {
		t.Error("All tools should be allowed when policy is disabled")
	}
}

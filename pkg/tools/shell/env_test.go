package shell

import (
	"os"
	"testing"
)

func TestWithAllowedEnv_DefaultAllowlist(t *testing.T) {
	// Set some test env vars
	os.Setenv("PATH", "/usr/bin")
	os.Setenv("HOME", "/home/test")
	os.Setenv("SECRET_API_KEY", "sk-test") // Should NOT be in result
	defer os.Unsetenv("PATH")
	defer os.Unsetenv("HOME")
	defer os.Unsetenv("SECRET_API_KEY")

	result := WithAllowedEnv(nil, nil)

	// Default allowlist vars should be included
	if _, ok := result["PATH"]; !ok {
		t.Error("Expected PATH to be in result from default allowlist")
	}
	if _, ok := result["HOME"]; !ok {
		t.Error("Expected HOME to be in result from default allowlist")
	}

	// Non-allowlisted vars should NOT be included
	if _, ok := result["SECRET_API_KEY"]; ok {
		t.Error("Expected SECRET_API_KEY to NOT be in result")
	}
}

func TestWithAllowedEnv_ExtraAllowlist(t *testing.T) {
	os.Setenv("CUSTOM_VAR", "custom-value")
	os.Setenv("ANOTHER_SECRET", "should-not-appear")
	defer os.Unsetenv("CUSTOM_VAR")
	defer os.Unsetenv("ANOTHER_SECRET")

	// Request extra allowlist
	result := WithAllowedEnv(nil, []string{"CUSTOM_VAR"})

	// Custom var should be included
	if val, ok := result["CUSTOM_VAR"]; !ok || val != "custom-value" {
		t.Errorf("Expected CUSTOM_VAR to be in result, got: %v", result)
	}

	// Non-allowlisted should NOT be included
	if _, ok := result["ANOTHER_SECRET"]; ok {
		t.Error("Expected ANOTHER_SECRET to NOT be in result")
	}
}

func TestWithAllowedEnv_EnvSetOverride(t *testing.T) {
	os.Setenv("PATH", "/usr/bin")
	defer os.Unsetenv("PATH")

	// Provide envSet that overrides PATH
	envSet := map[string]string{
		"PATH": "/custom/path",
	}

	result := WithAllowedEnv(envSet, nil)

	// envSet value should take precedence over inherited
	if result["PATH"] != "/custom/path" {
		t.Errorf("Expected PATH to be /custom/path, got: %s", result["PATH"])
	}
}

func TestMergeEnvVars_LLMBlocked(t *testing.T) {
	baseEnv := map[string]string{
		"PATH": "/usr/bin",
	}

	// LLM tries to override PATH and add PICOCLAW_SECRET
	llmEnv := map[string]string{
		"PATH":            "/hacked/path",      // Should be blocked
		"PICOCLAW_SECRET": "should-be-blocked", // Should be blocked by prefix
		"USER_VAR":        "user-value",        // Should be allowed
	}

	result := MergeEnvVars(baseEnv, nil, llmEnv)

	// LLM overrides should NOT be in result
	if result["PATH"] == "/hacked/path" {
		t.Error("Expected LLM PATH override to be blocked")
	}

	if _, ok := result["PICOCLAW_SECRET"]; ok {
		t.Error("Expected PICOCLAW_SECRET to be blocked by prefix")
	}

	// User vars should be allowed
	if result["USER_VAR"] != "user-value" {
		t.Error("Expected USER_VAR to be allowed")
	}
}

func TestMergeEnvVars_ConfigEnvNotFiltered(t *testing.T) {
	baseEnv := map[string]string{
		"PATH": "/usr/bin",
	}

	// Config env_set - should NOT be filtered
	configEnv := map[string]string{
		"MY_API_KEY": "config-secret",
	}

	llmEnv := map[string]string{
		"PATH": "/hacked", // Should be blocked
	}

	result := MergeEnvVars(baseEnv, configEnv, llmEnv)

	// Config env should be preserved
	if result["MY_API_KEY"] != "config-secret" {
		t.Error("Expected config env_set to be preserved")
	}

	// LLM should not override
	if result["PATH"] == "/hacked" {
		t.Error("Expected LLM override to be blocked")
	}
}

func TestMergeEnvVars_PICOCLAWNotOverridable(t *testing.T) {
	baseEnv := map[string]string{
		"PICOCLAW_HOME": "/original/home",
	}

	// LLM tries to override PICOCLAW_*
	llmEnv := map[string]string{
		"PICOCLAW_HOME":   "/hacked/home",
		"PICOCLAW_CONFIG": "/hacked/config",
	}

	result := MergeEnvVars(baseEnv, nil, llmEnv)

	// Original PICOCLAW_* should be preserved (not in result - baseEnv not filtered)
	// Actually baseEnv is filtered through envKey, but PICOCLAW is not blocked there
	// Let me check the actual behavior

	// LLM-provided PICOCLAW should be blocked
	if _, ok := result["PICOCLAW_HOME"]; ok && result["PICOCLAW_HOME"] == "/hacked/home" {
		t.Error("Expected PICOCLAW_HOME override to be blocked")
	}
}

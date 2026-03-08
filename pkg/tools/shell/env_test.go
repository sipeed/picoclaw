package shell

import (
	"strings"
	"testing"
)

func TestBuildSanitizedEnv_FiltersSecrets(t *testing.T) {
	// Set some secret env vars.
	t.Setenv("OPENAI_API_KEY", "sk-secret")
	t.Setenv("ANTHROPIC_API_KEY", "anthro-secret")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "aws-secret")
	t.Setenv("DATABASE_URL", "postgres://secret")
	// Set something that should pass.
	t.Setenv("PATH", "/usr/bin")
	t.Setenv("HOME", "/home/test")
	t.Setenv("LC_ALL", "en_US.UTF-8")

	env := BuildSanitizedEnv(nil, nil, nil, nil)

	assertEnvPresent(t, env, "PATH")
	assertEnvPresent(t, env, "HOME")
	assertEnvPresent(t, env, "LC_ALL")

	assertEnvAbsent(t, env, "OPENAI_API_KEY")
	assertEnvAbsent(t, env, "ANTHROPIC_API_KEY")
	assertEnvAbsent(t, env, "AWS_SECRET_ACCESS_KEY")
	assertEnvAbsent(t, env, "DATABASE_URL")
}

func TestBuildSanitizedEnv_ExtraAllowlist(t *testing.T) {
	t.Setenv("MY_CUSTOM_VAR", "hello")

	env := BuildSanitizedEnv(nil, []string{"MY_CUSTOM_VAR"}, nil, nil)
	assertEnvPresent(t, env, "MY_CUSTOM_VAR")
}

func TestBuildSanitizedEnv_EnvSet(t *testing.T) {
	env := BuildSanitizedEnv(nil, nil, map[string]string{
		"INJECTED": "value123",
	}, nil)

	v := getEnvValue(env, "INJECTED")
	if v == "" {
		t.Fatal("expected INJECTED to be present")
	}
	if v != "value123" {
		t.Errorf("INJECTED = %q, want %q", v, "value123")
	}
}

func TestBuildSanitizedEnv_EnvSetOverridesInherited(t *testing.T) {
	t.Setenv("PATH", "/original")

	env := BuildSanitizedEnv(nil, nil, map[string]string{
		"PATH": "/overridden",
	}, nil)

	v := getEnvValue(env, "PATH")
	if v != "/overridden" {
		t.Errorf("PATH = %q, want %q", v, "/overridden")
	}
}

func TestBuildSanitizedEnv_DefaultAllowlist(t *testing.T) {
	for name := range DefaultEnvAllowlist {
		t.Setenv(name, "test-"+name)
	}

	env := BuildSanitizedEnv(nil, nil, nil, nil)

	for name := range DefaultEnvAllowlist {
		v := getEnvValue(env, name)
		if v == "" {
			t.Errorf("expected %s to be in sanitized env", name)
		}
	}
}

func TestBuildSanitizedEnv_ReturnsSlice(t *testing.T) {
	env := BuildSanitizedEnv(nil, nil, map[string]string{
		"TEST_A": "a",
		"TEST_B": "b",
	}, nil)

	if env == nil {
		t.Fatal("env should not be nil")
	}

	// Should be a slice
	if len(env) == 0 {
		t.Error("expected non-empty env slice")
	}

	// Check format
	found := make(map[string]string)
	for _, entry := range env {
		k, v, ok := strings.Cut(entry, "=")
		if !ok {
			t.Errorf("invalid env entry: %q", entry)
			continue
		}
		found[k] = v
	}

	if found["TEST_A"] != "a" {
		t.Errorf("TEST_A = %q, want %q", found["TEST_A"], "a")
	}
	if found["TEST_B"] != "b" {
		t.Errorf("TEST_B = %q, want %q", found["TEST_B"], "b")
	}
}

func TestLLMBlocklist_BlocksSensitiveVars(t *testing.T) {
	// Set up inherited env
	t.Setenv("PATH", "/original/path")
	t.Setenv("HOME", "/original/home")

	// LLM tries to override these via extraEnv
	extraEnv := map[string]string{
		"PATH":            "/malicious/path",
		"HOME":            "/etc",
		"LD_PRELOAD":      "/evil.so",
		"MY_CUSTOM_VAR":   "allowed", // Not blocked
	}

	env := BuildSanitizedEnv(nil, nil, nil, extraEnv)

	// Blocked vars should keep their inherited value, not the LLM override
	v := getEnvValue(env, "PATH")
	if v != "/original/path" {
		t.Errorf("PATH should be /original/path, got %q - LLM override was not blocked", v)
	}

	v = getEnvValue(env, "HOME")
	if v != "/original/home" {
		t.Errorf("HOME should be /original/home, got %q - LLM override was not blocked", v)
	}

	// LD_PRELOAD was not in inherited, so should still be absent (blocked)
	v = getEnvValue(env, "LD_PRELOAD")
	if v != "" {
		t.Error("LD_PRELOAD should be blocked by LLMBlocklist")
	}

	// Non-blocked var should be present
	v = getEnvValue(env, "MY_CUSTOM_VAR")
	if v == "" {
		t.Error("MY_CUSTOM_VAR should be allowed")
	}
	if v != "allowed" {
		t.Errorf("MY_CUSTOM_VAR = %q, want %q", v, "allowed")
	}
}

// Helper to get value from env slice
func getEnvValue(env []string, name string) string {
	prefix := name + "="
	for _, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			return strings.TrimPrefix(entry, prefix)
		}
	}
	return ""
}

func assertEnvPresent(t *testing.T, env []string, name string) {
	t.Helper()
	if getEnvValue(env, name) == "" {
		t.Errorf("expected %s to be present in sanitized env", name)
	}
}

func assertEnvAbsent(t *testing.T, env []string, name string) {
	t.Helper()
	v := getEnvValue(env, name)
	if v != "" {
		t.Errorf("expected %s to be absent from sanitized env, got %q", name, v)
	}
}

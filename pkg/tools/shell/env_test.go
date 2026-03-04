package shell

import (
	"testing"

	"mvdan.cc/sh/v3/expand"
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

	env := BuildSanitizedEnv(nil, nil)

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

	env := BuildSanitizedEnv([]string{"MY_CUSTOM_VAR"}, nil)
	assertEnvPresent(t, env, "MY_CUSTOM_VAR")
}

func TestBuildSanitizedEnv_EnvSet(t *testing.T) {
	env := BuildSanitizedEnv(nil, map[string]string{
		"INJECTED": "value123",
	})

	v := env.Get("INJECTED")
	if !v.IsSet() {
		t.Fatal("expected INJECTED to be present")
	}
	if v.Str != "value123" {
		t.Errorf("INJECTED = %q, want %q", v.Str, "value123")
	}
}

func TestBuildSanitizedEnv_EnvSetOverridesInherited(t *testing.T) {
	t.Setenv("PATH", "/original")

	env := BuildSanitizedEnv(nil, map[string]string{
		"PATH": "/overridden",
	})

	v := env.Get("PATH")
	if v.Str != "/overridden" {
		t.Errorf("PATH = %q, want %q", v.Str, "/overridden")
	}
}

func TestBuildSanitizedEnv_DefaultAllowlist(t *testing.T) {
	for name := range DefaultEnvAllowlist {
		t.Setenv(name, "test-"+name)
	}

	env := BuildSanitizedEnv(nil, nil)

	for name := range DefaultEnvAllowlist {
		v := env.Get(name)
		if !v.IsSet() {
			t.Errorf("expected %s to be in sanitized env", name)
		}
	}
}

func TestBuildSanitizedEnv_Each(t *testing.T) {
	env := BuildSanitizedEnv(nil, map[string]string{
		"TEST_A": "a",
		"TEST_B": "b",
	})

	found := make(map[string]bool)
	env.Each(func(name string, vr expand.Variable) bool {
		found[name] = true
		return true
	})

	if !found["TEST_A"] || !found["TEST_B"] {
		t.Errorf("Each did not iterate over all vars: %v", found)
	}
}

func assertEnvPresent(t *testing.T, env expand.Environ, name string) {
	t.Helper()
	v := env.Get(name)
	if !v.IsSet() {
		t.Errorf("expected %s to be present in sanitized env", name)
	}
}

func assertEnvAbsent(t *testing.T, env expand.Environ, name string) {
	t.Helper()
	v := env.Get(name)
	if v.IsSet() {
		t.Errorf("expected %s to be absent from sanitized env, got %q", name, v.Str)
	}
}

package sandbox

import (
	"strings"
	"testing"
)

func TestValidateBindMounts_BlocksDangerousPath(t *testing.T) {
	err := validateBindMounts([]string{"/etc/passwd:/mnt/passwd:ro"})
	if err == nil {
		t.Fatal("expected blocked bind path error")
	}
	if !strings.Contains(err.Error(), "blocked path") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateBindMounts_BlocksNonAbsoluteSource(t *testing.T) {
	err := validateBindMounts([]string{"myvol:/mnt"})
	if err == nil {
		t.Fatal("expected non-absolute bind error")
	}
	if !strings.Contains(err.Error(), "non-absolute") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateBindMounts_AllowsProjectPath(t *testing.T) {
	if err := validateBindMounts([]string{"/home/user/project:/workspace:rw"}); err != nil {
		t.Fatalf("expected bind to pass, got %v", err)
	}
}

func TestValidateNetworkMode_BlocksHost(t *testing.T) {
	if err := validateNetworkMode("HOST"); err == nil {
		t.Fatal("expected host network mode to be blocked")
	}
}

func TestValidateProfiles_BlockUnconfined(t *testing.T) {
	if err := validateSeccompProfile("Unconfined"); err == nil {
		t.Fatal("expected seccomp unconfined to be blocked")
	}
	if err := validateApparmorProfile("unconfined"); err == nil {
		t.Fatal("expected apparmor unconfined to be blocked")
	}
}

func TestSanitizeEnvVars_BlocksSensitiveKeys(t *testing.T) {
	in := map[string]string{
		"LANG":           "C.UTF-8",
		"OPENAI_API_KEY": "secret",
		"GITHUB_TOKEN":   "secret2",
		"SAFE_NAME":      "ok",
		"NULLY":          "a\x00b",
	}
	got := sanitizeEnvVars(in)
	if got["LANG"] != "C.UTF-8" {
		t.Fatalf("LANG should be kept, got %q", got["LANG"])
	}
	if got["SAFE_NAME"] != "ok" {
		t.Fatalf("SAFE_NAME should be kept, got %q", got["SAFE_NAME"])
	}
	if _, ok := got["OPENAI_API_KEY"]; ok {
		t.Fatal("OPENAI_API_KEY should be blocked")
	}
	if _, ok := got["GITHUB_TOKEN"]; ok {
		t.Fatal("GITHUB_TOKEN should be blocked")
	}
	if _, ok := got["NULLY"]; ok {
		t.Fatal("NULLY should be blocked due to null byte")
	}
}

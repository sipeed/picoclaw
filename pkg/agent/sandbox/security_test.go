package sandbox

import (
	"os"
	"path/filepath"
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

func TestValidateSandboxSecurity_AllowsSafeConfig(t *testing.T) {
	cfg := ContainerSandboxConfig{
		Binds:           []string{"/tmp:/workspace:rw"},
		Network:         "none",
		SeccompProfile:  "default",
		ApparmorProfile: "docker-default",
	}
	if err := validateSandboxSecurity(cfg); err != nil {
		t.Fatalf("validateSandboxSecurity() error: %v", err)
	}
}

func TestValidateSandboxSecurity_ReturnsFirstPolicyError(t *testing.T) {
	if err := validateSandboxSecurity(ContainerSandboxConfig{
		Network: "host",
	}); err == nil {
		t.Fatal("expected network policy error")
	}

	if err := validateSandboxSecurity(ContainerSandboxConfig{
		SeccompProfile: "unconfined",
	}); err == nil {
		t.Fatal("expected seccomp policy error")
	}

	if err := validateSandboxSecurity(ContainerSandboxConfig{
		ApparmorProfile: "unconfined",
	}); err == nil {
		t.Fatal("expected apparmor policy error")
	}
}

func TestParseAndNormalizeHelpers(t *testing.T) {
	if got := parseBindSourcePath("/a:/b:ro"); got != "/a" {
		t.Fatalf("parseBindSourcePath() got %q, want /a", got)
	}
	if got := parseBindSourcePath("just-source"); got != "just-source" {
		t.Fatalf("parseBindSourcePath() got %q", got)
	}

	if got := normalizeHostPath(" "); got != "/" {
		t.Fatalf("normalizeHostPath(empty) got %q, want /", got)
	}
	if got := normalizeHostPath("/tmp///a/"); got != "/tmp/a" {
		t.Fatalf("normalizeHostPath() got %q, want /tmp/a", got)
	}
}

func TestTryRealpathAbsolute_Branches(t *testing.T) {
	if got := tryRealpathAbsolute("relative/path"); got != "relative/path" {
		t.Fatalf("tryRealpathAbsolute(relative) got %q", got)
	}

	root := t.TempDir()
	target := filepath.Join(root, "target")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatalf("mkdir target: %v", err)
	}
	link := filepath.Join(root, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatalf("symlink create: %v", err)
	}

	old := filepathEvalSymlinks
	t.Cleanup(func() { filepathEvalSymlinks = old })

	filepathEvalSymlinks = old
	if got := tryRealpathAbsolute(link); got == link {
		t.Fatalf("tryRealpathAbsolute(existing symlink) should resolve, got %q", got)
	}

	filepathEvalSymlinks = func(path string) (string, error) { return "", os.ErrPermission }
	if got := tryRealpathAbsolute(link); got != normalizeHostPath(link) {
		t.Fatalf("tryRealpathAbsolute(eval error) got %q", got)
	}

	nonexistent := filepath.Join(root, "does-not-exist")
	if got := tryRealpathAbsolute(nonexistent); got != nonexistent {
		t.Fatalf("tryRealpathAbsolute(nonexistent) got %q", got)
	}
}

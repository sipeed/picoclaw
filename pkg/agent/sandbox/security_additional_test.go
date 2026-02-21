package sandbox

import (
	"os"
	"path/filepath"
	"testing"
)

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

package shellguard

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

func TestValidator_BlocksDangerousPattern(t *testing.T) {
	validator := New(Config{
		DenyPatterns: []*regexp.Regexp{regexp.MustCompile(`\brm\s+-[rf]{1,2}\b`)},
	})

	decision := validator.Validate("rm -rf /", "")
	if decision.Allowed {
		t.Fatal("expected dangerous command to be blocked")
	}
	if decision.Category != "dangerous_pattern" {
		t.Fatalf("category = %q, want dangerous_pattern", decision.Category)
	}
}

func TestValidator_CustomAllowBypassesDenyPattern(t *testing.T) {
	validator := New(Config{
		DenyPatterns:        []*regexp.Regexp{regexp.MustCompile(`\bgit\s+push\b`)},
		CustomAllowPatterns: []*regexp.Regexp{regexp.MustCompile(`\bgit\s+push\s+origin\b`)},
	})

	decision := validator.Validate("git push origin feature", "")
	if !decision.Allowed {
		t.Fatalf("expected custom allow to bypass deny pattern: %s", decision.Reason)
	}
}

func TestValidator_AllowPatternsRequireMatch(t *testing.T) {
	validator := New(Config{
		AllowPatterns: []*regexp.Regexp{regexp.MustCompile(`^git status$`)},
	})

	decision := validator.Validate("git diff", "")
	if decision.Allowed {
		t.Fatal("expected command outside allowlist to be blocked")
	}
	if decision.Category != "not_allowlisted" {
		t.Fatalf("category = %q, want not_allowlisted", decision.Category)
	}
}

func TestValidator_StripsQuotedHeredocBodyBeforeDenyChecks(t *testing.T) {
	validator := New(Config{
		DenyPatterns: []*regexp.Regexp{regexp.MustCompile("`[^`]+`")},
	})

	command := "gh pr comment 2763 --body-file - <<'TXT'\n" +
		"Fixed `pkg/tools/integration/web_test.go`.\n" +
		"TXT"
	decision := validator.Validate(command, "")
	if !decision.Allowed {
		t.Fatalf("expected quoted heredoc body to be allowed: %s", decision.Reason)
	}
}

func TestValidator_BlocksWorkspacePathEscape(t *testing.T) {
	root := t.TempDir()
	validator := New(Config{RestrictToWorkspace: true})

	decision := validator.Validate("cat /etc/passwd", root)
	if decision.Allowed {
		t.Fatal("expected absolute path outside workspace to be blocked")
	}
	if decision.Category != "path_outside_working_dir" {
		t.Fatalf("category = %q, want path_outside_working_dir", decision.Category)
	}
}

func TestValidator_BlocksSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("secret"), 0o600); err != nil {
		t.Fatalf("write outside file: %v", err)
	}
	link := filepath.Join(root, "outside-link")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	validator := New(Config{RestrictToWorkspace: true})
	decision := validator.Validate("cat "+filepath.Join(link, "secret.txt"), root)
	if decision.Allowed {
		t.Fatal("expected symlink escape to be blocked")
	}
}

func TestValidator_AllowsConfiguredExternalPath(t *testing.T) {
	root := t.TempDir()
	allowed := t.TempDir()
	validator := New(Config{
		RestrictToWorkspace: true,
		AllowedPathPatterns: []*regexp.Regexp{
			regexp.MustCompile("^" + regexp.QuoteMeta(allowed)),
		},
	})

	decision := validator.Validate("cat "+filepath.Join(allowed, "file.txt"), root)
	if !decision.Allowed {
		t.Fatalf("expected allow-path match to be allowed: %s", decision.Reason)
	}
}

func TestValidator_AllowsSafePseudoDevices(t *testing.T) {
	root := t.TempDir()
	validator := New(Config{RestrictToWorkspace: true})

	for _, command := range []string{
		"cat /dev/urandom",
		"echo test > /dev/null",
	} {
		decision := validator.Validate(command, root)
		if !decision.Allowed {
			t.Fatalf("expected %q to be allowed: %s", command, decision.Reason)
		}
	}
}

func TestValidator_DoesNotTreatURLPathAsWorkspaceEscape(t *testing.T) {
	root := t.TempDir()
	validator := New(Config{RestrictToWorkspace: true})

	decision := validator.Validate("git clone https://github.com/sipeed/picoclaw", root)
	if !decision.Allowed {
		t.Fatalf("expected URL path component to be ignored: %s", decision.Reason)
	}
}

func TestClassifyCommand(t *testing.T) {
	tests := []struct {
		name    string
		command string
		want    string
	}{
		{name: "read only cat", command: "cat README.md", want: CommandClassReadOnly},
		{name: "read only full path", command: "/usr/bin/git status", want: CommandClassReadOnly},
		{name: "env assignment", command: "FOO=bar rg needle .", want: CommandClassReadOnly},
		{name: "write redirection", command: "echo hi > file.txt", want: CommandClassWrite},
		{name: "sed in place", command: "sed -i s/a/b/ file.txt", want: CommandClassWrite},
		{name: "git push destructive", command: "git push origin main", want: CommandClassDestructive},
		{name: "git fetch write", command: "git fetch origin main", want: CommandClassWrite},
		{name: "gh pr view read only", command: "gh pr view 17", want: CommandClassReadOnly},
		{name: "gh pr comment write", command: "gh pr comment 17 --body hi", want: CommandClassWrite},
		{name: "rm destructive", command: "rm -rf tmp", want: CommandClassDestructive},
		{name: "unknown", command: "custom-tool --flag", want: CommandClassUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ClassifyCommand(tt.command); got != tt.want {
				t.Fatalf("ClassifyCommand(%q) = %q, want %q", tt.command, got, tt.want)
			}
		})
	}
}

func TestValidatorDecisionIncludesCommandClass(t *testing.T) {
	validator := New(Config{
		DenyPatterns: []*regexp.Regexp{regexp.MustCompile(`\brm\s+-[rf]{1,2}\b`)},
	})

	decision := validator.Validate("rm -rf tmp", "")
	if decision.CommandClass != CommandClassDestructive {
		t.Fatalf("CommandClass = %q, want %q", decision.CommandClass, CommandClassDestructive)
	}
}

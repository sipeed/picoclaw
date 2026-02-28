package sandbox

import (
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
)

func TestIsToolSandboxEnabled_Default(t *testing.T) {
	if !IsToolSandboxEnabled(nil, "exec") {
		t.Fatal("expected exec to be sandbox-enabled by default")
	}
	if IsToolSandboxEnabled(nil, "list_dir") {
		t.Fatal("expected list_dir to be host by default")
	}
}

func TestIsToolSandboxEnabled_AllowDeny(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Tools.Sandbox.Tools.Allow = []string{"exec", "write_file"}
	cfg.Tools.Sandbox.Tools.Deny = []string{"write_file"}

	if !IsToolSandboxEnabled(cfg, "exec") {
		t.Fatal("expected exec to be enabled")
	}
	if IsToolSandboxEnabled(cfg, "read_file") {
		t.Fatal("expected read_file to be disabled by allow list")
	}
	if IsToolSandboxEnabled(cfg, "write_file") {
		t.Fatal("expected deny to override allow")
	}
}

// TestIsToolSandboxEnabled_EmptyAllowDeniesAll verifies BOUNDARY-1 fix:
// an explicitly empty allow list now means "deny all tools" (principle of least privilege).
func TestIsToolSandboxEnabled_EmptyAllowDeniesAll(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Tools.Sandbox.Tools.Allow = []string{}
	cfg.Tools.Sandbox.Tools.Deny = []string{"cron"}

	// Empty explicit allow should now deny everything (including read_file which was previously allowed)
	if IsToolSandboxEnabled(cfg, "read_file") {
		t.Fatal("expected read_file to be DISABLED when allow list is explicitly empty (deny all)")
	}
	if IsToolSandboxEnabled(cfg, "exec") {
		t.Fatal("expected exec to be DISABLED when allow list is explicitly empty (deny all)")
	}
	// Deny list still applies (as a belt-and-suspenders check)
	if IsToolSandboxEnabled(cfg, "cron") {
		t.Fatal("expected denied tool to be disabled")
	}
}

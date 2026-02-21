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

func TestIsToolSandboxEnabled_EmptyAllowMeansAll(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Tools.Sandbox.Tools.Allow = []string{}
	cfg.Tools.Sandbox.Tools.Deny = []string{"cron"}

	if !IsToolSandboxEnabled(cfg, "read_file") {
		t.Fatal("expected read_file to be enabled when allow is empty")
	}
	if IsToolSandboxEnabled(cfg, "cron") {
		t.Fatal("expected denied tool to be disabled")
	}
}

package providers

import (
	"reflect"
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
)

func testProviderWorkspace(t *testing.T, provider any) string {
	t.Helper()

	v := reflect.ValueOf(provider)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		t.Fatalf("provider = %T, want non-nil pointer", provider)
	}

	field := v.Elem().FieldByName("workspace")
	if !field.IsValid() || field.Kind() != reflect.String {
		t.Fatalf("provider %T does not expose workspace field", provider)
	}

	return field.String()
}

func TestCreateProvider_ClaudeCli(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.ModelList = []*config.ModelConfig{
		{ModelName: "claude-sonnet-4.6", Model: "claude-cli/claude-sonnet-4.6", Workspace: "/test/ws"},
	}
	cfg.Agents.Defaults.ModelName = "claude-sonnet-4.6"

	provider, _, err := CreateProvider(cfg)
	if err != nil {
		t.Fatalf("CreateProvider(claude-cli) error = %v", err)
	}

	cliProvider, ok := provider.(*ClaudeCliProvider)
	if !ok {
		t.Fatalf("CreateProvider(claude-cli) returned %T, want *ClaudeCliProvider", provider)
	}
	if got := testProviderWorkspace(t, cliProvider); got != "/test/ws" {
		t.Errorf("workspace = %q, want %q", got, "/test/ws")
	}
}

func TestCreateProvider_ClaudeCode(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.ModelList = []*config.ModelConfig{
		{ModelName: "claude-code", Model: "claude-cli/claude-code"},
	}
	cfg.Agents.Defaults.ModelName = "claude-code"

	provider, _, err := CreateProvider(cfg)
	if err != nil {
		t.Fatalf("CreateProvider(claude-code) error = %v", err)
	}
	if _, ok := provider.(*ClaudeCliProvider); !ok {
		t.Fatalf("CreateProvider(claude-code) returned %T, want *ClaudeCliProvider", provider)
	}
}

func TestCreateProvider_ClaudeCodec(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.ModelList = []*config.ModelConfig{
		{ModelName: "claudecode", Model: "claude-cli/claudecode"},
	}
	cfg.Agents.Defaults.ModelName = "claudecode"

	provider, _, err := CreateProvider(cfg)
	if err != nil {
		t.Fatalf("CreateProvider(claudecode) error = %v", err)
	}
	if _, ok := provider.(*ClaudeCliProvider); !ok {
		t.Fatalf("CreateProvider(claudecode) returned %T, want *ClaudeCliProvider", provider)
	}
}

func TestCreateProvider_ClaudeCliDefaultWorkspace(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.ModelList = []*config.ModelConfig{
		{ModelName: "claude-cli", Model: "claude-cli/claude-sonnet"},
	}
	cfg.Agents.Defaults.ModelName = "claude-cli"
	cfg.Agents.Defaults.Workspace = ""

	provider, _, err := CreateProvider(cfg)
	if err != nil {
		t.Fatalf("CreateProvider error = %v", err)
	}

	cliProvider, ok := provider.(*ClaudeCliProvider)
	if !ok {
		t.Fatalf("returned %T, want *ClaudeCliProvider", provider)
	}
	if got := testProviderWorkspace(t, cliProvider); got != "." {
		t.Errorf("workspace = %q, want %q (default)", got, ".")
	}
}

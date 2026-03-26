package agent

import (
	"context"
	"os"
	"testing"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/tools"
)

type isolationMockTool struct {
	name string
}

func (m *isolationMockTool) Name() string        { return m.name }
func (m *isolationMockTool) Description() string { return "mock tool" }
func (m *isolationMockTool) Parameters() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{}}
}
func (m *isolationMockTool) Execute(ctx context.Context, args map[string]any) *tools.ToolResult {
	return tools.SilentResult("executed")
}

func TestIsolationLacksManualTools(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "picoclaw-isolation-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{}
	cfg.Agents.Defaults.Workspace = tmpDir
	cfg.Agents.Defaults.ModelName = "test-model"

	msgBus := bus.NewMessageBus()
	provider := &isolationMockProvider{}
	al := NewAgentLoop(cfg, msgBus, provider)

	tool := &isolationMockTool{name: "my_custom_tool"}
	al.RegisterTool(tool)

	// chatID "direct" does NOT use isolation
	resp, err := al.ProcessDirectWithChannel(context.Background(), "hello", "session1", "cli", "direct")
	if err != nil {
		t.Errorf("ProcessDirectWithChannel failed: %v", err)
	}
	if resp != "Found tool" {
		t.Errorf("Direct response: %s, want Found tool", resp)
	}

	// chatID "chat1" DOES use isolation - transient agent instance is created
	resp, err = al.ProcessDirectWithChannel(context.Background(), "hello", "session1", "cli", "chat1")
	if err != nil {
		t.Errorf("ProcessDirectWithChannel (isolated) failed: %v", err)
	}
	if resp != "Found tool" {
		t.Errorf("Isolated response: %s, want Found tool (fixed)", resp)
	}
}

func TestManualToolsPreservedAfterReload(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "picoclaw-reload-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{}
	cfg.Agents.Defaults.Workspace = tmpDir
	cfg.Agents.Defaults.ModelName = "test-model"

	msgBus := bus.NewMessageBus()
	provider := &isolationMockProvider{}
	al := NewAgentLoop(cfg, msgBus, provider)

	tool := &isolationMockTool{name: "my_custom_tool"}
	al.RegisterTool(tool)

	// Reload with same config and provider - should preserve manual tools
	err = al.ReloadProviderAndConfig(context.Background(), provider, cfg)
	if err != nil {
		t.Fatalf("Reload failed: %v", err)
	}

	// Check if tool is still visible to the new registry
	resp, err := al.ProcessDirectWithChannel(context.Background(), "hello", "session1", "cli", "direct")
	if err != nil {
		t.Errorf("ProcessDirectWithChannel failed: %v", err)
	}
	if resp != "Found tool" {
		t.Errorf("Response after reload: %s, want Found tool", resp)
	}
}

type isolationMockProvider struct{}

func (m *isolationMockProvider) Chat(
	ctx context.Context,
	msgs []providers.Message,
	tools []providers.ToolDefinition,
	model string,
	opts map[string]any,
) (*providers.LLMResponse, error) {
	found := false
	for _, t := range tools {
		if t.Function.Name == "my_custom_tool" {
			found = true
			break
		}
	}
	if found {
		return &providers.LLMResponse{Content: "Found tool"}, nil
	}
	return &providers.LLMResponse{Content: "Tool NOT found"}, nil
}

func (m *isolationMockProvider) GetDefaultModel() string {
	return "mock"
}

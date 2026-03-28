package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
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

type tenantIsolationMockProvider struct {
	toolCalls []providers.ToolCall
	response  string
}

func (p *tenantIsolationMockProvider) Chat(
	ctx context.Context, msgs []providers.Message, tools []providers.ToolDefinition,
	model string, opts map[string]any,
) (*providers.LLMResponse, error) {
	if len(p.toolCalls) > 0 {
		res := &providers.LLMResponse{
			ToolCalls: p.toolCalls,
		}
		p.toolCalls = nil // Clear so it doesn't loop
		return res, nil
	}
	return &providers.LLMResponse{Content: p.response}, nil
}

func (p *tenantIsolationMockProvider) GetDefaultModel() string { return "test-model" }

func TestProcessMessage_IsolatedTenant_UsesPrivateWorkspace(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-isolation-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:           tmpDir,
				ModelName:           "test-model",
				MaxTokens:           4096,
				MaxToolIterations:   10,
				RestrictToWorkspace: true,
			},
		},
	}
	cfg.Tools.WriteFile.Enabled = true

	msgBus := bus.NewMessageBus()
	provider := &tenantIsolationMockProvider{
		toolCalls: []providers.ToolCall{
			{
				ID:   "call1",
				Type: "function",
				Name: "write_file",
				Arguments: map[string]any{
					"path":    "secret.txt",
					"content": "isolated-content",
				},
			},
		},
		response: "File written.",
	}
	al := NewAgentLoop(cfg, msgBus, provider)
	defer al.Close()

	isolationID := "tenant-A"
	msg := bus.InboundMessage{
		Channel:  "test-channel",
		SenderID: "user1",
		ChatID:   isolationID,
		Content:  "Write the secret file",
		Peer: bus.Peer{
			Kind: "direct",
			ID:   "user1",
		},
	}

	resp, err := al.processMessage(context.Background(), msg)
	if err != nil {
		t.Fatalf("processMessage failed: %v", err)
	}
	fmt.Printf("Agent Response: %s\n", resp)

	// Verify the file was written to the ISOLATED workspace, NOT the global one
	isolatedPath := filepath.Join(tmpDir, "sessions", isolationID, "workspace", "secret.txt")
	globalPath := filepath.Join(tmpDir, "secret.txt")

	// Debug: Print all files in tmpDir
	t.Logf("Listing all files in %s:", tmpDir)
	filepath.Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			t.Logf("Found file: %s", path)
		}
		return nil
	})

	if _, err := os.Stat(isolatedPath); os.IsNotExist(err) {
		t.Errorf("expected file at %s to exist", isolatedPath)
	}
	if _, err := os.Stat(globalPath); err == nil {
		t.Errorf("expected file at %s to NOT exist (leaked to global workspace)", globalPath)
	}

	// Verify history is in the base sessions directory with the isolated key
	// agent:main:tenant-A becomes agent_main_tenant-A
	isoSessionPath := filepath.Join(tmpDir, "sessions", "agent_main_tenant-A.jsonl")
	if _, err := os.Stat(isoSessionPath); os.IsNotExist(err) {
		t.Errorf("expected history at %s to exist", isoSessionPath)
	} else {
		t.Logf("History exists at: %s", isoSessionPath)
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

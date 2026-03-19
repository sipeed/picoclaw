package agent

import (
	"context"
	"os"
	"testing"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/routing"
)

type modelRecordingProvider struct {
	lastModel string
}

func (p *modelRecordingProvider) Chat(
	ctx context.Context,
	messages []providers.Message,
	tools []providers.ToolDefinition,
	model string,
	opts map[string]any,
) (*providers.LLMResponse, error) {
	p.lastModel = model
	return &providers.LLMResponse{
		Content:   "ok",
		ToolCalls: nil,
	}, nil
}

func (p *modelRecordingProvider) GetDefaultModel() string {
	return "gpt-5.4"
}

func TestProcessMessage_OpenAIRequestDoesNotPersistHistoryAndOverridesModel(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-openai-*")
	if err != nil {
		t.Fatalf("MkdirTemp() error = %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Workspace = tmpDir
	cfg.Agents.Defaults.Model = "gpt-5.4"
	cfg.Agents.Defaults.Provider = "openai"
	cfg.Agents.Defaults.MaxTokens = 4096
	cfg.Agents.Defaults.MaxToolIterations = 4

	messageBus := bus.NewMessageBus()
	provider := &modelRecordingProvider{}
	loop := NewAgentLoop(cfg, messageBus, provider)

	_, err = loop.processMessage(context.Background(), bus.InboundMessage{
		Channel:  "openai_api",
		SenderID: "client-1",
		ChatID:   "chat-1",
		Content:  "hello",
		Metadata: map[string]string{
			"no_history":      "true",
			"requested_model": "deepseek-chat",
		},
	})
	if err != nil {
		t.Fatalf("processMessage() error = %v", err)
	}

	if provider.lastModel != "deepseek-chat" {
		t.Fatalf("provider model = %q, want %q", provider.lastModel, "deepseek-chat")
	}

	agent := loop.GetRegistry().GetDefaultAgent()
	if agent == nil {
		t.Fatal("expected default agent")
	}

	sessionKey := routing.BuildAgentMainSessionKey(agent.ID)
	history := agent.Sessions.GetHistory(sessionKey)
	if len(history) != 0 {
		t.Fatalf("expected no persisted history, got %d entries", len(history))
	}
}

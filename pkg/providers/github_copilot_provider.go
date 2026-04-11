package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	copilot "github.com/github/copilot-sdk/go"
)

type GitHubCopilotProvider struct {
	uri         string
	connectMode string // "stdio" or "grpc"
	model       string
	closed      bool

	client  *copilot.Client
	session *copilot.Session

	mu sync.Mutex
}

func NewGitHubCopilotProvider(uri string, connectMode string, model string) (*GitHubCopilotProvider, error) {
	if connectMode == "" {
		connectMode = "grpc"
	}

	if connectMode != "stdio" && connectMode != "grpc" {
		return nil, fmt.Errorf("unknown connect mode: %s", connectMode)
	}

	provider := &GitHubCopilotProvider{
		uri:         uri,
		connectMode: connectMode,
		model:       model,
	}

	if connectMode == "stdio" {
		return provider, nil
	}

	if _, err := provider.ensureSession(context.Background(), model); err != nil {
		provider.Close()
		return nil, err
	}

	return provider, nil
}

func (p *GitHubCopilotProvider) newClient() (*copilot.Client, error) {
	switch p.connectMode {
	case "stdio":
		return copilot.NewClient(nil), nil
	case "grpc":
		return copilot.NewClient(&copilot.ClientOptions{
			CLIUrl: p.uri,
		}), nil
	default:
		return nil, fmt.Errorf("unknown connect mode: %s", p.connectMode)
	}
}

func (p *GitHubCopilotProvider) ensureSession(ctx context.Context, model string) (*copilot.Session, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil, fmt.Errorf("provider closed")
	}

	if p.session != nil {
		return p.session, nil
	}

	if p.client == nil {
		client, err := p.newClient()
		if err != nil {
			return nil, err
		}
		p.client = client
	}

	if err := p.client.Start(ctx); err != nil {
		p.client.Stop()
		p.client = nil
		return nil, fmt.Errorf(
			"can't connect to Github Copilot: %w; `https://github.com/github/copilot-sdk/blob/main/docs/getting-started.md#connecting-to-an-external-cli-server` for details",
			err,
		)
	}

	sessionModel := p.model
	if model != "" {
		sessionModel = model
	}

	session, err := p.client.CreateSession(ctx, &copilot.SessionConfig{
		Model:               sessionModel,
		OnPermissionRequest: copilot.PermissionHandler.ApproveAll,
		Hooks:               &copilot.SessionHooks{},
	})
	if err != nil {
		p.client.Stop()
		p.client = nil
		return nil, fmt.Errorf("create session failed: %w", err)
	}

	p.session = session
	return session, nil
}

func (p *GitHubCopilotProvider) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.closed = true
	if p.client != nil {
		p.client.Stop()
		p.client = nil
		p.session = nil
	}
}

func (p *GitHubCopilotProvider) Chat(
	ctx context.Context,
	messages []Message,
	tools []ToolDefinition,
	model string,
	options map[string]any,
) (*LLMResponse, error) {
	type tempMessage struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	out := make([]tempMessage, 0, len(messages))
	for _, msg := range messages {
		out = append(out, tempMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	fullcontent, err := json.Marshal(out)
	if err != nil {
		return nil, fmt.Errorf("marshal messages: %w", err)
	}
	session, err := p.ensureSession(ctx, model)
	if err != nil {
		return nil, err
	}

	resp, err := session.SendAndWait(ctx, copilot.MessageOptions{
		Prompt: string(fullcontent),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to send message to copilot: %w", err)
	}

	if resp == nil {
		return nil, fmt.Errorf("empty response from copilot")
	}
	if resp.Data.Content == nil {
		return nil, fmt.Errorf("no content in copilot response")
	}
	content := *resp.Data.Content

	return &LLMResponse{
		FinishReason: "stop",
		Content:      content,
	}, nil
}

func (p *GitHubCopilotProvider) GetDefaultModel() string {
	return "gpt-4.1"
}

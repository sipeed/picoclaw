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

	client  *copilot.Client
	session *copilot.Session

	mu sync.Mutex
}

func NewGitHubCopilotProvider(uri string, connectMode string, model string) (*GitHubCopilotProvider, error) {
	if connectMode == "" {
		connectMode = "grpc"
	}

	switch connectMode {
	case "stdio":
		// TODO: Implement stdio mode for GitHub Copilot provider
		// See https://github.com/github/copilot-sdk/blob/main/docs/getting-started.md for details
		return nil, fmt.Errorf("stdio mode not implemented for GitHub Copilot provider; please use 'grpc' mode instead")
	case "grpc":
		client := copilot.NewClient(&copilot.ClientOptions{
			CLIUrl: uri,
		})
		if err := client.Start(context.Background()); err != nil {
			return nil, fmt.Errorf(
				"can't connect to Github Copilot: %w; `https://github.com/github/copilot-sdk/blob/main/docs/getting-started.md#connecting-to-an-external-cli-server` for details",
				err,
			)
		}

		session, err := client.CreateSession(context.Background(), &copilot.SessionConfig{
			Model: model,
			Hooks: &copilot.SessionHooks{},
		})
		if err != nil {
			client.Stop()
			return nil, fmt.Errorf("create session failed: %w", err)
		}

		return &GitHubCopilotProvider{
			uri:         uri,
			connectMode: connectMode,
			client:      client,
			session:     session,
		}, nil
	default:
		return nil, fmt.Errorf("unknown connect mode: %s", connectMode)
	}
}

func (p *GitHubCopilotProvider) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
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
	p.mu.Lock()
	session := p.session
	p.mu.Unlock()

	if session == nil {
		return nil, fmt.Errorf("provider closed")
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

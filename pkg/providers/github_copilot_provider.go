package providers

import (
	"context"
	"encoding/json"
	"fmt"

	copilot "github.com/github/copilot-sdk/go"
)

type GitHubCopilotProvider struct {
	uri         string
	connectMode string // `stdio` or `grpc``

	session *copilot.Session
}

func NewGitHubCopilotProvider(uri string, connectMode string, model string) (*GitHubCopilotProvider, error) {
	var session *copilot.Session
	if connectMode == "" {
		connectMode = "grpc"
	}
	switch connectMode {

	case "stdio":
		return nil, fmt.Errorf("stdio connect mode is not yet supported for GitHub Copilot provider")
	case "grpc":
		client := copilot.NewClient(&copilot.ClientOptions{
			CLIUrl: uri,
		})
		if err := client.Start(context.Background()); err != nil {
			return nil, fmt.Errorf("can't connect to GitHub Copilot (see https://github.com/github/copilot-sdk/blob/main/docs/getting-started.md#connecting-to-an-external-cli-server): %w", err)
		}
		defer client.Stop()
		var err error
		session, err = client.CreateSession(context.Background(), &copilot.SessionConfig{
			Model: model,
			Hooks: &copilot.SessionHooks{},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create GitHub Copilot session: %w", err)
		}

	default:
		return nil, fmt.Errorf("unsupported connect mode %q for GitHub Copilot provider (supported: grpc)", connectMode)
	}

	return &GitHubCopilotProvider{
		uri:         uri,
		connectMode: connectMode,
		session:     session,
	}, nil
}

// Chat sends a chat request to GitHub Copilot
func (p *GitHubCopilotProvider) Chat(
	ctx context.Context, messages []Message, tools []ToolDefinition, model string, options map[string]any,
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
		return nil, fmt.Errorf("failed to marshal messages: %w", err)
	}

	content, err := p.session.Send(ctx, copilot.MessageOptions{
		Prompt: string(fullcontent),
	})
	if err != nil {
		return nil, fmt.Errorf("GitHub Copilot request failed: %w", err)
	}

	return &LLMResponse{
		FinishReason: "stop",
		Content:      content,
	}, nil
}

func (p *GitHubCopilotProvider) GetDefaultModel() string {
	return "gpt-4.1"
}

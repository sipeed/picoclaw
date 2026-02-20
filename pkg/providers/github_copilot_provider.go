package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	copilot "github.com/github/copilot-sdk/go"
)

type GitHubCopilotProvider struct {
	uri         string
	connectMode string // `stdio` or `grpc`

	client  *copilot.Client
	session *copilot.Session
}

func NewGitHubCopilotProvider(uri string, connectMode string, model string) (*GitHubCopilotProvider, error) {
	var session *copilot.Session
	var client *copilot.Client
	if connectMode == "" {
		connectMode = "grpc"
	}
	switch connectMode {
	case "stdio":
		// TODO: implement stdio connect mode
		return nil, fmt.Errorf("stdio connect mode is not yet implemented")
	case "grpc":
		client = copilot.NewClient(&copilot.ClientOptions{
			CLIUrl: uri,
		})
		connectCtx, connectCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer connectCancel()
		if err := client.Start(connectCtx); err != nil {
			return nil, fmt.Errorf("can't connect to GitHub Copilot: %w", err)
		}
		var err error
		session, err = client.CreateSession(connectCtx, &copilot.SessionConfig{
			Model: model,
			Hooks: &copilot.SessionHooks{},
		})
		if err != nil {
			client.Stop()
			return nil, fmt.Errorf("failed to create Copilot session: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported connect mode %q", connectMode)
	}

	return &GitHubCopilotProvider{
		uri:         uri,
		connectMode: connectMode,
		client:      client,
		session:     session,
	}, nil
}

// Chat sends a chat request to GitHub Copilot
func (p *GitHubCopilotProvider) Chat(ctx context.Context, messages []Message, tools []ToolDefinition, model string, options map[string]interface{}) (*LLMResponse, error) {
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

	if p.session == nil {
		return nil, fmt.Errorf("copilot session is not initialized")
	}

	fullcontent, err := json.Marshal(out)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal messages: %w", err)
	}

	event, err := p.session.SendAndWait(ctx, copilot.MessageOptions{
		Prompt: string(fullcontent),
	})
	if err != nil {
		return nil, fmt.Errorf("copilot error: %w", err)
	}

	if event == nil || event.Data.Content == nil {
		return nil, fmt.Errorf("empty response from Copilot")
	}

	return &LLMResponse{
		FinishReason: "stop",
		Content:      *event.Data.Content,
	}, nil
}

func (p *GitHubCopilotProvider) GetDefaultModel() string {
	return "gpt-4.1"
}

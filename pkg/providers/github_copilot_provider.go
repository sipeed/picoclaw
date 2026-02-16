package providers

import (
	"context"
	"fmt"

	json "encoding/json"

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
		//todo
	case "grpc":
		client := copilot.NewClient(&copilot.ClientOptions{
			CLIUrl: uri,
		})
		if err := client.Start(context.Background()); err != nil {
			return nil, fmt.Errorf("Can't connect to Github Copilot, https://github.com/github/copilot-sdk/blob/main/docs/getting-started.md#connecting-to-an-external-cli-server for details")
		}
		defer client.Stop()
		session, _ = client.CreateSession(context.Background(), &copilot.SessionConfig{
			Model: model,
			Hooks: &copilot.SessionHooks{},
		})

	}

	return &GitHubCopilotProvider{
		uri:         uri,
		connectMode: connectMode,
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

	fullcontent, _ := json.Marshal(out)

	content, _ := p.session.Send(ctx, copilot.MessageOptions{
		Prompt: string(fullcontent),
	})

	return &LLMResponse{
		FinishReason: "stop",
		Content:      content,
	}, nil

}

func (p *GitHubCopilotProvider) GetDefaultModel() string {

	return "gpt-4.1"
}

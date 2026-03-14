package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	copilot "github.com/github/copilot-sdk/go"
)

// GitHubCopilotProvider provides LLM capabilities via the GitHub Copilot SDK.
// It supports two connection modes:
//   - "stdio": spawns a local Copilot CLI process and communicates via stdin/stdout (default SDK behavior)
//   - "grpc": connects to an external Copilot CLI server over TCP
type GitHubCopilotProvider struct {
	uri         string
	connectMode string // "stdio" or "grpc"

	client  *copilot.Client
	session *copilot.Session

	mu sync.Mutex
}

// NewGitHubCopilotProvider creates a new GitHub Copilot provider.
//
// Parameters:
//   - uri: for "grpc" mode, the address of an external CLI server (e.g. "localhost:4321");
//     for "stdio" mode, the path to the Copilot CLI binary (empty string uses the default "copilot" from PATH)
//   - connectMode: "stdio" or "grpc" (defaults to "grpc" if empty)
//   - model: the model identifier to use for the session
func NewGitHubCopilotProvider(uri string, connectMode string, model string) (*GitHubCopilotProvider, error) {
	if connectMode == "" {
		connectMode = "grpc"
	}

	var client *copilot.Client

	switch connectMode {
	case "stdio":
		opts := &copilot.ClientOptions{}
		if uri != "" {
			opts.CLIPath = uri
		}
		client = copilot.NewClient(opts)
	case "grpc":
		client = copilot.NewClient(&copilot.ClientOptions{
			CLIUrl: uri,
		})
	default:
		return nil, fmt.Errorf("unknown connect mode: %s", connectMode)
	}

	if err := client.Start(context.Background()); err != nil {
		return nil, fmt.Errorf(
			"can't connect to GitHub Copilot (%s mode): %w",
			connectMode, err,
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

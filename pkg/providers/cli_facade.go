package providers

import (
	"time"

	cliprovider "github.com/sipeed/picoclaw/pkg/providers/cli"
)

type (
	ClaudeCliProvider     = cliprovider.ClaudeCliProvider
	CodexCliProvider      = cliprovider.CodexCliProvider
	CodexCliAuth          = cliprovider.CodexCliAuth
	GitHubCopilotProvider = cliprovider.GitHubCopilotProvider
)

const CodexHomeEnvVar = cliprovider.CodexHomeEnvVar

func NewClaudeCliProvider(workspace string) *ClaudeCliProvider {
	return cliprovider.NewClaudeCliProvider(workspace)
}

func NewCodexCliProvider(workspace string) *CodexCliProvider {
	return cliprovider.NewCodexCliProvider(workspace)
}

func NewGitHubCopilotProvider(uri string, connectMode string, model string) (*GitHubCopilotProvider, error) {
	return cliprovider.NewGitHubCopilotProvider(uri, connectMode, model)
}

func ReadCodexCliCredentials() (accessToken, accountID string, expiresAt time.Time, err error) {
	return cliprovider.ReadCodexCliCredentials()
}

func CreateCodexCliTokenSource() func() (string, string, error) {
	return cliprovider.CreateCodexCliTokenSource()
}

func NormalizeToolCall(tc ToolCall) ToolCall {
	return cliprovider.NormalizeToolCall(tc)
}

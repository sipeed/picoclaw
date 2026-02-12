package agent

import (
	"context"
	"time"

	"github.com/sipeed/picoclaw/pkg/providers"
)

// Hooks provides lifecycle integration points for the agent loop.
// All fields are optional — nil checks only, zero cost when unused.
type Hooks struct {
	// OnContextBuild is called before building messages to inject extra context.
	// Returns additional context string to include in the system prompt.
	OnContextBuild func(ctx context.Context, query string) (string, error)

	// OnPreTool is called before each tool execution.
	OnPreTool func(ctx context.Context, name string, args map[string]interface{}) error

	// OnPostTool is called after each tool execution with the result and duration.
	OnPostTool func(ctx context.Context, name string, result string, dur time.Duration)

	// OnPreLLM is called before each LLM call, allowing message mutation.
	OnPreLLM func(ctx context.Context, messages []providers.Message) []providers.Message

	// OnPostMessage is called after a complete message exchange is saved.
	OnPostMessage func(ctx context.Context, sessionKey, userMsg, response string)
}

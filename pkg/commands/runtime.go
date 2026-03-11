package commands

import "github.com/sipeed/picoclaw/pkg/config"

// TokenStats represents token usage statistics for the current session
type TokenStats struct {
	Model            string
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
	CallCount        int
}

// Runtime provides runtime dependencies to command handlers. It is constructed
// per-request by the agent loop so that per-request state (like session scope)
// can coexist with long-lived callbacks (like GetModelInfo).
type Runtime struct {
	Config             *config.Config
	GetModelInfo       func() (name, provider string)
	ListAgentIDs       func() []string
	ListDefinitions    func() []Definition
	GetEnabledChannels func() []string
	SwitchModel        func(value string) (oldModel string, err error)
	SwitchChannel      func(value string) error
	CancelCurrentTask  func() bool // Cancel the currently running task, returns true if a task was cancelled
	ClearHistory       func() error
	GetTokenStats      func() *TokenStats // Get token usage statistics for current session
}

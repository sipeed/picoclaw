package commands

import (
	"context"

	"github.com/sipeed/picoclaw/pkg/config"
)

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

	// Session mode control (session-scoped callbacks)
	GetSessionMode func() string // returns "pico" or "cmd"
	SetModeCmd     func() string // switches to cmd mode, returns prompt string
	SetModePico    func() string // switches to pico mode, returns status string

	// Working directory (session-scoped)
	GetWorkDir   func() string // current working directory for this session
	GetWorkspace func() string // agent workspace root

	// File editing — delegates to the loop's handleEditCommand with proper path resolution
	EditFile func(content string) string

	// Token and model usage stats
	GetTokenUsage func() (promptTokens, completionTokens, requests int64)

	// One-shot AI query from cmd mode (/hipico)
	RunOneShot func(ctx context.Context, message string) (string, error)

	// Session history management
	ClearSession   func() error // clears all history and summary, saves
	CompactSession func() error // runs summarization synchronously
}

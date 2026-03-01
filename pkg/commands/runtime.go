package commands

import (
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/session"
)

// SessionOps defines the session lifecycle operations command handlers rely on.
// Implementations are expected to be scope-aware and deterministic for a given scopeKey.
type SessionOps interface {
	// ResolveActive returns the active session key for scopeKey, creating default scope state if needed.
	ResolveActive(scopeKey string) (string, error)
	// StartNew creates and activates a new session for scopeKey and returns its key.
	StartNew(scopeKey string) (string, error)
	// List returns ordered session metadata for scopeKey.
	List(scopeKey string) ([]session.SessionMeta, error)
	// Resume activates the session at a 1-based index within scopeKey and returns its key.
	Resume(scopeKey string, index int) (string, error)
	// Prune deletes older sessions for scopeKey according to limit and returns deleted session keys.
	Prune(scopeKey string, limit int) ([]string, error)
}

// Runtime exposes the minimal agent runtime state needed by command handlers.
type Runtime interface {
	// Channel returns the inbound channel name (for example: telegram, whatsapp).
	Channel() string
	// ScopeKey returns the resolved scope identifier used for session operations.
	ScopeKey() string
	// SessionOps returns scoped session lifecycle operations.
	SessionOps() SessionOps
	// Config returns process config for read-only access by handlers; callers must not mutate it.
	Config() *config.Config
}

package session

import "github.com/sipeed/picoclaw/pkg/providers"

// SessionStore defines the persistence operations used by the agent loop.
// Both SessionManager (legacy JSON backend) and JSONLBackend satisfy this
// interface, allowing the storage layer to be swapped without touching the
// agent loop code.
type SessionStore interface {
	AddMessage(sessionKey, role, content string)
	AddFullMessage(sessionKey string, msg providers.Message)
	GetHistory(key string) []providers.Message
	GetSummary(key string) string
	SetSummary(key, summary string)
	SetHistory(key string, history []providers.Message)
	TruncateHistory(key string, keepLast int)
	Save(key string) error
	Close() error
}

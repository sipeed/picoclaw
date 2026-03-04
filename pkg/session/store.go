package session

import "time"

// SessionStore is the storage interface for sessions and turns.
// Phase 0 provides a SQLite implementation; LegacyAdapter wraps it to
// expose the same API as SessionManager.
type SessionStore interface { //nolint:interfacebloat // storage facade — methods are logically grouped
	// Session CRUD
	Create(key string, opts *CreateOpts) error
	Get(key string) (*SessionInfo, error)
	List(filter *ListFilter) ([]*SessionInfo, error)
	SetStatus(key, status string) error
	SetSummary(key, summary string) error
	Delete(key string) error
	Children(key string) ([]*SessionInfo, error)

	// Turn operations
	Append(sessionKey string, turn *Turn) error
	Turns(sessionKey string, sinceSeq int) ([]*Turn, error)
	LastTurn(sessionKey string) (*Turn, error)
	TurnCount(sessionKey string) (int, error)
	Compact(sessionKey string, upToSeq int, summary string) error

	// DAG operations
	Fork(parentKey, childKey string, opts *CreateOpts) error

	// Maintenance
	Prune(olderThan time.Duration) (int, error)
	Close() error
}

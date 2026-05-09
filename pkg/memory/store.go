package memory

import (
	"context"

	"github.com/sipeed/picoclaw/pkg/providers"
)

// SessionMessage represents a message in a session with optional tags and metadata.
type SessionMessage struct {
	Role     string
	Content  string
	Tags     []string
	Metadata map[string]interface{}
}

// Store defines an interface for persistent session storage.
// Each method is an atomic operation — there is no separate Save() call.
type Store interface {
	// AddMessage appends a simple text message to a session.
	AddMessage(ctx context.Context, sessionKey, role, content string) error

	// AddFullMessage appends a complete message (with tool calls, etc.) to a session.
	AddFullMessage(ctx context.Context, sessionKey string, msg providers.Message) error

	// GetHistory returns all messages for a session in insertion order.
	// Returns an empty slice (not nil) if the session does not exist.
	GetHistory(ctx context.Context, sessionKey string) ([]providers.Message, error)

	// GetSummary returns the conversation summary for a session.
	// Returns an empty string if no summary exists.
	GetSummary(ctx context.Context, sessionKey string) (string, error)

	// SetSummary updates the conversation summary for a session.
	SetSummary(ctx context.Context, sessionKey, summary string) error

	// TruncateHistory removes all but the last keepLast messages from a session.
	// If keepLast <= 0, all messages are removed.
	TruncateHistory(ctx context.Context, sessionKey string, keepLast int) error

	// SetHistory replaces all messages in a session with the provided history.
	SetHistory(ctx context.Context, sessionKey string, history []providers.Message) error

	// Compact reclaims storage by physically removing logically truncated
	// data. Backends that do not accumulate dead data may return nil.
	Compact(ctx context.Context, sessionKey string) error

	// ListSessions returns all known session keys.
	ListSessions() []string

	// Close releases any resources held by the store.
	Close() error

	// Research report methods
	ListResearchReports() ([]ResearchReport, error)
	UpdateResearchReport(report ResearchReport) error
}

// researchStore is a simple in-memory store for research reports.
type researchStore struct{}

// ListResearchReports returns all research reports from storage
func (s *researchStore) ListResearchReports() ([]ResearchReport, error) {
	// TODO: Implement SQLite query for research_reports table
	return []ResearchReport{
		{ID: "1", Title: "AI trends 2026", Pages: 18, Words: 5400, Status: "in-progress", Progress: 75},
		{ID: "2", Title: "Quantum computing", Pages: 42, Words: 12600, Status: "complete"},
	}, nil
}

// UpdateResearchReport updates a research report status or progress
func (s *researchStore) UpdateResearchReport(report ResearchReport) error {
	// TODO: Implement SQLite update for research_reports table
	return nil
}

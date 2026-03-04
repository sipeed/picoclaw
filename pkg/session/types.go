package session

import (
	"time"

	"github.com/sipeed/picoclaw/pkg/providers"
)

// TurnKind classifies a turn within a session.

type TurnKind int

const (
	TurnNormal TurnKind = iota // Regular conversation turn

	TurnReport // Subagent report turn

	TurnForkPoint // Fork point for child sessions

)

// Escalation turn kinds — explicit values to keep stable across versions.

const (
	TurnQuestion TurnKind = 10 // Subagent → conductor question (escalation)

	TurnPlanSubmit TurnKind = 11 // Subagent plan submission for review

)

// Turn represents a single conversation turn persisted in the store.

type Turn struct {
	ID string

	SessionKey string

	OriginKey string

	Summary string

	Author string

	Seq int

	Kind TurnKind

	Messages []providers.Message

	CreatedAt time.Time

	Meta map[string]string
}

// SessionInfo holds metadata about a session.

type SessionInfo struct {
	Key string

	ParentKey string

	ForkTurnID string

	Status string

	Label string

	Summary string

	TurnCount int

	CreatedAt time.Time

	UpdatedAt time.Time
}

// CreateOpts are options for creating a new session.

type CreateOpts struct {
	ParentKey string

	ForkTurnID string

	Label string
}

// ListFilter constrains which sessions are returned by List.

type ListFilter struct {
	ParentKey string

	Status string
}

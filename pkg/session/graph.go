package session

import (
	"errors"
	"sync"

	"github.com/sipeed/picoclaw/pkg/providers"
)

// SessionGraph is a thin wrapper around SessionStore that provides

// structured turn-writing via BeginTurn/TurnWriter.

// It does NOT replace LegacyAdapter — existing call sites remain unchanged.

// Future phases will migrate callers to use SessionGraph directly.

type SessionGraph struct {
	store SessionStore
}

// NewSessionGraph creates a SessionGraph backed by the given store.

func NewSessionGraph(store SessionStore) *SessionGraph {
	return &SessionGraph{store: store}
}

// Messages returns all messages for the session by reading turns from the store.

func (g *SessionGraph) Messages(sessionKey string) ([]providers.Message, error) {
	turns, err := g.store.Turns(sessionKey, 0)
	if err != nil {
		return nil, err
	}

	var msgs []providers.Message

	for _, t := range turns {
		msgs = append(msgs, t.Messages...)
	}

	if msgs == nil {
		msgs = []providers.Message{}
	}

	return msgs, nil
}

// BeginTurn starts a new turn that can be built up incrementally

// and committed atomically.

func (g *SessionGraph) BeginTurn(sessionKey string, kind TurnKind) *TurnWriter {
	return &TurnWriter{
		store: g.store,

		sessionKey: sessionKey,

		turn: Turn{
			SessionKey: sessionKey,

			Kind: kind,
		},
	}
}

// TurnWriter accumulates messages for a single turn and commits them atomically.

type TurnWriter struct {
	mu sync.Mutex

	store SessionStore

	sessionKey string

	turn Turn

	committed bool

	discarded bool
}

// Add appends a message to the pending turn.

func (tw *TurnWriter) Add(msg providers.Message) {
	tw.mu.Lock()

	defer tw.mu.Unlock()

	tw.turn.Messages = append(tw.turn.Messages, msg)
}

// SetOrigin sets the origin session key for this turn (e.g. subagent source).

func (tw *TurnWriter) SetOrigin(sessionKey string) {
	tw.mu.Lock()

	defer tw.mu.Unlock()

	tw.turn.OriginKey = sessionKey
}

// SetAuthor sets the author field for this turn.

func (tw *TurnWriter) SetAuthor(author string) {
	tw.mu.Lock()

	defer tw.mu.Unlock()

	tw.turn.Author = author
}

// Commit writes the accumulated turn to the store.

// Returns an error if already committed or discarded.

func (tw *TurnWriter) Commit() error {
	tw.mu.Lock()

	defer tw.mu.Unlock()

	if tw.committed {
		return errors.New("turn already committed")
	}

	if tw.discarded {
		return errors.New("turn already discarded")
	}

	tw.committed = true

	return tw.store.Append(tw.sessionKey, &tw.turn)
}

// Discard marks the turn as abandoned — nothing is written.

func (tw *TurnWriter) Discard() {
	tw.mu.Lock()

	defer tw.mu.Unlock()

	tw.discarded = true
}

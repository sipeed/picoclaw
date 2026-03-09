package commands

import (
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/session"
)

// SessionOps defines the session lifecycle operations that command handlers
// rely on. Implementations are expected to be scope-aware and deterministic
// for a given scopeKey. SessionManager satisfies this interface.
type SessionOps interface {
	StartNew(scopeKey string) (string, error)
	List(scopeKey string) ([]session.SessionMeta, error)
	Resume(scopeKey string, index int) (string, error)
	Prune(scopeKey string, limit int) ([]string, error)
}

// Runtime provides capabilities and services to command handlers — the
// "what can I do?" side of the handler contract.
//
// It is constructed per-request by the agent loop so that late-bound
// dependencies (e.g. session manager for the resolved agent) are always
// current. Fields here are either read-only accessors or mutation callbacks;
// none describe the identity of the incoming message (that belongs in
// [Request]).
//
// Guideline for placing a new field:
//
//	Request  — identity / context: channel, chat, sender, scope key, text.
//	Runtime  — service / capability: config access, query funcs, mutation funcs,
//	           service interfaces (SessionOps, etc.).
type Runtime struct {
	Config             *config.Config
	GetModelInfo       func() (name, provider string)
	ListAgentIDs       func() []string
	ListDefinitions    func() []Definition
	GetEnabledChannels func() []string
	SwitchModel        func(value string) (oldModel string, err error)
	SwitchChannel      func(value string) error
	SessionOps         SessionOps // nil when session commands are unavailable
}

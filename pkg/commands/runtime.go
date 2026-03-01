package commands

import (
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/session"
)

type SessionOps interface {
	ResolveActive(scopeKey string) (string, error)
	StartNew(scopeKey string) (string, error)
	List(scopeKey string) ([]session.SessionMeta, error)
	Resume(scopeKey string, index int) (string, error)
	Prune(scopeKey string, limit int) ([]string, error)
}

type Runtime interface {
	Channel() string
	ScopeKey() string
	SessionOps() SessionOps
	Config() *config.Config
}

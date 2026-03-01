package commands

import (
	"context"
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/session"
)

type sessionHandlerFakeSessionOps struct {
	startNewScopeKeys []string
	startNewValue     string
	startNewErr       error

	pruneScopeKeys []string
	pruneLimits    []int
	pruneValue     []string
	pruneErr       error

	resumeScopeKeys []string
	resumeIndices   []int
	resumeValue     string
	resumeErr       error
}

func (f *sessionHandlerFakeSessionOps) ResolveActive(scopeKey string) (string, error) {
	return "", nil
}

func (f *sessionHandlerFakeSessionOps) StartNew(scopeKey string) (string, error) {
	f.startNewScopeKeys = append(f.startNewScopeKeys, scopeKey)
	return f.startNewValue, f.startNewErr
}

func (f *sessionHandlerFakeSessionOps) List(scopeKey string) ([]session.SessionMeta, error) {
	return nil, nil
}

func (f *sessionHandlerFakeSessionOps) Resume(scopeKey string, index int) (string, error) {
	f.resumeScopeKeys = append(f.resumeScopeKeys, scopeKey)
	f.resumeIndices = append(f.resumeIndices, index)
	return f.resumeValue, f.resumeErr
}

func (f *sessionHandlerFakeSessionOps) Prune(scopeKey string, limit int) ([]string, error) {
	f.pruneScopeKeys = append(f.pruneScopeKeys, scopeKey)
	f.pruneLimits = append(f.pruneLimits, limit)
	return f.pruneValue, f.pruneErr
}

type sessionHandlerFakeRuntime struct {
	channel string
	scope   string
	ops     SessionOps
	cfg     *config.Config
}

func (f *sessionHandlerFakeRuntime) Channel() string {
	return f.channel
}

func (f *sessionHandlerFakeRuntime) ScopeKey() string {
	return f.scope
}

func (f *sessionHandlerFakeRuntime) SessionOps() SessionOps {
	return f.ops
}

func (f *sessionHandlerFakeRuntime) Config() *config.Config {
	return f.cfg
}

func TestSessionHandlers_New_UsesRuntimeSessionOps(t *testing.T) {
	t.Helper()

	ops := &sessionHandlerFakeSessionOps{
		startNewValue: "scope#2",
		pruneValue:    []string{"scope#1"},
	}
	runtime := &sessionHandlerFakeRuntime{
		channel: "whatsapp",
		scope:   "scope",
		ops:     ops,
		cfg: &config.Config{
			Session: config.SessionConfig{BacklogLimit: 7},
		},
	}

	ctx := WithRuntime(context.Background(), runtime)

	var reply string
	ex := NewExecutor(NewRegistry(BuiltinDefinitions(nil)))
	res := ex.Execute(ctx, Request{
		Channel: "whatsapp",
		Text:    "/new",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	})

	if res.Outcome != OutcomeHandled {
		t.Fatalf("outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	if len(ops.startNewScopeKeys) != 1 || ops.startNewScopeKeys[0] != "scope" {
		t.Fatalf("startNew calls=%v, want [scope]", ops.startNewScopeKeys)
	}
	if len(ops.pruneScopeKeys) != 1 || ops.pruneScopeKeys[0] != "scope" {
		t.Fatalf("prune scope calls=%v, want [scope]", ops.pruneScopeKeys)
	}
	if len(ops.pruneLimits) != 1 || ops.pruneLimits[0] != 7 {
		t.Fatalf("prune limits=%v, want [7]", ops.pruneLimits)
	}
	if reply != "Started new session: scope#2 (pruned 1 old session(s))" {
		t.Fatalf("reply=%q", reply)
	}
}

func TestSessionHandlers_SessionResume_UsesRuntimeSessionOps(t *testing.T) {
	t.Helper()

	ops := &sessionHandlerFakeSessionOps{resumeValue: "scope#3"}
	runtime := &sessionHandlerFakeRuntime{
		channel: "whatsapp",
		scope:   "scope",
		ops:     ops,
		cfg:     &config.Config{},
	}

	ctx := WithRuntime(context.Background(), runtime)

	var reply string
	ex := NewExecutor(NewRegistry(BuiltinDefinitions(nil)))
	res := ex.Execute(ctx, Request{
		Channel: "whatsapp",
		Text:    "/session resume 3",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	})

	if res.Outcome != OutcomeHandled {
		t.Fatalf("outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	if len(ops.resumeScopeKeys) != 1 || ops.resumeScopeKeys[0] != "scope" {
		t.Fatalf("resume scope calls=%v, want [scope]", ops.resumeScopeKeys)
	}
	if len(ops.resumeIndices) != 1 || ops.resumeIndices[0] != 3 {
		t.Fatalf("resume indices=%v, want [3]", ops.resumeIndices)
	}
	if reply != "Resumed session 3: scope#3" {
		t.Fatalf("reply=%q", reply)
	}
}

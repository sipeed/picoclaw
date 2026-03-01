package commands

import (
	"context"
	"errors"
	"testing"
	"time"

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

	listScopeKeys []string
	listValue     []session.SessionMeta
	listErr       error

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
	f.listScopeKeys = append(f.listScopeKeys, scopeKey)
	return f.listValue, f.listErr
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

	var reply string
	ex := NewExecutor(NewRegistry(BuiltinDefinitionsWithRuntime(nil, runtime)))
	res := ex.Execute(context.Background(), Request{
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
	ops := &sessionHandlerFakeSessionOps{resumeValue: "scope#3"}
	runtime := &sessionHandlerFakeRuntime{
		channel: "whatsapp",
		scope:   "scope",
		ops:     ops,
		cfg:     &config.Config{},
	}

	var reply string
	ex := NewExecutor(NewRegistry(BuiltinDefinitionsWithRuntime(nil, runtime)))
	res := ex.Execute(context.Background(), Request{
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

func TestSessionHandlers_SessionList_UsesRuntimeSessionOps(t *testing.T) {
	ops := &sessionHandlerFakeSessionOps{
		listValue: []session.SessionMeta{
			{
				Ordinal:    1,
				SessionKey: "scope#3",
				UpdatedAt:  time.Date(2026, 3, 1, 9, 7, 0, 0, time.UTC),
				MessageCnt: 4,
				Active:     true,
			},
		},
	}
	runtime := &sessionHandlerFakeRuntime{
		channel: "whatsapp",
		scope:   "scope",
		ops:     ops,
		cfg:     &config.Config{},
	}

	var reply string
	ex := NewExecutor(NewRegistry(BuiltinDefinitionsWithRuntime(nil, runtime)))
	res := ex.Execute(context.Background(), Request{
		Channel: "whatsapp",
		Text:    "/session list",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	})

	if res.Outcome != OutcomeHandled {
		t.Fatalf("outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	if len(ops.listScopeKeys) != 1 || ops.listScopeKeys[0] != "scope" {
		t.Fatalf("list scope calls=%v, want [scope]", ops.listScopeKeys)
	}
	if reply != "Sessions for current chat:\n1. [*] scope#3 (4 msgs, updated 2026-03-01 09:07)" {
		t.Fatalf("reply=%q", reply)
	}
}

func TestSessionHandlers_MissingRuntime_Passthrough(t *testing.T) {
	ex := NewExecutor(NewRegistry(BuiltinDefinitionsWithRuntime(nil, nil)))

	for _, input := range []string{"/new", "/session list"} {
		res := ex.Execute(context.Background(), Request{
			Channel: "whatsapp",
			Text:    input,
		})
		if res.Outcome != OutcomePassthrough {
			t.Fatalf("text=%q outcome=%v, want=%v", input, res.Outcome, OutcomePassthrough)
		}
	}
}

func TestSessionHandlers_NilSessionOps_Passthrough(t *testing.T) {
	runtime := &sessionHandlerFakeRuntime{
		channel: "whatsapp",
		scope:   "scope",
		ops:     nil,
		cfg:     &config.Config{},
	}
	ex := NewExecutor(NewRegistry(BuiltinDefinitionsWithRuntime(nil, runtime)))

	for _, input := range []string{"/new", "/session list"} {
		res := ex.Execute(context.Background(), Request{
			Channel: "whatsapp",
			Text:    input,
		})
		if res.Outcome != OutcomePassthrough {
			t.Fatalf("text=%q outcome=%v, want=%v", input, res.Outcome, OutcomePassthrough)
		}
	}
}

func TestSessionHandlers_EmptyScope_Passthrough(t *testing.T) {
	runtime := &sessionHandlerFakeRuntime{
		channel: "whatsapp",
		scope:   "   ",
		ops:     &sessionHandlerFakeSessionOps{},
		cfg:     &config.Config{},
	}
	ex := NewExecutor(NewRegistry(BuiltinDefinitionsWithRuntime(nil, runtime)))

	for _, input := range []string{"/new", "/session list"} {
		res := ex.Execute(context.Background(), Request{
			Channel: "whatsapp",
			Text:    input,
		})
		if res.Outcome != OutcomePassthrough {
			t.Fatalf("text=%q outcome=%v, want=%v", input, res.Outcome, OutcomePassthrough)
		}
	}
}

func TestSessionHandlers_ErrorAndValidationReplies(t *testing.T) {
	tests := []struct {
		name      string
		text      string
		ops       *sessionHandlerFakeSessionOps
		wantReply string
	}{
		{
			name:      "start new error",
			text:      "/new",
			ops:       &sessionHandlerFakeSessionOps{startNewErr: errors.New("boom")},
			wantReply: "Failed to start new session: boom",
		},
		{
			name: "prune error",
			text: "/new",
			ops: &sessionHandlerFakeSessionOps{
				startNewValue: "scope#2",
				pruneErr:      errors.New("prune failed"),
			},
			wantReply: "Started new session (scope#2), but pruning old sessions failed: prune failed",
		},
		{
			name:      "list error",
			text:      "/session list",
			ops:       &sessionHandlerFakeSessionOps{listErr: errors.New("list failed")},
			wantReply: "Failed to list sessions: list failed",
		},
		{
			name:      "resume error",
			text:      "/session resume 2",
			ops:       &sessionHandlerFakeSessionOps{resumeErr: errors.New("resume failed")},
			wantReply: "Failed to resume session 2: resume failed",
		},
		{
			name:      "resume missing index",
			text:      "/session resume",
			ops:       &sessionHandlerFakeSessionOps{},
			wantReply: "Usage: /session resume <index>",
		},
		{
			name:      "resume non numeric index",
			text:      "/session resume abc",
			ops:       &sessionHandlerFakeSessionOps{},
			wantReply: "Usage: /session resume <index>",
		},
		{
			name:      "resume zero index",
			text:      "/session resume 0",
			ops:       &sessionHandlerFakeSessionOps{},
			wantReply: "Usage: /session resume <index>",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runtime := &sessionHandlerFakeRuntime{
				channel: "whatsapp",
				scope:   "scope",
				ops:     tc.ops,
				cfg:     &config.Config{},
			}

			var reply string
			ex := NewExecutor(NewRegistry(BuiltinDefinitionsWithRuntime(nil, runtime)))
			res := ex.Execute(context.Background(), Request{
				Channel: "whatsapp",
				Text:    tc.text,
				Reply: func(text string) error {
					reply = text
					return nil
				},
			})

			if res.Outcome != OutcomeHandled {
				t.Fatalf("outcome=%v, want=%v", res.Outcome, OutcomeHandled)
			}
			if reply != tc.wantReply {
				t.Fatalf("reply=%q, want=%q", reply, tc.wantReply)
			}
		})
	}
}

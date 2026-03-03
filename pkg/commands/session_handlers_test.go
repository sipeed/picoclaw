package commands

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/session"
)

type fakeSessionOps struct {
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

func (f *fakeSessionOps) ResolveActive(scopeKey string) (string, error) { return "", nil }

func (f *fakeSessionOps) StartNew(scopeKey string) (string, error) {
	f.startNewScopeKeys = append(f.startNewScopeKeys, scopeKey)
	return f.startNewValue, f.startNewErr
}

func (f *fakeSessionOps) List(scopeKey string) ([]session.SessionMeta, error) {
	f.listScopeKeys = append(f.listScopeKeys, scopeKey)
	return f.listValue, f.listErr
}

func (f *fakeSessionOps) Resume(scopeKey string, index int) (string, error) {
	f.resumeScopeKeys = append(f.resumeScopeKeys, scopeKey)
	f.resumeIndices = append(f.resumeIndices, index)
	return f.resumeValue, f.resumeErr
}

func (f *fakeSessionOps) Prune(scopeKey string, limit int) ([]string, error) {
	f.pruneScopeKeys = append(f.pruneScopeKeys, scopeKey)
	f.pruneLimits = append(f.pruneLimits, limit)
	return f.pruneValue, f.pruneErr
}

func TestSessionHandlers_New_UsesRuntimeSessionOps(t *testing.T) {
	ops := &fakeSessionOps{
		startNewValue: "scope#2",
		pruneValue:    []string{"scope#1"},
	}
	rt := &Runtime{
		SessionOps: ops,
		Config:     &config.Config{Session: config.SessionConfig{BacklogLimit: 7}},
	}
	ex := NewExecutor(NewRegistry(BuiltinDefinitions()), rt)

	var reply string
	res := ex.Execute(context.Background(), Request{
		Channel:  "whatsapp",
		ScopeKey: "scope",
		Text:     "/new",
		Reply:    func(text string) error { reply = text; return nil },
	})

	if res.Outcome != OutcomeHandled {
		t.Fatalf("outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	if len(ops.startNewScopeKeys) != 1 || ops.startNewScopeKeys[0] != "scope" {
		t.Fatalf("startNew calls=%v, want [scope]", ops.startNewScopeKeys)
	}
	if len(ops.pruneLimits) != 1 || ops.pruneLimits[0] != 7 {
		t.Fatalf("prune limits=%v, want [7]", ops.pruneLimits)
	}
	if reply != "Started new session: scope#2 (pruned 1 old session(s))" {
		t.Fatalf("reply=%q", reply)
	}
}

func TestSessionHandlers_SessionResume(t *testing.T) {
	ops := &fakeSessionOps{resumeValue: "scope#3"}
	rt := &Runtime{SessionOps: ops}
	ex := NewExecutor(NewRegistry(BuiltinDefinitions()), rt)

	var reply string
	res := ex.Execute(context.Background(), Request{
		ScopeKey: "scope",
		Text:     "/session resume 3",
		Reply:    func(text string) error { reply = text; return nil },
	})

	if res.Outcome != OutcomeHandled {
		t.Fatalf("outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	if len(ops.resumeIndices) != 1 || ops.resumeIndices[0] != 3 {
		t.Fatalf("resume indices=%v, want [3]", ops.resumeIndices)
	}
	if reply != "Resumed session 3: scope#3" {
		t.Fatalf("reply=%q", reply)
	}
}

func TestSessionHandlers_SessionList(t *testing.T) {
	ops := &fakeSessionOps{
		listValue: []session.SessionMeta{
			{
				Ordinal:    1,
				SessionKey: "scope#3",
				UpdatedAt:  time.Date(2026, 3, 1, 9, 7, 0, 0, time.UTC),
				MessageCnt: 4,
				Active:     true,
				Summary:    "Discussing React performance",
			},
		},
	}
	rt := &Runtime{SessionOps: ops}
	ex := NewExecutor(NewRegistry(BuiltinDefinitions()), rt)

	var reply string
	res := ex.Execute(context.Background(), Request{
		ScopeKey: "scope",
		Text:     "/session list",
		Reply:    func(text string) error { reply = text; return nil },
	})

	if res.Outcome != OutcomeHandled {
		t.Fatalf("outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	// Check key elements: header, active marker, summary, message count, session tag
	for _, want := range []string{"Sessions:", "[*]", "Discussing React performance", "4 msgs", "(#3)"} {
		if !strings.Contains(reply, want) {
			t.Fatalf("reply missing %q, got %q", want, reply)
		}
	}
}

func TestSessionHandlers_NilRuntime_Unavailable(t *testing.T) {
	ex := NewExecutor(NewRegistry(BuiltinDefinitions()), nil)

	for _, text := range []string{"/new", "/session list"} {
		var reply string
		res := ex.Execute(context.Background(), Request{
			ScopeKey: "scope",
			Text:     text,
			Reply:    func(text string) error { reply = text; return nil },
		})
		if res.Outcome != OutcomeHandled {
			t.Fatalf("text=%q outcome=%v, want=%v", text, res.Outcome, OutcomeHandled)
		}
		if reply != unavailableMsg {
			t.Fatalf("text=%q reply=%q, want unavailable", text, reply)
		}
	}
}

func TestSessionHandlers_NilSessionOps_Unavailable(t *testing.T) {
	rt := &Runtime{} // SessionOps is nil
	ex := NewExecutor(NewRegistry(BuiltinDefinitions()), rt)

	for _, text := range []string{"/new", "/session list"} {
		var reply string
		res := ex.Execute(context.Background(), Request{
			ScopeKey: "scope",
			Text:     text,
			Reply:    func(text string) error { reply = text; return nil },
		})
		if res.Outcome != OutcomeHandled {
			t.Fatalf("text=%q outcome=%v, want=%v", text, res.Outcome, OutcomeHandled)
		}
		if reply != unavailableMsg {
			t.Fatalf("text=%q reply=%q, want unavailable", text, reply)
		}
	}
}

func TestSessionHandlers_EmptyScopeKey_Unavailable(t *testing.T) {
	ops := &fakeSessionOps{}
	rt := &Runtime{SessionOps: ops}
	ex := NewExecutor(NewRegistry(BuiltinDefinitions()), rt)

	for _, text := range []string{"/new", "/session list"} {
		var reply string
		res := ex.Execute(context.Background(), Request{
			ScopeKey: "   ", // whitespace-only
			Text:     text,
			Reply:    func(text string) error { reply = text; return nil },
		})
		if res.Outcome != OutcomeHandled {
			t.Fatalf("text=%q outcome=%v, want=%v", text, res.Outcome, OutcomeHandled)
		}
		if reply != unavailableMsg {
			t.Fatalf("text=%q reply=%q, want unavailable", text, reply)
		}
	}
}

func TestSessionHandlers_ErrorReplies(t *testing.T) {
	tests := []struct {
		name      string
		text      string
		scopeKey  string
		ops       *fakeSessionOps
		wantReply string
	}{
		{
			name:      "start new error",
			text:      "/new",
			scopeKey:  "scope",
			ops:       &fakeSessionOps{startNewErr: errors.New("boom")},
			wantReply: "Failed to start new session: boom",
		},
		{
			name:     "prune error",
			text:     "/new",
			scopeKey: "scope",
			ops: &fakeSessionOps{
				startNewValue: "scope#2",
				pruneErr:      errors.New("prune failed"),
			},
			wantReply: "Started new session (scope#2), but pruning old sessions failed: prune failed",
		},
		{
			name:      "list error",
			text:      "/session list",
			scopeKey:  "scope",
			ops:       &fakeSessionOps{listErr: errors.New("list failed")},
			wantReply: "Failed to list sessions: list failed",
		},
		{
			name:      "resume error",
			text:      "/session resume 2",
			scopeKey:  "scope",
			ops:       &fakeSessionOps{resumeErr: errors.New("resume failed")},
			wantReply: "Failed to resume session 2: resume failed",
		},
		{
			name:      "resume missing index",
			text:      "/session resume",
			scopeKey:  "scope",
			ops:       &fakeSessionOps{},
			wantReply: "Usage: /session resume <index>",
		},
		{
			name:      "resume non numeric index",
			text:      "/session resume abc",
			scopeKey:  "scope",
			ops:       &fakeSessionOps{},
			wantReply: "Usage: /session resume <index>",
		},
		{
			name:      "resume zero index",
			text:      "/session resume 0",
			scopeKey:  "scope",
			ops:       &fakeSessionOps{},
			wantReply: "Usage: /session resume <index>",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rt := &Runtime{SessionOps: tc.ops}
			ex := NewExecutor(NewRegistry(BuiltinDefinitions()), rt)

			var reply string
			res := ex.Execute(context.Background(), Request{
				ScopeKey: tc.scopeKey,
				Text:     tc.text,
				Reply:    func(text string) error { reply = text; return nil },
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

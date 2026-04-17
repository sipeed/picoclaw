package session_test

import (
	"fmt"
	"testing"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/memory"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/routing"
	"github.com/sipeed/picoclaw/pkg/session"
)

// Compile-time interface satisfaction checks.
var (
	_ session.SessionStore = (*session.SessionManager)(nil)
	_ session.SessionStore = (*session.JSONLBackend)(nil)
)

func newBackend(t *testing.T) *session.JSONLBackend {
	t.Helper()
	store, err := memory.NewJSONLStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	return session.NewJSONLBackend(store)
}

func TestJSONLBackend_AddAndGetHistory(t *testing.T) {
	b := newBackend(t)

	b.AddMessage("s1", "user", "hello")
	b.AddMessage("s1", "assistant", "hi")

	history := b.GetHistory("s1")
	if len(history) != 2 {
		t.Fatalf("got %d messages, want 2", len(history))
	}
	if history[0].Role != "user" || history[0].Content != "hello" {
		t.Errorf("msg[0] = %+v", history[0])
	}
	if history[1].Role != "assistant" || history[1].Content != "hi" {
		t.Errorf("msg[1] = %+v", history[1])
	}
}

func TestJSONLBackend_AddFullMessage(t *testing.T) {
	b := newBackend(t)

	msg := providers.Message{
		Role:    "assistant",
		Content: "done",
		ToolCalls: []providers.ToolCall{
			{ID: "tc1", Function: &providers.FunctionCall{Name: "read_file", Arguments: `{"path":"x"}`}},
		},
	}
	b.AddFullMessage("s1", msg)

	history := b.GetHistory("s1")
	if len(history) != 1 {
		t.Fatalf("got %d, want 1", len(history))
	}
	if len(history[0].ToolCalls) != 1 || history[0].ToolCalls[0].ID != "tc1" {
		t.Errorf("tool calls = %+v", history[0].ToolCalls)
	}
}

func TestJSONLBackend_Summary(t *testing.T) {
	b := newBackend(t)

	if got := b.GetSummary("s1"); got != "" {
		t.Errorf("got %q, want empty", got)
	}

	b.SetSummary("s1", "test summary")
	if got := b.GetSummary("s1"); got != "test summary" {
		t.Errorf("got %q, want %q", got, "test summary")
	}
}

func TestJSONLBackend_TruncateAndSave(t *testing.T) {
	b := newBackend(t)

	for i := 0; i < 10; i++ {
		b.AddMessage("s1", "user", fmt.Sprintf("msg %d", i))
	}
	b.TruncateHistory("s1", 3)

	history := b.GetHistory("s1")
	if len(history) != 3 {
		t.Fatalf("got %d, want 3", len(history))
	}
	if history[0].Content != "msg 7" {
		t.Errorf("got %q, want %q", history[0].Content, "msg 7")
	}

	// Save triggers compaction.
	if err := b.Save("s1"); err != nil {
		t.Fatal(err)
	}

	// Messages still accessible after compaction.
	history = b.GetHistory("s1")
	if len(history) != 3 {
		t.Fatalf("after save: got %d, want 3", len(history))
	}
}

func TestJSONLBackend_SetHistory(t *testing.T) {
	b := newBackend(t)
	b.AddMessage("s1", "user", "old")

	b.SetHistory("s1", []providers.Message{
		{Role: "user", Content: "new1"},
		{Role: "assistant", Content: "new2"},
	})

	history := b.GetHistory("s1")
	if len(history) != 2 {
		t.Fatalf("got %d, want 2", len(history))
	}
	if history[0].Content != "new1" {
		t.Errorf("got %q, want %q", history[0].Content, "new1")
	}
}

func TestJSONLBackend_EmptySession(t *testing.T) {
	b := newBackend(t)

	history := b.GetHistory("nonexistent")
	if history == nil {
		t.Fatal("got nil, want empty slice")
	}
	if len(history) != 0 {
		t.Errorf("got %d, want 0", len(history))
	}
}

func TestJSONLBackend_SessionIsolation(t *testing.T) {
	b := newBackend(t)
	b.AddMessage("s1", "user", "session1")
	b.AddMessage("s2", "user", "session2")

	h1 := b.GetHistory("s1")
	h2 := b.GetHistory("s2")

	if len(h1) != 1 || h1[0].Content != "session1" {
		t.Errorf("s1: %+v", h1)
	}
	if len(h2) != 1 || h2[0].Content != "session2" {
		t.Errorf("s2: %+v", h2)
	}
}

func TestJSONLBackend_SummarizeFlow(t *testing.T) {
	// Simulates the real summarization flow in the agent loop:
	// SetSummary → TruncateHistory → Save
	b := newBackend(t)

	for i := 0; i < 20; i++ {
		b.AddMessage("s1", "user", fmt.Sprintf("msg %d", i))
	}

	b.SetSummary("s1", "conversation about testing")
	b.TruncateHistory("s1", 4)
	if err := b.Save("s1"); err != nil {
		t.Fatal(err)
	}

	if got := b.GetSummary("s1"); got != "conversation about testing" {
		t.Errorf("summary = %q", got)
	}
	history := b.GetHistory("s1")
	if len(history) != 4 {
		t.Fatalf("got %d messages, want 4", len(history))
	}
	if history[0].Content != "msg 16" {
		t.Errorf("first message = %q, want %q", history[0].Content, "msg 16")
	}
}

func TestJSONLBackend_ResolveAliasAndPersistMetadata(t *testing.T) {
	b := newBackend(t)

	scope := &session.SessionScope{
		Version:    session.ScopeVersionV1,
		AgentID:    "main",
		Channel:    "telegram",
		Account:    "default",
		Dimensions: []string{"chat"},
		Values: map[string]string{
			"chat": "group:c1",
		},
	}
	b.EnsureSessionMetadata("canonical", scope, []string{"legacy"})

	if got := b.ResolveSessionKey("legacy"); got != "canonical" {
		t.Fatalf("ResolveSessionKey() = %q, want %q", got, "canonical")
	}

	b.AddMessage("legacy", "user", "hello through alias")
	history := b.GetHistory("canonical")
	if len(history) != 1 {
		t.Fatalf("len(history) = %d, want 1", len(history))
	}
	if history[0].Content != "hello through alias" {
		t.Fatalf("history[0].Content = %q, want %q", history[0].Content, "hello through alias")
	}

	resolvedScope := b.GetSessionScope("legacy")
	if resolvedScope == nil {
		t.Fatal("GetSessionScope() returned nil")
	}
	if resolvedScope.AgentID != scope.AgentID || resolvedScope.Values["chat"] != scope.Values["chat"] {
		t.Fatalf("GetSessionScope() = %+v, want %+v", resolvedScope, scope)
	}
}

func TestJSONLBackend_EnsureSessionMetadata_PromotesLegacyAliasHistory(t *testing.T) {
	b := newBackend(t)

	legacyKey := "agent:main:direct:legacy-user"
	b.AddMessage(legacyKey, "user", "legacy history")
	b.SetSummary(legacyKey, "legacy summary")

	canonicalKey := session.BuildOpaqueSessionKey(legacyKey)
	b.EnsureSessionMetadata(canonicalKey, &session.SessionScope{
		Version: session.ScopeVersionV1,
		AgentID: "main",
	}, []string{legacyKey})

	if got := b.ResolveSessionKey(legacyKey); got != canonicalKey {
		t.Fatalf("ResolveSessionKey() = %q, want %q", got, canonicalKey)
	}
	history := b.GetHistory(canonicalKey)
	if len(history) != 1 || history[0].Content != "legacy history" {
		t.Fatalf("promoted history = %+v", history)
	}
	if summary := b.GetSummary(canonicalKey); summary != "legacy summary" {
		t.Fatalf("promoted summary = %q, want %q", summary, "legacy summary")
	}
}

func TestJSONLBackend_EnsureSessionMetadata_PromotesLegacyPicoDirectAliasHistory(t *testing.T) {
	b := newBackend(t)

	legacyKey := "agent:main:pico:direct:pico:session-123"
	b.AddMessage(legacyKey, "user", "legacy pico history")

	scope := &session.SessionScope{
		Version:    session.ScopeVersionV1,
		AgentID:    "main",
		Channel:    "pico",
		Account:    "default",
		Dimensions: []string{"sender"},
		Values: map[string]string{
			"sender": "pico-user",
		},
	}
	allocation := session.AllocateRouteSession(session.AllocationInput{
		AgentID: "main",
		Context: bus.InboundContext{
			Channel:  "pico",
			Account:  "default",
			ChatID:   "pico:session-123",
			ChatType: "direct",
			SenderID: "pico-user",
		},
		SessionPolicy: routing.SessionPolicy{
			Dimensions: []string{"sender"},
		},
	})

	b.EnsureSessionMetadata(allocation.SessionKey, scope, allocation.SessionAliases)

	if got := b.ResolveSessionKey(legacyKey); got != allocation.SessionKey {
		t.Fatalf("ResolveSessionKey() = %q, want %q", got, allocation.SessionKey)
	}
	history := b.GetHistory(allocation.SessionKey)
	if len(history) != 1 || history[0].Content != "legacy pico history" {
		t.Fatalf("promoted history = %+v", history)
	}
}

func TestJSONLBackend_EnsureSessionMetadata_DoesNotOverwriteNonEmptyCanonicalHistory(t *testing.T) {
	b := newBackend(t)

	canonicalKey := session.BuildOpaqueSessionKey("agent:main:direct:current-user")
	legacyKey := "agent:main:direct:legacy-user"

	b.AddMessage(canonicalKey, "user", "current canonical history")
	b.AddMessage(legacyKey, "user", "legacy history")

	b.EnsureSessionMetadata(canonicalKey, &session.SessionScope{
		Version: session.ScopeVersionV1,
		AgentID: "main",
	}, []string{legacyKey})

	history := b.GetHistory(canonicalKey)
	if len(history) != 1 || history[0].Content != "current canonical history" {
		t.Fatalf("canonical history overwritten: %+v", history)
	}
}

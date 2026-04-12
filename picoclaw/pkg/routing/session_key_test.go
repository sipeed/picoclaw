package routing

import "testing"

func TestBuildAgentMainSessionKey(t *testing.T) {
	got := BuildAgentMainSessionKey("sales")
	want := "agent:sales:main"
	if got != want {
		t.Errorf("BuildAgentMainSessionKey('sales') = %q, want %q", got, want)
	}
}

func TestBuildAgentMainSessionKey_Normalizes(t *testing.T) {
	got := BuildAgentMainSessionKey("Sales Bot")
	want := "agent:sales-bot:main"
	if got != want {
		t.Errorf("BuildAgentMainSessionKey('Sales Bot') = %q, want %q", got, want)
	}
}

func TestBuildAgentPeerSessionKey_DMScopeMain(t *testing.T) {
	got := BuildAgentPeerSessionKey(SessionKeyParams{
		AgentID: "main",
		Channel: "telegram",
		Peer:    &RoutePeer{Kind: "direct", ID: "user123"},
		DMScope: DMScopeMain,
	})
	want := "agent:main:main"
	if got != want {
		t.Errorf("DMScopeMain = %q, want %q", got, want)
	}
}

func TestBuildAgentPeerSessionKey_DMScopePerPeer(t *testing.T) {
	got := BuildAgentPeerSessionKey(SessionKeyParams{
		AgentID: "main",
		Channel: "telegram",
		Peer:    &RoutePeer{Kind: "direct", ID: "user123"},
		DMScope: DMScopePerPeer,
	})
	want := "agent:main:direct:user123"
	if got != want {
		t.Errorf("DMScopePerPeer = %q, want %q", got, want)
	}
}

func TestBuildAgentPeerSessionKey_DMScopePerChannelPeer(t *testing.T) {
	got := BuildAgentPeerSessionKey(SessionKeyParams{
		AgentID: "main",
		Channel: "telegram",
		Peer:    &RoutePeer{Kind: "direct", ID: "user123"},
		DMScope: DMScopePerChannelPeer,
	})
	want := "agent:main:telegram:direct:user123"
	if got != want {
		t.Errorf("DMScopePerChannelPeer = %q, want %q", got, want)
	}
}

func TestBuildAgentPeerSessionKey_DMScopePerAccountChannelPeer(t *testing.T) {
	got := BuildAgentPeerSessionKey(SessionKeyParams{
		AgentID:   "main",
		Channel:   "telegram",
		AccountID: "bot1",
		Peer:      &RoutePeer{Kind: "direct", ID: "User123"},
		DMScope:   DMScopePerAccountChannelPeer,
	})
	want := "agent:main:telegram:bot1:direct:user123"
	if got != want {
		t.Errorf("DMScopePerAccountChannelPeer = %q, want %q", got, want)
	}
}

func TestBuildAgentPeerSessionKey_GroupPeer(t *testing.T) {
	got := BuildAgentPeerSessionKey(SessionKeyParams{
		AgentID: "main",
		Channel: "telegram",
		Peer:    &RoutePeer{Kind: "group", ID: "chat456"},
		DMScope: DMScopePerPeer,
	})
	want := "agent:main:telegram:group:chat456"
	if got != want {
		t.Errorf("GroupPeer = %q, want %q", got, want)
	}
}

func TestBuildAgentPeerSessionKey_NilPeer(t *testing.T) {
	got := BuildAgentPeerSessionKey(SessionKeyParams{
		AgentID: "main",
		Channel: "telegram",
		Peer:    nil,
		DMScope: DMScopePerPeer,
	})
	// nil peer defaults to direct with empty ID, falls to main
	want := "agent:main:main"
	if got != want {
		t.Errorf("NilPeer = %q, want %q", got, want)
	}
}

func TestBuildAgentPeerSessionKey_IdentityLink(t *testing.T) {
	links := map[string][]string{
		"john": {"telegram:user123", "discord:john#1234"},
	}
	got := BuildAgentPeerSessionKey(SessionKeyParams{
		AgentID:       "main",
		Channel:       "telegram",
		Peer:          &RoutePeer{Kind: "direct", ID: "user123"},
		DMScope:       DMScopePerPeer,
		IdentityLinks: links,
	})
	want := "agent:main:direct:john"
	if got != want {
		t.Errorf("IdentityLink = %q, want %q", got, want)
	}
}

func TestResolveLinkedPeerID_CanonicalPeerID(t *testing.T) {
	// When peerID is already in canonical "platform:id" format,
	// it should match identity_links that use the bare ID.
	links := map[string][]string{
		"john": {"123"},
	}
	got := resolveLinkedPeerID(links, "telegram", "telegram:123")
	if got != "john" {
		t.Errorf("resolveLinkedPeerID with canonical peerID = %q, want %q", got, "john")
	}
}

func TestResolveLinkedPeerID_CanonicalInLinks(t *testing.T) {
	// When identity_links contain canonical IDs and peerID is canonical too
	links := map[string][]string{
		"john": {"telegram:123", "discord:456"},
	}
	got := resolveLinkedPeerID(links, "telegram", "telegram:123")
	if got != "john" {
		t.Errorf("resolveLinkedPeerID canonical in links = %q, want %q", got, "john")
	}
}

func TestResolveLinkedPeerID_BarePeerIDMatchesCanonicalLink(t *testing.T) {
	// When peerID is bare "123" and links have "telegram:123",
	// the scoped candidate "telegram:123" should match.
	links := map[string][]string{
		"john": {"telegram:123"},
	}
	got := resolveLinkedPeerID(links, "telegram", "123")
	if got != "john" {
		t.Errorf("resolveLinkedPeerID bare peer matches canonical link = %q, want %q", got, "john")
	}
}

func TestResolveLinkedPeerID_NoMatch(t *testing.T) {
	links := map[string][]string{
		"john": {"telegram:123"},
	}
	got := resolveLinkedPeerID(links, "discord", "999")
	if got != "" {
		t.Errorf("resolveLinkedPeerID no match = %q, want empty", got)
	}
}

func TestParseAgentSessionKey_Valid(t *testing.T) {
	parsed := ParseAgentSessionKey("agent:sales:telegram:direct:user123")
	if parsed == nil {
		t.Fatal("expected non-nil result")
	}
	if parsed.AgentID != "sales" {
		t.Errorf("AgentID = %q, want 'sales'", parsed.AgentID)
	}
	if parsed.Rest != "telegram:direct:user123" {
		t.Errorf("Rest = %q, want 'telegram:direct:user123'", parsed.Rest)
	}
}

func TestParseAgentSessionKey_Invalid(t *testing.T) {
	tests := []string{
		"",
		"foo:bar",
		"notprefix:sales:main",
		"agent::main",
		"agent:sales:",
	}
	for _, input := range tests {
		if got := ParseAgentSessionKey(input); got != nil {
			t.Errorf("ParseAgentSessionKey(%q) = %+v, want nil", input, got)
		}
	}
}

func TestIsSubagentSessionKey(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"subagent:task-1", true},
		{"agent:main:subagent:task-1", true},
		{"agent:main:main", false},
		{"agent:main:telegram:direct:user123", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := IsSubagentSessionKey(tt.input); got != tt.want {
			t.Errorf("IsSubagentSessionKey(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

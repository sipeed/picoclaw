package events

import "testing"

func TestFilterKindPrefix(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		prefix string
		event  Event
		want   bool
	}{
		{
			name:   "matches agent prefix",
			prefix: "agent.",
			event:  Event{Kind: KindAgentTurnStart},
			want:   true,
		},
		{
			name:   "rejects different prefix",
			prefix: "channel.",
			event:  Event{Kind: KindAgentTurnStart},
			want:   false,
		},
		{
			name:   "empty prefix matches all",
			prefix: "",
			event:  Event{Kind: KindAgentTurnStart},
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := MatchKindPrefix(tt.prefix)(tt.event); got != tt.want {
				t.Fatalf("MatchKindPrefix(%q) = %v, want %v", tt.prefix, got, tt.want)
			}
		})
	}
}

func TestFilterScope(t *testing.T) {
	t.Parallel()

	evt := Event{
		Scope: Scope{
			AgentID:    "agent-a",
			SessionKey: "session-1",
			TurnID:     "turn-1",
			Channel:    "telegram",
			ChatID:     "chat-1",
			MessageID:  "msg-1",
		},
	}

	tests := []struct {
		name  string
		scope ScopeFilter
		want  bool
	}{
		{
			name:  "empty filter matches",
			scope: ScopeFilter{},
			want:  true,
		},
		{
			name: "matches selected fields",
			scope: ScopeFilter{
				AgentID: "agent-a",
				ChatID:  "chat-1",
			},
			want: true,
		},
		{
			name: "rejects mismatched field",
			scope: ScopeFilter{
				AgentID:   "agent-a",
				MessageID: "msg-2",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := MatchScope(tt.scope)(evt); got != tt.want {
				t.Fatalf("MatchScope(%+v) = %v, want %v", tt.scope, got, tt.want)
			}
		})
	}
}

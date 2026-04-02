package agent

import (
	"testing"

	"github.com/sipeed/picoclaw/pkg/providers"
)

func TestResolveSubagentDefaultModel(t *testing.T) {
	t.Run("prefers resolved candidate model", func(t *testing.T) {
		agent := &AgentInstance{
			Model: "openrouter/minimax/minimax-m2.5",
			Candidates: []providers.FallbackCandidate{
				{Provider: "openrouter", Model: "minimax/minimax-m2.5"},
			},
		}

		if got := resolveSubagentDefaultModel(agent); got != "minimax/minimax-m2.5" {
			t.Fatalf("resolveSubagentDefaultModel() = %q, want %q", got, "minimax/minimax-m2.5")
		}
	})

	t.Run("falls back to raw agent model when no candidates exist", func(t *testing.T) {
		agent := &AgentInstance{Model: "claude-sonnet-4.6"}

		if got := resolveSubagentDefaultModel(agent); got != "claude-sonnet-4.6" {
			t.Fatalf("resolveSubagentDefaultModel() = %q, want %q", got, "claude-sonnet-4.6")
		}
	})

	t.Run("returns empty string for nil agent", func(t *testing.T) {
		if got := resolveSubagentDefaultModel(nil); got != "" {
			t.Fatalf("resolveSubagentDefaultModel(nil) = %q, want empty string", got)
		}
	})
}

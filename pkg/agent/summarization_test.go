package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
)

type compactionSummaryMockProvider struct {
	prompts []string
}

func (m *compactionSummaryMockProvider) Chat(
	_ context.Context,
	messages []providers.Message,
	_ []providers.ToolDefinition,
	_ string,
	_ map[string]any,
) (*providers.LLMResponse, error) {
	prompt := messages[0].Content
	m.prompts = append(m.prompts, prompt)

	switch {
	case strings.Contains(prompt, "Update the running conversation summary for future context injection."):
		return &providers.LLMResponse{Content: `## Key Context
- user is working on compaction summaries
## Decisions
- store detailed compaction notes in timestamped files
## Open Loops
- none
## Preferences / Constraints
- prefer separate files over prompt inflation`}, nil
	case strings.Contains(prompt, "Write a detailed compaction memory note for this conversation segment."):
		return &providers.LLMResponse{Content: `## Session
- focus: compaction flow
## What Happened
- reviewed the summary flow and decided to persist a detailed note
## Decisions
- use timestamped compaction files
## Action Items
- none
## Open Questions
- none
## Artifacts Mentioned
- memory/YYYYMM/YYYYMMDD-HHMMSS.compactions.md
## Preferences / Working Style
- keep the context summary short`}, nil
	case strings.Contains(prompt, "Merge the existing running summary and the new partial summaries into one updated running summary."):
		return &providers.LLMResponse{Content: `## Key Context
- merged
## Decisions
- merged
## Open Loops
- none
## Preferences / Constraints
- none`}, nil
	case strings.Contains(prompt, "Merge these detailed compaction notes into one cohesive daily memory entry."):
		return &providers.LLMResponse{Content: `## Session
- merged
## What Happened
- merged
## Decisions
- merged
## Action Items
- none
## Open Questions
- none
## Artifacts Mentioned
- none
## Preferences / Working Style
- none`}, nil
	default:
		return &providers.LLMResponse{Content: "unexpected prompt"}, nil
	}
}

func (m *compactionSummaryMockProvider) GetDefaultModel() string {
	return "mock-model"
}

func TestSummarizeSessionWritesDetailedCompactionFile(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         workspace,
				Model:             "test-model",
				MaxTokens:         4096,
				MaxToolIterations: 10,
			},
		},
	}

	provider := &compactionSummaryMockProvider{}
	al := NewAgentLoop(cfg, bus.NewMessageBus(), provider)
	agent := al.registry.GetDefaultAgent()
	if agent == nil {
		t.Fatal("expected default agent")
	}

	sessionKey := "session-1"
	history := []providers.Message{
		{Role: "user", Content: "Need a better compaction summary flow."},
		{Role: "assistant", Content: "Current prompt is too generic."},
		{Role: "user", Content: "Write the detailed summary into a separate file."},
		{Role: "assistant", Content: "We should keep the running summary short."},
		{Role: "user", Content: "Add timestamps to the compaction filename."},
		{Role: "assistant", Content: "Use hours, minutes, and seconds."},
	}
	for _, msg := range history {
		agent.Sessions.AddFullMessage(sessionKey, msg)
	}

	al.summarizeSession(agent, sessionKey, "telegram", "chat-42")

	summary := agent.Sessions.GetSummary(sessionKey)
	if !strings.Contains(summary, "## Key Context") {
		t.Fatalf("summary missing structured heading: %q", summary)
	}

	finalHistory := agent.Sessions.GetHistory(sessionKey)
	if len(finalHistory) != 4 {
		t.Fatalf("history len = %d, want 4", len(finalHistory))
	}

	files, err := filepath.Glob(
		filepath.Join(workspace, "memory", "*", "*.compactions.md"),
	)
	if err != nil {
		t.Fatalf("Glob failed: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected one compaction file, got %d", len(files))
	}

	data, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "Session: `session-1`") {
		t.Fatalf("compaction file missing session metadata: %q", content)
	}
	if !strings.Contains(content, "Channel: `telegram`") {
		t.Fatalf("compaction file missing channel metadata: %q", content)
	}
	if !strings.Contains(content, "## What Happened") {
		t.Fatalf("compaction file missing detailed note body: %q", content)
	}

	if len(provider.prompts) != 2 {
		t.Fatalf("expected 2 provider calls, got %d", len(provider.prompts))
	}
	if !strings.Contains(provider.prompts[0], "Keep the result under 180 words.") {
		t.Fatalf("running summary prompt missing compactness rule: %q", provider.prompts[0])
	}
	if !strings.Contains(provider.prompts[1], "Target 250-500 words.") {
		t.Fatalf("detailed compaction prompt missing target length: %q", provider.prompts[1])
	}
	if !strings.Contains(provider.prompts[1], "<channel>telegram</channel>") {
		t.Fatalf("detailed compaction prompt missing channel metadata: %q", provider.prompts[1])
	}
}

func TestBuildRunningSummaryMergePromptIncludesConflictRules(t *testing.T) {
	t.Parallel()

	prompt := buildRunningSummaryMergePrompt(
		"## Key Context\n- prior summary",
		[]string{"## Key Context\n- first", "## Key Context\n- second"},
	)

	if !strings.Contains(prompt, "Prefer newer information when facts conflict.") {
		t.Fatalf("prompt missing conflict rule: %q", prompt)
	}
	if !strings.Contains(prompt, "<existing_summary>") {
		t.Fatalf("prompt missing existing summary block: %q", prompt)
	}
	if !strings.Contains(prompt, "<summary index=\"2\">") {
		t.Fatalf("prompt missing indexed summary block: %q", prompt)
	}
}

package evolution_test

import (
	"context"
	"errors"
	"testing"

	"github.com/sipeed/picoclaw/pkg/evolution"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/skills"
)

type recordingDraftGenerator struct {
	draft evolution.SkillDraft
	err   error
	calls int
}

func (g *recordingDraftGenerator) GenerateDraft(
	_ context.Context,
	_ evolution.LearningRecord,
	_ []skills.SkillInfo,
) (evolution.SkillDraft, error) {
	g.calls++
	return g.draft, g.err
}

type llmDraftTestProvider struct {
	response      *providers.LLMResponse
	err           error
	defaultModel  string
	lastModel     string
	lastMessages  []providers.Message
	chatCallCount int
}

func (p *llmDraftTestProvider) Chat(
	_ context.Context,
	messages []providers.Message,
	_ []providers.ToolDefinition,
	model string,
	_ map[string]any,
) (*providers.LLMResponse, error) {
	p.chatCallCount++
	p.lastModel = model
	p.lastMessages = append([]providers.Message(nil), messages...)
	return p.response, p.err
}

func (p *llmDraftTestProvider) GetDefaultModel() string {
	return p.defaultModel
}

func testLearningRule() evolution.LearningRecord {
	return evolution.LearningRecord{
		ID:                "rule-1",
		Summary:           "weather native-name path",
		EventCount:        7,
		SuccessRate:       0.86,
		WinningPath:       []string{"weather", "native-name"},
		MatchedSkillNames: []string{"weather"},
	}
}

func testSkillMatches() []skills.SkillInfo {
	return []skills.SkillInfo{
		{
			Name:        "weather",
			Path:        "/tmp/weather/SKILL.md",
			Source:      "workspace",
			Description: "Find weather details.",
		},
	}
}

func TestLLMDraftGenerator_GenerateDraft_ParsesJSONResponse(t *testing.T) {
	provider := &llmDraftTestProvider{
		defaultModel: "test-model",
		response: &providers.LLMResponse{
			Content: `{"target_skill_name":"weather","draft_type":"shortcut","change_kind":"append","human_summary":"Prefer native-name lookup first","body_or_patch":"## Start Here\nUse native-name first."}`,
		},
	}
	fallback := &recordingDraftGenerator{
		draft: evolution.SkillDraft{TargetSkillName: "fallback"},
	}
	generator := evolution.NewLLMDraftGenerator(provider, "", fallback)

	draft, err := generator.GenerateDraft(context.Background(), testLearningRule(), testSkillMatches())
	if err != nil {
		t.Fatalf("GenerateDraft: %v", err)
	}

	if provider.chatCallCount != 1 {
		t.Fatalf("chatCallCount = %d, want 1", provider.chatCallCount)
	}
	if provider.lastModel != "test-model" {
		t.Fatalf("lastModel = %q, want test-model", provider.lastModel)
	}
	if len(provider.lastMessages) == 0 {
		t.Fatal("expected prompt messages")
	}
	if fallback.calls != 0 {
		t.Fatalf("fallback.calls = %d, want 0", fallback.calls)
	}
	if draft.TargetSkillName != "weather" {
		t.Fatalf("TargetSkillName = %q, want weather", draft.TargetSkillName)
	}
	if draft.DraftType != evolution.DraftTypeShortcut {
		t.Fatalf("DraftType = %q, want %q", draft.DraftType, evolution.DraftTypeShortcut)
	}
	if draft.ChangeKind != evolution.ChangeKindAppend {
		t.Fatalf("ChangeKind = %q, want %q", draft.ChangeKind, evolution.ChangeKindAppend)
	}
	if draft.HumanSummary == "" || draft.BodyOrPatch == "" {
		t.Fatal("expected non-empty draft content")
	}
}

func TestLLMDraftGenerator_GenerateDraft_PrefersExplicitModelIDOverProviderDefault(t *testing.T) {
	provider := &llmDraftTestProvider{
		defaultModel: "provider-default-model",
		response: &providers.LLMResponse{
			Content: `{"target_skill_name":"weather","draft_type":"shortcut","change_kind":"append","human_summary":"Prefer native-name lookup first","body_or_patch":"## Start Here\nUse native-name first."}`,
		},
	}
	generator := evolution.NewLLMDraftGenerator(provider, "explicit-model-id", &recordingDraftGenerator{})

	_, err := generator.GenerateDraft(context.Background(), testLearningRule(), testSkillMatches())
	if err != nil {
		t.Fatalf("GenerateDraft: %v", err)
	}
	if provider.lastModel != "explicit-model-id" {
		t.Fatalf("lastModel = %q, want explicit-model-id", provider.lastModel)
	}
}

func TestLLMDraftGenerator_GenerateDraft_FallsBackOnProviderError(t *testing.T) {
	fallback := &recordingDraftGenerator{
		draft: evolution.SkillDraft{
			TargetSkillName: "weather-fallback",
			DraftType:       evolution.DraftTypeWorkflow,
			ChangeKind:      evolution.ChangeKindCreate,
			HumanSummary:    "fallback summary",
			BodyOrPatch:     "fallback body",
		},
	}
	generator := evolution.NewLLMDraftGenerator(&llmDraftTestProvider{
		defaultModel: "test-model",
		err:          errors.New("provider unavailable"),
	}, "", fallback)

	draft, err := generator.GenerateDraft(context.Background(), testLearningRule(), testSkillMatches())
	if err != nil {
		t.Fatalf("GenerateDraft: %v", err)
	}

	if fallback.calls != 1 {
		t.Fatalf("fallback.calls = %d, want 1", fallback.calls)
	}
	if draft.TargetSkillName != "weather-fallback" {
		t.Fatalf("TargetSkillName = %q, want weather-fallback", draft.TargetSkillName)
	}
}

func TestLLMDraftGenerator_GenerateDraft_FallsBackOnInvalidOrEmptyContent(t *testing.T) {
	testCases := []struct {
		name    string
		content string
	}{
		{name: "invalid json", content: `not-json`},
		{name: "empty content", content: ``},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			fallback := &recordingDraftGenerator{
				draft: evolution.SkillDraft{
					TargetSkillName: "weather-fallback",
					DraftType:       evolution.DraftTypeWorkflow,
					ChangeKind:      evolution.ChangeKindCreate,
					HumanSummary:    "fallback summary",
					BodyOrPatch:     "fallback body",
				},
			}
			generator := evolution.NewLLMDraftGenerator(&llmDraftTestProvider{
				defaultModel: "test-model",
				response:     &providers.LLMResponse{Content: tt.content},
			}, "", fallback)

			draft, err := generator.GenerateDraft(context.Background(), testLearningRule(), testSkillMatches())
			if err != nil {
				t.Fatalf("GenerateDraft: %v", err)
			}

			if fallback.calls != 1 {
				t.Fatalf("fallback.calls = %d, want 1", fallback.calls)
			}
			if draft.TargetSkillName != "weather-fallback" {
				t.Fatalf("TargetSkillName = %q, want weather-fallback", draft.TargetSkillName)
			}
		})
	}
}

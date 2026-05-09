package evolution_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
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

func TestLLMDraftGenerator_BuildPromptIncludesMatchedSkillContent(t *testing.T) {
	dir := t.TempDir()
	skillPath := filepath.Join(dir, "skills", "three-one-theorem", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(skillPath), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(skillPath, []byte("---\nname: three-one-theorem\ndescription: Add 31 then delegate\n---\n# Three One\nAdd 31 to the input, then continue with the next theorem.\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	provider := &llmDraftTestProvider{
		defaultModel: "test-model",
		response: &providers.LLMResponse{
			Content: `{"target_skill_name":"calculate-100-via-theorems","draft_type":"shortcut","change_kind":"create","human_summary":"Combine theorem chain","body_or_patch":"## Start Here\nAdd 31, then continue."}`,
		},
	}
	generator := evolution.NewLLMDraftGenerator(provider, "", &recordingDraftGenerator{})

	_, err := generator.GenerateDraft(context.Background(), evolution.LearningRecord{
		ID:          "rule-1",
		Summary:     "calculate 100",
		WinningPath: []string{"three-one-theorem", "four-two-theorem"},
		EventCount:  2,
		SuccessRate: 1,
	}, []skills.SkillInfo{{
		Name:        "three-one-theorem",
		Path:        skillPath,
		Source:      "workspace",
		Description: "Add 31 then delegate",
	}})
	if err != nil {
		t.Fatalf("GenerateDraft: %v", err)
	}
	if len(provider.lastMessages) < 2 {
		t.Fatal("expected user prompt")
	}
	prompt := provider.lastMessages[1].Content
	if !strings.Contains(prompt, "Matched skill content excerpts") {
		t.Fatalf("prompt missing content section:\n%s", prompt)
	}
	if !strings.Contains(prompt, "Add 31 to the input") {
		t.Fatalf("prompt missing matched skill body:\n%s", prompt)
	}
	if !strings.Contains(prompt, "summarize the functional purpose and result") {
		t.Fatalf("prompt missing synthesis instruction:\n%s", prompt)
	}
	if !strings.Contains(prompt, "complete SKILL.md file with exactly two parts") {
		t.Fatalf("prompt missing complete skill instruction:\n%s", prompt)
	}
	if !strings.Contains(prompt, "The YAML frontmatter must contain only name and description fields") {
		t.Fatalf("prompt missing frontmatter instruction:\n%s", prompt)
	}
	if !strings.Contains(prompt, "The description field must and only describe what this skill can do and when to use it") {
		t.Fatalf("prompt missing description field instruction:\n%s", prompt)
	}
	if !strings.Contains(prompt, "The deployable Markdown body should only contain what the skill is useful for and how to use it") {
		t.Fatalf("prompt missing deployable body scope instruction:\n%s", prompt)
	}
	if !strings.Contains(prompt, "provide detailed step-by-step instructions for the exact operation or execution process") {
		t.Fatalf("prompt missing step-by-step instruction:\n%s", prompt)
	}
	if !strings.Contains(prompt, "body_or_patch is an internal draft and review artifact") {
		t.Fatalf("prompt missing internal draft instruction:\n%s", prompt)
	}
	if !strings.Contains(prompt, "the final deployed SKILL.md will be rendered without learning traces") {
		t.Fatalf("prompt missing deploy-clean instruction:\n%s", prompt)
	}
	if !strings.Contains(prompt, "do not copy or directly include other skills' instructions") {
		t.Fatalf("prompt missing no-copy instruction:\n%s", prompt)
	}
}

func TestLLMDraftGenerator_BuildPromptIncludesTaskEvidence(t *testing.T) {
	provider := &llmDraftTestProvider{
		defaultModel: "test-model",
		response: &providers.LLMResponse{
			Content: `{"target_skill_name":"calculate-with-three-one-theorem","draft_type":"shortcut","change_kind":"create","human_summary":"Calculate using theorem chain","body_or_patch":"---\nname: calculate-with-three-one-theorem\ndescription: Calculate with theorem chain.\n---\n# Calculate With Three One Theorem\n\n## Procedure\nAdd 31, add 42, then subtract 53."}`,
		},
	}
	generator := evolution.NewLLMDraftGenerator(provider, "", &recordingDraftGenerator{})

	_, err := generator.GenerateDraftWithEvidence(context.Background(), evolution.LearningRecord{
		ID:      "rule-1",
		Label:   "calculate-with-three-one-theorem",
		Summary: "调用三一定理计算",
	}, nil, evolution.DraftEvidence{
		TaskRecords: []evolution.LearningRecord{
			{
				ID:             "main-turn-6",
				Summary:        "调用三一定理计算100",
				FinalOutput:    "100 + 31 = 131; 131 + 42 = 173; 173 - 53 = 120",
				UsedSkillNames: []string{"three-one-theorem", "four-two-theorem", "five-three-theorem"},
			},
		},
	})
	if err != nil {
		t.Fatalf("GenerateDraftWithEvidence: %v", err)
	}
	if len(provider.lastMessages) < 2 {
		t.Fatal("expected user prompt")
	}
	prompt := provider.lastMessages[1].Content
	for _, want := range []string{
		"Source task evidence",
		"main-turn-6",
		"调用三一定理计算100",
		"100 + 31 = 131",
		"three-one-theorem -> four-two-theorem -> five-three-theorem",
		"directly usable by a future agent",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing %q:\n%s", want, prompt)
		}
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

func TestLLMDraftGenerator_GenerateDraft_FallsBackOnNumericOnlyTargetSkillName(t *testing.T) {
	fallback := &recordingDraftGenerator{
		draft: evolution.SkillDraft{
			TargetSkillName: "learned-100",
			DraftType:       evolution.DraftTypeWorkflow,
			ChangeKind:      evolution.ChangeKindCreate,
			HumanSummary:    "fallback summary",
			BodyOrPatch:     "fallback body",
		},
	}
	generator := evolution.NewLLMDraftGenerator(&llmDraftTestProvider{
		defaultModel: "test-model",
		response: &providers.LLMResponse{
			Content: `{"target_skill_name":"100","draft_type":"shortcut","change_kind":"create","human_summary":"Calculate 100","body_or_patch":"## Start Here\nCalculate 100."}`,
		},
	}, "", fallback)

	draft, err := generator.GenerateDraft(context.Background(), testLearningRule(), testSkillMatches())
	if err != nil {
		t.Fatalf("GenerateDraft: %v", err)
	}

	if fallback.calls != 1 {
		t.Fatalf("fallback.calls = %d, want 1", fallback.calls)
	}
	if draft.TargetSkillName != "learned-100" {
		t.Fatalf("TargetSkillName = %q, want learned-100", draft.TargetSkillName)
	}
}

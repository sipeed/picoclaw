package evolution_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/evolution"
	"github.com/sipeed/picoclaw/pkg/providers"
)

type llmClusterTestProvider struct {
	content      string
	defaultModel string
	messages     []providers.Message
}

func (p *llmClusterTestProvider) Chat(
	_ context.Context,
	messages []providers.Message,
	_ []providers.ToolDefinition,
	_ string,
	_ map[string]any,
) (*providers.LLMResponse, error) {
	p.messages = append([]providers.Message(nil), messages...)
	return &providers.LLMResponse{Content: p.content}, nil
}

func (p *llmClusterTestProvider) GetDefaultModel() string {
	return p.defaultModel
}

func TestHeuristicPatternClusterer_GroupsChineseSummariesWithoutLLM(t *testing.T) {
	clusterer := evolution.NewHeuristicPatternClusterer(3, func() time.Time {
		return time.Unix(1700000000, 0).UTC()
	})
	success := true
	tasks := []evolution.LearningRecord{
		{
			ID:          "task-1",
			Kind:        evolution.RecordKindTask,
			WorkspaceID: "workspace",
			Summary:     "调用三一定理计算100",
			FinalOutput: "100 + 31 = 131; 131 + 42 = 173; 173 - 53 = 120",
			Status:      evolution.RecordStatus("new"),
			Success:     &success,
		},
		{
			ID:          "task-2",
			Kind:        evolution.RecordKindTask,
			WorkspaceID: "workspace",
			Summary:     "调用三一定理计算200",
			FinalOutput: "200 + 31 = 231; 231 + 42 = 273; 273 - 53 = 220",
			Status:      evolution.RecordStatus("new"),
			Success:     &success,
		},
		{
			ID:             "task-3",
			Kind:           evolution.RecordKindTask,
			WorkspaceID:    "workspace",
			Summary:        "调用三一定理计算300",
			FinalOutput:    "300 + 31 = 331; 331 + 42 = 373; 373 - 53 = 320",
			Status:         evolution.RecordStatus("new"),
			Success:        &success,
			UsedSkillNames: []string{"three-one-theorem", "four-two-theorem", "five-three-theorem"},
		},
	}

	patterns, clusteredIDs, err := clusterer.BuildPatterns(context.Background(), "workspace", tasks, nil)
	if err != nil {
		t.Fatalf("BuildPatterns: %v", err)
	}
	if len(patterns) != 1 {
		t.Fatalf("len(patterns) = %d, want 1: %#v", len(patterns), patterns)
	}
	if !strings.HasPrefix(patterns[0].Label, "task-") {
		t.Fatalf("Label = %q, want task-* fallback label", patterns[0].Label)
	}
	if patterns[0].Summary != "调用三一定理计算100" {
		t.Fatalf("Summary = %q, want representative Chinese summary", patterns[0].Summary)
	}
	if len(patterns[0].TaskRecordIDs) != 3 {
		t.Fatalf("TaskRecordIDs = %v, want 3 ids", patterns[0].TaskRecordIDs)
	}
	if len(clusteredIDs) != 3 {
		t.Fatalf("clusteredIDs = %v, want 3 ids", clusteredIDs)
	}
}

func TestLLMPatternClusterer_FallsBackWhenLLMReturnsNoUsableClusters(t *testing.T) {
	fallback := evolution.NewHeuristicPatternClusterer(2, func() time.Time {
		return time.Unix(1700000000, 0).UTC()
	})
	clusterer := evolution.NewLLMPatternClusterer(
		&llmClusterTestProvider{content: `{"clusters":[]}`, defaultModel: "test-model"},
		"test-model",
		fallback,
		2,
		func() time.Time { return time.Unix(1700000000, 0).UTC() },
	)
	success := true
	tasks := []evolution.LearningRecord{
		{
			ID:          "task-1",
			Kind:        evolution.RecordKindTask,
			WorkspaceID: "workspace",
			Summary:     "调用三一定理计算100",
			FinalOutput: "100 + 31 = 131",
			Status:      evolution.RecordStatus("new"),
			Success:     &success,
		},
		{
			ID:          "task-2",
			Kind:        evolution.RecordKindTask,
			WorkspaceID: "workspace",
			Summary:     "调用三一定理计算200",
			FinalOutput: "200 + 31 = 231",
			Status:      evolution.RecordStatus("new"),
			Success:     &success,
		},
	}

	patterns, clusteredIDs, err := clusterer.BuildPatterns(context.Background(), "workspace", tasks, nil)
	if err != nil {
		t.Fatalf("BuildPatterns: %v", err)
	}
	if len(patterns) != 1 {
		t.Fatalf("len(patterns) = %d, want fallback pattern: %#v", len(patterns), patterns)
	}
	if len(clusteredIDs) != 2 {
		t.Fatalf("clusteredIDs = %v, want 2 task IDs", clusteredIDs)
	}
}

func TestLLMPatternClusterer_PromptFiltersExistingPatternsByWorkspace(t *testing.T) {
	provider := &llmClusterTestProvider{
		content:      `{"clusters":[{"label":"current-weather-path","summary":"current summary","task_record_ids":["task-1"],"cluster_reason":"same goal"}]}`,
		defaultModel: "test-model",
	}
	clusterer := evolution.NewLLMPatternClusterer(
		provider,
		"test-model",
		evolution.NewHeuristicPatternClusterer(1, nil),
		1,
		func() time.Time { return time.Unix(1700000000, 0).UTC() },
	)
	success := true
	tasks := []evolution.LearningRecord{
		{
			ID:          "task-1",
			Kind:        evolution.RecordKindTask,
			WorkspaceID: "workspace-a",
			Summary:     "weather lookup",
			FinalOutput: "sunny",
			Status:      evolution.RecordStatus("new"),
			Success:     &success,
		},
	}
	existing := []evolution.LearningRecord{
		{
			ID:          "rule-a",
			Kind:        evolution.RecordKindPattern,
			WorkspaceID: "workspace-a",
			Label:       "current-weather-path",
			Summary:     "current workspace pattern",
		},
		{
			ID:          "rule-b",
			Kind:        evolution.RecordKindPattern,
			WorkspaceID: "workspace-b",
			Label:       "other-workspace-secret-path",
			Summary:     "other workspace pattern",
		},
	}

	if _, _, err := clusterer.BuildPatterns(context.Background(), "workspace-a", tasks, existing); err != nil {
		t.Fatalf("BuildPatterns: %v", err)
	}
	if len(provider.messages) != 2 {
		t.Fatalf("len(messages) = %d, want 2", len(provider.messages))
	}
	prompt := provider.messages[1].Content
	if !strings.Contains(prompt, "current-weather-path") {
		t.Fatalf("prompt = %q, want current workspace pattern", prompt)
	}
	if strings.Contains(prompt, "other-workspace-secret-path") || strings.Contains(prompt, "other workspace pattern") {
		t.Fatalf("prompt leaked other workspace pattern: %s", prompt)
	}
}

func TestLLMPatternClusterer_RejectsClusterBelowEvidenceSuccessRatio(t *testing.T) {
	provider := &llmClusterTestProvider{
		content:      `{"clusters":[{"label":"weather-lookup","summary":"lookup weather","task_record_ids":["task-success","task-failed"],"cluster_reason":"same weather lookup goal"}]}`,
		defaultModel: "test-model",
	}
	clusterer := evolution.NewLLMPatternClusterer(
		provider,
		"test-model",
		evolution.NewHeuristicPatternClusterer(1, nil),
		1,
		func() time.Time { return time.Unix(1700000000, 0).UTC() },
	)
	success := true
	failed := false
	successfulTasks := []evolution.LearningRecord{
		{
			ID:          "task-success",
			Kind:        evolution.RecordKindTask,
			WorkspaceID: "workspace-a",
			Summary:     "weather lookup shanghai",
			FinalOutput: "sunny",
			Status:      evolution.RecordStatus("new"),
			Success:     &success,
		},
	}
	evidenceTasks := []evolution.LearningRecord{
		successfulTasks[0],
		{
			ID:          "task-failed",
			Kind:        evolution.RecordKindTask,
			WorkspaceID: "workspace-a",
			Summary:     "forecast for shanghai",
			FinalOutput: "could not complete",
			Status:      evolution.RecordStatus("new"),
			Success:     &failed,
		},
	}

	patterns, clusteredIDs, err := clusterer.BuildPatternsWithEvidence(
		context.Background(),
		"workspace-a",
		successfulTasks,
		evidenceTasks,
		nil,
		0.8,
	)
	if err != nil {
		t.Fatalf("BuildPatternsWithEvidence: %v", err)
	}
	if len(patterns) != 0 {
		t.Fatalf("len(patterns) = %d, want 0: %#v", len(patterns), patterns)
	}
	if len(clusteredIDs) != 0 {
		t.Fatalf("clusteredIDs = %v, want none", clusteredIDs)
	}
	prompt := provider.messages[1].Content
	if !strings.Contains(prompt, `"success": true`) || !strings.Contains(prompt, `"success": false`) {
		t.Fatalf("prompt should include success and failure evidence:\n%s", prompt)
	}
}

func TestLLMPatternClusterer_RejectsIncompleteEvidenceAssignment(t *testing.T) {
	provider := &llmClusterTestProvider{
		content:      `{"clusters":[{"label":"weather-lookup","summary":"lookup weather","task_record_ids":["task-success"],"cluster_reason":"same weather lookup goal"}]}`,
		defaultModel: "test-model",
	}
	clusterer := evolution.NewLLMPatternClusterer(
		provider,
		"test-model",
		evolution.NewHeuristicPatternClusterer(1, nil),
		1,
		func() time.Time { return time.Unix(1700000000, 0).UTC() },
	)
	success := true
	failed := false
	successfulTasks := []evolution.LearningRecord{
		{
			ID:          "task-success",
			Kind:        evolution.RecordKindTask,
			WorkspaceID: "workspace-a",
			Summary:     "weather lookup shanghai",
			FinalOutput: "sunny",
			Status:      evolution.RecordStatus("new"),
			Success:     &success,
		},
	}
	evidenceTasks := []evolution.LearningRecord{
		successfulTasks[0],
		{
			ID:          "task-failed",
			Kind:        evolution.RecordKindTask,
			WorkspaceID: "workspace-a",
			Summary:     "forecast for shanghai",
			FinalOutput: "could not complete",
			Status:      evolution.RecordStatus("new"),
			Success:     &failed,
		},
	}

	patterns, clusteredIDs, err := clusterer.BuildPatternsWithEvidence(
		context.Background(),
		"workspace-a",
		successfulTasks,
		evidenceTasks,
		nil,
		0.8,
	)
	if err != nil {
		t.Fatalf("BuildPatternsWithEvidence: %v", err)
	}
	if len(patterns) != 0 {
		t.Fatalf("len(patterns) = %d, want 0: %#v", len(patterns), patterns)
	}
	if len(clusteredIDs) != 0 {
		t.Fatalf("clusteredIDs = %v, want none", clusteredIDs)
	}
}

func TestLLMPatternClusterer_MarksAllAcceptedEvidenceClusteredButStoresSuccessfulTaskIDs(t *testing.T) {
	provider := &llmClusterTestProvider{
		content:      `{"clusters":[{"label":"weather-lookup","summary":"lookup weather","task_record_ids":["task-success","task-failed"],"cluster_reason":"same weather lookup goal"}]}`,
		defaultModel: "test-model",
	}
	clusterer := evolution.NewLLMPatternClusterer(
		provider,
		"test-model",
		evolution.NewHeuristicPatternClusterer(1, nil),
		1,
		func() time.Time { return time.Unix(1700000000, 0).UTC() },
	)
	success := true
	failed := false
	successfulTasks := []evolution.LearningRecord{
		{
			ID:          "task-success",
			Kind:        evolution.RecordKindTask,
			WorkspaceID: "workspace-a",
			Summary:     "weather lookup shanghai",
			FinalOutput: "sunny",
			Status:      evolution.RecordStatus("new"),
			Success:     &success,
		},
	}
	evidenceTasks := []evolution.LearningRecord{
		successfulTasks[0],
		{
			ID:          "task-failed",
			Kind:        evolution.RecordKindTask,
			WorkspaceID: "workspace-a",
			Summary:     "forecast for shanghai",
			FinalOutput: "could not complete",
			Status:      evolution.RecordStatus("new"),
			Success:     &failed,
		},
	}

	patterns, clusteredIDs, err := clusterer.BuildPatternsWithEvidence(
		context.Background(),
		"workspace-a",
		successfulTasks,
		evidenceTasks,
		nil,
		0.5,
	)
	if err != nil {
		t.Fatalf("BuildPatternsWithEvidence: %v", err)
	}
	if len(patterns) != 1 {
		t.Fatalf("len(patterns) = %d, want 1: %#v", len(patterns), patterns)
	}
	if got := strings.Join(patterns[0].TaskRecordIDs, ","); got != "task-success" {
		t.Fatalf("pattern TaskRecordIDs = %v, want only successful task", patterns[0].TaskRecordIDs)
	}
	if got := strings.Join(clusteredIDs, ","); got != "task-success,task-failed" {
		t.Fatalf("clusteredIDs = %v, want all accepted evidence IDs", clusteredIDs)
	}
}

func TestLLMPatternClusterer_FallbackMarksAllAcceptedEvidenceClustered(t *testing.T) {
	provider := &llmClusterTestProvider{
		content:      `not-json`,
		defaultModel: "test-model",
	}
	clusterer := evolution.NewLLMPatternClusterer(
		provider,
		"test-model",
		evolution.NewHeuristicPatternClusterer(1, nil),
		1,
		func() time.Time { return time.Unix(1700000000, 0).UTC() },
	)
	success := true
	failed := false
	successfulTasks := []evolution.LearningRecord{
		{
			ID:          "task-success",
			Kind:        evolution.RecordKindTask,
			WorkspaceID: "workspace-a",
			Summary:     "weather lookup 100",
			FinalOutput: "sunny",
			Status:      evolution.RecordStatus("new"),
			Success:     &success,
		},
	}
	evidenceTasks := []evolution.LearningRecord{
		successfulTasks[0],
		{
			ID:          "task-failed",
			Kind:        evolution.RecordKindTask,
			WorkspaceID: "workspace-a",
			Summary:     "weather lookup 200",
			FinalOutput: "partial result",
			Status:      evolution.RecordStatus("new"),
			Success:     &failed,
		},
	}

	patterns, clusteredIDs, err := clusterer.BuildPatternsWithEvidence(
		context.Background(),
		"workspace-a",
		successfulTasks,
		evidenceTasks,
		nil,
		0.5,
	)
	if err != nil {
		t.Fatalf("BuildPatternsWithEvidence: %v", err)
	}
	if len(patterns) != 1 {
		t.Fatalf("len(patterns) = %d, want fallback pattern: %#v", len(patterns), patterns)
	}
	if got := strings.Join(patterns[0].TaskRecordIDs, ","); got != "task-success" {
		t.Fatalf("pattern TaskRecordIDs = %v, want only successful task", patterns[0].TaskRecordIDs)
	}
	if got := strings.Join(clusteredIDs, ","); got != "task-success,task-failed" {
		t.Fatalf("clusteredIDs = %v, want all accepted evidence IDs", clusteredIDs)
	}
}

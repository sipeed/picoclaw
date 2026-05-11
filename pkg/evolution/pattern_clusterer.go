package evolution

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/sipeed/picoclaw/pkg/providers"
)

type PatternClusterer interface {
	BuildPatterns(
		ctx context.Context,
		workspace string,
		tasks []LearningRecord,
		existing []LearningRecord,
	) ([]LearningRecord, []string, error)
}

type evidencePatternClusterer interface {
	BuildPatternsWithEvidence(
		ctx context.Context,
		workspace string,
		successfulTasks []LearningRecord,
		evidenceTasks []LearningRecord,
		existing []LearningRecord,
		minSuccessRatio float64,
	) ([]LearningRecord, []string, error)
}

type HeuristicPatternClusterer struct {
	minCaseCount int
	now          func() time.Time
}

func NewHeuristicPatternClusterer(minCaseCount int, now func() time.Time) *HeuristicPatternClusterer {
	if minCaseCount <= 0 {
		minCaseCount = 3
	}
	if now == nil {
		now = time.Now
	}
	return &HeuristicPatternClusterer{minCaseCount: minCaseCount, now: now}
}

func (c *HeuristicPatternClusterer) BuildPatterns(
	_ context.Context,
	workspace string,
	tasks []LearningRecord,
	existing []LearningRecord,
) ([]LearningRecord, []string, error) {
	groups := make(map[string][]LearningRecord)
	keys := make([]string, 0)
	for _, task := range tasks {
		if task.WorkspaceID != workspace {
			continue
		}
		key := heuristicClusterKey(task)
		if key == "" {
			continue
		}
		if _, ok := groups[key]; !ok {
			keys = append(keys, key)
		}
		groups[key] = append(groups[key], task)
	}
	sort.Strings(keys)

	existingByLabel := patternsByLabel(existing, workspace)
	patterns := make([]LearningRecord, 0, len(keys))
	clusteredIDs := make([]string, 0)
	for _, key := range keys {
		cluster := groups[key]
		label := heuristicClusterLabelForGroup(key, cluster)
		if label == "" {
			continue
		}
		existingPattern, hasExisting := existingByLabel[label]
		if !hasExisting && len(cluster) < c.minCaseCount {
			continue
		}
		pattern := buildPatternFromCluster(
			workspace,
			label,
			heuristicClusterSummary(label, cluster),
			"heuristic cluster by normalized task summary",
			cluster,
			existingPattern,
			c.now(),
		)
		patterns = append(patterns, pattern)
		clusteredIDs = append(clusteredIDs, collectRecordIDs(cluster)...)
	}
	return patterns, clusteredIDs, nil
}

type LLMPatternClusterer struct {
	provider providers.LLMProvider
	model    string
	fallback PatternClusterer
	minCount int
	now      func() time.Time
}

type llmClusterResponse struct {
	Clusters []llmCluster `json:"clusters"`
}

type llmCluster struct {
	Label         string   `json:"label"`
	Summary       string   `json:"summary"`
	TaskRecordIDs []string `json:"task_record_ids"`
	Reason        string   `json:"cluster_reason"`
}

func NewLLMPatternClusterer(
	provider providers.LLMProvider,
	model string,
	fallback PatternClusterer,
	minCount int,
	now func() time.Time,
) *LLMPatternClusterer {
	if fallback == nil {
		fallback = NewHeuristicPatternClusterer(minCount, now)
	}
	if minCount <= 0 {
		minCount = 3
	}
	if now == nil {
		now = time.Now
	}
	return &LLMPatternClusterer{
		provider: provider,
		model:    strings.TrimSpace(model),
		fallback: fallback,
		minCount: minCount,
		now:      now,
	}
}

func (c *LLMPatternClusterer) BuildPatterns(
	ctx context.Context,
	workspace string,
	tasks []LearningRecord,
	existing []LearningRecord,
) ([]LearningRecord, []string, error) {
	if c == nil {
		return NewHeuristicPatternClusterer(0, nil).BuildPatterns(ctx, workspace, tasks, existing)
	}
	fallback := c.fallback
	if fallback == nil {
		fallback = NewHeuristicPatternClusterer(c.minCount, c.now)
	}
	if c.provider == nil {
		return fallback.BuildPatterns(ctx, workspace, tasks, existing)
	}
	model := strings.TrimSpace(c.model)
	if model == "" {
		model = strings.TrimSpace(c.provider.GetDefaultModel())
	}
	if model == "" {
		return fallback.BuildPatterns(ctx, workspace, tasks, existing)
	}

	callCtx, cancel := withLLMCallTimeout(ctx, llmPatternClusterTimeout)
	defer cancel()
	resp, err := c.provider.Chat(callCtx, []providers.Message{
		{
			Role:    "system",
			Content: "Cluster agent task records by task meaning. Return exactly one JSON object with clusters:[{label,summary,task_record_ids,cluster_reason}]. No markdown fences.",
		},
		{
			Role:    "user",
			Content: buildPatternClusterPrompt(workspace, tasks, existing),
		},
	}, nil, model, map[string]any{"temperature": 0})
	if err != nil || resp == nil || strings.TrimSpace(resp.Content) == "" {
		return fallback.BuildPatterns(ctx, workspace, tasks, existing)
	}

	payload, ok := parseLLMClusterResponse(resp.Content)
	if !ok {
		return fallback.BuildPatterns(ctx, workspace, tasks, existing)
	}
	patterns, clusteredIDs := c.validateAndBuildPatterns(workspace, payload.Clusters, tasks, existing)
	if len(patterns) == 0 {
		return fallback.BuildPatterns(ctx, workspace, tasks, existing)
	}
	return patterns, clusteredIDs, nil
}

func (c *LLMPatternClusterer) BuildPatternsWithEvidence(
	ctx context.Context,
	workspace string,
	successfulTasks []LearningRecord,
	evidenceTasks []LearningRecord,
	existing []LearningRecord,
	minSuccessRatio float64,
) ([]LearningRecord, []string, error) {
	if c == nil {
		return NewHeuristicPatternClusterer(0, nil).BuildPatterns(ctx, workspace, successfulTasks, existing)
	}
	fallback := c.fallback
	if fallback == nil {
		fallback = NewHeuristicPatternClusterer(c.minCount, c.now)
	}
	if c.provider == nil {
		return buildFallbackPatternsWithEvidence(
			ctx,
			fallback,
			workspace,
			successfulTasks,
			evidenceTasks,
			existing,
			minSuccessRatio,
		)
	}
	model := strings.TrimSpace(c.model)
	if model == "" {
		model = strings.TrimSpace(c.provider.GetDefaultModel())
	}
	if model == "" {
		return buildFallbackPatternsWithEvidence(
			ctx,
			fallback,
			workspace,
			successfulTasks,
			evidenceTasks,
			existing,
			minSuccessRatio,
		)
	}
	if len(evidenceTasks) == 0 {
		evidenceTasks = successfulTasks
	}

	callCtx, cancel := withLLMCallTimeout(ctx, llmPatternClusterTimeout)
	defer cancel()
	resp, err := c.provider.Chat(callCtx, []providers.Message{
		{
			Role:    "system",
			Content: "Cluster agent task records by task meaning. Include successful and failed task IDs in the same cluster when they share the same reusable meaning. Return exactly one JSON object with clusters:[{label,summary,task_record_ids,cluster_reason}]. No markdown fences.",
		},
		{
			Role:    "user",
			Content: buildPatternClusterPrompt(workspace, evidenceTasks, existing),
		},
	}, nil, model, map[string]any{"temperature": 0})
	if err != nil || resp == nil || strings.TrimSpace(resp.Content) == "" {
		return buildFallbackPatternsWithEvidence(
			ctx,
			fallback,
			workspace,
			successfulTasks,
			evidenceTasks,
			existing,
			minSuccessRatio,
		)
	}

	payload, ok := parseLLMClusterResponse(resp.Content)
	if !ok {
		return buildFallbackPatternsWithEvidence(
			ctx,
			fallback,
			workspace,
			successfulTasks,
			evidenceTasks,
			existing,
			minSuccessRatio,
		)
	}
	if len(payload.Clusters) == 0 {
		return buildFallbackPatternsWithEvidence(
			ctx,
			fallback,
			workspace,
			successfulTasks,
			evidenceTasks,
			existing,
			minSuccessRatio,
		)
	}
	patterns, clusteredIDs := c.validateAndBuildPatternsWithEvidence(
		workspace,
		payload.Clusters,
		successfulTasks,
		evidenceTasks,
		existing,
		minSuccessRatio,
	)
	return patterns, clusteredIDs, nil
}

func buildFallbackPatternsWithEvidence(
	ctx context.Context,
	fallback PatternClusterer,
	workspace string,
	successfulTasks []LearningRecord,
	evidenceTasks []LearningRecord,
	existing []LearningRecord,
	minSuccessRatio float64,
) ([]LearningRecord, []string, error) {
	if fallback == nil {
		fallback = NewHeuristicPatternClusterer(0, nil)
	}
	patterns, _, err := fallback.BuildPatterns(ctx, workspace, successfulTasks, existing)
	if err != nil || len(patterns) == 0 {
		return patterns, nil, err
	}
	if len(evidenceTasks) == 0 {
		evidenceTasks = successfulTasks
	}

	successByID := make(map[string]LearningRecord, len(successfulTasks))
	for _, task := range successfulTasks {
		successByID[task.ID] = task
	}
	evidenceByKey := make(map[string][]LearningRecord)
	for _, task := range evidenceTasks {
		if task.WorkspaceID != workspace {
			continue
		}
		key := heuristicClusterKey(task)
		if key == "" {
			continue
		}
		evidenceByKey[key] = append(evidenceByKey[key], task)
	}

	filteredPatterns := make([]LearningRecord, 0, len(patterns))
	clusteredIDs := make([]string, 0)
	for _, pattern := range patterns {
		keys := make(map[string]struct{})
		for _, id := range pattern.TaskRecordIDs {
			task, ok := successByID[id]
			if !ok {
				continue
			}
			key := heuristicClusterKey(task)
			if key == "" {
				continue
			}
			keys[key] = struct{}{}
		}

		clusterEvidenceByID := make(map[string]LearningRecord)
		for key := range keys {
			for _, task := range evidenceByKey[key] {
				clusterEvidenceByID[task.ID] = task
			}
		}
		if len(clusterEvidenceByID) == 0 {
			for _, id := range pattern.TaskRecordIDs {
				if task, ok := successByID[id]; ok {
					clusterEvidenceByID[task.ID] = task
				}
			}
		}
		if len(clusterEvidenceByID) == 0 {
			continue
		}

		successes := 0
		clusterEvidence := make([]LearningRecord, 0, len(clusterEvidenceByID))
		for _, task := range clusterEvidenceByID {
			clusterEvidence = append(clusterEvidence, task)
			if task.Success != nil && *task.Success {
				successes++
			}
		}
		sort.Slice(clusterEvidence, func(i, j int) bool {
			leftSuccess := clusterEvidence[i].Success != nil && *clusterEvidence[i].Success
			rightSuccess := clusterEvidence[j].Success != nil && *clusterEvidence[j].Success
			if leftSuccess != rightSuccess {
				return leftSuccess
			}
			return clusterEvidence[i].ID < clusterEvidence[j].ID
		})
		if successes == 0 {
			continue
		}
		if minSuccessRatio > 0 {
			ratio := float64(successes) / float64(len(clusterEvidence))
			if ratio < minSuccessRatio {
				continue
			}
		}

		filteredPatterns = append(filteredPatterns, pattern)
		clusteredIDs = append(clusteredIDs, collectRecordIDs(clusterEvidence)...)
	}
	return filteredPatterns, appendUniqueStrings(nil, clusteredIDs...), nil
}

func (c *LLMPatternClusterer) validateAndBuildPatterns(
	workspace string,
	clusters []llmCluster,
	tasks []LearningRecord,
	existing []LearningRecord,
) ([]LearningRecord, []string) {
	taskByID := make(map[string]LearningRecord, len(tasks))
	for _, task := range tasks {
		taskByID[task.ID] = task
	}
	existingByLabel := patternsByLabel(existing, workspace)
	assigned := make(map[string]struct{}, len(tasks))
	patterns := make([]LearningRecord, 0, len(clusters))
	clusteredIDs := make([]string, 0)

	for _, cluster := range clusters {
		label := validSkillNameOrEmpty(cluster.Label)
		if label == "" {
			continue
		}
		clusterTasks := make([]LearningRecord, 0, len(cluster.TaskRecordIDs))
		for _, id := range cluster.TaskRecordIDs {
			id = strings.TrimSpace(id)
			if id == "" {
				continue
			}
			if _, exists := assigned[id]; exists {
				continue
			}
			task, ok := taskByID[id]
			if !ok {
				continue
			}
			clusterTasks = append(clusterTasks, task)
			assigned[id] = struct{}{}
		}
		existingPattern, hasExisting := existingByLabel[label]
		if !hasExisting && len(clusterTasks) < c.minCount {
			continue
		}
		if len(clusterTasks) == 0 {
			continue
		}
		pattern := buildPatternFromCluster(
			workspace,
			label,
			cluster.Summary,
			cluster.Reason,
			clusterTasks,
			existingPattern,
			c.now(),
		)
		patterns = append(patterns, pattern)
		clusteredIDs = append(clusteredIDs, collectRecordIDs(clusterTasks)...)
	}
	return patterns, clusteredIDs
}

func (c *LLMPatternClusterer) validateAndBuildPatternsWithEvidence(
	workspace string,
	clusters []llmCluster,
	successfulTasks []LearningRecord,
	evidenceTasks []LearningRecord,
	existing []LearningRecord,
	minSuccessRatio float64,
) ([]LearningRecord, []string) {
	evidenceByID := make(map[string]LearningRecord, len(evidenceTasks))
	for _, task := range evidenceTasks {
		evidenceByID[task.ID] = task
	}
	successfulByID := make(map[string]LearningRecord, len(successfulTasks))
	for _, task := range successfulTasks {
		successfulByID[task.ID] = task
	}
	existingByLabel := patternsByLabel(existing, workspace)
	assigned := make(map[string]struct{}, len(evidenceTasks))
	patterns := make([]LearningRecord, 0, len(clusters))
	clusteredIDs := make([]string, 0)

	for _, cluster := range clusters {
		label := validSkillNameOrEmpty(cluster.Label)
		if label == "" {
			continue
		}
		clusterEvidence := make([]LearningRecord, 0, len(cluster.TaskRecordIDs))
		clusterSuccesses := make([]LearningRecord, 0, len(cluster.TaskRecordIDs))
		for _, id := range cluster.TaskRecordIDs {
			id = strings.TrimSpace(id)
			if id == "" {
				continue
			}
			if _, exists := assigned[id]; exists {
				continue
			}
			task, ok := evidenceByID[id]
			if !ok {
				continue
			}
			clusterEvidence = append(clusterEvidence, task)
			if successTask, ok := successfulByID[id]; ok {
				clusterSuccesses = append(clusterSuccesses, successTask)
			}
			assigned[id] = struct{}{}
		}
		if len(clusterEvidence) == 0 || len(clusterSuccesses) == 0 {
			continue
		}
		if minSuccessRatio > 0 {
			ratio := float64(len(clusterSuccesses)) / float64(len(clusterEvidence))
			if ratio < minSuccessRatio {
				continue
			}
		}
		existingPattern, hasExisting := existingByLabel[label]
		if !hasExisting && len(clusterSuccesses) < c.minCount {
			continue
		}
		pattern := buildPatternFromCluster(
			workspace,
			label,
			cluster.Summary,
			cluster.Reason,
			clusterSuccesses,
			existingPattern,
			c.now(),
		)
		patterns = append(patterns, pattern)
		clusteredIDs = append(clusteredIDs, collectRecordIDs(clusterEvidence)...)
	}
	if len(assigned) != len(evidenceByID) {
		return nil, nil
	}
	return patterns, clusteredIDs
}

func parseLLMClusterResponse(content string) (llmClusterResponse, bool) {
	normalized := strings.TrimSpace(content)
	normalized = strings.TrimPrefix(normalized, "```json")
	normalized = strings.TrimPrefix(normalized, "```")
	normalized = strings.TrimSuffix(normalized, "```")
	normalized = strings.TrimSpace(normalized)
	var payload llmClusterResponse
	if err := json.Unmarshal([]byte(normalized), &payload); err != nil {
		return llmClusterResponse{}, false
	}
	return payload, true
}

func buildPatternClusterPrompt(workspace string, tasks []LearningRecord, existing []LearningRecord) string {
	type taskPayload struct {
		ID                 string `json:"id"`
		Summary            string `json:"summary"`
		FinalOutputExcerpt string `json:"final_output_excerpt"`
		Success            *bool  `json:"success,omitempty"`
	}
	type patternPayload struct {
		Label   string `json:"label"`
		Summary string `json:"summary"`
	}
	payload := struct {
		Instruction      string           `json:"instruction"`
		ExistingPatterns []patternPayload `json:"existing_patterns,omitempty"`
		Tasks            []taskPayload    `json:"tasks"`
	}{
		Instruction: "Group tasks that have the same reusable task meaning. Use existing pattern labels when they fit. Labels must be lowercase hyphenated and must not include concrete values.",
	}
	for _, pattern := range existing {
		if pattern.WorkspaceID != workspace {
			continue
		}
		if strings.TrimSpace(pattern.Label) == "" {
			continue
		}
		payload.ExistingPatterns = append(payload.ExistingPatterns, patternPayload{
			Label:   strings.TrimSpace(pattern.Label),
			Summary: strings.TrimSpace(pattern.Summary),
		})
	}
	for _, task := range tasks {
		payload.Tasks = append(payload.Tasks, taskPayload{
			ID:                 task.ID,
			Summary:            task.Summary,
			FinalOutputExcerpt: summarizeText(task.FinalOutput, 800),
			Success:            task.Success,
		})
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Sprintf("tasks: %d", len(tasks))
	}
	return string(data)
}

func buildPatternFromCluster(
	workspace, label, summary, reason string,
	tasks []LearningRecord,
	existing LearningRecord,
	now time.Time,
) LearningRecord {
	taskIDs := append([]string(nil), existing.TaskRecordIDs...)
	taskIDs = appendUniqueStrings(taskIDs, collectRecordIDs(tasks)...)
	if summary = strings.TrimSpace(summary); summary == "" {
		summary = labelSummary(label)
	}
	pattern := existing
	if strings.TrimSpace(pattern.ID) == "" {
		pattern = LearningRecord{
			ID:          stableRuleID(workspace, label),
			Kind:        RecordKindPattern,
			WorkspaceID: workspace,
			CreatedAt:   now,
			Status:      RecordStatus("ready"),
		}
	} else {
		updatedAt := now
		pattern.UpdatedAt = &updatedAt
	}
	pattern.Label = label
	pattern.Summary = summary
	pattern.TaskRecordIDs = taskIDs
	pattern.ClusterReason = strings.TrimSpace(reason)
	pattern.Status = RecordStatus("ready")
	pattern.Source = nil
	pattern.SourceRecordIDs = nil
	pattern.EventCount = 0
	pattern.SuccessRate = 0
	pattern.MaturityScore = 0
	pattern.WinningPath = nil
	pattern.LateAddedSkills = nil
	pattern.FinalSnapshotTrigger = ""
	pattern.MatchedSkillNames = nil
	return pattern
}

func patternsByLabel(patterns []LearningRecord, workspace string) map[string]LearningRecord {
	out := make(map[string]LearningRecord, len(patterns))
	for _, pattern := range patterns {
		if pattern.WorkspaceID != workspace {
			continue
		}
		label := strings.TrimSpace(pattern.Label)
		if label == "" {
			label = validSkillNameOrEmpty(pattern.Summary)
		}
		if label == "" {
			continue
		}
		out[label] = pattern
	}
	return out
}

func heuristicClusterLabel(record LearningRecord) string {
	if label := heuristicASCIIClusterLabel(record.Summary); label != "" {
		return label
	}
	if normalized := normalizeUnicodeTaskSummary(record.Summary); normalized != "" {
		return hashedTaskLabel(normalized)
	}
	return ""
}

func heuristicClusterKey(record LearningRecord) string {
	if label := heuristicASCIIClusterLabel(record.Summary); label != "" {
		return "ascii:" + label
	}
	if normalized := normalizeUnicodeTaskSummary(record.Summary); normalized != "" {
		return "unicode:" + hashedTaskLabel(normalized)
	}
	return ""
}

func heuristicClusterLabelForGroup(key string, cluster []LearningRecord) string {
	if strings.HasPrefix(key, "ascii:") || strings.HasPrefix(key, "unicode:") {
		return strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(key, "ascii:"), "unicode:"))
	}
	for _, record := range cluster {
		if label := heuristicClusterLabel(record); label != "" {
			return label
		}
	}
	return ""
}

func heuristicClusterSummary(label string, cluster []LearningRecord) string {
	for _, record := range cluster {
		if summary := strings.TrimSpace(record.Summary); summary != "" {
			return summary
		}
	}
	return labelSummary(label)
}

func heuristicASCIIClusterLabel(summary string) string {
	tokens := tokenizeForEvolution(summary)
	out := make([]string, 0, len(tokens))
	for _, token := range tokens {
		if isNumericToken(token) {
			continue
		}
		out = append(out, token)
		if len(out) >= 5 {
			break
		}
	}
	return validSkillNameOrEmpty(strings.Join(out, "-"))
}

func normalizeUnicodeTaskSummary(summary string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(summary)) {
		if unicode.IsDigit(r) || unicode.IsSpace(r) || unicode.IsPunct(r) || unicode.IsSymbol(r) {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func hashedTaskLabel(value string) string {
	sum := sha1.Sum([]byte(value))
	return "task-" + hex.EncodeToString(sum[:4])
}

func labelSummary(label string) string {
	label = strings.ReplaceAll(strings.TrimSpace(label), "-", " ")
	if label == "" {
		return "Learned task pattern."
	}
	return strings.ToUpper(label[:1]) + label[1:] + "."
}

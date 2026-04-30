package evolution

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/skills"
)

type LLMDraftGenerator struct {
	provider providers.LLMProvider
	model    string
	fallback DraftGenerator
}

type llmDraftResponse struct {
	TargetSkillName    string   `json:"target_skill_name"`
	DraftType          string   `json:"draft_type"`
	ChangeKind         string   `json:"change_kind"`
	HumanSummary       string   `json:"human_summary"`
	IntendedUseCases   []string `json:"intended_use_cases"`
	PreferredEntryPath []string `json:"preferred_entry_path"`
	AvoidPatterns      []string `json:"avoid_patterns"`
	BodyOrPatch        string   `json:"body_or_patch"`
}

func NewLLMDraftGenerator(provider providers.LLMProvider, model string, fallback DraftGenerator) *LLMDraftGenerator {
	return &LLMDraftGenerator{
		provider: provider,
		model:    strings.TrimSpace(model),
		fallback: fallback,
	}
}

func (g *LLMDraftGenerator) GenerateDraft(ctx context.Context, rule LearningRecord, matches []skills.SkillInfo) (SkillDraft, error) {
	if g == nil || g.provider == nil {
		return g.generateFallback(ctx, rule, matches)
	}

	model := g.model
	if model == "" {
		model = strings.TrimSpace(g.provider.GetDefaultModel())
	}
	if model == "" {
		return g.generateFallback(ctx, rule, matches)
	}

	resp, err := g.provider.Chat(ctx, []providers.Message{
		{
			Role:    "system",
			Content: "Return exactly one JSON object for a skill draft. Do not use markdown fences.",
		},
		{
			Role:    "user",
			Content: g.buildPrompt(rule, matches),
		},
	}, nil, model, map[string]any{"temperature": 0.2})
	if err != nil || resp == nil {
		return g.generateFallback(ctx, rule, matches)
	}

	content := strings.TrimSpace(resp.Content)
	if content == "" {
		return g.generateFallback(ctx, rule, matches)
	}

	draft, ok := parseLLMDraft(content)
	if !ok || len(ValidateDraft(draft)) > 0 {
		return g.generateFallback(ctx, rule, matches)
	}

	return draft, nil
}

func (g *LLMDraftGenerator) generateFallback(
	ctx context.Context,
	rule LearningRecord,
	matches []skills.SkillInfo,
) (SkillDraft, error) {
	if g == nil || g.fallback == nil {
		return SkillDraft{}, nil
	}
	return g.fallback.GenerateDraft(ctx, rule, matches)
}

func (g *LLMDraftGenerator) buildPrompt(rule LearningRecord, matches []skills.SkillInfo) string {
	return strings.Join([]string{
		"Generate a skill draft JSON object with these required string fields:",
		`target_skill_name, draft_type, change_kind, human_summary, body_or_patch.`,
		"Optional array fields: intended_use_cases, preferred_entry_path, avoid_patterns.",
		"",
		"Allowed values:",
		"- draft_type: workflow | shortcut",
		"- change_kind: create | append | replace | merge",
		"",
		"Rule summary: " + strings.TrimSpace(rule.Summary),
		"Winning path: " + joinOrFallback(rule.WinningPath, "none"),
		"Late-added successful skills: " + joinOrFallback(rule.LateAddedSkills, "none"),
		"Final snapshot trigger: " + fallbackString(rule.FinalSnapshotTrigger, "none"),
		fmt.Sprintf("Event count: %d", rule.EventCount),
		fmt.Sprintf("Success rate: %.2f", rule.SuccessRate),
		"Matched skill refs: " + summarizeSkillMatches(matches),
		"Matched skill names: " + joinOrFallback(rule.MatchedSkillNames, "none"),
		"",
		combinedSkillGuidance(rule),
		"body_or_patch should contain the full draft body or patch content as plain text.",
	}, "\n")
}

func combinedSkillGuidance(rule LearningRecord) string {
	if target := inferCombinedSkillName(rule); target != "" {
		return strings.Join([]string{
			"This rule represents a stable multi-step successful path.",
			"Prefer creating a new combined shortcut skill instead of modifying one component skill.",
			"Suggested target skill name: " + target,
		}, "\n")
	}
	return "Prefer updating an existing skill only when the learned pattern clearly belongs inside that single skill."
}

func parseLLMDraft(content string) (SkillDraft, bool) {
	normalized := strings.TrimSpace(content)
	normalized = strings.TrimPrefix(normalized, "```json")
	normalized = strings.TrimPrefix(normalized, "```")
	normalized = strings.TrimSuffix(normalized, "```")
	normalized = strings.TrimSpace(normalized)

	var payload llmDraftResponse
	if err := json.Unmarshal([]byte(normalized), &payload); err != nil {
		return SkillDraft{}, false
	}

	draft := SkillDraft{
		TargetSkillName:    strings.TrimSpace(payload.TargetSkillName),
		DraftType:          DraftType(strings.TrimSpace(payload.DraftType)),
		ChangeKind:         ChangeKind(strings.TrimSpace(payload.ChangeKind)),
		HumanSummary:       strings.TrimSpace(payload.HumanSummary),
		IntendedUseCases:   append([]string(nil), payload.IntendedUseCases...),
		PreferredEntryPath: append([]string(nil), payload.PreferredEntryPath...),
		AvoidPatterns:      append([]string(nil), payload.AvoidPatterns...),
		BodyOrPatch:        strings.TrimSpace(payload.BodyOrPatch),
	}
	return draft, true
}

func summarizeSkillMatches(matches []skills.SkillInfo) string {
	if len(matches) == 0 {
		return "none"
	}

	parts := make([]string, 0, len(matches))
	for _, match := range matches {
		part := strings.TrimSpace(match.Name)
		if desc := strings.TrimSpace(match.Description); desc != "" {
			part += ": " + desc
		}
		if path := strings.TrimSpace(match.Path); path != "" {
			part += " (" + path + ")"
		}
		if part != "" {
			parts = append(parts, part)
		}
	}
	if len(parts) == 0 {
		return "none"
	}
	return strings.Join(parts, "; ")
}

func joinOrFallback(parts []string, fallback string) string {
	if len(parts) == 0 {
		return fallback
	}
	return strings.Join(parts, " -> ")
}

func fallbackString(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

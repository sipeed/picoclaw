package evolution

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/skills"
)

type DraftGenerator interface {
	GenerateDraft(ctx context.Context, rule LearningRecord, matches []skills.SkillInfo) (SkillDraft, error)
}

func ValidateDraft(draft SkillDraft) []string {
	findings := make([]string, 0, 5)

	if strings.TrimSpace(draft.TargetSkillName) == "" {
		findings = append(findings, "target_skill_name is required")
	} else if err := skills.ValidateSkillName(draft.TargetSkillName); err != nil {
		findings = append(findings, "target_skill_name is invalid: "+err.Error())
	}
	if strings.TrimSpace(draft.HumanSummary) == "" {
		findings = append(findings, "human_summary is required")
	}
	if strings.TrimSpace(draft.BodyOrPatch) == "" {
		findings = append(findings, "body_or_patch is required")
	}

	switch draft.DraftType {
	case DraftTypeWorkflow, DraftTypeShortcut:
	default:
		findings = append(findings, "draft_type is invalid")
	}

	switch draft.ChangeKind {
	case ChangeKindCreate, ChangeKindAppend, ChangeKindReplace, ChangeKindMerge:
	default:
		findings = append(findings, "change_kind is invalid")
	}

	return findings
}

type DefaultDraftGenerator struct {
	loader *skills.SkillsLoader
}

func NewDefaultDraftGenerator(workspace string) *DefaultDraftGenerator {
	builtinSkillsDir := strings.TrimSpace(os.Getenv(config.EnvBuiltinSkills))
	if builtinSkillsDir == "" {
		wd, _ := os.Getwd()
		builtinSkillsDir = filepath.Join(wd, "skills")
	}

	globalSkillsDir := filepath.Join(config.GetHome(), "skills")
	return &DefaultDraftGenerator{
		loader: skills.NewSkillsLoader(workspace, globalSkillsDir, builtinSkillsDir),
	}
}

func (g *DefaultDraftGenerator) GenerateDraft(_ context.Context, rule LearningRecord, matches []skills.SkillInfo) (SkillDraft, error) {
	target := inferTargetSkillName(rule, matches)
	if target == "" {
		target = "learned-skill"
	}

	_, hasExisting, err := g.loadBaseSkillContent(target, matches)
	if err != nil {
		return SkillDraft{}, err
	}

	draftType := DraftTypeWorkflow
	if len(rule.WinningPath) <= 1 {
		draftType = DraftTypeShortcut
	}

	changeKind := ChangeKindCreate
	body := g.buildNewSkillBody(target, rule)
	if hasExisting {
		changeKind = ChangeKindAppend
		body = g.buildAppendBody(rule)
	}

	return SkillDraft{
		TargetSkillName:    target,
		DraftType:          draftType,
		ChangeKind:         changeKind,
		HumanSummary:       g.buildHumanSummary(target, rule, hasExisting),
		IntendedUseCases:   inferIntendedUseCases(rule),
		PreferredEntryPath: inferPreferredEntryPath(rule),
		AvoidPatterns:      inferAvoidPatterns(rule),
		BodyOrPatch:        body,
	}, nil
}

func inferTargetSkillName(rule LearningRecord, matches []skills.SkillInfo) string {
	if len(matches) > 0 && strings.TrimSpace(matches[0].Name) != "" {
		return strings.TrimSpace(matches[0].Name)
	}
	if len(rule.LateAddedSkills) > 0 && strings.TrimSpace(rule.LateAddedSkills[0]) != "" {
		return strings.TrimSpace(rule.LateAddedSkills[0])
	}
	if len(rule.WinningPath) > 0 && strings.TrimSpace(rule.WinningPath[0]) != "" {
		return strings.TrimSpace(rule.WinningPath[0])
	}
	if len(rule.MatchedSkillNames) > 0 && strings.TrimSpace(rule.MatchedSkillNames[0]) != "" {
		return strings.TrimSpace(rule.MatchedSkillNames[0])
	}

	tokens := tokenizeForEvolution(rule.Summary)
	if len(tokens) > 0 {
		return tokens[0]
	}
	return ""
}

func (g *DefaultDraftGenerator) loadBaseSkillContent(target string, matches []skills.SkillInfo) (string, bool, error) {
	for _, match := range matches {
		if match.Name != target || strings.TrimSpace(match.Path) == "" {
			continue
		}
		data, err := os.ReadFile(match.Path)
		if err != nil {
			return "", false, err
		}
		return string(data), true, nil
	}

	if g.loader == nil {
		return "", false, nil
	}
	content, ok := g.loader.LoadSkill(target)
	if !ok {
		return "", false, nil
	}
	description := fmt.Sprintf("Learned workflow for %s.", target)
	return buildSkillDocument(target, description, content), true, nil
}

func (g *DefaultDraftGenerator) buildHumanSummary(target string, rule LearningRecord, hasExisting bool) string {
	if hasExisting {
		return fmt.Sprintf("Refresh %s with learned pattern: %s", target, rule.Summary)
	}
	return fmt.Sprintf("Create %s from learned pattern: %s", target, rule.Summary)
}

func (g *DefaultDraftGenerator) buildNewSkillBody(target string, rule LearningRecord) string {
	description := fmt.Sprintf("Learned workflow for %s.", target)
	body := strings.Join([]string{
		"# " + titleCaseSkillName(target),
		"",
		"## Start Here",
		g.startHereLine(rule),
		"",
		"## When To Use",
		fmt.Sprintf("Use this skill when the task matches `%s`.", strings.TrimSpace(rule.Summary)),
		"",
		"## Learned Pattern",
		g.learnedPatternLine(rule),
		"",
		"## Winning Path",
		g.winningPathLine(rule),
		"",
		"## Evidence",
		g.evidenceLine(rule),
	}, "\n")
	return buildSkillDocument(target, description, body)
}

func (g *DefaultDraftGenerator) buildAppendBody(rule LearningRecord) string {
	return strings.Join([]string{
		"## Learned Evolution",
		fmt.Sprintf("- Summary: %s", strings.TrimSpace(rule.Summary)),
		fmt.Sprintf("- Learned pattern: %s", g.learnedPatternLine(rule)),
		fmt.Sprintf("- Winning path: %s", g.winningPathLine(rule)),
		fmt.Sprintf("- Evidence: %s", g.evidenceLine(rule)),
		"",
	}, "\n")
}

func buildSkillDocument(name, description, body string) string {
	return strings.Join([]string{
		"---",
		"name: " + strings.TrimSpace(name),
		"description: " + strings.TrimSpace(description),
		"---",
		"",
		strings.TrimSpace(body),
		"",
	}, "\n")
}

func titleCaseSkillName(name string) string {
	parts := strings.FieldsFunc(name, func(r rune) bool { return r == '-' || r == '_' || r == ' ' })
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	if len(parts) == 0 {
		return "Learned Skill"
	}
	return strings.Join(parts, " ")
}

func (g *DefaultDraftGenerator) startHereLine(rule LearningRecord) string {
	if len(rule.WinningPath) > 0 {
		return fmt.Sprintf("Start with `%s` before trying other paths.", strings.Join(rule.WinningPath, " -> "))
	}
	return fmt.Sprintf("Start from the learned path for `%s`.", strings.TrimSpace(rule.Summary))
}

func (g *DefaultDraftGenerator) learnedPatternLine(rule LearningRecord) string {
	if len(rule.LateAddedSkills) > 0 {
		return fmt.Sprintf(
			"Late-added skill `%s` was repeatedly introduced immediately before success%s.",
			strings.Join(rule.LateAddedSkills, " -> "),
			triggerSuffix(rule.FinalSnapshotTrigger),
		)
	}
	if len(rule.WinningPath) > 0 {
		return fmt.Sprintf("Prefer `%s` because it was the most reliable recent path.", strings.Join(rule.WinningPath, " -> "))
	}
	return fmt.Sprintf("Prefer the pattern summarized as `%s`.", strings.TrimSpace(rule.Summary))
}

func (g *DefaultDraftGenerator) winningPathLine(rule LearningRecord) string {
	if len(rule.WinningPath) == 0 {
		return "No explicit winning path was recorded."
	}
	return strings.Join(rule.WinningPath, " -> ")
}

func (g *DefaultDraftGenerator) evidenceLine(rule LearningRecord) string {
	return fmt.Sprintf("%d cases, success rate %.2f", rule.EventCount, rule.SuccessRate)
}

func triggerSuffix(trigger string) string {
	trigger = strings.TrimSpace(trigger)
	if trigger == "" {
		return ""
	}
	return fmt.Sprintf(" during `%s`", trigger)
}

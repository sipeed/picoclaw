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

type EvidenceAwareDraftGenerator interface {
	GenerateDraftWithEvidence(ctx context.Context, rule LearningRecord, matches []skills.SkillInfo, evidence DraftEvidence) (SkillDraft, error)
}

type DraftEvidence struct {
	TaskRecords []LearningRecord
}

func ValidateDraft(draft SkillDraft) []string {
	findings := make([]string, 0, 5)

	if strings.TrimSpace(draft.TargetSkillName) == "" {
		findings = append(findings, "target_skill_name is required")
	} else if err := skills.ValidateSkillName(draft.TargetSkillName); err != nil {
		findings = append(findings, "target_skill_name is invalid: "+err.Error())
	} else if isNumericToken(strings.TrimSpace(draft.TargetSkillName)) {
		findings = append(findings, "target_skill_name must be descriptive, not numeric-only")
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
	return g.GenerateDraftWithEvidence(context.Background(), rule, matches, DraftEvidence{})
}

func (g *DefaultDraftGenerator) GenerateDraftWithEvidence(_ context.Context, rule LearningRecord, matches []skills.SkillInfo, evidence DraftEvidence) (SkillDraft, error) {
	rule = enrichRuleWithDraftEvidence(rule, evidence)
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
	body := g.buildNewSkillBody(target, rule, evidence, matches)
	if hasExisting {
		changeKind = ChangeKindAppend
		body = g.buildAppendBody(rule, evidence, matches)
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
	if target := inferCombinedSkillName(rule); target != "" {
		return target
	}
	if label := validSkillNameOrEmpty(rule.Label); label != "" {
		return label
	}
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
		if len(tokens) == 1 && isNumericToken(tokens[0]) {
			return "learned-" + tokens[0]
		}
		return tokens[0]
	}
	return ""
}

func enrichRuleWithDraftEvidence(rule LearningRecord, evidence DraftEvidence) LearningRecord {
	if len(evidence.TaskRecords) == 0 {
		return rule
	}
	usedSkillNames := make([]string, 0)
	pathCounts := make(map[string]int)
	pathByKey := make(map[string][]string)
	for _, task := range evidence.TaskRecords {
		path := uniqueTrimmedNames(task.UsedSkillNames)
		if len(path) == 0 {
			continue
		}
		usedSkillNames = append(usedSkillNames, path...)
		key := strings.Join(path, "\x00")
		pathCounts[key]++
		pathByKey[key] = path
	}
	rule.MatchedSkillNames = appendUniqueStrings(rule.MatchedSkillNames, uniqueTrimmedNames(usedSkillNames)...)
	if len(rule.WinningPath) == 0 {
		bestKey := ""
		bestCount := 0
		for key, count := range pathCounts {
			if count > bestCount || (count == bestCount && key < bestKey) {
				bestKey = key
				bestCount = count
			}
		}
		if bestKey != "" {
			rule.WinningPath = append([]string(nil), pathByKey[bestKey]...)
		}
	}
	return rule
}

func inferCombinedSkillName(rule LearningRecord) string {
	path := normalizePath(rule.WinningPath)
	if len(path) < 2 {
		return ""
	}

	tokens := tokenizeForEvolution(rule.Summary)
	suffix := commonWinningPathSuffix(path)
	if len(tokens) == 1 && isNumericToken(tokens[0]) && suffix != "" {
		if candidate := validSkillNameOrEmpty("calculate-" + tokens[0] + "-via-" + pluralizeSuffix(suffix)); candidate != "" {
			return candidate
		}
	}
	if len(tokens) >= 2 {
		prefix := strings.Join(tokens[:minInt(len(tokens), 4)], "-")
		if suffix != "" {
			if candidate := validSkillNameOrEmpty(prefix + "-via-" + pluralizeSuffix(suffix)); candidate != "" {
				return candidate
			}
		}
		if candidate := validSkillNameOrEmpty(prefix + "-shortcut"); candidate != "" {
			return candidate
		}
	}

	compressedPath := compressedWinningPathName(path)
	if candidate := validSkillNameOrEmpty("combined-" + compressedPath); candidate != "" {
		return candidate
	}
	if candidate := validSkillNameOrEmpty(path[0] + "-to-" + path[len(path)-1] + "-shortcut"); candidate != "" {
		return candidate
	}
	return ""
}

func commonWinningPathSuffix(path []string) string {
	if len(path) < 2 {
		return ""
	}

	var suffix string
	for i, name := range path {
		parts := strings.Split(strings.TrimSpace(name), "-")
		if len(parts) == 0 {
			return ""
		}
		last := strings.TrimSpace(parts[len(parts)-1])
		if last == "" {
			return ""
		}
		if i == 0 {
			suffix = last
			continue
		}
		if suffix != last {
			return ""
		}
	}
	return suffix
}

func compressedWinningPathName(path []string) string {
	suffix := commonWinningPathSuffix(path)
	fragments := make([]string, 0, len(path)+1)
	for _, name := range path {
		trimmed := strings.TrimSpace(name)
		if trimmed == "" {
			continue
		}
		if suffix != "" {
			trimmed = strings.TrimSuffix(trimmed, "-"+suffix)
			trimmed = strings.TrimSuffix(trimmed, suffix)
			trimmed = strings.Trim(trimmed, "-")
		}
		if trimmed != "" {
			fragments = append(fragments, trimmed)
		}
	}
	if suffix != "" {
		fragments = append(fragments, pluralizeSuffix(suffix))
	}
	if len(fragments) == 0 {
		return strings.Join(path, "-")
	}
	return strings.Join(fragments, "-")
}

func pluralizeSuffix(suffix string) string {
	suffix = strings.TrimSpace(strings.ToLower(suffix))
	if suffix == "" {
		return ""
	}
	if strings.HasSuffix(suffix, "s") {
		return suffix
	}
	return suffix + "s"
}

func isNumericToken(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func validSkillNameOrEmpty(candidate string) string {
	candidate = strings.Trim(candidate, "-")
	candidate = strings.Join(strings.FieldsFunc(candidate, func(r rune) bool {
		return !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9')
	}), "-")
	candidate = strings.ToLower(strings.Trim(candidate, "-"))
	if candidate == "" {
		return ""
	}
	if len(candidate) > skills.MaxNameLength {
		return ""
	}
	if err := skills.ValidateSkillName(candidate); err != nil {
		return ""
	}
	return candidate
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
	description := fmt.Sprintf("Use this skill to %s when the task requires this workflow.", sentenceFragment(target))
	return buildSkillDocument(target, description, content), true, nil
}

func (g *DefaultDraftGenerator) buildHumanSummary(target string, rule LearningRecord, hasExisting bool) string {
	if hasExisting {
		return fmt.Sprintf("Refresh %s with learned pattern: %s", target, rule.Summary)
	}
	return fmt.Sprintf("Create %s from learned pattern: %s", target, rule.Summary)
}

func (g *DefaultDraftGenerator) buildNewSkillBody(target string, rule LearningRecord, evidence DraftEvidence, matches []skills.SkillInfo) string {
	description := fmt.Sprintf("Use this skill to %s when the task matches this workflow.", sentenceFragment(fallbackString(rule.Summary, target)))
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
		"## Procedure",
		g.procedureLine(rule, evidence),
		"",
		"## Expected Result",
		g.expectedResultLine(evidence),
		"",
		"## Source Skills",
		synthesizedComponentBreakdown(matches),
		"",
		"## Source Evidence",
		g.evidenceLine(rule, evidence),
	}, "\n")
	return buildSkillDocument(target, description, body)
}

func (g *DefaultDraftGenerator) buildAppendBody(rule LearningRecord, evidence DraftEvidence, matches []skills.SkillInfo) string {
	return strings.Join([]string{
		"## Learned Evolution",
		fmt.Sprintf("- Summary: %s", strings.TrimSpace(rule.Summary)),
		fmt.Sprintf("- Learned pattern: %s", g.learnedPatternLine(rule)),
		fmt.Sprintf("- Procedure: %s", g.procedureLine(rule, evidence)),
		fmt.Sprintf("- Expected result: %s", g.expectedResultLine(evidence)),
		fmt.Sprintf("- Evidence: %s", g.evidenceLine(rule, evidence)),
		"",
		"### Source Skills",
		synthesizedComponentBreakdown(matches),
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

func (g *DefaultDraftGenerator) procedureLine(rule LearningRecord, evidence DraftEvidence) string {
	if len(rule.WinningPath) > 0 {
		return fmt.Sprintf("Follow `%s`, applying the concrete operation from each source skill, then return the final result directly.", strings.Join(rule.WinningPath, " -> "))
	}
	if excerpt := firstFinalOutputExcerpt(evidence, 260); excerpt != "" {
		return "Use the same operation demonstrated by the source task result: " + excerpt
	}
	return fmt.Sprintf("Solve tasks matching `%s` using the learned successful workflow, then return the final result directly.", strings.TrimSpace(rule.Summary))
}

func (g *DefaultDraftGenerator) expectedResultLine(evidence DraftEvidence) string {
	if excerpt := firstFinalOutputExcerpt(evidence, 320); excerpt != "" {
		return excerpt
	}
	return "Return the completed result for the matched task without restating unrelated discovery steps."
}

func (g *DefaultDraftGenerator) evidenceLine(rule LearningRecord, evidence DraftEvidence) string {
	if len(evidence.TaskRecords) > 0 {
		ids := make([]string, 0, len(evidence.TaskRecords))
		for _, task := range evidence.TaskRecords {
			ids = append(ids, task.ID)
		}
		return fmt.Sprintf("Learned from task records: %s", strings.Join(ids, ", "))
	}
	if len(rule.TaskRecordIDs) > 0 {
		return fmt.Sprintf("Learned from task records: %s", strings.Join(rule.TaskRecordIDs, ", "))
	}
	return "Learned from the pattern record."
}

func firstFinalOutputExcerpt(evidence DraftEvidence, maxLen int) string {
	for _, task := range evidence.TaskRecords {
		if excerpt := summarizeText(task.FinalOutput, maxLen); excerpt != "" {
			return excerpt
		}
	}
	return ""
}

func triggerSuffix(trigger string) string {
	trigger = strings.TrimSpace(trigger)
	if trigger == "" {
		return ""
	}
	return fmt.Sprintf(" during `%s`", trigger)
}

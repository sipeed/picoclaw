package evolution

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/skills"
)

type SkillsRecaller struct {
	workspace string
	loader    *skills.SkillsLoader
}

func NewSkillsRecaller(workspace string) *SkillsRecaller {
	builtinSkillsDir := strings.TrimSpace(os.Getenv(config.EnvBuiltinSkills))
	if builtinSkillsDir == "" {
		wd, _ := os.Getwd()
		builtinSkillsDir = filepath.Join(wd, "skills")
	}

	globalSkillsDir := filepath.Join(config.GetHome(), "skills")
	return &SkillsRecaller{
		workspace: workspace,
		loader:    skills.NewSkillsLoader(workspace, globalSkillsDir, builtinSkillsDir),
	}
}

func (r *SkillsRecaller) RecallSimilarSkills(rule LearningRecord) ([]skills.SkillInfo, error) {
	if r == nil || r.loader == nil {
		return nil, nil
	}

	all := r.loader.ListSkills()
	if names := explicitRecallSkillNames(rule); len(names) > 0 {
		return filterSkillsByExplicitNames(all, names), nil
	}

	type scored struct {
		info       skills.SkillInfo
		score      int
		sourceRank int
	}

	scoredList := make([]scored, 0, len(all))
	for _, skill := range all {
		score := scoreSkillMatch(rule, skill)
		if score <= 0 {
			continue
		}

		if body, ok := r.loader.LoadSkill(skill.Name); ok {
			score += scoreSkillBody(rule, body)
		}

		scoredList = append(scoredList, scored{
			info:       skill,
			score:      score,
			sourceRank: skillSourceRank(skill.Source),
		})
	}

	sort.Slice(scoredList, func(i, j int) bool {
		if scoredList[i].score != scoredList[j].score {
			return scoredList[i].score > scoredList[j].score
		}
		if scoredList[i].sourceRank != scoredList[j].sourceRank {
			return scoredList[i].sourceRank < scoredList[j].sourceRank
		}
		return scoredList[i].info.Name < scoredList[j].info.Name
	})

	out := make([]skills.SkillInfo, 0, len(scoredList))
	for _, item := range scoredList {
		out = append(out, item.info)
	}
	return out, nil
}

func explicitRecallSkillNames(rule LearningRecord) []string {
	names := make([]string, 0, len(rule.WinningPath)+len(rule.MatchedSkillNames)+len(rule.LateAddedSkills))
	names = append(names, normalizePath(rule.WinningPath)...)
	names = append(names, normalizePath(rule.MatchedSkillNames)...)
	names = append(names, normalizePath(rule.LateAddedSkills)...)
	return uniqueTrimmedNames(names)
}

func filterSkillsByExplicitNames(all []skills.SkillInfo, names []string) []skills.SkillInfo {
	if len(all) == 0 || len(names) == 0 {
		return nil
	}

	byName := make(map[string]skills.SkillInfo, len(all))
	for _, skill := range all {
		name := strings.ToLower(strings.TrimSpace(skill.Name))
		if name == "" {
			continue
		}
		if _, exists := byName[name]; exists {
			continue
		}
		byName[name] = skill
	}

	out := make([]skills.SkillInfo, 0, len(names))
	for _, name := range names {
		if skill, ok := byName[strings.ToLower(strings.TrimSpace(name))]; ok {
			out = append(out, skill)
		}
	}
	return out
}

func scoreSkillMatch(rule LearningRecord, skill skills.SkillInfo) int {
	score := 0
	skillName := strings.ToLower(strings.TrimSpace(skill.Name))
	ruleSummary := strings.ToLower(rule.Summary)

	if skillName != "" {
		if containsNormalized(rule.WinningPath, skillName) {
			score += 8
		}
		if containsNormalized(rule.MatchedSkillNames, skillName) {
			score += 6
		}
		if strings.Contains(ruleSummary, skillName) {
			score += 4
		}
	}

	score += 2 * tokenOverlap(ruleTokens(rule), tokenizeForEvolution(skill.Name+" "+skill.Description))
	return score
}

func scoreSkillBody(rule LearningRecord, body string) int {
	return minInt(tokenOverlap(ruleTokens(rule), tokenizeForEvolution(body)), 3)
}

func skillSourceRank(source string) int {
	switch source {
	case "workspace":
		return 0
	case "global":
		return 1
	case "builtin":
		return 2
	default:
		return 3
	}
}

func ruleTokens(rule LearningRecord) []string {
	parts := make([]string, 0, len(rule.WinningPath)+len(rule.MatchedSkillNames)+4)
	parts = append(parts, normalizePath(rule.WinningPath)...)
	parts = append(parts, normalizePath(rule.MatchedSkillNames)...)
	parts = append(parts, tokenizeForEvolution(rule.Summary)...)
	return parts
}

func containsNormalized(values []string, target string) bool {
	target = strings.ToLower(strings.TrimSpace(target))
	for _, value := range values {
		if strings.ToLower(strings.TrimSpace(value)) == target {
			return true
		}
	}
	return false
}

func tokenOverlap(left, right []string) int {
	if len(left) == 0 || len(right) == 0 {
		return 0
	}

	leftSet := make(map[string]struct{}, len(left))
	for _, token := range left {
		leftSet[token] = struct{}{}
	}

	seen := make(map[string]struct{}, len(right))
	count := 0
	for _, token := range right {
		if _, ok := seen[token]; ok {
			continue
		}
		seen[token] = struct{}{}
		if _, ok := leftSet[token]; ok {
			count++
		}
	}
	return count
}

func tokenizeForEvolution(text string) []string {
	fields := strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9')
	})

	out := make([]string, 0, len(fields))
	for _, field := range fields {
		if field == "" {
			continue
		}
		out = append(out, field)
	}
	return out
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

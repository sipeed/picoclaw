package evolution

import (
	"strings"
	"time"
)

func SaveAppliedProfile(store *Store, workspace string, draft SkillDraft, now time.Time) error {
	return store.UpdateProfile(workspace, draft.TargetSkillName, func(profile *SkillProfile, exists bool) error {
		if !exists {
			*profile = SkillProfile{
				SkillName:   draft.TargetSkillName,
				WorkspaceID: workspace,
				Origin:      "evolved",
			}
		}

		profile.SkillName = draft.TargetSkillName
		profile.WorkspaceID = workspace
		profile.CurrentVersion = draft.ID
		profile.Status = SkillStatusActive
		profile.Origin = profileOrigin(profile.Origin)
		profile.HumanSummary = draft.HumanSummary
		profile.ChangeReason = draft.HumanSummary
		profile.IntendedUseCases = append([]string(nil), draft.IntendedUseCases...)
		profile.PreferredEntryPath = append([]string(nil), draft.PreferredEntryPath...)
		profile.AvoidPatterns = append([]string(nil), draft.AvoidPatterns...)
		profile.LastUsedAt = now
		if profile.RetentionScore <= 0 {
			profile.RetentionScore = 1
		}
		profile.VersionHistory = append(profile.VersionHistory, SkillVersionEntry{
			Version:   draft.ID,
			Action:    string(draft.ChangeKind),
			Timestamp: now,
			DraftID:   draft.ID,
			Summary:   draft.HumanSummary,
		})
		return nil
	})
}

func inferIntendedUseCases(rule LearningRecord) []string {
	summary := strings.TrimSpace(rule.Summary)
	if summary == "" {
		return nil
	}
	return []string{summary}
}

func inferPreferredEntryPath(rule LearningRecord) []string {
	if len(rule.WinningPath) == 0 {
		return nil
	}
	return append([]string(nil), rule.WinningPath...)
}

func inferAvoidPatterns(rule LearningRecord) []string {
	if len(rule.LateAddedSkills) == 0 || len(rule.WinningPath) <= len(rule.LateAddedSkills) {
		return nil
	}
	prefix := rule.WinningPath[:len(rule.WinningPath)-len(rule.LateAddedSkills)]
	if len(prefix) == 0 {
		return nil
	}
	return []string{
		"avoid starting with " + strings.Join(
			prefix,
			" -> ",
		) + " before using " + strings.Join(
			rule.LateAddedSkills,
			" -> ",
		),
	}
}

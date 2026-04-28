package evolutioncmd

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/sipeed/picoclaw/pkg/evolution"
)

func newStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show evolution status",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, store, workspace, err := loadEvolutionDeps()
			if err != nil {
				return err
			}
			paths := evolution.NewPaths(workspace, cfg.Evolution.StateDir)

			drafts, err := store.LoadDrafts()
			if err != nil {
				return err
			}
			profiles, err := store.LoadProfiles()
			if err != nil {
				return err
			}
			drafts = filterDraftsForWorkspace(paths, workspace, drafts)
			profiles = filterProfilesForWorkspace(paths, workspace, profiles)

			_, err = fmt.Fprintf(
				cmd.OutOrStdout(),
				"workspace=%s\nenabled=%v\nmode=%s\ndrafts=%d\ndrafts_by_status=%s\nprofiles=%d\nprofiles_by_status=%s\n%s%s",
				workspace,
				cfg.Evolution.Enabled,
				cfg.Evolution.EffectiveMode(),
				len(drafts),
				formatDraftStatusCounts(drafts),
				len(profiles),
				formatProfileStatusCounts(profiles),
				formatDraftItems(drafts),
				formatProfileItems(profiles),
			)
			return err
		},
	}
}

func filterDraftsForWorkspace(paths evolution.Paths, workspace string, drafts []evolution.SkillDraft) []evolution.SkillDraft {
	filtered := make([]evolution.SkillDraft, 0, len(drafts))
	for _, draft := range drafts {
		if draftBelongsToWorkspace(paths, workspace, draft) {
			filtered = append(filtered, draft)
		}
	}
	return filtered
}

func filterProfilesForWorkspace(paths evolution.Paths, workspace string, profiles []evolution.SkillProfile) []evolution.SkillProfile {
	filtered := make([]evolution.SkillProfile, 0, len(profiles))
	for _, profile := range profiles {
		if profileBelongsToWorkspace(paths, workspace, profile) {
			filtered = append(filtered, profile)
		}
	}
	return filtered
}

func formatDraftStatusCounts(drafts []evolution.SkillDraft) string {
	counts := map[evolution.DraftStatus]int{
		evolution.DraftStatusCandidate:   0,
		evolution.DraftStatusQuarantined: 0,
		evolution.DraftStatusAccepted:    0,
	}
	for _, draft := range drafts {
		counts[draft.Status]++
	}

	parts := []string{
		fmt.Sprintf("%s:%d", evolution.DraftStatusCandidate, counts[evolution.DraftStatusCandidate]),
		fmt.Sprintf("%s:%d", evolution.DraftStatusQuarantined, counts[evolution.DraftStatusQuarantined]),
		fmt.Sprintf("%s:%d", evolution.DraftStatusAccepted, counts[evolution.DraftStatusAccepted]),
	}
	return strings.Join(parts, " ")
}

func formatProfileStatusCounts(profiles []evolution.SkillProfile) string {
	counts := map[evolution.SkillStatus]int{
		evolution.SkillStatusActive:   0,
		evolution.SkillStatusCold:     0,
		evolution.SkillStatusArchived: 0,
		evolution.SkillStatusDeleted:  0,
	}
	for _, profile := range profiles {
		counts[profile.Status]++
	}

	parts := []string{
		fmt.Sprintf("%s:%d", evolution.SkillStatusActive, counts[evolution.SkillStatusActive]),
		fmt.Sprintf("%s:%d", evolution.SkillStatusCold, counts[evolution.SkillStatusCold]),
		fmt.Sprintf("%s:%d", evolution.SkillStatusArchived, counts[evolution.SkillStatusArchived]),
		fmt.Sprintf("%s:%d", evolution.SkillStatusDeleted, counts[evolution.SkillStatusDeleted]),
	}
	return strings.Join(parts, " ")
}

func formatDraftItems(drafts []evolution.SkillDraft) string {
	if len(drafts) == 0 {
		return "draft_items:\n"
	}

	items := append([]evolution.SkillDraft(nil), drafts...)
	sort.Slice(items, func(i, j int) bool {
		left := draftSortTime(items[i])
		right := draftSortTime(items[j])
		if !left.Equal(right) {
			return left.After(right)
		}
		return items[i].ID < items[j].ID
	})

	lines := []string{"draft_items:"}
	for _, draft := range items {
		lines = append(lines, fmt.Sprintf(
			"id=%s status=%s target=%s change=%s summary=%s",
			draft.ID,
			draft.Status,
			draft.TargetSkillName,
			draft.ChangeKind,
			draft.HumanSummary,
		))
	}
	return strings.Join(lines, "\n") + "\n"
}

func formatProfileItems(profiles []evolution.SkillProfile) string {
	if len(profiles) == 0 {
		return "profile_items:\n"
	}

	items := append([]evolution.SkillProfile(nil), profiles...)
	sort.Slice(items, func(i, j int) bool {
		if !items[i].LastUsedAt.Equal(items[j].LastUsedAt) {
			return items[i].LastUsedAt.After(items[j].LastUsedAt)
		}
		return items[i].SkillName < items[j].SkillName
	})

	lines := []string{"profile_items:"}
	for _, profile := range items {
		lines = append(lines, fmt.Sprintf(
			"skill=%s status=%s version=%s uses=%d reason=%s",
			profile.SkillName,
			profile.Status,
			profile.CurrentVersion,
			profile.UseCount,
			profile.ChangeReason,
		))
	}
	return strings.Join(lines, "\n") + "\n"
}

func draftSortTime(draft evolution.SkillDraft) time.Time {
	if draft.UpdatedAt != nil {
		return *draft.UpdatedAt
	}
	return draft.CreatedAt
}

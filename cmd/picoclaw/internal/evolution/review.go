package evolutioncmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/sipeed/picoclaw/pkg/evolution"
)

func newReviewCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "review <draft-id>",
		Short: "Show detailed information for one draft",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, store, workspace, err := loadEvolutionDeps()
			if err != nil {
				return err
			}

			draft, err := loadWorkspaceDraft(store, workspace, cfg.Evolution.StateDir, args[0])
			if err != nil {
				return err
			}
			profile, profileErr := store.LoadProfile(draft.TargetSkillName)
			hasProfile := profileErr == nil
			if profileErr != nil && !os.IsNotExist(profileErr) {
				return profileErr
			}
			preview, err := evolution.BuildDraftPreview(workspace, draft)
			if err != nil {
				return err
			}

			reviewNotes := strings.Join(draft.ReviewNotes, ", ")
			scanFindings := strings.Join(draft.ScanFindings, ", ")
			intendedUseCases := strings.Join(draft.IntendedUseCases, ", ")
			preferredEntryPath := strings.Join(draft.PreferredEntryPath, " -> ")
			avoidPatterns := strings.Join(draft.AvoidPatterns, ", ")
			if _, err := fmt.Fprintf(
				cmd.OutOrStdout(),
				"id=%s\nsource=%s\ntarget=%s\ntype=%s\nchange=%s\nstatus=%s\nsummary=%s\nreview_notes=%s\nscan_findings=%s\nintended_use_cases=%s\npreferred_entry_path=%s\navoid_patterns=%s\n%s%scurrent_body:\n%s\nrendered_body:\n%s\ndiff_preview:\n%s\nbody:\n%s\n",
				draft.ID,
				draft.SourceRecordID,
				draft.TargetSkillName,
				draft.DraftType,
				draft.ChangeKind,
				draft.Status,
				draft.HumanSummary,
				reviewNotes,
				scanFindings,
				intendedUseCases,
				preferredEntryPath,
				avoidPatterns,
				formatReviewProfileSection(profile, hasProfile),
				formatImpactPreview(draft, hasProfile),
				preview.CurrentBody,
				preview.RenderedBody,
				preview.DiffPreview,
				draft.BodyOrPatch,
			); err != nil {
				return err
			}
			return nil
		},
	}
}

func formatReviewProfileSection(profile evolution.SkillProfile, hasProfile bool) string {
	if !hasProfile {
		return "profile:\nmissing=true\nrecent_history:\n"
	}

	lines := []string{
		"profile:",
		fmt.Sprintf(
			"skill=%s status=%s version=%s uses=%d reason=%s",
			profile.SkillName,
			profile.Status,
			profile.CurrentVersion,
			profile.UseCount,
			profile.ChangeReason,
		),
		"current_preferred_entry_path=" + strings.Join(profile.PreferredEntryPath, " -> "),
		"recent_history:",
	}

	history := profile.VersionHistory
	for i := len(history) - 1; i >= 0 && i >= len(history)-3; i-- {
		entry := history[i]
		lines = append(lines, fmt.Sprintf(
			"version=%s action=%s draft_id=%s summary=%s",
			entry.Version,
			entry.Action,
			entry.DraftID,
			entry.Summary,
		))
	}
	return strings.Join(lines, "\n") + "\n"
}

func formatImpactPreview(draft evolution.SkillDraft, hasProfile bool) string {
	lines := []string{
		"impact_preview:",
		fmt.Sprintf("will_update_existing_skill=%t", hasProfile),
		"expected_effect=" + expectedDraftEffect(draft.ChangeKind),
	}
	return strings.Join(lines, "\n") + "\n"
}

func expectedDraftEffect(changeKind evolution.ChangeKind) string {
	switch changeKind {
	case evolution.ChangeKindCreate:
		return "create a brand-new skill file"
	case evolution.ChangeKindAppend:
		return "append a new section onto the current skill"
	case evolution.ChangeKindReplace:
		return "replace the current skill body with the drafted body"
	case evolution.ChangeKindMerge:
		return "merge the draft into the current skill with an extra merged section"
	default:
		return "apply the drafted change to the target skill"
	}
}

package evolutioncmd

import (
	"errors"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/sipeed/picoclaw/pkg/evolution"
)

func newApplyCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "apply <draft-id>",
		Short: "Apply one draft to the current workspace",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, store, workspace, err := loadEvolutionDeps()
			if err != nil {
				return err
			}

			drafts, err := store.LoadDrafts()
			if err != nil {
				return err
			}

			paths := evolution.NewPaths(workspace, cfg.Evolution.StateDir)
			index, draft, err := findWorkspaceDraft(drafts, paths, workspace, args[0])
			if err != nil {
				return err
			}

			now := time.Now().UTC()
			applier := evolution.NewApplier(paths, nil)
			if err := applier.ApplyDraft(cmd.Context(), workspace, draft); err != nil {
				drafts[index].Status = evolution.DraftStatusQuarantined
				drafts[index].UpdatedAt = timePtr(now)
				drafts[index].ScanFindings = appendUniqueStrings(drafts[index].ScanFindings, fmt.Sprintf("manual apply failed: %v", err))
				if saveErr := store.SaveDrafts(drafts); saveErr != nil {
					return errors.Join(err, saveErr)
				}
				return err
			}

			drafts[index].Status = evolution.DraftStatusAccepted
			drafts[index].UpdatedAt = timePtr(now)
			drafts[index].ReviewNotes = appendUniqueStrings(drafts[index].ReviewNotes, "manually applied via CLI")
			if err := store.SaveDrafts(drafts); err != nil {
				return err
			}
			if err := evolution.SaveAppliedProfile(store, workspace, draft, now); err != nil {
				return err
			}
			if err := annotateManualApplyProfile(store, draft.TargetSkillName, draft.ChangeKind); err != nil {
				return err
			}

			_, err = fmt.Fprintf(cmd.OutOrStdout(), "applied draft=%s target=%s\n", draft.ID, draft.TargetSkillName)
			return err
		},
	}
}

func annotateManualApplyProfile(store *evolution.Store, skillName string, changeKind evolution.ChangeKind) error {
	profile, err := store.LoadProfile(skillName)
	if err != nil {
		return err
	}
	if len(profile.VersionHistory) == 0 {
		return nil
	}
	last := len(profile.VersionHistory) - 1
	profile.VersionHistory[last].Action = "manual_apply:" + string(changeKind)
	profile.VersionHistory[last].Summary = "manual CLI apply: " + profile.VersionHistory[last].Summary
	return store.SaveProfile(profile)
}

package evolutioncmd

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"

	"github.com/sipeed/picoclaw/pkg/evolution"
)

func newDraftsCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "drafts",
		Short: "List skill drafts for the current workspace",
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

			filtered := make([]evolution.SkillDraft, 0, len(drafts))
			for _, draft := range drafts {
				if draftBelongsToWorkspace(paths, workspace, draft) {
					filtered = append(filtered, draft)
				}
			}
			sort.Slice(filtered, func(i, j int) bool {
				if filtered[i].CreatedAt.Equal(filtered[j].CreatedAt) {
					return filtered[i].ID < filtered[j].ID
				}
				return filtered[i].CreatedAt.Before(filtered[j].CreatedAt)
			})

			for _, draft := range filtered {
				if _, err := fmt.Fprintf(
					cmd.OutOrStdout(),
					"id=%s status=%s target=%s type=%s change=%s summary=%s\n",
					draft.ID,
					draft.Status,
					draft.TargetSkillName,
					draft.DraftType,
					draft.ChangeKind,
					draft.HumanSummary,
				); err != nil {
					return err
				}
			}
			return nil
		},
	}
}

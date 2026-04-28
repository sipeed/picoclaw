package evolutioncmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/sipeed/picoclaw/pkg/evolution"
)

func newPruneCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "prune",
		Short: "Recompute lifecycle states for learned skills",
		RunE: func(_ *cobra.Command, _ []string) error {
			cfg, store, workspace, err := loadEvolutionDeps()
			if err != nil {
				return err
			}

			profiles, err := store.LoadProfiles()
			if err != nil {
				return err
			}

			now := time.Now()
			paths := evolution.NewPaths(workspace, cfg.Evolution.StateDir)
			for _, profile := range profiles {
				if !profileBelongsToWorkspace(paths, workspace, profile) {
					continue
				}

				next := evolution.NextLifecycleState(profile, now)
				if next != profile.Status {
					if err := evolution.ApplyLifecycleState(paths, profile, next); err != nil {
						return err
					}
					profile.VersionHistory = append(profile.VersionHistory, evolution.SkillVersionEntry{
						Version:   profile.CurrentVersion,
						Action:    "lifecycle:" + string(next),
						Timestamp: now,
						Summary:   fmt.Sprintf("lifecycle transition: %s -> %s", profile.Status, next),
					})
					profile.Status = next
				}
				if err := store.SaveProfile(profile); err != nil {
					return err
				}
			}
			return nil
		},
	}
}

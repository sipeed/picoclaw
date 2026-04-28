package evolutioncmd

import (
	"github.com/spf13/cobra"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/evolution"
	"github.com/sipeed/picoclaw/pkg/providers"
)

var createEvolutionProvider = providers.CreateProvider

func newRunOnceCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "run-once",
		Short: "Run evolution cold path once",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, store, workspace, err := loadEvolutionDeps()
			if err != nil {
				return err
			}

			rt, err := evolution.NewRuntime(evolution.RuntimeOptions{
				Config: cfg.Evolution,
				Store:  store,
				Organizer: evolution.NewOrganizer(evolution.OrganizerOptions{
					MinCaseCount:   cfg.Evolution.MinCaseCount,
					MinSuccessRate: cfg.Evolution.MinSuccessRate,
				}),
				SkillsRecaller: evolution.NewSkillsRecaller(workspace),
				GeneratorFactory: func(workspace string) evolution.DraftGenerator {
					return draftGeneratorForRunOnce(cfg, workspace)
				},
				Applier:        evolution.NewApplier(evolution.NewPaths(workspace, cfg.Evolution.StateDir), nil),
			})
			if err != nil {
				return err
			}

			return rt.RunColdPathOnce(cmd.Context(), workspace)
		},
	}
}

func draftGeneratorForRunOnce(cfg *config.Config, workspace string) evolution.DraftGenerator {
	if cfg == nil {
		return evolution.NewDraftGeneratorForWorkspace(workspace, nil, "")
	}

	provider, modelID, err := createEvolutionProvider(cfg)
	if err != nil {
		return evolution.NewDraftGeneratorForWorkspace(workspace, nil, "")
	}
	return evolution.NewDraftGeneratorForWorkspace(workspace, provider, modelID)
}

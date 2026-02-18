package skills

import (
	"fmt"
	"path/filepath"

	internal2 "github.com/sipeed/picoclaw/cmd/picoclaw/internal"
	"github.com/sipeed/picoclaw/pkg/skills"
	"github.com/spf13/cobra"
)

func NewSkillsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skills",
		Short: "Manage skills (install, list, remove)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	var loaded bool

	cmd.PersistentPreRunE = func(cmd *cobra.Command, _ []string) error {
		cfg, err := internal2.LoadConfig()
		if err != nil {
			return fmt.Errorf("error loading config: %w", err)
		}

		workspace := cfg.WorkspacePath()
		installer := skills.NewSkillInstaller(workspace)

		globalDir := filepath.Dir(internal2.GetConfigPath())
		globalSkillsDir := filepath.Join(globalDir, "skills")
		builtinSkillsDir := filepath.Join(globalDir, "picoclaw", "skills")
		skillsLoader := skills.NewSkillsLoader(workspace, globalSkillsDir, builtinSkillsDir)

		if !loaded {
			cmd.AddCommand(
				newListCommand(skillsLoader),
				newInstallCommand(installer),
				newInstallBuiltinCommand(workspace),
				newListBuiltinCommand(),
				newRemoveCommand(installer),
				newSearchCommand(installer),
				newShowCommand(skillsLoader),
			)
			loaded = true
		}
		return nil
	}

	return cmd
}

package configcmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/sipeed/picoclaw/cmd/picoclaw/internal"
	"github.com/sipeed/picoclaw/pkg/config"
)

func newAgentUpdateCommand() *cobra.Command {
	var (
		name      string
		model     string
		workspace string
		defaultAgent bool
	)

	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update the first matching agent in agents.list",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runAgentUpdate(args[0], name, model, workspace, defaultAgent)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Display name")
	cmd.Flags().StringVar(&model, "model", "", "Model name (from model_list)")
	cmd.Flags().StringVar(&workspace, "workspace", "", "Workspace path")
	cmd.Flags().BoolVar(&defaultAgent, "default", false, "Set as default agent")

	return cmd
}

func runAgentUpdate(id, name, model, workspace string, defaultAgent bool) error {
	cfg, err := internal.LoadConfig()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	configPath := internal.GetConfigPath()
	if _, err := os.Stat(configPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("config not found; run: picoclaw onboard")
		}
		return err
	}

	idx := -1
	for i := range cfg.Agents.List {
		if cfg.Agents.List[i].ID == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		return fmt.Errorf("no agent with id %q", id)
	}

	entry := &cfg.Agents.List[idx]

	if name != "" {
		entry.Name = name
	}
	if model != "" {
		if entry.Model == nil {
			entry.Model = &config.AgentModelConfig{}
		}
		entry.Model.Primary = model
	}
	if workspace != "" {
		entry.Workspace = workspace
	}
	if defaultAgent {
		entry.Default = true
	}

	if err := config.SaveConfig(configPath, cfg); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	fmt.Printf("Updated agent %q in agents.list.\n", id)
	return nil
}

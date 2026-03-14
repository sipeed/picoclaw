package configcmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/sipeed/picoclaw/cmd/picoclaw/internal"
	"github.com/sipeed/picoclaw/pkg/config"
)

func newAgentAddCommand() *cobra.Command {
	var (
		name      string
		model     string
		workspace string
		defaultAgent bool
	)

	cmd := &cobra.Command{
		Use:   "add <id>",
		Short: "Add an agent to agents.list",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runAgentAdd(args[0], name, model, workspace, defaultAgent)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Display name")
	cmd.Flags().StringVar(&model, "model", "", "Model name (from model_list)")
	cmd.Flags().StringVar(&workspace, "workspace", "", "Workspace path")
	cmd.Flags().BoolVar(&defaultAgent, "default", false, "Set as default agent")

	return cmd
}

func runAgentAdd(id, name, model, workspace string, defaultAgent bool) error {
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

	// Interactive prompt when TTY and missing fields
	if IsTTY() {
		if name == "" {
			name, _ = Prompt("Name (display): ")
		}
		if model == "" {
			model, _ = Prompt("Model (model_name from model_list): ")
		}
		if workspace == "" {
			workspace, _ = Prompt("Workspace: ")
		}
	}

	entry := config.AgentConfig{
		ID:        id,
		Default:   defaultAgent,
		Name:      name,
		Workspace: workspace,
	}
	if model != "" {
		entry.Model = &config.AgentModelConfig{Primary: model}
	}

	cfg.Agents.List = append(cfg.Agents.List, entry)

	if err := config.SaveConfig(configPath, cfg); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	fmt.Printf("Added agent %q to agents.list.\n", id)
	return nil
}

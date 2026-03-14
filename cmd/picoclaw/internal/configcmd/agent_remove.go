package configcmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/sipeed/picoclaw/cmd/picoclaw/internal"
	"github.com/sipeed/picoclaw/pkg/config"
)

func newAgentRemoveCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <id>",
		Short: "Remove an agent from agents.list by id",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runAgentRemove(args[0])
		},
	}
}

func runAgentRemove(id string) error {
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

	var kept []config.AgentConfig
	for _, a := range cfg.Agents.List {
		if a.ID != id {
			kept = append(kept, a)
		}
	}

	if len(kept) == len(cfg.Agents.List) {
		return fmt.Errorf("no agent with id %q", id)
	}

	cfg.Agents.List = kept
	if err := config.SaveConfig(configPath, cfg); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	fmt.Printf("Removed agent %q from agents.list.\n", id)
	return nil
}

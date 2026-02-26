package configcmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/sipeed/picoclaw/cmd/picoclaw/internal"
)

func newAgentListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all agents in agents.list",
		Args:  cobra.NoArgs,
		RunE:  runAgentList,
	}
}

func runAgentList(_ *cobra.Command, _ []string) error {
	cfg, err := internal.LoadConfig()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	configPath := internal.GetConfigPath()
	if _, err := os.Stat(configPath); err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No config file found. Run: picoclaw onboard")
			return nil
		}
		return err
	}

	if len(cfg.Agents.List) == 0 {
		fmt.Println("agents.list is empty.")
		return nil
	}

	fmt.Printf("%-20s %-25s %-20s %s\n", "ID", "NAME", "MODEL", "WORKSPACE")
	fmt.Println(strings.Repeat("-", 85))

	for _, a := range cfg.Agents.List {
		model := ""
		if a.Model != nil && a.Model.Primary != "" {
			model = a.Model.Primary
		}
		if len(model) > 18 {
			model = model[:15] + "..."
		}
		name := a.Name
		if len(name) > 23 {
			name = name[:20] + "..."
		}
		id := a.ID
		if len(id) > 18 {
			id = id[:15] + "..."
		}
		ws := a.Workspace
		if len(ws) > 35 {
			ws = ws[:32] + "..."
		}
		fmt.Printf("%-20s %-25s %-20s %s\n", id, name, model, ws)
	}

	return nil
}

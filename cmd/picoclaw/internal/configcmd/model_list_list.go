package configcmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/sipeed/picoclaw/cmd/picoclaw/internal"
	"github.com/sipeed/picoclaw/pkg/config"
)

func newModelListListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all models in model_list",
		Args:  cobra.NoArgs,
		RunE:  runModelListList,
	}
}

func runModelListList(_ *cobra.Command, _ []string) error {
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

	if len(cfg.ModelList) == 0 {
		fmt.Println("model_list is empty.")
		return nil
	}

	// Table header
	fmt.Printf("%-20s %-35s %-40s %s\n", "MODEL_NAME", "MODEL", "API_BASE", "AUTH")
	fmt.Println(strings.Repeat("-", 100))

	for _, m := range cfg.ModelList {
		auth := authSummary(m)
		apiBase := m.APIBase
		if len(apiBase) > 38 {
			apiBase = apiBase[:35] + "..."
		}
		model := m.Model
		if len(model) > 33 {
			model = model[:30] + "..."
		}
		modelName := m.ModelName
		if len(modelName) > 18 {
			modelName = modelName[:15] + "..."
		}
		fmt.Printf("%-20s %-35s %-40s %s\n", modelName, model, apiBase, auth)
	}

	return nil
}

func authSummary(m config.ModelConfig) string {
	if m.AuthMethod != "" {
		return m.AuthMethod
	}
	if m.TokenURL != "" {
		return "litellm (keycloak)"
	}
	if m.APIKey != "" {
		return "api_key"
	}
	return "-"
}

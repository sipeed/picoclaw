package configcmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/sipeed/picoclaw/cmd/picoclaw/internal"
	"github.com/sipeed/picoclaw/pkg/config"
)

// agentDefaultsKeys in display order for get (all).
var agentDefaultsKeys = []string{
	"workspace", "restrict_to_workspace", "provider", "model_name", "model",
	"model_fallbacks", "image_model", "image_model_fallbacks",
	"max_tokens", "temperature", "max_tool_iterations",
}

var agentDefaultsKeySet map[string]bool

func init() {
	agentDefaultsKeySet = make(map[string]bool, len(agentDefaultsKeys))
	for _, k := range agentDefaultsKeys {
		agentDefaultsKeySet[k] = true
	}
}

func newAgentDefaultsGetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "get [key]",
		Short: "Get agents.defaults (all fields or one key)",
		Args:  cobra.MaximumNArgs(1),
		RunE:  runAgentDefaultsGet,
	}
}

func runAgentDefaultsGet(_ *cobra.Command, args []string) error {
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

	d := &cfg.Agents.Defaults

	if len(args) == 0 {
		for _, key := range agentDefaultsKeys {
			value := agentDefaultsGetValue(d, key)
			fmt.Printf("%s: %s\n", key, value)
		}
		return nil
	}

	key := args[0]
	if !agentDefaultsKeySet[key] {
		return fmt.Errorf("invalid key %q; allowed: %s", key, strings.Join(agentDefaultsKeys, ", "))
	}
	fmt.Println(agentDefaultsGetValue(d, key))
	return nil
}

func agentDefaultsGetValue(d *config.AgentDefaults, key string) string {
	switch key {
	case "workspace":
		return d.Workspace
	case "restrict_to_workspace":
		if d.RestrictToWorkspace {
			return "true"
		}
		return "false"
	case "provider":
		return d.Provider
	case "model_name":
		return d.ModelName
	case "model":
		return d.Model
	case "model_fallbacks":
		return strings.Join(d.ModelFallbacks, ",")
	case "image_model":
		return d.ImageModel
	case "image_model_fallbacks":
		return strings.Join(d.ImageModelFallbacks, ",")
	case "max_tokens":
		return fmt.Sprintf("%d", d.MaxTokens)
	case "temperature":
		if d.Temperature == nil {
			return ""
		}
		return fmt.Sprintf("%g", *d.Temperature)
	case "max_tool_iterations":
		return fmt.Sprintf("%d", d.MaxToolIterations)
	default:
		return ""
	}
}

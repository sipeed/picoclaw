package configcmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/sipeed/picoclaw/cmd/picoclaw/internal"
	"github.com/sipeed/picoclaw/pkg/config"
)

func newAgentDefaultsSetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a single field in agents.defaults",
		Args:  cobra.ExactArgs(2),
		RunE:  runAgentDefaultsSet,
	}
}

func runAgentDefaultsSet(_ *cobra.Command, args []string) error {
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

	key, value := args[0], args[1]
	if !agentDefaultsKeySet[key] {
		return fmt.Errorf("invalid key %q; allowed: %s", key, strings.Join(agentDefaultsKeys, ", "))
	}

	d := &cfg.Agents.Defaults

	switch key {
	case "workspace":
		d.Workspace = value
	case "restrict_to_workspace":
		b, err := parseBool(value)
		if err != nil {
			return fmt.Errorf("restrict_to_workspace: %w", err)
		}
		d.RestrictToWorkspace = b
	case "provider":
		d.Provider = value
	case "model_name":
		d.ModelName = value
	case "model":
		d.Model = value
	case "model_fallbacks":
		if value == "" {
			d.ModelFallbacks = nil
		} else {
			d.ModelFallbacks = strings.Split(value, ",")
			for i := range d.ModelFallbacks {
				d.ModelFallbacks[i] = strings.TrimSpace(d.ModelFallbacks[i])
			}
		}
	case "image_model":
		d.ImageModel = value
	case "image_model_fallbacks":
		if value == "" {
			d.ImageModelFallbacks = nil
		} else {
			d.ImageModelFallbacks = strings.Split(value, ",")
			for i := range d.ImageModelFallbacks {
				d.ImageModelFallbacks[i] = strings.TrimSpace(d.ImageModelFallbacks[i])
			}
		}
	case "max_tokens":
		n, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("max_tokens: %w", err)
		}
		d.MaxTokens = n
	case "temperature":
		if value == "" {
			d.Temperature = nil
		} else {
			f, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return fmt.Errorf("temperature: %w", err)
			}
			d.Temperature = &f
		}
	case "max_tool_iterations":
		n, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("max_tool_iterations: %w", err)
		}
		d.MaxToolIterations = n
	}

	if err := config.SaveConfig(configPath, cfg); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}
	fmt.Printf("Set agents.defaults %s.\n", key)
	return nil
}

func parseBool(s string) (bool, error) {
	switch strings.ToLower(s) {
	case "true", "1", "yes":
		return true, nil
	case "false", "0", "no":
		return false, nil
	default:
		return false, fmt.Errorf("expected true/false, got %q", s)
	}
}

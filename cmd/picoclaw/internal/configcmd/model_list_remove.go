package configcmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/sipeed/picoclaw/cmd/picoclaw/internal"
	"github.com/sipeed/picoclaw/pkg/config"
)

func newModelListRemoveCommand() *cobra.Command {
	var first bool

	cmd := &cobra.Command{
		Use:   "remove <model_name>",
		Short: "Remove model(s) from model_list by model_name",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runModelListRemove(args[0], first)
		},
	}

	cmd.Flags().BoolVar(&first, "first", false, "Remove only the first matching entry (default: remove all)")

	return cmd
}

func runModelListRemove(modelName string, firstOnly bool) error {
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

	var kept []config.ModelConfig
	removedFirst := false
	for _, m := range cfg.ModelList {
		if m.ModelName == modelName {
			if firstOnly {
				if !removedFirst {
					removedFirst = true
					continue
				}
			} else {
				continue
			}
		}
		kept = append(kept, m)
	}

	removed := len(cfg.ModelList) - len(kept)
	if removed == 0 {
		return fmt.Errorf("no model with model_name %q", modelName)
	}

	cfg.ModelList = kept
	if err := config.SaveConfig(configPath, cfg); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	fmt.Printf("Removed %d model(s) %q from model_list.\n", removed, modelName)
	return nil
}

package model

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/sipeed/picoclaw/cmd/picoclaw/internal"
	"github.com/sipeed/picoclaw/pkg/config"
)

// LocalModel is a special model name that indicates that the model is local and with or without api_key.
const LocalModel = "local-model"

func NewModelCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "model [model_name]",
		Short: "Show or change the default model",
		Long: `Show or change the default model configuration.

If no argument is provided, shows the current default model.
If a model name is provided, sets it as the default model.

To onboard a model from a custom OpenAI-compatible endpoint (fetch the
available list online and pick one), use the 'add' subcommand:

  picoclaw model add --help

Examples:
  picoclaw model                    # Show current default model
  picoclaw model gpt-5.2           # Set gpt-5.2 as default
  picoclaw model claude-sonnet-4.6 # Set claude-sonnet-4.6 as default
  picoclaw model local-model       # Set local vLLM/OVMS server as default
  picoclaw model add -b URL -k KEY # Add a model from a custom endpoint

Note: 'local-model' is a special value for using a local OpenAI-compatible server
(running at localhost:8000 by default) which does not require an API key.
It can be configured to point to vLLM, OVMS (OpenVINO Model Server), or other OpenAI-compatible servers.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath := internal.GetConfigPath()

			// Load current config
			cfg, err := config.LoadConfig(configPath)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			if len(args) == 0 {
				// Show current default model
				showCurrentModel(cfg)
				return nil
			}

			// Set new default model
			modelName := args[0]
			return setDefaultModel(configPath, cfg, modelName)
		},
	}

	cmd.AddCommand(newAddCommand())

	return cmd
}

func showCurrentModel(cfg *config.Config) {
	defaultModel := cfg.Agents.Defaults.ModelName

	if defaultModel == "" {
		fmt.Println("No default model is currently set.")
		fmt.Println("\nAvailable models in your config:")
		listAvailableModels(cfg)
	} else {
		fmt.Printf("Current default model: %s\n", defaultModel)
		fmt.Println("\nAvailable models in your config:")
		listAvailableModels(cfg)
	}

	fmt.Println("\nTip: 'picoclaw model add -b URL -k KEY' adds a model from a custom")
	fmt.Println("     OpenAI-compatible endpoint (see 'picoclaw model add --help').")
}

func listAvailableModels(cfg *config.Config) {
	if len(cfg.ModelList) == 0 {
		fmt.Println("  No models configured in model_list")
		return
	}

	defaultModel := cfg.Agents.Defaults.ModelName

	for _, model := range cfg.ModelList {
		marker := "  "
		if model.ModelName == defaultModel {
			marker = "> "
		}
		if !model.Enabled {
			continue
		}
		fmt.Printf("%s- %s (%s)\n", marker, model.ModelName, model.Model)
	}
}

func setDefaultModel(configPath string, cfg *config.Config, modelName string) error {
	// Validate that the model exists in model_list
	modelFound := false
	for _, model := range cfg.ModelList {
		if model.Enabled && model.ModelName == modelName {
			modelFound = true
			break
		}
	}

	if !modelFound && modelName != LocalModel {
		return fmt.Errorf("cannot found model '%s' in config", modelName)
	}

	// Update the default model
	// Clear old model field and set new model_name
	oldModel := cfg.Agents.Defaults.ModelName

	cfg.Agents.Defaults.ModelName = modelName

	// Save config back to file
	if err := config.SaveConfig(configPath, cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("✓ Default model changed from '%s' to '%s'\n",
		formatModelName(oldModel), modelName)
	fmt.Println("\nThe new default model will be used for all agent interactions.")

	return nil
}

func formatModelName(name string) string {
	if name == "" {
		return "(none)"
	}
	return name
}

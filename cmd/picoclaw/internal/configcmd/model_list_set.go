package configcmd

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/sipeed/picoclaw/cmd/picoclaw/internal"
	"github.com/sipeed/picoclaw/pkg/config"
)

func newModelListSetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "set <model_name> <key> <value>",
		Short: "Set a single field for a model in model_list",
		Args:  cobra.ExactArgs(3),
		RunE:  runModelListSet,
	}
}

func runModelListSet(_ *cobra.Command, args []string) error {
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

	modelName := args[0]
	key := args[1]
	value := args[2]

	if !isModelConfigKey(key) {
		return fmt.Errorf("invalid key %q; allowed: %s", key, allowedModelConfigKeysString())
	}

	idx, err := findModelIndex(cfg, modelName)
	if err != nil {
		return err
	}
	entry := &cfg.ModelList[idx]

	if isIntModelConfigKey(key) {
		n, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("key %q requires an integer: %w", key, err)
		}
		entry.RPM = n
	} else {
		modelConfigSetString(entry, key, value)
	}

	if err := entry.Validate(); err != nil {
		return err
	}

	if err := config.SaveConfig(configPath, cfg); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	fmt.Printf("Set %s for model %q.\n", key, entry.ModelName)
	return nil
}

func modelConfigSetString(m *config.ModelConfig, key, value string) {
	switch key {
	case "model_name":
		m.ModelName = value
	case "model":
		m.Model = value
	case "api_base":
		m.APIBase = value
	case "api_key":
		m.APIKey = value
	case "proxy":
		m.Proxy = value
	case "auth_method":
		m.AuthMethod = value
	case "connect_mode":
		m.ConnectMode = value
	case "workspace":
		m.Workspace = value
	case "token_url":
		m.TokenURL = value
	case "client_id":
		m.ClientID = value
	case "client_secret":
		m.ClientSecret = value
	case "max_tokens_field":
		m.MaxTokensField = value
	}
}

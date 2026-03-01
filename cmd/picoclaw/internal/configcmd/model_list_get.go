package configcmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/sipeed/picoclaw/cmd/picoclaw/internal"
	"github.com/sipeed/picoclaw/pkg/config"
)

func newModelListGetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "get <model_name> [key]",
		Short: "Get one model's config or a single field",
		Args:  cobra.MatchAll(cobra.MinimumNArgs(1), cobra.MaximumNArgs(2)),
		RunE:  runModelListGet,
	}
}

func runModelListGet(_ *cobra.Command, args []string) error {
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
	idx, err := findModelIndex(cfg, modelName)
	if err != nil {
		return err
	}
	entry := &cfg.ModelList[idx]

	if len(args) == 1 {
		// No key: print all common fields (mask secrets)
		for _, key := range modelConfigKeys {
			value, mask := modelConfigGetValue(entry, key)
			if mask && value != "" {
				value = "***"
			}
			fmt.Printf("%s: %s\n", key, value)
		}
		return nil
	}

	key := args[1]
	if !isModelConfigKey(key) {
		return fmt.Errorf("invalid key %q; allowed: %s", key, allowedModelConfigKeysString())
	}
	value, _ := modelConfigGetValue(entry, key)
	fmt.Println(value)
	return nil
}

// modelConfigGetValue returns the string value for key and whether it should be masked in "get all" output.
func modelConfigGetValue(m *config.ModelConfig, key string) (string, bool) {
	switch key {
	case "model_name":
		return m.ModelName, false
	case "model":
		return m.Model, false
	case "api_base":
		return m.APIBase, false
	case "api_key":
		return m.APIKey, true
	case "proxy":
		return m.Proxy, false
	case "auth_method":
		return m.AuthMethod, false
	case "connect_mode":
		return m.ConnectMode, false
	case "workspace":
		return m.Workspace, false
	case "token_url":
		return m.TokenURL, false
	case "client_id":
		return m.ClientID, false
	case "client_secret":
		return m.ClientSecret, true
	case "max_tokens_field":
		return m.MaxTokensField, false
	case "rpm":
		if m.RPM == 0 {
			return "0", false
		}
		return fmt.Sprintf("%d", m.RPM), false
	default:
		return "", false
	}
}

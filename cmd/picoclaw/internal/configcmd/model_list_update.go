package configcmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/sipeed/picoclaw/cmd/picoclaw/internal"
	"github.com/sipeed/picoclaw/pkg/config"
)

func newModelListUpdateCommand() *cobra.Command {
	var (
		model        string
		apiBase      string
		apiKey       string
		proxy        string
		authMethod   string
		maxTokensFld string
		tokenURL     string
		clientID     string
		clientSecret string
	)

	cmd := &cobra.Command{
		Use:   "update <model_name>",
		Short: "Update the first matching model in model_list",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runModelListUpdate(args[0], model, apiBase, apiKey, proxy, authMethod, maxTokensFld, tokenURL, clientID, clientSecret)
		},
	}

	cmd.Flags().StringVar(&model, "model", "", "Protocol/model (e.g. litellm/qwen-turbo)")
	cmd.Flags().StringVar(&apiBase, "api-base", "", "API base URL")
	cmd.Flags().StringVar(&apiKey, "api-key", "", "API key")
	cmd.Flags().StringVar(&proxy, "proxy", "", "HTTP proxy URL")
	cmd.Flags().StringVar(&authMethod, "auth-method", "", "Auth method: oauth, token")
	cmd.Flags().StringVar(&maxTokensFld, "max-tokens-field", "", "Field name for max tokens")
	cmd.Flags().StringVar(&tokenURL, "token-url", "", "Keycloak token URL (for litellm)")
	cmd.Flags().StringVar(&clientID, "client-id", "", "Client ID (for litellm)")
	cmd.Flags().StringVar(&clientSecret, "client-secret", "", "Client secret (for litellm)")

	return cmd
}

func runModelListUpdate(modelName, model, apiBase, apiKey, proxy, authMethod, maxTokensFld, tokenURL, clientID, clientSecret string) error {
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

	var idx int = -1
	for i := range cfg.ModelList {
		if cfg.ModelList[i].ModelName == modelName {
			idx = i
			break
		}
	}
	if idx < 0 {
		return fmt.Errorf("no model with model_name %q", modelName)
	}

	entry := &cfg.ModelList[idx]

	if model != "" {
		entry.Model = model
	}
	if apiBase != "" {
		entry.APIBase = apiBase
	}
	if apiKey != "" {
		entry.APIKey = apiKey
	}
	if proxy != "" {
		entry.Proxy = proxy
	}
	if authMethod != "" {
		entry.AuthMethod = authMethod
	}
	if maxTokensFld != "" {
		entry.MaxTokensField = maxTokensFld
	}
	if tokenURL != "" {
		entry.TokenURL = tokenURL
	}
	if clientID != "" {
		entry.ClientID = clientID
	}
	if clientSecret != "" {
		entry.ClientSecret = clientSecret
	}

	isLiteLLM := strings.HasPrefix(strings.ToLower(entry.Model), "litellm/")
	if IsTTY() && isLiteLLM {
		if entry.APIBase == "" {
			v, _ := Prompt("API base URL: ")
			entry.APIBase = v
		}
		if entry.TokenURL == "" {
			v, _ := Prompt("Token URL (Keycloak): ")
			entry.TokenURL = v
		}
		if entry.ClientID == "" {
			v, _ := Prompt("Client ID: ")
			entry.ClientID = v
		}
		if entry.ClientSecret == "" {
			v, _ := Prompt("Client secret: ")
			entry.ClientSecret = v
		}
	}

	if isLiteLLM && (entry.APIBase == "" || entry.TokenURL == "" || entry.ClientID == "" || entry.ClientSecret == "") {
		return fmt.Errorf("litellm requires api_base, token_url, client_id, client_secret")
	}

	if err := entry.Validate(); err != nil {
		return err
	}

	if err := config.SaveConfig(configPath, cfg); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	fmt.Printf("Updated model %q in model_list.\n", modelName)
	return nil
}

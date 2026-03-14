package configcmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/sipeed/picoclaw/cmd/picoclaw/internal"
	"github.com/sipeed/picoclaw/pkg/config"
)

func newModelListAddCommand() *cobra.Command {
	var (
		modelName     string
		model         string
		apiBase       string
		apiKey        string
		proxy         string
		authMethod    string
		maxTokensFld  string
		tokenURL      string
		clientID      string
		clientSecret  string
	)

	cmd := &cobra.Command{
		Use:   "add [model_name]",
		Short: "Add a model to model_list",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) > 0 && modelName == "" {
				modelName = args[0]
			}
			return runModelListAdd(modelName, model, apiBase, apiKey, proxy, authMethod, maxTokensFld, tokenURL, clientID, clientSecret)
		},
	}

	cmd.Flags().StringVar(&modelName, "model-name", "", "User-facing model name (e.g. qwen-turbo)")
	cmd.Flags().StringVar(&model, "model", "", "Protocol/model (e.g. litellm/qwen-turbo, openai/gpt-4o)")
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

func runModelListAdd(modelName, model, apiBase, apiKey, proxy, authMethod, maxTokensFld, tokenURL, clientID, clientSecret string) error {
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

	isLiteLLM := strings.HasPrefix(strings.ToLower(model), "litellm/")

	// Interactive prompt for missing required fields when TTY
	if IsTTY() {
		if modelName == "" {
			modelName, _ = Prompt("Model name (e.g. qwen-turbo): ")
		}
		if model == "" {
			model, _ = Prompt("Model (e.g. litellm/qwen-turbo or openai/gpt-4o): ")
			model = strings.TrimSpace(model)
			isLiteLLM = strings.HasPrefix(strings.ToLower(model), "litellm/")
		}
		if apiBase == "" {
			apiBase, _ = Prompt("API base URL: ")
		}
		if isLiteLLM {
			if tokenURL == "" {
				tokenURL, _ = Prompt("Token URL (Keycloak): ")
			}
			if clientID == "" {
				clientID, _ = Prompt("Client ID: ")
			}
			if clientSecret == "" {
				clientSecret, _ = Prompt("Client secret: ")
			}
		}
	}

	// Validate required
	if modelName == "" {
		return fmt.Errorf("model_name is required")
	}
	if model == "" {
		return fmt.Errorf("model is required (e.g. litellm/qwen-turbo)")
	}
	if isLiteLLM {
		if apiBase == "" || tokenURL == "" || clientID == "" || clientSecret == "" {
			return fmt.Errorf("litellm requires api_base, token_url, client_id, client_secret")
		}
	}

	entry := config.ModelConfig{
		ModelName:      modelName,
		Model:          model,
		APIBase:        apiBase,
		APIKey:         apiKey,
		Proxy:          proxy,
		AuthMethod:     authMethod,
		MaxTokensField: maxTokensFld,
		TokenURL:       tokenURL,
		ClientID:       clientID,
		ClientSecret:   clientSecret,
	}

	cfg.ModelList = append(cfg.ModelList, entry)
	if err := config.SaveConfig(configPath, cfg); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	fmt.Printf("Added model %q to model_list.\n", modelName)
	return nil
}

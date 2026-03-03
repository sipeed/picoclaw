package initcmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/sipeed/picoclaw/cmd/picoclaw/internal"
	"github.com/sipeed/picoclaw/pkg/config"
)

func NewInitCommand() *cobra.Command {
	var baseURL, apiKey, model string

	cmd := &cobra.Command{
		Use:   "init [auth <provider>]",
		Short: "Quick setup 鈥?API key or OAuth login",
		Long: `Initialize picoclaw with minimal configuration.

Two modes:

  1. API Key mode (most providers):
     picoclaw init --api-key <key> --model <model> [--base-url <url>]

  2. OAuth mode (OpenAI, Google Antigravity):
     picoclaw init auth openai
     picoclaw init auth google-antigravity
     picoclaw init auth anthropic  (paste token)

In API Key mode, only api-key is required. Model defaults to gpt-4o,
base-url defaults to https://api.openai.com/v1.`,
		Example: `  picoclaw init --api-key sk-xxx --model gpt-4o
  picoclaw init --base-url https://api.deepseek.com/v1 --api-key sk-xxx --model deepseek-chat
  picoclaw init auth openai
  picoclaw init  (interactive)`,
		Args: cobra.MaximumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			ensureConfigDir()
			runInit(cmd, baseURL, apiKey, model)
		},
	}

	cmd.Flags().StringVar(&baseURL, "base-url", "", "API base URL (default: https://api.openai.com/v1)")
	cmd.Flags().StringVar(&apiKey, "api-key", "", "API key")
	cmd.Flags().StringVar(&model, "model", "", "Model name")

	// Add auth subcommand.
	cmd.AddCommand(newInitAuthCommand())

	return cmd
}

func newInitAuthCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "auth <provider>",
		Short: "Initialize via OAuth or token (openai, anthropic, google-antigravity)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ensureConfigDir()
			return runInitAuth(args[0])
		},
	}
}

func ensureConfigDir() {
	configPath := internal.GetConfigPath()
	dir := filepath.Dir(configPath)
	os.MkdirAll(dir, 0755)
}

func runInit(cmd *cobra.Command, baseURL, apiKey, model string) {
	reader := bufio.NewReader(os.Stdin)

	if apiKey == "" {
		fmt.Print("API Key: ")
		apiKey, _ = reader.ReadString('\n')
		apiKey = strings.TrimSpace(apiKey)
	}
	if apiKey == "" {
		fmt.Println("API key is required.")
		fmt.Println("   Or use OAuth: picoclaw init auth openai")
		os.Exit(1)
	}

	if model == "" {
		fmt.Print("Model (default: gpt-4o): ")
		model, _ = reader.ReadString('\n')
		model = strings.TrimSpace(model)
		if model == "" {
			model = "gpt-4o"
		}
	}

	if baseURL == "" {
		fmt.Print("API Base URL (default: https://api.openai.com/v1): ")
		baseURL, _ = reader.ReadString('\n')
		baseURL = strings.TrimSpace(baseURL)
		if baseURL == "" {
			baseURL = "https://api.openai.com/v1"
		}
	}

	protocol := detectProtocol(baseURL)
	modelID := protocol + "/" + model

	saveAndPrint(cmd, model, modelID, baseURL, apiKey)
}

func runInitAuth(provider string) error {
	switch provider {
	case "openai", "anthropic", "google-antigravity", "antigravity":
		// Ensure base config exists before auth writes to it.
		configPath := internal.GetConfigPath()
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			// Create minimal base config so auth can append to it.
			defaults := config.DefaultConfig()
			cfg := &config.Config{
				Agents: config.AgentsConfig{
					Defaults: config.AgentDefaults{
						Workspace:           defaults.Agents.Defaults.Workspace,
						RestrictToWorkspace: true,
						MaxTokens:           32768,
						MaxToolIterations:   50,
					},
				},
				Gateway: defaults.Gateway,
				Tools: config.ToolsConfig{
					Exec: config.ExecConfig{EnableDenyPatterns: true},
					Web: config.WebToolsConfig{
						DuckDuckGo: config.DuckDuckGoConfig{Enabled: true, MaxResults: 5},
					},
				},
			}
			os.MkdirAll(defaults.Agents.Defaults.Workspace, 0755)
			if err := config.SaveConfig(configPath, cfg); err != nil {
				return fmt.Errorf("failed to create base config: %w", err)
			}
			fmt.Printf("Created base config at %s\n\n", configPath)
		}

		// Delegate to the existing auth command logic.
		fmt.Printf("Run: picoclaw auth login --provider %s\n", provider)
		fmt.Println("This will open a browser or prompt for your token.")
		return nil
	default:
		return fmt.Errorf("unsupported auth provider: %s\nSupported: openai, anthropic, google-antigravity", provider)
	}
}


func saveAndPrint(cmd *cobra.Command, model, modelID, baseURL, apiKey string) {
	configPath := internal.GetConfigPath()

	if _, err := os.Stat(configPath); err == nil {
		fmt.Printf("Config exists at %s. Overwrite? (y/n): ", configPath)
		var resp string
		fmt.Scanln(&resp)
		if resp != "y" {
			fmt.Println("Aborted.")
			return
		}
	}

	defaults := config.DefaultConfig()
	workspace := defaults.Agents.Defaults.Workspace

	cfgMap := map[string]any{
		"agents": map[string]any{
			"defaults": map[string]any{
				"workspace":            workspace,
				"restrict_to_workspace": true,
				"model_name":           model,
				"max_tokens":           32768,
				"max_tool_iterations":  50,
			},
		},
		"model_list": []map[string]any{
			{
				"model_name": model,
				"model":      modelID,
				"api_base":   baseURL,
				"api_key":    apiKey,
			},
		},
		"gateway": map[string]any{
			"host": defaults.Gateway.Host,
			"port": defaults.Gateway.Port,
		},
		"tools": map[string]any{
			"exec": map[string]any{"enable_deny_patterns": true},
		},
	}

	data, err := json.MarshalIndent(cfgMap, "", "  ")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	os.MkdirAll(filepath.Dir(configPath), 0755)
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		fmt.Printf("Error writing config: %v\n", err)
		os.Exit(1)
	}

	os.MkdirAll(workspace, 0755)

	fmt.Printf("\n%s picoclaw is ready!\n\n", internal.Logo)
	fmt.Printf("  Config:    %s\n", configPath)
	fmt.Printf("  Model:     %s\n", model)
	fmt.Printf("  API Base:  %s\n", baseURL)

	// Test via cobra root command (in-process).
	fmt.Println("\n  Testing: picoclaw agent -m \"Hello!\"")
	fmt.Println(strings.Repeat("\u2500", 50))
	rootCmd := cmd.Root()
	rootCmd.SetArgs([]string{"agent", "-m", "Hello!"})
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(strings.Repeat("\u2500", 50))
		fmt.Printf("  Test FAILED: %v\n", err)
		fmt.Println("  Possible fixes:")
		fmt.Println("    - Check your API key")
		fmt.Println("    - Check the API base URL")
		fmt.Printf("    - Edit: %s\n", configPath)
	} else {
		fmt.Println(strings.Repeat("\u2500", 50))
		fmt.Println("  Test OK!")
	}

	// Next steps.
	fmt.Println("\n  Quick Start:")
	fmt.Println("    picoclaw agent -m \"Hello!\"       # send a message")
	fmt.Println("")
	fmt.Println("  Add Channels (Telegram, Discord, etc):")
	fmt.Printf("    Edit %s\n", configPath)
	fmt.Println("    picoclaw gateway                  # start multi-channel server")
	fmt.Println("")
	fmt.Println("  Docs: https://github.com/sipeed/picoclaw")
	fmt.Println(strings.Repeat("\u2500", 50))
}

// detectProtocol guesses the provider protocol from the API base URL.
func detectProtocol(baseURL string) string {
	lower := strings.ToLower(baseURL)
	switch {
	case strings.Contains(lower, "anthropic"):
		return "anthropic"
	case strings.Contains(lower, "generativelanguage.googleapis"):
		return "gemini"
	case strings.Contains(lower, "dashscope.aliyuncs"):
		return "qwen"
	case strings.Contains(lower, "open.bigmodel.cn"):
		return "zhipu"
	case strings.Contains(lower, "moonshot"):
		return "moonshot"
	case strings.Contains(lower, "deepseek"):
		return "deepseek"
	case strings.Contains(lower, "openrouter"):
		return "openrouter"
	case strings.Contains(lower, "groq"):
		return "groq"
	case strings.Contains(lower, "localhost:11434"):
		return "ollama"
	case strings.Contains(lower, "volcengine") || strings.Contains(lower, "volces.com"):
		return "volcengine"
	case strings.Contains(lower, "cerebras"):
		return "cerebras"
	case strings.Contains(lower, "nvidia") || strings.Contains(lower, "integrate.api"):
		return "nvidia"
	case strings.Contains(lower, "mistral"):
		return "mistral"
	default:
		return "openai"
	}
}
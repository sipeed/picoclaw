// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT

package main

import (
	"bufio"
	"context"
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
)

//go:generate cp -r ../../workspace .
//go:embed workspace
var embeddedFiles embed.FS

// providerChoice holds the details for a user-selected provider
type providerChoice struct {
	name         string
	modelName    string
	needsAPIKey  bool
	keyPrompt    string
	validateURL  string
	validateFunc func(apiKey string) *http.Request
}

var providerChoices = []providerChoice{
	{
		name:      "Ollama",
		modelName: "llama3",
	},
	{
		name:        "OpenRouter",
		modelName:   "openrouter-auto",
		needsAPIKey: true,
		keyPrompt:   "Enter your OpenRouter API key: ",
		validateURL: "https://openrouter.ai/api/v1/models",
		validateFunc: func(apiKey string) *http.Request {
			req, _ := http.NewRequest("GET", "https://openrouter.ai/api/v1/models", nil)
			req.Header.Set("Authorization", "Bearer "+apiKey)
			return req
		},
	},
	{
		name:        "Anthropic",
		modelName:   "claude-sonnet-4.6",
		needsAPIKey: true,
		keyPrompt:   "Enter your Anthropic API key: ",
		validateURL: "https://api.anthropic.com/v1/models",
		validateFunc: func(apiKey string) *http.Request {
			req, _ := http.NewRequest("GET", "https://api.anthropic.com/v1/models", nil)
			req.Header.Set("x-api-key", apiKey)
			req.Header.Set("anthropic-version", "2023-06-01")
			return req
		},
	},
	{
		name:        "OpenAI",
		modelName:   "gpt-5.2",
		needsAPIKey: true,
		keyPrompt:   "Enter your OpenAI API key: ",
		validateURL: "https://api.openai.com/v1/models",
		validateFunc: func(apiKey string) *http.Request {
			req, _ := http.NewRequest("GET", "https://api.openai.com/v1/models", nil)
			req.Header.Set("Authorization", "Bearer "+apiKey)
			return req
		},
	},
	{
		name:        "DeepSeek",
		modelName:   "deepseek-chat",
		needsAPIKey: true,
		keyPrompt:   "Enter your DeepSeek API key: ",
		validateURL: "https://api.deepseek.com/v1/models",
		validateFunc: func(apiKey string) *http.Request {
			req, _ := http.NewRequest("GET", "https://api.deepseek.com/v1/models", nil)
			req.Header.Set("Authorization", "Bearer "+apiKey)
			return req
		},
	},
}

func onboard() {
	for _, arg := range os.Args[2:] {
		switch arg {
		case "--help", "-h":
			fmt.Println("Initialize picoclaw configuration and workspace")
			fmt.Println()
			fmt.Println("Usage: picoclaw onboard")
			fmt.Println()
			fmt.Println("Creates the default config file and workspace directory.")
			fmt.Println("If a config already exists, prompts before overwriting.")
			return
		}
	}

	configPath := getConfigPath()

	if _, err := os.Stat(configPath); err == nil {
		fmt.Printf("Config already exists at %s\n", configPath)
		fmt.Print("Overwrite? (y/n): ")
		var response string
		fmt.Scanln(&response)
		if response != "y" {
			fmt.Println("Aborted.")
			return
		}
	}

	reader := bufio.NewReader(os.Stdin)

	// 1. Show welcome and provider menu
	fmt.Printf("\n%s Welcome to PicoClaw!\n\n", logo)
	fmt.Println("Choose your AI provider:")
	fmt.Println("  1. Ollama (local, free — no API key needed) [default]")
	fmt.Println("  2. OpenRouter (100+ models, one API key)")
	fmt.Println("  3. Anthropic (Claude)")
	fmt.Println("  4. OpenAI (GPT)")
	fmt.Println("  5. DeepSeek")
	fmt.Println("  6. Skip — I'll configure manually")
	fmt.Println()
	fmt.Print("Enter choice [1]: ")

	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		input = "1"
	}

	// 2. Build config based on selection
	cfg := config.DefaultConfig()
	var apiKey string
	var choiceIdx int

	switch input {
	case "1":
		choiceIdx = 0
	case "2":
		choiceIdx = 1
	case "3":
		choiceIdx = 2
	case "4":
		choiceIdx = 3
	case "5":
		choiceIdx = 4
	case "6":
		// Skip — use defaults as-is
		choiceIdx = -1
	default:
		fmt.Printf("Unknown choice %q, using default (Ollama).\n", input)
		choiceIdx = 0
	}

	if choiceIdx >= 0 {
		choice := providerChoices[choiceIdx]
		cfg.Agents.Defaults.Model = choice.modelName

		if choice.needsAPIKey {
			fmt.Print(choice.keyPrompt)
			apiKey, _ = reader.ReadString('\n')
			apiKey = strings.TrimSpace(apiKey)

			// Set the API key on the matching model entry
			for i := range cfg.ModelList {
				if cfg.ModelList[i].ModelName == choice.modelName {
					cfg.ModelList[i].APIKey = apiKey
					break
				}
			}
		}
	}

	// 3. Save config
	if err := config.SaveConfig(configPath, cfg); err != nil {
		fmt.Printf("Error saving config: %v\n", err)
		os.Exit(1)
	}

	// 4. Copy workspace templates
	workspace := cfg.WorkspacePath()
	createWorkspaceTemplates(workspace)

	// 5. Verification
	fmt.Println("\nVerifying setup...")
	fmt.Printf("  [\u2713] Config written to %s\n", configPath)
	fmt.Printf("  [\u2713] Workspace initialized\n")

	if choiceIdx >= 0 {
		choice := providerChoices[choiceIdx]
		if choiceIdx == 0 {
			// Ollama: check if running
			verifyOllama()
		} else if choice.needsAPIKey && apiKey != "" {
			verifyAPIKey(choice, apiKey)
		}
	}

	// 6. Next steps
	fmt.Printf("\n%s PicoClaw is ready!\n\n", logo)
	fmt.Println("Try it: picoclaw agent -m \"Hello!\"")
	fmt.Println()
	fmt.Println("If something isn't working:")
	fmt.Println("  picoclaw doctor          Diagnose problems")
	fmt.Println("  picoclaw doctor --fix    Auto-fix what it can")
}

func verifyOllama() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", "http://localhost:11434/api/tags", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("  [!] Ollama doesn't seem to be running at localhost:11434")
		fmt.Println("      Start it with: ollama serve")
		return
	}
	resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		fmt.Printf("  [\u2713] Ollama is running (http://localhost:11434)\n")
	} else {
		fmt.Printf("  [!] Ollama returned HTTP %d at localhost:11434\n", resp.StatusCode)
		fmt.Println("      Start it with: ollama serve")
	}
}

func verifyAPIKey(choice providerChoice, apiKey string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := choice.validateFunc(apiKey)
	req = req.WithContext(ctx)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("  [!] Could not reach %s (network error)\n", choice.name)
		fmt.Println("      Double-check your connection and try again later.")
		return
	}
	resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		fmt.Printf("  [\u2713] API key valid\n")
	} else if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		fmt.Printf("  [!] API key may be invalid (got HTTP %d)\n", resp.StatusCode)
		fmt.Println("      Double-check your key and try again later.")
	} else {
		// Some APIs return non-200 for list but key might still be valid
		fmt.Printf("  [!] %s returned HTTP %d (key may still be valid)\n", choice.name, resp.StatusCode)
	}
}

func copyEmbeddedToTarget(targetDir string) error {
	// Ensure target directory exists
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return fmt.Errorf("Failed to create target directory: %w", err)
	}

	// Walk through all files in embed.FS
	err := fs.WalkDir(embeddedFiles, "workspace", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Read embedded file
		data, err := embeddedFiles.ReadFile(path)
		if err != nil {
			return fmt.Errorf("Failed to read embedded file %s: %w", path, err)
		}

		new_path, err := filepath.Rel("workspace", path)
		if err != nil {
			return fmt.Errorf("Failed to get relative path for %s: %v\n", path, err)
		}

		// Build target file path
		targetPath := filepath.Join(targetDir, new_path)

		// Ensure target file's directory exists
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return fmt.Errorf("Failed to create directory %s: %w", filepath.Dir(targetPath), err)
		}

		// Write file
		if err := os.WriteFile(targetPath, data, 0o644); err != nil {
			return fmt.Errorf("Failed to write file %s: %w", targetPath, err)
		}

		return nil
	})

	return err
}

func createWorkspaceTemplates(workspace string) {
	err := copyEmbeddedToTarget(workspace)
	if err != nil {
		fmt.Printf("Error copying workspace templates: %v\n", err)
	}
}

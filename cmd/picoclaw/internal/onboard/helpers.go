package onboard

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sipeed/picoclaw/cmd/picoclaw/internal"
	"github.com/sipeed/picoclaw/pkg/config"
)

var workspaceTemplates = map[string]string{
	"AGENTS.md": `# Agent Instructions

You are a helpful AI assistant. Be concise, accurate, and friendly.
`,
	"IDENTITY.md": `# Identity

## Name
PicoClaw 🦞
`,
	"SOUL.md": `# Soul

I am picoclaw, a lightweight AI assistant powered by AI.
`,
	"USER.md": `# User

Information about user goes here.
`,
	"memory/MEMORY.md": `# Long-term Memory

This file stores important information that should persist across sessions.
`,
}

func onboard() {
	configPath := internal.GetConfigPath()

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

	cfg := config.DefaultConfig()
	if err := config.SaveConfig(configPath, cfg); err != nil {
		fmt.Printf("Error saving config: %v\n", err)
		os.Exit(1)
	}

	workspace := cfg.WorkspacePath()
	createWorkspaceTemplates(workspace)

	fmt.Printf("%s picoclaw is ready!\n", internal.Logo)
	fmt.Println("\nNext steps:")
	fmt.Println("  1. Add your API key to", configPath)
	fmt.Println("")
	fmt.Println("     Recommended:")
	fmt.Println("     - OpenRouter: https://openrouter.ai/keys (access 100+ models)")
	fmt.Println("     - Ollama:     https://ollama.com (local, free)")
	fmt.Println("")
	fmt.Println("     See README.md for 17+ supported providers.")
	fmt.Println("")
	fmt.Println("  2. Chat: picoclaw agent -m \"Hello!\"")
}

func createWorkspaceTemplates(workspace string) {
	err := copyEmbeddedToTarget(workspace)
	if err != nil {
		fmt.Printf("Error copying workspace templates: %v\n", err)
	}
}

func copyEmbeddedToTarget(targetDir string) error {
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	for relPath, content := range workspaceTemplates {
		targetPath := filepath.Join(targetDir, relPath)
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", filepath.Dir(targetPath), err)
		}
		if err := os.WriteFile(targetPath, []byte(content), 0o644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", targetPath, err)
		}
	}

	return nil
}

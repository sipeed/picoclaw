package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sipeed/picoclaw/pkg/agent"
	"github.com/sipeed/picoclaw/pkg/config"
)

func main() {
	cfg, err := config.LoadConfig(os.ExpandEnv("$HOME/.picoclaw/config.json"))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	registry := agent.NewAgentRegistry(cfg, nil)
	defaultAgent := registry.GetDefaultAgent()
	if defaultAgent == nil {
		fmt.Println("No default agent")
		return
	}

	cooldownPath := filepath.Join(filepath.Dir(filepath.Clean(defaultAgent.Workspace)), "cooldowns.json")
	fmt.Printf("Workspace: %s\n", defaultAgent.Workspace)
	fmt.Printf("Cooldown Path: %s\n", cooldownPath)
}

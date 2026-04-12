// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	tuicfg "github.com/sipeed/picoclaw/cmd/picoclaw-launcher-tui/config"
	"github.com/sipeed/picoclaw/cmd/picoclaw-launcher-tui/ui"
)

func main() {
	configPath := tuicfg.DefaultConfigPath()
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}

	configDir := filepath.Dir(configPath)
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		cmd := exec.Command("picoclaw", "onboard")
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		_ = cmd.Run()
	}

	cfg, err := tuicfg.Load(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "picoclaw-launcher-tui: %v\n", err)
		os.Exit(1)
	}

	app := ui.New(cfg, configPath)
	// Bind model selection hook to sync to main config
	app.OnModelSelected = func(scheme tuicfg.Scheme, user tuicfg.User, modelID string) {
		_ = tuicfg.SyncSelectedModelToMainConfig(scheme, user, modelID)
	}
	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "picoclaw-launcher-tui: %v\n", err)
		os.Exit(1)
	}
}

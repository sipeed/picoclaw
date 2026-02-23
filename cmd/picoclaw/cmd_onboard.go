// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT

package main

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/setup"
)

//go:generate cp -r ../../workspace .
//go:embed workspace
var embeddedFiles embed.FS

func onboard(args []string) {
	configPath := getConfigPath()

	interactive := false
	for _, arg := range args {
		if arg == "--interactive" || arg == "-i" {
			interactive = true
			break
		}
	}

	s, err := setup.NewSetup(configPath)
	if err != nil {
		fmt.Printf("Error initializing setup: %v\n", err)
		os.Exit(1)
	}

	if interactive {
		if err := s.Run(); err != nil {
			fmt.Printf("TUI error: %v\n", err)
		}

		if err := s.AskMissing(); err != nil {
			fmt.Printf("Error updating config: %v\n", err)
			os.Exit(1)
		}
	} else {
		if err := s.RunNonInteractive(); err != nil {
			fmt.Printf("Error running non-interactive setup: %v\n", err)
			os.Exit(1)
		}
	}

	cfg := s.Cfg
	if err := config.SaveConfig(configPath, cfg); err != nil {
		fmt.Printf("Error saving config: %v\n", err)
		os.Exit(1)
	}
	workspace := cfg.WorkspacePath()
	createWorkspaceTemplates(workspace)
}

func copyEmbeddedToTarget(targetDir string) error {
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	err := fs.WalkDir(embeddedFiles, "workspace", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		data, err := embeddedFiles.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read embedded file %s: %w", path, err)
		}

		newPath, err := filepath.Rel("workspace", path)
		if err != nil {
			return fmt.Errorf("failed to get relative path for %s: %v\n", path, err)
		}

		targetPath := filepath.Join(targetDir, newPath)

		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", filepath.Dir(targetPath), err)
		}

		if err := os.WriteFile(targetPath, data, 0o644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", targetPath, err)
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

// PicoClaw - Ultra-lightweight personal AI agent
// Inspired by and based on nanobot: https://github.com/HKUDS/nanobot
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/skills"
)

var (
	version   = "dev"
	gitCommit string
	buildTime string
	goVersion string
)

const logo = "🦞"

// formatVersion returns the version string with optional git commit
func formatVersion() string {
	v := version
	if gitCommit != "" {
		v += fmt.Sprintf(" (git: %s)", gitCommit)
	}
	return v
}

// formatBuildInfo returns build time and go version info
func formatBuildInfo() (build string, goVer string) {
	if buildTime != "" {
		build = buildTime
	}
	goVer = goVersion
	if goVer == "" {
		goVer = runtime.Version()
	}
	return
}

func printVersion() {
	fmt.Printf("%s picoclaw %s\n", logo, formatVersion())
	build, goVer := formatBuildInfo()
	if build != "" {
		fmt.Printf("  Build: %s\n", build)
	}
	if goVer != "" {
		fmt.Printf("  Go: %s\n", goVer)
	}
}

func copyDirectory(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		srcFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer srcFile.Close()

		dstFile, err := os.OpenFile(dstPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, info.Mode())
		if err != nil {
			return err
		}
		defer dstFile.Close()

		_, err = io.Copy(dstFile, srcFile)
		return err
	})
}

func main() {
	if len(os.Args) < 2 {
		printHelp()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "onboard":
		onboard()
	case "agent":
		agentCmd()
	case "gateway":
		gatewayCmd()
	case "status":
		statusCmd()
	case "migrate":
		migrateCmd()
	case "auth":
		authCmd()
	case "cron":
		cronCmd()
	case "skills":
		if len(os.Args) < 3 {
			skillsHelp()
			return
		}

		subcommand := os.Args[2]

		cfg, err := loadConfig()
		if err != nil {
			fmt.Printf("Error loading config: %v\n", err)
			os.Exit(1)
		}

		workspace := cfg.WorkspacePath()
		installer := skills.NewSkillInstaller(workspace)
		// 获取全局配置目录和内置 skills 目录
		globalDir := filepath.Dir(getConfigPath())
		globalSkillsDir := filepath.Join(globalDir, "skills")
		builtinSkillsDir := filepath.Join(globalDir, "picoclaw", "skills")
		skillsLoader := skills.NewSkillsLoader(workspace, globalSkillsDir, builtinSkillsDir)

		switch subcommand {
		case "list":
			skillsListCmd(skillsLoader)
		case "install":
			skillsInstallCmd(installer, cfg)
		case "remove", "uninstall":
			if len(os.Args) < 4 {
				fmt.Println("Usage: picoclaw skills remove <skill-name>")
				return
			}
			skillsRemoveCmd(installer, os.Args[3])
		case "install-builtin":
			skillsInstallBuiltinCmd(workspace)
		case "list-builtin":
			skillsListBuiltinCmd()
		case "search":
			skillsSearchCmd(installer)
		case "show":
			if len(os.Args) < 4 {
				fmt.Println("Usage: picoclaw skills show <skill-name>")
				return
			}
			skillsShowCmd(skillsLoader, os.Args[3])
		default:
			fmt.Printf("Unknown skills command: %s\n", subcommand)
			skillsHelp()
		}
	case "version", "--version", "-v":
		printVersion()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printHelp()
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Printf("%s picoclaw - Personal AI Assistant v%s\n\n", logo, version)
	fmt.Println("Usage: picoclaw <command>")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  onboard     Initialize picoclaw configuration and workspace")
	fmt.Println("  agent       Interact with the agent directly")
	fmt.Println("  auth        Manage authentication (login, logout, status)")
	fmt.Println("  gateway     Start picoclaw gateway")
	fmt.Println("  status      Show picoclaw status")
	fmt.Println("  cron        Manage scheduled tasks")
	fmt.Println("  migrate     Migrate from OpenClaw to PicoClaw")
	fmt.Println("  skills      Manage skills (install, list, remove)")
	fmt.Println("  version     Show version information")
}

func getConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".picoclaw", "config.json")
}

func loadConfig() (*config.Config, error) {
	return config.LoadConfig(getConfigPath())
}

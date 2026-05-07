package agent

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ergochat/readline"

	"github.com/sipeed/picoclaw/cmd/picoclaw/internal"
	"github.com/sipeed/picoclaw/pkg/agent"
	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/pathutil"
	"github.com/sipeed/picoclaw/pkg/providers"
)

func agentCmd(message, sessionKey, model string, debug bool,
	workspace, configDir, toolsFlag, skillsFlag string,
) error {
	if sessionKey == "" {
		sessionKey = "agent:main:cli:default"
	} else if !strings.HasPrefix(sessionKey, "agent:") {
		sessionKey = "agent:main:cli:" + sessionKey
	}

	cfg, err := internal.LoadConfig()
	if err != nil {
		return fmt.Errorf("error loading config: %w", err)
	}

	// Snapshot workspace_root from the base config before any overlay can touch it.
	workspaceRoot := cfg.Agents.Defaults.WorkspaceRoot

	// Validate and resolve --workspace and --config-dir against workspace_root.
	var resolveErr error
	workspace, configDir, resolveErr = validateWorkspacePaths(workspaceRoot, workspace, configDir)
	if resolveErr != nil {
		return resolveErr
	}

	// Apply workspace-local config overrides from config-dir.
	// mergeAgentDefaults will re-validate any workspace field from the overlay
	// against the snapshotted workspace_root.
	if configDir != "" {
		wc, wcErr := config.LoadWorkspaceConfig(configDir)
		if wcErr != nil {
			return fmt.Errorf("error loading workspace config from %s: %w", configDir, wcErr)
		}
		if mergeErr := cfg.MergeWorkspaceConfig(wc); mergeErr != nil {
			return fmt.Errorf("error merging workspace config from %s: %w", configDir, mergeErr)
		}
	}

	logger.ConfigureFromEnv()

	if debug {
		logger.SetLevel(logger.DEBUG)
		fmt.Println("🔍 Debug mode enabled")
	}

	// CLI flags win over workspace config
	if model != "" {
		cfg.Agents.Defaults.ModelName = model
	}

	// Workspace override (already validated above)
	if workspace != "" {
		cfg.Agents.Defaults.Workspace = workspace
		os.MkdirAll(workspace, 0o755)
	}

	// Tool allowlist: disable all tools, then enable only the listed ones
	if toolsFlag != "" {
		toolList := strings.Split(toolsFlag, ",")
		for i := range toolList {
			toolList[i] = strings.TrimSpace(toolList[i])
		}
		applyToolAllowlist(cfg, toolList)
	}

	// Skills filter: inject into agent config so NewAgentInstance picks it up
	if skillsFlag != "" {
		skillList := strings.Split(skillsFlag, ",")
		for i := range skillList {
			skillList[i] = strings.TrimSpace(skillList[i])
		}
		applySkillsFilter(cfg, skillList)
	}

	provider, modelID, err := providers.CreateProvider(cfg)
	if err != nil {
		return fmt.Errorf("error creating provider: %w", err)
	}

	// Use the resolved model ID from provider creation
	if modelID != "" {
		cfg.Agents.Defaults.ModelName = modelID
	}

	msgBus := bus.NewMessageBus()
	defer msgBus.Close()
	agentLoop := agent.NewAgentLoop(cfg, msgBus, provider)
	defer agentLoop.Close()

	// Copy bootstrap files from config-dir to workspace
	if configDir != "" {
		copyBootstrapFiles(configDir, cfg.Agents.Defaults.Workspace)
	}

	// Print agent startup info (only for interactive mode)
	startupInfo := agentLoop.GetStartupInfo()
	logger.InfoCF("agent", "Agent initialized",
		map[string]any{
			"tools_count":      startupInfo["tools"].(map[string]any)["count"],
			"skills_total":     startupInfo["skills"].(map[string]any)["total"],
			"skills_available": startupInfo["skills"].(map[string]any)["available"],
		})

	if message != "" {
		ctx := context.Background()
		response, err := agentLoop.ProcessDirect(ctx, message, sessionKey)
		if err != nil {
			return fmt.Errorf("error processing message: %w", err)
		}
		fmt.Printf("\n%s %s\n", internal.Logo, response)
		return nil
	}

	fmt.Printf("%s Interactive mode (Ctrl+C to exit)\n\n", internal.Logo)
	interactiveMode(agentLoop, sessionKey)

	return nil
}

// applyToolAllowlist disables all tools, then enables only the listed ones.
func applyToolAllowlist(cfg *config.Config, allowed []string) {
	allowSet := make(map[string]bool, len(allowed))
	for _, t := range allowed {
		allowSet[t] = true
	}

	cfg.Tools.ReadFile.Enabled = allowSet["read_file"]
	cfg.Tools.WriteFile.Enabled = allowSet["write_file"]
	cfg.Tools.EditFile.Enabled = allowSet["edit_file"]
	cfg.Tools.AppendFile.Enabled = allowSet["append_file"]
	cfg.Tools.ListDir.Enabled = allowSet["list_dir"]
	cfg.Tools.Exec.Enabled = allowSet["exec"]
	cfg.Tools.Spawn.Enabled = allowSet["spawn"]
	cfg.Tools.Cron.Enabled = allowSet["cron"]
	cfg.Tools.Web.Enabled = allowSet["web"] || allowSet["web_search"]
	cfg.Tools.WebFetch.Enabled = allowSet["web_fetch"]
	cfg.Tools.Skills.Enabled = allowSet["skills"]
	cfg.Tools.FindSkills.Enabled = allowSet["find_skills"]
	cfg.Tools.InstallSkill.Enabled = allowSet["install_skill"]
	cfg.Tools.Subagent.Enabled = allowSet["subagent"]
	cfg.Tools.Message.Enabled = allowSet["message"]
	cfg.Tools.MCP.Enabled = allowSet["mcp"]
	cfg.Tools.I2C.Enabled = allowSet["i2c"]
	cfg.Tools.SPI.Enabled = allowSet["spi"]
}

// applySkillsFilter injects a skills filter into the agent config.
func applySkillsFilter(cfg *config.Config, skills []string) {
	if len(cfg.Agents.List) == 0 {
		// Create an implicit main agent with skills filter
		cfg.Agents.List = []config.AgentConfig{
			{ID: "main", Default: true, Skills: skills},
		}
	} else {
		// Apply to all agents
		for i := range cfg.Agents.List {
			cfg.Agents.List[i].Skills = skills
		}
	}
}

// validateWorkspacePaths resolves --workspace and --config-dir against workspace_root.
// Both must be relative subdirectories of workspace_root. Returns the resolved
// absolute paths or an error if validation fails.
func validateWorkspacePaths(workspaceRoot, workspace, configDir string) (string, string, error) {
	if workspace != "" {
		resolved, err := pathutil.ResolveWorkspacePath(workspaceRoot, workspace)
		if err != nil {
			return "", "", fmt.Errorf("invalid --workspace: %w", err)
		}
		workspace = resolved
	}
	if configDir != "" {
		resolved, err := pathutil.ResolveWorkspacePath(workspaceRoot, configDir)
		if err != nil {
			return "", "", fmt.Errorf("invalid --config-dir: %w", err)
		}
		configDir = resolved
	}
	return workspace, configDir, nil
}

// copyBootstrapFiles copies recognized bootstrap files (AGENTS.md, IDENTITY.md,
// SOUL.md, USER.md) from srcDir into the workspace directory.
func copyBootstrapFiles(srcDir, workspace string) {
	bootstrapFiles := []string{"AGENTS.md", "IDENTITY.md", "SOUL.md", "USER.md"}
	for _, filename := range bootstrapFiles {
		srcPath := filepath.Join(srcDir, filename)
		data, err := os.ReadFile(srcPath)
		if err != nil {
			continue // file not present in config-dir, skip
		}
		dstPath := filepath.Join(workspace, filename)
		if err := os.WriteFile(dstPath, data, 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to write %s: %v\n", dstPath, err)
		}
	}
}

func interactiveMode(agentLoop *agent.AgentLoop, sessionKey string) {
	prompt := fmt.Sprintf("%s You: ", internal.Logo)

	rl, err := readline.NewEx(&readline.Config{
		Prompt:          prompt,
		HistoryFile:     filepath.Join(os.TempDir(), ".picoclaw_history"),
		HistoryLimit:    100,
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		fmt.Printf("Error initializing readline: %v\n", err)
		fmt.Println("Falling back to simple input mode...")
		simpleInteractiveMode(agentLoop, sessionKey)
		return
	}
	defer rl.Close()

	for {
		line, err := rl.Readline()
		if err != nil {
			if err == readline.ErrInterrupt || err == io.EOF {
				fmt.Println("\nGoodbye!")
				return
			}
			fmt.Printf("Error reading input: %v\n", err)
			continue
		}

		input := strings.TrimSpace(line)
		if input == "" {
			continue
		}

		if input == "exit" || input == "quit" {
			fmt.Println("Goodbye!")
			return
		}

		ctx := context.Background()
		response, err := agentLoop.ProcessDirect(ctx, input, sessionKey)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		fmt.Printf("\n%s %s\n\n", internal.Logo, response)
	}
}

func simpleInteractiveMode(agentLoop *agent.AgentLoop, sessionKey string) {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print(fmt.Sprintf("%s You: ", internal.Logo))
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				fmt.Println("\nGoodbye!")
				return
			}
			fmt.Printf("Error reading input: %v\n", err)
			continue
		}

		input := strings.TrimSpace(line)
		if input == "" {
			continue
		}

		if input == "exit" || input == "quit" {
			fmt.Println("Goodbye!")
			return
		}

		ctx := context.Background()
		response, err := agentLoop.ProcessDirect(ctx, input, sessionKey)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		fmt.Printf("\n%s %s\n\n", internal.Logo, response)
	}
}

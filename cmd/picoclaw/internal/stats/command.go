package stats

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/sipeed/picoclaw/cmd/picoclaw/internal"
	"github.com/sipeed/picoclaw/pkg/llmlog"
)

func NewStatsCommand() *cobra.Command {
	var days int
	var model string
	var logDir string

	cmd := &cobra.Command{
		Use:     "stats",
		Aliases: []string{"stat"},
		Short:   "Show token usage statistics",
		Long:    "Display token usage statistics from LLM call logs",
		Example: `  picoclaw stats                  # Show all time statistics
  picoclaw stats --days 7         # Show last 7 days statistics
  picoclaw stats --days 30        # Show last 30 days statistics
  picoclaw stats --model gpt-4o   # Filter by model
  picoclaw stats --log-dir /path/to/logs  # Specify log directory`,
		Run: func(cmd *cobra.Command, args []string) {
			// Find log directory
			resolvedLogDir := findLogDir(logDir)
			if resolvedLogDir == "" {
				fmt.Println("❌ 无法找到日志目录")
				fmt.Println("提示: 使用 --log-dir 指定日志目录，或在配置文件中设置 tools.llm_call_log.log_dir")
				os.Exit(1)
			}

			// Calculate statistics
			result, err := llmlog.CalculateStats(resolvedLogDir, days, model)
			if err != nil {
				fmt.Printf("❌ Stats calculation failed: %v\n", err)
				os.Exit(1)
			}

			// Print result
			fmt.Println(llmlog.FormatStatsTable(result))
		},
	}

	cmd.Flags().IntVarP(&days, "days", "d", 0, "Number of days to include (0 means all time)")
	cmd.Flags().StringVarP(&model, "model", "m", "", "Filter by specific model")
	cmd.Flags().StringVarP(&logDir, "log-dir", "l", "", "Log directory path (overrides config)")

	return cmd
}

// findLogDir finds the LLM log directory with the following priority:
// 1. Explicitly provided logDir parameter
// 2. Config file: tools.llm_call_log.log_dir
// 3. Environment variable: PICOCLAW_TOOLS_LLM_CALL_LOG_DIR
// 4. Default locations: ~/.picoclaw/logs/llmcall, ~/picoclaw/workspace/logs/llmcall
func findLogDir(logDir string) string {
	// 1. Check explicit parameter
	if logDir != "" {
		if _, err := os.Stat(logDir); err == nil {
			return logDir
		}
		// If explicitly provided but doesn't exist, still return it
		// (error will be reported later)
		return logDir
	}

	// 2. Check config file and environment variable
	cfg, err := internal.LoadConfig()
	if err == nil && cfg.Tools.LLMCallLog.LogDir != "" {
		if _, err := os.Stat(cfg.Tools.LLMCallLog.LogDir); err == nil {
			return cfg.Tools.LLMCallLog.LogDir
		}
	}

	// 3. Check environment variable directly (in case config loading failed)
	if envDir := os.Getenv("PICOCLAW_TOOLS_LLM_CALL_LOG_DIR"); envDir != "" {
		if _, err := os.Stat(envDir); err == nil {
			return envDir
		}
	}

	// 4. Check default locations
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	// Default: ~/.picoclaw/logs/llmcall
	defaultDir := filepath.Join(homeDir, ".picoclaw", "logs", "llmcall")
	if _, err := os.Stat(defaultDir); err == nil {
		return defaultDir
	}

	// Alternative: workspace logs
	workspaceLogDir := filepath.Join(homeDir, "picoclaw", "workspace", "logs", "llmcall")
	if _, err := os.Stat(workspaceLogDir); err == nil {
		return workspaceLogDir
	}

	// Legacy: check old directory names for backward compatibility
	legacyDir := filepath.Join(homeDir, ".picoclaw", "logs", "llm")
	if _, err := os.Stat(legacyDir); err == nil {
		return legacyDir
	}

	legacyWorkspaceDir := filepath.Join(homeDir, "picoclaw", "workspace", "logs", "llm")
	if _, err := os.Stat(legacyWorkspaceDir); err == nil {
		return legacyWorkspaceDir
	}

	return ""
}
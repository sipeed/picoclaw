package web

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/sipeed/picoclaw/cmd/picoclaw/internal"
	"github.com/sipeed/picoclaw/pkg/agent"
	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/providers"
	pkgweb "github.com/sipeed/picoclaw/pkg/web"
)

// NewWebCommand creates the cobra command for the web management UI.
func NewWebCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "web",
		Short: "启动 Web 管理界面",
		Long: `启动 PicoClaw 的 Web 配置管理界面。

通过浏览器访问管理界面来配置 AI 模型、消息通道和工具选项。
配置保存到 config.json 后，需要重启相关服务才能生效。

首次使用前，请在 config.json 中设置 web.username 和 web.password：

  {
    "web": {
      "host": "0.0.0.0",
      "port": 18799,
      "username": "admin",
      "password": "your-secure-password"
    }
  }`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWeb()
		},
	}
	return cmd
}

func runWeb() error {
	cfg, err := internal.LoadConfig()
	if err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}

	configPath := internal.GetConfigPath()

	// Warn if password not set
	if cfg.Web.Password == "" {
		return fmt.Errorf(
			"Web 管理界面密码未配置。\n\n"+
				"请在 %s 中设置：\n\n"+
				"  \"web\": {\n"+
				"    \"host\": \"0.0.0.0\",\n"+
				"    \"port\": 18799,\n"+
				"    \"username\": \"admin\",\n"+
				"    \"password\": \"your-secure-password\"\n"+
				"  }\n",
			configPath,
		)
	}

	// Use default username if empty
	if cfg.Web.Username == "" {
		cfg.Web.Username = "admin"
	}

	// Create provider for agent loop
	provider, modelID, err := providers.CreateProvider(cfg)
	if err != nil {
		return fmt.Errorf("error creating provider: %w", err)
	}

	// Use the resolved model ID from provider creation
	if modelID != "" {
		cfg.Agents.Defaults.ModelName = modelID
	}

	// Create message bus and agent loop
	msgBus := bus.NewMessageBus()
	agentLoop := agent.NewAgentLoop(cfg, msgBus, provider)

	// Print agent startup info
	fmt.Println("\n📦 Agent Status:")
	startupInfo := agentLoop.GetStartupInfo()
	toolsInfo := startupInfo["tools"].(map[string]any)
	skillsInfo := startupInfo["skills"].(map[string]any)
	fmt.Printf("  • Tools: %d loaded\n", toolsInfo["count"])
	fmt.Printf("  • Skills: %d/%d available\n",
		skillsInfo["available"],
		skillsInfo["total"])

	logger.InfoCF("agent", "Agent initialized",
		map[string]any{
			"tools_count":      toolsInfo["count"],
			"skills_total":     skillsInfo["total"],
			"skills_available": skillsInfo["available"],
		})

	// Start agent loop in background
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		if err := agentLoop.Run(ctx); err != nil {
			logger.ErrorCF("web", "Agent loop error", map[string]any{
				"error": err.Error(),
			})
		}
	}()
	defer cancel()

	// Create web server
	server := pkgweb.NewServer(cfg, configPath)

	// Inject agent loop and message bus into web server
	server.SetAgentLoop(agentLoop, msgBus)

	return server.Start()
}

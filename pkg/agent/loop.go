// PicoClaw - 超轻量级个人 AI 智能体
// 籲发并基于 nanobot: https://github.com/HKUDS/nanobot
// 许可证: MIT
//
// Copyright (c) 2026 PicoClaw contributors

// ── agent 包 ──
//
// 本包实现了 PicoClaw 的核心 Agent Loop（智能体主循环)。
//
// ## 架构概览
//
// AgentLoop 是整个系统的大脑，负责：
//   1. 从消息总线(bus)接收用户消息
//   2. 通过路由系统(routing)将消息分发到对应的 Agent
//   3. 管理 LLM 对话上下文(会话历史、系统提示)
//   4. 调用 LLM 揸商商商模型)获取响应
//   5. 执行工具调用(文件读写、网页搜索、消息发送等)
//   6. 通过消息总线(bus)将响应发送回用户
//
// ## 栄心数据流
//
//   用户消息 → InboundChan → processMessage → runAgentLoop → runTurn
//     → BuildMessages(构建上下文) → LLM Chat → 解析响应
//     → 如果有工具调用 → ExecuteWithContext → 绔回结果给 LLM
//     → 循环直到 LLM 返回纯文本 → PublishOutbound → 用户
//
// ## 关键子系统
//
//   - 会话管理(Sessions): 存储对话历史,支持摘要
//   - 路由(Routing): 将消息分发到不同的 Agent
//   - 钩子(Hooks): 在 LLM 蛽用、工具执行前后插入自定义逻辑
//   - 转向(Steering): 在 LLM 変换中途中注入新的用户消息
//   - 压缩(Compression): 当上下文过长时自动压缩历史
//   - 摘要(Summarization): 定期将历史总结为摘要以减少 token 消耗
//
package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/commands"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/constants"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/media"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/routing"
	"github.com/sipeed/picoclaw/pkg/skills"
	"github.com/sipeed/picoclaw/pkg/state"
	"github.com/sipeed/picoclaw/pkg/tools"
	"github.com/sipeed/picoclaw/pkg/utils"
	"github.com/sipeed/picoclaw/pkg/voice"
)

// AgentLoop 是整个 AI Agent 的核心运行循环
//
// 它 它// 是系统中所有消息处理的入口点。一个 AgentLoop 宂例管理消息路由、 LLM 交互、会话历史、工具调用和事件发射等//
//
// 生命周期:
//   1. 创建时通过 NewAgentLoop 枝始化
//   2. 通过 Run() 同动主消息处理循环
//   3. 通过 Stop() 优雅关闭
//   4. 通过 Close() 释放资源
//
// 栯心设计原则:
//   - 单会处理并发： 通过 sessionKey 鯧分不同用户会话,并处理)
//   - 流式响应: 支持通过 MessageBus 异步发送 LLM 响应
//   - 可中断支持: steering 机制和硬中断
//   - 自动上下文管理: 压缩和摘要机制防止 token 溢出
//
// AgentLoop 结构体
//
// 注意: 这个结构体本身不是线程安全的,所有字段都通过各自机制保护,
// 例如: running 使用 atomic.Bool, channels 使用 sync.Map,
type AgentLoop struct {
	// === 核心依赖 ===
	bus      *bus.MessageBus      // 消息总线,所有消息通过这里传递
	cfg      *config.Config       // 全局配置
	registry *AgentRegistry       // Agent 注册表,管理多个 Agent 实例及其路由
	state    *state.Manager       // 持久化状态管理器,保存最近活跃通道等	// === 事件系统 ===
	eventBus *EventBus    // 事件总线,用于发布 Agent 事件(如 turn 开始/结束)
	hooks    *HookManager // 钩子管理器,在 LLM/工具调用前后插入自定义逻辑	// === 运行时状态 ===
	running        atomic.Bool             // 是否正在运行(原子操作,线程安全)
	summarizing    sync.Map                // 正在进行的摘要任务(sessionKey -> bool),防止并发摘要
	fallback       *providers.FallbackChain // LLM 降级链,支持多 Provider 自动切换
	channelManager *channels.Manager       // 渠道管理器,管理所有消息通道(Telegram/Discord 等)
	mediaStore     media.MediaStore        // 媒体存储(图片/音频/视频文件的临时存储)
	transcriber    voice.Transcriber       // 语音转录器(将音频转为文字)
	cmdRegistry    *commands.Registry      // 命令注册表(如 /help, /clear 等斜杠命令)
	mcp            mcpRuntime              // MCP 运行时(模型上下文协议,管理外部工具)
	hookRuntime    hookRuntime             // 钩子运行时(管理动态加载的钩子)
	steering       *steeringQueue          // 转向队列(在 LLM 对话中途中注入新的用户消息)
	pendingSkills  sync.Map                // 待应用技能(sessionKey -> []string)
	mu             sync.RWMutex            // 读写锁,保护配置热重载时 registry 原子切换

	// === 并发 Turn 管理 ===
	// 支持多个会话同时进行 LLM 对话(每个 sessionKey 对应一个独立的对话状态)
	activeTurnStates sync.Map     // 会话状态映射: sessionKey → *turnState
	subTurnCounter   atomic.Int64 // 子 Turn ID 生成器(用于 SubAgent 递归调用)

	// === Turn 追踪 ===
	turnSeq        atomic.Uint64   // Turn 序列号生成器(单调递增,用于生成 TurnID)
	activeRequests sync.WaitGroup  // 活跃请求计数器(等待所有 LLM 请求完成后再关闭)

	reloadFunc func() error
}

// processOptions 配置消息处理选项
//
// 控制单条消息如何被处理,包括:
//   - 路由到哪个 Agent/会话
//   - 使用哪个通道发送响应
//   - 是否加载历史记录
//   - 是否触发摘要
//   - 是否在对话中途注入新消息(steering)
type processOptions struct {
	SessionKey              string              // 会话标识符，用于定位历史记录和上下文
	Channel                 string              // 目标通道名称(如 "telegram", "discord")
	ChatID                  string              // 目标聊天 ID(如群组 ID、频道 ID)
	SenderID                string              // 当前发送者 ID，用于动态上下文
	SenderDisplayName       string              // 当前发送者显示名称(用于动态上下文)
	UserMessage             string              // 用户消息内容(可能包含命令前缀)
	ForcedSkills            []string            // 此消息显式请求使用的技能列表
	SystemPromptOverride    string              // 覆盖默认系统提示(用于子 Turn/SubAgent)
	Media                   []string            // media:// 引用列表(来自入站消息的附件媒体)
	InitialSteeringMessages []providers.Message // 初始注入的转向消息(来自重构/Agent 切换轮)
	DefaultResponse         string              // 当 LLM 返回空响应时的默认回复
	EnableSummary           bool                // 是否在 turn 结束后触发会话摘要
	SendResponse            bool                // 是否通过消息总线发送响应
	SuppressToolFeedback    bool                // 是否抑制工具执行的反馈消息(如 "正在执行 xxx...")
	NoHistory               bool                // 如果为 true,不加载会话历史(用于心跳等避免上下文膨胀)
	SkipInitialSteeringPoll bool                // 如果为 true,跳过循环开始时的转向消息轮询(用于 Continue)
}

// continuationTarget 表示转向续轮的目标
// 当一个 Turn 宎束后还有排队中的用户消息需要继续处理时,
// 使用此结构记录续轮的目标位置(通道+聊天ID+会话)
type continuationTarget struct {
	SessionKey string // 会话标识符
	Channel    string // 目标通道
	ChatID     string // 目标聊天 ID
}

// === 核心常量 ===
const (
	defaultResponse            = "The model returned an empty response. This may indicate a provider error or token limit." // LLM 返回空内容时的默认响应
	toolLimitResponse          = "I've reached `max_tool_iterations` without a final response. Increase `max_tool_iterations` in config.json if this task needs more tool steps." // 达到最大工具迭代次数时的提示
	handledToolResponseSummary = "Requested output delivered via tool attachment." // 工具已处理输出时的摘要
	sessionKeyAgentPrefix      = "agent:"                       // Agent 作用域会会话前缀
	metadataKeyAccountID       = "account_id"                   // 元数据键: 軬户 ID
	metadataKeyGuildID         = "guild_id"                     // 元数据键: 服务器 Guild ID(如 Discord)
	metadataKeyTeamID          = "team_id"                      // 元数据键: 团队 ID(如 Slack)
	metadataKeyParentPeerKind  = "parent_peer_kind"                // 元数据键: 父级对等方类型(用于回复场景)
	metadataKeyParentPeerID    = "parent_peer_id"                // 元数据键: 父级对等方 ID(用于回复场景)
)

// NewAgentLoop 创建并初始化 AgentLoop 宙例
//
// 初始化流程:
//   1. 创建 Agent 注册表(Registry),包含默认 Agent 和自定义 Agent
//   2. 创建降级链(FallbackChain),支持多 Provider 自动切换
//   3. 创建状态管理器(StateManager),用于持久化最近活跃通道
//   4. 创建事件总线(EventBus),用于发布 Agent 事件
//   5. 创建命令注册表(CommandRegistry),注册内置斜杠命令
//   6. 创建转向队列(SteeringQueue),管理对话途中的用户消息注入
//   7. 创建钩子管理器(HookManager),注册配置中定义的钩子
//   8. 注册共享工具(web_search, web_fetch, message, send_file 等)
//
// 注意: provider 参数在此处仅用于创建 Registry,
// 宾际的 LLM 豸用通过 AgentInstance.Provider 字段访问
func NewAgentLoop(
	cfg *config.Config,
	msgBus *bus.MessageBus,
	provider providers.LLMProvider,
) *AgentLoop {
	registry := NewAgentRegistry(cfg, provider)

	// Set up shared fallback chain
	cooldown := providers.NewCooldownTracker()
	fallbackChain := providers.NewFallbackChain(cooldown)

	// Create state manager using default agent's workspace for channel recording
	defaultAgent := registry.GetDefaultAgent()
	var stateManager *state.Manager
	if defaultAgent != nil {
		stateManager = state.NewManager(defaultAgent.Workspace)
	}

	eventBus := NewEventBus()
	al := &AgentLoop{
		bus:         msgBus,
		cfg:         cfg,
		registry:    registry,
		state:       stateManager,
		eventBus:    eventBus,
		summarizing: sync.Map{},
		fallback:    fallbackChain,
		cmdRegistry: commands.NewRegistry(commands.BuiltinDefinitions()),
		steering:    newSteeringQueue(parseSteeringMode(cfg.Agents.Defaults.SteeringMode)),
	}
	al.hooks = NewHookManager(eventBus)
	configureHookManagerFromConfig(al.hooks, cfg)

	// Register shared tools to all agents (now that al is created)
	registerSharedTools(al, cfg, msgBus, registry, provider)

	return al
}

// registerSharedTools 向所有 Agent 注册共享工具
//
// 共享工具是所有 Agent 都可以使用的工具,包括:
//   - web: 网页搜索(Brave/Tavily/DuckDuckGo/Perplexity/SearXNG/GLMSearch/BaiduSearch)
//   - web_fetch: 网页内容抓取
//   - i2c/spi: 硬件 I2C/SPI 接口(仅 Linux)
//   - message: 跨通道消息发送
//   - send_file: 文件发送(通过 MediaStore)
//   - skills/find_skills/install_skill: 技能发现与安装
//   - spawn/spawn_status/subagent: 子 Agent 生成与管理
//
// 工具注册遵循以下原则:
//   1. 每个工具通过 config.Tools.IsToolEnabled() 检查是否启用
//   2. 工具实例按 Agent 独立创建(避免共享状态)
//   3. 子 Agent 工具(spawn) 使用克隆的工具集(防止递归生成)
func registerSharedTools(
	al *AgentLoop,
	cfg *config.Config,
	msgBus *bus.MessageBus,
	registry *AgentRegistry,
	provider providers.LLMProvider,
) {
	allowReadPaths := buildAllowReadPatterns(cfg)

	for _, agentID := range registry.ListAgentIDs() {
		agent, ok := registry.GetAgent(agentID)
		if !ok {
			continue
		}

		if cfg.Tools.IsToolEnabled("web") {
			searchTool, err := tools.NewWebSearchTool(tools.WebSearchToolOptions{
				BraveAPIKeys:          cfg.Tools.Web.Brave.APIKeys.Values(),
				BraveMaxResults:       cfg.Tools.Web.Brave.MaxResults,
				BraveEnabled:          cfg.Tools.Web.Brave.Enabled,
				TavilyAPIKeys:         cfg.Tools.Web.Tavily.APIKeys.Values(),
				TavilyBaseURL:         cfg.Tools.Web.Tavily.BaseURL,
				TavilyMaxResults:      cfg.Tools.Web.Tavily.MaxResults,
				TavilyEnabled:         cfg.Tools.Web.Tavily.Enabled,
				DuckDuckGoMaxResults:  cfg.Tools.Web.DuckDuckGo.MaxResults,
				DuckDuckGoEnabled:     cfg.Tools.Web.DuckDuckGo.Enabled,
				PerplexityAPIKeys:     cfg.Tools.Web.Perplexity.APIKeys.Values(),
				PerplexityMaxResults:  cfg.Tools.Web.Perplexity.MaxResults,
				PerplexityEnabled:     cfg.Tools.Web.Perplexity.Enabled,
				SearXNGBaseURL:        cfg.Tools.Web.SearXNG.BaseURL,
				SearXNGMaxResults:     cfg.Tools.Web.SearXNG.MaxResults,
				SearXNGEnabled:        cfg.Tools.Web.SearXNG.Enabled,
				GLMSearchAPIKey:       cfg.Tools.Web.GLMSearch.APIKey.String(),
				GLMSearchBaseURL:      cfg.Tools.Web.GLMSearch.BaseURL,
				GLMSearchEngine:       cfg.Tools.Web.GLMSearch.SearchEngine,
				GLMSearchMaxResults:   cfg.Tools.Web.GLMSearch.MaxResults,
				GLMSearchEnabled:      cfg.Tools.Web.GLMSearch.Enabled,
				BaiduSearchAPIKey:     cfg.Tools.Web.BaiduSearch.APIKey.String(),
				BaiduSearchBaseURL:    cfg.Tools.Web.BaiduSearch.BaseURL,
				BaiduSearchMaxResults: cfg.Tools.Web.BaiduSearch.MaxResults,
				BaiduSearchEnabled:    cfg.Tools.Web.BaiduSearch.Enabled,
				Proxy:                 cfg.Tools.Web.Proxy,
			})
			if err != nil {
				logger.ErrorCF("agent", "Failed to create web search tool", map[string]any{"error": err.Error()})
			} else if searchTool != nil {
				agent.Tools.Register(searchTool)
			}
		}
		if cfg.Tools.IsToolEnabled("web_fetch") {
			fetchTool, err := tools.NewWebFetchToolWithProxy(
				50000,
				cfg.Tools.Web.Proxy,
				cfg.Tools.Web.Format,
				cfg.Tools.Web.FetchLimitBytes,
				cfg.Tools.Web.PrivateHostWhitelist)
			if err != nil {
				logger.ErrorCF("agent", "Failed to create web fetch tool", map[string]any{"error": err.Error()})
			} else {
				agent.Tools.Register(fetchTool)
			}
		}

		// Hardware tools (I2C, SPI) - Linux only, returns error on other platforms
		if cfg.Tools.IsToolEnabled("i2c") {
			agent.Tools.Register(tools.NewI2CTool())
		}
		if cfg.Tools.IsToolEnabled("spi") {
			agent.Tools.Register(tools.NewSPITool())
		}

		// Message tool
		if cfg.Tools.IsToolEnabled("message") {
			messageTool := tools.NewMessageTool()
			messageTool.SetSendCallback(func(channel, chatID, content string) error {
				pubCtx, pubCancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer pubCancel()
				return msgBus.PublishOutbound(pubCtx, bus.OutboundMessage{
					Channel: channel,
					ChatID:  chatID,
					Content: content,
				})
			})
			agent.Tools.Register(messageTool)
		}

		// Send file tool (outbound media via MediaStore — store injected later by SetMediaStore)
		if cfg.Tools.IsToolEnabled("send_file") {
			sendFileTool := tools.NewSendFileTool(
				agent.Workspace,
				cfg.Agents.Defaults.RestrictToWorkspace,
				cfg.Agents.Defaults.GetMaxMediaSize(),
				nil,
				allowReadPaths,
			)
			agent.Tools.Register(sendFileTool)
		}

		// Skill discovery and installation tools
		skills_enabled := cfg.Tools.IsToolEnabled("skills")
		find_skills_enable := cfg.Tools.IsToolEnabled("find_skills")
		install_skills_enable := cfg.Tools.IsToolEnabled("install_skill")
		if skills_enabled && (find_skills_enable || install_skills_enable) {
			clawHubConfig := cfg.Tools.Skills.Registries.ClawHub
			registryMgr := skills.NewRegistryManagerFromConfig(skills.RegistryConfig{
				MaxConcurrentSearches: cfg.Tools.Skills.MaxConcurrentSearches,
				ClawHub: skills.ClawHubConfig{
					Enabled:         clawHubConfig.Enabled,
					BaseURL:         clawHubConfig.BaseURL,
					AuthToken:       clawHubConfig.AuthToken.String(),
					SearchPath:      clawHubConfig.SearchPath,
					SkillsPath:      clawHubConfig.SkillsPath,
					DownloadPath:    clawHubConfig.DownloadPath,
					Timeout:         clawHubConfig.Timeout,
					MaxZipSize:      clawHubConfig.MaxZipSize,
					MaxResponseSize: clawHubConfig.MaxResponseSize,
				},
			})

			if find_skills_enable {
				searchCache := skills.NewSearchCache(
					cfg.Tools.Skills.SearchCache.MaxSize,
					time.Duration(cfg.Tools.Skills.SearchCache.TTLSeconds)*time.Second,
				)
				agent.Tools.Register(tools.NewFindSkillsTool(registryMgr, searchCache))
			}

			if install_skills_enable {
				agent.Tools.Register(tools.NewInstallSkillTool(registryMgr, agent.Workspace))
			}
		}

		// Spawn and spawn_status tools share a SubagentManager.
		// Construct it when either tool is enabled (both require subagent).
		spawnEnabled := cfg.Tools.IsToolEnabled("spawn")
		spawnStatusEnabled := cfg.Tools.IsToolEnabled("spawn_status")
		if (spawnEnabled || spawnStatusEnabled) && cfg.Tools.IsToolEnabled("subagent") {
			subagentManager := tools.NewSubagentManager(provider, agent.Model, agent.Workspace)
			subagentManager.SetLLMOptions(agent.MaxTokens, agent.Temperature)

			// Set the spawner that links into AgentLoop's turnState
			subagentManager.SetSpawner(func(
				ctx context.Context,
				task, label, targetAgentID string,
				tls *tools.ToolRegistry,
				maxTokens int,
				temperature float64,
				hasMaxTokens, hasTemperature bool,
			) (*tools.ToolResult, error) {
				// 1. Recover parent Turn State from Context
				parentTS := turnStateFromContext(ctx)
				if parentTS == nil {
					// Fallback: If no turnState exists in context, create an isolated ad-hoc root turn state
					// so that the tool can still function outside of an agent loop (e.g. tests, raw invocations).
					parentTS = &turnState{
						ctx:            ctx,
						turnID:         "adhoc-root",
						depth:          0,
						session:        nil, // Ephemeral session not needed for adhoc spawn
						pendingResults: make(chan *tools.ToolResult, 16),
						concurrencySem: make(chan struct{}, 5),
					}
				}

				// 2. Build Tools slice from registry
				var tlSlice []tools.Tool
				for _, name := range tls.List() {
					if t, ok := tls.Get(name); ok {
						tlSlice = append(tlSlice, t)
					}
				}

				// 3. System Prompt
				systemPrompt := "You are a subagent. Complete the given task independently and report the result.\n" +
					"You have access to tools - use them as needed to complete your task.\n" +
					"After completing the task, provide a clear summary of what was done.\n\n" +
					"Task: " + task

				// 4. Resolve Model
				modelToUse := agent.Model
				if targetAgentID != "" {
					if targetAgent, ok := al.GetRegistry().GetAgent(targetAgentID); ok {
						modelToUse = targetAgent.Model
					}
				}

				// 5. Build SubTurnConfig
				cfg := SubTurnConfig{
					Model:        modelToUse,
					Tools:        tlSlice,
					SystemPrompt: systemPrompt,
				}
				if hasMaxTokens {
					cfg.MaxTokens = maxTokens
				}

				// 6. Spawn SubTurn
				return spawnSubTurn(ctx, al, parentTS, cfg)
			})

			// Clone the parent's tool registry so subagents can use all
			// tools registered so far (file, web, etc.) but NOT spawn/
			// spawn_status which are added below — preventing recursive
			// subagent spawning.
			subagentManager.SetTools(agent.Tools.Clone())
			if spawnEnabled {
				spawnTool := tools.NewSpawnTool(subagentManager)
				spawnTool.SetSpawner(NewSubTurnSpawner(al))
				currentAgentID := agentID
				spawnTool.SetAllowlistChecker(func(targetAgentID string) bool {
					return registry.CanSpawnSubagent(currentAgentID, targetAgentID)
				})

				agent.Tools.Register(spawnTool)

				// Also register the synchronous subagent tool
				subagentTool := tools.NewSubagentTool(subagentManager)
				subagentTool.SetSpawner(NewSubTurnSpawner(al))
				agent.Tools.Register(subagentTool)
			}
			if spawnStatusEnabled {
				agent.Tools.Register(tools.NewSpawnStatusTool(subagentManager))
			}
		} else if (spawnEnabled || spawnStatusEnabled) && !cfg.Tools.IsToolEnabled("subagent") {
			logger.WarnCF("agent", "spawn/spawn_status tools require subagent to be enabled", nil)
		}
	}
}

// Run 吱动 AgentLoop 的主消息处理循环
//
// 这是是 AgentLoop 的主入口函数,它:
//   1. 从 InboundChan 读取用户消息
//   2. 对每条消息启动一个处理协程
//   3. 在处理期间,如果有转向目标匹配,启动 drain 协程将后续消息路由到转向队列
//   4. 处理完成后,检查是否有排队的转向消息需要续轮处理
//   5. 循环直到 ctx 被取消或 Stop() 被调用
//
// 关键设计:
//   - 消息处理是异步的(通过 goroutine),支持并发处理
//   - drain 机制确保在 LLM 对话中途中,同一会话的新消息被注入到当前对话(转向)而不是排队等待
//   - defer 机制确保 typing stop 在处理完成(无论成功/失败)后都会被调用
//   - 续轮机制允许在 turn 完成后继续处理排队的转向消息
func (al *AgentLoop) Run(ctx context.Context) error {
	al.running.Store(true)

	if err := al.ensureHooksInitialized(ctx); err != nil {
		return err
	}
	if err := al.ensureMCPInitialized(ctx); err != nil {
		return err
	}

	idleTicker := time.NewTicker(100 * time.Millisecond)
	defer idleTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-idleTicker.C:
			if !al.running.Load() {
				return nil
			}
		case msg, ok := <-al.bus.InboundChan():
			if !ok {
				return nil
			}

			// Start a goroutine that drains the bus while processMessage is
			// running. Only messages that resolve to the active turn scope are
			// redirected into steering; other inbound messages are requeued.
			drainCancel := func() {}
			if activeScope, activeAgentID, ok := al.resolveSteeringTarget(msg); ok {
				drainCtx, cancel := context.WithCancel(ctx)
				drainCancel = cancel
				go al.drainBusToSteering(drainCtx, activeScope, activeAgentID)
			}

			// Process message
			func() {
				defer func() {
					if al.channelManager != nil {
						al.channelManager.InvokeTypingStop(msg.Channel, msg.ChatID)
					}
				}()
				// TODO: Re-enable media cleanup after inbound media is properly consumed by the agent.
				// Currently disabled because files are deleted before the LLM can access their content.
				// defer func() {
				// 	if al.mediaStore != nil && msg.MediaScope != "" {
				// 		if releaseErr := al.mediaStore.ReleaseAll(msg.MediaScope); releaseErr != nil {
				// 			logger.WarnCF("agent", "Failed to release media", map[string]any{
				// 				"scope": msg.MediaScope,
				// 				"error": releaseErr.Error(),
				// 			})
				// 		}
				// 	}
				// }()

				drainCanceled := false
				cancelDrain := func() {
					if drainCanceled {
						return
					}
					drainCancel()
					drainCanceled = true
				}
				defer cancelDrain()

				response, err := al.processMessage(ctx, msg)
				if err != nil {
					response = fmt.Sprintf("Error processing message: %v", err)
				}
				finalResponse := response

				target, targetErr := al.buildContinuationTarget(msg)
				if targetErr != nil {
					logger.WarnCF("agent", "Failed to build steering continuation target",
						map[string]any{
							"channel": msg.Channel,
							"error":   targetErr.Error(),
						})
					return
				}
				if target == nil {
					cancelDrain()
					if finalResponse != "" {
						al.publishResponseIfNeeded(ctx, msg.Channel, msg.ChatID, finalResponse)
					}
					return
				}

				for al.pendingSteeringCountForScope(target.SessionKey) > 0 {
					logger.InfoCF("agent", "Continuing queued steering after turn end",
						map[string]any{
							"channel":     target.Channel,
							"chat_id":     target.ChatID,
							"session_key": target.SessionKey,
							"queue_depth": al.pendingSteeringCountForScope(target.SessionKey),
						})

					continued, continueErr := al.Continue(ctx, target.SessionKey, target.Channel, target.ChatID)
					if continueErr != nil {
						logger.WarnCF("agent", "Failed to continue queued steering",
							map[string]any{
								"channel": target.Channel,
								"chat_id": target.ChatID,
								"error":   continueErr.Error(),
							})
						return
					}
					if continued == "" {
						return
					}

					finalResponse = continued
				}

				cancelDrain()

				for al.pendingSteeringCountForScope(target.SessionKey) > 0 {
					logger.InfoCF("agent", "Draining steering queued during turn shutdown",
						map[string]any{
							"channel":     target.Channel,
							"chat_id":     target.ChatID,
							"session_key": target.SessionKey,
							"queue_depth": al.pendingSteeringCountForScope(target.SessionKey),
						})

					continued, continueErr := al.Continue(ctx, target.SessionKey, target.Channel, target.ChatID)
					if continueErr != nil {
						logger.WarnCF("agent", "Failed to continue queued steering after shutdown drain",
							map[string]any{
								"channel": target.Channel,
								"chat_id": target.ChatID,
								"error":   continueErr.Error(),
							})
						return
					}
					if continued == "" {
						break
					}

					finalResponse = continued
				}

				if finalResponse != "" {
					al.publishResponseIfNeeded(ctx, target.Channel, target.ChatID, finalResponse)
				}
			}()
		}
	}
}

// drainBusToSteering 消息总线消费与转向注入
//
// 在一个活跃的 Turn(对话轮)执行期间,此函数作为后台 goroutine 运行,
// 持续从消息总线消费新的入站消息:
//
// 消息路由规则:
//   - 匹配当前活动 scope 的消息 → 注入到转向队列(steeringQueue)
//     这些消息会在 LLM 下一次迭代时被注入到上下文中,实现"对话中途中插话"效果
//   - 不匹配的消息 → 重新入队(requeueInboundMessage)
//     等待当前 Turn 宝成后再被处理
//
// 阻塞行为:
//   - 第一条消息: 阻塞等待(直到 ctx 取消或消息到达)
//   - 后续消息: 非阻塞消费(立即返回当无消息时)
//
// 这个机制是 PicoClaw "对话中转向"功能的核心实现:
//   用户: "帮我查下天气"
//   Agent: [正在查天气...]
//   用户: "顺便也查下明天的" ← 这条消息通过 drain 被注入到当前对话
//   Agent: "今天晴,明天多云" ← 两条消息一起被处理
func (al *AgentLoop) drainBusToSteering(ctx context.Context, activeScope, activeAgentID string) {
	blocking := true
	for {
		var msg bus.InboundMessage

		if blocking {
			// Block waiting for the first available message or ctx cancellation.
			select {
			case <-ctx.Done():
				return
			case m, ok := <-al.bus.InboundChan():
				if !ok {
					return
				}
				msg = m
			}
		} else {
			// Non-blocking: drain any remaining queued messages, return when empty.
			select {
			case m, ok := <-al.bus.InboundChan():
				if !ok {
					return
				}
				msg = m
			default:
				return
			}
		}
		blocking = false

		msgScope, _, scopeOK := al.resolveSteeringTarget(msg)
		if !scopeOK || msgScope != activeScope {
			if err := al.requeueInboundMessage(msg); err != nil {
				logger.WarnCF("agent", "Failed to requeue non-steering inbound message", map[string]any{
					"error":     err.Error(),
					"channel":   msg.Channel,
					"sender_id": msg.SenderID,
				})
			}
			continue
		}

		// Transcribe audio if needed before steering, so the agent sees text.
		msg, _ = al.transcribeAudioInMessage(ctx, msg)

		logger.InfoCF("agent", "Redirecting inbound message to steering queue",
			map[string]any{
				"channel":     msg.Channel,
				"sender_id":   msg.SenderID,
				"content_len": len(msg.Content),
				"scope":       activeScope,
			})

		if err := al.enqueueSteeringMessage(activeScope, activeAgentID, providers.Message{
			Role:    "user",
			Content: msg.Content,
			Media:   append([]string(nil), msg.Media...),
		}); err != nil {
			logger.WarnCF("agent", "Failed to steer message, will be lost",
				map[string]any{
					"error":   err.Error(),
					"channel": msg.Channel,
				})
		}
	}
}

// Stop 停止 AgentLoop 的主消息处理循环
//
// 设置 running 标志为 false,主循环中的 select 将在下次检查时退出。
func (al *AgentLoop) Stop() {
	al.running.Store(false)
}

// publishResponseIfNeeded 在必要时将响应发布到消息总线
//
// 如果响应不为空且本轮没有通过 message 巯具发送过消息,则通过总线发送。
// 这个检查是为了避免重复发送:当 LLM 使用 message 巯具主动发送消息时,
// 不需要再通过总线发送一次相同的消息。
//
// 参数:
//   - ctx: 上下文
//   - channel: 目标通道
//   - chatID: 目标聊天 ID
//   - response: 响应内容
func (al *AgentLoop) publishResponseIfNeeded(ctx context.Context, channel, chatID, response string) {
	if response == "" {
		return
	}

	alreadySent := false
	defaultAgent := al.GetRegistry().GetDefaultAgent()
	if defaultAgent != nil {
		if tool, ok := defaultAgent.Tools.Get("message"); ok {
			if mt, ok := tool.(*tools.MessageTool); ok {
				alreadySent = mt.HasSentInRound()
			}
		}
	}

	if alreadySent {
		logger.DebugCF(
			"agent",
			"Skipped outbound (message tool already sent)",
			map[string]any{"channel": channel},
		)
		return
	}

	al.bus.PublishOutbound(ctx, bus.OutboundMessage{
		Channel: channel,
		ChatID:  chatID,
		Content: response,
	})
	logger.InfoCF("agent", "Published outbound response",
		map[string]any{
			"channel":     channel,
			"chat_id":     chatID,
			"content_len": len(response),
		})
}

// buildContinuationTarget 枾建续轮目标
//
// 在 Turn 宝成后,检查是否需要继续处理排队中的转向消息。
// 如果消息来自 "system" 通道,则返回 nil(不需要续轮)。
//
// 参数:
//   - msg: 原始入站消息
//
// 返回:
//   - *continuationTarget: 续轮目标(包含会话键、通道、聊天ID)
//   - error: 路由解析错误
func (al *AgentLoop) buildContinuationTarget(msg bus.InboundMessage) (*continuationTarget, error) {
	if msg.Channel == "system" {
		return nil, nil
	}

	route, _, err := al.resolveMessageRoute(msg)
	if err != nil {
		return nil, err
	}

	return &continuationTarget{
		SessionKey: resolveScopeKey(route, msg.SessionKey),
		Channel:    msg.Channel,
		ChatID:     msg.ChatID,
	}, nil
}

// Close 释放 AgentLoop 持有的所有资源
//
// 在调用 Stop() 后应调用此函数以清理:
//   - MCP 禥理理器(关闭外部工具连接)
//   - Agent 注册表(关闭所有 Agent 的会话存储)
//   - 钩子管理器(卸载所有钩子)
//   - 事件总线(关闭所有订阅者)
func (al *AgentLoop) Close() {
	mcpManager := al.mcp.takeManager()

	if mcpManager != nil {
		if err := mcpManager.Close(); err != nil {
			logger.ErrorCF("agent", "Failed to close MCP manager",
				map[string]any{
					"error": err.Error(),
				})
		}
	}

	al.GetRegistry().Close()
	if al.hooks != nil {
		al.hooks.Close()
	}
	if al.eventBus != nil {
		al.eventBus.Close()
	}
}

// MountHook registers an in-process hook on the agent loop.
func (al *AgentLoop) MountHook(reg HookRegistration) error {
	if al == nil || al.hooks == nil {
		return fmt.Errorf("hook manager is not initialized")
	}
	return al.hooks.Mount(reg)
}

// UnmountHook removes a previously registered in-process hook.
func (al *AgentLoop) UnmountHook(name string) {
	if al == nil || al.hooks == nil {
		return
	}
	al.hooks.Unmount(name)
}

// SubscribeEvents registers a subscriber for agent-loop events.
func (al *AgentLoop) SubscribeEvents(buffer int) EventSubscription {
	if al == nil || al.eventBus == nil {
		ch := make(chan Event)
		close(ch)
		return EventSubscription{C: ch}
	}
	return al.eventBus.Subscribe(buffer)
}

// UnsubscribeEvents removes a previously registered event subscriber.
func (al *AgentLoop) UnsubscribeEvents(id uint64) {
	if al == nil || al.eventBus == nil {
		return
	}
	al.eventBus.Unsubscribe(id)
}

// EventDrops returns the number of dropped events for the given kind.
func (al *AgentLoop) EventDrops(kind EventKind) int64 {
	if al == nil || al.eventBus == nil {
		return 0
	}
	return al.eventBus.Dropped(kind)
}

type turnEventScope struct {
	agentID    string
	sessionKey string
	turnID     string
}

func (al *AgentLoop) newTurnEventScope(agentID, sessionKey string) turnEventScope {
	seq := al.turnSeq.Add(1)
	return turnEventScope{
		agentID:    agentID,
		sessionKey: sessionKey,
		turnID:     fmt.Sprintf("%s-turn-%d", agentID, seq),
	}
}

func (ts turnEventScope) meta(iteration int, source, tracePath string) EventMeta {
	return EventMeta{
		AgentID:    ts.agentID,
		TurnID:     ts.turnID,
		SessionKey: ts.sessionKey,
		Iteration:  iteration,
		Source:     source,
		TracePath:  tracePath,
	}
}

func (al *AgentLoop) emitEvent(kind EventKind, meta EventMeta, payload any) {
	evt := Event{
		Kind:    kind,
		Meta:    meta,
		Payload: payload,
	}

	if al == nil || al.eventBus == nil {
		return
	}

	al.logEvent(evt)

	al.eventBus.Emit(evt)
}

func cloneEventArguments(args map[string]any) map[string]any {
	if len(args) == 0 {
		return nil
	}

	cloned := make(map[string]any, len(args))
	for k, v := range args {
		cloned[k] = v
	}
	return cloned
}

func (al *AgentLoop) hookAbortError(ts *turnState, stage string, decision HookDecision) error {
	reason := decision.Reason
	if reason == "" {
		reason = "hook requested turn abort"
	}

	err := fmt.Errorf("hook aborted turn during %s: %s", stage, reason)
	al.emitEvent(
		EventKindError,
		ts.eventMeta("hooks", "turn.error"),
		ErrorPayload{
			Stage:   "hook." + stage,
			Message: err.Error(),
		},
	)
	return err
}

func hookDeniedToolContent(prefix, reason string) string {
	if reason == "" {
		return prefix
	}
	return prefix + ": " + reason
}

func (al *AgentLoop) logEvent(evt Event) {
	fields := map[string]any{
		"event_kind":  evt.Kind.String(),
		"agent_id":    evt.Meta.AgentID,
		"turn_id":     evt.Meta.TurnID,
		"session_key": evt.Meta.SessionKey,
		"iteration":   evt.Meta.Iteration,
	}

	if evt.Meta.TracePath != "" {
		fields["trace"] = evt.Meta.TracePath
	}
	if evt.Meta.Source != "" {
		fields["source"] = evt.Meta.Source
	}

	switch payload := evt.Payload.(type) {
	case TurnStartPayload:
		fields["channel"] = payload.Channel
		fields["chat_id"] = payload.ChatID
		fields["user_len"] = len(payload.UserMessage)
		fields["media_count"] = payload.MediaCount
	case TurnEndPayload:
		fields["status"] = payload.Status
		fields["iterations_total"] = payload.Iterations
		fields["duration_ms"] = payload.Duration.Milliseconds()
		fields["final_len"] = payload.FinalContentLen
	case LLMRequestPayload:
		fields["model"] = payload.Model
		fields["messages"] = payload.MessagesCount
		fields["tools"] = payload.ToolsCount
		fields["max_tokens"] = payload.MaxTokens
	case LLMDeltaPayload:
		fields["content_delta_len"] = payload.ContentDeltaLen
		fields["reasoning_delta_len"] = payload.ReasoningDeltaLen
	case LLMResponsePayload:
		fields["content_len"] = payload.ContentLen
		fields["tool_calls"] = payload.ToolCalls
		fields["has_reasoning"] = payload.HasReasoning
	case LLMRetryPayload:
		fields["attempt"] = payload.Attempt
		fields["max_retries"] = payload.MaxRetries
		fields["reason"] = payload.Reason
		fields["error"] = payload.Error
		fields["backoff_ms"] = payload.Backoff.Milliseconds()
	case ContextCompressPayload:
		fields["reason"] = payload.Reason
		fields["dropped_messages"] = payload.DroppedMessages
		fields["remaining_messages"] = payload.RemainingMessages
	case SessionSummarizePayload:
		fields["summarized_messages"] = payload.SummarizedMessages
		fields["kept_messages"] = payload.KeptMessages
		fields["summary_len"] = payload.SummaryLen
		fields["omitted_oversized"] = payload.OmittedOversized
	case ToolExecStartPayload:
		fields["tool"] = payload.Tool
		fields["args_count"] = len(payload.Arguments)
	case ToolExecEndPayload:
		fields["tool"] = payload.Tool
		fields["duration_ms"] = payload.Duration.Milliseconds()
		fields["for_llm_len"] = payload.ForLLMLen
		fields["for_user_len"] = payload.ForUserLen
		fields["is_error"] = payload.IsError
		fields["async"] = payload.Async
	case ToolExecSkippedPayload:
		fields["tool"] = payload.Tool
		fields["reason"] = payload.Reason
	case SteeringInjectedPayload:
		fields["count"] = payload.Count
		fields["total_content_len"] = payload.TotalContentLen
	case FollowUpQueuedPayload:
		fields["source_tool"] = payload.SourceTool
		fields["channel"] = payload.Channel
		fields["chat_id"] = payload.ChatID
		fields["content_len"] = payload.ContentLen
	case InterruptReceivedPayload:
		fields["interrupt_kind"] = payload.Kind
		fields["role"] = payload.Role
		fields["content_len"] = payload.ContentLen
		fields["queue_depth"] = payload.QueueDepth
		fields["hint_len"] = payload.HintLen
	case SubTurnSpawnPayload:
		fields["child_agent_id"] = payload.AgentID
		fields["label"] = payload.Label
	case SubTurnEndPayload:
		fields["child_agent_id"] = payload.AgentID
		fields["status"] = payload.Status
	case SubTurnResultDeliveredPayload:
		fields["target_channel"] = payload.TargetChannel
		fields["target_chat_id"] = payload.TargetChatID
		fields["content_len"] = payload.ContentLen
	case ErrorPayload:
		fields["stage"] = payload.Stage
		fields["error"] = payload.Message
	}

	logger.InfoCF("eventbus", fmt.Sprintf("Agent event: %s", evt.Kind.String()), fields)
}

// RegisterTool 向所有 Agent 注册一个新的工具
//
// 此方法用于在 AgentLoop 宍行时动态注册工具(如 MCP 工具)。
// 巯具会被注册到注册表中每个 Agent 的工具集中。
//
// 参数:
//   - tool: 要注册的工具实例
func (al *AgentLoop) RegisterTool(tool tools.Tool) {
	registry := al.GetRegistry()
	for _, agentID := range registry.ListAgentIDs() {
		if agent, ok := registry.GetAgent(agentID); ok {
			agent.Tools.Register(tool)
		}
	}
}

// SetChannelManager 注入渠道管理器
//
// 渠道管理器用于:
//   - 发送"正在输入"指示(typing indicator)
//   - 管理占位消息("思考中...")
//   - 发送响应消息
//   - 获取渠道特定配置(如推理通道 ID)
func (al *AgentLoop) SetChannelManager(cm *channels.Manager) {
	al.channelManager = cm
}

// ReloadProviderAndConfig 原子性地切换 LLM Provider 和配置
//
// 此方法支持运行时热重载,不重启服务即可切换模型。
// 使用写锁确保读取器看到一致的 provider+config 对。
//
// 重载流程:
//   1. 验证输入(provider 和 config 不为 nil)
//   2. 在 goroutine 中创建新的 Agent 注册表(可能 panic,使用 recover 保护)
//   3. 重新注册共享工具到新注册表
//   4. 获取写锁,原子切换 config + registry + fallback chain
//   5. 重新配置钩子管理器
//   6. 关闭旧 provider(带 100ms 等待,让在途请求完成)
//
// 注意: 使用 context 控制超时,支持从调用方取消
func (al *AgentLoop) ReloadProviderAndConfig(
	ctx context.Context,
	provider providers.LLMProvider,
	cfg *config.Config,
) error {
	// Validate inputs
	if provider == nil {
		return fmt.Errorf("provider cannot be nil")
	}
	if cfg == nil {
		return fmt.Errorf("config cannot be nil")
	}

	// Create new registry with updated config and provider
	// Wrap in defer/recover to handle any panics gracefully
	var registry *AgentRegistry
	var panicErr error
	done := make(chan struct{}, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				panicErr = fmt.Errorf("panic during registry creation: %v", r)
				logger.ErrorCF("agent", "Panic during registry creation",
					map[string]any{"panic": r})
			}
			close(done)
		}()

		registry = NewAgentRegistry(cfg, provider)
	}()

	// Wait for completion or context cancellation
	select {
	case <-done:
		if registry == nil {
			if panicErr != nil {
				return fmt.Errorf("registry creation failed: %w", panicErr)
			}
			return fmt.Errorf("registry creation failed (nil result)")
		}
	case <-ctx.Done():
		return fmt.Errorf("context canceled during registry creation: %w", ctx.Err())
	}

	// Check context again before proceeding
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context canceled after registry creation: %w", err)
	}

	// Ensure shared tools are re-registered on the new registry
	registerSharedTools(al, cfg, al.bus, registry, provider)

	// Atomically swap the config and registry under write lock
	// This ensures readers see a consistent pair
	al.mu.Lock()
	oldRegistry := al.registry

	// Store new values
	al.cfg = cfg
	al.registry = registry

	// Also update fallback chain with new config
	al.fallback = providers.NewFallbackChain(providers.NewCooldownTracker())

	al.mu.Unlock()

	al.hookRuntime.reset(al)
	configureHookManagerFromConfig(al.hooks, cfg)

	// Close old provider after releasing the lock
	// This prevents blocking readers while closing
	if oldProvider, ok := extractProvider(oldRegistry); ok {
		if stateful, ok := oldProvider.(providers.StatefulProvider); ok {
			// Give in-flight requests a moment to complete
			// Use a reasonable timeout that balances cleanup vs resource usage
			select {
			case <-time.After(100 * time.Millisecond):
				stateful.Close()
			case <-ctx.Done():
				// Context canceled, close immediately but log warning
				logger.WarnCF("agent", "Context canceled during provider cleanup, forcing close",
					map[string]any{"error": ctx.Err()})
				stateful.Close()
			}
		}
	}

	logger.InfoCF("agent", "Provider and config reloaded successfully",
		map[string]any{
			"model": cfg.Agents.Defaults.GetModelName(),
		})

	return nil
}

// GetRegistry 返回当前的 Agent 注册表(线程安全)
//
// 使用读锁保护,确保在 ReloadProviderAndConfig 期间不会读到不一致的状态。
func (al *AgentLoop) GetRegistry() *AgentRegistry {
	al.mu.RLock()
	defer al.mu.RUnlock()
	return al.registry
}

// GetConfig 返回当前的配置(线程安全)
func (al *AgentLoop) GetConfig() *config.Config {
	al.mu.RLock()
	defer al.mu.RUnlock()
	return al.cfg
}

// SetMediaStore 注入媒体存储实例
//
// 媒体存储用于:
//   - 存储用户上传的图片/音频/视频
//   - 存储工具生成的文件
//   - 提供 media:// URI 的解析
//
// 同时将 store 传播到所有 Agent 的工具注册表。
func (al *AgentLoop) SetMediaStore(s media.MediaStore) {
	al.mediaStore = s

	// Propagate store to all registered tools that can emit media.
	registry := al.GetRegistry()
	for _, agentID := range registry.ListAgentIDs() {
		if agent, ok := registry.GetAgent(agentID); ok {
			agent.Tools.SetMediaStore(s)
		}
	}
}

// SetTranscriber 注入语音转录器
//
// 转录器将用户发送的音频消息转为文字,使 LLM 能理解语音内容。
func (al *AgentLoop) SetTranscriber(t voice.Transcriber) {
	al.transcriber = t
}

// SetReloadFunc 设置配置热重载回调函数
//
// 此回调由外部注入,当 Agent 收到 /reload 等命令时调用。
// 典型实现是重新读取配置文件并调用 ReloadProviderAndConfig。
func (al *AgentLoop) SetReloadFunc(fn func() error) {
	al.reloadFunc = fn
}

// audioAnnotationRe 匹配音频注释的正则表达式
// 格式: [voice] 或 [audio] 或 [voice:描述] 等
var audioAnnotationRe = regexp.MustCompile(`\[(voice|audio)(?::[^\]]*)?\]`)

// transcribeAudioInMessage 转录消息中的音频附件
//
// 处理流程:
//   1. 检查是否有可用的转录器和媒体存储
//   2. 遍历消息中的 media:// 引用,解析为文件路径
//   3. 对每个音频文件调用转录器,将语音转为文字
//   4. 替换消息内容中的音频注释为转录文本 [voice: 转录文字]
//   5. 如果有多余的转录文本(没有对应的注释),追加到消息末尾
//
// 参数:
//   - ctx: 上下文
//   - msg: 入站消息
//
// 返回:
//   - bus.InboundMessage: 修改后的消息(内容中音频注释被替换为转录文本)
//   - bool: 是否有音频被转录
func (al *AgentLoop) transcribeAudioInMessage(ctx context.Context, msg bus.InboundMessage) (bus.InboundMessage, bool) {
	if al.transcriber == nil || al.mediaStore == nil || len(msg.Media) == 0 {
		return msg, false
	}

	// Transcribe each audio media ref in order.
	var transcriptions []string
	for _, ref := range msg.Media {
		path, meta, err := al.mediaStore.ResolveWithMeta(ref)
		if err != nil {
			logger.WarnCF("voice", "Failed to resolve media ref", map[string]any{"ref": ref, "error": err})
			continue
		}
		if !utils.IsAudioFile(meta.Filename, meta.ContentType) {
			continue
		}
		result, err := al.transcriber.Transcribe(ctx, path)
		if err != nil {
			logger.WarnCF("voice", "Transcription failed", map[string]any{"ref": ref, "error": err})
			transcriptions = append(transcriptions, "")
			continue
		}
		transcriptions = append(transcriptions, result.Text)
	}

	if len(transcriptions) == 0 {
		return msg, false
	}

	al.sendTranscriptionFeedback(ctx, msg.Channel, msg.ChatID, msg.MessageID, transcriptions)

	// Replace audio annotations sequentially with transcriptions.
	idx := 0
	newContent := audioAnnotationRe.ReplaceAllStringFunc(msg.Content, func(match string) string {
		if idx >= len(transcriptions) {
			return match
		}
		text := transcriptions[idx]
		idx++
		return "[voice: " + text + "]"
	})

	// Append any remaining transcriptions not matched by an annotation.
	for ; idx < len(transcriptions); idx++ {
		newContent += "\n[voice: " + transcriptions[idx] + "]"
	}

	msg.Content = newContent
	return msg, true
}

// sendTranscriptionFeedback 向用户发送音频转录反馈
//
// 如果配置了 EchoTranscription,则将转录文本同步发送给用户。
// 使用 Manager.SendMessage 确保:
//   - 经速率限制和拆分处理
//   - 与后续的占位消息保持顺序(转录反馈在占位消息之前)
//
// 参数:
//   - ctx: 上下文
//   - channel: 目标通道
//   - chatID: 目标聊天 ID
//   - messageID: 原始消息 ID(用于回复引用)
//   - validTexts: 非空的转录文本列表
func (al *AgentLoop) sendTranscriptionFeedback(
	ctx context.Context,
	channel, chatID, messageID string,
	validTexts []string,
) {
	if !al.cfg.Voice.EchoTranscription {
		return
	}
	if al.channelManager == nil {
		return
	}

	var nonEmpty []string
	for _, t := range validTexts {
		if t != "" {
			nonEmpty = append(nonEmpty, t)
		}
	}

	var feedbackMsg string
	if len(nonEmpty) > 0 {
		feedbackMsg = "Transcript: " + strings.Join(nonEmpty, "\n")
	} else {
		feedbackMsg = "No voice detected in the audio"
	}

	err := al.channelManager.SendMessage(ctx, bus.OutboundMessage{
		Channel:          channel,
		ChatID:           chatID,
		Content:          feedbackMsg,
		ReplyToMessageID: messageID,
	})
	if err != nil {
		logger.WarnCF("voice", "Failed to send transcription feedback", map[string]any{"error": err.Error()})
	}
}

// inferMediaType 根据文件名和 MIME 类型推断媒体类型
//
// 推断逻辑:
//   1. 优先使用 Content-Type(如 "image/png" → "image")
//   2. 如果 Content-Type 不明确,使用文件扩展名(.jpg → "image")
//   3. 兜底返回 "file"(通用文件类型)
//
// 参数:
//   - filename: 文件名(如 "photo.jpg")
//   - contentType: MIME 类型(如 "image/jpeg")
//
// 返回: "image" | "audio" | "video" | "file"
func inferMediaType(filename, contentType string) string {
	ct := strings.ToLower(contentType)
	fn := strings.ToLower(filename)

	if strings.HasPrefix(ct, "image/") {
		return "image"
	}
	if strings.HasPrefix(ct, "audio/") || ct == "application/ogg" {
		return "audio"
	}
	if strings.HasPrefix(ct, "video/") {
		return "video"
	}

	// Fallback: infer from extension
	ext := filepath.Ext(fn)
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp", ".svg":
		return "image"
	case ".mp3", ".wav", ".ogg", ".m4a", ".flac", ".aac", ".wma", ".opus":
		return "audio"
	case ".mp4", ".avi", ".mov", ".webm", ".mkv":
		return "video"
	}

	return "file"
}

// RecordLastChannel 记录工作区最后活跃的通道
//
// 使用原子状态保存机制防止崩溃时数据丢失。
// 用于心跳通知等场景,让 Agent 知道上次对话发生在哪个通道。
func (al *AgentLoop) RecordLastChannel(channel string) error {
	if al.state == nil {
		return nil
	}
	return al.state.SetLastChannel(channel)
}

// RecordLastChatID 记录工作区最后活跃的聊天 ID
//
// 与 RecordLastChannel 配合使用,记录精确的聊天位置。
func (al *AgentLoop) RecordLastChatID(chatID string) error {
	if al.state == nil {
		return nil
	}
	return al.state.SetLastChatID(chatID)
}

// ProcessDirect 直接处理一条消息(不通过消息总线)
//
// 用于 CLI 等同步场景,直接将消息发送给 LLM 并返回响应。
// 通道固定为 "cli",聊天 ID 固定为 "direct"。
//
// 参数:
//   - ctx: 上下文
//   - content: 消息内容
//   - sessionKey: 会话标识符
//
// 返回: LLM 的响应文本
func (al *AgentLoop) ProcessDirect(
	ctx context.Context,
	content, sessionKey string,
) (string, error) {
	return al.ProcessDirectWithChannel(ctx, content, sessionKey, "cli", "direct")
}

// ProcessDirectWithChannel 直接处理一条消息到指定通道
//
// 与 ProcessDirect 类似,但允许指定通道和聊天 ID。
// 用于定时任务(如 heartbeat)等需要精确控制目标通道的场景。
//
// 参数:
//   - ctx: 上下文
//   - content: 消息内容
//   - sessionKey: 会话标识符
//   - channel: 目标通道(如 "telegram")
//   - chatID: 目标聊天 ID
func (al *AgentLoop) ProcessDirectWithChannel(
	ctx context.Context,
	content, sessionKey, channel, chatID string,
) (string, error) {
	if err := al.ensureHooksInitialized(ctx); err != nil {
		return "", err
	}
	if err := al.ensureMCPInitialized(ctx); err != nil {
		return "", err
	}

	msg := bus.InboundMessage{
		Channel:    channel,
		SenderID:   "cron",
		ChatID:     chatID,
		Content:    content,
		SessionKey: sessionKey,
	}

	return al.processMessage(ctx, msg)
}

// ProcessHeartbeat 处理心跳请求
//
// 心跳是一种特殊的消息处理:
//   - 不加载会话历史(NoHistory=true)
//   - 不累织上下文(每次心跳独立)
//   - 抑制工具反馈消息(SuppressToolFeedback=true)
//   - 不自动发送响应(SendResponse=false)
//
// 典型用途:
//   - 定时健康检查
//   - 定时任务触发(如 "每天早上 9 点报告待办事项")
//   - 通知提醒(如 "提醒我下午 3 点开会")
func (al *AgentLoop) ProcessHeartbeat(
	ctx context.Context,
	content, channel, chatID string,
) (string, error) {
	if err := al.ensureHooksInitialized(ctx); err != nil {
		return "", err
	}
	if err := al.ensureMCPInitialized(ctx); err != nil {
		return "", err
	}

	agent := al.GetRegistry().GetDefaultAgent()
	if agent == nil {
		return "", fmt.Errorf("no default agent for heartbeat")
	}
	return al.runAgentLoop(ctx, agent, processOptions{
		SessionKey:           "heartbeat",
		Channel:              channel,
		ChatID:               chatID,
		UserMessage:          content,
		DefaultResponse:      defaultResponse,
		EnableSummary:        false,
		SendResponse:         false,
		SuppressToolFeedback: true,
		NoHistory:            true, // Don't load session history for heartbeat
	})
}

// processMessage 夿息整条入站消息的核心处理函数
//
// 夌理流程:
//   1. 音频转录: 如果消息包含音频附件,使用 transcriber 将音频转为文字
//   2. 綈息路由: 根据 routing 觡将消息分发到对应的 Agent
//   3. 嶈息工具状态重 重: 重置 message 巯具的 "已发送" 标志
//   4. 挂起技能处理: 如果有待应用的技能,将其加入选项
//   5. 趈费 LLM: 调用 runAgentLoop 进入主对话循环
//
// 返回:
//   - response: LLM 的最终响应文本
//   - err: 处理过程中的错误
func (al *AgentLoop) processMessage(ctx context.Context, msg bus.InboundMessage) (string, error) {
	// Add message preview to log (show full content for error messages)
	var logContent string
	if strings.Contains(msg.Content, "Error:") || strings.Contains(msg.Content, "error") {
		logContent = msg.Content // Full content for errors
	} else {
		logContent = utils.Truncate(msg.Content, 80)
	}
	logger.InfoCF(
		"agent",
		fmt.Sprintf("Processing message from %s:%s: %s", msg.Channel, msg.SenderID, logContent),
		map[string]any{
			"channel":     msg.Channel,
			"chat_id":     msg.ChatID,
			"sender_id":   msg.SenderID,
			"session_key": msg.SessionKey,
		},
	)

	var hadAudio bool
	msg, hadAudio = al.transcribeAudioInMessage(ctx, msg)

	// For audio messages the placeholder was deferred by the channel.
	// Now that transcription (and optional feedback) is done, send it.
	if hadAudio && al.channelManager != nil {
		al.channelManager.SendPlaceholder(ctx, msg.Channel, msg.ChatID)
	}

	// Route system messages to processSystemMessage
	if msg.Channel == "system" {
		return al.processSystemMessage(ctx, msg)
	}

	route, agent, routeErr := al.resolveMessageRoute(msg)
	if routeErr != nil {
		return "", routeErr
	}

	// Reset message-tool state for this round so we don't skip publishing due to a previous round.
	if tool, ok := agent.Tools.Get("message"); ok {
		if resetter, ok := tool.(interface{ ResetSentInRound() }); ok {
			resetter.ResetSentInRound()
		}
	}

	// Resolve session key from route, while preserving explicit agent-scoped keys.
	scopeKey := resolveScopeKey(route, msg.SessionKey)
	sessionKey := scopeKey

	logger.InfoCF("agent", "Routed message",
		map[string]any{
			"agent_id":      agent.ID,
			"scope_key":     scopeKey,
			"session_key":   sessionKey,
			"matched_by":    route.MatchedBy,
			"route_agent":   route.AgentID,
			"route_channel": route.Channel,
		})

	opts := processOptions{
		SessionKey:        sessionKey,
		Channel:           msg.Channel,
		ChatID:            msg.ChatID,
		SenderID:          msg.SenderID,
		SenderDisplayName: msg.Sender.DisplayName,
		UserMessage:       msg.Content,
		Media:             msg.Media,
		DefaultResponse:   defaultResponse,
		EnableSummary:     true,
		SendResponse:      false,
	}

	// context-dependent commands check their own Runtime fields and report
	// "unavailable" when the required capability is nil.
	if response, handled := al.handleCommand(ctx, msg, agent, &opts); handled {
		return response, nil
	}

	if pending := al.takePendingSkills(opts.SessionKey); len(pending) > 0 {
		opts.ForcedSkills = append(opts.ForcedSkills, pending...)
		logger.InfoCF("agent", "Applying pending skill override",
			map[string]any{
				"session_key": opts.SessionKey,
				"skills":      strings.Join(pending, ","),
			})
	}

	return al.runAgentLoop(ctx, agent, opts)
}

// resolveMessageRoute 解析消息路由,将消息匹配到对应的 Agent
//
// 路由决策基于以下维度:
//   - Channel: 通道名称(如 "telegram", "discord")
//   - AccountID: 账户 ID
//   - Peer: 对等方(如私聊 peer="direct",群聊 peer="group")
//   - ParentPeer: 父级对等方(用于回复场景)
//   - GuildID/TeamID: 服务器/团队 ID
//
// 如果没有匹配的路由规则,返回默认 Agent。
func (al *AgentLoop) resolveMessageRoute(msg bus.InboundMessage) (routing.ResolvedRoute, *AgentInstance, error) {
	registry := al.GetRegistry()
	route := registry.ResolveRoute(routing.RouteInput{
		Channel:    msg.Channel,
		AccountID:  inboundMetadata(msg, metadataKeyAccountID),
		Peer:       extractPeer(msg),
		ParentPeer: extractParentPeer(msg),
		GuildID:    inboundMetadata(msg, metadataKeyGuildID),
		TeamID:     inboundMetadata(msg, metadataKeyTeamID),
	})

	agent, ok := registry.GetAgent(route.AgentID)
	if !ok {
		agent = registry.GetDefaultAgent()
	}
	if agent == nil {
		return routing.ResolvedRoute{}, nil, fmt.Errorf("no agent available for route (agent_id=%s)", route.AgentID)
	}

	return route, agent, nil
}

// resolveScopeKey 解析会话作用域键
//
// 如果消息显式指定了 agent 前缀的会话键(如 "agent:coder:main"),
// 直接使用(保持 Agent 作用域隔离)。
// 否则使用路由系统生成的 SessionKey(基于路由规则)。
func resolveScopeKey(route routing.ResolvedRoute, msgSessionKey string) string {
	if msgSessionKey != "" && strings.HasPrefix(msgSessionKey, sessionKeyAgentPrefix) {
		return msgSessionKey
	}
	return route.SessionKey
}

// resolveSteeringTarget 解析消息的转向目标
//
// 转向(Steering)是 PicoClaw 的核心特性之一,允许在 LLM 对话进行中
// 将新的用户消息注入到当前对话。
//
// 返回:
//   - scopeKey: 会话作用域键
//   - agentID: 匹配的 Agent ID
//   - bool: 是否匹配到转向目标(非 system 通道的消息总是匹配)
func (al *AgentLoop) resolveSteeringTarget(msg bus.InboundMessage) (string, string, bool) {
	if msg.Channel == "system" {
		return "", "", false
	}

	route, agent, err := al.resolveMessageRoute(msg)
	if err != nil || agent == nil {
		return "", "", false
	}

	return resolveScopeKey(route, msg.SessionKey), agent.ID, true
}

// requeueInboundMessage 将入站消息重新放回消息总线
//
// 当 drain 协程收到不匹配当前活跃 scope 的消息时,
// 需要将消息重新发布到总线,以便在当前 Turn 宝成后处理。
//
// 注意: 使用独立的超时上下文(1秒),避免在主 Turn 取消时丢失消息。
func (al *AgentLoop) requeueInboundMessage(msg bus.InboundMessage) error {
	if al.bus == nil {
		return nil
	}
	pubCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	return al.bus.PublishOutbound(pubCtx, bus.OutboundMessage{
		Channel: msg.Channel,
		ChatID:  msg.ChatID,
		Content: msg.Content,
	})
}

// processSystemMessage 处理系统消息
//
// 系统消息是内部消息,格式为:
//   - Channel: "system"
//   - ChatID: "channel:chatID"(包含原始通道信息)
//   - Content: 系统消息内容(如异步工具结果)
//
// 处理流程:
//   1. 解析 ChatID 提取原始通道和聊天 ID
//   2. 提取消息内容(去除包装格式)
//   3. 跳过内部通道的消息(仅日志记录)
//   4. 使用默认 Agent 处理消息(带 [System: sender] 前缀)
func (al *AgentLoop) processSystemMessage(
	ctx context.Context,
	msg bus.InboundMessage,
) (string, error) {
	if msg.Channel != "system" {
		return "", fmt.Errorf(
			"processSystemMessage called with non-system message channel: %s",
			msg.Channel,
		)
	}

	logger.InfoCF("agent", "Processing system message",
		map[string]any{
			"sender_id": msg.SenderID,
			"chat_id":   msg.ChatID,
		})

	// Parse origin channel from chat_id (format: "channel:chat_id")
	var originChannel, originChatID string
	if idx := strings.Index(msg.ChatID, ":"); idx > 0 {
		originChannel = msg.ChatID[:idx]
		originChatID = msg.ChatID[idx+1:]
	} else {
		originChannel = "cli"
		originChatID = msg.ChatID
	}

	// Extract subagent result from message content
	// Format: "Task 'label' completed.\n\nResult:\n<actual content>"
	content := msg.Content
	if idx := strings.Index(content, "Result:\n"); idx >= 0 {
		content = content[idx+8:] // Extract just the result part
	}

	// Skip internal channels - only log, don't send to user
	if constants.IsInternalChannel(originChannel) {
		logger.InfoCF("agent", "Subagent completed (internal channel)",
			map[string]any{
				"sender_id":   msg.SenderID,
				"content_len": len(content),
				"channel":     originChannel,
			})
		return "", nil
	}

	// Use default agent for system messages
	agent := al.GetRegistry().GetDefaultAgent()
	if agent == nil {
		return "", fmt.Errorf("no default agent for system message")
	}

	// Use the origin session for context
	sessionKey := routing.BuildAgentMainSessionKey(agent.ID)

	return al.runAgentLoop(ctx, agent, processOptions{
		SessionKey:      sessionKey,
		Channel:         originChannel,
		ChatID:          originChatID,
		UserMessage:     fmt.Sprintf("[System: %s] %s", msg.SenderID, msg.Content),
		DefaultResponse: "Background task completed.",
		EnableSummary:   false,
		SendResponse:    true,
	})
}

// runAgentLoop 是 Turn 的顶层入口函数,负责启动一个完整的对话轮(Turn)
//
// 一个 Turn 包含:
//   1. 讱管理上下文预算(可能触发压缩)
//   2. 将用户消息存入会话历史
//   3. 调用 LLM 获取响应
//   4. 执行工具调用(如果有)
//   5. 将结果存入会话历史
//   6. 触发摘要(如果需要)
//   7. 发布后续消息(follow-ups)
//
// 参数:
//   - agent: 目标 Agent 实例
//   - opts: 夺理选项(包含会话键、通道、消息等)
//
// 返回:
//   - string: LLM 的最终响应
//   - error: 处理错误
func (al *AgentLoop) runAgentLoop(
	ctx context.Context,
	agent *AgentInstance,
	opts processOptions,
) (string, error) {
	// Record last channel for heartbeat notifications (skip internal channels and cli)
	if opts.Channel != "" && opts.ChatID != "" && !constants.IsInternalChannel(opts.Channel) {
		channelKey := fmt.Sprintf("%s:%s", opts.Channel, opts.ChatID)
		if err := al.RecordLastChannel(channelKey); err != nil {
			logger.WarnCF(
				"agent",
				"Failed to record last channel",
				map[string]any{"error": err.Error()},
			)
		}
	}

	ts := newTurnState(agent, opts, al.newTurnEventScope(agent.ID, opts.SessionKey))
	result, err := al.runTurn(ctx, ts)
	if err != nil {
		return "", err
	}
	if result.status == TurnEndStatusAborted {
		return "", nil
	}

	for _, followUp := range result.followUps {
		if pubErr := al.bus.PublishInbound(ctx, followUp); pubErr != nil {
			logger.WarnCF("agent", "Failed to publish follow-up after turn",
				map[string]any{
					"turn_id": ts.turnID,
					"error":   pubErr.Error(),
				})
		}
	}

	if opts.SendResponse && result.finalContent != "" {
		al.bus.PublishOutbound(ctx, bus.OutboundMessage{
			Channel: opts.Channel,
			ChatID:  opts.ChatID,
			Content: result.finalContent,
		})
	}

	if result.finalContent != "" {
		responsePreview := utils.Truncate(result.finalContent, 120)
		logger.InfoCF("agent", fmt.Sprintf("Response: %s", responsePreview),
			map[string]any{
				"agent_id":     agent.ID,
				"session_key":  opts.SessionKey,
				"iterations":   ts.currentIteration(),
				"final_length": len(result.finalContent),
			})
	}

	return result.finalContent, nil
}

func (al *AgentLoop) targetReasoningChannelID(channelName string) (chatID string) {
	if al.channelManager == nil {
		return ""
	}
	if ch, ok := al.channelManager.GetChannel(channelName); ok {
		return ch.ReasoningChannelID()
	}
	return ""
}

// handleReasoning 夑达推理(reasoning)内容到指定的通道
//
// 推理是 LLM 在生成响应时的中间思考过程,通常使用支持推理的模型
// (如 Claude 的 extended thinking)。推理内容会被发送到一个专门的"推理通道",
// 用于调试和透明度。
//
// 参数:
//   - ctx: 上下文(用于超时控制)
//   - reasoningContent: 推理内容文本
//   - channelName: 通道名称(如 "telegram")
//   - channelID: 聊天 ID(即推理通道的 ID)
func (al *AgentLoop) handleReasoning(
	ctx context.Context,
	reasoningContent, channelName, channelID string,
) {
	if reasoningContent == "" || channelName == "" || channelID == "" {
		return
	}

	// Check context cancellation before attempting to publish,
	// since PublishOutbound's select may race between send and ctx.Done().
	if ctx.Err() != nil {
		return
	}

	// Use a short timeout so the goroutine does not block indefinitely when
	// the outbound bus is full.  Reasoning output is best-effort; dropping it
	// is acceptable to avoid goroutine accumulation.
	pubCtx, pubCancel := context.WithTimeout(ctx, 5*time.Second)
	defer pubCancel()

	if err := al.bus.PublishOutbound(pubCtx, bus.OutboundMessage{
		Channel: channelName,
		ChatID:  channelID,
		Content: reasoningContent,
	}); err != nil {
		// Treat context.DeadlineExceeded / context.Canceled as expected
		// (bus full under load, or parent canceled).  Check the error
		// itself rather than ctx.Err(), because pubCtx may time out
		// (5 s) while the parent ctx is still active.
		// Also treat ErrBusClosed as expected — it occurs during normal
		// shutdown when the bus is closed before all goroutines finish.
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) ||
			errors.Is(err, bus.ErrBusClosed) {
			logger.DebugCF("agent", "Reasoning publish skipped (timeout/cancel)", map[string]any{
				"channel": channelName,
				"error":   err.Error(),
			})
		} else {
			logger.WarnCF("agent", "Failed to publish reasoning (best-effort)", map[string]any{
				"channel": channelName,
				"error":   err.Error(),
			})
		}
	}
}

// runTurn 执行一个完整的 Turn(对话轮)
//
// Turn 是 AgentLoop 中最核心的执行单元,代表一次完整的 LLM 对话轮。
// 一个 Turn 可能包含多次 LLM 调用(迭代),每次迭代可能执行工具调用。
//
// 执行流程:
//   1. 创建 turn 上下文(可取消)
//   2. 注册 active turn(支持中断)
//   3. 发出 TurnStart 事件
//   4. 加载会话历史(如果没有标记 NoHistory)
//   5. 管理上下文预算(可能触发压缩)
//   6. 将用户消息存入会话历史
//   7. 选择模型候选(可能使用轻量模型)
//   8. 进入主循环:
//      a. 检查硬中断/优雅中断
//      b. 轮询转向消息
//      c. 调用 LLM(支持重试)
//      d. 处理 LLM 响应:
//         - 纯文本响应 → 结束 Turn
//         - 工具调用 → 执行工具 → 继续循环
//      e. 检查是否所有工具已处理
//   9. 保存最终响应到会话历史
//  10. 触发摘要(如果需要)
//  11. 发出 TurnEnd 事件
//
// 中断机制:
//   - 硬中断(HardAbort): 立即终止 Turn,恢复会话到快照
//   - 优雅中断(GracefulInterrupt): 完成当前迭代后终止,不恢复快照
//   - 转向(Steering): 在对话中途中注入新的用户消息,改变对话方向
//
// 上下文管理:
//   - 当 token 超过预算时,自动压缩历史(丢弃旧消息)
//   - 压缩策略: 按完整 Turn 边界分割,保持工具调用序列完整性
//   - 最后兜底: 仅保留最近的用户消息
//
// 参数:
//   - ctx: 父上下文(用于取消和超时)
//   - ts: Turn 状态(包含 Agent、选项、迭代计数等)
//
// 返回:
//   - turnResult: Turn 结果(包含最终内容、后续消息、状态)
//   - error: 执行错误
func (al *AgentLoop) runTurn(ctx context.Context, ts *turnState) (turnResult, error) {
	turnCtx, turnCancel := context.WithCancel(ctx)
	defer turnCancel()
	ts.setTurnCancel(turnCancel)

	// Inject turnState and AgentLoop into context so tools (e.g. spawn) can retrieve them.
	turnCtx = withTurnState(turnCtx, ts)
	turnCtx = WithAgentLoop(turnCtx, al)

	al.registerActiveTurn(ts)
	defer al.clearActiveTurn(ts)

	turnStatus := TurnEndStatusCompleted
	defer func() {
		al.emitEvent(
			EventKindTurnEnd,
			ts.eventMeta("runTurn", "turn.end"),
			TurnEndPayload{
				Status:          turnStatus,
				Iterations:      ts.currentIteration(),
				Duration:        time.Since(ts.startedAt),
				FinalContentLen: ts.finalContentLen(),
			},
		)
	}()

	al.emitEvent(
		EventKindTurnStart,
		ts.eventMeta("runTurn", "turn.start"),
		TurnStartPayload{
			Channel:     ts.channel,
			ChatID:      ts.chatID,
			UserMessage: ts.userMessage,
			MediaCount:  len(ts.media),
		},
	)

	var history []providers.Message
	var summary string
	if !ts.opts.NoHistory {
		history = ts.agent.Sessions.GetHistory(ts.sessionKey)
		summary = ts.agent.Sessions.GetSummary(ts.sessionKey)
	}
	ts.captureRestorePoint(history, summary)

	messages := ts.agent.ContextBuilder.BuildMessages(
		history,
		summary,
		ts.userMessage,
		ts.media,
		ts.channel,
		ts.chatID,
		ts.opts.SenderID,
		ts.opts.SenderDisplayName,
		activeSkillNames(ts.agent, ts.opts)...,
	)

	cfg := al.GetConfig()
	maxMediaSize := cfg.Agents.Defaults.GetMaxMediaSize()
	messages = resolveMediaRefs(messages, al.mediaStore, maxMediaSize)

	if !ts.opts.NoHistory {
		toolDefs := ts.agent.Tools.ToProviderDefs()
		if isOverContextBudget(ts.agent.ContextWindow, messages, toolDefs, ts.agent.MaxTokens) {
			logger.WarnCF("agent", "Proactive compression: context budget exceeded before LLM call",
				map[string]any{"session_key": ts.sessionKey})
			if compression, ok := al.forceCompression(ts.agent, ts.sessionKey); ok {
				al.emitEvent(
					EventKindContextCompress,
					ts.eventMeta("runTurn", "turn.context.compress"),
					ContextCompressPayload{
						Reason:            ContextCompressReasonProactive,
						DroppedMessages:   compression.DroppedMessages,
						RemainingMessages: compression.RemainingMessages,
					},
				)
				ts.refreshRestorePointFromSession(ts.agent)
			}
			newHistory := ts.agent.Sessions.GetHistory(ts.sessionKey)
			newSummary := ts.agent.Sessions.GetSummary(ts.sessionKey)
			messages = ts.agent.ContextBuilder.BuildMessages(
				newHistory, newSummary, ts.userMessage,
				ts.media, ts.channel, ts.chatID,
				ts.opts.SenderID, ts.opts.SenderDisplayName,
				activeSkillNames(ts.agent, ts.opts)...,
			)
			messages = resolveMediaRefs(messages, al.mediaStore, maxMediaSize)
		}
	}

	// Save user message to session (from Incoming)
	if !ts.opts.NoHistory && (strings.TrimSpace(ts.userMessage) != "" || len(ts.media) > 0) {
		rootMsg := providers.Message{
			Role:    "user",
			Content: ts.userMessage,
			Media:   append([]string(nil), ts.media...),
		}
		if len(rootMsg.Media) > 0 {
			ts.agent.Sessions.AddFullMessage(ts.sessionKey, rootMsg)
		} else {
			ts.agent.Sessions.AddMessage(ts.sessionKey, rootMsg.Role, rootMsg.Content)
		}
		ts.recordPersistedMessage(rootMsg)
	}

	activeCandidates, activeModel, usedLight := al.selectCandidates(ts.agent, ts.userMessage, messages)
	activeProvider := ts.agent.Provider
	if usedLight && ts.agent.LightProvider != nil {
		activeProvider = ts.agent.LightProvider
	}
	pendingMessages := append([]providers.Message(nil), ts.opts.InitialSteeringMessages...)
	var finalContent string

turnLoop:
	for ts.currentIteration() < ts.agent.MaxIterations || len(pendingMessages) > 0 || func() bool {
		graceful, _ := ts.gracefulInterruptRequested()
		return graceful
	}() {
		if ts.hardAbortRequested() {
			turnStatus = TurnEndStatusAborted
			return al.abortTurn(ts)
		}

		iteration := ts.currentIteration() + 1
		ts.setIteration(iteration)
		ts.setPhase(TurnPhaseRunning)

		if iteration > 1 {
			if steerMsgs := al.dequeueSteeringMessagesForScope(ts.sessionKey); len(steerMsgs) > 0 {
				pendingMessages = append(pendingMessages, steerMsgs...)
			}
		} else if !ts.opts.SkipInitialSteeringPoll {
			if steerMsgs := al.dequeueSteeringMessagesForScopeWithFallback(ts.sessionKey); len(steerMsgs) > 0 {
				pendingMessages = append(pendingMessages, steerMsgs...)
			}
		}

		// Check if parent turn has ended (SubTurn support from HEAD)
		if ts.parentTurnState != nil && ts.IsParentEnded() {
			if !ts.critical {
				logger.InfoCF("agent", "Parent turn ended, non-critical SubTurn exiting gracefully", map[string]any{
					"agent_id":  ts.agentID,
					"iteration": iteration,
					"turn_id":   ts.turnID,
				})
				break
			}
			logger.InfoCF("agent", "Parent turn ended, critical SubTurn continues running", map[string]any{
				"agent_id":  ts.agentID,
				"iteration": iteration,
				"turn_id":   ts.turnID,
			})
		}

		// Poll for pending SubTurn results (from HEAD)
		if ts.pendingResults != nil {
			select {
			case result, ok := <-ts.pendingResults:
				if ok && result != nil && result.ForLLM != "" {
					content := al.cfg.FilterSensitiveData(result.ForLLM)
					msg := providers.Message{Role: "user", Content: fmt.Sprintf("[SubTurn Result] %s", content)}
					pendingMessages = append(pendingMessages, msg)
				}
			default:
				// No results available
			}
		}

		// Inject pending steering messages
		if len(pendingMessages) > 0 {
			resolvedPending := resolveMediaRefs(pendingMessages, al.mediaStore, maxMediaSize)
			totalContentLen := 0
			for i, pm := range pendingMessages {
				messages = append(messages, resolvedPending[i])
				totalContentLen += len(pm.Content)
				if !ts.opts.NoHistory {
					ts.agent.Sessions.AddFullMessage(ts.sessionKey, pm)
					ts.recordPersistedMessage(pm)
				}
				logger.InfoCF("agent", "Injected steering message into context",
					map[string]any{
						"agent_id":    ts.agent.ID,
						"iteration":   iteration,
						"content_len": len(pm.Content),
						"media_count": len(pm.Media),
					})
			}
			al.emitEvent(
				EventKindSteeringInjected,
				ts.eventMeta("runTurn", "turn.steering.injected"),
				SteeringInjectedPayload{
					Count:           len(pendingMessages),
					TotalContentLen: totalContentLen,
				},
			)
			pendingMessages = nil
		}

		logger.DebugCF("agent", "LLM iteration",
			map[string]any{
				"agent_id":  ts.agent.ID,
				"iteration": iteration,
				"max":       ts.agent.MaxIterations,
			})

		gracefulTerminal, _ := ts.gracefulInterruptRequested()
		providerToolDefs := ts.agent.Tools.ToProviderDefs()

		// Native web search support (from HEAD)
		_, hasWebSearch := ts.agent.Tools.Get("web_search")
		useNativeSearch := al.cfg.Tools.Web.PreferNative &&
			hasWebSearch &&
			func() bool {
				// Check if provider supports native search
				if ns, ok := ts.agent.Provider.(interface{ SupportsNativeSearch() bool }); ok {
					return ns.SupportsNativeSearch()
				}
				return false
			}()

		if useNativeSearch {
			// Filter out client-side web_search tool
			filtered := make([]providers.ToolDefinition, 0, len(providerToolDefs))
			for _, td := range providerToolDefs {
				if td.Function.Name != "web_search" {
					filtered = append(filtered, td)
				}
			}
			providerToolDefs = filtered
		}

		callMessages := messages
		if gracefulTerminal {
			callMessages = append(append([]providers.Message(nil), messages...), ts.interruptHintMessage())
			providerToolDefs = nil
			ts.markGracefulTerminalUsed()
		}

		llmOpts := map[string]any{
			"max_tokens":       ts.agent.MaxTokens,
			"temperature":      ts.agent.Temperature,
			"prompt_cache_key": ts.agent.ID,
		}
		if useNativeSearch {
			llmOpts["native_search"] = true
		}
		if ts.agent.ThinkingLevel != ThinkingOff {
			if tc, ok := ts.agent.Provider.(providers.ThinkingCapable); ok && tc.SupportsThinking() {
				llmOpts["thinking_level"] = string(ts.agent.ThinkingLevel)
			} else {
				logger.WarnCF("agent", "thinking_level is set but current provider does not support it, ignoring",
					map[string]any{"agent_id": ts.agent.ID, "thinking_level": string(ts.agent.ThinkingLevel)})
			}
		}

		llmModel := activeModel
		if al.hooks != nil {
			llmReq, decision := al.hooks.BeforeLLM(turnCtx, &LLMHookRequest{
				Meta:             ts.eventMeta("runTurn", "turn.llm.request"),
				Model:            llmModel,
				Messages:         callMessages,
				Tools:            providerToolDefs,
				Options:          llmOpts,
				Channel:          ts.channel,
				ChatID:           ts.chatID,
				GracefulTerminal: gracefulTerminal,
			})
			switch decision.normalizedAction() {
			case HookActionContinue, HookActionModify:
				if llmReq != nil {
					llmModel = llmReq.Model
					callMessages = llmReq.Messages
					providerToolDefs = llmReq.Tools
					llmOpts = llmReq.Options
				}
			case HookActionAbortTurn:
				turnStatus = TurnEndStatusError
				return turnResult{}, al.hookAbortError(ts, "before_llm", decision)
			case HookActionHardAbort:
				_ = ts.requestHardAbort()
				turnStatus = TurnEndStatusAborted
				return al.abortTurn(ts)
			}
		}

		al.emitEvent(
			EventKindLLMRequest,
			ts.eventMeta("runTurn", "turn.llm.request"),
			LLMRequestPayload{
				Model:         llmModel,
				MessagesCount: len(callMessages),
				ToolsCount:    len(providerToolDefs),
				MaxTokens:     ts.agent.MaxTokens,
				Temperature:   ts.agent.Temperature,
			},
		)

		logger.DebugCF("agent", "LLM request",
			map[string]any{
				"agent_id":          ts.agent.ID,
				"iteration":         iteration,
				"model":             llmModel,
				"messages_count":    len(callMessages),
				"tools_count":       len(providerToolDefs),
				"max_tokens":        ts.agent.MaxTokens,
				"temperature":       ts.agent.Temperature,
				"system_prompt_len": len(callMessages[0].Content),
			})
		logger.DebugCF("agent", "Full LLM request",
			map[string]any{
				"iteration":     iteration,
				"messages_json": formatMessagesForLog(callMessages),
				"tools_json":    formatToolsForLog(providerToolDefs),
			})

		callLLM := func(messagesForCall []providers.Message, toolDefsForCall []providers.ToolDefinition) (*providers.LLMResponse, error) {
			providerCtx, providerCancel := context.WithCancel(turnCtx)
			ts.setProviderCancel(providerCancel)
			defer func() {
				providerCancel()
				ts.clearProviderCancel(providerCancel)
			}()

			al.activeRequests.Add(1)
			defer al.activeRequests.Done()

			if len(activeCandidates) > 1 && al.fallback != nil {
				fbResult, fbErr := al.fallback.Execute(
					providerCtx,
					activeCandidates,
					func(ctx context.Context, provider, model string) (*providers.LLMResponse, error) {
						return activeProvider.Chat(ctx, messagesForCall, toolDefsForCall, model, llmOpts)
					},
				)
				if fbErr != nil {
					return nil, fbErr
				}
				if fbResult.Provider != "" && len(fbResult.Attempts) > 0 {
					logger.InfoCF(
						"agent",
						fmt.Sprintf("Fallback: succeeded with %s/%s after %d attempts",
							fbResult.Provider, fbResult.Model, len(fbResult.Attempts)+1),
						map[string]any{"agent_id": ts.agent.ID, "iteration": iteration},
					)
				}
				return fbResult.Response, nil
			}
			return activeProvider.Chat(providerCtx, messagesForCall, toolDefsForCall, llmModel, llmOpts)
		}

		var response *providers.LLMResponse
		var err error
		maxRetries := 2
		for retry := 0; retry <= maxRetries; retry++ {
			response, err = callLLM(callMessages, providerToolDefs)
			if err == nil {
				break
			}
			if ts.hardAbortRequested() && errors.Is(err, context.Canceled) {
				turnStatus = TurnEndStatusAborted
				return al.abortTurn(ts)
			}

			errMsg := strings.ToLower(err.Error())
			isTimeoutError := errors.Is(err, context.DeadlineExceeded) ||
				strings.Contains(errMsg, "deadline exceeded") ||
				strings.Contains(errMsg, "client.timeout") ||
				strings.Contains(errMsg, "timed out") ||
				strings.Contains(errMsg, "timeout exceeded")

			isContextError := !isTimeoutError && (strings.Contains(errMsg, "context_length_exceeded") ||
				strings.Contains(errMsg, "context window") ||
				strings.Contains(errMsg, "context_window") ||
				strings.Contains(errMsg, "maximum context length") ||
				strings.Contains(errMsg, "token limit") ||
				strings.Contains(errMsg, "too many tokens") ||
				strings.Contains(errMsg, "max_tokens") ||
				strings.Contains(errMsg, "invalidparameter") ||
				strings.Contains(errMsg, "prompt is too long") ||
				strings.Contains(errMsg, "request too large"))

			if isTimeoutError && retry < maxRetries {
				backoff := time.Duration(retry+1) * 5 * time.Second
				al.emitEvent(
					EventKindLLMRetry,
					ts.eventMeta("runTurn", "turn.llm.retry"),
					LLMRetryPayload{
						Attempt:    retry + 1,
						MaxRetries: maxRetries,
						Reason:     "timeout",
						Error:      err.Error(),
						Backoff:    backoff,
					},
				)
				logger.WarnCF("agent", "Timeout error, retrying after backoff", map[string]any{
					"error":   err.Error(),
					"retry":   retry,
					"backoff": backoff.String(),
				})
				if sleepErr := sleepWithContext(turnCtx, backoff); sleepErr != nil {
					if ts.hardAbortRequested() {
						turnStatus = TurnEndStatusAborted
						return al.abortTurn(ts)
					}
					err = sleepErr
					break
				}
				continue
			}

			if isContextError && retry < maxRetries && !ts.opts.NoHistory {
				al.emitEvent(
					EventKindLLMRetry,
					ts.eventMeta("runTurn", "turn.llm.retry"),
					LLMRetryPayload{
						Attempt:    retry + 1,
						MaxRetries: maxRetries,
						Reason:     "context_limit",
						Error:      err.Error(),
					},
				)
				logger.WarnCF(
					"agent",
					"Context window error detected, attempting compression",
					map[string]any{
						"error": err.Error(),
						"retry": retry,
					},
				)

				if retry == 0 && !constants.IsInternalChannel(ts.channel) {
					al.bus.PublishOutbound(ctx, bus.OutboundMessage{
						Channel: ts.channel,
						ChatID:  ts.chatID,
						Content: "Context window exceeded. Compressing history and retrying...",
					})
				}

				if compression, ok := al.forceCompression(ts.agent, ts.sessionKey); ok {
					al.emitEvent(
						EventKindContextCompress,
						ts.eventMeta("runTurn", "turn.context.compress"),
						ContextCompressPayload{
							Reason:            ContextCompressReasonRetry,
							DroppedMessages:   compression.DroppedMessages,
							RemainingMessages: compression.RemainingMessages,
						},
					)
					ts.refreshRestorePointFromSession(ts.agent)
				}

				newHistory := ts.agent.Sessions.GetHistory(ts.sessionKey)
				newSummary := ts.agent.Sessions.GetSummary(ts.sessionKey)
				messages = ts.agent.ContextBuilder.BuildMessages(
					newHistory, newSummary, "",
					nil, ts.channel, ts.chatID, ts.opts.SenderID, ts.opts.SenderDisplayName,
					activeSkillNames(ts.agent, ts.opts)...,
				)
				callMessages = messages
				if gracefulTerminal {
					callMessages = append(append([]providers.Message(nil), messages...), ts.interruptHintMessage())
				}
				continue
			}
			break
		}

		if err != nil {
			turnStatus = TurnEndStatusError
			al.emitEvent(
				EventKindError,
				ts.eventMeta("runTurn", "turn.error"),
				ErrorPayload{
					Stage:   "llm",
					Message: err.Error(),
				},
			)
			logger.ErrorCF("agent", "LLM call failed",
				map[string]any{
					"agent_id":  ts.agent.ID,
					"iteration": iteration,
					"model":     llmModel,
					"error":     err.Error(),
				})
			return turnResult{}, fmt.Errorf("LLM call failed after retries: %w", err)
		}

		if al.hooks != nil {
			llmResp, decision := al.hooks.AfterLLM(turnCtx, &LLMHookResponse{
				Meta:     ts.eventMeta("runTurn", "turn.llm.response"),
				Model:    llmModel,
				Response: response,
				Channel:  ts.channel,
				ChatID:   ts.chatID,
			})
			switch decision.normalizedAction() {
			case HookActionContinue, HookActionModify:
				if llmResp != nil && llmResp.Response != nil {
					response = llmResp.Response
				}
			case HookActionAbortTurn:
				turnStatus = TurnEndStatusError
				return turnResult{}, al.hookAbortError(ts, "after_llm", decision)
			case HookActionHardAbort:
				_ = ts.requestHardAbort()
				turnStatus = TurnEndStatusAborted
				return al.abortTurn(ts)
			}
		}

		// Save finishReason to turnState for SubTurn truncation detection
		if innerTS := turnStateFromContext(ctx); innerTS != nil {
			innerTS.SetLastFinishReason(response.FinishReason)
			// Save usage for token budget tracking
			if response.Usage != nil {
				innerTS.SetLastUsage(response.Usage)
			}
		}

		reasoningContent := response.Reasoning
		if reasoningContent == "" {
			reasoningContent = response.ReasoningContent
		}
		go al.handleReasoning(
			turnCtx,
			reasoningContent,
			ts.channel,
			al.targetReasoningChannelID(ts.channel),
		)
		al.emitEvent(
			EventKindLLMResponse,
			ts.eventMeta("runTurn", "turn.llm.response"),
			LLMResponsePayload{
				ContentLen:   len(response.Content),
				ToolCalls:    len(response.ToolCalls),
				HasReasoning: response.Reasoning != "" || response.ReasoningContent != "",
			},
		)

		llmResponseFields := map[string]any{
			"agent_id":       ts.agent.ID,
			"iteration":      iteration,
			"content_chars":  len(response.Content),
			"tool_calls":     len(response.ToolCalls),
			"reasoning":      response.Reasoning,
			"target_channel": al.targetReasoningChannelID(ts.channel),
			"channel":        ts.channel,
		}
		if response.Usage != nil {
			llmResponseFields["prompt_tokens"] = response.Usage.PromptTokens
			llmResponseFields["completion_tokens"] = response.Usage.CompletionTokens
			llmResponseFields["total_tokens"] = response.Usage.TotalTokens
		}
		logger.DebugCF("agent", "LLM response", llmResponseFields)

		if len(response.ToolCalls) == 0 || gracefulTerminal {
			responseContent := response.Content
			if responseContent == "" && response.ReasoningContent != "" {
				responseContent = response.ReasoningContent
			}
			if steerMsgs := al.dequeueSteeringMessagesForScope(ts.sessionKey); len(steerMsgs) > 0 {
				logger.InfoCF("agent", "Steering arrived after direct LLM response; continuing turn",
					map[string]any{
						"agent_id":       ts.agent.ID,
						"iteration":      iteration,
						"steering_count": len(steerMsgs),
					})
				pendingMessages = append(pendingMessages, steerMsgs...)
				continue
			}
			finalContent = responseContent
			logger.InfoCF("agent", "LLM response without tool calls (direct answer)",
				map[string]any{
					"agent_id":      ts.agent.ID,
					"iteration":     iteration,
					"content_chars": len(finalContent),
				})
			break
		}

		normalizedToolCalls := make([]providers.ToolCall, 0, len(response.ToolCalls))
		for _, tc := range response.ToolCalls {
			normalizedToolCalls = append(normalizedToolCalls, providers.NormalizeToolCall(tc))
		}

		toolNames := make([]string, 0, len(normalizedToolCalls))
		for _, tc := range normalizedToolCalls {
			toolNames = append(toolNames, tc.Name)
		}
		logger.InfoCF("agent", "LLM requested tool calls",
			map[string]any{
				"agent_id":  ts.agent.ID,
				"tools":     toolNames,
				"count":     len(normalizedToolCalls),
				"iteration": iteration,
			})

		allResponsesHandled := len(normalizedToolCalls) > 0
		assistantMsg := providers.Message{
			Role:             "assistant",
			Content:          response.Content,
			ReasoningContent: response.ReasoningContent,
		}
		for _, tc := range normalizedToolCalls {
			argumentsJSON, _ := json.Marshal(tc.Arguments)
			extraContent := tc.ExtraContent
			thoughtSignature := ""
			if tc.Function != nil {
				thoughtSignature = tc.Function.ThoughtSignature
			}
			assistantMsg.ToolCalls = append(assistantMsg.ToolCalls, providers.ToolCall{
				ID:   tc.ID,
				Type: "function",
				Name: tc.Name,
				Function: &providers.FunctionCall{
					Name:             tc.Name,
					Arguments:        string(argumentsJSON),
					ThoughtSignature: thoughtSignature,
				},
				ExtraContent:     extraContent,
				ThoughtSignature: thoughtSignature,
			})
		}
		messages = append(messages, assistantMsg)
		if !ts.opts.NoHistory {
			ts.agent.Sessions.AddFullMessage(ts.sessionKey, assistantMsg)
			ts.recordPersistedMessage(assistantMsg)
		}

		ts.setPhase(TurnPhaseTools)
		for i, tc := range normalizedToolCalls {
			if ts.hardAbortRequested() {
				turnStatus = TurnEndStatusAborted
				return al.abortTurn(ts)
			}

			toolName := tc.Name
			toolArgs := cloneStringAnyMap(tc.Arguments)

			if al.hooks != nil {
				toolReq, decision := al.hooks.BeforeTool(turnCtx, &ToolCallHookRequest{
					Meta:      ts.eventMeta("runTurn", "turn.tool.before"),
					Tool:      toolName,
					Arguments: toolArgs,
					Channel:   ts.channel,
					ChatID:    ts.chatID,
				})
				switch decision.normalizedAction() {
				case HookActionContinue, HookActionModify:
					if toolReq != nil {
						toolName = toolReq.Tool
						toolArgs = toolReq.Arguments
					}
				case HookActionDenyTool:
					allResponsesHandled = false
					denyContent := hookDeniedToolContent("Tool execution denied by hook", decision.Reason)
					al.emitEvent(
						EventKindToolExecSkipped,
						ts.eventMeta("runTurn", "turn.tool.skipped"),
						ToolExecSkippedPayload{
							Tool:   toolName,
							Reason: denyContent,
						},
					)
					deniedMsg := providers.Message{
						Role:       "tool",
						Content:    denyContent,
						ToolCallID: tc.ID,
					}
					messages = append(messages, deniedMsg)
					if !ts.opts.NoHistory {
						ts.agent.Sessions.AddFullMessage(ts.sessionKey, deniedMsg)
						ts.recordPersistedMessage(deniedMsg)
					}
					continue
				case HookActionAbortTurn:
					turnStatus = TurnEndStatusError
					return turnResult{}, al.hookAbortError(ts, "before_tool", decision)
				case HookActionHardAbort:
					_ = ts.requestHardAbort()
					turnStatus = TurnEndStatusAborted
					return al.abortTurn(ts)
				}
			}

			if al.hooks != nil {
				approval := al.hooks.ApproveTool(turnCtx, &ToolApprovalRequest{
					Meta:      ts.eventMeta("runTurn", "turn.tool.approve"),
					Tool:      toolName,
					Arguments: toolArgs,
					Channel:   ts.channel,
					ChatID:    ts.chatID,
				})
				if !approval.Approved {
					allResponsesHandled = false
					denyContent := hookDeniedToolContent("Tool execution denied by approval hook", approval.Reason)
					al.emitEvent(
						EventKindToolExecSkipped,
						ts.eventMeta("runTurn", "turn.tool.skipped"),
						ToolExecSkippedPayload{
							Tool:   toolName,
							Reason: denyContent,
						},
					)
					deniedMsg := providers.Message{
						Role:       "tool",
						Content:    denyContent,
						ToolCallID: tc.ID,
					}
					messages = append(messages, deniedMsg)
					if !ts.opts.NoHistory {
						ts.agent.Sessions.AddFullMessage(ts.sessionKey, deniedMsg)
						ts.recordPersistedMessage(deniedMsg)
					}
					continue
				}
			}

			argsJSON, _ := json.Marshal(toolArgs)
			argsPreview := utils.Truncate(string(argsJSON), 200)
			logger.InfoCF("agent", fmt.Sprintf("Tool call: %s(%s)", toolName, argsPreview),
				map[string]any{
					"agent_id":  ts.agent.ID,
					"tool":      toolName,
					"iteration": iteration,
				})
			al.emitEvent(
				EventKindToolExecStart,
				ts.eventMeta("runTurn", "turn.tool.start"),
				ToolExecStartPayload{
					Tool:      toolName,
					Arguments: cloneEventArguments(toolArgs),
				},
			)

			// Send tool feedback to chat channel if enabled (from HEAD)
			if al.cfg.Agents.Defaults.IsToolFeedbackEnabled() &&
				ts.channel != "" &&
				!ts.opts.SuppressToolFeedback {
				feedbackPreview := utils.Truncate(
					string(argsJSON),
					al.cfg.Agents.Defaults.GetToolFeedbackMaxArgsLength(),
				)
				feedbackMsg := fmt.Sprintf("\U0001f527 `%s`\n```\n%s\n```", tc.Name, feedbackPreview)
				fbCtx, fbCancel := context.WithTimeout(turnCtx, 3*time.Second)
				_ = al.bus.PublishOutbound(fbCtx, bus.OutboundMessage{
					Channel: ts.channel,
					ChatID:  ts.chatID,
					Content: feedbackMsg,
				})
				fbCancel()
			}

			toolCallID := tc.ID
			toolIteration := iteration
			asyncToolName := toolName
			asyncCallback := func(_ context.Context, result *tools.ToolResult) {
				// Send ForUser content directly to the user (immediate feedback),
				// mirroring the synchronous tool execution path.
				if !result.Silent && result.ForUser != "" {
					outCtx, outCancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer outCancel()
					_ = al.bus.PublishOutbound(outCtx, bus.OutboundMessage{
						Channel: ts.channel,
						ChatID:  ts.chatID,
						Content: result.ForUser,
					})
				}

				// Determine content for the agent loop (ForLLM or error).
				content := result.ContentForLLM()
				if content == "" {
					return
				}

				// Filter sensitive data before publishing
				content = al.cfg.FilterSensitiveData(content)

				logger.InfoCF("agent", "Async tool completed, publishing result",
					map[string]any{
						"tool":        asyncToolName,
						"content_len": len(content),
						"channel":     ts.channel,
					})
				al.emitEvent(
					EventKindFollowUpQueued,
					ts.scope.meta(toolIteration, "runTurn", "turn.follow_up.queued"),
					FollowUpQueuedPayload{
						SourceTool: asyncToolName,
						Channel:    ts.channel,
						ChatID:     ts.chatID,
						ContentLen: len(content),
					},
				)

				pubCtx, pubCancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer pubCancel()
				_ = al.bus.PublishInbound(pubCtx, bus.InboundMessage{
					Channel:  "system",
					SenderID: fmt.Sprintf("async:%s", asyncToolName),
					ChatID:   fmt.Sprintf("%s:%s", ts.channel, ts.chatID),
					Content:  content,
				})
			}

			toolStart := time.Now()
			toolResult := ts.agent.Tools.ExecuteWithContext(
				turnCtx,
				toolName,
				toolArgs,
				ts.channel,
				ts.chatID,
				asyncCallback,
			)
			toolDuration := time.Since(toolStart)

			if ts.hardAbortRequested() {
				turnStatus = TurnEndStatusAborted
				return al.abortTurn(ts)
			}

			if al.hooks != nil {
				toolResp, decision := al.hooks.AfterTool(turnCtx, &ToolResultHookResponse{
					Meta:      ts.eventMeta("runTurn", "turn.tool.after"),
					Tool:      toolName,
					Arguments: toolArgs,
					Result:    toolResult,
					Duration:  toolDuration,
					Channel:   ts.channel,
					ChatID:    ts.chatID,
				})
				switch decision.normalizedAction() {
				case HookActionContinue, HookActionModify:
					if toolResp != nil {
						if toolResp.Tool != "" {
							toolName = toolResp.Tool
						}
						if toolResp.Result != nil {
							toolResult = toolResp.Result
						}
					}
				case HookActionAbortTurn:
					turnStatus = TurnEndStatusError
					return turnResult{}, al.hookAbortError(ts, "after_tool", decision)
				case HookActionHardAbort:
					_ = ts.requestHardAbort()
					turnStatus = TurnEndStatusAborted
					return al.abortTurn(ts)
				}
			}

			if toolResult == nil {
				toolResult = tools.ErrorResult("hook returned nil tool result")
			}
			if len(toolResult.Media) > 0 && toolResult.ResponseHandled {
				parts := make([]bus.MediaPart, 0, len(toolResult.Media))
				for _, ref := range toolResult.Media {
					part := bus.MediaPart{Ref: ref}
					if al.mediaStore != nil {
						if _, meta, err := al.mediaStore.ResolveWithMeta(ref); err == nil {
							part.Filename = meta.Filename
							part.ContentType = meta.ContentType
							part.Type = inferMediaType(meta.Filename, meta.ContentType)
						}
					}
					parts = append(parts, part)
				}
				outboundMedia := bus.OutboundMediaMessage{
					Channel: ts.channel,
					ChatID:  ts.chatID,
					Parts:   parts,
				}
				if al.channelManager != nil && ts.channel != "" && !constants.IsInternalChannel(ts.channel) {
					if err := al.channelManager.SendMedia(ctx, outboundMedia); err != nil {
						logger.WarnCF("agent", "Failed to deliver handled tool media",
							map[string]any{
								"agent_id": ts.agent.ID,
								"tool":     toolName,
								"channel":  ts.channel,
								"chat_id":  ts.chatID,
								"error":    err.Error(),
							})
						toolResult = tools.ErrorResult(fmt.Sprintf("failed to deliver attachment: %v", err)).WithError(err)
					}
				} else if al.bus != nil {
					al.bus.PublishOutboundMedia(ctx, outboundMedia)
					// Queuing media is only best-effort; it has not been delivered yet.
					toolResult.ResponseHandled = false
				}
			}

			if len(toolResult.Media) > 0 && !toolResult.ResponseHandled {
				toolResult.ArtifactTags = buildArtifactTags(al.mediaStore, toolResult.Media)
			}

			if !toolResult.ResponseHandled {
				allResponsesHandled = false
			}

			if !toolResult.Silent && toolResult.ForUser != "" && ts.opts.SendResponse {
				al.bus.PublishOutbound(ctx, bus.OutboundMessage{
					Channel: ts.channel,
					ChatID:  ts.chatID,
					Content: toolResult.ForUser,
				})
				logger.DebugCF("agent", "Sent tool result to user",
					map[string]any{
						"tool":        toolName,
						"content_len": len(toolResult.ForUser),
					})
			}

			contentForLLM := toolResult.ContentForLLM()

			// Filter sensitive data (API keys, tokens, secrets) before sending to LLM
			if al.cfg.Tools.IsFilterSensitiveDataEnabled() {
				contentForLLM = al.cfg.FilterSensitiveData(contentForLLM)
			}

			toolResultMsg := providers.Message{
				Role:       "tool",
				Content:    contentForLLM,
				ToolCallID: toolCallID,
			}
			al.emitEvent(
				EventKindToolExecEnd,
				ts.eventMeta("runTurn", "turn.tool.end"),
				ToolExecEndPayload{
					Tool:       toolName,
					Duration:   toolDuration,
					ForLLMLen:  len(contentForLLM),
					ForUserLen: len(toolResult.ForUser),
					IsError:    toolResult.IsError,
					Async:      toolResult.Async,
				},
			)
			messages = append(messages, toolResultMsg)
			if !ts.opts.NoHistory {
				ts.agent.Sessions.AddFullMessage(ts.sessionKey, toolResultMsg)
				ts.recordPersistedMessage(toolResultMsg)
			}

			if steerMsgs := al.dequeueSteeringMessagesForScope(ts.sessionKey); len(steerMsgs) > 0 {
				pendingMessages = append(pendingMessages, steerMsgs...)
			}

			skipReason := ""
			skipMessage := ""
			if len(pendingMessages) > 0 {
				skipReason = "queued user steering message"
				skipMessage = "Skipped due to queued user message."
			} else if gracefulPending, _ := ts.gracefulInterruptRequested(); gracefulPending {
				skipReason = "graceful interrupt requested"
				skipMessage = "Skipped due to graceful interrupt."
			}

			if skipReason != "" {
				remaining := len(normalizedToolCalls) - i - 1
				if remaining > 0 {
					logger.InfoCF("agent", "Turn checkpoint: skipping remaining tools",
						map[string]any{
							"agent_id":  ts.agent.ID,
							"completed": i + 1,
							"skipped":   remaining,
							"reason":    skipReason,
						})
					for j := i + 1; j < len(normalizedToolCalls); j++ {
						skippedTC := normalizedToolCalls[j]
						al.emitEvent(
							EventKindToolExecSkipped,
							ts.eventMeta("runTurn", "turn.tool.skipped"),
							ToolExecSkippedPayload{
								Tool:   skippedTC.Name,
								Reason: skipReason,
							},
						)
						skippedMsg := providers.Message{
							Role:       "tool",
							Content:    skipMessage,
							ToolCallID: skippedTC.ID,
						}
						messages = append(messages, skippedMsg)
						if !ts.opts.NoHistory {
							ts.agent.Sessions.AddFullMessage(ts.sessionKey, skippedMsg)
							ts.recordPersistedMessage(skippedMsg)
						}
					}
				}
				break
			}

			// Also poll for any SubTurn results that arrived during tool execution.
			if ts.pendingResults != nil {
				select {
				case result, ok := <-ts.pendingResults:
					if ok && result != nil && result.ForLLM != "" {
						content := al.cfg.FilterSensitiveData(result.ForLLM)
						msg := providers.Message{Role: "user", Content: fmt.Sprintf("[SubTurn Result] %s", content)}
						messages = append(messages, msg)
						ts.agent.Sessions.AddFullMessage(ts.sessionKey, msg)
					}
				default:
					// No results available
				}
			}
		}

		if allResponsesHandled {
			if len(pendingMessages) > 0 {
				logger.InfoCF("agent", "Pending steering exists after handled tool delivery; continuing turn before finalizing",
					map[string]any{
						"agent_id":       ts.agent.ID,
						"steering_count": len(pendingMessages),
						"session_key":    ts.sessionKey,
					})
				finalContent = ""
				goto turnLoop
			}

			if steerMsgs := al.dequeueSteeringMessagesForScope(ts.sessionKey); len(steerMsgs) > 0 {
				logger.InfoCF("agent", "Steering arrived after handled tool delivery; continuing turn before finalizing",
					map[string]any{
						"agent_id":       ts.agent.ID,
						"steering_count": len(steerMsgs),
						"session_key":    ts.sessionKey,
					})
				pendingMessages = append(pendingMessages, steerMsgs...)
				finalContent = ""
				goto turnLoop
			}

			summaryMsg := providers.Message{
				Role:    "assistant",
				Content: handledToolResponseSummary,
			}

			if !ts.opts.NoHistory {
				ts.agent.Sessions.AddMessage(ts.sessionKey, summaryMsg.Role, summaryMsg.Content)
				ts.recordPersistedMessage(summaryMsg)
				if err := ts.agent.Sessions.Save(ts.sessionKey); err != nil {
					turnStatus = TurnEndStatusError
					al.emitEvent(
						EventKindError,
						ts.eventMeta("runTurn", "turn.error"),
						ErrorPayload{
							Stage:   "session_save",
							Message: err.Error(),
						},
					)
					return turnResult{}, err
				}
			}
			if ts.opts.EnableSummary {
				al.maybeSummarize(ts.agent, ts.sessionKey, ts.scope)
			}

			ts.setPhase(TurnPhaseCompleted)
			ts.setFinalContent("")
			logger.InfoCF("agent", "Tool output satisfied delivery; ending turn without follow-up LLM",
				map[string]any{
					"agent_id":   ts.agent.ID,
					"iteration":  iteration,
					"tool_count": len(normalizedToolCalls),
				})
			return turnResult{
				finalContent: "",
				status:       turnStatus,
				followUps:    append([]bus.InboundMessage(nil), ts.followUps...),
			}, nil
		}

		ts.agent.Tools.TickTTL()
		logger.DebugCF("agent", "TTL tick after tool execution", map[string]any{
			"agent_id": ts.agent.ID, "iteration": iteration,
		})
	}

	if steerMsgs := al.dequeueSteeringMessagesForScope(ts.sessionKey); len(steerMsgs) > 0 {
		logger.InfoCF("agent", "Steering arrived after turn completion; continuing turn before finalizing",
			map[string]any{
				"agent_id":       ts.agent.ID,
				"steering_count": len(steerMsgs),
				"session_key":    ts.sessionKey,
			})
		pendingMessages = append(pendingMessages, steerMsgs...)
		finalContent = ""
		goto turnLoop
	}

	if ts.hardAbortRequested() {
		turnStatus = TurnEndStatusAborted
		return al.abortTurn(ts)
	}

	if finalContent == "" {
		if ts.currentIteration() >= ts.agent.MaxIterations && ts.agent.MaxIterations > 0 {
			finalContent = toolLimitResponse
		} else {
			finalContent = ts.opts.DefaultResponse
		}
	}

	ts.setPhase(TurnPhaseFinalizing)
	ts.setFinalContent(finalContent)
	if !ts.opts.NoHistory {
		finalMsg := providers.Message{Role: "assistant", Content: finalContent}
		ts.agent.Sessions.AddMessage(ts.sessionKey, finalMsg.Role, finalMsg.Content)
		ts.recordPersistedMessage(finalMsg)
		if err := ts.agent.Sessions.Save(ts.sessionKey); err != nil {
			turnStatus = TurnEndStatusError
			al.emitEvent(
				EventKindError,
				ts.eventMeta("runTurn", "turn.error"),
				ErrorPayload{
					Stage:   "session_save",
					Message: err.Error(),
				},
			)
			return turnResult{}, err
		}
	}

	if ts.opts.EnableSummary {
		al.maybeSummarize(ts.agent, ts.sessionKey, ts.scope)
	}

	ts.setPhase(TurnPhaseCompleted)
	return turnResult{
		finalContent: finalContent,
		status:       turnStatus,
		followUps:    append([]bus.InboundMessage(nil), ts.followUps...),
	}, nil
}

// abortTurn 中止当前 Turn
//
// 当收到硬中断(HardAbort)时调用,执行以下操作:
//   1. 将 Turn 状态设置为 TurnPhaseAborted
//   2. 恢复会话到快照(Turn 开始前的状态)
//      - 这确保了被中断的 Turn 的所有中间状态(工具调用等)都被丢弃
//      - 会话历史回滚到 Turn 开始前,好像这次对话从未发生
//
// 注意: abortTurn 不会发送任何消息给用户,它只清理内部状态。
func (al *AgentLoop) abortTurn(ts *turnState) (turnResult, error) {
	ts.setPhase(TurnPhaseAborted)
	if !ts.opts.NoHistory {
		if err := ts.restoreSession(ts.agent); err != nil {
			al.emitEvent(
				EventKindError,
				ts.eventMeta("abortTurn", "turn.error"),
				ErrorPayload{
					Stage:   "session_restore",
					Message: err.Error(),
				},
			)
			return turnResult{}, err
		}
	}
	return turnResult{status: TurnEndStatusAborted}, nil
}

// sleepWithContext 带上下文感知的 sleep
//
// 与 time.Sleep 不同,此函数在 ctx 被取消时会立即返回,
// 避免在 shutdown 期间不必要的等待。
//
// 返回:
//   - nil: sleep 正常完成
//   - ctx.Err(): ctx 被取消
func sleepWithContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// selectCandidates 根据消息复杂度选择合适的模型候选集和 resolved 模型名使用
//
// 当配置了模型路由且消息复杂度低于阈值时,返回轻量模型候选集。
// 否则返回主模型候选集。
//
// 一旦选择后同一 turn 内的所有 LLM 调用都使用同一组候选集,
// 以便多步骤工具链不会中途切换模型。
//
// 参数:
//   - agent: Agent 实例(包含候选集列表和路由器)
//   - userMsg: 用户消息(用于复杂度评估)
//   - history: 对话历史(用于复杂度评估)
//
// 返回:
//   - candidates: LLM Provider 候选集(每个包含 provider+model)
//   - model: 解析后的模型名
//   - usedLight: 是否使用了轻量模型
func (al *AgentLoop) selectCandidates(
	agent *AgentInstance,
	userMsg string,
	history []providers.Message,
) (candidates []providers.FallbackCandidate, model string, usedLight bool) {
	if agent.Router == nil || len(agent.LightCandidates) == 0 {
		return agent.Candidates, resolvedCandidateModel(agent.Candidates, agent.Model), false
	}

	_, usedLight, score := agent.Router.SelectModel(userMsg, history, agent.Model)
	if !usedLight {
		logger.DebugCF("agent", "Model routing: primary model selected",
			map[string]any{
				"agent_id":  agent.ID,
				"score":     score,
				"threshold": agent.Router.Threshold(),
			})
		return agent.Candidates, resolvedCandidateModel(agent.Candidates, agent.Model), false
	}

	logger.InfoCF("agent", "Model routing: light model selected",
		map[string]any{
			"agent_id":    agent.ID,
			"light_model": agent.Router.LightModel(),
			"score":       score,
			"threshold":   agent.Router.Threshold(),
		})
	return agent.LightCandidates, resolvedCandidateModel(agent.LightCandidates, agent.Router.LightModel()), true
}

// maybeSummarize 检查是否需要触发会话摘要(如果历史超过阈值)
//
// 触发条件(满足任一即可触发):
//   - 消息数量阈值: SummarizeMessageThreshold
//   - Token 估算阈值: ContextWindow * SummarizeTokenPercent / 100
//
// 摘要使用 goroutine 异步执行,通过 sync.Map 防止同一会话并发摘要
//
// 参数:
//   - agent: Agent 实例(包含会话和配置)
//   - sessionKey: 会话标识符
//   - turnScope: Turn 事件范围(用于事件发射)
func (al *AgentLoop) maybeSummarize(agent *AgentInstance, sessionKey string, turnScope turnEventScope) {
	newHistory := agent.Sessions.GetHistory(sessionKey)
	tokenEstimate := al.estimateTokens(newHistory)
	threshold := agent.ContextWindow * agent.SummarizeTokenPercent / 100

	if len(newHistory) > agent.SummarizeMessageThreshold || tokenEstimate > threshold {
		summarizeKey := agent.ID + ":" + sessionKey
		if _, loading := al.summarizing.LoadOrStore(summarizeKey, true); !loading {
			go func() {
				defer al.summarizing.Delete(summarizeKey)
				logger.Debug("Memory threshold reached. Optimizing conversation history...")
				al.summarizeSession(agent, sessionKey, turnScope)
			}()
		}
	}
}

// compressionResult 记录压缩操作的结果
type compressionResult struct {
	DroppedMessages   int
	RemainingMessages int
}

// forceCompression 濱遇上下文长度超限时强制压缩历史
//
// 压缩策略:
//   1. 按完整 Turn 边界分割(Turn = 用户→LLM→响应 的完整循环)
//   2. 丢弃最旧的 ~50% Turn,保留最新的 50%
//   3. 如果只有单个 Turn(无安全分割点),则仅保留最近一条用户消息(兜底方案)
//
// 注意: 此操作会修改会话历史和不会修改消息内容。
// 压缩信息会记录在会话摘要中,以便下次 BuildMessages 包其包含在系统提示中
//
// 参数:
//   - agent: Agent 实例
//   - sessionKey: 会话标识符
//
// 返回:
//   - compressionResult: 压缩结果(丢弃/保留消息数)
//   - bool: 是否执行了压缩
func (al *AgentLoop) forceCompression(agent *AgentInstance, sessionKey string) (compressionResult, bool) {
	history := agent.Sessions.GetHistory(sessionKey)
	if len(history) <= 2 {
		return compressionResult{}, false
	}

	// Split at a Turn boundary so no tool-call sequence is torn apart.
	// parseTurnBoundaries gives us the start of each Turn; we drop the
	// oldest half of Turns and keep the most recent ones.
	turns := parseTurnBoundaries(history)
	var mid int
	if len(turns) >= 2 {
		mid = turns[len(turns)/2]
	} else {
		// Fewer than 2 Turns — fall back to message-level midpoint
		// aligned to the nearest Turn boundary.
		mid = findSafeBoundary(history, len(history)/2)
	}
	var keptHistory []providers.Message
	if mid <= 0 {
		// No safe Turn boundary — the entire history is a single Turn
		// (e.g. one user message followed by a massive tool response).
		// Keeping everything would leave the agent stuck in a context-
		// exceeded loop, so fall back to keeping only the most recent
		// user message. This breaks Turn atomicity as a last resort.
		for i := len(history) - 1; i >= 0; i-- {
			if history[i].Role == "user" {
				keptHistory = []providers.Message{history[i]}
				break
			}
		}
	} else {
		keptHistory = history[mid:]
	}

	droppedCount := len(history) - len(keptHistory)

	// Record compression in the session summary so BuildMessages includes it
	// in the system prompt. We do not modify history messages themselves.
	existingSummary := agent.Sessions.GetSummary(sessionKey)
	compressionNote := fmt.Sprintf(
		"[Emergency compression dropped %d oldest messages due to context limit]",
		droppedCount,
	)
	if existingSummary != "" {
		compressionNote = existingSummary + "\n\n" + compressionNote
	}
	agent.Sessions.SetSummary(sessionKey, compressionNote)

	agent.Sessions.SetHistory(sessionKey, keptHistory)
	agent.Sessions.Save(sessionKey)

	logger.WarnCF("agent", "Forced compression executed", map[string]any{
		"session_key":  sessionKey,
		"dropped_msgs": droppedCount,
		"new_count":    len(keptHistory),
	})

	return compressionResult{
		DroppedMessages:   droppedCount,
		RemainingMessages: len(keptHistory),
	}, true
}

// GetStartupInfo 返回已加载的工具和技能信息
//
// 用于启动时的诊断日志,显示:
//   - 已注册的工具数量和名称列表
//   - 已安装的技能信息
//   - Agent 数量和 ID 列表
func (al *AgentLoop) GetStartupInfo() map[string]any {
	info := make(map[string]any)

	registry := al.GetRegistry()
	agent := registry.GetDefaultAgent()
	if agent == nil {
		return info
	}

	// Tools info
	toolsList := agent.Tools.List()
	info["tools"] = map[string]any{
		"count": len(toolsList),
		"names": toolsList,
	}

	// Skills info
	info["skills"] = agent.ContextBuilder.GetSkillsInfo()

	// Agents info
	info["agents"] = map[string]any{
		"count": len(registry.ListAgentIDs()),
		"ids":   registry.ListAgentIDs(),
	}

	return info
}

// formatMessagesForLog 将消息列表格式化为日志友好的字符串
//
// 输出格式:
//
//	[
//	  [0] Role: user
//	    Content: Hello...
//	  [1] Role: assistant
//	    Content: Hi there...
//	    ToolCalls:
//	      - ID: call_123, Type: function, Name: web_search
//	        Arguments: {"query": "weather"}
//	]
func formatMessagesForLog(messages []providers.Message) string {
	if len(messages) == 0 {
		return "[]"
	}

	var sb strings.Builder
	sb.WriteString("[\n")
	for i, msg := range messages {
		fmt.Fprintf(&sb, "  [%d] Role: %s\n", i, msg.Role)
		if len(msg.ToolCalls) > 0 {
			sb.WriteString("  ToolCalls:\n")
			for _, tc := range msg.ToolCalls {
				fmt.Fprintf(&sb, "    - ID: %s, Type: %s, Name: %s\n", tc.ID, tc.Type, tc.Name)
				if tc.Function != nil {
					fmt.Fprintf(
						&sb,
						"      Arguments: %s\n",
						utils.Truncate(tc.Function.Arguments, 200),
					)
				}
			}
		}
		if msg.Content != "" {
			content := utils.Truncate(msg.Content, 200)
			fmt.Fprintf(&sb, "  Content: %s\n", content)
		}
		if msg.ToolCallID != "" {
			fmt.Fprintf(&sb, "  ToolCallID: %s\n", msg.ToolCallID)
		}
		sb.WriteString("\n")
	}
	sb.WriteString("]")
	return sb.String()
}

// formatToolsForLog 将工具定义列表格式化为日志友好的字符串
func formatToolsForLog(toolDefs []providers.ToolDefinition) string {
	if len(toolDefs) == 0 {
		return "[]"
	}

	var sb strings.Builder
	sb.WriteString("[\n")
	for i, tool := range toolDefs {
		fmt.Fprintf(&sb, "  [%d] Type: %s, Name: %s\n", i, tool.Type, tool.Function.Name)
		fmt.Fprintf(&sb, "      Description: %s\n", tool.Function.Description)
		if len(tool.Function.Parameters) > 0 {
			fmt.Fprintf(
				&sb,
				"      Parameters: %s\n",
				utils.Truncate(fmt.Sprintf("%v", tool.Function.Parameters), 200),
			)
		}
	}
	sb.WriteString("]")
	return sb.String()
}

// summarizeSession summarizes the conversation history for a session.
// summarizeSession 对会话历史进行摘要压缩
//
// 使用 LLM 将旧的历史消息总结为简短摘要,以减少上下文 token 占用。
// 仅保留最近的几轮对话以保证连续性。
//
// 夑壕流程:
//   1. 检查历史是否够长(<=44条直接返回)
//   2. 找到安全的 Turn 边界切割点
//   3. 过滤超大消息(超过上下文窗口一半的消息)
//   4. 如果消息多于阈值,分批摘要后合并
//   5. 将摘要保存到会话并截断历史
//
// 参数:
//   - agent: Agent 实例
//   - sessionKey: 会话标识符
//   - turnScope: Turn 事件范围(用于事件发射)
func (al *AgentLoop) summarizeSession(agent *AgentInstance, sessionKey string, turnScope turnEventScope) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	history := agent.Sessions.GetHistory(sessionKey)
	summary := agent.Sessions.GetSummary(sessionKey)

	// Keep the most recent Turns for continuity, aligned to a Turn boundary
	// so that no tool-call sequence is split.
	if len(history) <= 4 {
		return
	}

	safeCut := findSafeBoundary(history, len(history)-4)
	if safeCut <= 0 {
		return
	}
	keepCount := len(history) - safeCut
	toSummarize := history[:safeCut]

	// Oversized Message Guard
	maxMessageTokens := agent.ContextWindow / 2
	validMessages := make([]providers.Message, 0)
	omitted := false

	for _, m := range toSummarize {
		if m.Role != "user" && m.Role != "assistant" {
			continue
		}
		msgTokens := len(m.Content) / 2
		if msgTokens > maxMessageTokens {
			omitted = true
			continue
		}
		validMessages = append(validMessages, m)
	}

	if len(validMessages) == 0 {
		return
	}

	const (
		maxSummarizationMessages = 10
		llmMaxRetries            = 3
		llmTemperature           = 0.3
		fallbackMaxContentLength = 200
	)

	// Multi-Part Summarization
	var finalSummary string
	if len(validMessages) > maxSummarizationMessages {
		mid := len(validMessages) / 2

		mid = al.findNearestUserMessage(validMessages, mid)

		part1 := validMessages[:mid]
		part2 := validMessages[mid:]

		s1, _ := al.summarizeBatch(ctx, agent, part1, "")
		s2, _ := al.summarizeBatch(ctx, agent, part2, "")

		mergePrompt := fmt.Sprintf(
			"Merge these two conversation summaries into one cohesive summary:\n\n1: %s\n\n2: %s",
			s1,
			s2,
		)

		resp, err := al.retryLLMCall(ctx, agent, mergePrompt, llmMaxRetries)
		if err == nil && resp.Content != "" {
			finalSummary = resp.Content
		} else {
			finalSummary = s1 + " " + s2
		}
	} else {
		finalSummary, _ = al.summarizeBatch(ctx, agent, validMessages, summary)
	}

	if omitted && finalSummary != "" {
		finalSummary += "\n[Note: Some oversized messages were omitted from this summary for efficiency.]"
	}

	if finalSummary != "" {
		agent.Sessions.SetSummary(sessionKey, finalSummary)
		agent.Sessions.TruncateHistory(sessionKey, keepCount)
		agent.Sessions.Save(sessionKey)
		al.emitEvent(
			EventKindSessionSummarize,
			turnScope.meta(0, "summarizeSession", "turn.session.summarize"),
			SessionSummarizePayload{
				SummarizedMessages: len(validMessages),
				KeptMessages:       keepCount,
				SummaryLen:         len(finalSummary),
				OmittedOversized:   omitted,
			},
		)
	}
}

// findNearestUserMessage 在消息列表中查找最近的用户消息
//
// 从 mid 位置开始,先向后搜索,如果没找到则向前搜索。
// 这样做是为了在分割消息时确保分割点在用户消息边界上,
// 避免将用户消息和对应的 LLM 响应拆开。
func (al *AgentLoop) findNearestUserMessage(messages []providers.Message, mid int) int {
	originalMid := mid

	for mid > 0 && messages[mid].Role != "user" {
		mid--
	}

	if messages[mid].Role == "user" {
		return mid
	}

	mid = originalMid
	for mid < len(messages) && messages[mid].Role != "user" {
		mid++
	}

	if mid < len(messages) {
		return mid
	}

	return originalMid
}

// retryLLMCall 带重试逻辑的 LLM 调用
//
// 在 LLM 调用失败或返回空内容时自动重试,最多重试 maxRetries 次。
// 重试间隔使用递增延迟(100ms * attempt)。
//
// 参数:
//   - ctx: 上下文
//   - agent: Agent 实例(包含 Provider 和 Model)
//   - prompt: 提示词文本
//   - maxRetries: 最大重试次数
//
// 返回: LLM 响应(可能为 nil)和错误
func (al *AgentLoop) retryLLMCall(
	ctx context.Context,
	agent *AgentInstance,
	prompt string,
	maxRetries int,
) (*providers.LLMResponse, error) {
	const (
		llmTemperature = 0.3
	)

	var resp *providers.LLMResponse
	var err error

	for attempt := 0; attempt < maxRetries; attempt++ {
		al.activeRequests.Add(1)
		resp, err = func() (*providers.LLMResponse, error) {
			defer al.activeRequests.Done()
			return agent.Provider.Chat(
				ctx,
				[]providers.Message{{Role: "user", Content: prompt}},
				nil,
				agent.Model,
				map[string]any{
					"max_tokens":       agent.MaxTokens,
					"temperature":      llmTemperature,
					"prompt_cache_key": agent.ID,
				},
			)
		}()

		if err == nil && resp != nil && resp.Content != "" {
			return resp, nil
		}
		if attempt < maxRetries-1 {
			time.Sleep(time.Duration(attempt+1) * 100 * time.Millisecond)
		}
	}

	return resp, err
}

// summarizeBatch 对一批消息进行摘要
//
// 使用 LLM 将一批对话消息总结为简短摘要。
// 如果 LLM 调用失败,回退到简单截断策略:
//   - 每条消息保留 fallbackMaxContentPercent% 的内容
//   - 最少保留 fallbackMinContentLength 个字符
//
// 参数:
//   - ctx: 上下文
//   - agent: Agent 实例
//   - batch: 待摘要的消息列表
//   - existingSummary: 已有的摘要(用于保持连续性)
//
// 返回: 摘要文本和错误
func (al *AgentLoop) summarizeBatch(
	ctx context.Context,
	agent *AgentInstance,
	batch []providers.Message,
	existingSummary string,
) (string, error) {
	const (
		llmMaxRetries             = 3
		llmTemperature            = 0.3
		fallbackMinContentLength  = 200
		fallbackMaxContentPercent = 10
	)

	var sb strings.Builder
	sb.WriteString(
		"Provide a concise summary of this conversation segment, preserving core context and key points.\n",
	)
	if existingSummary != "" {
		sb.WriteString("Existing context: ")
		sb.WriteString(existingSummary)
		sb.WriteString("\n")
	}
	sb.WriteString("\nCONVERSATION:\n")
	for _, m := range batch {
		fmt.Fprintf(&sb, "%s: %s\n", m.Role, m.Content)
	}
	prompt := sb.String()

	response, err := al.retryLLMCall(ctx, agent, prompt, llmMaxRetries)
	if err == nil && response.Content != "" {
		return strings.TrimSpace(response.Content), nil
	}

	var fallback strings.Builder
	fallback.WriteString("Conversation summary: ")
	for i, m := range batch {
		if i > 0 {
			fallback.WriteString(" | ")
		}
		content := strings.TrimSpace(m.Content)
		runes := []rune(content)
		if len(runes) == 0 {
			fallback.WriteString(fmt.Sprintf("%s: ", m.Role))
			continue
		}

		keepLength := len(runes) * fallbackMaxContentPercent / 100
		if keepLength < fallbackMinContentLength {
			keepLength = fallbackMinContentLength
		}

		if keepLength > len(runes) {
			keepLength = len(runes)
		}

		content = string(runes[:keepLength])
		if keepLength < len(runes) {
			content += "..."
		}
		fallback.WriteString(fmt.Sprintf("%s: %s", m.Role, content))
	}
	return fallback.String(), nil
}

// estimateTokens 估算消息列表的 token 数量
//
// 使用简单的字符数/2 估算(1个 token ≈ 2 个字符),并统计:
//   - Content: 消息正文
//   - ToolCalls: 巯具调用参数
//   - ToolCallID: 巯具调用 ID 元数据
//
// 这样工具密集型对话不会被系统性低估。
func (al *AgentLoop) estimateTokens(messages []providers.Message) int {
	total := 0
	for _, m := range messages {
		total += estimateMessageTokens(m)
	}
	return total
}

// handleCommand 处理斜杠命令
//
// 支持的命令类型:
//   - 显式技能命令(/use <skill> [message])
//   - 内置命令(/help, /clear, /model, /reload 等)
//
// 返回:
//   - response: 命令的响应文本
//   - handled: 是否已被命令处理器处理(如果为 true,不再传递给 LLM)
func (al *AgentLoop) handleCommand(
	ctx context.Context,
	msg bus.InboundMessage,
	agent *AgentInstance,
	opts *processOptions,
) (string, bool) {
	if !commands.HasCommandPrefix(msg.Content) {
		return "", false
	}

	if matched, handled, reply := al.applyExplicitSkillCommand(msg.Content, agent, opts); matched {
		return reply, handled
	}

	if al.cmdRegistry == nil {
		return "", false
	}

	rt := al.buildCommandsRuntime(agent, opts)
	executor := commands.NewExecutor(al.cmdRegistry, rt)

	var commandReply string
	result := executor.Execute(ctx, commands.Request{
		Channel:  msg.Channel,
		ChatID:   msg.ChatID,
		SenderID: msg.SenderID,
		Text:     msg.Content,
		Reply: func(text string) error {
			commandReply = text
			return nil
		},
	})

	switch result.Outcome {
	case commands.OutcomeHandled:
		if result.Err != nil {
			return mapCommandError(result), true
		}
		if commandReply != "" {
			return commandReply, true
		}
		return "", true
	default: // OutcomePassthrough — let the message fall through to LLM
		return "", false
	}
}

// activeSkillNames 收集所有应激活的技能名称
//
// 合并 Agent 配置的默认技能 + 消息选项中强制指定的技能。
// 通过 ContextBuilder.ResolveSkillName 进行规范化(支持别名)。
// 结果去重并保持顺序。
func activeSkillNames(agent *AgentInstance, opts processOptions) []string {
	if agent == nil {
		return nil
	}

	combined := make([]string, 0, len(agent.SkillsFilter)+len(opts.ForcedSkills))
	combined = append(combined, agent.SkillsFilter...)
	combined = append(combined, opts.ForcedSkills...)
	if len(combined) == 0 {
		return nil
	}

	var resolved []string
	seen := make(map[string]struct{}, len(combined))
	for _, name := range combined {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if agent.ContextBuilder != nil {
			if canonical, ok := agent.ContextBuilder.ResolveSkillName(name); ok {
				name = canonical
			}
		}
		key := strings.ToLower(name)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		resolved = append(resolved, name)
	}

	return resolved
}

// applyExplicitSkillCommand 处理 /use 技能命令
//
// 支持三种用法:
//   - /use <skill>       → 将技能加入待应用列表,下一条消息自动激活
//   - /use <skill> <msg> → 立即使用指定技能处理消息
//   - /use clear         → 清除待应用的技能覆盖
//
// 返回:
//   - matched: 是否匹配了 /use 命令
//   - handled: 是否已完全处理(不需要传给 LLM)
//   - reply: 处理结果文本
func (al *AgentLoop) applyExplicitSkillCommand(
	raw string,
	agent *AgentInstance,
	opts *processOptions,
) (matched bool, handled bool, reply string) {
	cmdName, ok := commands.CommandName(raw)
	if !ok || cmdName != "use" {
		return false, false, ""
	}

	if agent == nil || agent.ContextBuilder == nil {
		return true, true, commandsUnavailableSkillMessage()
	}

	parts := strings.Fields(strings.TrimSpace(raw))
	if len(parts) < 2 {
		return true, true, buildUseCommandHelp(agent)
	}

	arg := strings.TrimSpace(parts[1])
	if strings.EqualFold(arg, "clear") || strings.EqualFold(arg, "off") {
		if opts != nil {
			al.clearPendingSkills(opts.SessionKey)
		}
		return true, true, "Cleared pending skill override."
	}

	skillName, ok := agent.ContextBuilder.ResolveSkillName(arg)
	if !ok {
		return true, true, fmt.Sprintf("Unknown skill: %s\nUse /list skills to see installed skills.", arg)
	}

	if len(parts) < 3 {
		if opts == nil || strings.TrimSpace(opts.SessionKey) == "" {
			return true, true, commandsUnavailableSkillMessage()
		}
		al.setPendingSkills(opts.SessionKey, []string{skillName})
		return true, true, fmt.Sprintf(
			"Skill %q is armed for your next message. Send your next prompt normally, or use /use clear to cancel.",
			skillName,
		)
	}

	message := strings.TrimSpace(strings.Join(parts[2:], " "))
	if message == "" {
		return true, true, buildUseCommandHelp(agent)
	}

	if opts != nil {
		opts.ForcedSkills = append(opts.ForcedSkills, skillName)
		opts.UserMessage = message
	}

	return true, false, ""
}

// buildCommandsRuntime 构建命令执行运行时
//
// 运行时提供了命令执行所需的回调函数:
//   - ListAgentIDs: 列出所有 Agent ID
//   - ListDefinitions: 列出所有命令定义
//   - GetEnabledChannels: 获取已启用的通道列表
//   - GetActiveTurn: 获取当前活跃的 Turn 状态
//   - SwitchChannel: 切换通道
//   - ListSkillNames: 列出已安装的技能
//   - ReloadConfig: 热重载配置
//   - GetModelInfo: 获取当前模型信息
//   - SwitchModel: 切换模型
//   - ClearHistory: 清空会话历史
func (al *AgentLoop) buildCommandsRuntime(agent *AgentInstance, opts *processOptions) *commands.Runtime {
	registry := al.GetRegistry()
	cfg := al.GetConfig()
	rt := &commands.Runtime{
		Config:          cfg,
		ListAgentIDs:    registry.ListAgentIDs,
		ListDefinitions: al.cmdRegistry.Definitions,
		GetEnabledChannels: func() []string {
			if al.channelManager == nil {
				return nil
			}
			return al.channelManager.GetEnabledChannels()
		},
		GetActiveTurn: func() any {
			info := al.GetActiveTurn()
			if info == nil {
				return nil
			}
			return info
		},
		SwitchChannel: func(value string) error {
			if al.channelManager == nil {
				return fmt.Errorf("channel manager not initialized")
			}
			if _, exists := al.channelManager.GetChannel(value); !exists && value != "cli" {
				return fmt.Errorf("channel '%s' not found or not enabled", value)
			}
			return nil
		},
	}
	if agent != nil && agent.ContextBuilder != nil {
		rt.ListSkillNames = agent.ContextBuilder.ListSkillNames
	}
	rt.ReloadConfig = func() error {
		if al.reloadFunc == nil {
			return fmt.Errorf("reload not configured")
		}
		return al.reloadFunc()
	}
	if agent != nil {
		if agent.ContextBuilder != nil {
			rt.ListSkillNames = agent.ContextBuilder.ListSkillNames
		}
		rt.GetModelInfo = func() (string, string) {
			return agent.Model, resolvedCandidateProvider(agent.Candidates, cfg.Agents.Defaults.Provider)
		}
		rt.SwitchModel = func(value string) (string, error) {
			value = strings.TrimSpace(value)
			modelCfg, err := resolvedModelConfig(cfg, value, agent.Workspace)
			if err != nil {
				return "", err
			}

			nextProvider, _, err := providers.CreateProviderFromConfig(modelCfg)
			if err != nil {
				return "", fmt.Errorf("failed to initialize model %q: %w", value, err)
			}

			nextCandidates := resolveModelCandidates(cfg, cfg.Agents.Defaults.Provider, modelCfg.Model, agent.Fallbacks)
			if len(nextCandidates) == 0 {
				return "", fmt.Errorf("model %q did not resolve to any provider candidates", value)
			}

			oldModel := agent.Model
			oldProvider := agent.Provider
			agent.Model = value
			agent.Provider = nextProvider
			agent.Candidates = nextCandidates
			agent.ThinkingLevel = parseThinkingLevel(modelCfg.ThinkingLevel)

			if oldProvider != nil && oldProvider != nextProvider {
				if stateful, ok := oldProvider.(providers.StatefulProvider); ok {
					stateful.Close()
				}
			}
			return oldModel, nil
		}

		rt.ClearHistory = func() error {
			if opts == nil {
				return fmt.Errorf("process options not available")
			}
			if agent.Sessions == nil {
				return fmt.Errorf("sessions not initialized for agent")
			}

			agent.Sessions.SetHistory(opts.SessionKey, make([]providers.Message, 0))
			agent.Sessions.SetSummary(opts.SessionKey, "")
			agent.Sessions.Save(opts.SessionKey)
			return nil
		}
	}
	return rt
}

// commandsUnavailableSkillMessage 返回技能不可用提示
func commandsUnavailableSkillMessage() string {
	return "Skill selection is unavailable in the current context."
}

// buildUseCommandHelp 构建 /use 命令的帮助信息
//
// 显示用法说明和已安装技能列表。
func buildUseCommandHelp(agent *AgentInstance) string {
	if agent == nil || agent.ContextBuilder == nil {
		return "Usage: /use <skill> [message]"
	}

	names := agent.ContextBuilder.ListSkillNames()
	if len(names) == 0 {
		return "Usage: /use <skill> [message]\nNo installed skills found."
	}

	return fmt.Sprintf(
		"Usage: /use <skill> [message]\n\nInstalled Skills:\n- %s\n\nUse /use <skill> to apply a skill to your next message, or /use <skill> <message> to force it immediately.",
		strings.Join(names, "\n- "),
	)
}

// setPendingSkills 为指定会话设置待应用的技能列表
//
// 在收到 /use <skill> 命令时,技能不会立即应用,
// 而是存入 pendingSkills,在用户发送下一条消息时自动激活。
func (al *AgentLoop) setPendingSkills(sessionKey string, skillNames []string) {
	sessionKey = strings.TrimSpace(sessionKey)
	if sessionKey == "" || len(skillNames) == 0 {
		return
	}

	filtered := make([]string, 0, len(skillNames))
	for _, name := range skillNames {
		name = strings.TrimSpace(name)
		if name != "" {
			filtered = append(filtered, name)
		}
	}
	if len(filtered) == 0 {
		return
	}

	al.pendingSkills.Store(sessionKey, filtered)
}

// takePendingSkills 取出并清除指定会话的待应用技能列表
//
// 原子操作:读取后立即删除,确保技能只被应用一次。
// 返回技能名称的副本(避免外部修改内部数据)。
func (al *AgentLoop) takePendingSkills(sessionKey string) []string {
	sessionKey = strings.TrimSpace(sessionKey)
	if sessionKey == "" {
		return nil
	}

	value, ok := al.pendingSkills.LoadAndDelete(sessionKey)
	if !ok {
		return nil
	}

	skills, ok := value.([]string)
	if !ok {
		return nil
	}

	return append([]string(nil), skills...)
}

// clearPendingSkills 清除指定会话的待应用技能列表
func (al *AgentLoop) clearPendingSkills(sessionKey string) {
	sessionKey = strings.TrimSpace(sessionKey)
	if sessionKey == "" {
		return
	}
	al.pendingSkills.Delete(sessionKey)
}

// mapCommandError 将命令执行错误格式化为用户友好的错误消息
func mapCommandError(result commands.ExecuteResult) string {
	if result.Command == "" {
		return fmt.Sprintf("Failed to execute command: %v", result.Err)
	}
	return fmt.Sprintf("Failed to execute /%s: %v", result.Command, result.Err)
}

// extractPeer 从入站消息的结构化 Peer 字段提取路由对等方
//
// Peer 描述消息的来源类型和来源 ID:
//   - kind: "direct"(私聊), "group"(群聊), "channel"(频道)
//   - id: 对等方唯一标识(如果为空,使用 SenderID 或 ChatID 作为回退)
func extractPeer(msg bus.InboundMessage) *routing.RoutePeer {
	if msg.Peer.Kind == "" {
		return nil
	}
	peerID := msg.Peer.ID
	if peerID == "" {
		if msg.Peer.Kind == "direct" {
			peerID = msg.SenderID
		} else {
			peerID = msg.ChatID
		}
	}
	return &routing.RoutePeer{Kind: msg.Peer.Kind, ID: peerID}
}

// inboundMetadata 从入站消息的元数据中获取指定键的值
func inboundMetadata(msg bus.InboundMessage, key string) string {
	if msg.Metadata == nil {
		return ""
	}
	return msg.Metadata[key]
}

// extractParentPeer 从入站消息元数据中提取父级对等方(回复目标)
//
// 用于回复场景:当用户回复某条消息时,metadata 中包含父级对等方信息,
// 用于路由到正确的 Agent。
func extractParentPeer(msg bus.InboundMessage) *routing.RoutePeer {
	parentKind := inboundMetadata(msg, metadataKeyParentPeerKind)
	parentID := inboundMetadata(msg, metadataKeyParentPeerID)
	if parentKind == "" || parentID == "" {
		return nil
	}
	return &routing.RoutePeer{Kind: parentKind, ID: parentID}
}

// isNativeSearchProvider 检查 LLM 提供商是否支持原生搜索
func isNativeSearchProvider(p providers.LLMProvider) bool {
	if ns, ok := p.(providers.NativeSearchCapable); ok {
		return ns.SupportsNativeSearch()
	}
	return false
}

// filterClientWebSearch 过滤掉客户端侧的 web_search 巯具
//
// 当 Provider 支持原生搜索时(如 Google Gemini),移除客户端侧的 web_search 巯具,
// 使用 Provider 内置的搜索功能代替。
func filterClientWebSearch(tools []providers.ToolDefinition) []providers.ToolDefinition {
	result := make([]providers.ToolDefinition, 0, len(tools))
	for _, t := range tools {
		if strings.EqualFold(t.Function.Name, "web_search") {
			continue
		}
		result = append(result, t)
	}
	return result
}

// extractProvider 从注册表中提取 Provider(用于重载时的旧 Provider 清理)
func extractProvider(registry *AgentRegistry) (providers.LLMProvider, bool) {
	if registry == nil {
		return nil, false
	}
	// Get any agent to access the provider
	defaultAgent := registry.GetDefaultAgent()
	if defaultAgent == nil {
		return nil, false
	}
	return defaultAgent.Provider, true
}

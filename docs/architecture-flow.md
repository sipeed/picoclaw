# PicoClaw 代码执行流程分析

## 项目概述

PicoClaw 是一个用 Go 编写的超轻量级个人 AI 助手，由 Sipeed 发起。核心内存占用 <10MB，可在 $10 硬件上运行。

## 项目结构

```
picoclaw/
├── cmd/
│   ├── picoclaw/                    # 主 CLI 入口
│   │   ├── main.go                  # 程序入口，Cobra 命令注册
│   │   └── internal/                # CLI 子命令
│   │       ├── agent/               # picoclaw agent（交互式/单次对话）
│   │       ├── gateway/             # picoclaw gateway（长期运行网关）
│   │       ├── onboard/             # picoclaw onboard（初始化配置）
│   │       ├── cron/                # picoclaw cron（定时任务管理）
│   │       ├── skills/              # picoclaw skills（技能管理）
│   │       ├── auth/                # picoclaw auth（Provider 认证）
│   │       ├── migrate/             # picoclaw migrate（配置迁移）
│   │       ├── model/               # picoclaw model（模型管理）
│   │       ├── status/              # picoclaw status（状态查看）
│   │       └── version/             # picoclaw version
│   └── picoclaw-launcher-tui/       # TUI 启动器（终端 UI + WebUI）
├── pkg/                             # 核心库
│   ├── agent/                       # Agent 运行时（核心中的核心）
│   │   ├── loop.go                  # AgentLoop 主循环
│   │   ├── turn.go                  # 单轮对话状态管理
│   │   ├── instance.go              # AgentInstance 实例创建
│   │   ├── context.go               # 上下文构建（系统提示词）
│   │   ├── hooks.go                 # Hook 系统
│   │   ├── steering.go              # 消息注入/转向
│   │   ├── subturn.go               # 子 Agent 并发执行
│   │   ├── eventbus.go              # 事件总线
│   │   ├── registry.go              # Agent 注册表
│   │   ├── definition.go            # AGENT.md/SOUL.md 解析
│   │   ├── memory.go                # 记忆存储
│   │   └── model_resolution.go      # 模型解析
│   ├── bus/                         # 消息总线
│   │   ├── types.go                 # InboundMessage/OutboundMessage
│   │   └── bus.go                   # 消息传递
│   ├── channels/                    # 聊天平台集成（17+）
│   │   ├── manager.go               # ChannelManager 统一管理
│   │   ├── telegram/
│   │   ├── discord/
│   │   ├── weixin/                  # 微信
│   │   ├── wecom/                   # 企业微信
│   │   ├── qq/
│   │   ├── feishu/                  # 飞书
│   │   ├── slack/
│   │   ├── whatsapp/
│   │   └── ...                      # 更多平台
│   ├── providers/                   # LLM Provider 适配器（30+）
│   │   ├── types.go                 # LLMProvider 接口定义
│   │   ├── fallback.go              # 降级链
│   │   ├── cooldown.go              # 冷却追踪
│   │   ├── openai/
│   │   ├── anthropic/
│   │   ├── gemini/
│   │   ├── deepseek/
│   │   └── ...                      # 更多 Provider
│   ├── gateway/                     # HTTP 网关服务
│   │   └── gateway.go               # Gateway 启动与服务编排
│   ├── config/                      # 配置管理
│   │   └── config.go                # Config 结构与加载
│   ├── tools/                       # 工具注册与执行
│   ├── session/                     # 会话持久化（JSONL）
│   ├── memory/                      # 对话记忆
│   ├── skills/                      # 技能系统
│   ├── routing/                     # 模型路由
│   ├── cron/                        # 定时任务服务
│   ├── heartbeat/                   # 心跳服务
│   ├── media/                       # 媒体文件存储
│   ├── voice/                       # 语音转录
│   ├── health/                      # 健康检查 HTTP 服务
│   └── commands/                    # 斜杠命令系统
├── docker/                          # Docker 部署
│   ├── docker-compose.yml           # 三种 profile 部署
│   ├── entrypoint.sh                # 容器入口脚本
│   └── Dockerfile                   # 多种构建方式
├── workspace/                       # 默认工作空间
│   ├── AGENT.md                     # Agent 身份与工具定义
│   ├── SOUL.md                      # Agent 人格设定
│   └── USER.md                      # 用户偏好
└── docs/                            # 文档
```

---

## 一、程序启动流程

### 1.1 CLI 入口 (`cmd/picoclaw/main.go`)

```
main()
  │
  ├── 打印 ASCII Banner
  ├── 创建 Cobra Root Command
  ├── 注册子命令:
  │     ├── onboard    → 初始化 ~/.picoclaw/ 目录与配置
  │     ├── agent      → 单次/交互式对话
  │     ├── gateway    → 启动长期运行网关
  │     ├── cron       → 管理定时任务
  │     ├── skills     → 安装/搜索技能
  │     ├── auth       → Provider OAuth 认证
  │     ├── migrate    → 配置版本迁移
  │     ├── model      → 模型管理
  │     ├── status     → 运行状态查看
  │     └── version    → 版本信息
  └── cmd.Execute()
```

### 1.2 Gateway 启动流程 (`pkg/gateway/gateway.go:Run()`)

这是最核心的运行模式，完整启动流程如下：

```
gateway.Run(debug, homePath, configPath, allowEmptyStartup)
  │
  ├── 1. 初始化日志系统
  │     ├── logger.InitPanic()        → Panic 日志
  │     └── logger.EnableFileLogging() → 文件日志
  │
  ├── 2. 加载配置
  │     └── config.LoadConfig(configPath) → ~/.picoclaw/config.json
  │
  ├── 3. 创建 LLM Provider
  │     └── createStartupProvider(cfg) → OpenAI/Anthropic/Gemini 等
  │
  ├── 4. 创建 MessageBus
  │     └── bus.NewMessageBus() → 消息总线（Inbound/Outbound 通道）
  │
  ├── 5. 创建 AgentLoop
  │     └── agent.NewAgentLoop(cfg, msgBus, provider)
  │           ├── NewAgentRegistry()        → 创建 Agent 注册表
  │           ├── NewCooldownTracker()      → 冷却追踪器
  │           ├── NewFallbackChain()        → 降级链
  │           ├── NewEventManager()         → 事件总线
  │           ├── NewHookManager()          → Hook 管理器
  │           ├── configureHookManager()    → 从配置加载 Hooks
  │           └── registerSharedTools()     → 注册共享工具（web/message/spawn 等）
  │
  ├── 6. 启动后台服务
  │     └── setupAndStartServices()
  │           ├── CronService.Start()       → 定时任务服务
  │           ├── HeartbeatService.Start()  → 心跳监控
  │           ├── MediaStore.Start()        → 媒体文件管理（含清理）
  │           ├── ChannelManager.Start()    → 聊天平台管理器
  │           ├── DeviceService             → 设备管理
  │           └── HealthServer.Start()      → HTTP 健康检查
  │
  ├── 7. 启动 AgentLoop（goroutine）
  │     └── go agentLoop.Run(ctx)
  │
  ├── 8. 配置热重载（可选）
  │     └── setupConfigWatcherPolling() → 监听 config.json 变化
  │
  └── 9. 主循环等待信号
        ├── <-sigChan (Ctrl+C)          → 优雅关闭
        ├── <-configReloadChan          → 配置热重载
        └── <-manualReloadChan          → /reload 端点触发
```

---

## 二、核心数据结构

### 2.1 AgentLoop（`pkg/agent/loop.go`）

AgentLoop 是整个系统的核心调度器：

```go
type AgentLoop struct {
    bus            *bus.MessageBus         // 消息总线
    cfg            *config.Config          // 全局配置
    registry       *AgentRegistry          // Agent 注册表（多 Agent 支持）
    state          *state.Manager          // 状态管理
    eventBus       *EventBus               // 事件总线
    hooks          *HookManager            // Hook 管理器
    fallback       *providers.FallbackChain // LLM 降级链
    channelManager *channels.Manager       // 聊天平台管理器
    mediaStore     media.MediaStore        // 媒体存储
    transcriber    voice.Transcriber       // 语音转录
    cmdRegistry    *commands.Registry      // 斜杠命令注册表
    mcp            mcpRuntime              // MCP 协议运行时
    steering       *steeringQueue          // 消息转向队列
    activeTurnStates sync.Map              // 活跃 Turn 状态表
    // ...
}
```

### 2.2 AgentInstance（`pkg/agent/instance.go`）

每个 Agent 的完整配置实例：

```go
type AgentInstance struct {
    ID             string                    // Agent 标识
    Name           string                    // Agent 名称
    Model          string                    // 主模型
    Fallbacks      []string                  // 降级模型列表
    Workspace      string                    // 工作目录
    MaxIterations  int                       // 最大工具调用迭代次数
    MaxTokens      int                       // 最大输出 Token
    ContextWindow  int                       // 上下文窗口大小
    Provider       providers.LLMProvider     // LLM Provider 实例
    Sessions       session.SessionStore       // 会话存储
    ContextBuilder *ContextBuilder            // 上下文构建器
    Tools          *tools.ToolRegistry        // 工具注册表
    Router         *routing.Router            // 模型路由器
    LightProvider  providers.LLMProvider      // 轻量模型 Provider
    // ...
}
```

### 2.3 TurnState（`pkg/agent/turn.go`）

单轮对话的完整状态：

```go
type TurnPhase string
const (
    TurnPhaseSetup      = "setup"       // 初始化阶段
    TurnPhaseRunning    = "running"     // LLM 调用中
    TurnPhaseTools      = "tools"       // 工具执行中
    TurnPhaseFinalizing = "finalizing"  // 收尾阶段
    TurnPhaseCompleted  = "completed"   // 已完成
    TurnPhaseAborted    = "aborted"     // 已中止
)

type turnState struct {
    turnID      string
    agent       *AgentInstance
    phase       TurnPhase
    iteration   int                // 当前迭代次数
    userMessage string             // 用户消息
    media       []string           // 媒体附件
    finalContent string            // 最终回复
    // 中断控制
    gracefulInterrupt     bool
    hardAbort             bool
    providerCancel        context.CancelFunc
    // SubTurn 支持
    depth                 int
    parentTurnState       *turnState
    childTurnIDs          []string
    pendingResults        chan *ToolResult
    // ...
}
```

### 2.4 消息结构（`pkg/bus/types.go`）

```go
type InboundMessage struct {
    Channel    string            // 平台: "telegram", "discord", "cli" 等
    SenderID   string            // 发送者 ID
    Sender     SenderInfo        // 结构化发送者信息
    ChatID     string            // 会话 ID
    Content    string            // 消息文本
    Media      []string          // 媒体附件 (media:// refs)
    Peer       Peer              // 路由对等体 (direct/group/channel)
    SessionKey string            // 会话标识
    Metadata   map[string]string // 自定义元数据
}

type OutboundMessage struct {
    Channel          string // 平台
    ChatID           string // 会话 ID
    Content          string // 回复文本
    ReplyToMessageID string // 引用回复的消息 ID
}
```

### 2.5 LLM Provider 接口（`pkg/providers/types.go`）

```go
type LLMProvider interface {
    Chat(ctx context.Context, messages []Message, tools []ToolDefinition,
         model string, options map[string]any) (*LLMResponse, error)
    GetDefaultModel() string
}

// 可选接口
type StreamingProvider interface {
    ChatStream(ctx context.Context, ..., onChunk func(accumulated string)) (*LLMResponse, error)
}

type ThinkingCapable interface {
    SupportsThinking() bool
}

type NativeSearchCapable interface {
    SupportsNativeSearch() bool
}
```

---

## 三、消息处理完整流程

### 3.1 AgentLoop 主循环

```
AgentLoop.Run(ctx)                          [loop.go:380]
  │
  ├── ensureHooksInitialized()              → 初始化 Hook 系统
  ├── ensureMCPInitialized()                → 初始化 MCP 运行时
  │
  └── for { select {
        │
        ├── <-ctx.Done()                    → 退出
        │
        └── msg := <-bus.InboundChan()       → 收到入站消息
              │
              ├── resolveSteeringTarget()    → 检查是否有活跃 Turn 需要转向
              │     └── 若有 → drainBusToSteering()  → 排水到转向队列
              │
              ├── processMessage(msg)        → 处理消息（核心）
              │
              ├── buildContinuationTarget()  → 构建延续目标
              │
              └── 处理转向队列中的排队消息
                    └── Continue() → 继续对话
      }}
```

### 3.2 processMessage 详细流程

```
processMessage(ctx, msg)                    [loop.go:1255]
  │
  ├── 1. 日志记录
  │
  ├── 2. 语音转录（如有音频）
  │     └── transcribeAudioInMessage()
  │
  ├── 3. 系统消息路由
  │     └── if channel == "system" → processSystemMessage()
  │
  ├── 4. 消息路由解析
  │     └── resolveMessageRoute(msg)        → 确定 Agent 和会话
  │           ├── registry.ResolveRoute()   → 根据 channel/peer/guild 路由
  │           └── 返回 (route, agentInstance, error)
  │
  ├── 5. 重置 message 工具状态
  │
  ├── 6. 解析 SessionKey
  │     └── resolveScopeKey(route, sessionKey)
  │
  ├── 7. 构建 processOptions
  │     ├── SessionKey
  │     ├── Channel / ChatID / SenderID
  │     ├── UserMessage
  │     ├── Media
  │     └── DefaultResponse
  │
  ├── 8. 斜杠命令检查
  │     └── handleCommand() → 如匹配则直接返回
  │
  ├── 9. 挂起技能检查
  │     └── takePendingSkills() → 应用技能覆盖
  │
  └── 10. runAgentLoop(ctx, agent, opts)   → 执行 Agent 循环
```

### 3.3 runAgentLoop → runTurn（核心执行循环）

```
runAgentLoop(ctx, agent, opts)              [loop.go:1467]
  │
  ├── RecordLastChannel()                   → 记录最后活跃频道
  ├── newTurnState(agent, opts, scope)      → 创建 Turn 状态
  └── runTurn(ctx, ts)                      → 执行 Turn
        │
        ├── 注册活跃 Turn → registerActiveTurn()
        ├── 发射 TurnStart 事件
        │
        ├── 加载会话历史
        │     ├── Sessions.GetHistory(sessionKey)
        │     └── Sessions.GetSummary(sessionKey)
        │
        ├── 构建消息列表
        │     └── ContextBuilder.BuildMessages(history, summary, userMessage, ...)
        │           ├── 构建 System Prompt（含缓存）
        │           │     ├── AGENT.md → Agent 身份、工具、技能
        │           │     ├── SOUL.md → Agent 人格
        │           │     ├── USER.md → 用户偏好
        │           │     ├── Skills → 已安装技能
        │           │     └── 动态上下文（时间、会话信息）
        │           ├── 拼接对话历史
        │           └── 添加当前用户消息 + 媒体
        │
        ├── 解析媒体引用
        │     └── resolveMediaRefs(messages, mediaStore)
        │
        ├── 上下文预算检查
        │     └── isOverContextBudget() → 若超限则 forceCompression()
        │
        ├── 保存用户消息到会话
        │
        ├── 模型选择
        │     └── selectCandidates(agent, userMessage) → 主模型/轻量模型
        │
        └── ═══════════════════════════════════════════
            ║            Turn 迭代循环                   ║
            ═══════════════════════════════════════════
            for iteration < maxIterations {
              │
              ├── 检查硬中止 → hardAbortRequested()
              ├── 检查父 Turn 状态（SubTurn）
              ├── 轮询 SubTurn 结果
              ├── 注入 steering 消息
              │
              ├── 调用 LLM ────────────────────────
              │     ├── 流式: provider.ChatStream()
              │     │     └── onChunk → 实时推送到 Channel
              │     └── 非流式: provider.Chat()
              │
              ├── Hook: Observer 回调
              ├── Hook: Interceptor 拦截
              ├── Hook: Approval 审批
              │
              ├── 解析 LLM 响应
              │     ├── 提取 finalContent（文本回复）
              │     └── 提取 toolCalls（工具调用）
              │
              ├── if 无 toolCalls → 跳出循环
              │
              ├── 执行工具调用
              │     for _, toolCall := range toolCalls {
              │       │
              │       ├── 工具查找 → Tools.Get(toolName)
              │       ├── 工具执行 → tool.Execute(params)
              │       ├── 结果添加到 messages
              │       └── Hook 回调
              │     }
              │
              ├── 模型降级检查
              │     └── fallbackChain.Execute() → 如失败则切换 Provider
              │
              └── 继续迭代 → 再次调用 LLM
            }
```

### 3.4 Turn 完成后的处理

```
runTurn() 返回 turnResult
  │
  ├── 保存 assistant 回复到会话
  ├── 发射 TurnEnd 事件
  │
  └── 回到 runAgentLoop()
        ├── 发布 followUp 消息
        ├── 若 SendResponse → 发送到 OutboundMessage
        └── 返回 finalContent

回到 AgentLoop.Run() 的消息处理
  │
  ├── publishResponseIfNeeded()            → 通过 Bus 发送回复
  │     └── bus.PublishOutbound()
  │           └── ChannelManager 接收
  │                 └── channelWorker 发送到平台
  │
  ├── 处理排队的 steering 消息
  │     └── Continue() → 新一轮 Turn
  │
  └── 最终回复发布
```

---

## 四、Channel 系统架构

### 4.1 ChannelManager（`pkg/channels/manager.go`）

```
ChannelManager
  ├── channels map[string]Channel     → 已注册的平台通道
  ├── workers  map[string]*channelWorker → 每个 Channel 一个 Worker
  ├── mediaStore                       → 媒体存储
  │
  ├── 启动流程:
  │     ├── 遍历配置中的 Channel
  │     ├── 创建 Channel 实例
  │     ├── channel.Start(ctx)         → 启动平台连接
  │     └── 启动 channelWorker goroutine
  │
  ├── 入站流程:
  │     Channel 收到消息
  │       └── 转换为 InboundMessage
  │             └── bus.PublishInbound() → AgentLoop 消费
  │
  └── 出站流程:
        channelWorker 监听 queue chan
          └── 收到 OutboundMessage
                └── channel.SendMessage()
```

### 4.2 支持的 Channel 平台

| Channel | 包名 | 协议 |
|---------|------|------|
| Telegram | `telegram` | Long Polling / Webhook |
| Discord | `discord` | WebSocket Gateway |
| 微信 | `weixin` | iLink API |
| 企业微信 | `wecom` | WebSocket |
| QQ | `qq` | WebSocket |
| 飞书 | `feishu` | WebSocket / SDK |
| Slack | `slack` | Socket Mode |
| WhatsApp | `whatsapp` | Bridge 协议 |
| WhatsApp Native | `whatsapp_native` | 原生协议 |
| 钉钉 | `dingtalk` | Stream Mode |
| Matrix | (内置) | Sync API |
| IRC | `irc` | IRC 协议 |
| LINE | `line` | Messaging API |
| MaixCam | `maixcam` | 设备集成 |
| OneBot | `onebot` | OneBot 协议 |
| Pico | `pico` | 硬件设备 |

### 4.3 Channel 扩展接口

```go
type Channel interface {
    Start(ctx context.Context) error
    HandleMessage(ctx context.Context, msg InboundMessage) error
    Stop(ctx context.Context) error
}

// 可选能力接口
type TypingCapable interface {
    StartTyping(ctx context.Context, chatID string) (stop func(), error)
}

type StreamingCapable interface {
    BeginStream(ctx context.Context, chatID string) (Streamer, error)
}

type PlaceholderCapable interface {
    SendPlaceholder(ctx context.Context, chatID string) error
}

type MessageEditor interface {
    EditMessage(ctx context.Context, chatID, msgID, newContent string) error
    DeleteMessage(ctx context.Context, chatID, msgID string) error
}
```

---

## 五、LLM Provider 系统

### 5.1 Provider 架构

```
LLMProvider 接口
  │
  ├── OpenAI (gpt-4, gpt-4o, o1, o3...)
  ├── Anthropic (Claude 系列)
  ├── Google (Gemini)
  ├── DeepSeek
  ├── Zhipu (GLM)
  ├── OpenRouter
  ├── Azure OpenAI
  ├── AWS Bedrock
  ├── xAI (Grok)
  ├── Ollama (本地模型)
  └── 30+ 更多 Provider
```

### 5.2 降级链（Fallback Chain）

```
selectCandidates(agent, userMessage)
  │
  ├── 模型路由（如启用）
  │     ├── Router.Score(message) → 复杂度评分
  │     ├── 低复杂度 → LightProvider（轻量模型）
  │     └── 高复杂度 → 主 Provider
  │
  └── 降级策略
        ├── 主 Provider 调用失败
        ├── FailoverError 分类:
        │     ├── auth → 认证错误
        │     ├── rate_limit → 限流
        │     ├── billing → 计费问题
        │     ├── timeout → 超时
        │     └── context_overflow → 上下文溢出
        ├── CooldownTracker 记录失败 Provider
        ├── FallbackChain 尝试下一个候选
        └── 所有候选失败 → 返回错误
```

---

## 六、工具系统

### 6.1 内置工具

| 工具名 | 功能 | 说明 |
|--------|------|------|
| `read_file` | 读取文件 | 支持行号范围，有大小限制 |
| `write_file` | 写入文件 | 创建或覆盖文件 |
| `edit_file` | 编辑文件 | 字符串替换 |
| `append_file` | 追加文件 | 追加内容到文件末尾 |
| `list_dir` | 列出目录 | 目录内容浏览 |
| `exec` | 执行命令 | 带沙箱限制的命令执行 |
| `message` | 发送消息 | 通过 Bus 发送到其他 Channel |
| `send_file` | 发送文件 | 文件附件发送 |
| `web_search` | 网页搜索 | Brave/Tavily/DuckDuckGo/Perplexity/SearXNG/百度 |
| `web_fetch` | 网页抓取 | HTTP 请求获取网页内容 |
| `spawn` | 子 Agent | 创建并发 SubTurn |
| `skills_search` | 搜索技能 | 从技能仓库搜索 |
| `skills_install` | 安装技能 | 安装技能到工作空间 |
| `mcp_tool` | MCP 工具 | MCP 协议工具调用 |

### 6.2 工具注册流程

```
NewAgentInstance()
  │
  ├── 创建 ToolRegistry
  │
  ├── 注册基础工具（根据配置开关）:
  │     ├── read_file  → NewReadFileTool(workspace, restrict, maxSize, allowPaths)
  │     ├── write_file → NewWriteFileTool(workspace, restrict, allowPaths)
  │     ├── edit_file  → NewEditFileTool(workspace, restrict)
  │     ├── append_file→ NewAppendFileTool(workspace, restrict)
  │     ├── list_dir   → NewListDirTool(workspace, restrict)
  │     └── exec       → NewExecTool(workspace, restrict, timeout)
  │
  └── registerSharedTools() (AgentLoop 级别):
        ├── web_search  → NewWebSearchTool(brave, tavily, duckduckgo, ...)
        ├── web_fetch   → NewWebFetchTool()
        ├── message     → NewMessageTool(bus)
        ├── send_file   → NewSendFileTool(bus)
        └── spawn       → NewSpawnTool()
```

---

## 七、上下文构建与缓存

### 7.1 ContextBuilder（`pkg/agent/context.go`）

```
ContextBuilder
  │
  ├── workspace          → 工作目录
  ├── skillsLoader       → 技能加载器
  ├── memory             → 记忆存储
  │
  ├── 系统提示词缓存:
  │     ├── cachedSystemPrompt  → 缓存的完整 System Prompt
  │     ├── cachedAt            → 缓存构建时各文件的 mtime
  │     ├── existedAtCache      → 缓存时存在的文件路径集合
  │     └── skillFilesAtCache   → 缓存时技能文件的 mtime 快照
  │
  ├── BuildMessages():
  │     ├── 构建 System Prompt（有缓存则复用）
  │     │     ├── AGENT.md 内容
  │     │     ├── SOUL.md 内容
  │     │     ├── USER.md 内容
  │     │     ├── 已激活技能描述
  │     │     ├── 工具发现提示（BM25/Regex）
  │     │     ├── 动态上下文（当前时间、会话信息）
  │     │     └── 记忆上下文
  │     │
  │     ├── 拼接 System Message
  │     ├── 拼接对话历史（如有摘要则先放摘要）
  │     └── 拼接当前用户消息（含媒体）
  │
  └── 缓存失效检测:
        ├── sourceFilesChanged()  → 检查 mtime 变化
        ├── 检查文件新增/删除
        └── 检查技能文件变化
```

---

## 八、Hook 与事件系统

### 8.1 事件类型

```
EventBus
  │
  ├── EventKindTurnStart         → Turn 开始
  ├── EventKindTurnEnd           → Turn 结束
  ├── EventKindToolCall          → 工具调用
  ├── EventKindToolResult        → 工具结果
  ├── EventKindLLMRequest        → LLM 请求
  ├── EventKindLLMResponse       → LLM 响应
  ├── EventKindSteeringInjected  → Steering 消息注入
  ├── EventKindContextCompress   → 上下文压缩
  └── EventKindError             → 错误
```

### 8.2 Hook 类型

```
HookRegistration
  │
  ├── Observer Hook（观察者）
  │     └── 只读回调，不影响执行流程
  │
  ├── Interceptor Hook（拦截器）
  │     └── 可修改消息内容
  │
  └── Approval Hook（审批）
        └── 可阻断执行，等待人工审批
```

---

## 九、SubTurn 并发子 Agent

### 9.1 SubTurn 流程

```
spawn 工具被调用
  │
  ├── 创建子 turnState
  │     ├── depth = parent.depth + 1
  │     ├── parentTurnState = 当前 Turn
  │     ├── pendingResults channel
  │     └── concurrencySem 信号量（控制并发数）
  │
  ├── goroutine 启动子 Turn
  │     └── runTurn(ctx, childTurnState)
  │           └── 独立的 LLM 调用循环
  │
  ├── 结果通过 pendingResults 传回父 Turn
  │
  └── 父 Turn 在每次迭代开始时轮询结果
        └── select { case result := <-pendingResults }
```

---

## 十、会话与记忆管理

### 10.1 会话存储（JSONL）

```
SessionStore (JSONL 后端)
  │
  ├── AddMessage(sessionKey, role, content)
  ├── AddFullMessage(sessionKey, message)     → 含媒体的消息
  ├── GetHistory(sessionKey) []Message
  ├── GetSummary(sessionKey) string
  │
  ├── 上下文压缩:
  │     ├── 检查触发条件:
  │     │     ├── 消息数 > summarizeMessageThreshold
  │     │     └── Token 使用 > contextWindow * summarizeTokenPercent%
  │     ├── 调用 LLM 生成摘要
  │     ├── 替换历史为摘要 + 最近消息
  │     └── 发射 ContextCompress 事件
  │
  └── 主动压缩:
        └── forceCompression() → 在 LLM 调用前检查预算
```

---

## 十一、配置管理

### 11.1 配置文件路径

```
~/.picoclaw/
  ├── config.json          → 主配置文件
  ├── workspace/
  │   ├── AGENT.md         → Agent 定义
  │   ├── SOUL.md          → Agent 人格
  │   └── USER.md          → 用户偏好
  ├── skills/              → 全局技能目录
  ├── logs/                → 日志目录
  └── .security.yml        → 安全过滤配置
```

### 11.2 关键配置项

```json
{
  "version": 1,
  "agents": {
    "defaults": {
      "model_name": "gpt-4o",
      "max_tool_iterations": 20,
      "max_tokens": 8192,
      "context_window": 32768,
      "steering_mode": "async",
      "restrict_to_workspace": true
    },
    "list": [{ "id": "main", "workspace": "~/.picoclaw/workspace" }]
  },
  "model_list": [
    { "model_name": "gpt-4o", "model": "openai/gpt-4o", "api_key": "sk-..." }
  ],
  "gateway": { "host": "127.0.0.1", "port": 18790 },
  "routing": { "enabled": true, "light_model": "gpt-4o-mini" },
  "tools": { "web": { "brave": { "enabled": true } }, "mcp": { "enabled": true } }
}
```

---

## 十二、端到端消息流示例

以 Telegram 用户发送消息为例：

```
1. Telegram Bot API → telegram channel 收到 Update
      ↓
2. telegram 转换为 InboundMessage:
      {Channel:"telegram", ChatID:"123", SenderID:"alice", Content:"今天天气怎样?"}
      ↓
3. ChannelManager → bus.PublishInbound()
      ↓
4. AgentLoop.Run() 从 bus.InboundChan() 收到消息
      ↓
5. processMessage()
      ├── transcribeAudioInMessage() → 无音频，跳过
      ├── resolveMessageRoute() → 路由到 "main" Agent
      └── runAgentLoop()
            ↓
6. runTurn()
      ├── 加载会话历史
      ├── ContextBuilder.BuildMessages()
      │     ├── 读取 AGENT.md + SOUL.md + USER.md
      │     ├── 生成 System Prompt
      │     └── 拼接历史 + 当前消息
      ↓
7. selectCandidates() → 选择主模型 gpt-4o
      ↓
8. provider.Chat(messages, toolDefs) → 调用 OpenAI API
      │
      返回: { toolCalls: [{name: "web_search", params: {query: "今天天气"}}] }
      ↓
9. 执行 web_search 工具 → 获取搜索结果
      ↓
10. 将工具结果添加到 messages → 再次调用 LLM
      │
      返回: { content: "根据搜索结果，今天北京晴，气温15°C..." }
      ↓
11. 保存 assistant 回复到会话
      ↓
12. runTurn() 返回 → runAgentLoop() 返回
      ↓
13. publishResponseIfNeeded()
      └── bus.PublishOutbound({Channel:"telegram", ChatID:"123", Content:"..."})
      ↓
14. ChannelManager 分发给 telegram channelWorker
      ↓
15. telegram channel 发送回复消息给用户
```

---

## 十三、Docker 部署

### 13.1 三种运行模式

| Profile | 命令 | 用途 |
|---------|------|------|
| `agent` | `docker compose run --rm picoclaw-agent -m "Hello"` | 单次查询 |
| `gateway` | `docker compose --profile gateway up` | 长期运行网关（无 WebUI） |
| `launcher` | `docker compose --profile launcher up` | 网关 + WebUI |

### 13.2 容器内流程

```
entrypoint.sh
  │
  ├── 检查 ~/.picoclaw/ 是否存在
  │     └── 不存在 → picoclaw onboard（自动初始化）
  │
  ├── 检查 config.json 是否存在
  │     └── 不存在 → 从环境变量生成配置
  │
  └── 执行传入的命令
        ├── picoclaw gateway → 启动网关
        └── picoclaw agent -m "..." → 单次对话
```

---

## 十四、关键设计亮点

1. **系统提示词缓存**: 通过 mtime 检测文件变化，避免每次请求重建完整 System Prompt
2. **JSONL 会话存储**: 比 JSON 更高效的日志式存储，适合追加写入
3. **Steering 消息注入**: 允许在 Turn 执行过程中动态注入新消息，实现对话中续
4. **SubTurn 并发**: 通过信号量控制并发子 Agent 数量，结果异步传回父 Turn
5. **Fallback 降级链**: 自动分类错误原因，智能切换备用 Provider
6. **模型路由**: 根据消息复杂度动态选择轻量/重量模型，优化成本
7. **插件化架构**: Channel 和 Provider 均通过接口实现，支持热插拔
8. **配置热重载**: 支持不重启服务更新配置

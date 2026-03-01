# PicoClaw Channel System：完整开发指南

> **影响范围**: `pkg/channels/`, `pkg/bus/`, `pkg/media/`, `pkg/identity/`, `cmd/picoclaw/internal/gateway/`

---

## 目录

- [第一部分：架构总览](#第一部分架构总览)
- [第二部分：迁移指南——从 main 分支迁移到重构分支](#第二部分迁移指南从-main-分支迁移到重构分支)
- [第三部分：新 Channel 开发指南——从零实现一个新 Channel](#第三部分新-channel-开发指南从零实现一个新-channel)
- [第四部分：核心子系统详解](#第四部分核心子系统详解)
- [第五部分：关键设计决策与约定](#第五部分关键设计决策与约定)
- [附录：完整文件清单与接口速查表](#附录完整文件清单与接口速查表)

---

## 第一部分：架构总览

### 1.1 重构前后对比

**重构前（main 分支）**：

```
pkg/channels/
├── telegram.go          # 每个 channel 直接放在 channels 包内
├── discord.go
├── slack.go
├── manager.go           # Manager 直接引用各 channel 类型
├── ...
```

- Channel 实现全部在 `pkg/channels/` 包的顶层
- Manager 通过 `switch` 或 `if-else` 链条直接构造各 channel
- Peer、MessageID 等路由信息埋在 `Metadata map[string]string` 中
- 消息发送没有速率限制和重试
- 没有统一的媒体文件生命周期管理
- 各 channel 各自启动 HTTP 服务器
- 群聊触发过滤逻辑分散在各 channel 中

**重构后（refactor/channel-system 分支）**：

```
pkg/channels/
├── base.go              # BaseChannel 共享抽象层
├── interfaces.go        # 可选能力接口（TypingCapable, MessageEditor, ReactionCapable, PlaceholderCapable, PlaceholderRecorder）
├── README.md            # 英文文档
├── README.zh.md         # 中文文档
├── media.go             # MediaSender 可选接口
├── webhook.go           # WebhookHandler, HealthChecker 可选接口
├── errors.go            # 错误哨兵值（ErrNotRunning, ErrRateLimit, ErrTemporary, ErrSendFailed）
├── errutil.go           # 错误分类帮助函数
├── registry.go          # 工厂注册表（RegisterFactory / getFactory）
├── manager.go           # 统一编排：Worker 队列、速率限制、重试、Typing/Placeholder、共享 HTTP
├── split.go             # 长消息智能分割（保留代码块完整性）
├── telegram/            # 每个 channel 独立子包
│   ├── init.go          # 工厂注册
│   ├── telegram.go      # 实现
│   └── telegram_commands.go
├── discord/
│   ├── init.go
│   └── discord.go
├── slack/ line/ onebot/ dingtalk/ feishu/ wecom/ qq/ whatsapp/ whatsapp_native/ maixcam/ pico/
│   └── ...

pkg/bus/
├── bus.go               # MessageBus（缓冲区 64，安全关闭+排水）
├── types.go             # 结构化消息类型（Peer, SenderInfo, MediaPart, InboundMessage, OutboundMessage, OutboundMediaMessage）

pkg/media/
├── store.go             # MediaStore 接口 + FileMediaStore 实现（两阶段释放，TTL 清理）

pkg/identity/
├── identity.go          # 统一用户身份：规范 "platform:id" 格式 + 向后兼容匹配
```

### 1.2 消息流转全景图

```
┌────────────┐      InboundMessage       ┌───────────┐      LLM + Tools      ┌────────────┐
│  Telegram   │──┐                        │           │                        │            │
│  Discord    │──┤   PublishInbound()     │           │   PublishOutbound()   │            │
│  Slack      │──┼──────────────────────▶ │ MessageBus │ ◀─────────────────── │ AgentLoop  │
│  LINE       │──┤   (buffered chan, 64)  │           │   (buffered chan, 64) │            │
│  ...        │──┘                        │           │                        │            │
└────────────┘                            └─────┬─────┘                        └────────────┘
                                                │
                            SubscribeOutbound() │  SubscribeOutboundMedia()
                                                ▼
                                    ┌───────────────────┐
                                    │   Manager          │
                                    │   ├── dispatchOutbound()    路由到 Worker 队列
                                    │   ├── dispatchOutboundMedia()
                                    │   ├── runWorker()           消息分割 + sendWithRetry()
                                    │   ├── runMediaWorker()      sendMediaWithRetry()
                                    │   ├── preSend()             停止 Typing + 撤销 Reaction + 编辑 Placeholder
                                    │   └── runTTLJanitor()       清理过期 Typing/Placeholder
                                    └────────┬──────────┘
                                             │
                                   channel.Send() / SendMedia()
                                             │
                                             ▼
                                    ┌────────────────┐
                                    │ 各平台 API/SDK  │
                                    └────────────────┘
```

### 1.3 关键设计原则

| 原则 | 说明 |
|------|------|
| **子包隔离** | 每个 channel 一个独立 Go 子包，依赖 `channels` 父包提供的 `BaseChannel` 和接口 |
| **工厂注册** | 各子包通过 `init()` 自注册，Manager 通过名字查找工厂，消除 import 耦合 |
| **能力发现** | 可选能力通过接口（`MediaSender`, `TypingCapable`, `ReactionCapable`, `PlaceholderCapable`, `MessageEditor`, `WebhookHandler`, `HealthChecker`）声明，Manager 运行时类型断言发现 |
| **结构化消息** | Peer、MessageID、SenderInfo 从 Metadata 提升为 InboundMessage 的一等字段 |
| **错误分类** | Channel 返回哨兵错误（`ErrRateLimit`, `ErrTemporary` 等），Manager 据此决定重试策略 |
| **集中编排** | 速率限制、消息分割、重试、Typing/Reaction/Placeholder 全部由 Manager 和 BaseChannel 统一处理，Channel 只负责 Send |

---

## 第二部分：迁移指南——从 main 分支迁移到重构分支

### 2.1 如果你有未合并的 Channel 修改

#### 步骤 1：确认你修改了哪些文件

在 main 分支上，Channel 文件直接位于 `pkg/channels/` 顶层，例如：
- `pkg/channels/telegram.go`
- `pkg/channels/discord.go`

重构后，这些文件已被删除，代码移动到了对应子包：
- `pkg/channels/telegram/telegram.go`
- `pkg/channels/discord/discord.go`

#### 步骤 2：理解结构变化映射

| main 分支文件 | 重构分支位置 | 变化 |
|---|---|---|
| `pkg/channels/telegram.go` | `pkg/channels/telegram/telegram.go` + `init.go` | 包名从 `channels` 变为 `telegram` |
| `pkg/channels/discord.go` | `pkg/channels/discord/discord.go` + `init.go` | 同上 |
| `pkg/channels/manager.go` | `pkg/channels/manager.go` | 大幅重写 |
| _(不存在)_ | `pkg/channels/base.go` | 新增共享抽象层 |
| _(不存在)_ | `pkg/channels/registry.go` | 新增工厂注册表 |
| _(不存在)_ | `pkg/channels/errors.go` + `errutil.go` | 新增错误分类体系 |
| _(不存在)_ | `pkg/channels/interfaces.go` | 新增可选能力接口 |
| _(不存在)_ | `pkg/channels/media.go` | 新增 MediaSender 接口 |
| _(不存在)_ | `pkg/channels/webhook.go` | 新增 WebhookHandler/HealthChecker |
| _(不存在)_ | `pkg/channels/whatsapp_native/` | 新增 WhatsApp 原生模式（whatsmeow） |
| _(不存在)_ | `pkg/channels/split.go` | 新增消息分割（从 utils 迁入） |
| _(不存在)_ | `pkg/bus/types.go` | 新增结构化消息类型 |
| _(不存在)_ | `pkg/media/store.go` | 新增媒体文件生命周期管理 |
| _(不存在)_ | `pkg/identity/identity.go` | 新增统一用户身份 |

#### 步骤 3：迁移你的 Channel 代码

以 Telegram 为例，主要改动项：

**3a. 包声明和导入**

```go
// 旧代码（main 分支）
package channels

import (
    "github.com/sipeed/picoclaw/pkg/bus"
    "github.com/sipeed/picoclaw/pkg/config"
)

// 新代码（重构分支）
package telegram

import (
    "github.com/sipeed/picoclaw/pkg/bus"
    "github.com/sipeed/picoclaw/pkg/channels"     // 引用父包
    "github.com/sipeed/picoclaw/pkg/config"
    "github.com/sipeed/picoclaw/pkg/identity"      // 新增
    "github.com/sipeed/picoclaw/pkg/media"          // 新增（如需媒体）
)
```

**3b. 结构体嵌入 BaseChannel**

```go
// 旧代码：直接持有 bus、config 等字段
type TelegramChannel struct {
    bus       *bus.MessageBus
    config    *config.Config
    running   bool
    allowList []string
    // ...
}

// 新代码：嵌入 BaseChannel，它提供 bus、running、allowList 等
type TelegramChannel struct {
    *channels.BaseChannel          // 嵌入共享抽象
    bot    *telego.Bot
    config *config.Config
    // ... 只保留 channel 特有字段
}
```

**3c. 构造函数**

```go
// 旧代码：直接赋值
func NewTelegramChannel(cfg *config.Config, bus *bus.MessageBus) (*TelegramChannel, error) {
    return &TelegramChannel{
        bus:       bus,
        config:    cfg,
        allowList: cfg.Channels.Telegram.AllowFrom,
        // ...
    }, nil
}

// 新代码：使用 NewBaseChannel + 功能选项
func NewTelegramChannel(cfg *config.Config, bus *bus.MessageBus) (*TelegramChannel, error) {
    base := channels.NewBaseChannel(
        "telegram",                    // 名称
        cfg.Channels.Telegram,         // 原始配置（any 类型）
        bus,                           // 消息总线
        cfg.Channels.Telegram.AllowFrom, // 允许列表
        channels.WithMaxMessageLength(4096),                     // 平台消息长度上限
        channels.WithGroupTrigger(cfg.Channels.Telegram.GroupTrigger), // 群聊触发配置
        channels.WithReasoningChannelID(cfg.Channels.Telegram.ReasoningChannelID), // 思维链路由
    )
    return &TelegramChannel{
        BaseChannel: base,
        bot:         bot,
        config:      cfg,
    }, nil
}
```

**3d. Start/Stop 生命周期**

```go
// 新代码：使用 SetRunning 原子操作
func (c *TelegramChannel) Start(ctx context.Context) error {
    // ... 初始化 bot、webhook 等
    c.SetRunning(true)    // 必须在就绪后调用
    go bh.Start()
    return nil
}

func (c *TelegramChannel) Stop(ctx context.Context) error {
    c.SetRunning(false)   // 必须在清理前调用
    // ... 停止 bot handler、取消 context
    return nil
}
```

**3e. Send 方法的错误返回**

```go
// 旧代码：返回普通 error
func (c *TelegramChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
    if !c.running { return fmt.Errorf("not running") }
    // ...
    if err != nil { return err }
}

// 新代码：必须返回哨兵错误，供 Manager 判断重试策略
func (c *TelegramChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
    if !c.IsRunning() {
        return channels.ErrNotRunning    // ← Manager 不会重试
    }
    // ...
    if err != nil {
        // 使用 ClassifySendError 根据 HTTP 状态码包装错误
        return channels.ClassifySendError(statusCode, err)
        // 或手动包装：
        // return fmt.Errorf("%w: %v", channels.ErrTemporary, err)
        // return fmt.Errorf("%w: %v", channels.ErrRateLimit, err)
        // return fmt.Errorf("%w: %v", channels.ErrSendFailed, err)
    }
    return nil
}
```

**3f. 消息接收（Inbound）**

```go
// 旧代码：直接构造 InboundMessage 并发布
msg := bus.InboundMessage{
    Channel:  "telegram",
    SenderID: senderID,
    ChatID:   chatID,
    Content:  content,
    Metadata: map[string]string{
        "peer_kind": "group",     // 路由信息埋在 metadata
        "peer_id":   chatID,
        "message_id": msgID,
    },
}
c.bus.PublishInbound(ctx, msg)

// 新代码：使用 BaseChannel.HandleMessage，传入结构化字段
sender := bus.SenderInfo{
    Platform:    "telegram",
    PlatformID:  strconv.FormatInt(from.ID, 10),
    CanonicalID: identity.BuildCanonicalID("telegram", strconv.FormatInt(from.ID, 10)),
    Username:    from.Username,
    DisplayName: from.FirstName,
}

peer := bus.Peer{
    Kind: "group",    // 或 "direct"
    ID:   chatID,
}

// HandleMessage 内部调用 IsAllowedSender 检查权限，构建 MediaScope，发布到 bus
c.HandleMessage(ctx, peer, messageID, senderID, chatID, content, mediaRefs, metadata, sender)
```

**3g. 添加工厂注册（必需）**

为你的 channel 创建 `init.go`：

```go
// pkg/channels/telegram/init.go
package telegram

import (
    "github.com/sipeed/picoclaw/pkg/bus"
    "github.com/sipeed/picoclaw/pkg/channels"
    "github.com/sipeed/picoclaw/pkg/config"
)

func init() {
    channels.RegisterFactory("telegram", func(cfg *config.Config, b *bus.MessageBus) (channels.Channel, error) {
        return NewTelegramChannel(cfg, b)
    })
}
```

**3h. 在 Gateway 中导入子包**

```go
// cmd/picoclaw/internal/gateway/helpers.go
import (
    _ "github.com/sipeed/picoclaw/pkg/channels/telegram"   // 触发 init() 注册
    _ "github.com/sipeed/picoclaw/pkg/channels/discord"
    _ "github.com/sipeed/picoclaw/pkg/channels/your_new_channel"  // 新增
)
```

#### 步骤 4：迁移 Bus 消息使用方式

如果你的代码直接读取 `InboundMessage.Metadata` 中的路由字段：

```go
// 旧代码
peerKind := msg.Metadata["peer_kind"]
peerID   := msg.Metadata["peer_id"]
msgID    := msg.Metadata["message_id"]

// 新代码
peerKind := msg.Peer.Kind      // 一等字段
peerID   := msg.Peer.ID        // 一等字段
msgID    := msg.MessageID       // 一等字段
sender   := msg.Sender          // bus.SenderInfo 结构体
scope    := msg.MediaScope       // 媒体生命周期作用域
```

#### 步骤 5：迁移允许列表检查

```go
// 旧代码
if !c.isAllowed(senderID) { return }

// 新代码：优先使用结构化检查
if !c.IsAllowedSender(sender) { return }
// 或回退到字符串检查：
if !c.IsAllowed(senderID) { return }
```

`BaseChannel.HandleMessage` 方法内部已经处理了这个逻辑，无需在 channel 中重复检查。

### 2.2 如果你有 Manager 的修改

Manager 已被完全重写。你的修改需要理解新架构：

| 旧 Manager 职责 | 新 Manager 职责 |
|---|---|
| 直接构造 channel（switch/if-else） | 通过工厂注册表查找并构造 |
| 直接调用 channel.Send | 通过 per-channel Worker 队列 + 速率限制 + 重试 |
| 无消息分割 | 自动根据 MaxMessageLength 分割长消息 |
| 各 channel 自建 HTTP 服务器 | 统一共享 HTTP 服务器 |
| 无 Typing/Placeholder 管理 | 统一 preSend 处理 Typing 停止 + Reaction 撤销 + Placeholder 编辑；入站侧 BaseChannel.HandleMessage 自动编排 Typing/Reaction/Placeholder |
| 无 TTL 清理 | runTTLJanitor 定期清理过期 Typing/Reaction/Placeholder 条目 |

### 2.3 如果你有 Agent Loop 的修改

Agent Loop 的主要变化：

1. **MediaStore 注入**：`agentLoop.SetMediaStore(mediaStore)` — Agent 通过 MediaStore 解析工具产生的媒体引用
2. **ChannelManager 注入**：`agentLoop.SetChannelManager(channelManager)` — Agent 可查询 channel 状态
3. **OutboundMediaMessage**：Agent 现在通过 `bus.PublishOutboundMedia()` 发送媒体消息，而非嵌入文本回复
4. **extractPeer**：路由使用 `msg.Peer` 结构化字段而非 Metadata 查找

---

## 第三部分：新 Channel 开发指南——从零实现一个新 Channel

### 3.1 最小实现清单

要添加一个新的聊天平台（例如 `matrix`），你需要：

1. ✅ 创建子包目录 `pkg/channels/matrix/`
2. ✅ 创建 `init.go` — 工厂注册
3. ✅ 创建 `matrix.go` — Channel 实现
4. ✅ 在 Gateway helpers 中添加 blank import
5. ✅ 在 Manager.initChannels() 中添加配置检查
6. ✅ 在 `pkg/config/` 中添加配置结构体

### 3.2 完整模板

#### `pkg/channels/matrix/init.go`

```go
package matrix

import (
    "github.com/sipeed/picoclaw/pkg/bus"
    "github.com/sipeed/picoclaw/pkg/channels"
    "github.com/sipeed/picoclaw/pkg/config"
)

func init() {
    channels.RegisterFactory("matrix", func(cfg *config.Config, b *bus.MessageBus) (channels.Channel, error) {
        return NewMatrixChannel(cfg, b)
    })
}
```

#### `pkg/channels/matrix/matrix.go`

```go
package matrix

import (
    "context"
    "fmt"

    "github.com/sipeed/picoclaw/pkg/bus"
    "github.com/sipeed/picoclaw/pkg/channels"
    "github.com/sipeed/picoclaw/pkg/config"
    "github.com/sipeed/picoclaw/pkg/identity"
    "github.com/sipeed/picoclaw/pkg/logger"
)

// MatrixChannel implements channels.Channel for the Matrix protocol.
type MatrixChannel struct {
    *channels.BaseChannel            // 必须嵌入
    config *config.Config
    ctx    context.Context
    cancel context.CancelFunc
    // ... Matrix SDK 客户端等
}

func NewMatrixChannel(cfg *config.Config, msgBus *bus.MessageBus) (*MatrixChannel, error) {
    matrixCfg := cfg.Channels.Matrix // 假设配置中有此字段

    base := channels.NewBaseChannel(
        "matrix",                           // channel 名称（全局唯一）
        matrixCfg,                          // 原始配置
        msgBus,                             // 消息总线
        matrixCfg.AllowFrom,                // 允许列表
        channels.WithMaxMessageLength(65536), // Matrix 消息长度限制
        channels.WithGroupTrigger(matrixCfg.GroupTrigger),
        channels.WithReasoningChannelID(matrixCfg.ReasoningChannelID), // 思维链路由（可选）
    )

    return &MatrixChannel{
        BaseChannel: base,
        config:      cfg,
    }, nil
}

// ========== 必须实现的 Channel 接口方法 ==========

func (c *MatrixChannel) Start(ctx context.Context) error {
    c.ctx, c.cancel = context.WithCancel(ctx)

    // 1. 初始化 Matrix 客户端
    // 2. 开始监听消息
    // 3. 标记为运行中
    c.SetRunning(true)

    logger.InfoC("matrix", "Matrix channel started")
    return nil
}

func (c *MatrixChannel) Stop(ctx context.Context) error {
    c.SetRunning(false)

    if c.cancel != nil {
        c.cancel()
    }

    logger.InfoC("matrix", "Matrix channel stopped")
    return nil
}

func (c *MatrixChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
    // 1. 检查运行状态
    if !c.IsRunning() {
        return channels.ErrNotRunning
    }

    // 2. 发送消息到 Matrix
    err := c.sendToMatrix(ctx, msg.ChatID, msg.Content)
    if err != nil {
        // 3. 必须使用错误分类包装
        //    如果你有 HTTP 状态码：
        //    return channels.ClassifySendError(statusCode, err)
        //    如果是网络错误：
        //    return channels.ClassifyNetError(err)
        //    如果需要手动分类：
        return fmt.Errorf("%w: %v", channels.ErrTemporary, err)
    }

    return nil
}

// ========== 消息接收处理 ==========

func (c *MatrixChannel) handleIncoming(roomID, senderID, displayName, content string, msgID string) {
    // 1. 构造结构化发送者身份
    sender := bus.SenderInfo{
        Platform:    "matrix",
        PlatformID:  senderID,
        CanonicalID: identity.BuildCanonicalID("matrix", senderID),
        Username:    senderID,
        DisplayName: displayName,
    }

    // 2. 确定 Peer 类型（直聊 vs 群聊）
    peer := bus.Peer{
        Kind: "group",    // 或 "direct"
        ID:   roomID,
    }

    // 3. 群聊过滤（如适用）
    isGroup := peer.Kind == "group"
    if isGroup {
        isMentioned := false // 根据平台特性检测 @提及
        shouldRespond, cleanContent := c.ShouldRespondInGroup(isMentioned, content)
        if !shouldRespond {
            return
        }
        content = cleanContent
    }

    // 4. 处理媒体附件（如有）
    var mediaRefs []string
    store := c.GetMediaStore()
    if store != nil {
        // 下载附件到本地 → store.Store() → 获取 ref
        // mediaRefs = append(mediaRefs, ref)
    }

    // 5. 调用 HandleMessage 发布到 bus
    //    HandleMessage 内部会：
    //    - 检查 IsAllowedSender/IsAllowed
    //    - 构建 MediaScope
    //    - 发布 InboundMessage
    c.HandleMessage(
        c.ctx,
        peer,
        msgID,                   // 平台消息 ID
        senderID,                // 原始发送者 ID
        roomID,                  // 聊天/房间 ID
        content,                 // 消息内容
        mediaRefs,               // 媒体引用列表
        nil,                     // 额外 metadata（通常 nil）
        sender,                  // SenderInfo（variadic 参数）
    )
}

// ========== 内部方法 ==========

func (c *MatrixChannel) sendToMatrix(ctx context.Context, roomID, content string) error {
    // 实际的 Matrix SDK 调用
    return nil
}
```

### 3.3 可选能力接口

根据平台能力，你的 Channel 可以选择性实现以下接口：

#### MediaSender — 发送媒体附件

```go
// 如果平台支持发送图片/文件/音频/视频
func (c *MatrixChannel) SendMedia(ctx context.Context, msg bus.OutboundMediaMessage) error {
    if !c.IsRunning() {
        return channels.ErrNotRunning
    }

    store := c.GetMediaStore()
    if store == nil {
        return fmt.Errorf("no media store: %w", channels.ErrSendFailed)
    }

    for _, part := range msg.Parts {
        localPath, err := store.Resolve(part.Ref)
        if err != nil {
            logger.ErrorCF("matrix", "Failed to resolve media", map[string]any{
                "ref": part.Ref, "error": err.Error(),
            })
            continue
        }

        // 根据 part.Type ("image"|"audio"|"video"|"file") 调用对应 API
        switch part.Type {
        case "image":
            // 上传图片到 Matrix
        default:
            // 上传文件到 Matrix
        }
    }
    return nil
}
```

#### TypingCapable — Typing 指示器

```go
// 如果平台支持 "正在输入..." 提示
func (c *MatrixChannel) StartTyping(ctx context.Context, chatID string) (stop func(), err error) {
    // 调用 Matrix API 发送 typing 指示器
    // 返回的 stop 函数必须是幂等的
    stopped := false
    return func() {
        if !stopped {
            stopped = true
            // 调用 Matrix API 停止 typing
        }
    }, nil
}
```

#### ReactionCapable — 消息反应指示器

```go
// 如果平台支持对入站消息添加 emoji 反应（如 Slack 的 👀、OneBot 的表情 289）
func (c *MatrixChannel) ReactToMessage(ctx context.Context, chatID, messageID string) (undo func(), err error) {
    // 调用 Matrix API 添加反应到消息
    // 返回的 undo 函数移除反应，必须是幂等的
    err = c.addReaction(chatID, messageID, "eyes")
    if err != nil {
        return func() {}, err
    }
    return func() {
        c.removeReaction(chatID, messageID, "eyes")
    }, nil
}
```

#### MessageEditor — 消息编辑

```go
// 如果平台支持编辑已发送的消息（用于 Placeholder 替换）
func (c *MatrixChannel) EditMessage(ctx context.Context, chatID, messageID, content string) error {
    // 调用 Matrix API 编辑消息
    return nil
}
```

#### PlaceholderCapable — 占位消息

```go
// 如果平台支持发送占位消息（如 "Thinking... 💭"），并且实现了 MessageEditor，
// 则 Manager 的 preSend 会在出站时自动将占位消息编辑为最终回复。
// SendPlaceholder 内部根据 PlaceholderConfig.Enabled 决定是否发送；
// 返回 ("", nil) 表示跳过。
func (c *MatrixChannel) SendPlaceholder(ctx context.Context, chatID string) (string, error) {
    cfg := c.config.Channels.Matrix.Placeholder
    if !cfg.Enabled {
        return "", nil
    }
    text := cfg.Text
    if text == "" {
        text = "Thinking... 💭"
    }
    // 调用 Matrix API 发送占位消息
    msg, err := c.sendText(ctx, chatID, text)
    if err != nil {
        return "", err
    }
    return msg.ID, nil
}
```

#### WebhookHandler — HTTP Webhook 接收

```go
// 如果 channel 通过 webhook 接收消息（而非长轮询/WebSocket）
func (c *MatrixChannel) WebhookPath() string {
    return "/webhook/matrix"   // 路径会被注册到共享 HTTP 服务器
}

func (c *MatrixChannel) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // 处理 webhook 请求
}
```

#### HealthChecker — 健康检查端点

```go
func (c *MatrixChannel) HealthPath() string {
    return "/health/matrix"
}

func (c *MatrixChannel) HealthHandler(w http.ResponseWriter, r *http.Request) {
    if c.IsRunning() {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("OK"))
    } else {
        w.WriteHeader(http.StatusServiceUnavailable)
    }
}
```

### 3.4 入站侧 Typing/Reaction/Placeholder 自动编排

`BaseChannel.HandleMessage` 在发布入站消息**之前**，自动检测 channel 是否实现了 `TypingCapable`、`ReactionCapable` 和/或 `PlaceholderCapable`，并触发相应的指示器。三条管道完全独立，互不干扰：

```go
// BaseChannel.HandleMessage 内部自动执行（无需 channel 手动调用）：
if c.owner != nil && c.placeholderRecorder != nil {
    // Typing — 独立管道
    if tc, ok := c.owner.(TypingCapable); ok {
        if stop, err := tc.StartTyping(ctx, chatID); err == nil {
            c.placeholderRecorder.RecordTypingStop(c.name, chatID, stop)
        }
    }
    // Reaction — 独立管道
    if rc, ok := c.owner.(ReactionCapable); ok && messageID != "" {
        if undo, err := rc.ReactToMessage(ctx, chatID, messageID); err == nil {
            c.placeholderRecorder.RecordReactionUndo(c.name, chatID, undo)
        }
    }
    // Placeholder — 独立管道
    if pc, ok := c.owner.(PlaceholderCapable); ok {
        if phID, err := pc.SendPlaceholder(ctx, chatID); err == nil && phID != "" {
            c.placeholderRecorder.RecordPlaceholder(c.name, chatID, phID)
        }
    }
}
```

**这意味着**：
- 实现 `TypingCapable` 的 channel（Telegram、Discord、LINE、Pico）无需在 `handleMessage` 中手动调用 `StartTyping` + `RecordTypingStop`
- 实现 `ReactionCapable` 的 channel（Slack、OneBot）无需在 `handleMessage` 中手动调用 `AddReaction` + `RecordTypingStop`
- 实现 `PlaceholderCapable` 的 channel（Telegram、Discord、Pico）无需在 `handleMessage` 中手动发送占位消息并调用 `RecordPlaceholder`
- Channel 只需实现对应接口，`HandleMessage` 会自动完成编排
- 不实现这些接口的 channel 不受影响（类型断言会失败，跳过）
- `PlaceholderCapable` 的 `SendPlaceholder` 方法内部根据配置的 `PlaceholderConfig.Enabled` 决定是否发送；返回 `("", nil)` 时跳过注册

**Owner 注入**：Manager 在 `initChannel` 中自动调用 `SetOwner(ch)` 将具体 channel 注入 BaseChannel，无需开发者手动设置。

当 Agent 处理完消息后，Manager 的 `preSend` 会自动：
1. 调用已记录的 `stop()` 停止 Typing
2. 调用已记录的 `undo()` 撤销 Reaction
3. 如果有 Placeholder，且 channel 实现了 `MessageEditor`，尝试编辑 Placeholder 为最终回复（跳过 Send）

### 3.5 注册配置和 Gateway 接入

#### 在 `pkg/config/config.go` 中添加配置

```go
type ChannelsConfig struct {
    // ... 现有 channels
    Matrix  MatrixChannelConfig  `json:"matrix"`
}

type MatrixChannelConfig struct {
    Enabled    bool     `json:"enabled"`
    HomeServer string   `json:"home_server"`
    Token      string   `json:"token"`
    AllowFrom  []string `json:"allow_from"`
    GroupTrigger GroupTriggerConfig `json:"group_trigger"`
    Placeholder  PlaceholderConfig  `json:"placeholder"`
    ReasoningChannelID string `json:"reasoning_channel_id"`
}
```

#### 在 Manager.initChannels() 中添加入口

```go
// pkg/channels/manager.go 的 initChannels() 方法中
if m.config.Channels.Matrix.Enabled && m.config.Channels.Matrix.Token != "" {
    m.initChannel("matrix", "Matrix")
}
```

> **注意**：如果你的 channel 有多种模式（如 WhatsApp Bridge vs Native），需要在 initChannels 中根据配置分支：
> ```go
> if cfg.UseNative {
>     m.initChannel("whatsapp_native", "WhatsApp Native")
> } else {
>     m.initChannel("whatsapp", "WhatsApp")
> }
> ```

#### 在 Gateway 中添加 blank import

```go
// cmd/picoclaw/internal/gateway/helpers.go
import (
    _ "github.com/sipeed/picoclaw/pkg/channels/matrix"
)
```

---

## 第四部分：核心子系统详解

### 4.1 MessageBus

**文件**：`pkg/bus/bus.go`、`pkg/bus/types.go`

```go
type MessageBus struct {
    inbound       chan InboundMessage       // 缓冲区 = 64
    outbound      chan OutboundMessage      // 缓冲区 = 64
    outboundMedia chan OutboundMediaMessage  // 缓冲区 = 64
    done          chan struct{}             // 关闭信号
    closed        atomic.Bool              // 防止重复关闭
}
```

**关键行为**：

| 方法 | 行为 |
|------|------|
| `PublishInbound(ctx, msg)` | 检查 closed → 发送到 inbound channel → 阻塞/超时/关闭 |
| `ConsumeInbound(ctx)` | 从 inbound 读取 → 阻塞/关闭/取消 |
| `PublishOutbound(ctx, msg)` | 发送到 outbound channel |
| `SubscribeOutbound(ctx)` | 从 outbound 读取（Manager dispatcher 调用） |
| `PublishOutboundMedia(ctx, msg)` | 发送到 outboundMedia channel |
| `SubscribeOutboundMedia(ctx)` | 从 outboundMedia 读取（Manager media dispatcher 调用） |
| `Close()` | CAS 关闭 → close(done) → 排水所有 channel（**不关闭 channel 本身**，避免并发 send-on-closed panic） |

**设计要点**：
- 缓冲区从 16 增至 64，减少突发负载下的阻塞
- `Close()` 不关闭底层 channel（只关闭 `done` 信号通道），因为可能有正在并发 `Publish` 的 goroutine
- 排水循环确保 buffered 消息不被静默丢弃

### 4.2 结构化消息类型

**文件**：`pkg/bus/types.go`

```go
// 路由对等体
type Peer struct {
    Kind string `json:"kind"`  // "direct" | "group" | "channel" | ""
    ID   string `json:"id"`
}

// 发送者身份信息
type SenderInfo struct {
    Platform    string `json:"platform,omitempty"`     // "telegram", "discord", ...
    PlatformID  string `json:"platform_id,omitempty"`  // 平台原始 ID
    CanonicalID string `json:"canonical_id,omitempty"` // "platform:id" 规范格式
    Username    string `json:"username,omitempty"`
    DisplayName string `json:"display_name,omitempty"`
}

// 入站消息
type InboundMessage struct {
    Channel    string            // 来源 channel 名称
    SenderID   string            // 发送者 ID（优先使用 CanonicalID）
    Sender     SenderInfo        // 结构化发送者信息
    ChatID     string            // 聊天/房间 ID
    Content    string            // 消息文本
    Media      []string          // 媒体引用列表（media://...）
    Peer       Peer              // 路由对等体（一等字段）
    MessageID  string            // 平台消息 ID（一等字段）
    MediaScope string            // 媒体生命周期作用域
    SessionKey string            // 会话键
    Metadata   map[string]string // 仅用于 channel 特有扩展
}

// 出站文本消息
type OutboundMessage struct {
    Channel string
    ChatID  string
    Content string
}

// 出站媒体消息
type OutboundMediaMessage struct {
    Channel string
    ChatID  string
    Parts   []MediaPart
}

// 媒体片段
type MediaPart struct {
    Type        string // "image" | "audio" | "video" | "file"
    Ref         string // "media://uuid"
    Caption     string
    Filename    string
    ContentType string
}
```

### 4.3 BaseChannel

**文件**：`pkg/channels/base.go`

BaseChannel 是所有 channel 的共享抽象层，提供以下能力：

| 方法/特性 | 说明 |
|---|---|
| `Name() string` | Channel 名称 |
| `IsRunning() bool` | 原子读取运行状态 |
| `SetRunning(bool)` | 原子设置运行状态 |
| `MaxMessageLength() int` | 消息长度限制（rune 计数），0 = 无限制 |
| `ReasoningChannelID() string` | 思维链路由目标 channel ID（空 = 不路由） |
| `IsAllowed(senderID string) bool` | 旧格式允许列表检查（支持 `"id\|username"` 和 `"@username"` 格式） |
| `IsAllowedSender(sender SenderInfo) bool` | 新格式允许列表检查（委托给 `identity.MatchAllowed`） |
| `ShouldRespondInGroup(isMentioned, content) (bool, string)` | 统一群聊触发过滤逻辑 |
| `HandleMessage(...)` | 统一入站消息处理：权限检查 → 构建 MediaScope → 自动触发 Typing/Reaction/Placeholder → 发布到 Bus |
| `SetMediaStore(s) / GetMediaStore()` | Manager 注入的媒体存储 |
| `SetPlaceholderRecorder(r) / GetPlaceholderRecorder()` | Manager 注入的占位符记录器 |
| `SetOwner(ch) ` | Manager 注入的具体 channel 引用（用于 HandleMessage 内部的 Typing/Reaction/Placeholder 类型断言） |

**功能选项**：

```go
channels.WithMaxMessageLength(4096)        // 设置平台消息长度限制
channels.WithGroupTrigger(groupTriggerCfg) // 设置群聊触发配置
channels.WithReasoningChannelID(id)        // 设置思维链路由目标 channel
```

### 4.4 工厂注册表

**文件**：`pkg/channels/registry.go`

```go
type ChannelFactory func(cfg *config.Config, bus *bus.MessageBus) (Channel, error)

func RegisterFactory(name string, f ChannelFactory)   // 子包 init() 中调用
func getFactory(name string) (ChannelFactory, bool)    // Manager 内部调用
```

工厂注册表使用 `sync.RWMutex` 保护，在 `init()` 阶段注册（进程启动时完成）。Manager 在 `initChannel()` 中通过名字查找工厂并调用它。

### 4.5 错误分类与重试

**文件**：`pkg/channels/errors.go`、`pkg/channels/errutil.go`

#### 哨兵错误

```go
var (
    ErrNotRunning = errors.New("channel not running")   // 永久：不重试
    ErrRateLimit  = errors.New("rate limited")           // 固定延迟：1s 后重试
    ErrTemporary  = errors.New("temporary failure")      // 指数退避：500ms * 2^attempt，最大 8s
    ErrSendFailed = errors.New("send failed")            // 永久：不重试
)
```

#### 错误分类帮助函数

```go
// 根据 HTTP 状态码自动分类
func ClassifySendError(statusCode int, rawErr error) error {
    // 429 → ErrRateLimit
    // 5xx → ErrTemporary
    // 4xx → ErrSendFailed
}

// 网络错误统一包装为临时错误
func ClassifyNetError(err error) error {
    // → ErrTemporary
}
```

#### Manager 重试策略（`sendWithRetry`）

```
最大重试次数: 3
速率限制延迟: 1 秒
基础退避:     500 毫秒
最大退避:     8 秒

重试逻辑:
  ErrNotRunning → 立即失败，不重试
  ErrSendFailed → 立即失败，不重试
  ErrRateLimit  → 等待 1s → 重试
  ErrTemporary  → 等待 500ms * 2^attempt（最大 8s） → 重试
  其他未知错误  → 等待 500ms * 2^attempt（最大 8s） → 重试
```

### 4.6 Manager 编排

**文件**：`pkg/channels/manager.go`

#### Per-channel Worker 架构

```go
type channelWorker struct {
    ch         Channel                      // channel 实例
    queue      chan bus.OutboundMessage      // 出站文本队列（缓冲 16）
    mediaQueue chan bus.OutboundMediaMessage // 出站媒体队列（缓冲 16）
    done       chan struct{}                // 文本 worker 完成信号
    mediaDone  chan struct{}                // 媒体 worker 完成信号
    limiter    *rate.Limiter                // per-channel 速率限制器
}
```

#### Per-channel 速率限制配置

```go
var channelRateConfig = map[string]float64{
    "telegram": 20,   // 20 msg/s
    "discord":  1,    // 1 msg/s
    "slack":    1,    // 1 msg/s
    "line":     10,   // 10 msg/s
}
// 默认: 10 msg/s
// burst = max(1, ceil(rate/2))
```

#### 生命周期管理

```
StartAll:
  1. 遍历已注册 channels → channel.Start(ctx)
  2. 为每个启动成功的 channel 创建 channelWorker
  3. 启动 goroutines:
     - runWorker (per-channel 出站文本)
     - runMediaWorker (per-channel 出站媒体)
     - dispatchOutbound (从 bus 路由到 worker 队列)
     - dispatchOutboundMedia (从 bus 路由到 media worker 队列)
     - runTTLJanitor (每 10s 清理过期 typing/reaction/placeholder)
  4. 启动共享 HTTP 服务器（如已配置）

StopAll:
  1. 关闭共享 HTTP 服务器（5s 超时）
  2. 取消 dispatcher context
  3. 关闭 text worker 队列 → 等待排水完成
  4. 关闭 media worker 队列 → 等待排水完成
  5. 停止每个 channel（channel.Stop）
```

#### Typing/Reaction/Placeholder 管理

```go
// Manager 实现 PlaceholderRecorder 接口
func (m *Manager) RecordPlaceholder(channel, chatID, placeholderID string)
func (m *Manager) RecordTypingStop(channel, chatID string, stop func())
func (m *Manager) RecordReactionUndo(channel, chatID string, undo func())

// 入站侧：BaseChannel.HandleMessage 自动编排
// BaseChannel.HandleMessage 在 PublishInbound 之前，通过 owner 类型断言自动触发：
//   - TypingCapable.StartTyping       → RecordTypingStop
//   - ReactionCapable.ReactToMessage  → RecordReactionUndo
//   - PlaceholderCapable.SendPlaceholder → RecordPlaceholder
// 三者独立，互不干扰。Channel 无需手动调用。

// 出站侧：发送前处理
func (m *Manager) preSend(ctx, name, msg, ch) bool {
    key := name + ":" + msg.ChatID
    // 1. 停止 Typing（调用存储的 stop 函数）
    // 2. 撤销 Reaction（调用存储的 undo 函数）
    // 3. 尝试编辑 Placeholder（如果 channel 实现了 MessageEditor）
    //    成功 → return true（跳过 Send）
    //    失败 → return false（继续 Send）
}
```

Manager 存储完全分离，三条管道互不干扰：

```go
Manager {
    typingStops   sync.Map  // "channel:chatID" → typingEntry    ← 管 TypingCapable
    reactionUndos sync.Map  // "channel:chatID" → reactionEntry  ← 管 ReactionCapable
    placeholders  sync.Map  // "channel:chatID" → placeholderEntry
}
```

TTL 清理：
- Typing 停止函数：5 分钟 TTL（到期后自动调用 stop 并删除）
- Reaction 撤销函数：5 分钟 TTL（到期后自动调用 undo 并删除）
- Placeholder ID：10 分钟 TTL（到期后删除）
- 清理间隔：10 秒

### 4.7 消息分割

**文件**：`pkg/channels/split.go`

`SplitMessage(content string, maxLen int) []string`

智能分割策略：
1. 计算有效分割点 = maxLen - 10% 缓冲区（为代码块闭合留空间）
2. 优先在换行符处分割
3. 其次在空格/制表符处分割
4. 检测未闭合的代码块（` ``` `）
5. 如果代码块未闭合：
   - 尝试扩展到 maxLen 以包含闭合围栏
   - 如果代码块太长，注入闭合/重开围栏（`\n```\n` + header）
   - 最后手段：在代码块开始前分割

### 4.8 MediaStore

**文件**：`pkg/media/store.go`

```go
type MediaStore interface {
    Store(localPath string, meta MediaMeta, scope string) (ref string, err error)
    Resolve(ref string) (localPath string, err error)
    ResolveWithMeta(ref string) (localPath string, meta MediaMeta, err error)
    ReleaseAll(scope string) error
}
```

**FileMediaStore 实现**：
- 纯内存映射，不复制/移动文件
- 引用格式：`media://<uuid>`
- Scope 格式：`channel:chatID:messageID`（由 `BuildMediaScope` 生成）
- **两阶段操作**：
  - Phase 1（持锁）：从 map 中收集并删除条目
  - Phase 2（无锁）：从磁盘删除文件
  - 目的：最小化锁争用
- **TTL 清理**：`NewFileMediaStoreWithCleanup` → `Start()` 启动后台清理协程
- 清理间隔和最大存活时间由配置控制

### 4.9 Identity

**文件**：`pkg/identity/identity.go`

```go
// 构建规范 ID
func BuildCanonicalID(platform, platformID string) string
// → "telegram:123456"

// 解析规范 ID
func ParseCanonicalID(canonical string) (platform, id string, ok bool)

// 匹配允许列表（向后兼容）
func MatchAllowed(sender bus.SenderInfo, allowed string) bool
```

`MatchAllowed` 支持的允许列表格式：
| 格式 | 匹配方式 |
|------|----------|
| `"123456"` | 匹配 `sender.PlatformID` |
| `"@alice"` | 匹配 `sender.Username` |
| `"123456\|alice"` | 匹配 PlatformID 或 Username（旧格式兼容） |
| `"telegram:123456"` | 精确匹配 `sender.CanonicalID`（新格式） |

### 4.10 共享 HTTP 服务器

**文件**：`pkg/channels/manager.go` 的 `SetupHTTPServer`

Manager 创建单一 `http.Server`，自动发现和注册：
- 实现 `WebhookHandler` 的 channel → 挂载到 `wh.WebhookPath()`
- 实现 `HealthChecker` 的 channel → 挂载到 `hc.HealthPath()`
- Health 全局端点由 `health.Server.RegisterOnMux` 注册

超时配置：ReadTimeout = 30s, WriteTimeout = 30s

---

## 第五部分：关键设计决策与约定

### 5.1 必须遵守的约定

1. **错误分类是合约**：Channel 的 `Send` 方法**必须**返回哨兵错误（或包装它们）。Manager 的重试策略完全依赖 `errors.Is` 检查。如果返回未分类的错误，Manager 会按"未知错误"处理（指数退避重试）。

2. **SetRunning 是生命周期信号**：`Start` 成功后**必须**调用 `c.SetRunning(true)`，`Stop` 开始时**必须**调用 `c.SetRunning(false)`。`Send` 中**必须**检查 `c.IsRunning()` 并返回 `ErrNotRunning`。

3. **HandleMessage 包含权限检查**：不要在调用 `HandleMessage` 之前自行进行权限检查（除非你需要在检查前做平台特定的预处理）。`HandleMessage` 内部已经调用 `IsAllowedSender`/`IsAllowed`。

4. **消息分割由 Manager 处理**：Channel 的 `Send` 方法不需要处理长消息分割。Manager 会在调用 `Send` 之前根据 `MaxMessageLength()` 自动分割。Channel 只需通过 `WithMaxMessageLength` 声明限制。

5. **Typing/Reaction/Placeholder 由 BaseChannel + Manager 自动处理**：Channel 的 `Send` 方法不需要管理 Typing 停止、Reaction 撤销或 Placeholder 编辑。`BaseChannel.HandleMessage` 在入站侧自动触发 `TypingCapable`、`ReactionCapable` 和 `PlaceholderCapable`（通过 `owner` 类型断言）；Manager 的 `preSend` 在出站侧自动停止 Typing、撤销 Reaction、编辑 Placeholder。Channel 只需实现对应接口即可。

6. **工厂注册在 init() 中**：每个子包必须有 `init.go` 文件调用 `channels.RegisterFactory`。Gateway 必须通过 blank import（`_ "pkg/channels/xxx"`）触发注册。

### 5.2 Metadata 字段使用约定

**不要再把以下信息放入 Metadata**：
- `peer_kind` / `peer_id` → 使用 `InboundMessage.Peer`
- `message_id` → 使用 `InboundMessage.MessageID`
- `sender_platform` / `sender_username` → 使用 `InboundMessage.Sender`

**Metadata 仅用于**：
- Channel 特有的扩展信息（如 Telegram 的 `reply_to_message_id`）
- 不适合放入结构化字段的临时信息

### 5.3 并发安全约定

- `BaseChannel.running`：使用 `atomic.Bool`，线程安全
- `Manager.channels` / `Manager.workers`：使用 `sync.RWMutex` 保护
- `Manager.placeholders` / `Manager.typingStops` / `Manager.reactionUndos`：使用 `sync.Map`
- `MessageBus.closed`：使用 `atomic.Bool`
- `FileMediaStore`：使用 `sync.RWMutex`，两阶段操作减少持锁时间
- Channel Worker queue：Go channel，天然并发安全

### 5.4 测试约定

已有测试文件：
- `pkg/channels/base_test.go` — BaseChannel 单元测试
- `pkg/channels/manager_test.go` — Manager 单元测试
- `pkg/channels/split_test.go` — 消息分割测试
- `pkg/channels/errors_test.go` — 错误类型测试
- `pkg/channels/errutil_test.go` — 错误分类测试

为新 channel 添加测试时：
```bash
go test ./pkg/channels/matrix/ -v              # 子包测试
go test ./pkg/channels/ -run TestSpecific -v    # 框架测试
make test                                       # 全量测试
```

---

## 附录：完整文件清单与接口速查表

### A.1 框架层文件

| 文件 | 职责 |
|------|------|
| `pkg/channels/base.go` | BaseChannel 结构体、Channel 接口、MessageLengthProvider、BaseChannelOption、HandleMessage |
| `pkg/channels/interfaces.go` | TypingCapable、MessageEditor、ReactionCapable、PlaceholderCapable、PlaceholderRecorder 接口 |
| `pkg/channels/media.go` | MediaSender 接口 |
| `pkg/channels/webhook.go` | WebhookHandler、HealthChecker 接口 |
| `pkg/channels/errors.go` | ErrNotRunning、ErrRateLimit、ErrTemporary、ErrSendFailed 哨兵 |
| `pkg/channels/errutil.go` | ClassifySendError、ClassifyNetError 帮助函数 |
| `pkg/channels/registry.go` | RegisterFactory、getFactory 工厂注册表 |
| `pkg/channels/manager.go` | Manager：Worker 队列、速率限制、重试、preSend、共享 HTTP、TTL janitor |
| `pkg/channels/split.go` | SplitMessage 长消息分割 |
| `pkg/bus/bus.go` | MessageBus 实现 |
| `pkg/bus/types.go` | Peer、SenderInfo、InboundMessage、OutboundMessage、OutboundMediaMessage、MediaPart |
| `pkg/media/store.go` | MediaStore 接口、FileMediaStore 实现 |
| `pkg/identity/identity.go` | BuildCanonicalID、ParseCanonicalID、MatchAllowed |

### A.2 Channel 子包

| 子包 | 注册名 | 可选接口 |
|------|--------|----------|
| `pkg/channels/telegram/` | `"telegram"` | TypingCapable, PlaceholderCapable, MessageEditor, MediaSender |
| `pkg/channels/discord/` | `"discord"` | TypingCapable, PlaceholderCapable, MessageEditor, MediaSender |
| `pkg/channels/slack/` | `"slack"` | ReactionCapable, MediaSender |
| `pkg/channels/line/` | `"line"` | TypingCapable, MediaSender, WebhookHandler |
| `pkg/channels/onebot/` | `"onebot"` | ReactionCapable, MediaSender |
| `pkg/channels/dingtalk/` | `"dingtalk"` | — |
| `pkg/channels/feishu/` | `"feishu"` | — (架构特定 build tags: `feishu_32.go` / `feishu_64.go`) |
| `pkg/channels/wecom/` | `"wecom"` | WebhookHandler, HealthChecker |
| `pkg/channels/wecom/` | `"wecom_app"` | MediaSender, WebhookHandler, HealthChecker |
| `pkg/channels/qq/` | `"qq"` | — |
| `pkg/channels/whatsapp/` | `"whatsapp"` | — (Bridge 模式) |
| `pkg/channels/whatsapp_native/` | `"whatsapp_native"` | — (原生 whatsmeow 模式) |
| `pkg/channels/maixcam/` | `"maixcam"` | — |
| `pkg/channels/pico/` | `"pico"` | TypingCapable, PlaceholderCapable, MessageEditor, WebhookHandler |

### A.3 接口速查表

```go
// ===== 必须实现 =====
type Channel interface {
    Name() string
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    Send(ctx context.Context, msg bus.OutboundMessage) error
    IsRunning() bool
    IsAllowed(senderID string) bool
    IsAllowedSender(sender bus.SenderInfo) bool
    ReasoningChannelID() string
}

// ===== 可选实现 =====
type MediaSender interface {
    SendMedia(ctx context.Context, msg bus.OutboundMediaMessage) error
}

type TypingCapable interface {
    StartTyping(ctx context.Context, chatID string) (stop func(), err error)
}

type ReactionCapable interface {
    ReactToMessage(ctx context.Context, chatID, messageID string) (undo func(), err error)
}

type PlaceholderCapable interface {
    SendPlaceholder(ctx context.Context, chatID string) (messageID string, err error)
}

type MessageEditor interface {
    EditMessage(ctx context.Context, chatID, messageID, content string) error
}

type WebhookHandler interface {
    WebhookPath() string
    http.Handler
}

type HealthChecker interface {
    HealthPath() string
    HealthHandler(w http.ResponseWriter, r *http.Request)
}

type MessageLengthProvider interface {
    MaxMessageLength() int
}

// ===== 由 Manager 注入 =====
type PlaceholderRecorder interface {
    RecordPlaceholder(channel, chatID, placeholderID string)
    RecordTypingStop(channel, chatID string, stop func())
    RecordReactionUndo(channel, chatID string, undo func())
}
```

### A.4 Gateway 启动序列（完整引导流程）

```go
// 1. 创建核心组件
msgBus     := bus.NewMessageBus()
provider   := providers.CreateProvider(cfg)
agentLoop  := agent.NewAgentLoop(cfg, msgBus, provider)

// 2. 创建媒体存储（带 TTL 清理）
mediaStore := media.NewFileMediaStoreWithCleanup(cleanerConfig)
mediaStore.Start()

// 3. 创建 Channel Manager（触发 initChannels → 工厂查找 → 构造 → 注入 MediaStore/PlaceholderRecorder/Owner）
channelManager := channels.NewManager(cfg, msgBus, mediaStore)

// 4. 注入引用
agentLoop.SetChannelManager(channelManager)
agentLoop.SetMediaStore(mediaStore)

// 5. 配置共享 HTTP 服务器
channelManager.SetupHTTPServer(addr, healthServer)

// 6. 启动
channelManager.StartAll(ctx)  // 启动 channels + workers + dispatchers + HTTP server
go agentLoop.Run(ctx)          // 启动 Agent 消息循环

// 7. 关闭（信号触发）
cancel()                       // 取消 context
msgBus.Close()                 // 信号关闭 + 排水
channelManager.StopAll(shutdownCtx)  // 停止 HTTP + workers + channels
mediaStore.Stop()              // 停止 TTL 清理
agentLoop.Stop()               // 停止 Agent
```

### A.5 Per-channel 速率限制参考

| Channel | 速率 (msg/s) | Burst |
|---------|-------------|-------|
| telegram | 20 | 10 |
| discord | 1 | 1 |
| slack | 1 | 1 |
| line | 10 | 5 |
| _其他_ | 10 (默认) | 5 |

### A.6 已知限制和注意事项

1. **媒体清理暂时禁用**：Agent loop 中的 `ReleaseAll` 调用被注释掉了（`refactor(loop): disable media cleanup to prevent premature file deletion`），因为会话边界尚未明确定义。TTL 清理仍然有效。

2. **Feishu 架构特定编译**：Feishu channel 使用 build tags 区分 32 位和 64 位架构（`feishu_32.go` / `feishu_64.go`）。Feishu 使用 SDK 的 WebSocket 模式（非 HTTP webhook），因此不实现 `WebhookHandler`。

3. **WeCom 有两个工厂**：`"wecom"`（Bot 模式，纯 webhook）和 `"wecom_app"`（应用模式，支持 MediaSender）分别注册。两者都实现了 `WebhookHandler` 和 `HealthChecker`。

4. **Pico Protocol**：`pkg/channels/pico/` 实现了一个自定义的 PicoClaw 原生协议 channel，通过 WebSocket webhook (`/pico/ws`) 接收消息。

5. **WhatsApp 有两种模式**：`"whatsapp"`（Bridge 模式，通过外部 bridge URL 通信）和 `"whatsapp_native"`（原生 whatsmeow 模式，直接连接 WhatsApp）。Manager 根据 `WhatsAppConfig.UseNative` 决定初始化哪个。

6. **DingTalk 使用 Stream 模式**：DingTalk 使用 SDK 的 Stream/WebSocket 模式（非 HTTP webhook），因此不实现 `WebhookHandler`。

7. **PlaceholderConfig 的配置与实现**：`PlaceholderConfig` 出现在 6 个 channel config 中（Telegram、Discord、Slack、LINE、OneBot、Pico），但只有实现了 `PlaceholderCapable` + `MessageEditor` 的 channel（Telegram、Discord、Pico）能真正使用占位消息编辑功能。其余 channel 的 `PlaceholderConfig` 为预留字段。

8. **ReasoningChannelID**：大多数 channel config 都包含 `reasoning_channel_id` 字段，用于将 LLM 的思维链（reasoning/thinking）路由到指定 channel（WhatsApp、Telegram、Feishu、Discord、MaixCam、QQ、DingTalk、Slack、LINE、OneBot、WeCom、WeComApp）。注意：`PicoConfig` 目前不包含该字段。`BaseChannel` 通过 `WithReasoningChannelID` 选项和 `ReasoningChannelID()` 方法暴露此配置。
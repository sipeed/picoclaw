# 跨 Channel 命令单一来源与平台注册设计

## 背景

当前 Telegram 命令在 `pkg/channels/telegram/telegram.go` 与 `pkg/channels/telegram/telegram_commands.go` 中分散定义，存在以下问题：

- 新增命令需要在多个位置重复修改，容易漏改。
- Telegram 平台侧菜单命令（Bot Commands）未自动注册。
- 未来扩展到 WhatsApp 等 channel 时，无法复用命令定义与行为。

## 目标

- 建立命令定义单一来源（Single Source of Truth）。
- 新增命令时只改一处定义，即可同步到：
  - 命令解析与执行；
  - Telegram 平台命令菜单注册；
  - `help` 命令展示。
- 支持多 channel 分层能力：
  - 所有 channel 共享统一命令解析/执行；
  - 仅对支持平台注册 API 的 channel 启用平台注册（如 Telegram）。
- 启动时命令注册失败不阻塞 channel 启动，采用告警 + 后台重试。

## 非目标

- 本轮不强制所有 channel 都提供平台侧命令菜单。
- 不改动 Agent 主业务消息处理逻辑，只处理命令入口与注册层。
- 不引入复杂命令权限体系（保持 YAGNI，复用现有 allow-list 机制）。

## 决策记录

- 采用方案 A：能力接口分层 + 统一命令目录。
- WhatsApp 等不具备平台注册能力的 channel，仅实现统一解析执行。
- 注册失败策略固定为非阻塞（warn + retry），不引入严格模式开关。

## 架构设计

### 1) 统一命令目录

新增 `pkg/commands`（命令域）作为唯一命令定义源，定义结构包含：

- `Name`：命令名（如 `help`）。
- `Description`：平台菜单描述。
- `Usage`：用户提示（如 `/show [model|channel]`）。
- `Aliases`：可选别名。
- `Channels`：可选 channel 白名单（为空表示全 channel）。
- `Handler`：统一执行入口。

### 2) 统一分发器

新增 `CommandDispatcher`：

- 输入：`CommandRequest`（channel/chat/sender/text/message_id 等）。
- 输出：`DispatchResult`（matched/executed/error）。
- 语义：
  - 命中命令：执行 handler 并返回已处理；
  - 未命中：交还 channel 走普通消息流程（进入 agent）。

### 3) Channel 能力接口分层

在 `pkg/channels` 增加可选接口（不修改现有 `Channel` 主接口）：

- `CommandParserCapable`（可选）：声明 channel 具备命令入口解析。
- `CommandRegistrarCapable`（可选）：声明 channel 支持平台菜单注册。

Telegram 实现 `CommandRegistrarCapable`；WhatsApp 可不实现该接口。

### 4) Telegram 适配层

Telegram channel 在 `Start()` 中执行两条并行职责：

- 消息处理链启动（立即可用）；
- 异步命令注册流程（不阻塞可用性）。

命令注册数据来自统一命令目录，通过映射转换为 Telegram `BotCommand`。

## 启动时序与数据流

### 启动时序

1. 创建 channel 时注入命令定义与 dispatcher。
2. `Start()` 建立连接并启动消息监听。
3. 若 channel 支持注册能力，异步执行 `RegisterCommands()`。
4. 注册失败：记录 warning，按退避策略重试；channel 保持 running。

### 入站消息流

1. channel 收到文本消息。
2. 转换为 `CommandRequest` 并调用 dispatcher。
3. 命中命令：执行并回复。
4. 未命中：按原流程进入 agent。

### 平台注册流

1. 统一命令定义按 channel 过滤可见命令。
2. 转换为平台命令结构并提交平台 API。
3. 成功后标记已注册；失败进入重试。

## 错误处理与可观测性

### 错误分级

- 用户输入错误：返回 usage（非系统错误）。
- 命令执行错误：返回用户可读错误 + error 日志。
- 平台注册错误：warning 日志 + 自动重试，不中断启动。

### 日志建议

- `command registration started/succeeded/failed`
- `command dispatch matched/unmatched`
- `command execution succeeded/failed`

建议字段：`channel`, `command`, `attempt`, `next_retry_seconds`, `error`.

### 重试策略

- 指数退避：`5s -> 15s -> 60s -> 5m -> 10m(cap)`。
- 成功即停止。
- `Stop()` 必须 cancel 重试 goroutine，防止泄漏。

## 测试与验收

### 测试范围

- 单元：registry 唯一性、channel 过滤、dispatcher 匹配与参数解析。
- 集成：Telegram 注册失败不阻塞启动；重试成功后停止重试。
- 回归：现有 `/help /start /show /list` 行为不退化；非命令消息仍进入 agent。

### 验收标准

- 新增命令只改统一定义一处。
- Telegram 自动同步平台菜单命令。
- WhatsApp 等 channel 在无平台注册能力时仍可统一解析执行命令。
- 命令注册失败不阻塞启动，且可观测到重试日志。

## 风险与缓解

- 风险：定义与执行耦合过紧导致测试困难。
  - 缓解：将命令元数据与执行器解耦，执行器可依赖接口注入。
- 风险：channel 适配差异导致行为漂移。
  - 缓解：统一 dispatcher 测试矩阵覆盖不同 channel 输入。
- 风险：重试逻辑 goroutine 泄漏。
  - 缓解：统一 context 生命周期和 `Stop()` 回收测试。


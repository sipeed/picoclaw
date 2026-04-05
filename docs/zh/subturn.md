# 🔄 子回合机制

> 返回 [README](../README.md)

## 概述

`SubTurn` 机制是 PicoClaw 的一项核心功能，允许工具生成孤立的嵌套代理循环来处理复杂的子任务。

通过使用 SubTurn，代理可以将问题分解并在独立的临时会话中运行单独的 LLM 调用。这确保中间推理、后台任务或子代理输出不会污染主对话历史。

## 核心能力

- **上下文隔离**：每个 SubTurn 使用 `ephemeralSessionStore`。其消息历史不会泄露到父任务中，并在完成时被销毁。临时会话最多保存 **50 条消息**；达到此限制时，旧消息会自动截断。
- **深度和并发限制**：防止无限循环和资源耗尽。
  - **最大深度**：最多 3 层嵌套。
  - **最大并发**：每个父回合最多 5 个并发子回合（通过超时为 30 秒的信号量管理）。
- **上下文保护**：支持软上下文限制 (`MaxContextRunes`)。它在达到提供商的硬上下文窗口限制之前主动截断旧消息（同时保留系统提示和最近上下文）。
- **错误恢复**：通过压缩历史和重试自动检测并从提供商上下文长度超出错误和截断错误中恢复。

## 配置 (`SubTurnConfig`)

生成 SubTurn 时，必须提供 `SubTurnConfig`：

| 字段 | 类型 | 描述 |
| :--- | :--- | :--- |
| `Model` | `string` | 子回合使用的 LLM 模型（例如 `gpt-4o-mini`）。**必填。** |
| `Tools` | `[]tools.Tool` | 授予子回合的工具。如果为空，则继承父工具。 |
| `SystemPrompt` | `string` | 子回合的任务描述。作为第一条用户消息发送给 LLM（而不是作为系统提示覆盖）。 |
| `ActualSystemPrompt` | `string` | 可选的显式系统提示，用于替换代理的默认设置。留空以继承父代理的系统提示。 |
| `MaxTokens` | `int` | 生成响应的最大令牌数。 |
| `Async` | `bool` | 控制结果传递模式（同步 vs 异步）。 |
| `Critical` | `bool` | 如果为 `true`，即使父任务正常完成，子回合也会继续运行。 |
| `Timeout` | `time.Duration` | 最大执行时间（默认：5 分钟）。 |
| `MaxContextRunes`| `int` | 软上下文限制。`0` = 自动计算（模型上下文窗口的 75%，推荐），`-1` = 无限制（禁用软截断，仅依赖硬上下文错误恢复），`>0` = 使用指定的字符限制。 |

> **注意：** `Async` 标志**不会**使调用非阻塞。它仅控制结果是否也传递到父级的 `pendingResults` 通道。两种模式都阻塞调用者直到子回合完成。若要实现真正的非阻塞执行，调用者必须在单独的 goroutine 中生成子回合。

## 执行模式

### 同步模式 (`Async: false`)

这是标准模式，调用者需要立即获得结果才能继续。

- 调用者阻塞直到子回合完成。
- 结果**仅**通过函数返回值直接返回。
- 它**不会**传递到父级的待处理结果通道。

**示例：**
```go
cfg := agent.SubTurnConfig{
    Model:        "gpt-4o-mini",
    SystemPrompt: "Analyze the provided codebase...",
    Async:        false,
}
result, err := agent.SpawnSubTurn(ctx, cfg)
// 立即处理结果
```

### 异步模式 (`Async: true`)

用于"发射后不管"操作或并行处理，父回合稍后收集结果。

- 结果传递到父回合的 `pendingResults` 通道。
- 结果也通过函数返回值返回（为保持一致性）。
- 父级的代理循环会在后续迭代中轮询此通道，并将结果自动注入到进行中的对话上下文中作为 `[SubTurn Result]`。

**示例：**
```go
cfg := agent.SubTurnConfig{
    Model:        "gpt-4o-mini",
    SystemPrompt: "Run a background security scan...",
    Async:        true,
}
result, err := agent.SpawnSubTurn(ctx, cfg)
// 结果稍后也会通过通道注入到父循环中
```

## 错误恢复和重试

SubTurn 为临时错误实现了自动重试机制：

| 错误类型 | 最大重试次数 | 恢复操作 |
|:-----------|:------------|:----------------|
| 上下文长度超出 | 2 | 强制压缩历史并重试 |
| 响应截断 (`finish_reason="truncated"`) | 2 | 注入恢复提示并重试 |

### 截断恢复
当 LLM 响应被截断 (`finish_reason="truncated"`) 时，SubTurn 自动：
1. 从 `turnState.lastFinishReason` 检测截断
2. 注入恢复提示："您的上一个响应因长度被截断。请提供更短的完整响应..."
3. 最多重试 2 次

### 上下文错误恢复
当提供商返回上下文长度错误（例如 `context_length_exceeded`）时：
1. 强制压缩消息历史（删除最旧的 50% 对话）
2. 使用压缩后的上下文重试
3. 失败前最多重试 2 次

## 生命周期和取消

SubTurn 在独立上下文中操作，但保持与其父 `turnState` 的结构链接。

### 父任务正常完成
当父任务正常完成时 (`Finish(false)`)：
- **非关键**子回合收到信号，正常退出而不抛出错误。
- **关键** (`Critical: true`) 子回合在后台继续运行。完成后，其结果作为**孤立结果**发出，以免丢失数据。

### 硬中止
当父任务被强制中止时（例如用户使用 `/stop` 中断）：
- 触发级联取消，立即终止所有子回合和孙回合。
- 根回合的会话历史回滚到回合开始时拍摄的快照 (`initialHistoryLength`)，防止脏上下文。SubTurn 不受此回滚影响，因为它们使用无论如何都会被丢弃的临时会话。

## 代理循环集成

### 处理期间的公交耗尽

当消息进入 `Run()` 循环时，代理在调用 `processMessage` 之前启动一个 `drainBusToSteering` goroutine。此 goroutine 与整个处理生命周期并发运行，持续消费总线上任何新的入站消息，将它们重定向到**转向队列**而不是丢弃它们。

这确保如果用户在代理处理（包括 SubTurn 执行期间）发送后续消息，消息不会丢失——它将通过 `dequeueSteeringMessages` 在工具调用迭代之间被取出。

当 `processMessage` 返回时，耗尽 goroutine 自动停止（通过可取消的上下文）。

### 待处理结果轮询

代理循环在每次迭代的两个点轮询异步 SubTurn 结果：
1. **LLM 调用之前**：将任何到达的结果作为 `[SubTurn Result]` 消息注入对话上下文。
2. **所有工具执行之后**：在工具循环期间再次轮询，以捕获工具执行期间到达的结果。
3. **最后一次迭代之后**：回合结束前最后一次轮询，以避免丢失延迟到达的结果。

### 回合状态跟踪

所有活动根回合都注册在 `AgentLoop.activeTurnStates` (`sync.Map`，按键为会话键) 中。这允许 `HardAbort` 和 `/subagents` 可观测性命令找到并操作活动回合。

## 事件总线集成

SubTurn 向 PicoClaw `EventBus` 发出特定事件以进行可观测性和调试：

| 事件类型 | 发出时机 | 负载 |
|:------|:-------------|:--------|
| `subturn_spawn` | 子回合成功初始化 | `SubTurnSpawnPayload{AgentID, Label, ParentTurnID}` |
| `subturn_end` | 子回合完成（成功或错误） | `SubTurnEndPayload{AgentID, Status}` |
| `subturn_result_delivered` | 异步结果成功传递到父级 | `SubTurnResultDeliveredPayload{TargetChannel, TargetChatID, ContentLen}` |
| `subturn_orphan` | 结果无法传递（父级完成或通道已满） | `SubTurnOrphanPayload{ParentTurnID, ChildTurnID, Reason}` |

## API 参考

### SpawnSubTurn（公共入口点）

```go
func SpawnSubTurn(ctx context.Context, cfg SubTurnConfig) (*tools.ToolResult, error)
```

这是代理内部代码（例如测试、直接调用）的导出包级入口点。它从上下文获取 `AgentLoop` 和 `turnState`，并委托给内部的 `spawnSubTurn`。

**要求：**
- `AgentLoop` 必须通过 `WithAgentLoop()` 注入到上下文中
- 父 `turnState` 必须存在于上下文中（从工具调用时自动设置）

**返回：**
- `*tools.ToolResult`：包含带有子回合输出的 `ForLLM` 字段
- `error`：定义错误类型之一或上下文错误

### AgentLoopSpawner（接口实现）

```go
type AgentLoopSpawner struct { al *AgentLoop }

func (s *AgentLoopSpawner) SpawnSubTurn(ctx context.Context, cfg tools.SubTurnConfig) (*tools.ToolResult, error)
```

这实现了 `tools.SubTurnSpawner` 接口，供需要生成子回合而无需直接导入 `agent` 包的工具使用（避免循环依赖）。它在委托给内部 `spawnSubTurn` 之前将 `tools.SubTurnConfig` → `agent.SubTurnConfig`。

### NewSubTurnSpawner

```go
func NewSubTurnSpawner(al *AgentLoop) *AgentLoopSpawner
```

为给定的 AgentLoop 创建一个新的生成器实例。在工具注册期间将返回值传递给 `SpawnTool.SetSpawner()` 或 `SubagentTool.SetSpawner()`。

### Continue

```go
func (al *AgentLoop) Continue(ctx context.Context, sessionKey string) error
```

通过将任何排队的转向消息作为新的 LLM 迭代注入来恢复空闲的代理回合。当代理正在等待且需要处理延迟的转向消息而没有新的入站消息到达时使用。

## 上下文传播

SubTurn 依赖上下文值进行正确操作：

| 上下文键 | 用途 |
|:------------|:---|
| `agentLoopKey` | 存储 `*AgentLoop` 以供工具访问和 SubTurn 生成 |
| `turnStateKey` | 存储 `*turnState` 用于层次跟踪和结果传递 |

### 注入依赖

```go
// 在调用可能生成 SubTurn 的工具之前
ctx = WithAgentLoop(ctx, agentLoop)
ctx = withTurnState(ctx, turnState)
```

### 独立的子上下文

**重要**：子 SubTurn 使用从 `context.Background()` 派生的**独立上下文**，而不是父上下文。这一设计选择：

- 允许关键 SubTurn 在父取消后继续
- 防止父超时影响子执行
- 子有自己的超时保护（`Timeout` 配置或默认 5 分钟）

## 错误类型

| 错误 | 条件 |
|:------|:----------|
| `ErrDepthLimitExceeded` | SubTurn 深度超过 3 层 |
| `ErrInvalidSubTurnConfig` | 必填字段 `Model` 为空 |
| `ErrConcurrencyTimeout` | 所有 5 个并发槽位被占用 30 秒以上 |
| 上下文错误 | 信号量获取期间父上下文被取消 |

## 线程安全

SubTurn 设计用于并发执行：

- **父子关系**：在互斥锁下管理 (`parentTS.mu.Lock()`)
- **活动回合跟踪**：使用 `sync.Map` 进行并发访问 `activeTurnStates`
- **ID 生成**：使用 `atomic.Int64` 生成唯一 SubTurn ID（格式：`subturn-N`，每个 `AgentLoop` 实例全局单调）
- **结果传递**：在锁定下读取父状态，在通道发送前释放（可接受的小竞争窗口）

## 孤立结果

当发生以下情况时结果变为孤立：
1. 父回合在 SubTurn 完成之前完成
2. `pendingResults` 通道已满（缓冲区大小：16）

当结果变为孤立时：
- 向 EventBus 发出 `SubTurnOrphanResultEvent`
- 结果**不会**传递到 LLM 上下文
- 外部系统可以监听此事件以进行自定义处理

### 防止孤立结果
- 对必须完成的重要 SubTurn 使用 `Critical: true`
- 监听 `SubTurnOrphanResultEvent` 以进行可观测性
- 生成许多异步 SubTurn 时考虑 16 缓冲区限制

## 工具继承

### 当 `cfg.Tools` 为空时：
- SubTurn 继承父代理的**所有**工具
- 工具注册在新的 `ToolRegistry` 实例中
- 工具 TTL 与父级独立管理

### 当指定了 `cfg.Tools` 时：
- 只有指定的工具对 SubTurn 可用
- 父工具**不会**合并
- 使用此选项可限制 SubTurn 能力以提高安全性或专注度

**受限 SubTurn 示例：**
```go
cfg := agent.SubTurnConfig{
    Model: "gpt-4o-mini",
    Tools: []tools.Tool{readOnlyTool}, // 仅只读访问
    SystemPrompt: "Analyze the file structure...",
}
```

## 参考

| 常量 | 值 |
|:---------|:------|
| `maxSubTurnDepth` | 3 |
| `maxConcurrentSubTurns` | 5 |
| `concurrencyTimeout` | 30s |
| `defaultSubTurnTimeout` | 5m |
| `maxEphemeralHistorySize` | 50 条消息 |
| `pendingResults` 缓冲区 | 16 |
| `MaxContextRunes` 默认值 | 模型上下文窗口的 75% |

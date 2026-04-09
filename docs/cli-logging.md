# 📝 Picoclaw CLI 日志增强功能

## 🎯 问题描述

旧版本的 Picoclaw 命令行交互模式下，用户发送问题后看不到任何处理进度，只能空白等待，体验不佳。

## ✨ 新增功能

为 `picoclaw agent` 命令添加了**语义化友好的实时日志输出**，让用户清楚看到 AI 的处理过程。

### 功能特性

| 功能 | 说明 | 示例输出 |
|------|------|----------|
| **思考指示** | 显示 AI 开始处理请求 | `⚙️ Thinking (session: cli:default)` |
| **API 调用** | 显示 LLM API 请求详情 | `📡 Calling API: gpt-4 (messages: 5, tools: 3)` |
| **工具执行** | 实时显示工具调用和结果 | `🔧 Tool calling: web_search(query=...)`<br>`✅ Tool completed: web_search (15ms)` |
| **上下文压缩** | 显示会话历史压缩情况 | `🗜️ Context compressed: dropped 10, kept 5 messages` |
| **错误提示** | 友好的错误信息展示 | `❌ Error [LLM]: API timeout` |
| **性能统计** | 显示处理耗时和迭代次数 | `✅ Turn ended: 3 iterations, 1.2s, 500 chars` |

---

## 🚀 使用方法

### 1. 基本交互模式（默认，无日志）

```bash
picoclaw agent
```

### 2. 详细日志模式（推荐）

```bash
# 显示所有日志
picoclaw agent --verbose

# 或简写
picoclaw agent -v
```

### 3. 显示思考过程

```bash
# 只显示思考指示和 API 调用
picoclaw agent --show-think
```

### 4. 显示工具执行日志

```bash
# 显示工具调用和结果
picoclaw agent --show-tools
```

### 5. 组合使用

```bash
# 显示所有日志类型
picoclaw agent --verbose --show-tools --show-think
```

### 6. 发送单条消息

```bash
# 非交互模式，直接输出结果
picoclaw agent -m "今天的天气怎么样？"

# 带日志输出
picoclaw agent -m "查询最新新闻" --verbose
```

---

## 📋 日志类型说明

### 思考过程日志 (`--show-think`)

```
⚙️  Thinking (session: cli:default)
📡 Calling API: gpt-4 (messages: 5, tools: 3)
✅ API response (content: 120 chars, tool calls: 2)
💬 Response:
```

### 工具执行日志 (`--show-tools`)

```
🔧 Tool calling: web_search(query="最新 AI 新闻")
✅ Tool completed: web_search (23ms)
🔧 Tool calling: web_fetch(url="https://...")
✅ Tool completed: web_fetch (156ms)
```

### 详细模式日志 (`--verbose`)

```
⚙️  Thinking (session: cli:default)
📡 Calling API: gpt-4 (messages: 5, tools: 3)
✅ API response (content: 120 chars, tool calls: 2)
🔧 Tool calling: web_search(query="...")
✅ Tool completed: web_search (23ms)
🗜️  Context compressed: dropped 10, kept 5 messages
✅ Turn ended: 3 iterations, 1.2s, 500 chars
⏱️  Completed in 1.5s
```

---

## 🛠️ 技术实现

### 核心文件

- `cmd/picoclaw/internal/agent/interactive.go` - 新增的交互式日志模块
- `cmd/picoclaw/internal/agent/command.go` - 更新的命令定义

### 事件订阅机制

通过订阅 Agent 的 EventBus 实现实时日志：

```go
// 订阅事件
il.StartEventSubscription(agentLoop)

// 处理事件
func (il *InteractiveLogger) handleEvent(evt agent.Event) {
    switch evt.Kind {
    case agent.EventKindTurnStart:
        // 显示思考指示
    case agent.EventKindLLMRequest:
        // 显示 API 调用
    case agent.EventKindToolExecStart:
        // 显示工具调用
    // ... 更多事件类型
    }
}
```

### 支持的事件类型

| 事件类型 | 说明 |
|---------|------|
| `EventKindTurnStart` | 开始处理用户请求 |
| `EventKindLLMRequest` | 发起 LLM API 调用 |
| `EventKindLLMResponse` | 收到 LLM 响应 |
| `EventKindLLMRetry` | API 重试 |
| `EventKindToolExecStart` | 工具开始执行 |
| `EventKindToolExecEnd` | 工具执行完成 |
| `EventKindContextCompress` | 上下文压缩 |
| `EventKindTurnEnd` | 处理完成 |
| `EventKindError` | 错误信息 |

---

## 🎨 日志样式

使用 ANSI 颜色码和 Emoji 图标增强可读性：

- 🔵 蓝色 - 响应输出
- 🟣 紫色 - API 调用
- 🟢 绿色 - 成功操作
- 🟡 黄色 - 工具调用/警告
- 🔴 红色 - 错误信息
- ⚪ 灰色 - 详细信息

---

## 📝 示例会话

```bash
$ picoclaw agent --verbose --show-tools

██████╗ ██╗ ██████╗ ██████╗  ██████╗██╗      █████╗ ██╗    ██╗
██╔══██╗██║██╔════╝██╔═══██╗██╔════╝██║     ██╔══██╗██║    ██║
██████╔╝██║██║     ██║   ██║██║     ██║     ███████║██║ █╗ ██║
██╔═══╝ ██║██║     ██║   ██║██║     ██║     ██╔══██║██║███╗██║
██║     ██║╚██████╗╚██████╔╝╚██████╗███████╗██║  ██║╚███╔███╔╝
╚═╝     ╚═╝ ╚═════╝ ╚═════╝  ╚═════╝╚══════╝╚═╝  ╚═╝ ╚══╝╚══╝

Interactive mode (type 'exit' to quit, Ctrl+C to interrupt)
Verbose logging enabled
Tool execution logging enabled

🟢 You: 查询最新的 AI 新闻

⚙️  Thinking (session: cli:default)
📡 Calling API: gpt-4 (messages: 3, tools: 5)
✅ API response (content: 85 chars, tool calls: 1)
🔧 Tool calling: web_search(query="最新 AI 新闻 2026")
✅ Tool completed: web_search (45ms)
📡 Calling API: gpt-4 (messages: 5, tools: 5)
✅ API response (content: 520 chars)
💬 Response:
🟢 根据最新搜索结果，以下是 AI 领域的最新动态：
1. PicoClaw 项目达到 26K Stars...
2. 新的多模态模型发布...

⏱️  Completed in 2.3s

🟢 You: exit

Goodbye!
```

---

## 🔧 开发调试

### 查看帮助

```bash
picoclaw agent --help
```

### 调试模式

```bash
# 开启 debug 日志级别
picoclaw agent --debug --verbose
```

### 日志级别

可以通过环境变量控制日志级别：

```bash
export PICOCLAW_LOG_LEVEL=debug
picoclaw agent --verbose
```

---

## 📦 构建说明

由于网络问题，可能需要配置 Go 代理：

```bash
# 设置 Go 代理
export GOPROXY=https://goproxy.cn,direct

# 生成嵌入文件
go generate ./...

# 构建
go build -o picoclaw ./cmd/picoclaw
```

---

## 🎯 未来改进

- [ ] 支持进度条显示（长任务处理）
- [ ] 支持日志输出到文件
- [ ] 支持自定义日志格式
- [ ] 支持流式输出（Streaming）
- [ ] 支持日志级别过滤

---

**让等待不再空白，让处理过程透明化！** 🚀

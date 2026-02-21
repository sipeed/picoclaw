# System Architecture

This document provides an overview of PicoClaw's system architecture, including its core components and how they interact.

## High-Level Architecture

PicoClaw follows a **message bus architecture** pattern where components communicate asynchronously through a central message bus. This design enables loose coupling between components and makes it easy to add new channels or modify processing logic.

```
┌─────────────────────────────────────────────────────────────────┐
│                        External Platforms                        │
│    Telegram    Discord    Slack    WhatsApp    CLI    etc.     │
└───────┬───────────┬─────────┬─────────┬─────────┬───────────────┘
        │           │         │         │         │
        ▼           ▼         ▼         ▼         ▼
┌─────────────────────────────────────────────────────────────────┐
│                         Channel Layer                            │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐           │
│  │Telegram  │ │ Discord  │ │  Slack   │ │  CLI     │  ...      │
│  │Channel   │ │ Channel  │ │ Channel  │ │ Channel  │           │
│  └────┬─────┘ └────┬─────┘ └────┬─────┘ └────┬─────┘           │
└───────┼────────────┼────────────┼────────────┼──────────────────┘
        │            │            │            │
        ▼            ▼            ▼            ▼
┌─────────────────────────────────────────────────────────────────┐
│                        Message Bus                               │
│  ┌─────────────────────┐    ┌─────────────────────┐            │
│  │   Inbound Channel   │    │  Outbound Channel   │            │
│  │    (buffer: 100)    │    │    (buffer: 100)    │            │
│  └──────────┬──────────┘    └──────────┬──────────┘            │
└─────────────┼──────────────────────────┼────────────────────────┘
              │                          ▲
              ▼                          │
┌─────────────────────────────────────────────────────────────────┐
│                          Agent Layer                             │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                     AgentLoop                            │   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐     │   │
│  │  │   Router    │→ │   Agent     │→ │   LLM       │     │   │
│  │  │             │  │  Registry   │  │  Provider   │     │   │
│  │  └─────────────┘  └─────────────┘  └─────────────┘     │   │
│  │                          │                               │   │
│  │                          ▼                               │   │
│  │                   ┌─────────────┐                       │   │
│  │                   │    Tools    │                       │   │
│  │                   │  Registry   │                       │   │
│  │                   └─────────────┘                       │   │
│  └─────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
              │
              ▼
┌─────────────────────────────────────────────────────────────────┐
│                        Storage Layer                             │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐            │
│  │  Sessions   │  │   Memory    │  │    Cron     │            │
│  │  Manager    │  │   (MD)      │  │    Jobs     │            │
│  └─────────────┘  └─────────────┘  └─────────────┘            │
└─────────────────────────────────────────────────────────────────┘
```

## Core Components

### 1. Message Bus (`pkg/bus/`)

The message bus is the central communication hub. It provides:

- **Inbound messages**: Messages from external platforms
- **Outbound messages**: Responses to be sent to platforms
- **Handler registration**: Maps channel names to handlers

```go
type MessageBus struct {
    inbound  chan InboundMessage
    outbound chan OutboundMessage
    handlers map[string]MessageHandler
}
```

Key features:
- Buffered channels (capacity: 100) for async processing
- Thread-safe operations with mutex protection
- Context-aware consumption for graceful shutdown

### 2. Agent Loop (`pkg/agent/loop.go`)

The AgentLoop is the core message processing engine:

```go
type AgentLoop struct {
    bus            *bus.MessageBus
    cfg            *config.Config
    registry       *AgentRegistry
    state          *state.Manager
    fallback       *providers.FallbackChain
    channelManager *channels.Manager
}
```

Responsibilities:
- Consumes messages from the inbound bus
- Routes messages to appropriate agents
- Builds context with history and system prompts
- Calls LLM providers with fallback support
- Executes tool calls from LLM responses
- Publishes responses to the outbound bus

### 3. Agent Registry (`pkg/agent/registry.go`)

Manages multiple agents with different configurations:

```go
type AgentRegistry struct {
    agents  map[string]*AgentInstance
    default string
    routes  []routing.Rule
}
```

Features:
- Multi-agent support with isolated workspaces
- Message routing based on channel, peer type, etc.
- Agent-specific tool configurations

### 4. Agent Instance (`pkg/agent/instance.go`)

Represents a single agent configuration:

```go
type AgentInstance struct {
    ID             string
    Model          string
    Candidates     []providers.ModelCandidate
    Workspace      string
    Tools          *tools.ToolRegistry
    Sessions       *session.SessionManager
    ContextBuilder *ContextBuilder
    MaxIterations  int
    ContextWindow  int
}
```

### 5. Channels (`pkg/channels/`)

Channels handle communication with external platforms:

```go
type Channel interface {
    Name() string
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    Send(ctx context.Context, msg bus.OutboundMessage) error
    IsRunning() bool
    IsAllowed(senderID string) bool
}
```

Built-in channels:
- Telegram
- Discord
- Slack
- WhatsApp
- LINE
- QQ (OneBot)
- DingTalk
- Feishu/Lark
- MaixCam

### 6. LLM Providers (`pkg/providers/`)

Providers implement the LLM API integration:

```go
type LLMProvider interface {
    Chat(ctx context.Context, messages []Message, tools []ToolDefinition,
         model string, options map[string]interface{}) (*LLMResponse, error)
    GetDefaultModel() string
}
```

Built-in providers:
- OpenAI-compatible (OpenRouter, Groq, etc.)
- Anthropic/Claude
- Claude CLI
- Codex CLI
- GitHub Copilot

### 7. Tools (`pkg/tools/`)

Tools enable the agent to interact with the world:

```go
type Tool interface {
    Name() string
    Description() string
    Parameters() map[string]interface{}
    Execute(ctx context.Context, args map[string]interface{}) *ToolResult
}
```

Built-in tools:
- `files_read` / `files_write` / `files_list` - File operations
- `exec` - Shell command execution
- `web_search` / `web_fetch` - Web operations
- `message` - Send messages to users
- `spawn` - Spawn subagents
- `cron` - Schedule tasks
- `i2c` / `spi` - Hardware interfaces (Linux only)

### 8. Session Manager (`pkg/session/manager.go`)

Manages conversation history:

```go
type SessionManager struct {
    sessions map[string]*Session
    mu       sync.RWMutex
    storage  string
}

type Session struct {
    Key      string
    Messages []providers.Message
    Summary  string
    Created  time.Time
    Updated  time.Time
}
```

Features:
- In-memory caching with disk persistence
- Automatic summarization when history exceeds thresholds
- Thread-safe operations

### 9. Skills Loader (`pkg/skills/loader.go`)

Loads custom skills from markdown files:

```go
type SkillsLoader struct {
    workspace       string
    workspaceSkills string
    globalSkills    string
    builtinSkills   string
}
```

Skills are loaded from three locations (in priority order):
1. Workspace skills (project-level)
2. Global skills (`~/.picoclaw/skills`)
3. Built-in skills

## Data Flow

See [Data Flow](data-flow.md) for a detailed explanation of how messages flow through the system.

## Configuration

Configuration is loaded from `~/.picoclaw/config.json`:

```json
{
  "providers": {
    "openrouter": {
      "api_key": "sk-or-..."
    }
  },
  "agents": {
    "defaults": {
      "model": "openrouter/anthropic/claude-opus-4-5"
    },
    "list": [
      {
        "id": "main",
        "workspace": "~/.picoclaw/workspace"
      }
    ]
  }
}
```

## Multi-Agent Support

PicoClaw supports multiple agents with isolated configurations:

1. **Agent Definition**: Each agent has a unique ID and workspace
2. **Routing**: Messages are routed based on rules (channel, peer type, etc.)
3. **Isolation**: Each agent has its own sessions, tools, and configuration

See [Multi-Agent Guide](../user-guide/advanced/multi-agent.md) for details.

## Security Sandbox

When `restrict_to_workspace: true` (default):

- File operations are limited to the workspace directory
- Shell commands must execute within workspace
- Dangerous commands are always blocked (`rm -rf`, `format`, `dd`, `shutdown`)

See [Security Sandbox](../user-guide/advanced/security-sandbox.md) for details.

## Workspace Structure

```
~/.picoclaw/workspace/
├── sessions/          # Conversation history
├── memory/            # Long-term memory (MEMORY.md)
├── cron/              # Scheduled jobs
├── skills/            # Custom skills
├── AGENT.md           # Agent behavior guide
├── IDENTITY.md        # Agent identity
└── HEARTBEAT.md       # Periodic task prompts
```

## Key Design Patterns

### 1. Interface-Based Design

Core components are defined as interfaces, making it easy to:
- Add new LLM providers
- Create custom tools
- Implement new channels

### 2. Dependency Injection

Components receive dependencies through constructors:
```go
func NewAgentLoop(cfg *config.Config, msgBus *bus.MessageBus,
                  provider providers.LLMProvider) *AgentLoop
```

### 3. Context Propagation

All long-running operations accept `context.Context` for:
- Cancellation signals
- Timeout handling
- Request-scoped values

### 4. Error Wrapping

Errors are wrapped with context:
```go
return fmt.Errorf("failed to process message: %w", err)
```

### 5. Structured Logging

Logging uses structured fields:
```go
logger.InfoCF("agent", "Processing message",
    map[string]interface{}{
        "channel": msg.Channel,
        "chat_id": msg.ChatID,
    })
```

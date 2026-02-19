# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build and Development Commands

```bash
# Download dependencies
make deps

# Build for current platform (runs go generate first)
make build

# Build for all platforms (linux-amd64, linux-arm64, linux-loong64, linux-riscv64, darwin-arm64, windows-amd64)
make build-all

# Run tests
make test

# Run specific test
go test ./pkg/agent/... -v
go test ./pkg/providers/... -v -run TestFallbackChain

# Run linter
make vet

# Format code
make fmt

# Full check (deps, fmt, vet, test)
make check

# Install to ~/.local/bin
make install

# Run with message
./build/picoclaw agent -m "Hello"

# Run interactive mode
./build/picoclaw agent

# Start gateway (connects to Telegram, Discord, etc.)
./build/picoclaw gateway

# Debug mode
./build/picoclaw agent --debug -m "Hello"
```

## Architecture Overview

PicoClaw is an ultra-lightweight AI assistant written in Go. It follows a message bus architecture where channels (Telegram, Discord, etc.) publish inbound messages and subscribe to outbound responses.

### Core Components

```
cmd/picoclaw/main.go     # Entry point, CLI commands
pkg/
├── agent/               # Core agent logic
│   ├── loop.go          # AgentLoop - message processing, LLM iteration
│   ├── instance.go      # AgentInstance - per-agent configuration
│   ├── registry.go      # AgentRegistry - multi-agent support
│   └── context.go       # ContextBuilder - builds LLM messages
├── bus/                 # Message bus for async communication
├── channels/            # Chat platform integrations (Telegram, Discord, etc.)
├── providers/           # LLM provider implementations
│   ├── openai_compat/   # OpenAI-compatible API (OpenRouter, Groq, etc.)
│   ├── anthropic/       # Anthropic/Claude API
│   └── fallback.go      # Fallback chain for model redundancy
├── tools/               # Tool implementations (files, exec, web, spawn)
├── session/             # Session/history management
├── config/              # Configuration loading
├── routing/             # Message routing to agents
├── skills/              # Skill loading system
├── cron/                # Scheduled tasks
└── heartbeat/           # Periodic task execution
```

### Data Flow

1. **Inbound**: Channel receives message → publishes to bus → AgentLoop consumes
2. **Processing**: AgentLoop routes to agent → builds context → calls LLM → executes tools
3. **Outbound**: Tool/agent publishes response → bus → channel sends to platform

### Key Patterns

- **Tool Interface** ([base.go](pkg/tools/base.go)): All tools implement `Tool` interface with `Name()`, `Description()`, `Parameters()`, `Execute()`
- **ContextualTool**: Tools can implement `SetContext(channel, chatID)` to receive message context
- **AsyncTool**: Tools can implement `SetCallback()` for async operations (spawn, cron)
- **LLMProvider Interface** ([types.go](pkg/providers/types.go)): `Chat()` method with messages, tools, model, options
- **Message Bus** ([bus.go](pkg/bus/bus.go)): Buffered channels (100 capacity) for inbound/outbound messages

### LLM Provider Selection

Providers are selected via config:
1. Check `providers.openrouter.api_key` → use OpenRouter
2. Check `providers.zhipu.api_key` → use Zhipu
3. Check `providers.openai.api_key` → use OpenAI
4. etc.

Model format: `"provider/model"` (e.g., `"openrouter/anthropic/claude-opus-4-5"`) or just model name if provider is inferred.

### Multi-Agent Support

Agents are defined in `config.json` under `agents.list`. Each agent has:
- `id`: Unique identifier
- `workspace`: Isolated workspace directory
- `model`: Model configuration with optional fallbacks
- `subagents`: Allowed subagent IDs for spawn tool

Routing binds channels to specific agents via `bindings` array.

### Session Management

Sessions are stored in `workspace/sessions/<session_key>.json`. Each session tracks:
- Message history
- Summary (auto-generated when history exceeds threshold)
- Last access time

### Security Sandbox

When `restrict_to_workspace: true` (default):
- File operations limited to workspace directory
- Shell commands must execute within workspace
- Dangerous commands always blocked (rm -rf, format, dd, shutdown)

## Configuration

Config file: `~/.picoclaw/config.json`

Key environment variables (override config):
- `PICOCLAW_AGENTS_DEFAULTS_MODEL` - Default model
- `PICOCLAW_HEARTBEAT_ENABLED` - Enable periodic tasks
- `PICOCLAW_TOOLS_WEB_DUCKDUCKGO_ENABLED` - Enable DuckDuckGo search

Workspace layout:
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

## Testing

Tests use standard Go testing. Run with:
```bash
make test              # All tests
go test ./pkg/... -v   # Verbose
```

Integration tests (require external APIs) are tagged with `//go:build integration`.

## Code Style

- Standard Go formatting (`gofmt`, `go fmt`)
- Error wrapping with `fmt.Errorf("context: %w", err)`
- Structured logging via `pkg/logger` with component and fields
- JSON config uses `json:` tags with snake_case

## Reference Documentation

Read these documents when working on specific areas:

| Document | When to Read |
|----------|--------------|
| [docs/developer-guide/architecture.md](docs/developer-guide/architecture.md) | Understanding system architecture, components, data flow |
| [docs/developer-guide/data-flow.md](docs/developer-guide/data-flow.md) | Message bus, event processing, request lifecycle |
| [docs/developer-guide/building.md](docs/developer-guide/building.md) | Building from source, cross-compilation, dependencies |
| [docs/developer-guide/testing.md](docs/developer-guide/testing.md) | Running tests, writing new tests, test patterns |
| [docs/developer-guide/extending/creating-tools.md](docs/developer-guide/extending/creating-tools.md) | Implementing new tools, Tool interface, parameters |
| [docs/developer-guide/extending/creating-providers.md](docs/developer-guide/extending/creating-providers.md) | Adding LLM providers, LLMProvider interface |
| [docs/developer-guide/extending/creating-channels.md](docs/developer-guide/extending/creating-channels.md) | Adding chat platforms, message handling |
| [docs/developer-guide/extending/creating-skills.md](docs/developer-guide/extending/creating-skills.md) | Creating custom skills, skill structure |
| [docs/configuration/config-file.md](docs/configuration/config-file.md) | Configuration schema, all options, environment variables |
| [docs/operations/troubleshooting.md](docs/operations/troubleshooting.md) | Common issues, debugging, error resolution |
| [docs/deployment/docker.md](docs/deployment/docker.md) | Docker setup, compose configuration |
| [docs/deployment/security.md](docs/deployment/security.md) | Production security, sandbox configuration |
| [ROADMAP.md](ROADMAP.md) | Project direction, planned features, priorities |

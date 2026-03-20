# AGENTS.md

This file provides guidance to WARP (warp.dev) when working with code in this repository.

## Build & Development Commands

```bash
make build          # Build for current platform (runs go generate first)
make build-all      # Build for all supported platforms
make test           # Run all tests (runs go generate first)
make lint           # Run golangci-lint
make fmt            # Format code via golangci-lint fmt
make vet            # Run go vet static analysis
make check          # Full pre-commit: deps + fmt + vet + test
make deps           # Download and verify Go module dependencies
make generate       # Run go generate (required before build/test)
```

**Running a single test:**
```bash
go test -run TestName -v ./pkg/session/
```

**Build tags:** The default build uses `-tags stdjson`. WhatsApp native support requires `-tags whatsapp_native` and is limited to platforms where `modernc.org/sqlite` compiles (linux amd64/arm/arm64, darwin, windows amd64).

**CGO is disabled** by default (`CGO_ENABLED=0`).

## Architecture

PicoClaw is a Go monorepo structured as a CLI application with two primary execution modes:

- **Agent mode** (`picoclaw agent`): Interactive or one-shot LLM chat via the terminal.
- **Gateway mode** (`picoclaw gateway`): Long-running process that bridges LLM agents to messaging channels (Telegram, Discord, WhatsApp, Matrix, QQ, DingTalk, LINE, WeCom, Slack, IRC, etc.).

### Core Data Flow

Messages flow through an internal **MessageBus** (`pkg/bus`):

```
Channel (inbound) → MessageBus → AgentLoop → LLM Provider → Tool execution loop → MessageBus → Channel (outbound)
```

The `AgentLoop` (`pkg/agent/loop.go`) is the central orchestrator. It:
1. Consumes inbound messages from the bus
2. Resolves which `AgentInstance` handles the message via `AgentRegistry` + `RouteResolver`
3. Builds context (system prompt, session history, identity/memory/skills files from workspace)
4. Calls the LLM provider with tools
5. Runs a tool-call loop until the LLM produces a final text response or hits `max_tool_iterations`
6. Publishes the response back through the bus

### Key Packages

- **`pkg/agent`**: Agent loop, registry (multi-agent support), instance configuration, context builder. An `AgentInstance` encapsulates workspace, session store, tool registry, model config, and fallback candidates.
- **`pkg/bus`**: Channel-based message bus with typed inbound/outbound/media channels. Decouples channel adapters from the agent loop.
- **`pkg/channels`**: Channel adapters. Each channel implements the `Channel` interface (`Name`, `Start`, `Stop`, `Send`, `IsRunning`, `IsAllowed`). Optional interfaces: `TypingCapable`, `MessageEditor`, `ReactionCapable`, `PlaceholderCapable`. The `Manager` handles lifecycle, message splitting, placeholder editing, and outbound routing.
- **`pkg/providers`**: LLM provider abstraction. `LLMProvider` interface has `Chat()` and `GetDefaultModel()`. Implementations: `openai_compat` (covers most vendors), `anthropic`, `anthropic_messages`, `claude_cli`, `codex_cli`, `github_copilot`, `antigravity`. Includes `FallbackChain` for automatic failover with cooldown tracking.
- **`pkg/tools`**: Tool system. Each tool implements `Tool` interface (`Name`, `Description`, `Parameters`, `Execute`). `ToolRegistry` manages registration with core/hidden tool distinction and TTL-based promotion for MCP-discovered tools. Async tools implement `AsyncExecutor`.
- **`pkg/config`**: Configuration loading from `~/.picoclaw/config.json` with env var overrides (`PICOCLAW_CONFIG`, `PICOCLAW_HOME`). `model_list` is the modern provider config; legacy `providers` is still supported.
- **`pkg/routing`**: Multi-agent routing. `RouteResolver` matches inbound messages to agents based on channel/peer/account bindings. Supports a light-model router for cost optimization.
- **`pkg/session`**: Conversation history storage (file-based per session key). Supports summarization when context grows.
- **`pkg/skills`**: Skill loading from workspace, global, and builtin directories.
- **`pkg/mcp`**: Model Context Protocol integration for dynamic tool discovery.
- **`pkg/commands`**: Unified slash-command registry and executor. Channel adapters forward commands to the central executor rather than handling them locally.
- **`pkg/heartbeat`**: Periodic task execution from `HEARTBEAT.md`.
- **`pkg/memory`**: Long-term memory via `MEMORY.md` in workspace.

### CLI Structure

Entry point: `cmd/picoclaw/main.go` using `cobra`. Subcommands in `cmd/picoclaw/internal/` (agent, gateway, onboard, auth, cron, migrate, model, skills, status, version).

### Web Launcher

`web/frontend` (pnpm-based) + `web/backend` (Go). Built via `make build-launcher`. Provides a browser UI at port 18800 that manages the gateway process.

### Provider Model

Providers are resolved from `model_list` config entries using `vendor/model` format (e.g. `openai/gpt-5.4`). The vendor prefix determines the protocol/API base. `FallbackChain` tries candidates in order with cooldown on failures. `FailoverError` classifies failure reasons to decide retriability.

### Workspace Files

The agent reads several markdown files from its workspace directory at context-build time:
- `AGENTS.md` — agent behavior instructions
- `IDENTITY.md` — agent persona
- `SOUL.md` — agent personality/values
- `USER.md` — user preferences
- `MEMORY.md` — long-term memory
- `HEARTBEAT.md` — periodic task definitions

### Security Sandbox

When `restrict_to_workspace: true` (default), file and exec tools are confined to the workspace directory. Dangerous command patterns (rm -rf, format, dd, shutdown, fork bomb) are blocked even when restrictions are disabled.

## Code Conventions

- Go 1.25+, module path `github.com/sipeed/picoclaw`
- Linting: `golangci-lint` with most linters enabled (see `.golangci.yaml` for disabled list)
- Formatters: `gofmt`, `gofumpt`, `goimports`, `golines` (max 120 chars), `gci` (import ordering: standard → third-party → local module)
- Test framework: standard `testing` + `github.com/stretchr/testify`
- Logging: `pkg/logger` wrapping `zerolog`. Use structured logging: `logger.InfoCF("component", "message", map[string]any{...})`
- All CI checks must pass (`make check`) before merge
- Branch off `main`, squash merge, conventional commits
- AI-assisted contributions require disclosure in PR template

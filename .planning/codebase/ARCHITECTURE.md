# Architecture

**Analysis Date:** 2026-04-10

## Pattern Overview

**Overall:** Event-driven agent gateway with message bus architecture

**Key Characteristics:**
- Multi-agent support with per-instance workspace, session, and tool registry
- Message bus (`pkg/bus`) decouples channels from agent processing loops
- Channel manager (`pkg/channels`) abstracts 17+ messaging platforms behind unified interfaces
- Hot-reloadable gateway with graceful shutdown and provider fallback chains
- Embedded web launcher that serves both the dashboard UI and manages the gateway process

## Layers

**Gateway Layer (`pkg/gateway/`):**
- Purpose: Top-level runtime orchestrator — starts agent loops, channels, services
- Location: `pkg/gateway/gateway.go`
- Contains: Service lifecycle, config loading, signal handling
- Depends on: `pkg/agent`, `pkg/bus`, `pkg/channels`, `pkg/cron`, `pkg/health`
- Used by: CLI entry point (`cmd/picoclaw`), Web launcher (`web/backend/`)

**Agent Layer (`pkg/agent/`):**
- Purpose: Core AI agent loop — LLM interaction, tool execution, context management
- Location: `pkg/agent/loop.go`, `pkg/agent/instance.go`, `pkg/agent/turn.go`
- Contains: AgentLoop, AgentInstance, AgentRegistry, EventBus, HookManager
- Depends on: `pkg/bus`, `pkg/providers`, `pkg/tools`, `pkg/session`, `pkg/memory`
- Used by: Gateway service

**Bus Layer (`pkg/bus/`):**
- Purpose: Asynchronous message routing between channels and agents
- Location: `pkg/bus/bus.go`, `pkg/bus/types.go`
- Contains: MessageBus with inbound, outbound, media, audio, voice channels
- Depends on: `pkg/logger`
- Used by: All agent and channel code

**Channel Layer (`pkg/channels/`):**
- Purpose: Platform-agnostic messaging interface with per-platform adapters
- Location: `pkg/channels/` with subpackages for each platform
- Contains: Channel interface, Manager, dynamic mux, per-platform implementations
- Depends on: `pkg/bus`, `pkg/config`, `pkg/health`
- Used by: Gateway service

**Provider Layer (`pkg/providers/`):**
- Purpose: LLM provider abstraction with fallback, routing, and rate limiting
- Location: `pkg/providers/` with subpackages for Anthropic, OpenAI, Bedrock, etc.
- Contains: LLMProvider interface, factory, fallback chain, model router
- Depends on: `pkg/logger`, `pkg/config`
- Used by: Agent instances

**Tool Layer (`pkg/tools/`):**
- Purpose: Tool registry and built-in tool implementations
- Location: `pkg/tools/`
- Contains: ToolRegistry, shell, filesystem, web, MCP, spawn, SPI/I2C hardware tools
- Depends on: `pkg/logger`, `pkg/providers`
- Used by: Agent instances during turn execution

**Session Layer (`pkg/session/`):**
- Purpose: Session persistence and management with JSONL backend
- Location: `pkg/session/manager.go`, `pkg/session/session_store.go`
- Contains: SessionManager, SessionStore interface, JSONL backend
- Depends on: `pkg/providers`
- Used by: Agent instances for conversation history

## Data Flow

**Message Processing Flow:**

1. External message arrives on a channel (e.g., Telegram webhook, Discord event, Pico WebSocket)
2. Channel adapter converts platform message to `bus.InboundMessage` and publishes to the bus
3. `AgentLoop` receives the inbound message from the bus
4. `EventBus` fires pre-processing hooks; `HookManager` executes registered hooks
5. `ContextBuilder` assembles the prompt (system, history, skills, tools)
6. Agent routes to the correct `AgentInstance` via `AgentRegistry` (with optional model routing)
7. LLM call via `Provider` (with fallback chain on failure)
8. Tool calls are dispatched via `ToolRegistry.Execute()` in a tool loop
9. Final content is published as `bus.OutboundMessage`
10. `ChannelManager` routes the response back through the originating channel
11. Channel sends response to the user on the platform

**State Management:**
- Session state persisted as JSONL files in `~/.picoclaw/sessions/`
- Agent context managed in-memory with `ContextBuilder` + `ContextManager`
- Long-term memory via `pkg/memory` package (JSONL-based store)
- Config loaded from `~/.picoclaw/config.json` with hot-reload support

## Key Abstractions

**Channel Interface:**
- Purpose: Abstracts 17+ messaging platforms behind a common interface
- Examples: `pkg/channels/telegram/`, `pkg/channels/discord/`, `pkg/channels/pico/`
- Pattern: Capability-based interfaces (`TypingCapable`, `StreamingCapable`, `MessageEditor`, `ReactionCapable`, `PlaceholderCapable`, `CommandRegistrarCapable`)

**LLMProvider Interface:**
- Purpose: Unified LLM interaction contract
- Examples: `pkg/providers/anthropic/`, `pkg/providers/openai_compat/`, `pkg/providers/bedrock/`
- Pattern: Factory-based provider creation with per-candidate credentials

**Tool Interface:**
- Purpose: Pluggable tool execution with TTL-based registration
- Examples: `pkg/tools/shell.go`, `pkg/tools/filesystem.go`, `pkg/tools/mcp_tool.go`
- Pattern: `Tool` interface with `Name()`, `Description()`, `Parameters()`, `Execute()`

**ContextManager:**
- Purpose: Manages conversation context window with budget-based truncation
- Examples: `pkg/agent/context_budget.go`, `pkg/agent/context_seahorse.go`, `pkg/agent/context_legacy.go`
- Pattern: Strategy pattern with multiple implementations for different context strategies

## Entry Points

**CLI Agent (`cmd/picoclaw/main.go`):**
- Location: `cmd/picoclaw/main.go`
- Triggers: Command-line invocation
- Responsibilities: Subcommands for agent, auth, cron, gateway, skills, model, migrate, status, version

**Web Launcher (`web/backend/main.go`):**
- Location: `web/backend/main.go`
- Triggers: Direct execution or system tray
- Responsibilities: Embedded HTTP server (default port 18800), dashboard auth, gateway auto-start, system tray

**Gateway Service (`pkg/gateway/gateway.go`):**
- Location: `pkg/gateway/gateway.go`
- Triggers: `gateway` subcommand or launcher auto-start
- Responsibilities: Full runtime — agent loops, channel workers, cron, heartbeat, health server

## Error Handling

**Strategy:** Structured logging with provider fallback chains

**Patterns:**
- `providers.FallbackChain` — automatic failover to backup models on error
- `routing.Router` — intelligent model selection based on message complexity
- `channels.Manager` — per-channel rate limiting with exponential backoff
- `providers.ErrorClassifier` — categorizes errors (rate limit, auth, context, etc.) for appropriate retry behavior

## Cross-Cutting Concerns

**Logging:** `pkg/logger` — structured JSON logging with console/file modes, component-tagged output (`InfoCF`, `ErrorCF`, etc.)
**Validation:** `pkg/tools/validate.go` — tool parameter validation with JSON schema
**Authentication:** Dashboard auth (`web/backend/dashboardauth/`), OAuth flows (`pkg/auth/oauth.go`), PKCE for external platforms
**Configuration:** JSON/YAML config with environment variable override support (`pkg/config/`)
**Security:** Sensitive data filtering before LLM calls, credential storage isolation

---

*Architecture analysis: 2026-04-10*

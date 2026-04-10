# Codebase Structure

**Analysis Date:** 2026-04-10

## Directory Layout

```
picoclaw/
├── cmd/                          # CLI entry points
│   ├── picoclaw/                 # Main CLI agent
│   │   ├── main.go               # CLI root command
│   │   └── internal/             # CLI subcommand implementations
│   │       ├── agent/            # `picoclaw agent` command
│   │       ├── auth/             # `picoclaw auth` command
│   │       ├── cron/             # `picoclaw cron` subcommands
│   │       ├── gateway/          # `picoclaw gateway` command
│   │       ├── migrate/          # `picoclaw migrate` command
│   │       ├── model/            # `picoclaw model` command
│   │       ├── onboard/          # `picoclaw onboard` command
│   │       ├── skills/           # `picoclaw skills` subcommands
│   │       ├── status/           # `picoclaw status` command
│   │       └── version/          # `picoclaw version` command
│   ├── picoclaw-launcher-tui/    # Terminal UI launcher
│   │   ├── main.go
│   │   ├── ui/                   # TUI screen components
│   │   └── config/               # TUI configuration
│   └── membench/                 # Memory/performance benchmarking
├── pkg/                          # Shared libraries (all reusable packages)
│   ├── agent/                    # Agent loop, instances, registry
│   ├── audio/                    # Audio processing (ASR/TTS)
│   │   ├── asr/                  # Speech recognition (Whisper, ElevenLabs, etc.)
│   │   └── tts/                  # Text-to-speech (OpenAI, Mimo, etc.)
│   ├── auth/                     # OAuth, PKCE, token management
│   ├── bus/                      # Message bus (core async messaging)
│   ├── channels/                 # Messaging platform adapters
│   │   ├── dingtalk/             # DingTalk (钉钉) integration
│   │   ├── discord/              # Discord bot + voice
│   │   ├── feishu/               # Feishu/Lark integration
│   │   ├── irc/                  # IRC client
│   │   ├── line/                 # LINE messaging
│   │   ├── maixcam/              # MaixCam device channel
│   │   ├── matrix/               # Matrix protocol
│   │   ├── onebot/               # OneBot protocol (QQ)
│   │   ├── pico/                 # Built-in WebSocket channel (for web UI)
│   │   ├── qq/                   # QQ Bot
│   │   ├── slack/                # Slack integration
│   │   ├── teams_webhook/        # Microsoft Teams webhook
│   │   ├── telegram/             # Telegram bot
│   │   ├── vk/                   # VK (ВКонтакте)
│   │   ├── wecom/                # WeChat Work (企业微信)
│   │   ├── weixin/               # WeChat (微信)
│   │   ├── whatsapp/             # WhatsApp (web-based)
│   │   └── whatsapp_native/      # WhatsApp native integration
│   ├── commands/                 # Command definition and registry
│   ├── config/                   # Configuration loading, validation, migration
│   ├── constants/                # Shared constants
│   ├── credential/               # Secure credential storage
│   ├── cron/                     # Cron job scheduler
│   ├── devices/                  # Device management (hardware I/O)
│   ├── fileutil/                 # File system utilities
│   ├── gateway/                  # Gateway runtime orchestrator
│   ├── health/                   # Health check server
│   ├── heartbeat/                # Heartbeat service
│   ├── identity/                 # User identity management
│   ├── isolation/                # Process isolation/sandboxing
│   ├── logger/                   # Structured logging
│   ├── mcp/                      # MCP (Model Context Protocol) support
│   ├── media/                    # Media file storage
│   ├── memory/                   # Long-term conversation memory (JSONL)
│   ├── migrate/                  # Config/data migration
│   ├── pid/                      # PID file management
│   ├── providers/                # LLM provider implementations
│   │   ├── anthropic/            # Anthropic API (Claude)
│   │   ├── anthropic_messages/   # Anthropic Messages API
│   │   ├── azure/                # Azure OpenAI
│   │   ├── bedrock/              # AWS Bedrock
│   │   ├── openai_compat/        # OpenAI-compatible providers
│   │   └── common/               # Shared provider utilities
│   ├── routing/                  # Model routing (smart model selection)
│   ├── seahorse/                 # Context compression engine (FTS5-based)
│   ├── session/                  # Session management (JSONL backend)
│   ├── skills/                   # Skill system (agent capabilities)
│   ├── state/                    # State management
│   ├── tokenizer/                # Token counting/estimation
│   ├── tools/                    # Tool implementations
│   ├── updater/                  # Self-update mechanism
│   └── utils/                    # General utilities
├── web/
│   ├── backend/                  # Web launcher backend (Go)
│   │   ├── main.go               # Launcher entry point
│   │   ├── api/                  # REST API handlers
│   │   │   ├── router.go         # Route registration
│   │   │   ├── channels.go       # Channel CRUD
│   │   │   ├── config.go         # Config management
│   │   │   ├── gateway.go        # Gateway start/stop/logs
│   │   │   ├── models.go         # Model list management
│   │   │   ├── oauth.go          # OAuth flow handlers
│   │   │   ├── pico.go           # WebSocket chat endpoint
│   │   │   ├── session.go        # Session history API
│   │   │   ├── skills.go         # Skills management
│   │   │   └── tools.go          # Tool actions
│   │   ├── dashboardauth/        # Dashboard authentication
│   │   ├── launcherconfig/       # Launcher-specific config
│   │   ├── middleware/           # HTTP middleware (auth, access control)
│   │   ├── model/                # Status models
│   │   └── utils/                # Backend utilities
│   └── frontend/                 # Web dashboard UI (React + TypeScript)
│       ├── src/
│       │   ├── api/              # API client layer
│       │   ├── components/       # Reusable UI components
│       │   │   ├── agent/        # Agent-related components
│       │   │   │   ├── hub/      # Agent Hub marketplace
│       │   │   │   ├── skills/   # Skills display
│       │   │   │   └── tools/    # Tool configuration
│       │   │   ├── channels/     # Channel management
│       │   │   ├── chat/         # Chat UI components
│       │   │   ├── config/       # Configuration forms
│       │   │   ├── credentials/  # Credential management
│       │   │   ├── logs/         # Log viewer
│       │   │   ├── models/       # Model selector
│       │   │   ├── tour/         # Onboarding tour
│       │   │   └── ui/           # Base UI components (shadcn)
│       │   ├── features/         # Feature modules
│       │   │   └── chat/         # Chat feature (controller, state, protocol)
│       │   ├── hooks/            # React custom hooks
│       │   ├── i18n/             # Internationalization
│       │   │   └── locales/      # Locale JSON files
│       │   ├── lib/              # Utility libraries
│       │   ├── routes/           # TanStack Router routes
│       │   │   ├── agent/        # Agent management page
│       │   │   └── channels/     # Channels management page
│       │   └── store/            # Jotai stores (state management)
│       └── public/               # Static assets
├── workspace/
│   ├── memory/                   # Agent memory templates
│   └── skills/                   # Built-in skill definitions
│       ├── agent-browser/        # Browser automation skill
│       ├── github/               # GitHub integration skill
│       ├── hardware/             # Hardware control skill
│       ├── skill-creator/        # Skill creation helper
│       ├── summarize/            # Conversation summarization
│       ├── tmux/                 # tmux session management
│       └── weather/              # Weather lookup
├── docs/                         # Documentation (multi-language)
│   ├── design/                   # Design documents
│   ├── zh/                       # Chinese docs
│   ├── ja/                       # Japanese docs
│   ├── pt-br/                    # Portuguese (BR) docs
│   └── vi/                       # Vietnamese docs
└── config/                       # Example configuration files
```

## Directory Purposes

**`cmd/`:** CLI binary entry points. Each subdirectory produces a separate binary.

**`pkg/`:** Shared Go packages. All business logic lives here. Packages are designed for reuse across binaries.

**`cmd/picoclaw/internal/`:** CLI subcommand implementations. Thin wrappers around `pkg/` packages with Cobra integration.

**`web/backend/`:** Desktop launcher backend. Embeds frontend assets and provides HTTP API + gateway management.

**`web/frontend/`:** React + TypeScript dashboard. Built with Vite, TanStack Router, shadcn/ui. Output embedded into Go binary.

**`workspace/`:** Runtime workspace. Memory templates, skill definitions, agent-specific data. Copied to `~/.picoclaw/` on first run.

## Key File Locations

**Entry Points:**
- `cmd/picoclaw/main.go`: CLI root (Cobra-based subcommands)
- `web/backend/main.go`: Desktop launcher (HTTP server + system tray)
- `pkg/gateway/gateway.go`: Core gateway runtime (agent loops, channels, services)
- `cmd/picoclaw-launcher-tui/main.go`: Terminal UI launcher

**Configuration:**
- `pkg/config/config.go`: Config loading and environment variable integration
- `pkg/config/config_struct.go`: Config type definitions
- `web/backend/launcherconfig/config.go`: Launcher-specific settings

**Core Logic:**
- `pkg/agent/loop.go`: Main agent event loop
- `pkg/agent/instance.go`: Agent instance with provider, tools, sessions
- `pkg/agent/turn.go`: Turn execution (LLM call + tool loop)
- `pkg/agent/registry.go`: Multi-agent management
- `pkg/bus/bus.go`: Message bus (inbound/outbound/media/audio/voice)
- `pkg/channels/manager.go`: Channel lifecycle and message routing
- `pkg/channels/interfaces.go`: Capability interfaces (streaming, typing, reactions)
- `pkg/tools/registry.go`: Tool registration and execution

**Routing & API:**
- `web/backend/api/router.go`: All API route registration
- `web/frontend/src/routes/`: TanStack Router route definitions
- `web/frontend/src/routeTree.gen.ts`: Auto-generated route tree

**Testing:**
- Co-located `*_test.go` files alongside source files throughout `pkg/` and `cmd/`

## Naming Conventions

**Files:**
- Go: snake_case for test files (`context_budget_test.go`), CamelCase for implementation files (`context_budget.go`)
- TypeScript: kebab-case for components (`channel-config-fields.ts`), camelCase for hooks (`use-chat-models.ts`)
- Routes: kebab-case (`launcher-login.tsx`)

**Directories:**
- Go packages: lowercase, single word where possible (`agent`, `channels`, `tools`)
- Frontend features: kebab-case (`features/chat/`)
- Component groups: kebab-case (`components/agent/hub/`)

**Functions:**
- Go: PascalCase for exported, camelCase for unexported
- Structured logging: `InfoCF`, `ErrorCF`, `DebugCF` (component + field variants)

## Where to Add New Code

**New Messaging Channel:**
- Implementation: `pkg/channels/<channel_name>/` (with `init.go` for self-registration)
- Register: Import with blank identifier in `pkg/gateway/gateway.go`
- Capability interfaces: `pkg/channels/interfaces.go`

**New LLM Provider:**
- Implementation: `pkg/providers/<provider_name>/`
- Factory: Register in `pkg/providers/factory.go`
- Test: `<provider_name>_test.go`

**New Tool:**
- Implementation: `pkg/tools/<tool_name>.go`
- Register: Add to tool list in `pkg/gateway/gateway.go` or agent loop setup

**New Skill:**
- Definition: `workspace/skills/<skill_name>/` (SKILL.md + references)

**New CLI Subcommand:**
- Implementation: `cmd/picoclaw/internal/<subcommand>/`
- Register: Add to `cmd/picoclaw/main.go` command list

**New API Endpoint:**
- Handler: `web/backend/api/<resource>.go`
- Route: Register in `web/backend/api/router.go`

**New Frontend Page:**
- Route: `web/frontend/src/routes/<page_name>.tsx`
- API client: `web/frontend/src/api/<resource>.ts`
- Hook: `web/frontend/src/hooks/use-<feature>.ts`

## Special Directories

**`web/backend/dist/`:**
- Purpose: Compiled frontend assets (embedded into Go binary)
- Generated: Yes (by `npm run build:backend`)
- Committed: Yes (for self-contained Go builds)

**`workspace/`:**
- Purpose: Default skill and memory templates copied to user home on first run
- Generated: No (hand-authored)
- Committed: Yes

**`docs/`:**
- Purpose: Project documentation in multiple languages
- Languages: English (root), zh, ja, pt-br, vi, my
- Committed: Yes

---

*Structure analysis: 2026-04-10*

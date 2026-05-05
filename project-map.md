# Project Map
_Generated: 2026-05-05 | Git: 07107384_

## Directory Structure
cmd/ — CLI entry points (picoclaw main, membench, internal subcommands)
pkg/ — Core library packages (agent, channels, providers, tools, etc.)
web/frontend/ — Frontend UI (React/TypeScript with TanStack)
web/backend/ — Backend API server (Go, dashboard auth, middleware)
workspace/ — Runtime workspace (skills, memory)
docs/ — Documentation (architecture, channels, guides, migration, reference)
docs/reference/tools-api.md — Complete tools API documentation: available tools, data structures, backend API endpoints, MCP integration
docs/reference/exec-tool.md — Exec tool deep dive: how it works, security measures, how to disable/sandbox/remove completely
config/ — Configuration templates and examples
build/ — Build scripts and artifacts
docker/ — Docker containerization files
scripts/ — Automation and utility scripts
examples/ — Example projects (pico-echo-server)
assets/ — Static assets (logo, images)

## Key Files
cmd/picoclaw/main.go — Main CLI entry point using Cobra; registers subcommands (agent, auth, gateway, mcp, migrate, model, skills, etc.)
cmd/picoclaw/internal/ — Internal CLI command implementations (agent, auth, gateway, mcp, migrate, model, skills, status, version, onboard, cron, cliui)
pkg/agent/ — Core agent logic: context management, pipelines (setup/llm/finalize), turn coordination, event handling, hooks, steering, thinking, prompt contributors
pkg/agent/context_manager.go — Manages LLM context lifecycle, caching, and budget enforcement
pkg/agent/pipeline.go — Orchestrates agent execution phases (setup → LLM → tools → finalize)
pkg/channels/ — Multi-platform chat integrations: Discord, Telegram, Slack, WeChat, WeCom, Feishu, DingTalk, IRC, LINE, Matrix, VK, WhatsApp, OneBot, MaixCam, Pico
pkg/providers/ — AI model provider integrations: Anthropic, OpenAI-compatible, Azure, AWS Bedrock, CLI, HTTP API; shared protocol types and OAuth
pkg/config/ — Configuration loading, validation, and environment variable handling
pkg/skills/ — Skills system for extending agent capabilities
pkg/tools/ — Built-in tools: filesystem (fs), hardware interaction, shared utilities, integration tools
pkg/mcp/ — Model Context Protocol (MCP) server implementation for tool/resource exposure
pkg/memory/ — Agent memory management (short-term/long-term, persistence)
pkg/gateway/ — Gateway for routing messages between channels and agents
pkg/auth/ — Authentication and credential management (OAuth, API keys, encryption)
pkg/identity/ — Identity and user/session management
pkg/session/ — Session state management across channels
pkg/state/ — Application state persistence
pkg/credential/ — Secure credential storage (ChaCha20-Poly1305 encryption)
pkg/routing/ — Message routing logic between channels, agents, and models
pkg/bus/ — Internal event bus for decoupled communication
pkg/events/ — Event definitions and handling (device events, system events)
pkg/cron/ — Cron-based scheduling for periodic tasks
pkg/logger/ — Logging infrastructure
pkg/health/ — Health check endpoints and diagnostics
pkg/heartbeat/ — Heartbeat/keepalive mechanism for long-running processes
pkg/updater/ — Self-update functionality (minio/selfupdate)
pkg/migrate/ — Database and config migration utilities
pkg/media/ — Media processing (images, audio)
pkg/audio/asr/ — Automatic Speech Recognition (ASR) providers
pkg/audio/tts/ — Text-to-Speech (TTS) providers
pkg/tokenizer/ — Token counting and management for LLM context budgets
pkg/netbind/ — Network binding utilities for embedded/specific network configs
pkg/fileutil/ — File utility functions
pkg/devices/ — Device management (events, sources) for hardware integrations
pkg/isolation/ — Sandboxing and isolation for security
pkg/seahorse/ — Seahorse integration (encrypted storage)
pkg/constants/ — Package-level constants
web/backend/api/ — Backend API route definitions
web/backend/middleware/ — HTTP middleware (auth, CORS, logging)
web/backend/dashboardauth/ — Dashboard authentication logic
web/backend/model/ — Backend data models
web/backend/launcherconfig/ — Launcher configuration
web/frontend/src/ — Frontend source (components, routes, store, features, hooks, lib, api, i18n)
go.mod — Go 1.25.9 module definition; key deps: Cobra, DiscordGo, Telego, Anthropic SDK, AWS SDK v2, MCP SDK, gRPC, various channel SDKs
go.sum — Dependency checksums
Makefile — Build targets (build, test, lint, release)
.goreleaser.yaml — GoReleaser config for cross-platform releases
.golangci.yaml — GolangCI-Lint configuration
README.md — Project overview: ultra-lightweight AI assistant for $10 hardware, <10MB RAM, inspired by NanoBot
ROADMAP.md — Vision: lightweight, secure, autonomous AI Agent; core optimization, security hardening, protocol-first architecture
CONTRIBUTING.md — Contribution guidelines
LICENSE — MIT License
.env.example — Example environment variables template
.dockerignore / .gitignore — Ignore rules for Docker and Git

## Critical Constraints
- Target: Runs on $10 hardware (e.g., RISC-V SBCs) with <10MB RAM, core process <20MB for 64MB boards
- Go 1.25.9 required (very recent version)
- Self-bootstrapped: AI Agent drove architecture migration and optimization (not a fork)
- Memory optimization takes precedence over storage size
- Security: Prompt injection defense, tool abuse prevention, SSRF protection, filesystem sandbox, context isolation, privacy redaction
- Crypto: Uses ChaCha20-Poly1305 for secret storage (upgrade from older algorithms)
- OAuth 2.0 Flow: Deprecating hardcoded API keys in CLI
- Architecture: Migrating from "Vendor-based" to "Protocol-based" classification (OpenAI-compatible, Ollama-compatible)
- Multi-architecture: x86_64, ARM64, MIPS, RISC-V, LoongArch
- Channel diversity: 14+ chat platforms supported with platform-specific adapters
- Provider diversity: Anthropic, OpenAI-compat, Azure, Bedrock, local (Ollama, vLLM, LM Studio, Mistral)
- Frontend: TypeScript/React with TanStack router/query
- Build: Makefile + GoReleaser for cross-platform binaries
- Workspace: Skills and memory stored in workspace/ directory at runtime

## Hot Files
pkg/agent/agent.go, pkg/agent/pipeline.go, pkg/agent/context_manager.go, pkg/agent/definition.go, pkg/channels/ (multiple files), pkg/providers/ (multiple files), cmd/picoclaw/main.go, pkg/config/, web/backend/api/, web/frontend/src/

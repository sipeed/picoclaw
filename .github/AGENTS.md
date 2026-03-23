# AGENTS.md — AI Agent Instructions for PicoClaw

> Instructions for AI coding agents working on this repository.

## Project Overview

PicoClaw is an ultra-lightweight personal AI assistant written in **Go**. It connects chat **Channels** (Telegram, Discord, WeChat, etc.) to LLM **Providers** (OpenAI, Anthropic, Ollama, etc.) via a central **Message Bus**. Designed to run on $10 hardware with <10MB RAM.

## Architecture

```
cmd/picoclaw/main.go       ← CLI entry (cobra): agent, gateway, onboard
cmd/picoclaw-launcher-tui/  ← TUI launcher
web/backend/                ← Web launcher + control panel

pkg/gateway/    → Orchestrator: initializes bus, channels, agent, cron
pkg/bus/        → Message Bus: InboundMessage ↔ OutboundMessage decoupling
pkg/agent/      → Agent Loop: session history, LLM interaction, tool execution
pkg/channels/   → Channel Manager: chat integrations lifecycle + routing
pkg/providers/  → LLM Providers: OpenAI-compatible, Anthropic, etc.
pkg/tools/      → Tool/Skill definitions (shell, browser, MCP, etc.)
pkg/config/     → Config struct with env tag overrides, JSON merge on defaults
pkg/session/    → Session management and persistence
pkg/mcp/        → Model Context Protocol integration
pkg/routing/    → Smart model routing (simple → cheap, complex → SOTA)

workspace/      → Agent identity files (SOUL.md, USER.md, AGENT.md)
config/         → config.example.json (reference for all options)
docker/         → Dockerfile.heavy, docker-compose.yml, entrypoint.sh
```

## Build & Test

```bash
make build          # Build binary (runs go generate first)
make generate       # Run go generate only
make check          # Full pre-commit: deps + fmt + vet + test
make test           # Run all tests
make fmt            # Format code
make vet            # Static analysis
make lint           # Full linter (golangci-lint)
make build-launcher # Build web launcher (requires pnpm for frontend)
```

Single test: `go test -run TestName -v ./pkg/session/`

## Coding Conventions

### Go Style
- **Go 1.25+**, module path `github.com/sipeed/picoclaw`
- Build tags: `-tags stdjson`; CGO disabled by default (`CGO_ENABLED=0`)
- Error wrapping: `fmt.Errorf("context: %w", err)`
- Logging: `zerolog` via `pkg/logger` — use `log.Debug()`, `log.Info()`, `log.Error()`, etc.
- Testing: `testify/assert` and `testify/require`, co-located `_test.go` files
- Commits: imperative mood, conventional-ish ("Add retry logic", "Fix session leak (#123)")

### Config Pattern
- Central `Config` struct in `pkg/config/config.go` with `env` tags
- `LoadConfig`: starts from `DefaultConfig()`, merges user JSON on top
- Reference: `config/config.example.json`

### Key Patterns
- Channels/providers registered via side-effect imports in `pkg/gateway/`
- Workspace files (`~/.picoclaw/workspace/`) are the agent's persistent home
- `workspace/` in the repo is the template; `docker/data/` is the Docker volume mount

## Docker

The "heavy" image (`docker/Dockerfile.heavy`) is a multi-stage build:
1. Go builder → builds `picoclaw` + `picoclaw-launcher`
2. Node builder → builds web frontend
3. Runtime: `node:24-slim` + Chromium, Python/uv, Xvfb

**`docker/entrypoint.sh`**: runs `onboard` → starts Xvfb → starts Copilot proxy → execs launcher

**Volume**: `docker/data/` mounts to `/root/.picoclaw` in the container. All workspace, skills, memory, and session files the agent uses live here.

⚠️ **When editing files the Docker agent reads, edit under `docker/data/`, NOT `workspace/`** — the latter is the repo template, not what the running container sees.

## Documentation

Existing docs are in `docs/`. Key references:
- [docs/configuration.md](docs/configuration.md) — Config file guide
- [docs/providers.md](docs/providers.md) — LLM provider setup
- [docs/docker.md](docs/docker.md) — Docker deployment
- [docs/tools_configuration.md](docs/tools_configuration.md) — Tool/skill config
- [docs/spawn-tasks.md](docs/spawn-tasks.md) — Sub-agent spawning
- [docs/steering.md](docs/steering.md) — Model steering
- [docs/channels/](docs/channels/) — Per-channel setup guides
- [docs/hooks/](docs/hooks/) — Hook system
- [docs/agent-refactor/](docs/agent-refactor/) — Architecture deep-dives
- [CONTRIBUTING.md](CONTRIBUTING.md) — Contribution workflow, AI disclosure requirements
- [ROADMAP.md](ROADMAP.md) — Feature roadmap and priorities

Link to these docs rather than duplicating their content.

## Security Notes

- File operations are sandboxed to workspace (`restrict_to_workspace: true` by default)
- Never hardcode API keys — use env vars or `config.json` (gitignored)
- The `exec` tool has `custom_allow_patterns` to whitelist commands
- Review AI-generated code for path traversal, injection, SSRF (see CONTRIBUTING.md)
- `.env` and `config.json` are gitignored; `config.example.json` is the reference

## Cross-Platform

PicoClaw builds for: linux/amd64, linux/arm64, linux/arm (ARMv7), linux/mipsle, linux/riscv64, linux/loong64, darwin/arm64, windows/amd64. The Makefile has per-platform targets and a MIPS ELF e_flags patch for NaN2008 kernels.

# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Fork Workflow

This is a fork of `sipeed/picoclaw`. Branch conventions:
- `main` — pure mirror of `upstream/main`, never commit here directly
- `sushi30` — fork-specific commits rebased on top of `main`; all PRs target this branch

PRs go from feature branches → `origin/sushi30`, never → `origin/main`.

## Build Commands

```bash
make deps              # Download Go dependencies
make build             # Build picoclaw for current platform
make build-all         # Build for all target platforms (x86_64, ARM, ARM64, RISC-V, MIPS, LoongArch)
make install           # Build and install to ~/.local/bin
make build-launcher    # Build WebUI launcher
make build-launcher-tui # Build TUI launcher
```

## Testing & Code Quality

```bash
make test              # Run Go tests + frontend linting
make vet               # go vet static analysis
make lint              # golangci-lint
make fmt               # Format code
make check             # vet + fmt + verify dependencies
```

## Docker

```bash
make docker-build      # Minimal Alpine image
make docker-build-full # With Node.js 24 for full MCP support
make docker-run        # Run gateway in Docker
make docker-test       # Test MCP tools in Docker
```

## Web Frontend (in `/web/`)

```bash
make dev               # Start backend + frontend dev servers concurrently
make build             # Build frontend and embed into Go binary
make test              # Run backend + frontend tests
```

Frontend runs at `localhost:5173`, backend at `localhost:18800`.

## Architecture

Three binaries from one repo:

1. **`cmd/picoclaw`** — main agent binary (CLI via Cobra)
2. **`web/`** — WebUI launcher: Go backend (`localhost:18800`) + React/Vite frontend embedded in binary
3. **`cmd/picoclaw-launcher-tui`** — terminal UI launcher (tcell/tview)

### Core packages (`pkg/`)

| Package | Role |
|---------|------|
| `agent/` | Central message-processing loop, tool execution, context budgeting, hooks system |
| `channels/` | 17+ messaging adapters: Telegram, Discord, WhatsApp, WeChat, Slack, Matrix, Email, IRC, etc. |
| `providers/` | 30+ LLM provider implementations: OpenAI, Anthropic, Gemini, DeepSeek, Azure, Bedrock, Ollama, etc. |
| `tools/` | Built-in tools (web search, code execution, file ops) |
| `mcp/` | Model Context Protocol integration (stdio, SSE, HTTP transports) |
| `memory/` | JSONL-based persistent conversation storage |
| `session/` | Conversation session management |
| `gateway/` | HTTP server for webhook-based channel ingestion |
| `routing/` | Smart model routing (routes simple queries to lightweight models) |
| `config/` | JSON config management with security isolation |
| `auth/` | OAuth flows and credential management |
| `skills/` | Modular capability extensions (Markdown + embedded code) |
| `cron/` | Scheduled tasks and reminders |

### How channels, providers, and the agent connect

Each **channel** (e.g. Telegram) receives messages and forwards them to the **agent loop** (`pkg/agent/`). The agent loop selects a **provider** (LLM backend) based on routing rules, manages tool calls, and sends responses back through the channel. The **gateway** (`pkg/gateway/`) is a single HTTP server that handles webhooks for all channels simultaneously.

### Configuration

Runtime config lives in `~/.picoclaw/workspace/config.json` (template: `config/config.example.json`). API keys can be injected via environment variables or stored encrypted in `.security.yml`. The `env://` scheme passes env var names directly as key values without encryption.

### Fork-specific testing conventions

New functionality added in the `sushi30` fork **must be tested in isolated test files** named `*_sushi30_test.go`. This prevents upstream rebases from silently overwriting tests and makes fork additions easy to identify.

Example: `pkg/config/resolve_key_sushi30_test.go` guards the `env://` fix in `resolveKey()`.

### Frontend stack

React 19 + Vite + TanStack Router + Tailwind CSS 4 + Radix UI + Jotai (state). Build output is embedded into the Go web backend binary via `go:embed`.

### Build notes

- All binaries use `CGO_ENABLED=0` for static linking
- Build tags `goolm,stdjson` enable optional MCP support
- Version/commit/timestamp are injected via `ldflags` at build time
- Target architectures: x86_64, ARM64, ARM32, MIPS, RISC-V, LoongArch

# Technology Stack

**Analysis Date:** 2026-04-10

## Languages

**Primary:**
- **Go 1.25.9** (per `go.mod` line 3) - Backend runtime, all core logic
- **TypeScript ~5.9.3** - Web frontend (`web/frontend/`)

**Secondary:**
- **JavaScript/Node.js 24** - Full Docker image runtime for MCP tools (`docker/Dockerfile.full`)

## Runtime

**Environment:**
- Go 1.25.9 (compiled binary, CGO_ENABLED=0 by default; CGO_ENABLED=1 on macOS for systray)
- Node.js ^20.19.0 || ^22.13.0 || >=24 (frontend via `web/frontend/package.json` engines field)

**Package Managers:**
- **Go modules** - Lockfile: `go.sum` present
- **pnpm** - Lockfile: `web/frontend/pnpm-lock.yaml` present (inferred from pnpm-lock.yaml)

## Frameworks

**Core (Go):**
- **cobra v1.10.2** (`cmd/picoclaw/main.go`) - CLI framework, root command with subcommands: `onboard`, `agent`, `auth`, `gateway`, `status`, `cron`, `migrate`, `skills`, `model`, `version`, `update`
- **modelcontextprotocol/go-sdk v1.5.0** (`pkg/mcp/manager.go`) - MCP (Model Context Protocol) client for external tool servers
- **zerolog v1.35.0** (`pkg/logger/`) - Structured logging
- **modernc.org/sqlite v1.48.2** - SQLite database (pure Go, no CGO)

**Frontend (web/frontend/):**
- **React 19.2.5** - UI framework
- **Vite 8.0.8** - Build tool and dev server
- **TailwindCSS 4.2.2** - Styling with `@tailwindcss/vite` plugin
- **Radix UI 1.4.3** - Headless component primitives
- **TanStack Router 1.167.0 + React Query 5.97.0** - Routing and data fetching
- **Jotai 2.19.1** - Atomic state management
- **i18next 26.0.3** - Internationalization (en/zh)

**Backend Web (web/backend/):**
- **net/http** (stdlib) - HTTP server for launcher dashboard

**Testing:**
- **testify v1.11.1** - Assertion library and mocking (`github.com/stretchr/testify`)

**Build/Dev:**
- **golangci-lint** - Linting (configured via `.golangci.yaml`)
- **Make** - Build orchestration (`Makefile`, 399 lines)

## Key Dependencies

**AI Model Providers (SDK clients):**
- `anthropic-sdk-go v1.26.0` - Anthropic Claude API
- `openai-go/v3 v3.22.0` - OpenAI API
- `aws-sdk-go-v2 + bedrockruntime v1.50.4` - AWS Bedrock
- `github/copilot-sdk/go v0.2.0` - GitHub Copilot CLI

**Messaging/Chat SDKs (Channels):**
- `slack-go/slack v0.17.3` - Slack
- `bwmarrin/discordgo v0.29.0` (replaced with `yeongaori/discordgo-fork`) - Discord
- `mymmrac/telego v1.8.0` - Telegram
- `larksuite/oapi-sdk-go/v3 v3.5.3` - Feishu/Lark
- `open-dingtalk/dingtalk-stream-sdk-go v0.9.1` - DingTalk
- `tencent-connect/botgo v0.2.1` - QQ
- `ergochat/irc-go v0.6.0` - IRC
- `SevereCloud/vksdk/v3 v3.3.1` - VK
- `atc0005/go-teams-notify/v2 v2.14.0` - Microsoft Teams
- `maunium.net/go/mautrix v0.26.4` - Matrix
- `go.mau.fi/whatsmeow` - WhatsApp (native)
- `gorilla/websocket v1.5.3` - WebSocket (Pico channel)

**Audio (ASR/TTS):**
- `pion/webrtc/v3 v3.3.6` + `pion/rtp v1.10.1` - WebRTC (Discord voice, audio streaming)
- ElevenLabs transcriber (`pkg/audio/asr/elevenlabs_transcriber.go`)
- OpenAI-compatible TTS (`pkg/audio/tts/openai_tts.go`)
- Mimo TTS (`pkg/audio/tts/mimo_tts.go`)

**Web/HTTP:**
- `valyala/fasthttp v1.69.0` - High-performance HTTP (Feishu channel)
- `go-resty/resty/v2 v2.17.1` - HTTP client
- `klauspost/compress v1.18.4` - Compression

**Terminal UI (Launcher TUI):**
- `gdamore/tcell/v2 v2.13.8` - Terminal cell library
- `rivo/tview v0.42.0` - Terminal UI widgets
- `ergochat/readline v0.1.3` - Readline support

**Utilities:**
- `spf13/cobra v1.10.2` - CLI framework
- `caarlos0/env/v11 v11.4.0` - Environment variable parsing with struct tags
- `BurntSushi/toml v1.6.0` - TOML parsing
- `adhocore/gronx v1.19.6` - Cron expression parsing
- `google/uuid v1.6.0` - UUID generation
- `h2non/filetype v1.1.3` - File type detection
- `mdP/qrterminal/v3 v3.2.1` - QR code terminal output
- `minio/selfupdate v0.6.0` - Binary self-updates
- `creack/pty v1.1.24` - PTY allocation (shell tool)
- `gomarkdown/markdown` - Markdown parsing

## Database/Storage

**Primary:**
- **SQLite** via `modernc.org/sqlite v1.48.2` (pure Go, no CGO dependency) - Used by dashboard auth store (`web/backend/dashboardauth/sql.go`)
- **JSONL files** (`pkg/memory/jsonl.go`) - Session memory storage, append-only JSONL format with per-session metadata files
- **JSON files** - Config (`config.json`), auth store (`auth.json`), state (`state/state.json`)

**No external database server required** - all storage is file-based.

## Authentication Mechanisms

1. **API Key auth** - Per-model `api_keys` in config (supports multiple keys for failover, `SecureString` wrapper)
2. **OAuth/PKCE** (`pkg/auth/`) - Anthropic and OpenAI OAuth login flows with token refresh
3. **Platform-specific tokens** - Each channel has its own token/secret (Telegram bot token, Discord token, Feishu app secret, etc.)
4. **Dashboard auth** - Launcher token-based (`PICOCLAW_LAUNCHER_TOKEN` env var) for web console
5. **WeCom/Weixin** - Custom OAuth flows (`pkg/auth/wecom.go`, `pkg/auth/weixin.go`)

## Configuration

**Method:**
- JSON config file (`~/.picoclaw/config.json`)
- Environment variables with `env:` struct tags (e.g., `PICOCLAW_CHANNELS_TELEGRAM_TOKEN`)
- `pkg/config/` handles loading, validation, and secure string masking

**Key directories:**
- `~/.picoclaw/` - Home directory (config, auth, workspace)
- `~/.picoclaw/workspace/` - Workspace (state, skills, session memory)
- `~/.picoclaw/logs/` - Log files

## Build System

**Makefile targets** (key ones):
- `make build` - Build for current platform (runs `go generate` first)
- `make build-all` - Cross-compile for 10+ platforms
- `make build-launcher` - Build web console binary
- `make build-launcher-tui` - Build terminal UI binary
- `make build-whatsapp-native` - Build with native WhatsApp support (larger binary)
- `make build-linux-arm` / `build-linux-arm64` / `build-linux-mipsle` / `build-pi-zero` - Embedded targets
- `make test` - Run all tests
- `make lint` / `make fmt` / `make vet` - Code quality
- `make docker-build` / `docker-build-full` - Docker images
- `make install` / `make uninstall` - Local install to `~/.local/bin`

**Build tags:** `goolm,stdjson` (default); `whatsapp_native` for native WhatsApp

**Supported platforms:** linux/amd64, linux/arm, linux/arm64, linux/loong64, linux/riscv64, linux/mipsle, darwin/arm64, windows/amd64, netbsd/amd64, netbsd/arm64

## Platform Requirements

**Development:**
- Go 1.25.9+
- Node.js 20+ (for frontend builds)
- pnpm (for frontend dependencies)
- golangci-lint (for linting)
- Make

**Production:**
- Single static binary (no runtime dependencies when CGO_ENABLED=0)
- Alpine 3.23 (minimal Docker image)
- Node.js 24 (full Docker image for MCP tool support)

---

*Stack analysis: 2026-04-10*

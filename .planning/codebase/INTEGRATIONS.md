# External Integrations

**Analysis Date:** 2026-04-10

## AI Model Providers

PicoClaw supports a protocol-based model routing system (`pkg/providers/factory_provider.go`). Models are specified as `protocol/model-id` (e.g., `openai/gpt-4o`, `anthropic/claude-sonnet-4.6`). Default protocol is `openai` when no prefix.

**Native protocol providers (non-OpenAI-compatible):**
- **Anthropic** (`pkg/providers/anthropic/provider.go`) - Claude models, OAuth or API key auth
- **Anthropic Messages** (`pkg/providers/anthropic_messages/provider.go`) - Messages API variant
- **Claude CLI** (`pkg/providers/claude_cli_provider.go`) - Claude Code CLI via stdio
- **Codex CLI** (`pkg/providers/codex_cli_provider.go`) - OpenAI Codex CLI via stdio
- **AWS Bedrock** (`pkg/providers/bedrock/provider_bedrock.go`) - AWS SDK v2, IAM or credentials
- **Azure OpenAI** (`pkg/providers/azure/provider.go`) - Azure-specific endpoint handling
- **GitHub Copilot** (`pkg/providers/github_copilot_provider.go`) - Copilot SDK integration
- **Antigravity** (`pkg/providers/antigravity_provider.go`) - Custom provider

**OpenAI-compatible protocols** (`pkg/providers/factory_provider.go`, lines 25-61):
- `openai` (api.openai.com/v1)
- `openrouter` (openrouter.ai/api/v1)
- `groq` (api.groq.com/openai/v1)
- `deepseek` (api.deepseek.com/v1)
- `ollama` (localhost:11434/v1, no API key required)
- `lmstudio` (localhost:1234/v1, no API key required)
- `vllm` (localhost:8000/v1, no API key required)
- `gemini` (generativelanguage.googleapis.com/v1beta)
- `litellm` (localhost:4000/v1)
- `qwen` / `qwen-intl` / `qwen-us` (Dashscope, Alibaba Cloud)
- `moonshot`, `mistral`, `cerebras`, `nvidia`, `volcengine`, `modelscope`, `novita`, `minimax`, `longcat`, `avian`, `zhipu`, `venice`, `vivgrid`
- Coding-specific: `coding-plan`, `alibaba-coding`, `coding-plan-anthropic`

## Messaging Channels (Chat Platforms)

Registered via blank imports in `pkg/gateway/gateway.go` (lines 21-37). Each channel lives in `pkg/channels/<name>/`.

| Channel | Package | SDK | Auth | Protocol |
|---------|---------|-----|------|----------|
| Telegram | `pkg/channels/telegram/` | telego v1.8.0 | Bot token | Polling |
| Discord | `pkg/channels/discord/` | discordgo (fork) v0.29.0 | Bot token | WebSocket |
| Feishu/Lark | `pkg/channels/feishu/` | oapi-sdk-go v3.5.3 | App ID + secret | Webhook |
| Slack | `pkg/channels/slack/` | slack-go v0.17.3 | Bot token + App token | WebSocket (Socket Mode) |
| DingTalk | `pkg/channels/dingtalk/` | dingtalk-stream-sdk v0.9.1 | Client ID + secret | Stream |
| QQ | `pkg/channels/qq/` | botgo v0.2.1 | App ID + secret | WebSocket |
| WhatsApp | `pkg/channels/whatsapp/` | Custom bridge | Bridge URL | HTTP bridge |
| WhatsApp Native | `pkg/channels/whatsapp_native/` | whatsmeow | Direct login | WebSocket |
| WeCom | `pkg/channels/wecom/` | Custom | Bot ID + secret | WebSocket |
| Weixin | `pkg/channels/weixin/` | Custom | Token | HTTP webhook |
| Matrix | `pkg/channels/matrix/` | mautrix v0.26.4 | Homeserver + access token | Matrix API |
| IRC | `pkg/channels/irc/` | irc-go v0.6.0 | Nick/password/SASL | IRC |
| LINE | `pkg/channels/line/` | REST client | Channel secret + token | Webhook |
| OneBot | `pkg/channels/onebot/` | WebSocket client | Access token | WebSocket |
| VK | `pkg/channels/vk/` | vksdk v3.3.1 | Group token | Long polling |
| Teams | `pkg/channels/teams_webhook/` | go-teams-notify v2.14.0 | Webhook URL | HTTP POST |
| MaixCam | `pkg/channels/maixcam/` | HTTP client | None | HTTP |
| Pico | `pkg/channels/pico/` | Custom WebSocket | Token | WebSocket (built-in web channel) |
| PicoClient | `pkg/channels/pico/` | Custom WebSocket client | Token | WebSocket (connect to remote Pico) |

## MCP (Model Context Protocol) Integration

**Manager:** `pkg/mcp/manager.go`

- Uses `github.com/modelcontextprotocol/go-sdk v1.5.0`
- Supports **stdio** transport (spawn external process) and **Streamable HTTP** transport
- Reads env files for MCP server configuration (`loadEnvFile()` line 49)
- Custom HTTP headers supported via `headerTransport` wrapper
- MCP tools are wrapped as native PicoClaw tools via `pkg/tools/mcp_tool.go`
- Isolated command transport for sandboxed MCP servers (`pkg/mcp/isolated_command_transport.go`)

**MCP server configuration** is defined in `ToolsConfig.MCPServers` with fields: `Command`, `Args`, `Env`, `URL`, `Headers`, `EnvFile`.

## Built-in Tools

Defined in `pkg/tools/`:

- **Shell execution** (`shell.go`) - Run shell commands with PTY support, timeout, allow/deny patterns
- **File operations** (`filesystem.go`) - Read, write, edit, list files (workspace-restricted)
- **Edit** (`edit.go`) - Search-and-replace file editing
- **Web search** (`search_tool.go`) - Multiple backends: Brave, Tavily, DuckDuckGo, Perplexity, SearXNG, GLM Search, Baidu Search
- **Web fetch** (`web.go`) - Fetch and extract web page content
- **Subagent** (`subagent.go`) - Spawn sub-agents for delegated tasks
- **Cron** (`cron.go`) - Schedule and manage cron jobs
- **Message tools** (`message.go`) - Send messages, reactions
- **File send** (`send_file.go`) - Send files to chat
- **Image loading** (`load_image.go`) - Load and process images
- **SPI/I2C** (`spi.go`, `i2c.go`) - Hardware bus access (Linux-only, for embedded devices)
- **Skills** (`skills_install.go`, `skills_search.go`) - Install and search skills marketplace
- **TTS send** (`tts_send.go`) - Text-to-speech output
- **Spawn** (`spawn.go`) - Process spawning with status tracking

## Audio Integrations (ASR/TTS)

**Speech-to-Text (ASR)** (`pkg/audio/asr/`):
- **OpenAI-compatible Whisper** (`whisper_transcriber.go`) - Via any OpenAI-compatible provider
- **ElevenLabs** (`elevenlabs_transcriber.go`) - ElevenLabs STT API
- **Audio model transcriber** (`audio_model_transcriber.go`) - Generic audio-capable LLM

**Text-to-Speech (TTS)** (`pkg/audio/tts/`):
- **OpenAI-compatible TTS** (`openai_tts.go`) - Any OpenAI-compatible endpoint
- **Mimo TTS** (`mimo_tts.go`) - Xiaomi Mimo TTS service

**Audio codecs:**
- OGG Opus encoding (`pkg/audio/ogg.go`)
- Sentence segmentation (`pkg/audio/sentence.go`)

## Third-Party Library Integrations

**Data processing:**
- `tidwall/gjson/sjson/pretty/match` - JSON manipulation
- `segmentio/encoding` - High-performance JSON
- `bytedance/sonic` - Fast JSON serialization
- `vmihailenco/msgpack/v5` - MessagePack serialization

**Security:**
- `cloudflare/circl` - Cryptographic primitives
- `aead.dev/minisign` - Minisign signature verification (for release updates)
- `filippo.io/edwards25519` - Ed25519 curves

**Observability:**
- `opentelemetry.io/otel` (v1.35.0) - OpenTelemetry tracing (auto-instrumentation)
- `zerolog` - Structured logging with sensitive data filtering

## Docker/Containerization

**Images:**
- `docker/Dockerfile` - Minimal Alpine image (`alpine:3.23`), Go 1.25 builder, health check on `:18790/health`
- `docker/Dockerfile.full` - Full image with Node.js 24 + `uv`/`uvx` (Python) for MCP tool support
- `docker/Dockerfile.goreleaser` - Release build variant
- `docker/Dockerfile.goreleaser.launcher` - Launcher-specific variant
- `docker/Dockerfile.heavy` - Heavy variant
- `docker/entrypoint.sh` - First-run entrypoint

**Docker Compose** (`docker/docker-compose.yml`):
- `picoclaw-agent` - One-shot agent mode (profile: `agent`)
- `picoclaw-gateway` - Long-running bot (profile: `gateway`)
- `picoclaw-launcher` - Web console (profile: `launcher`, ports 18800/18790)

**Docker Compose Full** (`docker/docker-compose.full.yml`):
- Same services but with full MCP tool support (Node.js runtime)

**Image registry:** `docker.io/sipeed/picoclaw:latest` / `:full` / `:launcher`

## CI/CD Configuration

**GitHub Actions** (`.github/workflows/`):

| Workflow | File | Purpose |
|----------|------|---------|
| Build | `build.yml` | On push to `main` - `make build-all` on ubuntu-latest |
| PR | `pr.yml` | On pull requests - build + test |
| Release | `release.yml` | On tags - multi-platform release artifacts |
| Nightly | `nightly.yml` | Scheduled nightly builds |
| Docker | `docker-build.yml` | Docker image build and push |
| macOS DMG | `create_dmg.yml` | macOS application bundle |
| TOS upload | `upload-tos.yml` | Terms of service upload |

**Dependabot** (`.github/dependabot.yml`) - Automated dependency updates

## Plugin/Extension Systems

**Skills** - Marketplace-style extension system:
- Skills installed to `~/.picoclaw/workspace/skills/`
- Built-in skills in `skills/` directory at project root
- CLI commands: `picoclaw skills install`, `skills list`, `skills search`, `skills remove`, `skills list-builtin`, `skills show`
- Tools: `pkg/tools/skills_install.go`, `pkg/tools/skills_search.go`
- Skills package: `pkg/skills/`

**MCP Servers** - External tool providers:
- Configured via `config.json` under `tools.mcp_servers`
- Spawned as child processes (stdio) or connected via HTTP (Streamable HTTP)
- Tools auto-discovered and registered as native tools

**Process Hooks** (`HooksConfig` in `pkg/config/config.go`):
- Observer/interceptor hooks on agent events
- Built-in hooks (e.g., content moderation)
- Custom process hooks (spawn external process, observe/intercept events)

**Channel registry** (`pkg/channels/registry.go`):
- Channels auto-register via `init()` functions in their packages
- New channels added by creating a package under `pkg/channels/<name>/` with an `init.go` that calls `Register()`

**Provider factory** (`pkg/providers/factory_provider.go`):
- New OpenAI-compatible providers added by adding a protocol entry to `protocolMetaByName` map
- No code change needed for most new providers - just config

## Environment Variables

**Critical configuration:**
- `PICOCLAW_CHANNELS_<NAME>_TOKEN` - Channel authentication tokens
- `PICOCLAW_AGENTS_DEFAULTS_PROVIDER` / `MODEL_NAME` - Default model
- `PICOCLAW_TOOLS_WEB_BRAVE_API_KEYS` / `TAVILY_API_KEYS` - Search API keys
- `PICOCLAW_HOME` - Override home directory (default: `~/.picoclaw`)
- `TZ` / `ZONEINFO` - Timezone configuration

**Secrets:** Stored in `~/.picoclaw/config.json` (with `SecureString` masking) and `~/.picoclaw/auth.json` (OAuth tokens). Neither file should be committed.

## Webhooks & Callbacks

**Incoming:**
- Feishu webhook endpoint (channel-specific)
- LINE webhook (`webhook_host:webhook_port/webhook_path`)
- Weixin webhook
- LINE webhook for callback events
- Teams webhook (output-only, no incoming)

**Outgoing:**
- Teams webhook notifications (`pkg/channels/teams_webhook/`)
- Web search API calls (Brave, Tavily, Perplexity, etc.)

---

*Integration audit: 2026-04-10*

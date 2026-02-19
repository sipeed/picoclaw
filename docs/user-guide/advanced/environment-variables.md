# Environment Variables Reference

All PicoClaw configuration options can be overridden with environment variables. This enables configuration management through environment files, container orchestration, and CI/CD pipelines.

## Naming Convention

Environment variables follow the pattern:

```
PICOCLAW_<SECTION>_<SUBSECTION>_<KEY>
```

The path corresponds to the JSON configuration structure:

```json
{
  "section": {
    "subsection": {
      "key": "value"
    }
  }
}
```

Becomes: `PICOCLAW_SECTION_SUBSECTION_KEY`

## Agent Configuration

### Defaults

| Variable | Config Path | Description |
|----------|-------------|-------------|
| `PICOCLAW_AGENTS_DEFAULTS_WORKSPACE` | `agents.defaults.workspace` | Working directory |
| `PICOCLAW_AGENTS_DEFAULTS_RESTRICT_TO_WORKSPACE` | `agents.defaults.restrict_to_workspace` | Enable sandbox |
| `PICOCLAW_AGENTS_DEFAULTS_PROVIDER` | `agents.defaults.provider` | Default provider |
| `PICOCLAW_AGENTS_DEFAULTS_MODEL` | `agents.defaults.model` | Primary model |
| `PICOCLAW_AGENTS_DEFAULTS_MODEL_FALLBACKS` | `agents.defaults.model_fallbacks` | Fallback models (JSON array) |
| `PICOCLAW_AGENTS_DEFAULTS_IMAGE_MODEL` | `agents.defaults.image_model` | Image model |
| `PICOCLAW_AGENTS_DEFAULTS_IMAGE_MODEL_FALLBACKS` | `agents.defaults.image_model_fallbacks` | Image fallbacks |
| `PICOCLAW_AGENTS_DEFAULTS_MAX_TOKENS` | `agents.defaults.max_tokens` | Max response tokens |
| `PICOCLAW_AGENTS_DEFAULTS_TEMPERATURE` | `agents.defaults.temperature` | Response randomness |
| `PICOCLAW_AGENTS_DEFAULTS_MAX_TOOL_ITERATIONS` | `agents.defaults.max_tool_iterations` | Max tool loops |

### Examples

```bash
# Set primary model
export PICOCLAW_AGENTS_DEFAULTS_MODEL="anthropic/claude-opus-4-5"

# Set fallback models (JSON array)
export PICOCLAW_AGENTS_DEFAULTS_MODEL_FALLBACKS='["gpt-4o", "glm-4.7"]'

# Disable workspace restriction
export PICOCLAW_AGENTS_DEFAULTS_RESTRICT_TO_WORKSPACE="false"
```

## Provider Configuration

### Common Variables

| Variable Pattern | Description |
|------------------|-------------|
| `PICOCLAW_PROVIDERS_<NAME>_API_KEY` | API key for provider |
| `PICOCLAW_PROVIDERS_<NAME>_API_BASE` | API base URL |
| `PICOCLAW_PROVIDERS_<NAME>_PROXY` | Proxy URL |
| `PICOCLAW_PROVIDERS_<NAME>_AUTH_METHOD` | Auth method |

### Provider Names

Replace `<NAME>` with: `ANTHROPIC`, `OPENAI`, `OPENROUTER`, `GROQ`, `ZHIPU`, `GEMINI`, `OLLAMA`, `VLLM`, `NVIDIA`, `MOONSHOT`, `DEEPSEEK`, `GITHUB_COPILOT`

### Examples

```bash
# OpenRouter
export PICOCLAW_PROVIDERS_OPENROUTER_API_KEY="sk-or-v1-xxx"
export PICOCLAW_PROVIDERS_OPENROUTER_API_BASE="https://openrouter.ai/api/v1"

# Anthropic
export PICOCLAW_PROVIDERS_ANTHROPIC_API_KEY="sk-ant-xxx"

# Groq
export PICOCLAW_PROVIDERS_GROQ_API_KEY="gsk_xxx"

# OpenAI
export PICOCLAW_PROVIDERS_OPENAI_API_KEY="sk-xxx"
export PICOCLAW_PROVIDERS_OPENAI_WEB_SEARCH="true"

# Ollama (local)
export PICOCLAW_PROVIDERS_OLLAMA_API_BASE="http://localhost:11434/v1"

# vLLM (local)
export PICOCLAW_PROVIDERS_VLLM_API_BASE="http://localhost:8000/v1"
```

## Channel Configuration

### Telegram

| Variable | Description |
|----------|-------------|
| `PICOCLAW_CHANNELS_TELEGRAM_ENABLED` | Enable Telegram |
| `PICOCLAW_CHANNELS_TELEGRAM_TOKEN` | Bot token |
| `PICOCLAW_CHANNELS_TELEGRAM_PROXY` | Proxy URL |
| `PICOCLAW_CHANNELS_TELEGRAM_ALLOW_FROM` | Allowed user IDs (JSON array) |

### Discord

| Variable | Description |
|----------|-------------|
| `PICOCLAW_CHANNELS_DISCORD_ENABLED` | Enable Discord |
| `PICOCLAW_CHANNELS_DISCORD_TOKEN` | Bot token |
| `PICOCLAW_CHANNELS_DISCORD_ALLOW_FROM` | Allowed user IDs |

### Slack

| Variable | Description |
|----------|-------------|
| `PICOCLAW_CHANNELS_SLACK_ENABLED` | Enable Slack |
| `PICOCLAW_CHANNELS_SLACK_BOT_TOKEN` | Bot token (xoxb-...) |
| `PICOCLAW_CHANNELS_SLACK_APP_TOKEN` | App token (xapp-...) |
| `PICOCLAW_CHANNELS_SLACK_ALLOW_FROM` | Allowed user IDs |

### Other Channels

| Variable | Description |
|----------|-------------|
| `PICOCLAW_CHANNELS_WHATSAPP_ENABLED` | Enable WhatsApp |
| `PICOCLAW_CHANNELS_WHATSAPP_BRIDGE_URL` | Bridge WebSocket URL |
| `PICOCLAW_CHANNELS_FEISHU_ENABLED` | Enable Feishu |
| `PICOCLAW_CHANNELS_FEISHU_APP_ID` | Feishu App ID |
| `PICOCLAW_CHANNELS_FEISHU_APP_SECRET` | Feishu App Secret |
| `PICOCLAW_CHANNELS_LINE_ENABLED` | Enable LINE |
| `PICOCLAW_CHANNELS_LINE_CHANNEL_SECRET` | LINE Channel Secret |
| `PICOCLAW_CHANNELS_LINE_CHANNEL_ACCESS_TOKEN` | LINE Access Token |
| `PICOCLAW_CHANNELS_DINGTALK_ENABLED` | Enable DingTalk |
| `PICOCLAW_CHANNELS_DINGTALK_CLIENT_ID` | DingTalk Client ID |
| `PICOCLAW_CHANNELS_QQ_ENABLED` | Enable QQ |
| `PICOCLAW_CHANNELS_ONEBOT_ENABLED` | Enable OneBot |
| `PICOCLAW_CHANNELS_ONEBOT_WS_URL` | OneBot WebSocket URL |
| `PICOCLAW_CHANNELS_MAIXCAM_ENABLED` | Enable MaixCam |

### Examples

```bash
# Telegram
export PICOCLAW_CHANNELS_TELEGRAM_ENABLED="true"
export PICOCLAW_CHANNELS_TELEGRAM_TOKEN="123456:ABC..."
export PICOCLAW_CHANNELS_TELEGRAM_ALLOW_FROM='["123456789"]'

# Discord
export PICOCLAW_CHANNELS_DISCORD_ENABLED="true"
export PICOCLAW_CHANNELS_DISCORD_TOKEN="MTk4NjIyNDgzNDc..."

# Slack
export PICOCLAW_CHANNELS_SLACK_ENABLED="true"
export PICOCLAW_CHANNELS_SLACK_BOT_TOKEN="xoxb-xxx"
export PICOCLAW_CHANNELS_SLACK_APP_TOKEN="xapp-xxx"
```

## Tool Configuration

### Web Tools

| Variable | Description |
|----------|-------------|
| `PICOCLAW_TOOLS_WEB_BRAVE_ENABLED` | Enable Brave Search |
| `PICOCLAW_TOOLS_WEB_BRAVE_API_KEY` | Brave API key |
| `PICOCLAW_TOOLS_WEB_BRAVE_MAX_RESULTS` | Max results |
| `PICOCLAW_TOOLS_WEB_DUCKDUCKGO_ENABLED` | Enable DuckDuckGo |
| `PICOCLAW_TOOLS_WEB_DUCKDUCKGO_MAX_RESULTS` | Max results |
| `PICOCLAW_TOOLS_WEB_PERPLEXITY_ENABLED` | Enable Perplexity |
| `PICOCLAW_TOOLS_WEB_PERPLEXITY_API_KEY` | Perplexity API key |
| `PICOCLAW_TOOLS_WEB_PERPLEXITY_MAX_RESULTS` | Max results |

### Exec Tools

| Variable | Description |
|----------|-------------|
| `PICOCLAW_TOOLS_EXEC_ENABLE_DENY_PATTERNS` | Enable command filtering |
| `PICOCLAW_TOOLS_EXEC_CUSTOM_DENY_PATTERNS` | Custom blocked patterns |

### Cron Tools

| Variable | Description |
|----------|-------------|
| `PICOCLAW_TOOLS_CRON_EXEC_TIMEOUT_MINUTES` | Cron job timeout |

### Examples

```bash
# Enable Brave Search
export PICOCLAW_TOOLS_WEB_BRAVE_ENABLED="true"
export PICOCLAW_TOOLS_WEB_BRAVE_API_KEY="BSA..."
export PICOCLAW_TOOLS_WEB_BRAVE_MAX_RESULTS="5"

# Disable DuckDuckGo
export PICOCLAW_TOOLS_WEB_DUCKDUCKGO_ENABLED="false"

# Disable command filtering (not recommended)
export PICOCLAW_TOOLS_EXEC_ENABLE_DENY_PATTERNS="false"
```

## Gateway Configuration

| Variable | Description |
|----------|-------------|
| `PICOCLAW_GATEWAY_HOST` | Server host |
| `PICOCLAW_GATEWAY_PORT` | Server port |

### Example

```bash
export PICOCLAW_GATEWAY_HOST="0.0.0.0"
export PICOCLAW_GATEWAY_PORT="18790"
```

## Heartbeat Configuration

| Variable | Description |
|----------|-------------|
| `PICOCLAW_HEARTBEAT_ENABLED` | Enable periodic tasks |
| `PICOCLAW_HEARTBEAT_INTERVAL` | Check interval (minutes) |

### Example

```bash
export PICOCLAW_HEARTBEAT_ENABLED="true"
export PICOCLAW_HEARTBEAT_INTERVAL="30"
```

## Devices Configuration

| Variable | Description |
|----------|-------------|
| `PICOCLAW_DEVICES_ENABLED` | Enable device monitoring |
| `PICOCLAW_DEVICES_MONITOR_USB` | Monitor USB devices |

## Docker Example

Complete Docker environment configuration:

```dockerfile
# Dockerfile
FROM alpine:latest

ENV PICOCLAW_AGENTS_DEFAULTS_MODEL="anthropic/claude-opus-4-5"
ENV PICOCLAW_AGENTS_DEFAULTS_RESTRICT_TO_WORKSPACE="true"
ENV PICOCLAW_PROVIDERS_OPENROUTER_API_KEY="sk-or-v1-xxx"
ENV PICOCLAW_CHANNELS_TELEGRAM_ENABLED="true"
ENV PICOCLAW_CHANNELS_TELEGRAM_TOKEN="123456:ABC..."
ENV PICOCLAW_HEARTBEAT_ENABLED="false"

COPY picoclaw /usr/local/bin/
CMD ["picoclaw", "gateway"]
```

## Docker Compose Example

```yaml
version: '3'
services:
  picoclaw:
    image: picoclaw:latest
    environment:
      - PICOCLAW_AGENTS_DEFAULTS_MODEL=anthropic/claude-opus-4-5
      - PICOCLAW_PROVIDERS_OPENROUTER_API_KEY=${OPENROUTER_API_KEY}
      - PICOCLAW_PROVIDERS_GROQ_API_KEY=${GROQ_API_KEY}
      - PICOCLAW_CHANNELS_TELEGRAM_ENABLED=true
      - PICOCLAW_CHANNELS_TELEGRAM_TOKEN=${TELEGRAM_TOKEN}
      - PICOCLAW_CHANNELS_DISCORD_ENABLED=true
      - PICOCLAW_CHANNELS_DISCORD_TOKEN=${DISCORD_TOKEN}
    volumes:
      - picoclaw-data:/root/.picoclaw
    restart: unless-stopped

volumes:
  picoclaw-data:
```

## Kubernetes Example

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: picoclaw-secrets
type: Opaque
stringData:
  openrouter-api-key: "sk-or-v1-xxx"
  telegram-token: "123456:ABC..."
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: picoclaw-config
data:
  PICOCLAW_AGENTS_DEFAULTS_MODEL: "anthropic/claude-opus-4-5"
  PICOCLAW_CHANNELS_TELEGRAM_ENABLED: "true"
  PICOCLAW_HEARTBEAT_ENABLED: "true"
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: picoclaw
spec:
  template:
    spec:
      containers:
        - name: picoclaw
          image: picoclaw:latest
          envFrom:
            - configMapRef:
                name: picoclaw-config
          env:
            - name: PICOCLAW_PROVIDERS_OPENROUTER_API_KEY
              valueFrom:
                secretKeyRef:
                  name: picoclaw-secrets
                  key: openrouter-api-key
            - name: PICOCLAW_CHANNELS_TELEGRAM_TOKEN
              valueFrom:
                secretKeyRef:
                  name: picoclaw-secrets
                  key: telegram-token
```

## .env File Example

Create a `.env` file for local development:

```bash
# .env
PICOCLAW_AGENTS_DEFAULTS_MODEL=anthropic/claude-opus-4-5
PICOCLAW_AGENTS_DEFAULTS_MODEL_FALLBACKS='["gpt-4o", "glm-4.7"]'

# Providers
PICOCLAW_PROVIDERS_OPENROUTER_API_KEY=sk-or-v1-xxx
PICOCLAW_PROVIDERS_GROQ_API_KEY=gsk_xxx
PICOCLAW_PROVIDERS_ANTHROPIC_API_KEY=sk-ant-xxx

# Channels
PICOCLAW_CHANNELS_TELEGRAM_ENABLED=true
PICOCLAW_CHANNELS_TELEGRAM_TOKEN=123456:ABC...
PICOCLAW_CHANNELS_DISCORD_ENABLED=false

# Tools
PICOCLAW_TOOLS_WEB_DUCKDUCKGO_ENABLED=true
PICOCLAW_TOOLS_WEB_BRAVE_ENABLED=false
```

Load with:

```bash
source .env
picoclaw gateway
```

## Related Topics

- [Configuration File Reference](../../configuration/config-file.md) - JSON configuration
- [Security Sandbox](security-sandbox.md) - Security settings
- [Providers](../providers/README.md) - Provider configuration

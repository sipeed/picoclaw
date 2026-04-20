# 🐳 Docker & Quick Start Guide

> Back to [README](../README.md)

## 🐳 Docker Compose

You can also run PicoClaw using Docker Compose without installing anything locally.

```bash
# 1. Clone this repo
git clone https://github.com/sipeed/picoclaw.git
cd picoclaw

# 2. First run — auto-generates docker/data/config.json then exits
#    (only triggers when both config.json and workspace/ are missing)
docker compose -f docker/docker-compose.yml --profile gateway up
# The container prints "First-run setup complete." and stops.

# 3. Set your API keys
vim docker/data/config.json   # Set provider API keys, bot tokens, etc.

# 4. Start
docker compose -f docker/docker-compose.yml --profile gateway up -d
```

> [!TIP]
> **Docker Users**: By default, the Gateway listens on `127.0.0.1` which is not accessible from the host. If you need to access the health endpoints or expose ports, set `PICOCLAW_GATEWAY_HOST=0.0.0.0` in your environment or update `config.json`.

> [!NOTE]
> The `gateway` profile only serves the webhook handlers (including Pico when enabled) and health endpoints on the gateway port, so it does not expose generic REST chat endpoints such as `/chat` or `/a2a`. Launcher mode adds the browser UI plus `/api/pico/info` and an authenticated `/pico/ws` proxy on the launcher port, but `/pico/ws` is also available directly on the gateway whenever the Pico channel is enabled.

```bash
# 5. Check logs
docker compose -f docker/docker-compose.yml logs -f picoclaw-gateway

# 6. Stop
docker compose -f docker/docker-compose.yml --profile gateway down
```

### Launcher Mode (Web Console)

The `launcher` image includes all three binaries (`picoclaw`, `picoclaw-launcher`, `picoclaw-launcher-tui`) and starts the web console by default, which provides a browser-based UI for configuration and chat.

```bash
docker compose -f docker/docker-compose.yml --profile launcher up -d
```

Open http://localhost:18800 in your browser. The launcher manages the gateway process automatically.

> [!WARNING]
> The web console uses a dashboard token (in-memory per run unless `PICOCLAW_LAUNCHER_TOKEN` is set). **Do not** expose the launcher to untrusted networks or the public internet. See [Web launcher dashboard](configuration.md#web-launcher-dashboard) in the Configuration Guide.

### Agent Mode (One-shot)

```bash
# Ask a question
docker compose -f docker/docker-compose.yml run --rm picoclaw-agent -m "What is 2+2?"

# Interactive mode
docker compose -f docker/docker-compose.yml run --rm picoclaw-agent
```

### Update

```bash
docker compose -f docker/docker-compose.yml pull
docker compose -f docker/docker-compose.yml --profile gateway up -d
```

### 🚀 Quick Start

> [!TIP]
> Set your API Key in `~/.picoclaw/config.json`. Get API Keys: [Volcengine (CodingPlan)](https://www.volcengine.com/activity/codingplan?utm_campaign=PicoClaw&utm_content=PicoClaw&utm_medium=devrel&utm_source=OWO&utm_term=PicoClaw) (LLM) · [OpenRouter](https://openrouter.ai/keys) (LLM) · [Zhipu](https://open.bigmodel.cn/usercenter/proj-mgmt/apikeys) (LLM). Web search is optional — get a free [Tavily API](https://tavily.com) (1000 free queries/month) or [Brave Search API](https://brave.com/search/api) (2000 free queries/month).

**1. Initialize**

```bash
picoclaw onboard
```

**2. Configure** (`~/.picoclaw/config.json`)

```json
{
  "agents": {
    "defaults": {
      "workspace": "~/.picoclaw/workspace",
      "model_name": "gpt-5.4",
      "max_tokens": 8192,
      "temperature": 0.7,
      "max_tool_iterations": 20
    }
  },
  "model_list": [
    {
      "model_name": "ark-code-latest",
      "model": "volcengine/ark-code-latest",
      "api_keys": ["sk-your-api-key"],
      "api_base":"https://ark.cn-beijing.volces.com/api/coding/v3"
    },
    {
      "model_name": "gpt-5.4",
      "model": "openai/gpt-5.4",
      "api_keys": ["your-api-key"],
      "request_timeout": 300
    },
    {
      "model_name": "claude-sonnet-4.6",
      "model": "anthropic/claude-sonnet-4.6",
      "api_keys": ["your-anthropic-key"]
    }
  ],
  "tools": {
    "web": {
      "enabled": true,
      "fetch_limit_bytes": 10485760,
      "format": "plaintext",
      "brave": {
        "enabled": false,
        "api_key": "YOUR_BRAVE_API_KEY",
        "max_results": 5
      },
      "tavily": {
        "enabled": false,
        "api_key": "YOUR_TAVILY_API_KEY",
        "max_results": 5
      },
      "duckduckgo": {
        "enabled": true,
        "max_results": 5
      },
      "perplexity": {
        "enabled": false,
        "api_key": "YOUR_PERPLEXITY_API_KEY",
        "max_results": 5
      },
      "searxng": {
        "enabled": false,
        "base_url": "http://your-searxng-instance:8888",
        "max_results": 5
      }
    }
  }
}
```

> **New**: The `model_list` configuration format allows zero-code provider addition. See [Model Configuration](#model-configuration-model_list) for details.
> `request_timeout` is optional and uses seconds. If omitted or set to `<= 0`, PicoClaw uses the default timeout (120s).

**3. Get API Keys**

* **LLM Provider**: [OpenRouter](https://openrouter.ai/keys) · [Zhipu](https://open.bigmodel.cn/usercenter/proj-mgmt/apikeys) · [Anthropic](https://console.anthropic.com) · [OpenAI](https://platform.openai.com) · [Gemini](https://aistudio.google.com/api-keys)
* **Web Search** (optional):
  * [Brave Search](https://brave.com/search/api) - Paid ($5/1000 queries, ~$5-6/month)
  * [Perplexity](https://www.perplexity.ai) - AI-powered search with chat interface
  * [SearXNG](https://github.com/searxng/searxng) - Self-hosted metasearch engine (free, no API key needed)
  * [Tavily](https://tavily.com) - Optimized for AI Agents (1000 requests/month)
  * DuckDuckGo - Built-in fallback (no API key required)

> **Note**: See `config.example.json` for a complete configuration template.

**4. Chat**

```bash
picoclaw agent -m "What is 2+2?"
```

That's it! You have a working AI assistant in 2 minutes.

---

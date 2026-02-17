# ☁️ Deploying PicoClaw on Coolify

Deploy PicoClaw as a self-hosted AI assistant on [Coolify](https://coolify.io) — the open-source Heroku/Vercel alternative.

> **Looking for the quick 3-step version?** See [README.md → Deploy on Coolify](README.md#%EF%B8%8F-deploy-on-coolify-3-steps)

---

## 📋 Prerequisites

- A Coolify instance (v4+)
- A GitHub account (to fork the repo)
- At least one LLM API key (e.g., [Gemini](https://aistudio.google.com/apikey), [OpenRouter](https://openrouter.ai/keys))

## 🚀 Quick Deploy

### Step 1: Fork the Repository

Fork [mrbeandev/picoclaw](https://github.com/mrbeandev/picoclaw) to your GitHub account.

### Step 2: Create a New Service in Coolify

1. Go to your Coolify dashboard → **Projects** → select or create a project
2. Click **+ New** → **Docker Compose**
3. Connect your forked GitHub repo
4. Set the following:
   - **Branch:** `deploy/coolify`
   - **Docker Compose File:** `docker-compose-coolify.yml`
   - **Base Directory:** `/` (root)

### Step 3: Configure Environment Variables

Go to **Environment Variables** tab and add your keys. See [Configuration](#-configuration) below.

### Step 4: Deploy!

Hit **Deploy** and wait for the build to complete (~30 seconds).

---

## ⚙️ Configuration

PicoClaw on Coolify supports **3 configuration methods**. The entrypoint script checks them in order — **first match wins**.

### Which Method Should I Use?

| | Method 1: JSON Env Var | Method 2: Mounted File | Method 3: Individual Env Vars |
|---|---|---|---|
| **Difficulty** | Medium | Easy | Easiest |
| **Flexibility** | ✅ Full control | ✅ Full control | ⚠️ Limited |
| **Custom providers (Ollama, vLLM)** | ✅ Yes | ✅ Yes | ❌ No |
| **Allow-lists** | ✅ Yes | ✅ Yes | ✅ Yes (comma-separated) |
| **Feishu, DingTalk, QQ, OneBot** | ✅ Yes | ✅ Yes | ❌ No |
| **Custom API base URLs** | ✅ Yes | ✅ Yes | ❌ No |
| **Requires JSON minification** | ⚠️ Yes | ❌ No | ❌ No |
| **Edit without rebuild** | ✅ Redeploy only | ✅ Restart only | ✅ Redeploy only |
| **Pretty-printed JSON** | ❌ Must minify | ✅ Yes | N/A |

---

### Method 1: Full JSON Config (Most Flexible) ⭐

**Best for:** Full control, custom providers (Ollama, vLLM), complex setups.

Paste your **entire** `config.json` as a single environment variable:

| Key | Value |
|-----|-------|
| `PICOCLAW_CONFIG_JSON` | `{"agents":{"defaults":{"provider":"gemini",...}},...}` |

#### Example: Gemini + Telegram + Ollama

```json
{
  "agents": {
    "defaults": {
      "provider": "gemini",
      "model": "gemini-2.5-flash-lite",
      "max_tokens": 8192,
      "temperature": 0.7,
      "max_tool_iterations": 20,
      "workspace": "~/.picoclaw/workspace",
      "restrict_to_workspace": true
    }
  },
  "providers": {
    "gemini": {
      "api_key": "AIzaSy..."
    },
    "vllm": {
      "api_key": "dummy",
      "api_base": "http://your-ollama-server:11434/v1"
    },
    "openrouter": {
      "api_key": "sk-or-..."
    }
  },
  "channels": {
    "telegram": {
      "enabled": true,
      "token": "123456:ABC-DEF...",
      "allow_from": ["your_telegram_user_id"]
    },
    "discord": {
      "enabled": false,
      "token": "",
      "allow_from": []
    }
  },
  "tools": {
    "web": {
      "duckduckgo": { "enabled": true, "max_results": 5 },
      "brave": { "enabled": false, "api_key": "", "max_results": 5 }
    },
    "firecrawl": { "enabled": false, "api_key": "", "api_base": "https://api.firecrawl.dev/v1" },
    "serpapi": { "enabled": false, "api_key": "", "max_results": 10 }
  },
  "heartbeat": { "enabled": true, "interval": 30 },
  "gateway": { "host": "0.0.0.0", "port": 18790 },
  "devices": { "enabled": false, "monitor_usb": false }
}
```

#### ⚠️ Important: You MUST minify the JSON!

Coolify environment variables are single-line. You need to compress the JSON into **one line** before pasting.

**🔧 JSON Tools for Minifying:**

| Tool | Type | URL |
|------|------|-----|
| **JSON Minifier** | Web | [jsonformatter.org/json-minify](https://jsonformatter.org/json-minify) |
| **JSON Formatter** | Web | [jsonformatter.curiousconcept.com](https://jsonformatter.curiousconcept.com/) |
| **JSON Editor Online** | Web | [jsoneditoronline.org](https://jsoneditoronline.org/) — edit visually, then copy minified |
| **jq** | CLI | `cat config.json \| jq -c .` — outputs minified JSON |
| **Python** | CLI | `python3 -c "import json,sys;print(json.dumps(json.load(sys.stdin)))" < config.json` |
| **Node.js** | CLI | `node -e "process.stdin.on('data',d=>console.log(JSON.stringify(JSON.parse(d))))"< config.json` |

**Workflow:**
1. Write your config in a pretty-printed JSON editor
2. Validate it (the tools above show errors)
3. Minify / compress to one line
4. Paste the single line as the `PICOCLAW_CONFIG_JSON` value in Coolify

**Example minified output:**
```
{"agents":{"defaults":{"provider":"gemini","model":"gemini-2.5-flash-lite","max_tokens":8192,"temperature":0.7,"max_tool_iterations":20,"workspace":"~/.picoclaw/workspace","restrict_to_workspace":true}},"providers":{"gemini":{"api_key":"AIzaSy..."},"vllm":{"api_key":"dummy","api_base":"http://ollama:11434/v1"}},"channels":{"telegram":{"enabled":true,"token":"123456:ABC-DEF...","allow_from":["123456789"]}},"tools":{"web":{"duckduckgo":{"enabled":true,"max_results":5}}},"heartbeat":{"enabled":true,"interval":30},"gateway":{"host":"0.0.0.0","port":18790},"devices":{"enabled":false,"monitor_usb":false}}
```

---

### Method 2: Mounted Config File

**Best for:** Users who prefer editing a normal file, and want pretty-printed JSON without minification.

Use Coolify's **Storages** feature to mount a config file into the container:

#### Step-by-step:

1. Go to your PicoClaw service in Coolify
2. Click the **Storages** tab
3. Click **+ Add** and configure:
   - **Source Path:** Leave empty (Coolify auto-creates it) or set to `/data/coolify/applications/<your-app-uuid>/config.json`
   - **Destination Path:** `/config/config.json`
4. Save the storage mount
5. SSH into your Coolify server and create the config file:
   ```bash
   # Find your app's data directory
   ls /data/coolify/applications/

   # Create the config file (replace <uuid> with your app's UUID)
   nano /data/coolify/applications/<uuid>/config.json
   ```
6. Paste your full config JSON (pretty-printed is fine!) and save
7. **Restart** the service in Coolify (no rebuild needed)

The entrypoint will automatically detect `/config/config.json` and use it.

#### ✅ Advantages
- **Pretty-printed JSON** — no minification needed, easy to read and edit
- **Full control** — same flexibility as Method 1
- **Edit without rebuild** — just edit the file on disk and restart the container
- **Custom providers** — Ollama, vLLM, and any other custom provider work fine

#### ❌ Limitations
- **Requires SSH access** — you need SSH into the Coolify server to create/edit the file
- **No Coolify UI editing** — you can't edit the file content from Coolify's web UI (only set the mount path)
- **File must exist before starting** — if the file doesn't exist, this method is skipped and it falls through to Method 3
- **Not portable** — the config lives on the server's filesystem, not in Coolify's database

---

### Method 3: Individual Environment Variables

**Best for:** Simple setups — just Gemini + one channel, defaults for everything else.

Set these in Coolify's **Environment Variables** tab:

#### Required

| Variable | Description | Example |
|----------|-------------|---------|
| `PICOCLAW_PROVIDERS_GEMINI_API_KEY` | Gemini API key | `AIzaSy...` |

#### Provider Keys (optional)

| Variable | Description |
|----------|-------------|
| `PICOCLAW_PROVIDERS_OPENROUTER_API_KEY` | OpenRouter API key |
| `PICOCLAW_PROVIDERS_OPENAI_API_KEY` | OpenAI API key |
| `PICOCLAW_PROVIDERS_ANTHROPIC_API_KEY` | Anthropic API key |
| `PICOCLAW_PROVIDERS_GROQ_API_KEY` | Groq API key |
| `PICOCLAW_PROVIDERS_MISTRAL_API_KEY` | Mistral API key |
| `PICOCLAW_PROVIDERS_DEEPSEEK_API_KEY` | DeepSeek API key |

#### Channel Config (optional)

| Variable | Description |
|----------|-------------|
| `PICOCLAW_CHANNELS_TELEGRAM_ENABLED` | `true` / `false` |
| `PICOCLAW_CHANNELS_TELEGRAM_TOKEN` | Telegram bot token |
| `PICOCLAW_CHANNELS_TELEGRAM_ALLOW_FROM` | Comma-separated user IDs (e.g., `123,456`) |
| `PICOCLAW_CHANNELS_DISCORD_ENABLED` | `true` / `false` |
| `PICOCLAW_CHANNELS_DISCORD_TOKEN` | Discord bot token |
| `PICOCLAW_CHANNELS_DISCORD_ALLOW_FROM` | Comma-separated user IDs |

#### Model Config (optional)

| Variable | Default | Description |
|----------|---------|-------------|
| `PICOCLAW_AGENTS_DEFAULTS_PROVIDER` | `gemini` | LLM provider name |
| `PICOCLAW_AGENTS_DEFAULTS_MODEL` | `gemini-2.5-flash-lite` | Model name |
| `PICOCLAW_AGENTS_DEFAULTS_MAX_TOKENS` | `8192` | Max output tokens |
| `PICOCLAW_AGENTS_DEFAULTS_TEMPERATURE` | `0.7` | Temperature |

#### Other (optional)

| Variable | Default | Description |
|----------|---------|-------------|
| `TZ` | `UTC` | Timezone (e.g., `Asia/Kolkata` for IST) |
| `PICOCLAW_HEARTBEAT_ENABLED` | `true` | Enable heartbeat |
| `PICOCLAW_HEARTBEAT_INTERVAL` | `30` | Heartbeat interval (minutes) |
| `PICOCLAW_TOOLS_WEB_DUCKDUCKGO_ENABLED` | `true` | DuckDuckGo search |
| `PICOCLAW_TOOLS_WEB_BRAVE_ENABLED` | `false` | Brave search |
| `PICOCLAW_TOOLS_WEB_BRAVE_API_KEY` | | Brave API key |

#### ✅ Advantages
- **Zero JSON knowledge needed** — just set key=value pairs
- **Easiest to set up** — add a few env vars and deploy
- **Good for quick testing** — get running in under a minute

#### ❌ Limitations
- **No custom providers** — only the built-in providers are supported (Gemini, OpenRouter, OpenAI, Anthropic, Groq, Mistral, DeepSeek, Zhipu, Moonshot, Nvidia, vLLM). You **cannot** add Ollama or other custom OpenAI-compatible providers
- **No custom API base URLs** — you can't override `api_base` for providers (needed for self-hosted models)
- **Limited channel support** — only Telegram, Discord, Slack, and LINE are configurable. Feishu, DingTalk, QQ, WhatsApp, MaixCam, and OneBot are **not** configurable via env vars
- **No proxy settings** — provider proxy configuration is not available
- **Hardcoded defaults** — many settings like `max_results`, `webhook_port`, etc. use hardcoded defaults that can't be changed
- **Allow-lists are comma-separated strings** — works but less flexible than JSON arrays (no spaces in IDs)

> **💡 Tip:** Start with Method 3 to get running quickly, then switch to Method 1 when you need more control.

---

## 🔧 Running Agent & Doctor Commands

The gateway service runs automatically. To run one-shot commands, SSH into your Coolify server and use:

```bash
# Run agent mode (one-shot question)
docker compose -f docker-compose-coolify.yml --profile agent run --rm picoclaw-agent -m "Hello!"

# Run agent mode (interactive)
docker compose -f docker-compose-coolify.yml --profile agent run --rm picoclaw-agent

# Run doctor (diagnostics)
docker compose -f docker-compose-coolify.yml --profile doctor run --rm picoclaw-doctor
```

### 🎮 Chat Commands (Telegram/Discord)

You can check status and swap models directly from your chat app:

| Command | Action |
|---------|--------|
| `/models` | View active model, provider, and all configured endpoints. |
| `/model <name>` | Switch the **model name** (keeping current provider). |
| `/model <provider>/<model>` | Switch **both** provider and model (e.g., `vllm/qwen3-coder-next:cloud`). |

> [!NOTE]
> Changes made via `/model` are active in memory. If the container restarts, it will revert to the default model defined in your `PICOCLAW_CONFIG_JSON` or environment variables.

---

## �️ Troubleshooting

### "No API key configured"
Your config isn't being loaded. Check:
- For Method 1: Is `PICOCLAW_CONFIG_JSON` set? Is it valid JSON?
- For Method 3: Is `PICOCLAW_PROVIDERS_GEMINI_API_KEY` set?
- Check container logs: `docker logs picoclaw-gateway` — look for the `📝 Using config from...` line.

### Container keeps restarting
Check logs: `docker logs picoclaw-gateway --tail 50`

Common issues:
- Invalid JSON in `PICOCLAW_CONFIG_JSON` (use a validator!)
- Missing API key for the configured provider

### Build fails
- Ensure you're using the `deploy/coolify` branch
- Check that both `Dockerfile.coolify` and `entrypoint-coolify.sh` exist in the repo

---

## 📁 File Structure (Coolify-specific)

```
picoclaw/
├── docker-compose-coolify.yml   # Coolify-optimized compose file
├── Dockerfile.coolify           # Coolify Dockerfile with entrypoint
├── entrypoint-coolify.sh        # Config generator script
├── Dockerfile                   # Original Dockerfile (not used by Coolify)
├── docker-compose.yml           # Original compose (not used by Coolify)
└── config/
    └── config.example.json      # Reference config with all options
```

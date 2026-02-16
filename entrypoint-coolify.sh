#!/bin/sh
# ============================================================
# PicoClaw Coolify Entrypoint
# Generates config.json before starting PicoClaw
#
# CONFIG PRIORITY (first match wins):
#   1. PICOCLAW_CONFIG_JSON env var — paste your entire JSON config
#   2. Mounted file at /config/config.json — use Coolify Storages
#   3. Auto-generated from individual env vars (basic setup)
# ============================================================
set -e

CONFIG_DIR="/root/.picoclaw"
CONFIG_FILE="${CONFIG_DIR}/config.json"

mkdir -p "${CONFIG_DIR}"

# ─────────────────────────────────────────────────
# METHOD 1: Full JSON config via env var
#   Set PICOCLAW_CONFIG_JSON in Coolify env vars
#   with your entire config.json content
# ─────────────────────────────────────────────────
if [ -n "${PICOCLAW_CONFIG_JSON}" ]; then
  echo "📝 Using config from PICOCLAW_CONFIG_JSON env var"
  echo "${PICOCLAW_CONFIG_JSON}" > "${CONFIG_FILE}"

  # CRITICAL: Unset all PICOCLAW_* env vars (except PICOCLAW_CONFIG_JSON)
  # so that Go's env.Parse() does NOT override values from the JSON file.
  # Without this, Dockerfile ENV defaults (e.g. PICOCLAW_AGENTS_DEFAULTS_PROVIDER=gemini)
  # would silently overwrite the user's JSON config every time.
  for var in $(env | grep '^PICOCLAW_' | cut -d= -f1); do
    if [ "$var" != "PICOCLAW_CONFIG_JSON" ]; then
      unset "$var"
    fi
  done

  exec picoclaw "$@"
fi

# ─────────────────────────────────────────────────
# METHOD 2: Mounted config file
#   In Coolify → Storages → Add:
#     Source: /data/coolify/applications/<uuid>/config.json
#     Destination: /config/config.json
#   Then paste your JSON in the file content
# ─────────────────────────────────────────────────
if [ -f "/config/config.json" ]; then
  echo "📝 Using mounted config from /config/config.json"
  cp /config/config.json "${CONFIG_FILE}"

  # Same protection as Method 1: unset PICOCLAW_* env vars
  for var in $(env | grep '^PICOCLAW_' | cut -d= -f1); do
    unset "$var"
  done

  exec picoclaw "$@"
fi

# ─────────────────────────────────────────────────
# METHOD 3: Auto-generate from individual env vars
#   Good for simple setups (Gemini + Telegram, etc.)
# ─────────────────────────────────────────────────
echo "📝 Generating config from individual env vars"

# Helper: comma-separated string → JSON array
csv_to_json_array() {
  input="$1"
  if [ -z "$input" ]; then echo "[]"; return; fi
  result="["
  first=true
  OLD_IFS="$IFS"; IFS=","
  for item in $input; do
    item=$(echo "$item" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
    if [ -n "$item" ]; then
      if [ "$first" = true ]; then first=false; else result="${result},"; fi
      result="${result}\"${item}\""
    fi
  done
  IFS="$OLD_IFS"
  echo "${result}]"
}

# ── Resolve env vars ─────────────────────────────
# Accept both Coolify-friendly short names and PICOCLAW_* internal names
R_PROVIDER="${PICOCLAW_AGENTS_DEFAULTS_PROVIDER:-${LLM_PROVIDER:-gemini}}"
R_MODEL="${PICOCLAW_AGENTS_DEFAULTS_MODEL:-${LLM_MODEL:-gemini-2.5-flash-lite}}"

R_GEMINI_KEY="${PICOCLAW_PROVIDERS_GEMINI_API_KEY:-${GEMINI_API_KEY:-}}"
R_OPENROUTER_KEY="${PICOCLAW_PROVIDERS_OPENROUTER_API_KEY:-${OPENROUTER_API_KEY:-}}"
R_OPENAI_KEY="${PICOCLAW_PROVIDERS_OPENAI_API_KEY:-${OPENAI_API_KEY:-}}"
R_ANTHROPIC_KEY="${PICOCLAW_PROVIDERS_ANTHROPIC_API_KEY:-${ANTHROPIC_API_KEY:-}}"
R_GROQ_KEY="${PICOCLAW_PROVIDERS_GROQ_API_KEY:-${GROQ_API_KEY:-}}"
R_MISTRAL_KEY="${PICOCLAW_PROVIDERS_MISTRAL_API_KEY:-${MISTRAL_API_KEY:-}}"
R_DEEPSEEK_KEY="${PICOCLAW_PROVIDERS_DEEPSEEK_API_KEY:-${DEEPSEEK_API_KEY:-}}"
R_VLLM_KEY="${PICOCLAW_PROVIDERS_VLLM_API_KEY:-${VLLM_API_KEY:-}}"
R_VLLM_BASE="${PICOCLAW_PROVIDERS_VLLM_API_BASE:-${VLLM_API_BASE:-}}"

R_TELEGRAM="${PICOCLAW_CHANNELS_TELEGRAM_TOKEN:-${TELEGRAM_BOT_TOKEN:-}}"
R_DISCORD="${PICOCLAW_CHANNELS_DISCORD_TOKEN:-${DISCORD_BOT_TOKEN:-}}"
R_LINE_SECRET="${PICOCLAW_CHANNELS_LINE_CHANNEL_SECRET:-${LINE_CHANNEL_SECRET:-}}"
R_LINE_ACCESS="${PICOCLAW_CHANNELS_LINE_CHANNEL_ACCESS_TOKEN:-${LINE_CHANNEL_ACCESS_TOKEN:-}}"

R_BRAVE_KEY="${PICOCLAW_TOOLS_WEB_BRAVE_API_KEY:-${BRAVE_SEARCH_API_KEY:-}}"
R_BRAVE_ON="${PICOCLAW_TOOLS_WEB_BRAVE_ENABLED:-${BRAVE_SEARCH_ENABLED:-false}}"
R_DDG_ON="${PICOCLAW_TOOLS_WEB_DUCKDUCKGO_ENABLED:-${DUCKDUCKGO_ENABLED:-true}}"
R_HB_ON="${PICOCLAW_HEARTBEAT_ENABLED:-${HEARTBEAT_ENABLED:-true}}"
R_HB_INT="${PICOCLAW_HEARTBEAT_INTERVAL:-${HEARTBEAT_INTERVAL:-30}}"

# ── Auto-enable channels when token is provided ──
if [ -n "$R_TELEGRAM" ]; then TG_ON="true"; else TG_ON="${PICOCLAW_CHANNELS_TELEGRAM_ENABLED:-false}"; fi
if [ -n "$R_DISCORD" ]; then DC_ON="true"; else DC_ON="${PICOCLAW_CHANNELS_DISCORD_ENABLED:-false}"; fi
if [ -n "$R_LINE_SECRET" ] && [ -n "$R_LINE_ACCESS" ]; then LN_ON="true"; else LN_ON="${PICOCLAW_CHANNELS_LINE_ENABLED:-false}"; fi

TELEGRAM_ALLOW=$(csv_to_json_array "${PICOCLAW_CHANNELS_TELEGRAM_ALLOW_FROM:-}")
DISCORD_ALLOW=$(csv_to_json_array "${PICOCLAW_CHANNELS_DISCORD_ALLOW_FROM:-}")
SLACK_ALLOW=$(csv_to_json_array "${PICOCLAW_CHANNELS_SLACK_ALLOW_FROM:-}")
LINE_ALLOW=$(csv_to_json_array "${PICOCLAW_CHANNELS_LINE_ALLOW_FROM:-}")

cat > "${CONFIG_FILE}" <<ENDOFCONFIG
{
  "agents": {
    "defaults": {
      "workspace": "~/.picoclaw/workspace",
      "restrict_to_workspace": true,
      "provider": "${R_PROVIDER}",
      "model": "${R_MODEL}",
      "max_tokens": ${PICOCLAW_AGENTS_DEFAULTS_MAX_TOKENS:-8192},
      "temperature": ${PICOCLAW_AGENTS_DEFAULTS_TEMPERATURE:-0.7},
      "max_tool_iterations": ${PICOCLAW_AGENTS_DEFAULTS_MAX_TOOL_ITERATIONS:-20}
    }
  },
  "channels": {
    "telegram": { "enabled": ${TG_ON}, "token": "${R_TELEGRAM}", "allow_from": ${TELEGRAM_ALLOW} },
    "discord": { "enabled": ${DC_ON}, "token": "${R_DISCORD}", "allow_from": ${DISCORD_ALLOW} },
    "slack": { "enabled": ${PICOCLAW_CHANNELS_SLACK_ENABLED:-false}, "bot_token": "${PICOCLAW_CHANNELS_SLACK_BOT_TOKEN:-}", "app_token": "${PICOCLAW_CHANNELS_SLACK_APP_TOKEN:-}", "allow_from": ${SLACK_ALLOW} },
    "line": { "enabled": ${LN_ON}, "channel_secret": "${R_LINE_SECRET}", "channel_access_token": "${R_LINE_ACCESS}", "webhook_host": "0.0.0.0", "webhook_port": 18791, "webhook_path": "/webhook/line", "allow_from": ${LINE_ALLOW} },
    "maixcam": { "enabled": false, "host": "0.0.0.0", "port": 18790, "allow_from": [] },
    "whatsapp": { "enabled": false, "bridge_url": "ws://localhost:3001", "allow_from": [] },
    "feishu": { "enabled": false, "app_id": "", "app_secret": "", "encrypt_key": "", "verification_token": "", "allow_from": [] },
    "dingtalk": { "enabled": false, "client_id": "", "client_secret": "", "allow_from": [] },
    "onebot": { "enabled": false, "ws_url": "ws://127.0.0.1:3001", "access_token": "", "reconnect_interval": 5, "group_trigger_prefix": [], "allow_from": [] }
  },
  "providers": {
    "gemini": { "api_key": "${R_GEMINI_KEY}", "api_base": "${PICOCLAW_PROVIDERS_GEMINI_API_BASE:-}" },
    "openrouter": { "api_key": "${R_OPENROUTER_KEY}", "api_base": "${PICOCLAW_PROVIDERS_OPENROUTER_API_BASE:-}" },
    "openai": { "api_key": "${R_OPENAI_KEY}", "api_base": "${PICOCLAW_PROVIDERS_OPENAI_API_BASE:-}" },
    "anthropic": { "api_key": "${R_ANTHROPIC_KEY}", "api_base": "${PICOCLAW_PROVIDERS_ANTHROPIC_API_BASE:-}" },
    "groq": { "api_key": "${R_GROQ_KEY}", "api_base": "${PICOCLAW_PROVIDERS_GROQ_API_BASE:-}" },
    "mistral": { "api_key": "${R_MISTRAL_KEY}", "api_base": "${PICOCLAW_PROVIDERS_MISTRAL_API_BASE:-}" },
    "zhipu": { "api_key": "${PICOCLAW_PROVIDERS_ZHIPU_API_KEY:-}", "api_base": "${PICOCLAW_PROVIDERS_ZHIPU_API_BASE:-}" },
    "moonshot": { "api_key": "${PICOCLAW_PROVIDERS_MOONSHOT_API_KEY:-}", "api_base": "${PICOCLAW_PROVIDERS_MOONSHOT_API_BASE:-}" },
    "deepseek": { "api_key": "${R_DEEPSEEK_KEY}", "api_base": "${PICOCLAW_PROVIDERS_DEEPSEEK_API_BASE:-}" },
    "nvidia": { "api_key": "${PICOCLAW_PROVIDERS_NVIDIA_API_KEY:-}", "api_base": "${PICOCLAW_PROVIDERS_NVIDIA_API_BASE:-}" },
    "vllm": { "api_key": "${R_VLLM_KEY}", "api_base": "${R_VLLM_BASE}" }
  },
  "tools": {
    "web": {
      "brave": { "enabled": ${R_BRAVE_ON}, "api_key": "${R_BRAVE_KEY}", "max_results": 5 },
      "duckduckgo": { "enabled": ${R_DDG_ON}, "max_results": 5 }
    },
    "firecrawl": { "enabled": ${PICOCLAW_TOOLS_FIRECRAWL_ENABLED:-false}, "api_key": "${PICOCLAW_TOOLS_FIRECRAWL_API_KEY:-}", "api_base": "https://api.firecrawl.dev/v1" },
    "serpapi": { "enabled": ${PICOCLAW_TOOLS_SERPAPI_ENABLED:-false}, "api_key": "${PICOCLAW_TOOLS_SERPAPI_API_KEY:-}", "max_results": 10 }
  },
  "heartbeat": { "enabled": ${R_HB_ON}, "interval": ${R_HB_INT} },
  "devices": { "enabled": false, "monitor_usb": false },
  "gateway": { "host": "0.0.0.0", "port": 18790 }
}
ENDOFCONFIG

exec picoclaw "$@"

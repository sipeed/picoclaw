# Configuration File Reference

The main configuration file is located at `~/.picoclaw/config.json`. This page provides a complete reference for all configuration options.

## Configuration File Location

```
~/.picoclaw/config.json
```

Use `picoclaw onboard` to create a default configuration file.

## Configuration Structure

```json
{
  "agents": { ... },
  "bindings": [ ... ],
  "session": { ... },
  "channels": { ... },
  "providers": { ... },
  "gateway": { ... },
  "tools": { ... },
  "heartbeat": { ... },
  "devices": { ... }
}
```

## Agents Configuration

Configure agent behavior and model settings.

### agents.defaults

Default settings for all agents.

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `workspace` | string | `~/.picoclaw/workspace` | Working directory for the agent |
| `restrict_to_workspace` | bool | `true` | Restrict file/command access to workspace |
| `provider` | string | `""` | Explicit provider name (optional) |
| `model` | string | `glm-4.7` | Primary model to use |
| `model_fallbacks` | []string | `[]` | Fallback models if primary fails |
| `image_model` | string | `""` | Model for image tasks |
| `image_model_fallbacks` | []string | `[]` | Image model fallbacks |
| `max_tokens` | int | `8192` | Maximum tokens in response |
| `temperature` | float | `0.7` | Response randomness (0-1) |
| `max_tool_iterations` | int | `20` | Max tool call iterations |

**Example:**

```json
{
  "agents": {
    "defaults": {
      "workspace": "~/.picoclaw/workspace",
      "restrict_to_workspace": true,
      "model": "anthropic/claude-opus-4-5",
      "model_fallbacks": ["anthropic/claude-sonnet-4", "gpt-4o"],
      "max_tokens": 8192,
      "temperature": 0.7,
      "max_tool_iterations": 20
    }
  }
}
```

### agents.list (Multi-Agent)

Define multiple agents with different configurations.

| Option | Type | Description |
|--------|------|-------------|
| `id` | string | Unique agent identifier |
| `default` | bool | Is this the default agent |
| `name` | string | Display name |
| `workspace` | string | Agent-specific workspace |
| `model` | object/string | Model config with primary and fallbacks |
| `skills` | []string | Filter available skills |
| `subagents` | object | Subagent configuration |

**Example:**

```json
{
  "agents": {
    "defaults": { ... },
    "list": [
      {
        "id": "assistant",
        "default": true,
        "name": "General Assistant",
        "workspace": "~/.picoclaw/workspace/assistant",
        "model": {
          "primary": "anthropic/claude-opus-4-5",
          "fallbacks": ["gpt-4o"]
        },
        "subagents": {
          "allow_agents": ["researcher", "coder"]
        }
      },
      {
        "id": "researcher",
        "name": "Research Agent",
        "model": "perplexity/llama-3.1-sonar-large-128k-online",
        "skills": ["web-search"]
      }
    ]
  }
}
```

## Bindings Configuration

Route messages from channels to specific agents.

| Option | Type | Description |
|--------|------|-------------|
| `agent_id` | string | Target agent ID |
| `match.channel` | string | Channel name |
| `match.account_id` | string | Account filter |
| `match.peer.kind` | string | Peer type (user, group, channel) |
| `match.peer.id` | string | Peer ID |
| `match.guild_id` | string | Discord guild ID |
| `match.team_id` | string | Slack team ID |

**Example:**

```json
{
  "bindings": [
    {
      "agent_id": "personal",
      "match": { "channel": "telegram", "peer": { "kind": "user", "id": "123456789" } }
    },
    {
      "agent_id": "work",
      "match": { "channel": "slack", "team_id": "T12345" }
    }
  ]
}
```

## Session Configuration

Configure session management.

| Option | Type | Description |
|--------|------|-------------|
| `dm_scope` | string | DM session scope: `main`, `per-agent`, `per-channel` |
| `identity_links` | object | Map user IDs across platforms |

**Example:**

```json
{
  "session": {
    "dm_scope": "per-agent",
    "identity_links": {
      "telegram:123456789": ["discord:987654321", "slack:U12345"]
    }
  }
}
```

## Channels Configuration

Configure chat platform integrations.

### Telegram

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enabled` | bool | `false` | Enable Telegram channel |
| `token` | string | `""` | Bot token from @BotFather |
| `proxy` | string | `""` | Proxy URL (optional) |
| `allow_from` | []string | `[]` | Allowed user IDs (empty = all) |

```json
{
  "channels": {
    "telegram": {
      "enabled": true,
      "token": "123456:ABC...",
      "allow_from": ["123456789"]
    }
  }
}
```

### Discord

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enabled` | bool | `false` | Enable Discord channel |
| `token` | string | `""` | Bot token |
| `allow_from` | []string | `[]` | Allowed user IDs |

```json
{
  "channels": {
    "discord": {
      "enabled": true,
      "token": "MTk4NjIyNDgzNDc...",
      "allow_from": ["123456789012345678"]
    }
  }
}
```

### Slack

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enabled` | bool | `false` | Enable Slack channel |
| `bot_token` | string | `""` | Bot token (xoxb-...) |
| `app_token` | string | `""` | App token (xapp-...) |
| `allow_from` | []string | `[]` | Allowed user IDs |

### LINE

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enabled` | bool | `false` | Enable LINE channel |
| `channel_secret` | string | `""` | Channel secret |
| `channel_access_token` | string | `""` | Channel access token |
| `webhook_host` | string | `0.0.0.0` | Webhook host |
| `webhook_port` | int | `18791` | Webhook port |
| `webhook_path` | string | `/webhook/line` | Webhook path |
| `allow_from` | []string | `[]` | Allowed user IDs |

### WhatsApp

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enabled` | bool | `false` | Enable WhatsApp channel |
| `bridge_url` | string | `ws://localhost:3001` | Bridge WebSocket URL |
| `allow_from` | []string | `[]` | Allowed user IDs |

### Feishu/Lark

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enabled` | bool | `false` | Enable Feishu channel |
| `app_id` | string | `""` | App ID |
| `app_secret` | string | `""` | App secret |
| `encrypt_key` | string | `""` | Encryption key |
| `verification_token` | string | `""` | Verification token |
| `allow_from` | []string | `[]` | Allowed user IDs |

### DingTalk

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enabled` | bool | `false` | Enable DingTalk channel |
| `client_id` | string | `""` | Client ID |
| `client_secret` | string | `""` | Client secret |
| `allow_from` | []string | `[]` | Allowed user IDs |

### QQ

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enabled` | bool | `false` | Enable QQ channel |
| `app_id` | string | `""` | App ID |
| `app_secret` | string | `""` | App secret |
| `allow_from` | []string | `[]` | Allowed user IDs |

### OneBot

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enabled` | bool | `false` | Enable OneBot channel |
| `ws_url` | string | `ws://127.0.0.1:3001` | WebSocket URL |
| `access_token` | string | `""` | Access token |
| `reconnect_interval` | int | `5` | Reconnect interval (seconds) |
| `group_trigger_prefix` | []string | `[]` | Trigger prefixes for groups |
| `allow_from` | []string | `[]` | Allowed user IDs |

### MaixCam

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enabled` | bool | `false` | Enable MaixCam channel |
| `host` | string | `0.0.0.0` | Server host |
| `port` | int | `18790` | Server port |
| `allow_from` | []string | `[]` | Allowed user IDs |

## Providers Configuration

Configure LLM providers. See [Providers Guide](../user-guide/providers/README.md) for detailed setup.

### Common Provider Options

| Option | Type | Description |
|--------|------|-------------|
| `api_key` | string | API key |
| `api_base` | string | API base URL (optional) |
| `proxy` | string | Proxy URL (optional) |
| `auth_method` | string | Auth method: `oauth`, `token`, `codex-cli` |

### Provider-Specific Options

**OpenAI:**

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `web_search` | bool | `true` | Enable web search in responses |

**Example:**

```json
{
  "providers": {
    "openrouter": {
      "api_key": "sk-or-v1-xxx",
      "api_base": "https://openrouter.ai/api/v1"
    },
    "zhipu": {
      "api_key": "xxx.xxx",
      "api_base": "https://open.bigmodel.cn/api/paas/v4"
    },
    "anthropic": {
      "api_key": "",
      "auth_method": "token"
    },
    "openai": {
      "api_key": "",
      "web_search": true,
      "auth_method": "oauth"
    },
    "groq": {
      "api_key": "gsk_xxx"
    },
    "gemini": {
      "api_key": ""
    },
    "ollama": {
      "api_key": "",
      "api_base": "http://localhost:11434/v1"
    },
    "vllm": {
      "api_key": "local",
      "api_base": "http://localhost:8000/v1"
    }
  }
}
```

## Tools Configuration

Configure tool behavior.

### Web Tools

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `brave.enabled` | bool | `false` | Enable Brave Search |
| `brave.api_key` | string | `""` | Brave API key |
| `brave.max_results` | int | `5` | Max search results |
| `duckduckgo.enabled` | bool | `true` | Enable DuckDuckGo |
| `duckduckgo.max_results` | int | `5` | Max search results |
| `perplexity.enabled` | bool | `false` | Enable Perplexity |
| `perplexity.api_key` | string | `""` | Perplexity API key |
| `perplexity.max_results` | int | `5` | Max search results |

### Cron Tools

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `exec_timeout_minutes` | int | `5` | Timeout for cron jobs (0 = no timeout) |

### Exec Tools

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enable_deny_patterns` | bool | `true` | Enable dangerous command blocking |
| `custom_deny_patterns` | []string | `[]` | Additional patterns to block |

**Example:**

```json
{
  "tools": {
    "web": {
      "brave": {
        "enabled": true,
        "api_key": "BSA...",
        "max_results": 5
      },
      "duckduckgo": {
        "enabled": true,
        "max_results": 5
      }
    },
    "cron": {
      "exec_timeout_minutes": 5
    },
    "exec": {
      "enable_deny_patterns": true,
      "custom_deny_patterns": ["wget.*"]
    }
  }
}
```

## Gateway Configuration

Configure the gateway server.

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `host` | string | `0.0.0.0` | Server host |
| `port` | int | `18790` | Server port |

```json
{
  "gateway": {
    "host": "0.0.0.0",
    "port": 18790
  }
}
```

## Heartbeat Configuration

Configure periodic tasks.

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enabled` | bool | `true` | Enable heartbeat service |
| `interval` | int | `30` | Check interval in minutes (min: 5) |

```json
{
  "heartbeat": {
    "enabled": true,
    "interval": 30
  }
}
```

## Devices Configuration

Configure device monitoring.

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enabled` | bool | `false` | Enable device service |
| `monitor_usb` | bool | `true` | Monitor USB devices (Linux only) |

```json
{
  "devices": {
    "enabled": false,
    "monitor_usb": true
  }
}
```

## Environment Variables

All configuration options can be overridden with environment variables using the pattern:

```
PICOCLAW_<SECTION>_<KEY>
```

### Examples

| Config Path | Environment Variable |
|-------------|---------------------|
| `agents.defaults.model` | `PICOCLAW_AGENTS_DEFAULTS_MODEL` |
| `providers.openrouter.api_key` | `PICOCLAW_PROVIDERS_OPENROUTER_API_KEY` |
| `channels.telegram.enabled` | `PICOCLAW_CHANNELS_TELEGRAM_ENABLED` |
| `tools.web.duckduckgo.enabled` | `PICOCLAW_TOOLS_WEB_DUCKDUCKGO_ENABLED` |
| `heartbeat.enabled` | `PICOCLAW_HEARTBEAT_ENABLED` |
| `heartbeat.interval` | `PICOCLAW_HEARTBEAT_INTERVAL` |

**Usage:**

```bash
export PICOCLAW_AGENTS_DEFAULTS_MODEL="gpt-4o"
export PICOCLAW_PROVIDERS_OPENROUTER_API_KEY="sk-or-v1-xxx"
export PICOCLAW_HEARTBEAT_ENABLED="false"

picoclaw agent -m "Hello"
```

## Complete Example

```json
{
  "agents": {
    "defaults": {
      "workspace": "~/.picoclaw/workspace",
      "restrict_to_workspace": true,
      "model": "anthropic/claude-opus-4-5",
      "model_fallbacks": ["gpt-4o", "glm-4.7"],
      "max_tokens": 8192,
      "temperature": 0.7,
      "max_tool_iterations": 20
    }
  },
  "channels": {
    "telegram": {
      "enabled": true,
      "token": "123456:ABC...",
      "allow_from": ["123456789"]
    },
    "discord": {
      "enabled": true,
      "token": "MTk4NjIyNDgzNDc...",
      "allow_from": ["123456789012345678"]
    }
  },
  "providers": {
    "openrouter": {
      "api_key": "sk-or-v1-xxx"
    },
    "groq": {
      "api_key": "gsk_xxx"
    },
    "zhipu": {
      "api_key": "xxx.xxx"
    }
  },
  "tools": {
    "web": {
      "brave": {
        "enabled": true,
        "api_key": "BSA...",
        "max_results": 5
      },
      "duckduckgo": {
        "enabled": true,
        "max_results": 5
      }
    }
  },
  "gateway": {
    "host": "0.0.0.0",
    "port": 18790
  },
  "heartbeat": {
    "enabled": true,
    "interval": 30
  }
}
```

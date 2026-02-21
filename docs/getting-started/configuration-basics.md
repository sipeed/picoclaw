# Configuration Basics

This guide explains the fundamental concepts of PicoClaw configuration.

## Configuration File Location

PicoClaw stores its configuration in:

```
~/.picoclaw/config.json
```

Use `picoclaw onboard` to create a default configuration file.

## Configuration Sections

The configuration file has several main sections:

### agents

Controls agent behavior and model settings.

```json
{
  "agents": {
    "defaults": {
      "workspace": "~/.picoclaw/workspace",
      "model": "glm-4.7",
      "max_tokens": 8192,
      "temperature": 0.7
    }
  }
}
```

Key options:
- `workspace` - Where the agent stores files and sessions
- `model` - The LLM model to use
- `restrict_to_workspace` - Security sandbox (default: true)

### providers

API keys for LLM providers.

```json
{
  "providers": {
    "openrouter": {
      "api_key": "sk-or-v1-xxx"
    },
    "zhipu": {
      "api_key": "your-zhipu-key"
    }
  }
}
```

### channels

Chat platform integrations.

```json
{
  "channels": {
    "telegram": {
      "enabled": true,
      "token": "YOUR_BOT_TOKEN",
      "allow_from": ["YOUR_USER_ID"]
    }
  }
}
```

### tools

Tool configuration.

```json
{
  "tools": {
    "web": {
      "brave": {
        "enabled": true,
        "api_key": "YOUR_BRAVE_KEY"
      }
    }
  }
}
```

## Environment Variables

All config options can be overridden with environment variables:

```bash
# Pattern: PICOCLAW_<SECTION>_<KEY>
export PICOCLAW_AGENTS_DEFAULTS_MODEL="gpt-4o"
export PICOCLAW_PROVIDERS_OPENROUTER_API_KEY="sk-or-v1-xxx"
```

## Provider Selection

PicoClaw selects providers in this order:

1. Explicit `provider` in config
2. Model name prefix (e.g., `openrouter/claude-...`)
3. First configured API key found

## Model Format

Models can be specified in two ways:

**With provider prefix:**
```json
{
  "model": "openrouter/anthropic/claude-opus-4-5"
}
```

**Without prefix (uses default provider):**
```json
{
  "model": "glm-4.7"
}
```

## Minimal Configuration

The minimal config to get started:

```json
{
  "providers": {
    "openrouter": {
      "api_key": "sk-or-v1-xxx"
    }
  }
}
```

All other options have sensible defaults.

## Next Steps

- [Configuration Reference](../configuration/config-file.md) - All config options
- [Providers Guide](../user-guide/providers/README.md) - Provider setup details
- [Quick Start](quick-start.md) - Get running quickly

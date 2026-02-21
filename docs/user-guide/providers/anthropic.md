# Anthropic Provider

Use Anthropic's Claude models directly through their API.

## Why Anthropic?

- **Direct access**: No middleman, direct API access
- **Latest models**: Immediate access to newest Claude versions
- **Native features**: Full feature support
- **OAuth support**: Secure authentication

## Setup

### Step 1: Get API Key

1. Go to [console.anthropic.com](https://console.anthropic.com)
2. Sign in or create an account
3. Navigate to API Keys
4. Create a new API key
5. Copy the key (starts with `sk-ant-`)

### Step 2: Configure

Edit `~/.picoclaw/config.json`:

```json
{
  "agents": {
    "defaults": {
      "model": "claude-opus-4-5"
    }
  },
  "providers": {
    "anthropic": {
      "api_key": "sk-ant-YOUR-API-KEY",
      "api_base": "https://api.anthropic.com"
    }
  }
}
```

## Configuration Options

| Option | Required | Default | Description |
|--------|----------|---------|-------------|
| `api_key` | Yes* | - | Your Anthropic API key |
| `api_base` | No | `https://api.anthropic.com` | API endpoint |

*Required unless using OAuth

## Available Models

| Model | Model Name | Best For |
|-------|------------|----------|
| Claude Opus 4.5 | `claude-opus-4-5` | Most capable, complex tasks |
| Claude Opus 4 | `claude-opus-4-20250514` | High-intelligence tasks |
| Claude Sonnet 4 | `claude-sonnet-4-20250514` | Fast + capable |
| Claude Sonnet 3.7 | `claude-3-7-sonnet-20250219` | Balanced performance |
| Claude Sonnet 3.5 | `claude-3-5-sonnet-20241022` | Fast, efficient |

## OAuth Authentication

For secure authentication without storing API keys:

```bash
# Login with OAuth
picoclaw auth login --provider anthropic

# Use device code flow (for headless environments)
picoclaw auth login --provider anthropic --device-code
```

After OAuth login, you don't need to store the API key in config.

## Model Format

Use the model name directly (no provider prefix needed):

```json
{
  "agents": {
    "defaults": {
      "model": "claude-opus-4-5"
    }
  }
}
```

Or with provider prefix:

```json
{
  "agents": {
    "defaults": {
      "model": "anthropic/claude-opus-4-5"
    }
  }
}
```

## Fallback Configuration

```json
{
  "agents": {
    "defaults": {
      "model": "claude-opus-4-5",
      "model_fallbacks": [
        "claude-sonnet-4-20250514",
        "claude-3-5-sonnet-20241022"
      ]
    }
  }
}
```

## Environment Variable

```bash
export PICOCLAW_PROVIDERS_ANTHROPIC_API_KEY="sk-ant-xxx"
```

## Pricing

Check [anthropic.com/pricing](https://www.anthropic.com/pricing) for current pricing.

Claude models are typically:
- Opus: Highest cost, best capability
- Sonnet: Mid-range cost, good balance
- Haiku: Lowest cost, fastest

## Rate Limits

Anthropic applies rate limits based on your tier:

| Tier | Requests per Minute | Tokens per Minute |
|------|---------------------|-------------------|
| Free | Varies | Varies |
| Tier 1 | 50 | 100,000 |
| Tier 2 | 100 | 200,000 |
| Higher | Custom | Custom |

Handle rate limits with fallbacks:

```json
{
  "agents": {
    "defaults": {
      "model": "claude-opus-4-5",
      "model_fallbacks": ["claude-sonnet-4-20250514"]
    }
  }
}
```

## Usage Examples

```bash
# Use default model
picoclaw agent -m "Hello!"

# Specify model
picoclaw agent -m "Hello!" # Uses config model
```

## Features

### Extended Thinking

Claude models support extended thinking for complex reasoning:

```json
{
  "agents": {
    "defaults": {
      "model": "claude-opus-4-5",
      "thinking": {
        "enabled": true,
        "budget_tokens": 10000
      }
    }
  }
}
```

### Vision

Claude can analyze images (when supported by the channel):

```json
{
  "agents": {
    "defaults": {
      "model": "claude-opus-4-5",
      "vision": true
    }
  }
}
```

## Troubleshooting

### Invalid API Key

```
Error: invalid_api_key
```

Verify your API key:
1. Check key starts with `sk-ant-`
2. Ensure key is active in console
3. Check for typos

### Rate Limited

```
Error: rate_limit_exceeded
```

Solutions:
1. Wait and retry
2. Set up fallback models
3. Upgrade your tier

### Model Not Found

```
Error: model_not_found
```

Check model name is correct. Use exact model IDs from the Anthropic documentation.

## See Also

- [Providers Overview](README.md)
- [Configuration Reference](../../configuration/config-file.md)
- [Model Fallbacks](../advanced/model-fallbacks.md)

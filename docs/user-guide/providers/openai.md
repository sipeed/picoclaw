# OpenAI Provider

Use OpenAI's GPT models directly through their API.

## Why OpenAI?

- **GPT models**: Access to GPT-4, GPT-4o, and GPT-3.5
- **DALL-E**: Image generation capabilities
- **Whisper**: Audio transcription
- **Native features**: Full OpenAI API support

## Setup

### Step 1: Get API Key

1. Go to [platform.openai.com](https://platform.openai.com)
2. Sign in or create an account
3. Navigate to API Keys
4. Create a new secret key
5. Copy the key (starts with `sk-`)

### Step 2: Configure

Edit `~/.picoclaw/config.json`:

```json
{
  "agents": {
    "defaults": {
      "model": "gpt-4o"
    }
  },
  "providers": {
    "openai": {
      "api_key": "sk-YOUR-API-KEY",
      "api_base": "https://api.openai.com/v1"
    }
  }
}
```

## Configuration Options

| Option | Required | Default | Description |
|--------|----------|---------|-------------|
| `api_key` | Yes* | - | Your OpenAI API key |
| `api_base` | No | `https://api.openai.com/v1` | API endpoint |
| `organization` | No | - | Organization ID |

*Required unless using OAuth

## Available Models

### Chat Models

| Model | Model Name | Best For |
|-------|------------|----------|
| GPT-4o | `gpt-4o` | Latest flagship, multimodal |
| GPT-4o Mini | `gpt-4o-mini` | Fast, cost-effective |
| GPT-4 Turbo | `gpt-4-turbo` | Previous generation flagship |
| GPT-4 | `gpt-4` | Standard GPT-4 |
| GPT-3.5 Turbo | `gpt-3.5-turbo` | Fast, affordable |

### Reasoning Models

| Model | Model Name | Best For |
|-------|------------|----------|
| o1 | `o1` | Complex reasoning |
| o1-mini | `o1-mini` | Fast reasoning tasks |
| o1-preview | `o1-preview` | Preview reasoning model |

## OAuth Authentication

For secure authentication without storing API keys:

```bash
# Login with OAuth
picoclaw auth login --provider openai

# Use device code flow (for headless environments)
picoclaw auth login --provider openai --device-code
```

## Model Format

Use the model name directly:

```json
{
  "agents": {
    "defaults": {
      "model": "gpt-4o"
    }
  }
}
```

Or with provider prefix:

```json
{
  "agents": {
    "defaults": {
      "model": "openai/gpt-4o"
    }
  }
}
```

## Fallback Configuration

```json
{
  "agents": {
    "defaults": {
      "model": "gpt-4o",
      "model_fallbacks": [
        "gpt-4o-mini",
        "gpt-3.5-turbo"
      ]
    }
  }
}
```

## Environment Variable

```bash
export PICOCLAW_PROVIDERS_OPENAI_API_KEY="sk-xxx"
```

## Pricing

Check [openai.com/pricing](https://openai.com/pricing) for current pricing.

Approximate costs (per 1M tokens):

| Model | Input | Output |
|-------|-------|--------|
| GPT-4o | $2.50 | $10.00 |
| GPT-4o Mini | $0.15 | $0.60 |
| GPT-4 Turbo | $10.00 | $30.00 |
| GPT-3.5 Turbo | $0.50 | $1.50 |

## Rate Limits

OpenAI applies rate limits:

| Tier | Requests per Minute | Tokens per Minute |
|------|---------------------|-------------------|
| Free | 3 | 40,000 |
| Tier 1 | 500 | 200,000 |
| Tier 2 | 5,000 | 2,000,000 |

## Usage Examples

```bash
# Use default model
picoclaw agent -m "Hello!"

# With specific model in config
picoclaw agent -m "Explain quantum computing"
```

## Special Features

### Function Calling

OpenAI models support function calling, which PicoClaw uses for tools.

### Vision

GPT-4o and GPT-4 Turbo support image analysis:

```json
{
  "agents": {
    "defaults": {
      "model": "gpt-4o",
      "vision": true
    }
  }
}
```

### Structured Outputs

For applications requiring structured responses:

```json
{
  "agents": {
    "defaults": {
      "model": "gpt-4o",
      "response_format": "json_object"
    }
  }
}
```

## Using with Azure OpenAI

For Azure OpenAI Service:

```json
{
  "providers": {
    "openai": {
      "api_key": "your-azure-key",
      "api_base": "https://your-resource.openai.azure.com/openai/deployments/your-deployment",
      "api_version": "2024-02-15-preview",
      "azure": true
    }
  }
}
```

## Troubleshooting

### Invalid API Key

```
Error: incorrect_api_key
```

Verify your API key:
1. Check key starts with `sk-`
2. Ensure key is active
3. Check for typos

### Insufficient Quota

```
Error: insufficient_quota
```

Solutions:
1. Add billing information
2. Wait for quota reset
3. Use fallback models

### Rate Limited

```
Error: rate_limit_exceeded
```

Solutions:
1. Wait and retry
2. Set up fallback models
3. Upgrade your tier

### Context Length Exceeded

```
Error: context_length_exceeded
```

Solutions:
1. Reduce message length
2. Clear session history
3. Use a model with larger context

## See Also

- [Providers Overview](README.md)
- [Configuration Reference](../../configuration/config-file.md)
- [Model Fallbacks](../advanced/model-fallbacks.md)

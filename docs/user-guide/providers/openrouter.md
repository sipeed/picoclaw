# OpenRouter Provider

OpenRouter is the recommended provider for most users. It provides access to many LLM models through a single API.

## Why OpenRouter?

- **Multi-model access**: Claude, GPT, Llama, Mistral, and more
- **Free tier**: 200K tokens per month
- **Simple pricing**: Pay per use after free tier
- **Single API key**: Access all models with one key

## Setup

### Step 1: Get API Key

1. Go to [openrouter.ai/keys](https://openrouter.ai/keys)
2. Sign in or create an account
3. Create an API key
4. Copy the key (starts with `sk-or-v1-`)

### Step 2: Configure

Edit `~/.picoclaw/config.json`:

```json
{
  "agents": {
    "defaults": {
      "model": "anthropic/claude-opus-4-5"
    }
  },
  "providers": {
    "openrouter": {
      "api_key": "sk-or-v1-YOUR-API-KEY",
      "api_base": "https://openrouter.ai/api/v1"
    }
  }
}
```

## Configuration Options

| Option | Required | Default | Description |
|--------|----------|---------|-------------|
| `api_key` | Yes | - | Your OpenRouter API key |
| `api_base` | No | `https://openrouter.ai/api/v1` | API endpoint |

## Popular Models

| Model | Model Name | Notes |
|-------|------------|-------|
| Claude Opus 4.5 | `anthropic/claude-opus-4-5` | Most capable |
| Claude Sonnet 4 | `anthropic/claude-sonnet-4` | Fast + capable |
| GPT-4o | `openai/gpt-4o` | OpenAI flagship |
| GPT-4o Mini | `openai/gpt-4o-mini` | Fast, cheaper |
| Llama 3.3 70B | `meta-llama/llama-3.3-70b-instruct` | Open model |
| Gemini 2.0 Flash | `google/gemini-2.0-flash-001` | Fast Google model |

## Model Format

Use the format `provider/model-name`:

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

Set up fallbacks for reliability:

```json
{
  "agents": {
    "defaults": {
      "model": "anthropic/claude-opus-4-5",
      "model_fallbacks": [
        "anthropic/claude-sonnet-4",
        "openai/gpt-4o"
      ]
    }
  }
}
```

## Usage Examples

```bash
# Use default model
picoclaw agent -m "Hello!"

# With specific model
picoclaw agent -m "Hello!" # Uses config model
```

## Pricing

Check [openrouter.ai/models](https://openrouter.ai/models) for current pricing.

## Environment Variable

```bash
export PICOCLAW_PROVIDERS_OPENROUTER_API_KEY="sk-or-v1-xxx"
```

## See Also

- [Providers Overview](README.md)
- [Configuration Reference](../../configuration/config-file.md)

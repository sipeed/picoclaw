# Gemini Provider

Use Google's Gemini models through the Gemini API.

## Why Gemini?

- **Free tier**: Generous free tier for development
- **Large context**: Up to 1M tokens context window
- **Multimodal**: Native support for text, images, audio, video
- **Fast**: Quick response times

## Setup

### Step 1: Get API Key

1. Go to [aistudio.google.com](https://aistudio.google.com)
2. Sign in with your Google account
3. Click "Get API Key"
4. Create a new API key
5. Copy the key

### Step 2: Configure

Edit `~/.picoclaw/config.json`:

```json
{
  "agents": {
    "defaults": {
      "model": "gemini-2.0-flash"
    }
  },
  "providers": {
    "gemini": {
      "api_key": "YOUR-API-KEY",
      "api_base": "https://generativelanguage.googleapis.com/v1beta"
    }
  }
}
```

## Configuration Options

| Option | Required | Default | Description |
|--------|----------|---------|-------------|
| `api_key` | Yes | - | Your Gemini API key |
| `api_base` | No | `https://generativelanguage.googleapis.com/v1beta` | API endpoint |

## Available Models

| Model | Model Name | Best For |
|-------|------------|----------|
| Gemini 2.0 Flash | `gemini-2.0-flash` | Fast, efficient |
| Gemini 2.0 Pro | `gemini-2.0-pro-exp` | Experimental, advanced |
| Gemini 1.5 Pro | `gemini-1.5-pro` | Large context, complex tasks |
| Gemini 1.5 Flash | `gemini-1.5-flash` | Fast, cost-effective |
| Gemini 1.5 Flash-8B | `gemini-1.5-flash-8b` | Smallest, fastest |

## Model Capabilities

| Model | Context | Multimodal | Best Use |
|-------|---------|------------|----------|
| Gemini 2.0 Flash | 1M tokens | Yes | General use |
| Gemini 1.5 Pro | 2M tokens | Yes | Long documents |
| Gemini 1.5 Flash | 1M tokens | Yes | Fast responses |

## Model Format

Use the model name directly:

```json
{
  "agents": {
    "defaults": {
      "model": "gemini-2.0-flash"
    }
  }
}
```

Or with provider prefix:

```json
{
  "agents": {
    "defaults": {
      "model": "gemini/gemini-2.0-flash"
    }
  }
}
```

## Fallback Configuration

```json
{
  "agents": {
    "defaults": {
      "model": "gemini-2.0-flash",
      "model_fallbacks": [
        "gemini-1.5-flash",
        "gemini-1.5-flash-8b"
      ]
    }
  }
}
```

## Environment Variable

```bash
export PICOCLAW_PROVIDERS_GEMINI_API_KEY="your-api-key"
```

## Pricing

Check [ai.google.dev/pricing](https://ai.google.dev/pricing) for current pricing.

### Free Tier

| Model | Requests per Day | Tokens per Minute |
|-------|------------------|-------------------|
| Gemini 2.0 Flash | 1,500 | 1M |
| Gemini 1.5 Flash | 1,500 | 1M |
| Gemini 1.5 Pro | 50 | 32K |

### Paid Tier

Higher limits available with billing enabled.

## Usage Examples

```bash
# Use default model
picoclaw agent -m "Hello!"

# Gemini handles long contexts well
picoclaw agent -m "Summarize this 100-page document..."
```

## Special Features

### Large Context Window

Gemini supports up to 2M token context, ideal for:

- Long document analysis
- Multi-document summarization
- Extended conversation history
- Large codebase analysis

### Multimodal Input

Gemini can process:

- Text
- Images
- Audio
- Video

Configure for vision:

```json
{
  "agents": {
    "defaults": {
      "model": "gemini-2.0-flash",
      "vision": true
    }
  }
}
```

### Safety Settings

Configure content filtering:

```json
{
  "providers": {
    "gemini": {
      "api_key": "your-key",
      "safety_settings": {
        "harassment": "BLOCK_MEDIUM_AND_ABOVE",
        "hate_speech": "BLOCK_MEDIUM_AND_ABOVE",
        "sexually_explicit": "BLOCK_MEDIUM_AND_ABOVE",
        "dangerous_content": "BLOCK_MEDIUM_AND_ABOVE"
      }
    }
  }
}
```

## Troubleshooting

### Invalid API Key

```
Error: API key not valid
```

Verify your API key:
1. Check key is correctly copied
2. Ensure key is active in AI Studio
3. Try regenerating the key

### Resource Exhausted

```
Error: RESOURCE_EXHAUSTED
```

Solutions:
1. Wait and retry (rate limit)
2. Check daily quota
3. Enable billing for higher limits

### Content Filtered

```
Error: content was blocked
```

Solutions:
1. Adjust safety settings
2. Rephrase your prompt
3. Use a different model

### Context Too Long

Even with large context, you may hit limits:

1. Reduce context size
2. Use document summarization
3. Split into multiple requests

## Using via OpenRouter

Alternatively, access Gemini through OpenRouter:

```json
{
  "providers": {
    "openrouter": {
      "api_key": "sk-or-v1-xxx"
    }
  },
  "agents": {
    "defaults": {
      "model": "google/gemini-2.0-flash-001"
    }
  }
}
```

## See Also

- [Providers Overview](README.md)
- [Configuration Reference](../../configuration/config-file.md)
- [Model Fallbacks](../advanced/model-fallbacks.md)

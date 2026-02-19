# Groq Provider

Use Groq's ultra-fast inference for LLMs and voice transcription.

## Why Groq?

- **Extremely fast**: Sub-second response times
- **Free tier**: Generous free tier available
- **Voice transcription**: Whisper-large-v3 for audio
- **Open models**: Llama, Mixtral, and more

## Setup

### Step 1: Get API Key

1. Go to [console.groq.com](https://console.groq.com)
2. Sign in or create an account
3. Navigate to API Keys
4. Create a new API key
5. Copy the key (starts with `gsk_`)

### Step 2: Configure

Edit `~/.picoclaw/config.json`:

```json
{
  "agents": {
    "defaults": {
      "model": "llama-3.3-70b-versatile"
    }
  },
  "providers": {
    "groq": {
      "api_key": "gsk_YOUR-API-KEY",
      "api_base": "https://api.groq.com/openai/v1"
    }
  }
}
```

## Configuration Options

| Option | Required | Default | Description |
|--------|----------|---------|-------------|
| `api_key` | Yes | - | Your Groq API key |
| `api_base` | No | `https://api.groq.com/openai/v1` | API endpoint |

## Available Models

### Chat Models

| Model | Model Name | Best For |
|-------|------------|----------|
| Llama 3.3 70B | `llama-3.3-70b-versatile` | General use, balanced |
| Llama 3.3 70B Specdec | `llama-3.3-70b-specdec` | Speculative decoding, faster |
| Llama 3.1 8B | `llama-3.1-8b-instant` | Fast, simple tasks |
| Mixtral 8x7B | `mixtral-8x7b-32768` | Multilingual, 32K context |
| Gemma 2 9B | `gemma2-9b-it` | Lightweight, efficient |

### Audio Models

| Model | Model Name | Use |
|-------|------------|-----|
| Whisper Large v3 | `whisper-large-v3` | Audio transcription |
| Whisper Large v3 Turbo | `whisper-large-v3-turbo` | Faster transcription |

## Model Format

Use the model name directly:

```json
{
  "agents": {
    "defaults": {
      "model": "llama-3.3-70b-versatile"
    }
  }
}
```

Or with provider prefix:

```json
{
  "agents": {
    "defaults": {
      "model": "groq/llama-3.3-70b-versatile"
    }
  }
}
```

## Fallback Configuration

```json
{
  "agents": {
    "defaults": {
      "model": "llama-3.3-70b-versatile",
      "model_fallbacks": [
        "llama-3.1-8b-instant",
        "mixtral-8x7b-32768"
      ]
    }
  }
}
```

## Environment Variable

```bash
export PICOCLAW_PROVIDERS_GROQ_API_KEY="gsk_xxx"
```

## Pricing

Groq offers a generous free tier.

### Free Tier

| Resource | Limit |
|----------|-------|
| Requests per minute | 30 |
| Tokens per minute | 18,000 |
| Requests per day | 14,400 |

### Paid Tier

Higher limits available with paid plans. Check [groq.com/pricing](https://groq.com/pricing) for details.

## Voice Transcription

Groq provides ultra-fast Whisper transcription.

### Enable Voice Transcription

```json
{
  "providers": {
    "groq": {
      "api_key": "gsk_xxx"
    }
  },
  "agents": {
    "defaults": {
      "voice_transcription": {
        "enabled": true,
        "provider": "groq",
        "model": "whisper-large-v3"
      }
    }
  }
}
```

### Supported Formats

- MP3
- MP4
- MPEG
- MPGA
- M4A
- WAV
- WebM

### Use with Telegram

Voice messages in Telegram will be automatically transcribed:

```json
{
  "channels": {
    "telegram": {
      "enabled": true,
      "token": "your-bot-token"
    }
  },
  "agents": {
    "defaults": {
      "voice_transcription": {
        "enabled": true,
        "provider": "groq"
      }
    }
  }
}
```

## Usage Examples

```bash
# Use default model
picoclaw agent -m "Hello!"

# Groq is extremely fast
picoclaw agent -m "Write a quick summary of..."
```

## Performance

Groq uses custom LPU (Language Processing Unit) hardware:

| Model | Typical Latency |
|-------|-----------------|
| Llama 3.3 70B | < 500ms |
| Llama 3.1 8B | < 200ms |
| Whisper Large | Real-time |

## Troubleshooting

### Invalid API Key

```
Error: invalid_api_key
```

Verify your API key:
1. Check key starts with `gsk_`
2. Ensure key is active
3. Check for typos

### Rate Limited

```
Error: rate_limit_exceeded
```

Solutions:
1. Wait and retry
2. Set up fallback models
3. Upgrade to paid tier

### Model Not Available

```
Error: model_not_found
```

Check available models at [console.groq.com/models](https://console.groq.com/models)

### Audio File Too Large

```
Error: file_too_large
```

Audio file limits:
- Maximum file size: 25MB
- Maximum duration: ~30 minutes

## Best Practices

### For Chat

1. Use `llama-3.3-70b-versatile` for best quality
2. Use `llama-3.1-8b-instant` for simple, fast tasks
3. Set up fallbacks for reliability

### For Voice

1. Use `whisper-large-v3-turbo` for faster transcription
2. Use `whisper-large-v3` for better accuracy
3. Ensure audio is clear for best results

## See Also

- [Providers Overview](README.md)
- [Configuration Reference](../../configuration/config-file.md)
- [Voice Transcription](../advanced/voice-transcription.md)

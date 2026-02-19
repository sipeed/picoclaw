# DeepSeek Provider

Use DeepSeek's models for reasoning and general tasks.

## Why DeepSeek?

- **Strong reasoning**: Excellent at complex reasoning tasks
- **Cost-effective**: Competitive pricing
- **Long context**: Up to 64K context
- **Code generation**: Strong coding capabilities

## Setup

### Step 1: Get API Key

1. Go to [platform.deepseek.com](https://platform.deepseek.com)
2. Sign in or create an account
3. Navigate to API Keys
4. Create a new API key
5. Copy the key (starts with `sk-`)

### Step 2: Configure

Edit `~/.picoclaw/config.json`:

```json
{
  "agents": {
    "defaults": {
      "model": "deepseek-chat"
    }
  },
  "providers": {
    "deepseek": {
      "api_key": "sk-YOUR-API-KEY",
      "api_base": "https://api.deepseek.com"
    }
  }
}
```

## Configuration Options

| Option | Required | Default | Description |
|--------|----------|---------|-------------|
| `api_key` | Yes | - | Your DeepSeek API key |
| `api_base` | No | `https://api.deepseek.com` | API endpoint |

## Available Models

| Model | Model Name | Best For |
|-------|------------|----------|
| DeepSeek Chat | `deepseek-chat` | General conversation |
| DeepSeek Reasoner | `deepseek-reasoner` | Complex reasoning |

### Model Comparison

| Feature | deepseek-chat | deepseek-reasoner |
|---------|---------------|-------------------|
| Context | 64K | 64K |
| Best for | General tasks | Complex reasoning |
| Speed | Fast | Slower (thinks more) |

## Model Format

Use the model name directly:

```json
{
  "agents": {
    "defaults": {
      "model": "deepseek-chat"
    }
  }
}
```

Or with provider prefix:

```json
{
  "agents": {
    "defaults": {
      "model": "deepseek/deepseek-chat"
    }
  }
}
```

## Fallback Configuration

```json
{
  "agents": {
    "defaults": {
      "model": "deepseek-reasoner",
      "model_fallbacks": [
        "deepseek-chat"
      ]
    }
  }
}
```

## Environment Variable

```bash
export PICOCLAW_PROVIDERS_DEEPSEEK_API_KEY="sk-xxx"
```

## Pricing

Check [platform.deepseek.com/pricing](https://platform.deepseek.com/pricing) for current pricing.

Approximate costs (per 1M tokens):

| Model | Input | Output |
|-------|-------|--------|
| deepseek-chat | $0.14 | $0.28 |
| deepseek-reasoner | $0.55 | $2.19 |

## Usage Examples

```bash
# General chat
picoclaw agent -m "Hello!"

# Complex reasoning (with reasoner model)
picoclaw agent -m "Solve this step by step: What is 15% of 847?"
```

## DeepSeek Reasoner

The `deepseek-reasoner` model is optimized for complex reasoning:

- Shows thinking process
- Better at math and logic
- Good for analysis tasks

### When to Use Reasoner

- Mathematical problems
- Logical reasoning
- Multi-step analysis
- Complex decision making

### Example

```
User: If I have 5 apples and give away 2, then buy 3 more, how many do I have?

Agent (using reasoner):
Let me work through this step by step:
1. Start with 5 apples
2. Give away 2: 5 - 2 = 3 apples
3. Buy 3 more: 3 + 3 = 6 apples

You have 6 apples.
```

## Troubleshooting

### Invalid API Key

```
Error: Authentication Error
```

Verify your API key:
1. Check key is correctly copied
2. Ensure key is active in platform
3. Check for typos

### Rate Limited

```
Error: Rate limit exceeded
```

Solutions:
1. Wait and retry
2. Set up fallback models
3. Check your usage limits

### Insufficient Balance

```
Error: Insufficient balance
```

Add funds to your DeepSeek account at [platform.deepseek.com](https://platform.deepseek.com).

### Context Length Exceeded

```
Error: context_length_exceeded
```

Solutions:
1. Reduce message length
2. Clear session history
3. Start a new session

## Best Practices

1. **Use chat for general tasks** - Faster and more efficient
2. **Use reasoner for complex problems** - Better reasoning capability
3. **Set up fallbacks** - For reliability
4. **Monitor usage** - Track token consumption

## Using via OpenRouter

Alternatively, access DeepSeek through OpenRouter:

```json
{
  "providers": {
    "openrouter": {
      "api_key": "sk-or-v1-xxx"
    }
  },
  "agents": {
    "defaults": {
      "model": "deepseek/deepseek-chat"
    }
  }
}
```

## See Also

- [Providers Overview](README.md)
- [Configuration Reference](../../configuration/config-file.md)
- [Model Fallbacks](../advanced/model-fallbacks.md)

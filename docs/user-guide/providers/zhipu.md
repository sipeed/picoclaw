# Zhipu Provider

Zhipu (智谱) provides Chinese-optimized AI models through their GLM series.

## Why Zhipu?

- **Chinese language**: Optimized for Chinese users
- **Free tier**: 200K tokens per month
- **Multiple models**: GLM-4.7, GLM-4-Plus, and more
- **Direct access**: No intermediary

## Setup

### Step 1: Get API Key

1. Go to [bigmodel.cn](https://open.bigmodel.cn/usercenter/proj-mgmt/apikeys)
2. Register or sign in
3. Create an API key
4. Copy the key

### Step 2: Configure

Edit `~/.picoclaw/config.json`:

```json
{
  "agents": {
    "defaults": {
      "model": "glm-4.7"
    }
  },
  "providers": {
    "zhipu": {
      "api_key": "YOUR_ZHIPU_API_KEY",
      "api_base": "https://open.bigmodel.cn/api/paas/v4"
    }
  }
}
```

## Configuration Options

| Option | Required | Default | Description |
|--------|----------|---------|-------------|
| `api_key` | Yes | - | Your Zhipu API key |
| `api_base` | No | `https://open.bigmodel.cn/api/paas/v4` | API endpoint |

## Available Models

| Model | Description |
|-------|-------------|
| `glm-4.7` | Latest GLM model (default) |
| `glm-4-plus` | Enhanced capabilities |
| `glm-4` | Standard GLM-4 |
| `glm-4-air` | Faster, lighter model |
| `glm-4-flash` | Fastest model |

## Model Format

Use just the model name (no provider prefix needed):

```json
{
  "agents": {
    "defaults": {
      "model": "glm-4.7"
    }
  }
}
```

## Features

### Chinese Language Support

Zhipu models are optimized for:
- Chinese conversations
- Chinese document processing
- Chinese cultural context

### Multi-modal

Some Zhipu models support:
- Text generation
- Image understanding
- Code generation

## Content Filtering

Zhipu has content filtering. If you encounter filtering errors:

1. Rephrase your query
2. Avoid sensitive topics
3. Use a different model

## Environment Variable

```bash
export PICOCLAW_PROVIDERS_ZHIPU_API_KEY="your-zhipu-key"
```

## Usage Examples

```bash
# Use GLM-4.7
picoclaw agent -m "你好，请介绍一下自己"

# Code generation
picoclaw agent -m "写一个Python快速排序"
```

## Pricing

Check [bigmodel.cn](https://open.bigmodel.cn) for current pricing.

## See Also

- [Providers Overview](README.md)
- [Configuration Reference](../../configuration/config-file.md)

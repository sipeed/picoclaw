# Providers Overview

PicoClaw supports multiple LLM providers. This guide helps you choose and configure the right provider.

## Quick Comparison

| Provider | Free Tier | Best For | Auth Method |
|----------|-----------|----------|-------------|
| [OpenRouter](openrouter.md) | 200K tokens/mo | Multi-model access | API Key |
| [Zhipu](zhipu.md) | 200K tokens/mo | Chinese users | API Key |
| [Anthropic](anthropic.md) | Varies | Claude models | API Key, OAuth |
| [OpenAI](openai.md) | Varies | GPT models | API Key, OAuth |
| [Gemini](gemini.md) | Yes | Google models | API Key |
| [Groq](groq.md) | Yes | Fast inference | API Key |
| [DeepSeek](deepseek.md) | Yes | Reasoning | API Key |
| [Ollama](ollama.md) | Free | Local models | None |
| [vLLM](vllm.md) | Free | Self-hosted | API Key |

## Recommended Providers

### For Most Users: OpenRouter

OpenRouter provides access to many models (Claude, GPT, Llama, etc.) with a single API key.

```json
{
  "agents": { "defaults": { "model": "anthropic/claude-opus-4-5" } },
  "providers": {
    "openrouter": {
      "api_key": "sk-or-v1-xxx"
    }
  }
}
```

### For Chinese Users: Zhipu

Zhipu offers Chinese-optimized models with good free tier.

```json
{
  "agents": { "defaults": { "model": "glm-4.7" } },
  "providers": {
    "zhipu": {
      "api_key": "your-zhipu-key"
    }
  }
}
```

### For Local/Offline: Ollama

Run models locally without internet.

```json
{
  "agents": { "defaults": { "model": "llama3.2" } },
  "providers": {
    "ollama": {
      "api_base": "http://localhost:11434/v1"
    }
  }
}
```

## Provider Selection

PicoClaw selects providers in this order:

1. **Explicit provider** in config:
   ```json
   "provider": "openrouter"
   ```

2. **Model prefix**:
   ```json
   "model": "openrouter/anthropic/claude-opus-4-5"
   ```

3. **First API key found** (default behavior)

## Model Fallbacks

Configure automatic fallback if primary model fails:

```json
{
  "agents": {
    "defaults": {
      "model": "claude-opus-4-5",
      "model_fallbacks": ["gpt-4o", "glm-4.7"]
    }
  }
}
```

## Authentication Methods

### API Key (Recommended)

Simplest method - add your key to config:

```json
{
  "providers": {
    "openrouter": {
      "api_key": "sk-or-v1-xxx"
    }
  }
}
```

### Environment Variable

Override config with environment variables:

```bash
export PICOCLAW_PROVIDERS_OPENROUTER_API_KEY="sk-or-v1-xxx"
```

### OAuth

For OpenAI and Anthropic, use OAuth login:

```bash
picoclaw auth login --provider openai
```

## Getting API Keys

| Provider | Get API Key |
|----------|-------------|
| OpenRouter | [openrouter.ai/keys](https://openrouter.ai/keys) |
| Zhipu | [bigmodel.cn](https://open.bigmodel.cn/usercenter/proj-mgmt/apikeys) |
| Anthropic | [console.anthropic.com](https://console.anthropic.com) |
| OpenAI | [platform.openai.com](https://platform.openai.com) |
| Gemini | [aistudio.google.com](https://aistudio.google.com) |
| Groq | [console.groq.com](https://console.groq.com) |
| DeepSeek | [platform.deepseek.com](https://platform.deepseek.com) |

## Provider-Specific Guides

- [OpenRouter](openrouter.md) - Multi-model access
- [Zhipu](zhipu.md) - Chinese AI models
- [Anthropic](anthropic.md) - Claude models
- [OpenAI](openai.md) - GPT models
- [Gemini](gemini.md) - Google Gemini
- [Groq](groq.md) - Fast inference + voice
- [DeepSeek](deepseek.md) - DeepSeek models
- [Ollama](ollama.md) - Local models
- [vLLM](vllm.md) - Self-hosted models

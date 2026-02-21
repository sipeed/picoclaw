# Quick Start

Get PicoClaw running in 5 minutes with this step-by-step guide.

## Prerequisites

- PicoClaw installed (see [Installation](installation.md))
- An API key from an LLM provider

## Step 1: Initialize

Run the onboard command to set up the default configuration:

```bash
picoclaw onboard
```

This creates:
- `~/.picoclaw/config.json` - Main configuration file
- `~/.picoclaw/workspace/` - Working directory for the agent

## Step 2: Get API Keys

You need at least one LLM provider API key.

### Recommended: OpenRouter

[OpenRouter](https://openrouter.ai/keys) provides access to many models with a free tier.

1. Go to [openrouter.ai/keys](https://openrouter.ai/keys)
2. Sign in and create an API key
3. Copy the key (starts with `sk-or-v1-`)

### Alternative: Zhipu (Chinese Users)

[Zhipu](https://open.bigmodel.cn/usercenter/proj-mgmt/apikeys) offers Chinese AI models.

1. Go to [bigmodel.cn](https://open.bigmodel.cn/usercenter/proj-mgmt/apikeys)
2. Register and create an API key
3. Copy the key

### Other Providers

| Provider | Get API Key | Notes |
|----------|-------------|-------|
| [Anthropic](https://console.anthropic.com) | Claude models | |
| [OpenAI](https://platform.openai.com) | GPT models | |
| [Gemini](https://aistudio.google.com) | Google models | Free tier available |
| [Groq](https://console.groq.com) | Fast inference | Free tier + voice transcription |
| [DeepSeek](https://platform.deepseek.com) | DeepSeek models | |

## Step 3: Configure

Edit `~/.picoclaw/config.json` and add your API key:

### Using OpenRouter

```json
{
  "agents": {
    "defaults": {
      "workspace": "~/.picoclaw/workspace",
      "model": "anthropic/claude-opus-4-5",
      "max_tokens": 8192,
      "temperature": 0.7,
      "max_tool_iterations": 20
    }
  },
  "providers": {
    "openrouter": {
      "api_key": "sk-or-v1-YOUR-API-KEY-HERE",
      "api_base": "https://openrouter.ai/api/v1"
    }
  }
}
```

### Using Zhipu

```json
{
  "agents": {
    "defaults": {
      "workspace": "~/.picoclaw/workspace",
      "model": "glm-4.7",
      "max_tokens": 8192,
      "temperature": 0.7,
      "max_tool_iterations": 20
    }
  },
  "providers": {
    "zhipu": {
      "api_key": "YOUR-ZHIPU-API-KEY-HERE",
      "api_base": "https://open.bigmodel.cn/api/paas/v4"
    }
  }
}
```

## Step 4: Chat

Now you're ready to chat with PicoClaw!

### One-shot Message

```bash
picoclaw agent -m "What is 2+2?"
```

### Interactive Mode

```bash
picoclaw agent
```

This starts an interactive chat session where you can have a continuous conversation.

### Debug Mode

For troubleshooting:

```bash
picoclaw agent --debug -m "Hello"
```

## Optional: Enable Web Search

PicoClaw can search the web. DuckDuckGo is enabled by default (no API key needed).

For better results, get a free [Brave Search API](https://brave.com/search/api) key (2000 queries/month):

```json
{
  "tools": {
    "web": {
      "brave": {
        "enabled": true,
        "api_key": "YOUR_BRAVE_API_KEY",
        "max_results": 5
      },
      "duckduckgo": {
        "enabled": true,
        "max_results": 5
      }
    }
  }
}
```

## Optional: Connect to Chat Apps

To use PicoClaw with Telegram, Discord, or other chat platforms:

### Telegram

1. Create a bot via [@BotFather](https://t.me/BotFather)
2. Get your user ID from [@userinfobot](https://t.me/userinfobot)
3. Configure:

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

4. Start the gateway:

```bash
picoclaw gateway
```

See [Telegram Setup](../user-guide/channels/telegram.md) for detailed instructions.

### Discord

See [Discord Setup](../user-guide/channels/discord.md) for detailed instructions.

## Next Steps

- [Configuration Basics](configuration-basics.md) - Learn about all config options
- [CLI Reference](../user-guide/cli-reference.md) - All commands and flags
- [Workspace Guide](../user-guide/workspace/structure.md) - Customize agent behavior
- [Tools Overview](../user-guide/tools/README.md) - Available tools

## Troubleshooting

### Web search shows "API configuration error"

This is normal without a Brave API key. DuckDuckGo fallback works automatically. To enable Brave search, add your API key to the config.

### Content filtering errors

Some providers have content filtering. Try:
- Rephrasing your query
- Using a different model
- Switching to a different provider

### Model not found

Make sure the model name is correct:
- OpenRouter: Use format like `anthropic/claude-opus-4-5`
- Zhipu: Use `glm-4.7`, `glm-4-plus`, etc.
- OpenAI: Use `gpt-4`, `gpt-4o`, etc.

See the [Troubleshooting Guide](../operations/troubleshooting.md) for more solutions.

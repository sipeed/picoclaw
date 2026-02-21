# Basic Assistant Setup Tutorial

This tutorial guides you through setting up your first PicoClaw AI assistant.

## Prerequisites

- 10 minutes
- A computer running Linux, macOS, or Windows
- Internet connection
- An API key from an LLM provider

## Step 1: Install PicoClaw

### Download Binary (Recommended)

```bash
# Download the latest release for your platform
# Linux (amd64)
curl -LO https://github.com/sipeed/picoclaw/releases/latest/download/picoclaw-linux-amd64

# macOS (arm64)
curl -LO https://github.com/sipeed/picoclaw/releases/latest/download/picoclaw-darwin-arm64

# Windows (amd64)
curl -LO https://github.com/sipeed/picoclaw/releases/latest/download/picoclaw-windows-amd64.exe
```

### Make Executable

```bash
# Linux/macOS
chmod +x picoclaw-*

# Move to PATH (optional)
sudo mv picoclaw-* /usr/local/bin/picoclaw
```

### Verify Installation

```bash
picoclaw version
```

## Step 2: Get an API Key

### Recommended: OpenRouter

OpenRouter provides access to many models with a free tier (200K tokens/month).

1. Go to [openrouter.ai/keys](https://openrouter.ai/keys)
2. Sign in or create an account
3. Click "Create Key"
4. Copy your API key (starts with `sk-or-v1-`)

### Alternative Providers

| Provider | Sign Up | Free Tier |
|----------|---------|-----------|
| Zhipu | [bigmodel.cn](https://open.bigmodel.cn) | 200K tokens |
| Groq | [console.groq.com](https://console.groq.com) | Yes |
| Gemini | [aistudio.google.com](https://aistudio.google.com) | Yes |

## Step 3: Initialize Configuration

Run the onboard command to create default configuration:

```bash
picoclaw onboard
```

This creates:
- `~/.picoclaw/config.json` - Main configuration file
- `~/.picoclaw/workspace/` - Working directory

## Step 4: Add Your API Key

Edit the configuration file:

```bash
nano ~/.picoclaw/config.json
```

Add your OpenRouter API key:

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

Replace `YOUR-API-KEY-HERE` with your actual API key.

## Step 5: Your First Chat

### One-Shot Message

Send a single message:

```bash
picoclaw agent -m "Hello, what can you help me with?"
```

Expected output:

```
Hello! I'm your PicoClaw assistant. I can help you with:

- Answering questions and providing information
- Reading, writing, and managing files
- Running shell commands
- Searching the web
- And much more!

What would you like to do today?
```

### Interactive Mode

Start a continuous conversation:

```bash
picoclaw agent
```

```
> Hello!
Hello! How can I help you today?

> What is 2 + 2?
2 + 2 = 4

> Can you write that to a file?
[Uses write_file tool]
I've written "4" to result.txt in the workspace.

> Thanks!
You're welcome! Let me know if you need anything else.

> exit
Goodbye!
```

## Step 6: Explore the Workspace

The workspace is where PicoClaw stores files and sessions:

```bash
# View workspace structure
ls -la ~/.picoclaw/workspace/
```

Output:

```
drwxr-xr-x  sessions/   # Conversation history
drwxr-xr-x  memory/     # Long-term memory
-rw-r--r--  AGENT.md     # Agent behavior guide
-rw-r--r--  IDENTITY.md  # Agent identity
```

### View Session History

```bash
ls ~/.picoclaw/workspace/sessions/
```

Sessions are stored as JSON files with your conversation history.

## Step 7: Customize Your Agent

### AGENT.md

The AGENT.md file defines how your agent behaves:

```bash
cat ~/.picoclaw/workspace/AGENT.md
```

Edit it to customize behavior:

```markdown
# Agent Behavior

You are a helpful AI assistant.

## Capabilities
- Answer questions
- Help with coding
- Manage files

## Personality
- Friendly and professional
- Clear and concise explanations
```

### IDENTITY.md

Define your agent's identity:

```markdown
# Identity

Name: PicoClaw Assistant
Version: 1.0
Created: 2024-01-15

I am an AI assistant powered by PicoClaw.
```

## Step 8: Enable Web Search (Optional)

PicoClaw can search the web. DuckDuckGo works without configuration.

For better results, add Brave Search:

1. Get a free API key from [brave.com/search/api](https://brave.com/search/api) (2000 queries/month)

2. Add to config:

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

## Step 9: Try Web Search

```bash
picoclaw agent -m "What's the latest news about AI?"
```

The agent will search the web and summarize relevant information.

## Understanding Tools

PicoClaw has built-in tools the agent can use:

| Tool | Description |
|------|-------------|
| `read_file` | Read file contents |
| `write_file` | Write to files |
| `list_files` | List directory contents |
| `exec` | Run shell commands |
| `web_search` | Search the web |
| `web_fetch` | Fetch web page content |

### Example: File Operations

```bash
picoclaw agent -m "Create a file called notes.txt with today's date"
```

The agent will:
1. Use `write_file` to create the file
2. Confirm the action

### Example: Shell Commands

```bash
picoclaw agent -m "What's the current system time?"
```

The agent will:
1. Use `exec` to run `date`
2. Report the time

## Debug Mode

When troubleshooting, use debug mode:

```bash
picoclaw agent --debug -m "Hello"
```

Debug mode shows:
- LLM requests and responses
- Tool calls and results
- Configuration details

## Common Issues

### "no API key configured"

Make sure your config file has the API key:

```bash
cat ~/.picoclaw/config.json | grep api_key
```

### "model not found"

Check the model name format:
- OpenRouter: `anthropic/claude-opus-4-5`
- Zhipu: `glm-4.7`

### "rate limit exceeded"

Wait a moment and try again, or set up fallback models:

```json
{
  "agents": {
    "defaults": {
      "model": "anthropic/claude-opus-4-5",
      "model_fallbacks": ["anthropic/claude-sonnet-4", "openai/gpt-4o"]
    }
  }
}
```

## Next Steps

Now that you have a basic assistant running, explore:

- [Telegram Bot Tutorial](telegram-bot.md) - Connect to Telegram
- [Scheduled Tasks Tutorial](scheduled-tasks.md) - Automate tasks
- [CLI Reference](../user-guide/cli-reference.md) - All commands
- [Tools Overview](../user-guide/tools/README.md) - Available tools

## Summary

You learned:
- How to install PicoClaw
- How to configure an LLM provider
- How to chat with the agent
- How to customize behavior
- How to enable web search
- Basic troubleshooting

Congratulations! You have a working AI assistant.

# IDE Setup

PicoClaw can integrate with various IDEs and development environments through different providers. This section covers how to set up PicoClaw to work with your preferred development workflow.

## Available IDE Integrations

| Integration | Type | Description |
|-------------|------|-------------|
| **Antigravity** | Google Cloud Code Assist | Free access to Claude and Gemini models via Google Cloud |
| **GitHub Copilot** | GitHub | Use GitHub Copilot models with PicoClaw |

## Antigravity (Google Cloud Code Assist)

Antigravity provides free access to powerful AI models (Claude Opus, Gemini) through Google Cloud's infrastructure. This is ideal for developers who want to use PicoClaw without managing multiple API keys.

**Key Features:**
- Free tier with generous quotas
- Access to Claude Opus 4.6 and Gemini models
- OAuth authentication (no API key management)
- Usage tracking and quota management

**Setup Guide:** [Antigravity Setup](antigravity.md)

## Configuration

Once you've set up an IDE provider, configure it in your `~/.picoclaw/config.json`:

```json
{
  "model_list": [
    {
      "model_name": "gemini-flash",
      "model": "antigravity/gemini-3-flash",
      "auth_method": "oauth"
    }
  ],
  "agents": {
    "defaults": {
      "model": "gemini-flash"
    }
  }
}
```

## Authentication

IDE integrations typically use OAuth authentication instead of API keys:

```bash
# Authenticate with Antigravity
picoclaw auth login --provider antigravity

# List available models
picoclaw auth models
```

## Related Documentation

- [Provider Configuration](../providers/README.md)
- [Model List Configuration](../advanced/model-fallbacks.md)
- [CLI Reference](../cli-reference.md)

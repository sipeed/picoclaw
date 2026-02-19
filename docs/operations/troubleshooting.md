# Troubleshooting

This guide covers common issues and their solutions when using PicoClaw.

## Installation Issues

### Binary won't execute (Permission denied)

**Error:**
```
bash: ./picoclaw-*: Permission denied
```

**Solution:**
```bash
chmod +x picoclaw-*
```

### Command not found

**Error:**
```
bash: picoclaw: command not found
```

**Solution:**

Make sure the binary is in your PATH:
```bash
# Add current directory to PATH temporarily
export PATH="$PWD:$PATH"

# Or move to a standard location
sudo mv picoclaw-* /usr/local/bin/picoclaw
```

### Go version too old

**Error:**
```
go: cannot use Go 1.20, requires Go 1.21 or later
```

**Solution:**
```bash
# Check your Go version
go version

# Upgrade Go to 1.21 or later from https://go.dev/dl/
```

### Docker issues

**Error:**
```
Cannot connect to the Docker daemon
```

**Solution:**
Ensure Docker is running and you have permissions:
```bash
# Start Docker service
sudo systemctl start docker

# Add your user to docker group (requires logout/login)
sudo usermod -aG docker $USER
```

## Configuration Issues

### Config file not found

**Error:**
```
Error loading config: open ~/.picoclaw/config.json: no such file or directory
```

**Solution:**
Run the onboard command to create the default configuration:
```bash
picoclaw onboard
```

### Invalid JSON in config

**Error:**
```
Error loading config: invalid character '}' looking for beginning of value
```

**Solution:**
Validate your JSON syntax. Common issues:
- Missing commas between fields
- Trailing commas
- Unquoted strings

Use a JSON validator or:
```bash
cat ~/.picoclaw/config.json | python3 -m json.tool
```

## Provider Issues

### No API key configured

**Error:**
```
Error creating provider: no API key configured
```

**Solution:**
Add your API key to `~/.picoclaw/config.json`:
```json
{
  "providers": {
    "openrouter": {
      "api_key": "sk-or-v1-xxx"
    }
  }
}
```

Or use environment variable:
```bash
export PICOCLAW_PROVIDERS_OPENROUTER_API_KEY="sk-or-v1-xxx"
```

### Model not found

**Error:**
```
Error: model 'xxx' not found
```

**Solution:**
Use the correct model name format:
- OpenRouter: `anthropic/claude-opus-4-5`, `openai/gpt-4o`
- Zhipu: `glm-4.7`, `glm-4-plus`
- OpenAI: `gpt-4`, `gpt-4o`
- Anthropic: `claude-opus-4-5`, `claude-sonnet-4`

### Rate limit exceeded

**Error:**
```
Error: rate limit exceeded
```

**Solution:**
1. Wait and retry
2. Set up fallback models:
```json
{
  "agents": {
    "defaults": {
      "model": "primary-model",
      "model_fallbacks": ["backup-model-1", "backup-model-2"]
    }
  }
}
```

### Content filtering errors

**Error:**
```
Error: content filtered
```

**Solution:**
Some providers have content filtering. Try:
1. Rephrasing your query
2. Using a different model
3. Switching to a different provider

## Channel Issues

### Telegram: Conflict error

**Error:**
```
Conflict: terminated by other getUpdates request
```

**Solution:**
Only one instance can use the bot token at a time:
1. Stop other running instances of the bot
2. Make sure only one `picoclaw gateway` is running

### Telegram: Bot not responding

**Possible causes:**

1. Bot not enabled:
```json
{
  "channels": {
    "telegram": {
      "enabled": true
    }
  }
}
```

2. User not in allow list:
```json
{
  "channels": {
    "telegram": {
      "allow_from": ["123456789"]
    }
  }
}
```

3. Gateway not running:
```bash
picoclaw gateway
```

### Discord: Missing intents

**Error:**
```
Bot requires MESSAGE CONTENT intent
```

**Solution:**
1. Go to [Discord Developer Portal](https://discord.com/developers/applications)
2. Select your application → Bot
3. Enable "MESSAGE CONTENT INTENT"
4. Save changes and restart gateway

### LINE: Webhook verification failed

**Error:**
```
Webhook verification failed
```

**Solution:**
1. Ensure webhook URL is accessible (use ngrok for testing)
2. Verify webhook path matches config
3. Check channel secret is correct

## Tool Issues

### Web search: API configuration error

**Error:**
```
Web search: API configuration error
```

**Solution:**
This is normal without a Brave API key. DuckDuckGo works automatically.

To enable Brave Search:
```json
{
  "tools": {
    "web": {
      "brave": {
        "enabled": true,
        "api_key": "YOUR_BRAVE_API_KEY"
      }
    }
  }
}
```

### Exec: Command blocked

**Error:**
```
Command blocked by safety guard (dangerous pattern detected)
```

**Solution:**
Some dangerous commands are always blocked for safety:
- `rm -rf`, `del /f`, `rmdir /s`
- `format`, `mkfs`, `diskpart`
- `dd if=`
- `shutdown`, `reboot`, `poweroff`

If you need to run restricted commands in a trusted environment:
```json
{
  "agents": {
    "defaults": {
      "restrict_to_workspace": false
    }
  }
}
```

### Exec: Path outside working directory

**Error:**
```
Command blocked by safety guard (path outside working dir)
```

**Solution:**
The agent can only access files in the workspace. Either:
1. Move files into the workspace
2. Disable workspace restriction (security risk):
```bash
export PICOCLAW_AGENTS_DEFAULTS_RESTRICT_TO_WORKSPACE=false
```

## Session Issues

### Session history too large

**Symptom:**
Slow responses or memory issues.

**Solution:**
Sessions are automatically summarized when they exceed thresholds. You can also:
1. Delete old sessions: `rm ~/.picoclaw/workspace/sessions/*.json`
2. Start a new session: `picoclaw agent -s "new-session"`

### Session not persisting

**Solution:**
1. Check workspace directory exists: `ls ~/.picoclaw/workspace/sessions/`
2. Ensure write permissions: `chmod 755 ~/.picoclaw/workspace/sessions/`

## Performance Issues

### High memory usage

**Solution:**
1. Use a smaller model
2. Reduce `max_tokens`
3. Clear old sessions
4. Restart the gateway periodically

### Slow responses

**Possible causes:**

1. Large model - use a faster/smaller model
2. Network latency - use a provider closer to you
3. Rate limiting - set up fallback models
4. Complex tool calls - reduce tool iterations

### Gateway won't start

**Error:**
```
Error starting gateway: address already in use
```

**Solution:**
Change the gateway port:
```json
{
  "gateway": {
    "port": 18791
  }
}
```

## Authentication Issues

### OAuth login failed

**Solution:**
1. Try device code flow for headless environments:
```bash
picoclaw auth login --provider openai --device-code
```

2. Check system clock is synchronized

### Token expired

**Error:**
```
Token has expired
```

**Solution:**
```bash
picoclaw auth login --provider openai
```

## Debug Mode

Enable debug logging for detailed output:

```bash
# Agent debug mode
picoclaw agent --debug -m "Hello"

# Gateway debug mode
picoclaw gateway --debug
```

Debug mode shows:
- LLM requests and responses
- Tool calls and results
- Message bus activity
- Configuration loading

## Getting Help

If you can't find a solution:

1. **GitHub Issues**: [Report a bug](https://github.com/sipeed/picoclaw/issues)
2. **GitHub Discussions**: [Ask a question](https://github.com/sipeed/picoclaw/discussions)
3. **Discord**: [Join our server](https://discord.gg/V4sAZ9XWpN)

When reporting issues, include:
- PicoClaw version (`picoclaw version`)
- Operating system and architecture
- Debug output (use `--debug`)
- Steps to reproduce

## Related Documentation

- [Configuration Reference](../configuration/config-file.md)
- [CLI Reference](../user-guide/cli-reference.md)
- [Quick Start](../getting-started/quick-start.md)

# Getting Help

This guide covers how to get help with PicoClaw.

## Quick Help

### Common Issues

| Issue | Solution |
|-------|----------|
| Command not found | Add binary to PATH or use full path |
| No API key | Add provider API key to config |
| Permission denied | Check file permissions |
| Rate limited | Wait or set up fallback models |

See [Troubleshooting](../operations/troubleshooting.md) for more solutions.

## Documentation

### Start Here

1. [Installation Guide](../getting-started/installation.md)
2. [Quick Start](../getting-started/quick-start.md)
3. [Configuration Basics](../getting-started/configuration-basics.md)

### Reference

- [CLI Reference](../user-guide/cli-reference.md)
- [Configuration Reference](../configuration/config-file.md)
- [Tools Overview](../user-guide/tools/README.md)

### Tutorials

- [Basic Assistant](../tutorials/basic-assistant.md)
- [Telegram Bot](../tutorials/telegram-bot.md)
- [Scheduled Tasks](../tutorials/scheduled-tasks.md)

## Community Support

### Discord

Join our Discord server for real-time help:

[https://discord.gg/V4sAZ9XWpN](https://discord.gg/V4sAZ9XWpN)

**Channels:**

- `#general` - General discussion
- `#help` - Get help with issues
- `#showcase` - Share your projects
- `#announcements` - Project updates

### GitHub Discussions

For longer-form questions and feature discussions:

[https://github.com/sipeed/picoclaw/discussions](https://github.com/sipeed/picoclaw/discussions)

**Categories:**

- Q&A - Questions and answers
- Ideas - Feature suggestions
- Show and Tell - Project showcases
- General - Other discussions

## Reporting Issues

### Before Reporting

1. **Search existing issues** - Your problem may already be reported
2. **Check documentation** - Review relevant docs
3. **Try debug mode** - Run with `--debug` flag
4. **Update PicoClaw** - Ensure you're on the latest version

### How to Report

Open an issue at: [https://github.com/sipeed/picoclaw/issues](https://github.com/sipeed/picoclaw/issues)

### Issue Template

```markdown
## Description
Brief description of the problem

## Steps to Reproduce
1. Run command `picoclaw ...`
2. See error

## Expected Behavior
What should happen

## Actual Behavior
What actually happens

## Environment
- PicoClaw version: [run `picoclaw version`]
- OS: [e.g., Ubuntu 22.04, macOS 14]
- Architecture: [e.g., amd64, arm64]

## Debug Output
```
[paste debug output here, redact sensitive info]
```

## Additional Context
Any other relevant information
```

### What to Include

**Always Include:**

- PicoClaw version
- Operating system
- Steps to reproduce
- Error messages

**If Relevant:**

- Configuration file (redact API keys)
- Debug output
- Logs (redact sensitive info)
- Screenshots

### Redacting Sensitive Information

Before sharing logs or config:

```bash
# Redact API keys
sed -i 's/sk-[a-zA-Z0-9-]*/sk-***/g' config.json

# Redact tokens
sed -i 's/[0-9]\{10,\}:AAH[a-zA-Z0-9_-]*/TOKEN/g' log.txt
```

## Asking Good Questions

### Be Specific

```
Bad: "It doesn't work"
Good: "When I run `picoclaw agent -m 'hello'`, I get 'no API key configured' error"
```

### Provide Context

```
Bad: "How do I use Telegram?"
Good: "I'm trying to set up a Telegram bot. I've created the bot with BotFather and got the token. How do I configure it in PicoClaw?"
```

### Show Your Work

```
Bad: "The config is wrong"
Good: "Here's my config.json:
{
  "providers": {
    "openrouter": {
      "api_key": "sk-or-v1-***"
    }
  }
}

And I'm getting error: 'model not found'"
```

### One Issue at a Time

Focus on one problem per question or issue. This makes it easier to help and for others to find solutions later.

## Response Times

| Channel | Typical Response |
|---------|------------------|
| GitHub Issues | 1-3 days |
| GitHub Discussions | 1-3 days |
| Discord | Minutes to hours |

Response times vary based on complexity, timezone, and community availability.

## Self-Help Resources

### Debug Mode

Enable detailed logging:

```bash
picoclaw agent --debug -m "your message"
picoclaw gateway --debug
```

### Check Configuration

```bash
# Validate JSON syntax
cat ~/.picoclaw/config.json | python3 -m json.tool

# Check config is loaded
picoclaw status
```

### Check Logs

```bash
# If logging to file
tail -f /var/log/picoclaw/app.log

# Systemd journal
journalctl -u picoclaw -f
```

### Test Components

```bash
# Test provider connection
picoclaw agent -m "Hello"

# Test channel
picoclaw gateway --debug

# Check status
picoclaw status
```

## Security Issues

For security vulnerabilities:

1. **Do not** open a public issue
2. Email security concerns privately
3. Allow time for response before disclosure
4. See [Security Policy](https://github.com/sipeed/picoclaw/security/policy)

## Feature Requests

For feature suggestions:

1. Check [existing discussions](https://github.com/sipeed/picoclaw/discussions/categories/ideas)
2. Open a new discussion with "Idea" category
3. Describe the use case
4. Explain the expected benefit

## Commercial Support

For enterprise or commercial support needs, contact the maintainers through GitHub.

## FAQ

### General Questions

**Q: Is PicoClaw free?**

A: Yes, PicoClaw is open source under the MIT license. You need an LLM provider API key which may have costs.

**Q: What's the difference between PicoClaw and OpenClaw?**

A: PicoClaw is written in Go, optimized for minimal resource usage (<10MB RAM vs >1GB for OpenClaw).

**Q: Can I run PicoClaw on a Raspberry Pi?**

A: Yes! PicoClaw is designed for embedded and low-resource environments.

### Technical Questions

**Q: Which provider should I use?**

A: OpenRouter is recommended for most users - it provides access to many models with a free tier.

**Q: How do I update PicoClaw?**

A: Download the latest binary and replace the old one.

**Q: Where are my conversations stored?**

A: In `~/.picoclaw/workspace/sessions/` as JSON files.

## Still Need Help?

If you've tried everything and still need help:

1. Open a [GitHub Issue](https://github.com/sipeed/picoclaw/issues) with full details
2. Join [Discord](https://discord.gg/V4sAZ9XWpN) and ask in `#help`
3. Start a [GitHub Discussion](https://github.com/sipeed/picoclaw/discussions)

We're here to help!

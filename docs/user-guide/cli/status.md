# picoclaw status

Show PicoClaw system status and configuration.

## Usage

```bash
picoclaw status
```

## Description

The `status` command displays:

- Version and build information
- Configuration file location
- Workspace location
- API key configuration status
- OAuth authentication status

## Example Output

```
🦞 picoclaw Status
Version: 0.1.1 (git: abc123)

Config: /home/user/.picoclaw/config.json ✓
Workspace: /home/user/.picoclaw/workspace ✓
Model: anthropic/claude-opus-4-5
OpenRouter API: ✓
Anthropic API: not set
OpenAI API: ✓
Gemini API: not set
Zhipu API: not set
Groq API: ✓
vLLM/Local: not set

OAuth/Token Auth:
  openai (oauth): active
  anthropic (token): needs refresh
```

## Status Indicators

| Symbol | Meaning |
|--------|---------|
| ✓ | Configured and available |
| ✗ | Not configured or not found |
| `active` | OAuth token is valid |
| `expired` | OAuth token has expired |
| `needs refresh` | OAuth token should be refreshed |

## Use Cases

1. **Verify configuration** - Check API keys are set
2. **Debug issues** - Confirm config file is found
3. **Check authentication** - Verify OAuth tokens

## See Also

- [CLI Reference](../cli-reference.md)
- [Authentication](auth.md)

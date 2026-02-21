# picoclaw auth

Manage authentication for LLM providers.

## Usage

```bash
# Login
picoclaw auth login --provider <name> [--device-code]

# Logout
picoclaw auth logout [--provider <name>]

# Status
picoclaw auth status
```

## Subcommands

### auth login

Authenticate with a provider.

```bash
# Login via OAuth (opens browser)
picoclaw auth login --provider openai

# Login via device code (headless)
picoclaw auth login --provider openai --device-code

# Login with token (Anthropic)
picoclaw auth login --provider anthropic
```

| Flag | Short | Description |
|------|-------|-------------|
| `--provider` | `-p` | Provider name (openai, anthropic) |
| `--device-code` | | Use device code flow for headless environments |

**Supported Providers:**

| Provider | Auth Methods |
|----------|-------------|
| `openai` | OAuth, device code |
| `anthropic` | Token paste |

### auth logout

Remove stored credentials.

```bash
# Logout from specific provider
picoclaw auth logout --provider openai

# Logout from all providers
picoclaw auth logout
```

### auth status

Show current authentication status.

```bash
picoclaw auth status
```

Output:
```
Authenticated Providers:
------------------------
  openai:
    Method: oauth
    Status: active
    Account: user@example.com
    Expires: 2025-03-01 12:00
  anthropic:
    Method: token
    Status: active
```

## When to Use OAuth vs API Key

**Use OAuth/Token when:**
- You don't want to manage API keys
- You want automatic token refresh
- Using provider-specific features

**Use API Key when:**
- Simpler setup preferred
- Using multiple providers
- Server/automated environments

## Configuration

After login, set the auth method in config:

```json
{
  "providers": {
    "openai": {
      "auth_method": "oauth"
    },
    "anthropic": {
      "auth_method": "token"
    }
  }
}
```

## Credential Storage

Credentials are stored in:
```
~/.picoclaw/credentials.json
```

This file contains sensitive tokens and should be protected.

## Examples

```bash
# Login to OpenAI
picoclaw auth login --provider openai

# Check status
picoclaw auth status

# If token expired, re-login
picoclaw auth login --provider openai

# Switch to API key instead
picoclaw auth logout --provider openai
# Then add api_key to config.json
```

## See Also

- [CLI Reference](../cli-reference.md)
- [Configuration](../../configuration/config-file.md)
- [OpenAI Provider](../providers/openai.md)

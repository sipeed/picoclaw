# Antigravity Setup (Google Cloud Code Assist)

**Antigravity** (Google Cloud Code Assist) is a Google-backed AI model provider that offers free access to models like Claude Opus 4.6 and Gemini through Google's Cloud infrastructure.

## Key Features

- **Free Tier**: Generous quotas for development use
- **Multiple Models**: Access to Claude Opus, Gemini Flash, and more
- **OAuth Authentication**: No API key management required
- **Usage Tracking**: Monitor your quota consumption

## Prerequisites

1. A Google account
2. Google Cloud Code Assist enabled (usually available via "Gemini for Google Cloud" onboarding)

## Quick Start

### 1. Authenticate

```bash
picoclaw auth login --provider antigravity
```

This will:
1. Open your browser for Google OAuth login
2. Request necessary permissions
3. Store credentials in `~/.picoclaw/auth.json`

### 2. List Available Models

```bash
picoclaw auth models
```

This shows which models your project has access to and their current quotas.

### 3. Configure PicoClaw

Add to `~/.picoclaw/config.json`:

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

### 4. Start Chatting

```bash
picoclaw agent -m "Hello, how can you help me today?"
```

## Authentication Methods

### Automatic Flow (Local Machine)

On a local machine with a browser:

```bash
picoclaw auth login --provider antigravity
```

The browser opens automatically and authentication completes without additional steps.

### Manual Flow (Headless/VPS/Docker)

On a server without browser access:

1. Run the auth command:
   ```bash
   picoclaw auth login --provider antigravity
   ```

2. Copy the URL displayed and open it in your local browser

3. Complete the Google login

4. Your browser will redirect to a `localhost:51121` URL (which will fail to load)

5. **Copy that final URL** from your browser's address bar

6. **Paste it back into the terminal** where PicoClaw is waiting

PicoClaw will extract the authorization code and complete the process automatically.

### Copy Credentials to Server

If you've authenticated locally, you can copy credentials to a server:

```bash
scp ~/.picoclaw/auth.json user@your-server:~/.picoclaw/
```

## Available Models

Based on testing, these models are most reliable:

| Model | Description | Quota |
|-------|-------------|-------|
| `gemini-3-flash` | Fast, highly available | High |
| `gemini-2.5-flash-lite` | Lightweight option | High |
| `claude-opus-4-6-thinking` | Powerful, includes reasoning | Limited |

Use `picoclaw auth models` to see your actual available models and quotas.

## Switching Models

### Via Config File

Edit `~/.picoclaw/config.json`:

```json
{
  "agents": {
    "defaults": {
      "model": "claude-opus-4-6-thinking"
    }
  }
}
```

### Via CLI Override

```bash
picoclaw agent -m "Hello" --model claude-opus-4-6-thinking
```

### Via Environment Variable

```bash
export PICOCLAW_AGENTS_DEFAULTS_MODEL=gemini-3-flash
picoclaw agent -m "Hello"
```

## Docker/Coolify Deployment

For containerized deployments:

### Environment Variables

```bash
PICOCLAW_AGENTS_DEFAULTS_MODEL=gemini-flash
```

### Volume Mount for Auth

```yaml
# docker-compose.yml
services:
  picoclaw:
    volumes:
      - ~/.picoclaw/auth.json:/root/.picoclaw/auth.json:ro
```

### Pre-authenticated Setup

1. Authenticate locally first
2. Copy `auth.json` to the server
3. Mount or copy into container

## Troubleshooting

### Empty Response

If a model returns an empty reply:
- The model may be restricted for your project
- Try `gemini-3-flash` or `claude-opus-4-6-thinking`
- Check quotas with `picoclaw auth models`

### 429 Rate Limit

Antigravity has strict quotas. When you hit a limit:
- PicoClaw displays the "reset time" in the error message
- Wait for the quota to reset
- Consider using a different model temporarily

### 404 Not Found

- Ensure you're using a model ID from `picoclaw auth models`
- Use the short ID (e.g., `gemini-3-flash`), not the full path

### Token Expired

Refresh your OAuth tokens:

```bash
picoclaw auth login --provider antigravity
```

### Gemini for Google Cloud Not Enabled

Enable the API in your [Google Cloud Console](https://console.cloud.google.com).

### Models Not Appearing

1. Verify OAuth completed successfully
2. Check `~/.picoclaw/auth.json` for stored credentials
3. Re-run `picoclaw auth login --provider antigravity`

## Quota Management

### Check Quotas

```bash
picoclaw auth models
```

This shows:
- Available models
- Remaining quota percentage
- Reset time for exhausted quotas

### Quota Best Practices

1. **Use lighter models for simple tasks**: `gemini-3-flash` for quick queries
2. **Reserve Claude for complex tasks**: Use `claude-opus-4-6-thinking` for reasoning
3. **Monitor usage**: Check quotas regularly
4. **Have fallbacks**: Configure multiple models in `model_list`

## Technical Details

### OAuth Scopes

Antigravity requires these Google OAuth scopes:
- `cloud-platform` - Google Cloud access
- `userinfo.email` - User identification
- `userinfo.profile` - Profile information
- `cclog` - Cloud Code logging
- `experimentsandconfigs` - Feature flags

### Credential Storage

Credentials are stored in `~/.picoclaw/auth.json`:

```json
{
  "credentials": {
    "google-antigravity": {
      "access_token": "ya29...",
      "refresh_token": "1//...",
      "expires_at": "2026-01-01T00:00:00Z",
      "provider": "google-antigravity",
      "auth_method": "oauth",
      "email": "user@example.com",
      "project_id": "my-project-id"
    }
  }
}
```

### Token Refresh

Access tokens expire and are automatically refreshed using the refresh token. The refresh happens transparently when making API calls.

## Related Documentation

- [Provider Configuration](../providers/README.md)
- [Model Fallbacks](../advanced/model-fallbacks.md)
- [CLI Auth Commands](../cli/auth.md)

## Advanced: Provider Implementation

For developers extending PicoClaw, see [Antigravity Auth Implementation](../../developer-guide/extending/antigravity-implementation.md) for technical details on the OAuth flow and API integration.

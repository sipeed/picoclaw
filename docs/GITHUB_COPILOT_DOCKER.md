# Using PicoClaw with GitHub Copilot in Docker Mode

This guide walks through running PicoClaw as a Docker gateway (e.g. Discord bot) using GitHub Copilot as the LLM provider.

## Prerequisites

- [Docker Desktop](https://www.docker.com/products/docker-desktop/) installed and running
- [GitHub Copilot CLI](https://docs.github.com/en/copilot/github-copilot-in-the-cli) installed and authenticated
- A Discord bot token (see [Discord Setup](#discord-setup) below)

---

## Step 1: Start the Copilot CLI in Headless Mode

PicoClaw connects to the Copilot CLI over gRPC. You must start it with `--headless` (not `--acp`):

```bash
copilot --headless --port 4321
```

Leave this running in a terminal. PicoClaw will connect to it from inside Docker via `host.docker.internal:4321`.

> **Note:** `--acp` starts a different protocol and will not work. You must use `--headless`.

---

## Step 2: Configure PicoClaw

Edit `config/config.json`:

```json
{
  "agents": {
    "defaults": {
      "provider": "github_copilot",
      "model": "copilot-model"
    }
  },
  "model_list": [
    {
      "model_name": "copilot-model",
      "model": "github-copilot/gpt-4.1",
      "api_base": "host.docker.internal:4321",
      "connect_mode": "grpc"
    }
  ]
}
```

**Key fields:**
- `model_name` — an alias used by `agents.defaults.model` to reference this entry
- `model` — `github-copilot/<model-id>`, e.g. `github-copilot/gpt-4.1` or `github-copilot/claude-sonnet-4.6`
- `api_base` — use `host.docker.internal:4321` to reach the host from inside Docker
- `connect_mode` — must be `grpc`

---

## Step 3: Configure Docker Networking (Linux hosts only)

If running Docker on Linux, add `extra_hosts` to `docker-compose.yml` so the container can reach `host.docker.internal`:

```yaml
services:
  picoclaw-gateway:
    extra_hosts:
      - "host.docker.internal:host-gateway"
```

Docker Desktop on Windows/Mac handles this automatically.

---

## Step 4: Discord Setup

1. Go to [discord.com/developers/applications](https://discord.com/developers/applications) and create a new application
2. Under **Bot**, create a bot and copy the token
3. Enable **Message Content Intent** under Bot → Privileged Gateway Intents
4. Under **OAuth2 → URL Generator**, select scopes: `bot` — permissions: `Send Messages`, `Read Message History`, `View Channels`
5. Use the generated URL to invite the bot to your server

Add to `config/config.json`:

```json
"channels": {
  "discord": {
    "enabled": true,
    "token": "YOUR_DISCORD_BOT_TOKEN",
    "allow_from": ["YOUR_DISCORD_USER_ID"],
    "mention_only": false
  }
}
```

To find your Discord user ID: **Settings → Advanced → Developer Mode**, then right-click your username → **Copy User ID**.

---

## Step 5: Build and Run

```bash
# Build the image
docker compose --profile gateway build

# Start the gateway
docker compose --profile gateway up -d picoclaw-gateway

# Follow logs
docker compose --profile gateway logs -f picoclaw-gateway
```

A successful startup looks like:
```
✓ Channels enabled: [discord]
✓ Gateway started on 0.0.0.0:18790
✓ Discord bot connected {username=picoclaw, ...}
✓ Health endpoints available at http://0.0.0.0:18790/health and /ready
```

---

## Step 6: Test

In your Discord server, mention the bot:
```
@picoclaw introduce yourself
```

---

## Workspace Volume

PicoClaw persists memory and workspace files in a Docker volume:

```bash
docker volume inspect picoclaw-docker_picoclaw-workspace
```

On Windows (Docker Desktop + WSL2), browse it at:
```
\\wsl$\docker-desktop-data\data\docker\volumes\picoclaw-docker_picoclaw-workspace\_data
```

---

## Restarting After Changes

```bash
docker compose --profile gateway down
docker compose --profile gateway build
docker compose --profile gateway up -d picoclaw-gateway
```

Config changes (e.g. `config/config.json`) take effect on restart without a rebuild, since the config is mounted as a volume.

# PicoClaw Coolify Deployment

Deploy PicoClaw on Coolify with persistent storage and easy configuration editing.

## Quick Start

### 1. Configuration

Edit `docker-compose.coolify.yml` to set the desired port:

```yaml
services:
  picoclaw-gateway:
    ports:
      - "${HOST_PORT:-18790}:18790"
```

Change `HOST_PORT` to your desired port (e.g., `18800:18790`).

### 2. Persistent Storage (Bind Mount)

In Coolify UI, add a persistent storage:

- **Type:** Bind Mount
- **Source Path:** `/apps/picoclaw/`
- **Destination Path:** `/root/.picoclaw`

### 3. Deploy

Deploy from Coolify. The init container will run `picoclaw onboard` on first run, creating:
- `config.json`
- `workspace/` (AGENTS.md, SOUL.md, MEMORY.md, USER.md)
- `sessions/`
- `skills/`

## Editing Configuration

### Via SSH

```bash
nano /apps/picoclaw/config.json
```

### Via VSCodium

1. Install Remote-SSH extension
2. Connect to server: `ssh user@your-server`
3. Open folder: `/apps/picoclaw/`

### Via VSCode

Same as VSCodium - use Remote SSH extension.

## File Structure

After first run, the data folder contains:

```
/apps/picoclaw/
├── config.json          # Main configuration (edit this!)
├── workspace/           # Agent workspace
│   ├── AGENTS.md       # Agent instructions
│   ├── SOUL.md         # Personality/beliefs
│   ├── MEMORY.md       # Long-term memory
│   ├── USER.md         # User preferences
│   └── skills/         # Installed skills
├── sessions/           # Session history
└── skills/             # Global skills
```

## Configuration

Edit `config.json` to configure:
- Model provider (OpenAI, Anthropic, DeepSeek, etc.)
- Channels (Telegram, Discord, WhatsApp, QQ, etc.)
- Tools (web search, exec, cron, MCP)
- Agent defaults

See `config/config.example.json` in the project root for full reference.

## Troubleshooting

### Container won't start

Check logs: `docker logs picoclaw-gateway`

### Config not being read

Ensure bind mount path `/apps/picoclaw/` is correctly set in Coolify persistent storage.

### Port already in use

Edit `docker-compose.coolify.yml` and change the port mapping.

## Example Coolify Setup

```
Persistent Storage:
  - Source: /apps/picoclaw/
  - Destination: /root/.picoclaw
  - Type: Bind Mount
```

## Building from Source (Optional)

If you need to build your own image:

```bash
cd picoclaw
docker build -f docker/Dockerfile -t my-picoclaw:latest .
```

Then update `docker-compose.coolify.yml` to use `my-picoclaw:latest`.

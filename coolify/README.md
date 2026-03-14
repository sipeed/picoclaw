# PicoClaw on Coolify

Deploy PicoClaw on Coolify with persistent storage and easy configuration editing.

## How it works

- Init container runs `picoclaw onboard` to create config + workspace
- Gateway container runs the bot
- Data persists via bind mount

## Change bind folder & port

Edit `docker-compose.coolify.yml`:

```yaml
services:
  picoclaw-init:
    volumes:
      - /YOUR/PATH/:/root/.picoclaw

  picoclaw-gateway:
    volumes:
      - /YOUR/PATH/:/root/.picoclaw
    ports:
      - "YOUR_PORT:18790"
```

Replace:
- `/YOUR/PATH/` - your folder path (e.g., `/home/lou/Dev/apps/picoclaw/`)
- `YOUR_PORT` - your desired port (e.g., `18800`)

## Deploy

1. Create folder on server: `mkdir -p /YOUR/PATH/`
2. In Coolify: Add Docker Compose resource
3. Paste `docker-compose.coolify.yml`
4. Add persistent storage:
   - Type: Bind Mount
   - Source: `/YOUR/PATH/`
   - Destination: `/root/.picoclaw`
5. Deploy
6. Edit config at `/YOUR/PATH/config.json`

## Edit Config

### Via SSH

```bash
nano /YOUR/PATH/config.json
```

### Via VSCodium/VSCode

1. Install Remote-SSH extension
2. Connect to server: `ssh user@your-server`
3. Open folder: `/YOUR/PATH/`

## Troubleshooting

### Container won't start

Check logs: `docker logs picoclaw-gateway`

### Config not being read

Ensure bind mount path is correctly set in Coolify persistent storage.

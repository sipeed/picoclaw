# picoclaw gateway

Start the gateway for all enabled chat channels.

## Usage

```bash
# Start gateway
picoclaw gateway

# With debug logging
picoclaw gateway --debug
```

## Options

| Flag | Short | Description |
|------|-------|-------------|
| `--debug` | `-d` | Enable debug logging |

## Description

The gateway command starts PicoClaw in server mode, enabling:

1. **Chat Channels** - Telegram, Discord, Slack, etc.
2. **Heartbeat Service** - Periodic task execution
3. **Cron Service** - Scheduled jobs
4. **Device Monitoring** - USB device events (Linux)
5. **Health Endpoints** - `/health` and `/ready`

## What It Starts

```
✓ Channels enabled: telegram, discord
✓ Gateway started on 0.0.0.0:18790
Press Ctrl+C to stop
✓ Cron service started
✓ Heartbeat service started
✓ Health endpoints available at http://0.0.0.0:18790/health and /ready
```

## Configuration

Enable channels in `~/.picoclaw/config.json`:

```json
{
  "channels": {
    "telegram": {
      "enabled": true,
      "token": "YOUR_BOT_TOKEN",
      "allow_from": ["YOUR_USER_ID"]
    },
    "discord": {
      "enabled": true,
      "token": "YOUR_BOT_TOKEN"
    }
  }
}
```

## Gateway Settings

Configure in `~/.picoclaw/config.json`:

```json
{
  "gateway": {
    "host": "0.0.0.0",
    "port": 18790
  }
}
```

## Health Endpoints

| Endpoint | Purpose |
|----------|---------|
| `/health` | Liveness check |
| `/ready` | Readiness check |

```bash
curl http://localhost:18790/health
# OK

curl http://localhost:18790/ready
# OK
```

## Stopping

Press `Ctrl+C` to stop the gateway gracefully:

```
^C
Shutting down...
✓ Gateway stopped
```

## Running as a Service

For production, run as a systemd service. See [Systemd Deployment](../../deployment/systemd.md).

## Examples

```bash
# Start with Telegram and Discord
picoclaw gateway

# Debug mode for troubleshooting
picoclaw gateway --debug

# Run in background
nohup picoclaw gateway > picoclaw.log 2>&1 &
```

## See Also

- [Channels Guide](../channels/README.md)
- [Systemd Deployment](../../deployment/systemd.md)
- [Health Endpoints](../../operations/health-endpoints.md)

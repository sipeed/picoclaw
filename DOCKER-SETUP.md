# PicoClaw Docker Setup

This directory contains Docker configuration for running PicoClaw in a containerized environment.

## Quick Start

1. **Clone the repository**
   ```bash
   git clone https://github.com/sipeed/picoclaw.git
   cd picoclaw
   ```

2. **Create configuration**
   ```bash
   cp config.example.json config.json
   # Edit config.json with your API keys
   ```

3. **Start PicoClaw**
   ```bash
   docker-compose up -d
   ```

4. **Use PicoClaw**
   ```bash
   # Interactive chat
   docker-compose exec picoclaw picoclaw agent
   
   # Single message
   docker-compose exec picoclaw picoclaw agent -m "Help me with..."
   
   # View logs
   docker-compose logs -f picoclaw
   ```

## Configuration

### Config File

The `config.json` file is mounted into the container at runtime. Edit it on your host machine and restart the container to apply changes:

```bash
docker-compose restart
```

### Environment Variables

You can also configure PicoClaw using environment variables in `docker-compose.yml`:

```yaml
environment:
  - PICOCLAW_MODEL=glm-4.7
  - PICOCLAW_OPENROUTER_API_KEY=your_key_here
```

Or create a `.env` file:

```bash
VERSION=1.0.0
OPENROUTER_API_KEY=sk-xxx
BRAVE_API_KEY=BSA-xxx
```

## Volumes

The setup uses Docker volumes for data persistence:

- **config.json**: Configuration file (mounted read-only)
- **picoclaw-workspace**: Agent workspace directory (persistent volume)

### Backup Workspace

```bash
docker run --rm -v picoclaw_picoclaw-workspace:/workspace -v $(pwd):/backup alpine tar czf /backup/workspace-backup.tar.gz -C /workspace .
```

### Restore Workspace

```bash
docker run --rm -v picoclaw_picoclaw-workspace:/workspace -v $(pwd):/backup alpine tar xzf /backup/workspace-backup.tar.gz -C /workspace
```

## Network Channels

If you're using network-based channels (Telegram, Discord, etc.), the container runs in the background and handles messages automatically.

For channels that need specific ports (like MaixCAM on port 18790), the ports are already exposed in `docker-compose.yml`.

## Using with Chat Apps

### Telegram Bot
```bash
# Configure telegram in config.json
docker-compose up -d
# Bot will automatically start listening
docker-compose logs -f picoclaw
```

### Discord Bot
```bash
# Configure discord in config.json  
docker-compose up -d
# Bot will connect automatically
```

## Building Custom Image

To build with a specific version:

```bash
docker build --build-arg VERSION=1.0.0 -t picoclaw:1.0.0 .
```

## Resource Limits

Uncomment the `deploy.resources` section in `docker-compose.yml` to set CPU and memory limits:

```yaml
deploy:
  resources:
    limits:
      cpus: '1'
      memory: 512M
```

## Troubleshooting

### Check container status
```bash
docker-compose ps
```

### View logs
```bash
docker-compose logs picoclaw
```

### Restart container
```bash
docker-compose restart
```

### Rebuild image
```bash
docker-compose build --no-cache
docker-compose up -d
```

### Enter container shell
```bash
docker-compose exec picoclaw sh
```

### Remove everything and start fresh
```bash
docker-compose down -v
docker-compose up -d
```

## Multi-Platform Build

To build for multiple architectures:

```bash
docker buildx create --use
docker buildx build --platform linux/amd64,linux/arm64,linux/riscv64 -t picoclaw:latest .
```

## Production Deployment

For production use:

1. Use specific version tags instead of `latest`
2. Set appropriate resource limits
3. Configure proper logging with log rotation
4. Use secrets management for API keys
5. Enable health checks (already configured)
6. Consider using Docker Swarm or Kubernetes for orchestration

Example with Docker secrets:

```bash
echo "your-api-key" | docker secret create openrouter_key -
```

## Performance Notes

PicoClaw is designed to be ultra-lightweight:
- **Image size**: ~50MB (multi-stage build)
- **Memory usage**: <10MB for core functionality
- **Startup time**: <1 second

Perfect for running on:
- Raspberry Pi
- NAS devices
- Home servers
- Edge devices
- Cloud VMs (even the smallest instances)

## Support

For issues specific to Docker deployment, please check:
1. Docker logs: `docker-compose logs`
2. Container status: `docker-compose ps`
3. Configuration: Verify `config.json` is valid JSON
4. Network: Ensure required ports are available

For general PicoClaw issues, see the main [README.md](../README.md).

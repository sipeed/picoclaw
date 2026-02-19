# Docker Deployment

PicoClaw provides official Docker support with multi-stage builds for minimal image sizes. This guide covers deploying PicoClaw using Docker and Docker Compose.

## Prerequisites

- Docker 20.10 or later
- Docker Compose V2 (recommended)
- API keys for your LLM providers

## Quick Start

### Using Docker Compose (Recommended)

```bash
# Clone the repository
git clone https://github.com/sipeed/picoclaw.git
cd picoclaw

# Create configuration
mkdir -p config
cp config/config.example.json config/config.json

# Edit configuration with your API keys
nano config/config.json

# Start gateway mode (always-on bot)
docker compose --profile gateway up -d

# View logs
docker compose logs -f picoclaw-gateway
```

### One-Shot Agent Mode

Run a single query using Docker:

```bash
# Run a one-shot query
docker compose run --rm picoclaw-agent -m "Hello, how are you?"

# Interactive mode
docker compose run --rm -it picoclaw-agent
```

## Docker Compose Configuration

The included `docker-compose.yml` defines two services:

### Gateway Service (Always-On Bot)

```yaml
picoclaw-gateway:
  build:
    context: .
    dockerfile: Dockerfile
  container_name: picoclaw-gateway
  restart: unless-stopped
  profiles:
    - gateway
  volumes:
    - ./config/config.json:/home/picoclaw/.picoclaw/config.json:ro
    - picoclaw-workspace:/home/picoclaw/.picoclaw/workspace
  command: ["gateway"]
```

### Agent Service (One-Shot)

```yaml
picoclaw-agent:
  build:
    context: .
    dockerfile: Dockerfile
  container_name: picoclaw-agent
  profiles:
    - agent
  volumes:
    - ./config/config.json:/home/picoclaw/.picoclaw/config.json:ro
    - picoclaw-workspace:/home/picoclaw/.picoclaw/workspace
  entrypoint: ["picoclaw", "agent"]
  stdin_open: true
  tty: true
```

## Building the Image

### Standard Build

```bash
# Build using Docker Compose
docker compose build

# Or build directly
docker build -t picoclaw:latest .
```

### Build with Custom Arguments

```bash
# Build with specific Go version
docker build --build-arg GO_VERSION=1.23 -t picoclaw:latest .
```

### Multi-Platform Build

Build for multiple architectures using buildx:

```bash
# Enable buildx
docker buildx create --use

# Build for multiple platforms
docker buildx build --platform linux/amd64,linux/arm64 -t picoclaw:latest .
```

## Running the Container

### Basic Run

```bash
# Run gateway mode
docker run -d \
  --name picoclaw \
  -v $(pwd)/config/config.json:/home/picoclaw/.picoclaw/config.json:ro \
  -v picoclaw-workspace:/home/picoclaw/.picoclaw/workspace \
  picoclaw:latest gateway

# Run agent mode (one-shot)
docker run --rm \
  -v $(pwd)/config/config.json:/home/picoclaw/.picoclaw/config.json:ro \
  -v picoclaw-workspace:/home/picoclaw/.picoclaw/workspace \
  picoclaw:latest agent -m "Hello"
```

### With Environment Variables

```bash
docker run -d \
  --name picoclaw \
  -e PICOCLAW_AGENTS_DEFAULTS_MODEL="gpt-4o" \
  -e PICOCLAW_PROVIDERS_OPENROUTER_API_KEY="sk-or-v1-xxx" \
  -v picoclaw-workspace:/home/picoclaw/.picoclaw/workspace \
  picoclaw:latest gateway
```

### With Port Mapping

If you need to expose the gateway HTTP server:

```bash
docker run -d \
  --name picoclaw \
  -p 18790:18790 \
  -v $(pwd)/config/config.json:/home/picoclaw/.picoclaw/config.json:ro \
  -v picoclaw-workspace:/home/picoclaw/.picoclaw/workspace \
  picoclaw:latest gateway
```

## Volume Management

### Persistent Workspace

The workspace volume stores sessions, memory, and other persistent data:

```bash
# List volumes
docker volume ls

# Inspect workspace volume
docker volume inspect picoclaw-workspace

# Backup workspace
docker run --rm -v picoclaw-workspace:/data -v $(pwd):/backup alpine tar czf /backup/workspace-backup.tar.gz /data

# Restore workspace
docker run --rm -v picoclaw-workspace:/data -v $(pwd):/backup alpine tar xzf /backup/workspace-backup.tar.gz -C /
```

### Configuration Mount

Configuration is mounted read-only for security:

```bash
# Mount config from host
-v /path/to/config.json:/home/picoclaw/.picoclaw/config.json:ro
```

## Health Checks

The Dockerfile includes a built-in health check:

```dockerfile
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -q --spider http://localhost:18790/health || exit 1
```

Check container health:

```bash
# View health status
docker ps --format "table {{.Names}}\t{{.Status}}"

# Inspect health details
docker inspect --format='{{json .State.Health}}' picoclaw-gateway | jq
```

## Logging

### View Logs

```bash
# Docker Compose logs
docker compose logs -f picoclaw-gateway

# Docker logs
docker logs -f picoclaw-gateway

# Last 100 lines
docker logs --tail 100 picoclaw-gateway
```

### Configure Log Driver

```yaml
services:
  picoclaw-gateway:
    # ... other config ...
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
```

## Advanced Configuration

### Custom Docker Compose File

Create `docker-compose.prod.yml` for production:

```yaml
services:
  picoclaw-gateway:
    image: picoclaw:latest
    restart: always
    environment:
      - PICOCLAW_HEARTBEAT_ENABLED=true
      - PICOCLAW_HEARTBEAT_INTERVAL=30
    volumes:
      - ./config/production.json:/home/picoclaw/.picoclaw/config.json:ro
      - picoclaw-workspace:/home/picoclaw/.picoclaw/workspace
    logging:
      driver: "json-file"
      options:
        max-size: "50m"
        max-file: "5"
    deploy:
      resources:
        limits:
          memory: 512M
        reservations:
          memory: 128M
    command: ["gateway"]

volumes:
  picoclaw-workspace:
```

Run with:

```bash
docker compose -f docker-compose.prod.yml up -d
```

### Network Configuration

For connecting to other services:

```yaml
services:
  picoclaw-gateway:
    networks:
      - picoclaw-net
      - external-net

networks:
  picoclaw-net:
    driver: bridge
  external-net:
    external: true
```

### Secrets Management

Using Docker secrets (Swarm mode):

```yaml
services:
  picoclaw-gateway:
    secrets:
      - openrouter_api_key
    environment:
      - PICOCLAW_PROVIDERS_OPENROUTER_API_KEY_FILE=/run/secrets/openrouter_api_key

secrets:
  openrouter_api_key:
    external: true
```

## Production Deployment

### Recommendations

1. **Use specific version tags** instead of `latest`:
   ```bash
   docker pull picoclaw:v0.1.1
   ```

2. **Set resource limits**:
   ```yaml
   deploy:
     resources:
       limits:
         memory: 512M
         cpus: '1.0'
   ```

3. **Configure log rotation** to prevent disk exhaustion

4. **Use health checks** for orchestration

5. **Mount configuration read-only** for security

### Example Production docker-compose.yml

```yaml
version: "3.8"

services:
  picoclaw-gateway:
    image: ghcr.io/sipeed/picoclaw:v0.1.1
    container_name: picoclaw-gateway
    restart: always
    profiles:
      - gateway
    volumes:
      - ./config/production.json:/home/picoclaw/.picoclaw/config.json:ro
      - picoclaw-workspace:/home/picoclaw/.picoclaw/workspace
    environment:
      - TZ=UTC
    logging:
      driver: "json-file"
      options:
        max-size: "50m"
        max-file: "5"
    deploy:
      resources:
        limits:
          memory: 512M
        reservations:
          memory: 128M
    healthcheck:
      test: ["CMD", "wget", "-q", "--spider", "http://localhost:18790/health"]
      interval: 30s
      timeout: 3s
      retries: 3
      start_period: 10s
    command: ["gateway"]

volumes:
  picoclaw-workspace:
    driver: local
```

## Troubleshooting

### Container won't start

```bash
# Check logs
docker compose logs picoclaw-gateway

# Check config file exists
ls -la config/config.json

# Verify JSON syntax
cat config/config.json | jq .
```

### Permission issues

```bash
# The container runs as non-root user (uid 1000)
# Ensure config file is readable
chmod 644 config/config.json
```

### API connection issues

```bash
# Verify network connectivity from container
docker compose exec picoclaw-gateway wget -q -O- https://api.openai.com/v1/models

# Check DNS resolution
docker compose exec picoclaw-gateway nslookup api.openai.com
```

### Volume issues

```bash
# List volumes
docker volume ls

# Remove and recreate volume (WARNING: destroys data)
docker compose down -v
docker compose up -d
```

### Reset everything

```bash
# Stop containers
docker compose down

# Remove volumes
docker compose down -v

# Rebuild image
docker compose build --no-cache

# Start fresh
docker compose --profile gateway up -d
```

# Deployment Guide

This section covers deploying PicoClaw in various environments, from local development to production servers and embedded devices.

## Deployment Options

PicoClaw is designed to run anywhere - from powerful servers to ultra-low-cost single-board computers. Choose the deployment method that fits your needs:

### By Environment

| Environment | Best For | Guide |
|-------------|----------|-------|
| **Docker** | Containerized deployments, easy scaling | [Docker Guide](docker.md) |
| **Systemd** | Linux servers, always-on services | [Systemd Guide](systemd.md) |
| **Termux** | Android phones/tablets | [Termux Guide](termux.md) |
| **Single-Board Computers** | Edge devices, IoT | [SBC Guide](sbc/README.md) |

### By Hardware

| Hardware | Cost | Use Case |
|----------|------|----------|
| Any Linux server | Varies | Production deployments |
| Raspberry Pi 4/5 | $35-80 | Home automation, bots |
| LicheeRV Nano | $9.9 | Ultra-low-cost edge AI |
| MaixCAM | $50 | AI camera applications |
| Android (Termux) | Free | Repurpose old phones |

## Quick Start

### Option 1: Binary (Simplest)

```bash
# Download precompiled binary
wget https://github.com/sipeed/picoclaw/releases/latest/download/picoclaw-linux-amd64
chmod +x picoclaw-linux-amd64

# Initialize configuration
./picoclaw-linux-amd64 onboard

# Edit config with your API keys
nano ~/.picoclaw/config.json

# Run gateway (always-on bot)
./picoclaw-linux-amd64 gateway
```

### Option 2: Docker

```bash
# Clone repository
git clone https://github.com/sipeed/picoclaw.git
cd picoclaw

# Configure
cp config/config.example.json config/config.json
# Edit config.json with your API keys

# Start gateway
docker compose --profile gateway up -d
```

### Option 3: Build from Source

```bash
# Requires Go 1.21+
git clone https://github.com/sipeed/picoclaw.git
cd picoclaw

make deps
make build
make install

picoclaw onboard
picoclaw gateway
```

## Operating Modes

PicoClaw supports two main operating modes:

### Agent Mode (One-Shot)

Run a single query and exit. Useful for CLI integration, scripts, and testing.

```bash
# Single query
picoclaw agent -m "What is the weather today?"

# Interactive session
picoclaw agent

# Debug mode
picoclaw agent --debug -m "Hello"
```

### Gateway Mode (Always-On)

Run as a long-running bot that connects to chat platforms.

```bash
# Start gateway (connects to Telegram, Discord, etc.)
picoclaw gateway

# With debug logging
picoclaw gateway --debug
```

## Configuration

PicoClaw uses a single configuration file at `~/.picoclaw/config.json`. See the [Configuration File Reference](../configuration/config-file.md) for complete details.

### Minimum Configuration

```json
{
  "agents": {
    "defaults": {
      "model": "openrouter/anthropic/claude-opus-4-5"
    }
  },
  "providers": {
    "openrouter": {
      "api_key": "sk-or-v1-your-key-here"
    }
  }
}
```

### Environment Variables

All config options can be set via environment variables:

```bash
export PICOCLAW_AGENTS_DEFAULTS_MODEL="gpt-4o"
export PICOCLAW_PROVIDERS_OPENROUTER_API_KEY="sk-or-v1-xxx"

picoclaw agent -m "Hello"
```

## Next Steps

1. **Docker Deployment** - [docker.md](docker.md) - Container-based deployment
2. **Systemd Service** - [systemd.md](systemd.md) - Run as a Linux service
3. **Security Checklist** - [security.md](security.md) - Production security
4. **Single-Board Computers** - [sbc/README.md](sbc/README.md) - Raspberry Pi, LicheeRV, MaixCAM
5. **Android/Termux** - [termux.md](termux.md) - Run on Android devices

## Troubleshooting

### Common Issues

**Permission denied:**
```bash
chmod +x picoclaw-*
```

**Command not found:**
```bash
# Add to PATH
export PATH="$PWD:$PATH"
# Or install to system
sudo mv picoclaw-* /usr/local/bin/picoclaw
```

**Config file not found:**
```bash
# Initialize config
picoclaw onboard
```

**API errors:**
- Verify API keys are correct
- Check network connectivity
- Review provider-specific documentation

For more help, see the [Troubleshooting Guide](../operations/troubleshooting.md).

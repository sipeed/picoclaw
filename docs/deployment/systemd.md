# Running PicoClaw as a Systemd Service

This guide explains how to run PicoClaw as a systemd service on Linux systems. This is the recommended approach for production deployments on servers and single-board computers.

## Prerequisites

- Linux system with systemd
- Root/sudo access
- PicoClaw binary installed

## Installation

### Step 1: Install the Binary

```bash
# Download the binary
wget https://github.com/sipeed/picoclaw/releases/latest/download/picoclaw-linux-amd64
chmod +x picoclaw-linux-amd64
sudo mv picoclaw-linux-amd64 /usr/local/bin/picoclaw

# Or build from source
git clone https://github.com/sipeed/picoclaw.git
cd picoclaw
make deps
make build
sudo make install
```

### Step 2: Create a Dedicated User

Create a system user for running PicoClaw:

```bash
sudo useradd -r -s /bin/false -d /var/lib/picoclaw picoclaw
```

### Step 3: Create Configuration Directory

```bash
# Create config directory
sudo mkdir -p /etc/picoclaw

# Create workspace directory
sudo mkdir -p /var/lib/picoclaw/workspace

# Set ownership
sudo chown -R picoclaw:picoclaw /var/lib/picoclaw
```

### Step 4: Initialize Configuration

```bash
# Run onboard as the picoclaw user
sudo -u picoclaw /usr/local/bin/picoclaw onboard

# Or create config manually
sudo tee /etc/picoclaw/config.json > /dev/null <<'EOF'
{
  "agents": {
    "defaults": {
      "workspace": "/var/lib/picoclaw/workspace",
      "model": "openrouter/anthropic/claude-opus-4-5"
    }
  },
  "providers": {
    "openrouter": {
      "api_key": "sk-or-v1-your-key-here"
    }
  },
  "gateway": {
    "host": "0.0.0.0",
    "port": 18790
  }
}
EOF

sudo chown picoclaw:picoclaw /etc/picoclaw/config.json
sudo chmod 600 /etc/picoclaw/config.json
```

## Systemd Service File

### Basic Service

Create `/etc/systemd/system/picoclaw.service`:

```ini
[Unit]
Description=PicoClaw AI Assistant Gateway
After=network.target
Wants=network-online.target

[Service]
Type=simple
User=picoclaw
Group=picoclaw
ExecStart=/usr/local/bin/picoclaw gateway
Restart=on-failure
RestartSec=5

# Environment
Environment=PICOCLAW_CONFIG=/etc/picoclaw/config.json

# Security hardening
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/picoclaw
PrivateTmp=true

# Logging
StandardOutput=journal
StandardError=journal
SyslogIdentifier=picoclaw

[Install]
WantedBy=multi-user.target
```

### Service with Environment Variables

For using environment variables instead of config file:

```ini
[Unit]
Description=PicoClaw AI Assistant Gateway
After=network.target
Wants=network-online.target

[Service]
Type=simple
User=picoclaw
Group=picoclaw
ExecStart=/usr/local/bin/picoclaw gateway
Restart=on-failure
RestartSec=5

# Environment variables
Environment="PICOCLAW_AGENTS_DEFAULTS_MODEL=openrouter/anthropic/claude-opus-4-5"
Environment="PICOCLAW_PROVIDERS_OPENROUTER_API_KEY=sk-or-v1-your-key-here"
Environment="PICOCLAW_HEARTBEAT_ENABLED=true"
Environment="PICOCLAW_HEARTBEAT_INTERVAL=30"

# Security hardening
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/picoclaw
PrivateTmp=true

[Install]
WantedBy=multi-user.target
```

### Production Service with Full Hardening

```ini
[Unit]
Description=PicoClaw AI Assistant Gateway
Documentation=https://github.com/sipeed/picoclaw
After=network.target network-online.target
Wants=network-online.target

[Service]
Type=simple
User=picoclaw
Group=picoclaw
WorkingDirectory=/var/lib/picoclaw

# Main command
ExecStart=/usr/local/bin/picoclaw gateway
ExecReload=/bin/kill -HUP $MAINPID

# Restart policy
Restart=on-failure
RestartSec=5
TimeoutStartSec=30
TimeoutStopSec=30

# Environment
Environment=PICOCLAW_CONFIG=/etc/picoclaw/config.json
Environment=PICOCLAW_WORKSPACE=/var/lib/picoclaw/workspace

# Resource limits
LimitNOFILE=65536
MemoryMax=512M
MemoryHigh=400M

# Security hardening
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ProtectKernelTunables=true
ProtectControlGroups=true
ProtectKernelModules=true
ProtectHostname=true
ProtectClock=true
ProtectProc=invisible
ProcSubset=pid
PrivateTmp=true
PrivateDevices=true
PrivateUsers=true
PrivateMounts=true
CapabilityBoundingSet=
AmbientCapabilities=
LockPersonality=true
RestrictNamespaces=true
RestrictRealtime=true
RestrictSUIDSGID=true
RestrictAddressFamilies=AF_INET AF_INET6
SystemCallFilter=@system-service
SystemCallArchitectures=native
MemoryDenyWriteExecute=true

# Allow access to workspace
ReadWritePaths=/var/lib/picoclaw
ReadOnlyPaths=/etc/picoclaw

# Logging
StandardOutput=journal
StandardError=journal
SyslogIdentifier=picoclaw

[Install]
WantedBy=multi-user.target
```

## Managing the Service

### Enable and Start

```bash
# Reload systemd to pick up new service
sudo systemctl daemon-reload

# Enable to start on boot
sudo systemctl enable picoclaw

# Start the service
sudo systemctl start picoclaw

# Check status
sudo systemctl status picoclaw
```

### Common Operations

```bash
# Stop the service
sudo systemctl stop picoclaw

# Restart the service
sudo systemctl restart picoclaw

# Disable from starting on boot
sudo systemctl disable picoclaw

# View status
sudo systemctl status picoclaw

# Check if enabled
systemctl is-enabled picoclaw
```

### Viewing Logs

```bash
# View recent logs
sudo journalctl -u picoclaw

# Follow logs in real-time
sudo journalctl -u picoclaw -f

# Last 100 lines
sudo journalctl -u picoclaw -n 100

# Logs from last hour
sudo journalctl -u picoclaw --since "1 hour ago"

# Logs since boot
sudo journalctl -u picoclaw -b

# With priority filtering
sudo journalctl -u picoclaw -p err
```

## Configuration Management

### Using Config File with Symlink

Link the default config location to `/etc/picoclaw/config.json`:

```bash
# Remove default config location
sudo rm -rf /home/picoclaw/.picoclaw

# Create symlink
sudo ln -s /etc/picoclaw /home/picoclaw/.picoclaw
```

### Using Environment File

Create `/etc/picoclaw/picoclaw.env`:

```bash
PICOCLAW_AGENTS_DEFAULTS_MODEL=openrouter/anthropic/claude-opus-4-5
PICOCLAW_PROVIDERS_OPENROUTER_API_KEY=sk-or-v1-your-key-here
PICOCLAW_CHANNELS_TELEGRAM_ENABLED=true
PICOCLAW_CHANNELS_TELEGRAM_TOKEN=123456:ABC
PICOCLAW_HEARTBEAT_ENABLED=true
PICOCLAW_HEARTBEAT_INTERVAL=30
```

Update service file:

```ini
[Service]
EnvironmentFile=/etc/picoclaw/picoclaw.env
```

### Secure API Key Storage

Store API keys in a separate file with restricted permissions:

```bash
# Create secrets file
sudo tee /etc/picoclaw/secrets.env > /dev/null <<'EOF'
PICOCLAW_PROVIDERS_OPENROUTER_API_KEY=sk-or-v1-your-key-here
PICOCLAW_PROVIDERS_OPENAI_API_KEY=sk-your-key-here
EOF

# Restrict permissions
sudo chmod 600 /etc/picoclaw/secrets.env
sudo chown picoclaw:picoclaw /etc/picoclaw/secrets.env
```

## Multiple Instances

### Running Multiple Agents

Create separate service files for different agents:

`/etc/systemd/system/picoclaw-telegram.service`:

```ini
[Unit]
Description=PicoClaw Telegram Bot
After=network.target

[Service]
Type=simple
User=picoclaw
Group=picoclaw
ExecStart=/usr/local/bin/picoclaw gateway
Environment=PICOCLAW_CONFIG=/etc/picoclaw/telegram.json
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
```

`/etc/systemd/system/picoclaw-discord.service`:

```ini
[Unit]
Description=PicoClaw Discord Bot
After=network.target

[Service]
Type=simple
User=picoclaw
Group=picoclaw
ExecStart=/usr/local/bin/picoclaw gateway
Environment=PICOCLAW_CONFIG=/etc/picoclaw/discord.json
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
```

Enable both:

```bash
sudo systemctl daemon-reload
sudo systemctl enable picoclaw-telegram picoclaw-discord
sudo systemctl start picoclaw-telegram picoclaw-discord
```

## Updates and Maintenance

### Updating the Binary

```bash
# Stop service
sudo systemctl stop picoclaw

# Download new version
wget https://github.com/sipeed/picoclaw/releases/latest/download/picoclaw-linux-amd64
chmod +x picoclaw-linux-amd64
sudo mv picoclaw-linux-amd64 /usr/local/bin/picoclaw

# Start service
sudo systemctl start picoclaw

# Verify
sudo systemctl status picoclaw
```

### Backup Configuration

```bash
# Backup config and workspace
sudo tar czf picoclaw-backup-$(date +%Y%m%d).tar.gz \
  /etc/picoclaw \
  /var/lib/picoclaw
```

### Health Check Script

Create `/usr/local/bin/picoclaw-health-check.sh`:

```bash
#!/bin/bash
# Health check script for PicoClaw

HEALTH_URL="http://localhost:18790/health"
TIMEOUT=5

if curl -sf --max-time $TIMEOUT "$HEALTH_URL" > /dev/null; then
    echo "PicoClaw is healthy"
    exit 0
else
    echo "PicoClaw health check failed"
    exit 1
fi
```

Add to systemd timer or cron:

```bash
# Add to cron
*/5 * * * * /usr/local/bin/picoclaw-health-check.sh || systemctl restart picoclaw
```

## Troubleshooting

### Service Fails to Start

```bash
# Check service status
sudo systemctl status picoclaw

# View logs
sudo journalctl -u picoclaw -n 50

# Check if binary exists and is executable
ls -la /usr/local/bin/picoclaw

# Check config file
sudo -u picoclaw cat /etc/picoclaw/config.json

# Test running manually
sudo -u picoclaw /usr/local/bin/picoclaw gateway
```

### Permission Issues

```bash
# Check ownership
ls -la /etc/picoclaw/
ls -la /var/lib/picoclaw/

# Fix ownership
sudo chown -R picoclaw:picoclaw /var/lib/picoclaw
sudo chown picoclaw:picoclaw /etc/picoclaw/config.json

# Check config permissions
sudo chmod 600 /etc/picoclaw/config.json
```

### Network Issues

```bash
# Check if port is listening
sudo ss -tlnp | grep 18790

# Check firewall
sudo ufw status
sudo iptables -L -n

# Test health endpoint
curl http://localhost:18790/health
```

### Reset Service

```bash
# Full reset
sudo systemctl stop picoclaw
sudo systemctl reset-failed picoclaw
sudo systemctl daemon-reload
sudo systemctl start picoclaw
```

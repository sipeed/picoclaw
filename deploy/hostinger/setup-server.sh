#!/usr/bin/env bash
# ============================================================
# PicoClaw - Hostinger VPS Initial Setup
# ============================================================
# Run this script ONCE on a fresh Hostinger VPS to prepare it
# for PicoClaw deployment.
#
# Usage:
#   ssh root@YOUR_VPS_IP 'bash -s' < deploy/hostinger/setup-server.sh
#
# Or copy and run directly on the server:
#   scp deploy/hostinger/setup-server.sh root@YOUR_VPS_IP:/tmp/
#   ssh root@YOUR_VPS_IP 'bash /tmp/setup-server.sh'
# ============================================================

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log()   { echo -e "${GREEN}[SETUP]${NC} $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC} $*"; }
error() { echo -e "${RED}[ERROR]${NC} $*"; exit 1; }

# ── Configuration ──────────────────────────────────────
PICOCLAW_USER="picoclaw"
PICOCLAW_HOME="/opt/picoclaw"
DEPLOY_METHOD="${1:-docker}"  # "docker" or "binary"

log "PicoClaw Hostinger VPS Setup"
log "Deploy method: ${DEPLOY_METHOD}"
echo ""

# ── 1. System Update ──────────────────────────────────
log "Updating system packages..."
if command -v apt-get &>/dev/null; then
    export DEBIAN_FRONTEND=noninteractive
    apt-get update -qq
    apt-get upgrade -y -qq
    apt-get install -y -qq curl wget git ufw fail2ban unzip jq
elif command -v yum &>/dev/null; then
    yum update -y -q
    yum install -y -q curl wget git firewalld fail2ban unzip jq
elif command -v dnf &>/dev/null; then
    dnf update -y -q
    dnf install -y -q curl wget git firewalld fail2ban unzip jq
else
    error "Unsupported package manager. This script supports apt, yum, and dnf."
fi

# ── 2. Create dedicated user ─────────────────────────
if ! id "${PICOCLAW_USER}" &>/dev/null; then
    log "Creating dedicated user: ${PICOCLAW_USER}"
    useradd --system --create-home --home-dir "${PICOCLAW_HOME}" \
        --shell /bin/bash "${PICOCLAW_USER}"
else
    log "User ${PICOCLAW_USER} already exists"
fi

# ── 3. Create directory structure ─────────────────────
log "Creating directory structure..."
mkdir -p "${PICOCLAW_HOME}"/{bin,config,workspace,logs,backups}
chown -R "${PICOCLAW_USER}:${PICOCLAW_USER}" "${PICOCLAW_HOME}"

# ── 4. Install Docker (if docker method) ─────────────
if [ "${DEPLOY_METHOD}" = "docker" ]; then
    if ! command -v docker &>/dev/null; then
        log "Installing Docker..."
        curl -fsSL https://get.docker.com | sh
        systemctl enable docker
        systemctl start docker
        usermod -aG docker "${PICOCLAW_USER}"
        log "Docker installed successfully"
    else
        log "Docker already installed: $(docker --version)"
    fi

    # Install Docker Compose plugin if not present
    if ! docker compose version &>/dev/null; then
        log "Installing Docker Compose plugin..."
        apt-get install -y -qq docker-compose-plugin 2>/dev/null || \
        yum install -y -q docker-compose-plugin 2>/dev/null || \
        dnf install -y -q docker-compose-plugin 2>/dev/null || {
            COMPOSE_VERSION=$(curl -s https://api.github.com/repos/docker/compose/releases/latest | jq -r .tag_name)
            curl -fsSL "https://github.com/docker/compose/releases/download/${COMPOSE_VERSION}/docker-compose-$(uname -s)-$(uname -m)" \
                -o /usr/local/bin/docker-compose
            chmod +x /usr/local/bin/docker-compose
        }
    fi
    log "Docker Compose ready: $(docker compose version 2>/dev/null || docker-compose --version 2>/dev/null)"
fi

# ── 5. Install Go (if binary method) ─────────────────
if [ "${DEPLOY_METHOD}" = "binary" ]; then
    if ! command -v go &>/dev/null; then
        log "Installing Go..."
        GO_VERSION="1.23.4"
        ARCH=$(uname -m)
        case "${ARCH}" in
            x86_64)  GO_ARCH="amd64" ;;
            aarch64) GO_ARCH="arm64" ;;
            *)       GO_ARCH="${ARCH}" ;;
        esac
        curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-${GO_ARCH}.tar.gz" -o /tmp/go.tar.gz
        rm -rf /usr/local/go
        tar -C /usr/local -xzf /tmp/go.tar.gz
        rm /tmp/go.tar.gz
        echo 'export PATH=$PATH:/usr/local/go/bin' > /etc/profile.d/go.sh
        export PATH=$PATH:/usr/local/go/bin
        log "Go installed: $(go version)"
    else
        log "Go already installed: $(go version)"
    fi

    # Install make if not present
    if ! command -v make &>/dev/null; then
        apt-get install -y -qq make 2>/dev/null || \
        yum install -y -q make 2>/dev/null || \
        dnf install -y -q make 2>/dev/null
    fi
fi

# ── 6. Configure Firewall ────────────────────────────
log "Configuring firewall..."
if command -v ufw &>/dev/null; then
    ufw --force reset
    ufw default deny incoming
    ufw default allow outgoing
    ufw allow ssh
    # Port 18790 is NOT opened publicly - accessible only via Tailscale
    ufw --force enable
    log "UFW firewall configured (port 18790 is tailscale-only)"
elif command -v firewall-cmd &>/dev/null; then
    systemctl enable firewalld
    systemctl start firewalld
    firewall-cmd --permanent --add-service=ssh
    # Port 18790 is NOT opened publicly - accessible only via Tailscale
    firewall-cmd --reload
    log "firewalld configured (port 18790 is tailscale-only)"
fi

# ── 6b. Install and configure Tailscale ──────────────
log "Installing Tailscale..."
if ! command -v tailscale &>/dev/null; then
    curl -fsSL https://tailscale.com/install.sh | sh
    log "Tailscale installed"
else
    log "Tailscale already installed: $(tailscale version 2>/dev/null | head -1)"
fi

if [ -n "${TAILSCALE_AUTH_KEY:-}" ]; then
    log "Authenticating Tailscale with auth key..."
    tailscale up --authkey="${TAILSCALE_AUTH_KEY}" --hostname="picoclaw" --ssh
    log "Tailscale authenticated. Configuring serve..."
    tailscale serve --bg http://localhost:18790
    log "Tailscale serve active: https://picoclaw.TAILNET.ts.net -> localhost:18790"
else
    warn "TAILSCALE_AUTH_KEY not set. Run manually after setup:"
    warn "  tailscale up --hostname=picoclaw --ssh"
    warn "  tailscale serve --bg http://localhost:18790"
fi

# ── 7. Configure fail2ban ────────────────────────────
log "Configuring fail2ban..."
systemctl enable fail2ban
systemctl start fail2ban

# ── 8. Create environment template ───────────────────
if [ ! -f "${PICOCLAW_HOME}/config/.env" ]; then
    log "Creating environment template..."
    cat > "${PICOCLAW_HOME}/config/.env" <<'ENVEOF'
# ============================================================
# PicoClaw Production Environment
# ============================================================
# Edit this file with your actual API keys and tokens.
# NEVER commit this file to version control.
# ============================================================

# ── LLM Provider (uncomment one) ──────────────────────
# ANTHROPIC_API_KEY=sk-ant-xxx
# OPENAI_API_KEY=sk-xxx
# OPENROUTER_API_KEY=sk-or-v1-xxx
# GEMINI_API_KEY=xxx

# ── Telegram Bot ─────────────────────────────────────
# Get token from @BotFather on Telegram
PICOCLAW_CHANNELS_TELEGRAM_ENABLED=true
PICOCLAW_CHANNELS_TELEGRAM_TOKEN=

# ── Other Chat Channels ──────────────────────────────
# DISCORD_BOT_TOKEN=xxx

# ── Web Search (optional) ────────────────────────────
# BRAVE_SEARCH_API_KEY=BSA...

# ── Timezone ─────────────────────────────────────────
TZ=America/Sao_Paulo
ENVEOF
    chown "${PICOCLAW_USER}:${PICOCLAW_USER}" "${PICOCLAW_HOME}/config/.env"
    chmod 600 "${PICOCLAW_HOME}/config/.env"
fi

# ── 9. Create config.json template ───────────────────
if [ ! -f "${PICOCLAW_HOME}/config/config.json" ]; then
    log "Creating config.json template..."
    cat > "${PICOCLAW_HOME}/config/config.json" <<'JSONEOF'
{
  "agents": {
    "defaults": {
      "workspace": "/opt/picoclaw/workspace",
      "restrict_to_workspace": true,
      "model": "claude-sonnet-4-20250514",
      "max_tokens": 8192,
      "temperature": 0.7,
      "max_tool_iterations": 20
    }
  },
  "channels": {
    "telegram": {
      "enabled": true,
      "token": "",
      "proxy": "",
      "allow_from": []
    }
  },
  "providers": {
    "anthropic": {
      "api_key": "",
      "api_base": ""
    }
  },
  "heartbeat": {
    "enabled": true,
    "interval": 30
  },
  "gateway": {
    "host": "0.0.0.0",
    "port": 18790
  }
}
JSONEOF
    chown "${PICOCLAW_USER}:${PICOCLAW_USER}" "${PICOCLAW_HOME}/config/config.json"
    chmod 600 "${PICOCLAW_HOME}/config/config.json"
fi

# ── 10. Install systemd service (for binary method) ──
if [ "${DEPLOY_METHOD}" = "binary" ]; then
    log "Installing systemd service..."
    cat > /etc/systemd/system/picoclaw.service <<SVCEOF
[Unit]
Description=PicoClaw AI Assistant Gateway
After=network-online.target
Wants=network-online.target
StartLimitIntervalSec=300
StartLimitBurst=5

[Service]
Type=simple
User=${PICOCLAW_USER}
Group=${PICOCLAW_USER}
WorkingDirectory=${PICOCLAW_HOME}
ExecStart=${PICOCLAW_HOME}/bin/picoclaw gateway
Restart=on-failure
RestartSec=10
StandardOutput=append:${PICOCLAW_HOME}/logs/picoclaw.log
StandardError=append:${PICOCLAW_HOME}/logs/picoclaw-error.log

# Environment
EnvironmentFile=-${PICOCLAW_HOME}/config/.env
Environment=HOME=${PICOCLAW_HOME}
Environment=PICOCLAW_CONFIG=${PICOCLAW_HOME}/config/config.json

# Security hardening
NoNewPrivileges=yes
ProtectSystem=strict
ProtectHome=yes
ReadWritePaths=${PICOCLAW_HOME}
PrivateTmp=yes
ProtectKernelModules=yes
ProtectKernelTunables=yes

[Install]
WantedBy=multi-user.target
SVCEOF
    systemctl daemon-reload
    systemctl enable picoclaw
    log "systemd service installed and enabled"
fi

# ── 11. Create logrotate config ──────────────────────
cat > /etc/logrotate.d/picoclaw <<LOGEOF
${PICOCLAW_HOME}/logs/*.log {
    daily
    missingok
    rotate 14
    compress
    delaycompress
    notifempty
    create 0640 ${PICOCLAW_USER} ${PICOCLAW_USER}
    postrotate
        systemctl reload picoclaw 2>/dev/null || true
    endscript
}
LOGEOF

# ── Done ─────────────────────────────────────────────
echo ""
log "=========================================="
log "  Server setup complete!"
log "=========================================="
echo ""
log "Next steps:"
echo "  1. Edit API keys:  nano ${PICOCLAW_HOME}/config/.env"
echo "  2. Edit config:    nano ${PICOCLAW_HOME}/config/config.json"
if [ "${DEPLOY_METHOD}" = "docker" ]; then
    echo "  3. Deploy:         Run 'make deploy-hostinger' from your local machine"
else
    echo "  3. Deploy:         Run 'make deploy-hostinger' from your local machine"
    echo "  4. Start service:  systemctl start picoclaw"
    echo "  5. Check status:   systemctl status picoclaw"
    echo "  6. View logs:      tail -f ${PICOCLAW_HOME}/logs/picoclaw.log"
fi
echo ""
log "Firewall ports open: SSH (22) only. Port 18790 accessible via Tailscale only."
echo ""

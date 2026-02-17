#!/usr/bin/env bash
# ============================================================
# PicoClaw - Tailscale Setup Script
# ============================================================
# Guides you through Tailscale setup and configuration
#
# Usage:
#   bash deploy/hostinger/setup-tailscale.sh
#   OR
#   make setup-tailscale
# ============================================================

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log()   { echo -e "${GREEN}✓${NC} $*"; }
warn()  { echo -e "${YELLOW}⚠${NC} $*"; }
error() { echo -e "${RED}✗${NC} $*"; exit 1; }
info()  { echo -e "${BLUE}ℹ${NC} $*"; }
header() { echo -e "\n${BLUE}══════════════════════════════════════${NC}\n$*\n${BLUE}══════════════════════════════════════${NC}\n"; }

# ── Configuration ──────────────────────────────────────
SSH_HOST=""
SSH_USER="root"
SSH_PORT="22"

header "🔐 PicoClaw Tailscale Setup"

# ── Step 1: Collect SSH Details ────────────────────────
step_ssh_details() {
    header "Step 1️⃣  SSH Connection Details"

    info "Enter your Hostinger VPS details"
    read -p "Server IP or hostname: " SSH_HOST
    read -p "SSH user (default: root): " SSH_USER_INPUT
    SSH_USER="${SSH_USER_INPUT:-root}"
    read -p "SSH port (default: 22): " SSH_PORT_INPUT
    SSH_PORT="${SSH_PORT_INPUT:-22}"

    log "SSH Details: ${GREEN}${SSH_USER}@${SSH_HOST}:${SSH_PORT}${NC}"
}

# ── Step 2: Install Tailscale ──────────────────────────
step_install_tailscale() {
    header "Step 2️⃣  Install Tailscale on Server"

    info "Installing Tailscale..."

    ssh -p "$SSH_PORT" "${SSH_USER}@${SSH_HOST}" <<'EOF'
        if command -v tailscale &>/dev/null; then
            echo "✓ Tailscale already installed"
            exit 0
        fi

        echo "Installing Tailscale..."
        curl -fsSL https://tailscale.com/install.sh | sh

        if command -v tailscale &>/dev/null; then
            echo "✓ Tailscale installed successfully"
        else
            echo "✗ Failed to install Tailscale"
            exit 1
        fi
EOF

    log "Tailscale installed"
}

# ── Step 3: Authenticate with Tailscale ───────────────
step_authenticate() {
    header "Step 3️⃣  Authenticate with Tailscale"

    info "Opening Tailscale authentication..."
    info "A URL will appear below. Open it in your browser and authorize."
    echo ""

    ssh -p "$SSH_PORT" "${SSH_USER}@${SSH_HOST}" <<'EOF'
        echo "Starting Tailscale authentication..."
        echo "Click the link below or open it in your browser:"
        echo ""

        tailscale up --hostname=picoclaw --ssh

        echo ""
        echo "✓ Tailscale authentication complete"
        tailscale ip -4
EOF

    log "Tailscale authenticated"
}

# ── Step 4: Configure Tailscale Serve ──────────────────
step_configure_serve() {
    header "Step 4️⃣  Configure Tailscale Serve"

    info "Configuring Tailscale to expose PicoClaw on tailnet..."

    ssh -p "$SSH_PORT" "${SSH_USER}@${SSH_HOST}" <<'EOF'
        tailscale serve --bg http://localhost:18790

        echo "✓ Tailscale serve configured"
        echo ""
        echo "Your PicoClaw is now accessible at:"
        tailscale ip -4 | while read ip; do
            echo "  http://$ip:18790"
        done
        echo "  https://picoclaw.$(tailscale status --json | grep -o '"Self":{"ID":"[^"]*' | grep -o '"[^"]*$' | tr -d '"' | sed 's/.*\.//' | head -1).ts.net"
EOF

    log "Tailscale serve active"
}

# ── Step 5: Verify Access ──────────────────────────────
step_verify_access() {
    header "Step 5️⃣  Verify Access"

    info "Testing Tailscale connection..."

    TAILNET_IP=$(ssh -p "$SSH_PORT" "${SSH_USER}@${SSH_HOST}" "tailscale ip -4" 2>/dev/null || echo "")

    if [ -z "$TAILNET_IP" ]; then
        warn "Could not get Tailscale IP"
        return 0
    fi

    log "Tailscale IP: ${BLUE}${TAILNET_IP}${NC}"

    # Try to curl the health endpoint
    if curl -sf "http://${TAILNET_IP}:18790/health" >/dev/null 2>&1; then
        log "✨ PicoClaw is accessible via Tailscale!"
        echo ""
        info "Access PicoClaw at: ${BLUE}http://${TAILNET_IP}:18790${NC}"
    else
        warn "Could not connect to PicoClaw (might be still starting)"
        echo ""
        info "Try manually: ${BLUE}curl http://${TAILNET_IP}:18790/health${NC}"
    fi
}

# ── Step 6: Show Next Steps ────────────────────────────
step_next_steps() {
    header "✨ Tailscale Setup Complete!"

    echo ""
    echo "  ${GREEN}Your PicoClaw VPS is now secured with Tailscale${NC}"
    echo ""
    echo "  ${BLUE}Port 18790 is ${GREEN}NOT${BLUE} accessible from the internet${NC}"
    echo "  ${BLUE}Only accessible via your Tailnet${NC}"
    echo ""
    echo "  ${YELLOW}Next steps:${NC}"
    echo "    1. Get your Tailscale IP:"
    echo "       ${BLUE}ssh ${SSH_USER}@${SSH_HOST} -p ${SSH_PORT} 'tailscale ip -4'${NC}"
    echo ""
    echo "    2. Access PicoClaw:"
    echo "       ${BLUE}http://<TAILSCALE_IP>:18790${NC}"
    echo ""
    echo "    3. SSH via Tailscale:"
    echo "       ${BLUE}tailscale list${NC} (to see devices)"
    echo "       ${BLUE}ssh picoclaw.${USER}.ts.net${NC}"
    echo ""
    echo "    4. Set up Telegram bot:"
    echo "       ${BLUE}make setup-telegram${NC}"
    echo ""
}

# ── Main Execution ────────────────────────────────────
main() {
    info "This script will set up Tailscale to secure your PicoClaw"
    echo ""
    echo "  What it does:"
    echo "    • Installs Tailscale on your Hostinger VPS"
    echo "    • Authenticates with your Tailnet"
    echo "    • Exposes PicoClaw only on Tailscale"
    echo "    • Blocks public internet access to port 18790"
    echo ""

    read -p "Continue? (y/n): " proceed
    [ "$proceed" != "y" ] && error "Aborted by user"

    step_ssh_details
    step_install_tailscale
    step_authenticate
    step_configure_serve
    step_verify_access
    step_next_steps
}

# ── Run ────────────────────────────────────────────────
main

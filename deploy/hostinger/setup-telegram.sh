#!/usr/bin/env bash
# ============================================================
# PicoClaw - Telegram Bot Setup Script
# ============================================================
# Interactive setup for Telegram bot integration
#
# Usage:
#   bash deploy/hostinger/setup-telegram.sh
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
PICOCLAW_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
TELEGRAM_TOKEN=""
GITHUB_REPO=""
GITHUB_TOKEN=""
SSH_HOST=""

header "🤖 PicoClaw Telegram Bot Setup"

# ── Step 1: Create Bot with BotFather ──────────────────
step_create_bot() {
    header "Step 1️⃣  Create Telegram Bot with @BotFather"

    echo "Follow these steps in Telegram:"
    echo ""
    echo "  1. Open Telegram and search for: ${BLUE}@BotFather${NC}"
    echo "  2. Send: ${GREEN}/start${NC}"
    echo "  3. Send: ${GREEN}/newbot${NC}"
    echo "  4. Give it a ${BLUE}Name${NC} (e.g., 'PicoClaw AI')"
    echo "  5. Give it a ${BLUE}Username${NC} (e.g., 'picoclaw_bot')"
    echo "     ${YELLOW}⚠ Must be unique and end with _bot${NC}"
    echo "  6. ${GREEN}Copy the token${NC} provided by BotFather"
    echo ""

    read -p "Paste your bot token here: " TELEGRAM_TOKEN

    if [ -z "$TELEGRAM_TOKEN" ]; then
        error "Bot token cannot be empty!"
    fi

    # Basic validation: should be numbers:letters format
    if [[ ! $TELEGRAM_TOKEN =~ ^[0-9]+:[A-Za-z0-9_-]+$ ]]; then
        warn "Token format looks unusual. Continue? (y/n)"
        read -p "" confirm
        [ "$confirm" != "y" ] && error "Aborted"
    fi

    log "Bot token saved: ${TELEGRAM_TOKEN:0:20}..."
}

# ── Step 2: Test Bot ───────────────────────────────────
step_test_bot() {
    header "Step 2️⃣  Test Your Bot Token"

    info "Validating token with Telegram API..."

    RESPONSE=$(curl -s "https://api.telegram.org/bot${TELEGRAM_TOKEN}/getMe")

    if echo "$RESPONSE" | grep -q '"ok":true'; then
        BOT_USERNAME=$(echo "$RESPONSE" | grep -o '"username":"[^"]*' | cut -d'"' -f4)
        BOT_NAME=$(echo "$RESPONSE" | grep -o '"first_name":"[^"]*' | cut -d'"' -f4)

        log "Bot token is valid! ✨"
        log "Bot Name: ${BLUE}$BOT_NAME${NC}"
        log "Bot Username: ${BLUE}@$BOT_USERNAME${NC}"
        echo ""
        info "Find your bot on Telegram: ${GREEN}@$BOT_USERNAME${NC}"
    else
        error "Invalid bot token! Please check and try again."
    fi
}

# ── Step 3: GitHub Setup ───────────────────────────────
step_github_setup() {
    header "Step 3️⃣  Configure GitHub Secrets"

    info "Checking for 'gh' CLI..."
    if ! command -v gh &>/dev/null; then
        warn "GitHub CLI not installed. Please add the secret manually:"
        echo ""
        echo "  1. Go to: ${BLUE}https://github.com/YOUR_USER/YOUR_REPO/settings/secrets/actions${NC}"
        echo "  2. Click ${GREEN}New repository secret${NC}"
        echo "  3. Name: ${GREEN}PICOCLAW_TELEGRAM_BOT_TOKEN${NC}"
        echo "  4. Value: ${GREEN}${TELEGRAM_TOKEN:0:30}...${NC}"
        echo ""
        return 0
    fi

    # Get repo info
    if [ -z "$GITHUB_REPO" ]; then
        info "Detecting GitHub repository..."
        GITHUB_REPO=$(cd "$PICOCLAW_DIR" && git config --get remote.origin.url | sed 's/.*:\(.*\)\.git/\1/')
    fi

    if [ -z "$GITHUB_REPO" ]; then
        warn "Could not detect GitHub repo. Please enter manually:"
        read -p "GitHub repo (user/repo): " GITHUB_REPO
    fi

    info "Repository: ${BLUE}$GITHUB_REPO${NC}"

    # Try to set secret with gh CLI
    if gh secret set PICOCLAW_TELEGRAM_BOT_TOKEN --body "$TELEGRAM_TOKEN" -R "$GITHUB_REPO" 2>/dev/null; then
        log "GitHub secret configured! 🔐"
    else
        warn "Could not set GitHub secret via CLI"
        echo ""
        echo "Set it manually:"
        echo "  ${BLUE}gh secret set PICOCLAW_TELEGRAM_BOT_TOKEN -b '$TELEGRAM_TOKEN' -R '$GITHUB_REPO'${NC}"
    fi
}

# ── Step 4: Configure Locally (for testing) ───────────
step_configure_local() {
    header "Step 4️⃣  Configure Locally (Optional - for testing)"

    read -p "Configure locally for testing? (y/n): " configure_local

    if [ "$configure_local" != "y" ]; then
        info "Skipping local configuration"
        return 0
    fi

    ENV_FILE="${PICOCLAW_DIR}/config/.env"
    CONFIG_FILE="${PICOCLAW_DIR}/config/config.json"

    info "Updating .env file..."
    if grep -q "PICOCLAW_CHANNELS_TELEGRAM_TOKEN" "$ENV_FILE" 2>/dev/null; then
        sed -i "s|^PICOCLAW_CHANNELS_TELEGRAM_TOKEN=.*|PICOCLAW_CHANNELS_TELEGRAM_TOKEN=$TELEGRAM_TOKEN|" "$ENV_FILE"
    else
        echo "PICOCLAW_CHANNELS_TELEGRAM_TOKEN=$TELEGRAM_TOKEN" >> "$ENV_FILE"
    fi

    if grep -q "PICOCLAW_CHANNELS_TELEGRAM_ENABLED" "$ENV_FILE" 2>/dev/null; then
        sed -i 's/^PICOCLAW_CHANNELS_TELEGRAM_ENABLED=.*/PICOCLAW_CHANNELS_TELEGRAM_ENABLED=true/' "$ENV_FILE"
    else
        echo "PICOCLAW_CHANNELS_TELEGRAM_ENABLED=true" >> "$ENV_FILE"
    fi

    log ".env file updated"

    # Update config.json
    if command -v jq &>/dev/null && [ -f "$CONFIG_FILE" ]; then
        info "Updating config.json..."
        jq '.channels.telegram.enabled = true | .channels.telegram.token = ""' "$CONFIG_FILE" > "${CONFIG_FILE}.tmp"
        mv "${CONFIG_FILE}.tmp" "$CONFIG_FILE"
        log "config.json updated"
    fi
}

# ── Step 5: Deploy ───────────────────────────────────
step_deploy() {
    header "Step 5️⃣  Deploy to Hostinger"

    read -p "Ready to deploy? (y/n): " deploy_ready

    if [ "$deploy_ready" != "y" ]; then
        info "Skipping deployment"
        return 0
    fi

    info "Pushing code changes..."
    cd "$PICOCLAW_DIR"

    if git diff --quiet; then
        info "No local changes to commit"
    else
        warn "Local changes detected"
        git status
        read -p "Commit and push? (y/n): " commit_ready
        if [ "$commit_ready" = "y" ]; then
            git add .
            git commit -m "chore: configure telegram bot integration"
            git push origin claude/hostinger-remote-deployment-TGVof
        fi
    fi

    info "GitHub Actions deployment triggered..."
    info "Check status at: ${BLUE}https://github.com/$GITHUB_REPO/actions${NC}"
}

# ── Step 6: Verification ───────────────────────────────
step_verify() {
    header "Step 6️⃣  Verify Installation"

    info "Your Telegram bot is now live!"
    echo ""
    echo "  ${GREEN}Find your bot on Telegram and send: /start${NC}"
    echo ""

    read -p "Check logs on server? (y/n): " check_logs

    if [ "$check_logs" != "y" ]; then
        return 0
    fi

    info "Enter SSH details:"
    read -p "Server IP or hostname: " SSH_HOST
    read -p "SSH user (default: root): " SSH_USER
    SSH_USER="${SSH_USER:-root}"

    info "Connecting to server..."
    ssh -t "${SSH_USER}@${SSH_HOST}" \
        'tail -50 /opt/picoclaw/logs/picoclaw.log | grep -i telegram'
}

# ── Main Execution ────────────────────────────────────
main() {
    info "This script will:"
    echo "  1. Create a Telegram bot with @BotFather"
    echo "  2. Validate the bot token"
    echo "  3. Configure GitHub Secrets for CI/CD"
    echo "  4. (Optional) Configure locally for testing"
    echo "  5. Deploy to your Hostinger VPS"
    echo "  6. Verify the installation"
    echo ""

    read -p "Continue? (y/n): " proceed
    [ "$proceed" != "y" ] && error "Aborted by user"

    step_create_bot
    step_test_bot
    step_github_setup
    step_configure_local
    step_deploy
    step_verify

    header "✨ Setup Complete!"
    echo ""
    echo "  ${GREEN}Your PicoClaw Telegram bot is ready!${NC}"
    echo ""
    echo "  Next steps:"
    echo "    1. Open Telegram and find your bot"
    echo "    2. Send /start"
    echo "    3. Start chatting!"
    echo ""
    echo "  For troubleshooting, see: ${BLUE}docs/TELEGRAM_SETUP.md${NC}"
    echo ""
}

# ── Run ────────────────────────────────────────────────
main

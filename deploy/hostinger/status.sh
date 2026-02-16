#!/usr/bin/env bash
# ============================================================
# PicoClaw - Check Remote Server Status
# ============================================================
# Usage:
#   ./deploy/hostinger/status.sh -h YOUR_VPS_IP
# ============================================================

set -euo pipefail

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log()  { echo -e "${GREEN}[STATUS]${NC} $*"; }
info() { echo -e "${BLUE}[INFO]${NC} $*"; }

HOST="${HOSTINGER_HOST:-}"
USER="${HOSTINGER_USER:-root}"
SSH_KEY="${HOSTINGER_SSH_KEY:-${HOME}/.ssh/id_rsa}"
SSH_PORT="${HOSTINGER_SSH_PORT:-22}"
METHOD="${HOSTINGER_DEPLOY_METHOD:-docker}"

while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--host)   HOST="$2"; shift 2 ;;
        -u|--user)   USER="$2"; shift 2 ;;
        -k|--key)    SSH_KEY="$2"; shift 2 ;;
        -m|--method) METHOD="$2"; shift 2 ;;
        -p|--port)   SSH_PORT="$2"; shift 2 ;;
        *) shift ;;
    esac
done

[ -z "${HOST}" ] && { echo "Usage: $0 -h HOST"; exit 1; }

SSH_OPTS="-o StrictHostKeyChecking=accept-new -o ConnectTimeout=10 -p ${SSH_PORT} -i ${SSH_KEY}"
SSH_CMD="ssh ${SSH_OPTS} ${USER}@${HOST}"

log "Checking PicoClaw status on ${HOST}..."
echo ""

${SSH_CMD} <<REMOTEOF
echo "── System ──────────────────────────────────"
echo "Hostname:  \$(hostname)"
echo "Uptime:    \$(uptime -p)"
echo "Memory:    \$(free -h | awk '/Mem:/ {print \$3 "/" \$2}')"
echo "Disk:      \$(df -h / | awk 'NR==2 {print \$3 "/" \$2 " (" \$5 " used)"}')"
echo "Load:      \$(cat /proc/loadavg | awk '{print \$1, \$2, \$3}')"
echo ""

echo "── PicoClaw ────────────────────────────────"
if [ "${METHOD}" = "docker" ]; then
    if command -v docker &>/dev/null; then
        echo "Method:    Docker"
        docker compose -f /opt/picoclaw/docker-compose.yml ps 2>/dev/null || echo "Container: not running"
        echo ""
        echo "── Recent Logs ─────────────────────────────"
        docker compose -f /opt/picoclaw/docker-compose.yml logs --tail=10 picoclaw-gateway 2>/dev/null || true
    else
        echo "Docker not installed"
    fi
else
    echo "Method:    Binary (systemd)"
    if systemctl is-active --quiet picoclaw 2>/dev/null; then
        echo "Status:    RUNNING"
        version=\$(/opt/picoclaw/bin/picoclaw version 2>/dev/null || echo "unknown")
        echo "Version:   \${version}"
    else
        echo "Status:    STOPPED"
    fi
    systemctl status picoclaw --no-pager -l 2>/dev/null || true
    echo ""
    echo "── Recent Logs ─────────────────────────────"
    tail -10 /opt/picoclaw/logs/picoclaw.log 2>/dev/null || echo "No logs found"
fi

echo ""
echo "── Health Check ────────────────────────────"
if curl -sf http://localhost:18790/health > /dev/null 2>&1; then
    echo "Gateway:   HEALTHY (port 18790)"
else
    echo "Gateway:   UNREACHABLE"
fi

echo ""
echo "── Firewall ────────────────────────────────"
if command -v ufw &>/dev/null; then
    ufw status 2>/dev/null | head -10
elif command -v firewall-cmd &>/dev/null; then
    firewall-cmd --list-all 2>/dev/null | head -10
fi
REMOTEOF

echo ""
log "Status check complete"

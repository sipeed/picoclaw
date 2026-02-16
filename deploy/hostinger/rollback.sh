#!/usr/bin/env bash
# ============================================================
# PicoClaw - Rollback to Previous Version
# ============================================================
# Usage:
#   ./deploy/hostinger/rollback.sh -h YOUR_VPS_IP
# ============================================================

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log()   { echo -e "${GREEN}[ROLLBACK]${NC} $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC} $*"; }
error() { echo -e "${RED}[ERROR]${NC} $*"; exit 1; }

# Configuration
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
        *) error "Unknown option: $1" ;;
    esac
done

[ -z "${HOST}" ] && error "Host required. Use -h/--host or set HOSTINGER_HOST env var."

SSH_OPTS="-o StrictHostKeyChecking=accept-new -o ConnectTimeout=10 -p ${SSH_PORT} -i ${SSH_KEY}"
SSH_CMD="ssh ${SSH_OPTS} ${USER}@${HOST}"

log "Rolling back PicoClaw on ${HOST}..."

if [ "${METHOD}" = "docker" ]; then
    ${SSH_CMD} <<'REMOTEOF'
set -e
cd /opt/picoclaw

# List available images
echo "Available images:"
docker images --format "table {{.Repository}}\t{{.Tag}}\t{{.CreatedAt}}" | grep picoclaw || true

# Docker rollback: restart with previous image
echo "[REMOTE] Stopping current container..."
docker compose down --timeout 30

echo "[REMOTE] Starting previous version..."
# The previous image should still be cached
docker compose up -d picoclaw-gateway

sleep 5
if docker compose ps picoclaw-gateway | grep -q "Up"; then
    echo "[REMOTE] Rollback successful!"
    docker compose ps
else
    echo "[REMOTE] Rollback failed!"
    docker compose logs --tail=20 picoclaw-gateway
    exit 1
fi
REMOTEOF
else
    ${SSH_CMD} <<'REMOTEOF'
set -e
BACKUP_DIR="/opt/picoclaw/backups"

# Find latest backup
LATEST_BACKUP=$(ls -t ${BACKUP_DIR}/picoclaw-*.bak 2>/dev/null | head -1)

if [ -z "${LATEST_BACKUP}" ]; then
    echo "[REMOTE] No backup found to rollback to!"
    exit 1
fi

echo "[REMOTE] Rolling back to: ${LATEST_BACKUP}"

# Stop service
systemctl stop picoclaw

# Restore binary
cp "${LATEST_BACKUP}" /opt/picoclaw/bin/picoclaw
chmod +x /opt/picoclaw/bin/picoclaw

# Start service
systemctl start picoclaw

sleep 3
if systemctl is-active --quiet picoclaw; then
    echo "[REMOTE] Rollback successful!"
    systemctl status picoclaw --no-pager
else
    echo "[REMOTE] Rollback failed!"
    journalctl -u picoclaw --no-pager -n 20
    exit 1
fi
REMOTEOF
fi

log "Rollback complete!"

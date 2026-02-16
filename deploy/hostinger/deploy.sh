#!/usr/bin/env bash
# ============================================================
# PicoClaw - Remote Deploy to Hostinger VPS
# ============================================================
# Deploys PicoClaw to a remote Hostinger VPS via SSH.
# Supports both Docker and binary deployment methods.
#
# Usage:
#   ./deploy/hostinger/deploy.sh [OPTIONS]
#
# Options:
#   -h, --host HOST       VPS IP or hostname (required, or set HOSTINGER_HOST)
#   -u, --user USER       SSH user (default: root)
#   -k, --key KEY         SSH key path (default: ~/.ssh/id_rsa)
#   -m, --method METHOD   Deploy method: "docker" or "binary" (default: docker)
#   -p, --port PORT       SSH port (default: 22)
#   --build-local         Build binary locally and upload (for binary method)
#   --help                Show this help
# ============================================================

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log()   { echo -e "${GREEN}[DEPLOY]${NC} $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC} $*"; }
error() { echo -e "${RED}[ERROR]${NC} $*"; exit 1; }
info()  { echo -e "${BLUE}[INFO]${NC} $*"; }

# ── Default Configuration ────────────────────────────
HOST="${HOSTINGER_HOST:-}"
USER="${HOSTINGER_USER:-root}"
SSH_KEY="${HOSTINGER_SSH_KEY:-${HOME}/.ssh/id_rsa}"
SSH_PORT="${HOSTINGER_SSH_PORT:-22}"
METHOD="${HOSTINGER_DEPLOY_METHOD:-docker}"
BUILD_LOCAL=false
REMOTE_DIR="/opt/picoclaw"

# ── Parse Arguments ──────────────────────────────────
while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--host)     HOST="$2"; shift 2 ;;
        -u|--user)     USER="$2"; shift 2 ;;
        -k|--key)      SSH_KEY="$2"; shift 2 ;;
        -m|--method)   METHOD="$2"; shift 2 ;;
        -p|--port)     SSH_PORT="$2"; shift 2 ;;
        --build-local) BUILD_LOCAL=true; shift ;;
        --help)
            head -20 "$0" | tail -15
            exit 0
            ;;
        *) error "Unknown option: $1" ;;
    esac
done

# ── Validate ─────────────────────────────────────────
[ -z "${HOST}" ] && error "Host required. Use -h/--host or set HOSTINGER_HOST env var."
[ ! -f "${SSH_KEY}" ] && error "SSH key not found: ${SSH_KEY}"

SSH_OPTS="-o StrictHostKeyChecking=accept-new -o ConnectTimeout=10 -p ${SSH_PORT} -i ${SSH_KEY}"
SSH_CMD="ssh ${SSH_OPTS} ${USER}@${HOST}"
SCP_CMD="scp ${SSH_OPTS}"

# ── Verify Connection ────────────────────────────────
log "Verifying SSH connection to ${USER}@${HOST}:${SSH_PORT}..."
${SSH_CMD} "echo 'Connection OK'" || error "Cannot connect to ${HOST}. Check SSH settings."

# ── Get project root ─────────────────────────────────
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${PROJECT_ROOT}"

log "Deploying PicoClaw to ${HOST} via ${METHOD} method"
echo ""

# ── Deploy via Docker ────────────────────────────────
deploy_docker() {
    log "=== Docker Deployment ==="

    # 1. Sync project files to server
    log "Syncing project files..."
    rsync -avz --delete \
        -e "ssh ${SSH_OPTS}" \
        --exclude '.git' \
        --exclude 'build/' \
        --exclude '.env' \
        --exclude 'config/config.json' \
        --exclude '*.test' \
        "${PROJECT_ROOT}/" "${USER}@${HOST}:${REMOTE_DIR}/src/"

    # 2. Copy docker-compose production file
    log "Copying Docker Compose production config..."
    ${SCP_CMD} "${PROJECT_ROOT}/deploy/hostinger/docker-compose.production.yml" \
        "${USER}@${HOST}:${REMOTE_DIR}/docker-compose.yml"

    # 3. Build and restart on server
    log "Building and starting containers on server..."
    ${SSH_CMD} <<'REMOTEOF'
set -e
cd /opt/picoclaw

# Build the image
echo "[REMOTE] Building Docker image..."
docker compose build --no-cache picoclaw-gateway

# Stop existing container gracefully
echo "[REMOTE] Stopping existing container..."
docker compose down --timeout 30 2>/dev/null || true

# Start new container
echo "[REMOTE] Starting PicoClaw gateway..."
docker compose up -d picoclaw-gateway

# Wait and verify
sleep 5
if docker compose ps picoclaw-gateway | grep -q "Up"; then
    echo "[REMOTE] PicoClaw is running!"
    docker compose ps
else
    echo "[REMOTE] ERROR: Container failed to start"
    docker compose logs --tail=50 picoclaw-gateway
    exit 1
fi

# Health check
echo "[REMOTE] Checking health..."
sleep 3
if curl -sf http://localhost:18790/health > /dev/null 2>&1; then
    echo "[REMOTE] Health check PASSED"
else
    echo "[REMOTE] Health check failed (may still be starting up)"
fi

# Cleanup old images
echo "[REMOTE] Cleaning up old Docker images..."
docker image prune -f
REMOTEOF

    log "Docker deployment complete!"
}

# ── Deploy via Binary ────────────────────────────────
deploy_binary() {
    log "=== Binary Deployment ==="

    if [ "${BUILD_LOCAL}" = true ]; then
        # Build locally for linux/amd64
        log "Building binary locally for linux/amd64..."
        cd "${PROJECT_ROOT}"
        make generate
        GOOS=linux GOARCH=amd64 go build -v \
            -ldflags "-X main.version=$(git describe --tags --always --dirty 2>/dev/null || echo dev) \
                      -X main.gitCommit=$(git rev-parse --short=8 HEAD 2>/dev/null || echo dev) \
                      -X main.buildTime=$(date +%FT%T%z)" \
            -o build/picoclaw-linux-amd64 ./cmd/picoclaw
        BINARY_PATH="build/picoclaw-linux-amd64"
        log "Binary built: ${BINARY_PATH}"
    else
        # Build on server
        log "Syncing source code to server..."
        rsync -avz --delete \
            -e "ssh ${SSH_OPTS}" \
            --exclude '.git' \
            --exclude 'build/' \
            --exclude '.env' \
            --exclude 'config/config.json' \
            "${PROJECT_ROOT}/" "${USER}@${HOST}:${REMOTE_DIR}/src/"

        log "Building on server..."
        ${SSH_CMD} <<'REMOTEOF'
set -e
cd /opt/picoclaw/src
export PATH=$PATH:/usr/local/go/bin
make build
cp build/picoclaw /opt/picoclaw/bin/picoclaw
chmod +x /opt/picoclaw/bin/picoclaw
echo "[REMOTE] Build complete: $(/opt/picoclaw/bin/picoclaw version 2>/dev/null || echo 'built')"
REMOTEOF
    fi

    if [ "${BUILD_LOCAL}" = true ]; then
        # Upload binary
        log "Uploading binary to server..."
        ${SCP_CMD} "${BINARY_PATH}" "${USER}@${HOST}:${REMOTE_DIR}/bin/picoclaw"
        ${SSH_CMD} "chmod +x ${REMOTE_DIR}/bin/picoclaw"

        # Initialize workspace if needed
        ${SSH_CMD} <<REMOTEOF
set -e
if [ ! -d "${REMOTE_DIR}/workspace/memory" ]; then
    echo "[REMOTE] Initializing workspace..."
    sudo -u picoclaw ${REMOTE_DIR}/bin/picoclaw onboard
fi
REMOTEOF
    fi

    # Restart service
    log "Restarting PicoClaw service..."
    ${SSH_CMD} <<'REMOTEOF'
set -e
echo "[REMOTE] Restarting picoclaw service..."
systemctl restart picoclaw

sleep 3
if systemctl is-active --quiet picoclaw; then
    echo "[REMOTE] PicoClaw is running!"
    systemctl status picoclaw --no-pager
else
    echo "[REMOTE] ERROR: Service failed to start"
    journalctl -u picoclaw --no-pager -n 30
    exit 1
fi

# Health check
sleep 3
if curl -sf http://localhost:18790/health > /dev/null 2>&1; then
    echo "[REMOTE] Health check PASSED"
else
    echo "[REMOTE] Health check failed (may still be starting up)"
    tail -20 /opt/picoclaw/logs/picoclaw.log 2>/dev/null || true
fi
REMOTEOF

    log "Binary deployment complete!"
}

# ── Execute Deploy ───────────────────────────────────
DEPLOY_START=$(date +%s)

case "${METHOD}" in
    docker) deploy_docker ;;
    binary) deploy_binary ;;
    *) error "Unknown method: ${METHOD}. Use 'docker' or 'binary'." ;;
esac

DEPLOY_END=$(date +%s)
DEPLOY_DURATION=$((DEPLOY_END - DEPLOY_START))

echo ""
log "=========================================="
log "  Deployment successful!"
log "  Duration: ${DEPLOY_DURATION}s"
log "  Host: ${HOST}"
log "  Method: ${METHOD}"
log "=========================================="
echo ""
info "Useful commands:"
echo "  Check status:  ${SSH_CMD} 'systemctl status picoclaw'"
echo "  View logs:     ${SSH_CMD} 'tail -f /opt/picoclaw/logs/picoclaw.log'"
echo "  Health check:  curl http://${HOST}:18790/health"
echo ""

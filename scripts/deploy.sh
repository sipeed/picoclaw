#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
BINARY_NAME="picoclaw"
BUILD_DIR="build"
INSTALL_BIN_DIR="${HOME}/.local/bin"
GATEWAY_LOG="/tmp/picoclaw-gateway.log"
HEALTH_URL="http://localhost:18790/health"

# ── step 1: build ────────────────────────────────────────────
echo "==> Building..."
"${SCRIPT_DIR}/build.sh"

# ── step 2: stop old gateway ─────────────────────────────────
OLD_PID=$(pgrep -f "${BINARY_NAME} gateway" 2>/dev/null || true)

if [[ -n "$OLD_PID" ]]; then
    echo "==> Stopping gateway (PID ${OLD_PID})..."
    kill "$OLD_PID" 2>/dev/null || true

    # Wait up to 5 seconds for graceful shutdown
    for i in $(seq 1 10); do
        if ! kill -0 "$OLD_PID" 2>/dev/null; then
            break
        fi
        sleep 0.5
    done

    # Force kill if still running
    if kill -0 "$OLD_PID" 2>/dev/null; then
        echo "    Force killing..."
        kill -9 "$OLD_PID" 2>/dev/null || true
        sleep 0.5
    fi
    echo "    Gateway stopped."
else
    echo "==> No running gateway found."
fi

# ── step 3: install new binary ───────────────────────────────
echo "==> Installing..."
mkdir -p "$INSTALL_BIN_DIR"
cp "${BUILD_DIR}/${BINARY_NAME}" "${INSTALL_BIN_DIR}/${BINARY_NAME}.new"
chmod +x "${INSTALL_BIN_DIR}/${BINARY_NAME}.new"
mv -f "${INSTALL_BIN_DIR}/${BINARY_NAME}.new" "${INSTALL_BIN_DIR}/${BINARY_NAME}"
echo "    Installed to ${INSTALL_BIN_DIR}/${BINARY_NAME}"

# ── step 4: start new gateway ────────────────────────────────
echo "==> Starting gateway..."
nohup "${INSTALL_BIN_DIR}/${BINARY_NAME}" gateway > "$GATEWAY_LOG" 2>&1 &
NEW_PID=$!
echo "    PID: ${NEW_PID}"
echo "    Log: ${GATEWAY_LOG}"

# ── step 5: health check ─────────────────────────────────────
echo "==> Waiting for health check..."
sleep 2

if curl -sf "$HEALTH_URL" > /dev/null 2>&1; then
    echo "    Health check passed."
    echo ""
    echo "Deploy complete. Gateway running as PID ${NEW_PID}."
else
    echo "    Health check failed (gateway may still be starting)."
    echo "    Check logs: tail -f ${GATEWAY_LOG}"
fi

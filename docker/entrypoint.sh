#!/bin/sh
set -e

# First-run: neither config nor workspace exists.
# If config.json is already mounted but workspace is missing we skip onboard to
# avoid the interactive "Overwrite? (y/n)" prompt hanging in a non-TTY container.
if [ ! -d "${HOME}/.picoclaw/workspace" ] && [ ! -f "${HOME}/.picoclaw/config.json" ]; then
    picoclaw onboard
    echo ""
    echo "First-run setup complete."
    echo "Edit ${HOME}/.picoclaw/config.json (add your API key, etc.) then restart the container."
    exit 0
fi

# Ensure bsky CLI is on PATH (symlink from skill scripts)
BSKY_SCRIPT="${HOME}/.picoclaw/workspace/skills/bluesky/scripts/bsky.py"
if [ -f "$BSKY_SCRIPT" ] && [ ! -e /usr/local/bin/bsky ]; then
    chmod +x "$BSKY_SCRIPT"
    ln -sf "$BSKY_SCRIPT" /usr/local/bin/bsky
    echo "bsky CLI symlinked to PATH"
fi

# Start virtual framebuffer for headed chromium (agent-browser)
if [ -n "$DISPLAY" ] && command -v Xvfb >/dev/null 2>&1; then
    Xvfb "$DISPLAY" -screen 0 1280x1024x24 -ac &
    sleep 1
    echo "Xvfb running on $DISPLAY"
fi

# Start Copilot headless proxy if token is provided
if [ -n "$COPILOT_GITHUB_TOKEN" ] && command -v copilot >/dev/null 2>&1; then
    COPILOT_PORT="${COPILOT_PORT:-4321}"
    echo "Starting Copilot headless proxy on port ${COPILOT_PORT}..."
    copilot --headless --port "$COPILOT_PORT" &
    COPILOT_PID=$!
    sleep 2
    if kill -0 "$COPILOT_PID" 2>/dev/null; then
        echo "Copilot proxy running (PID ${COPILOT_PID})"
    else
        echo "WARNING: Copilot proxy failed to start, continuing without it"
    fi
fi

# Start launcher (web UI) if available, otherwise fall back to gateway only.
# The launcher auto-starts its own gateway subprocess.
# Tail log files to stdout so they appear in `docker logs`.
LOG_DIR="${HOME}/.picoclaw/logs"
mkdir -p "$LOG_DIR"
touch "$LOG_DIR/launcher.log" "$LOG_DIR/gateway.log"
tail -F "$LOG_DIR/launcher.log" "$LOG_DIR/gateway.log" &

if command -v picoclaw-launcher >/dev/null 2>&1; then
    exec picoclaw-launcher -public -no-browser "$@"
else
    exec picoclaw gateway -d "$@"
fi
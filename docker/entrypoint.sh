#!/bin/sh
set -e

# First-run: neither config nor workspace exists.
# If config.json is already mounted but workspace is missing we skip onboard to
# avoid the interactive "Overwrite? (y/n)" prompt hanging in a non-TTY container.
BIN="${JANE_AI_BINARY:-jane-ai}"
if ! command -v "$BIN" >/dev/null 2>&1; then
    BIN="picoclaw"
fi

HOME_DIR="${JANE_AI_HOME:-${PICOCLAW_HOME:-${HOME}/.jane-ai}}"
if [ ! -e "${HOME_DIR}/config.json" ] && [ -e "${HOME}/.picoclaw/config.json" ]; then
    HOME_DIR="${HOME}/.picoclaw"
fi

if [ ! -d "${HOME_DIR}/workspace" ] && [ ! -f "${HOME_DIR}/config.json" ]; then
    "$BIN" onboard
    echo ""
    echo "First-run setup complete."
    echo "Edit ${HOME_DIR}/config.json (add your API key, etc.) then restart the container."
    exit 0
fi

exec "$BIN" gateway "$@"

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

# Run bootstrap on every start
BOOTSTRAP="${HOME}/.picoclaw/workspace/bootstrap.sh"
ERROR_LOG="${HOME}/.picoclaw/workspace/bootstrap.error.log"

if [ -f "$BOOTSTRAP" ]; then
    echo "[bootstrap] Running bootstrap.sh..."
    if sh "$BOOTSTRAP" 2>"$ERROR_LOG"; then
        # Success — clear any previous error log
        rm -f "$ERROR_LOG"
    else
        echo "[bootstrap] WARNING: bootstrap.sh failed. See $ERROR_LOG for details."
        echo "[bootstrap] Some capabilities may be missing. The agent will attempt to fix bootstrap.sh."
        # Do NOT exit — continue starting the gateway so the agent can be reached
    fi
fi

exec picoclaw gateway "$@"

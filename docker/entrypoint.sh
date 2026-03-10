#!/bin/sh
set -e

if [ "$#" -eq 0 ]; then
    set -- gateway
fi

# First-run bootstrap for non-interactive Docker execution.
# We key off config.json because `picoclaw onboard` only prompts when the config
# already exists. This also repairs partial states where the workspace exists
# but config.json was never created.
if [ "$1" = "gateway" ] || [ "$1" = "agent" ]; then
    if [ ! -f "${HOME}/.picoclaw/config.json" ]; then
        picoclaw onboard
        echo ""
        echo "First-run setup complete."
        echo "Edit ${HOME}/.picoclaw/config.json (add your API key, etc.) then restart the container."
        exit 0
    fi
fi

exec picoclaw "$@"

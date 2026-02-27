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

exec picoclaw gateway "$@"

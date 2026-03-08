#!/bin/sh
# bootstrap.sh — runs on every container start.
# Edit this file to persist capabilities across restarts.
# The agent will modify this file when you ask it to install tools.

# Example: GitHub CLI (gh)
if ! command -v gh >/dev/null 2>&1; then
    echo "[bootstrap] Installing gh CLI..."
    apk add --no-cache github-cli
fi

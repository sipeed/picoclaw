#!/bin/bash
# ntfy_send.sh — ntfy notification sender for prayer-times skill
# Auto-loads config from skills/prayer-times/data/config (reads NTFY_TOPIC)
#
# Usage (from workspace root):
#   bash skills/prayer-times/scripts/ntfy_send.sh "message"
#   bash skills/prayer-times/scripts/ntfy_send.sh "message" --title "T" --tags "t"
#
# If NTFY_TOPIC is empty or config missing, silently exits (exit 0).

set -euo pipefail

# Auto-load config relative to this script
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SKILL_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
CONF_FILE="$SKILL_DIR/data/config"

# Source config if exists
NTFY_TOPIC=""
if [ -f "$CONF_FILE" ]; then
    . "$CONF_FILE"
fi

if [ -z "$NTFY_TOPIC" ]; then
    exit 0
fi

message="$1"
if [ -z "$message" ]; then
    echo "Usage: ntfy_send.sh message [--title T] [--tags T] [--priority P]"
    exit 1
fi
shift

# Parse optional flags
title="" tags="" priority=""
while [ $# -gt 0 ]; do
    case "$1" in
        --title)    title="$2"; shift 2 ;;
        --tags)     tags="$2"; shift 2 ;;
        --priority) priority="$2"; shift 2 ;;
        *) shift ;;
    esac
done

# Send notification
HEADERS=""
if [ -n "$title" ]; then
    HEADERS="$HEADERS -H \"Title: $title\""
fi
if [ -n "$tags" ]; then
    HEADERS="$HEADERS -H \"Tags: $tags\""
fi
if [ -n "$priority" ]; then
    HEADERS="$HEADERS -H \"Priority: $priority\""
fi

eval curl -sf $HEADERS -d "\"$message\"" "\"$NTFY_TOPIC\"" || true

#!/bin/bash
# ntfy_send.sh — Stateless ntfy notification sender utility
# Does NOT manage its own config. Reads NTFY_TOPIC from environment variable.
# Each skill is responsible for setting NTFY_TOPIC from its own config.
#
# Usage:
#   NTFY_TOPIC=https://ntfy.sh/topic ntfy_send.sh "message"
#   NTFY_TOPIC=https://ntfy.sh/topic ntfy_send.sh "message" --title "T" --tags "t" --priority "high"
#
# If NTFY_TOPIC is empty, silently skips (exit 0).

set -euo pipefail

if [ -z "${NTFY_TOPIC:-}" ]; then
    # No topic configured — skip silently so cron jobs don't fail
    exit 0
fi

message="${1:-}"
if [ -z "$message" ]; then
    echo "Usage: ntfy_send.sh \"message\" [--title T] [--tags T] [--priority P]"
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

# Build curl args
curl_args=(-sf)
[ -n "$title" ]    && curl_args+=(-H "Title: $title")
[ -n "$tags" ]     && curl_args+=(-H "Tags: $tags")
[ -n "$priority" ] && curl_args+=(-H "Priority: $priority")
curl_args+=(-d "$message" "$NTFY_TOPIC")

curl "${curl_args[@]}" > /dev/null 2>&1 || true

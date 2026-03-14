#!/bin/bash

# service.sh - Service management with auto-rollback support
# A "dead man's switch" for configuration changes.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Validate identifier (alphanumeric, dash, underscore)
assert_is_identifier() {
    if ! echo "$1" | grep -qE '^[a-zA-Z_][a-zA-Z0-9_-]*$'; then
        echo "Error: $2" >&2
        return 1
    fi
}

assert_is_number ()
{
    # Use grep for robust, portable POSIX regex matching.
    if ! echo "$1" | grep -qE '^[0-9]+$';
    then
        echo "Error: $2" >&2
        return 1
    fi
}

SERVICE_NAME="${PICOCLAW_SERVICE_NAME:-picoclaw}"

COMMAND="${1:-help}"
assert_is_identifier "$COMMAND" "Action must be a valid identifier" || exit 1

SERVICE="${2:-$SERVICE_NAME}"
assert_is_identifier "$SERVICE" "Service name must be a valid identifier" || exit 1

CONFIG="${3:-${PICOCLAW_CONFIG:-$HOME/.picoclaw/config.json}}"

[[ -n "$CONFIG" ]] && ! [[ -f "$CONFIG" ]] && echo "Expected file" && exit 1

case "$COMMAND" in

    restart)

        echo "Restarting service: $SERVICE"
        systemctl --user restart "$SERVICE"
        echo "Done."
    ;;

    _rollback)
        
        # Check if marker file exists (auto-rollback was set)
        MARKER="${CONFIG}-PENDING-ROLLBACK"

        if [ ! -f "$MARKER" ]; then
            # No marker = nothing to rollback = success
            exit 0
        fi
    
        # Marker exists - perform rollback
        rm -f "$MARKER"
    
        "$SCRIPT_DIR/config.sh" rollback "$CONFIG"

        echo "[ROLLBACK] Restarting service: $SERVICE"
        systemctl --user restart "$SERVICE" || true
        echo "Rollback complete."

    ;;

    restart-auto-rollback)

        TIMEOUT="${TIMEOUT:-}"
        TIMEOUT="${4:-120}"
        assert_is_number "$TIMEOUT" "Timer must be number of seconds" || exit 1

        # Create marker file
        MARKER="${CONFIG}-PENDING-ROLLBACK"
        
        # Restart service first
        echo "Restarting service: $SERVICE"
        systemctl --user restart "$SERVICE"
        
        echo "Auto-rollback armed for $SERVICE (timeout: ${TIMEOUT}s)"
        echo "Marker: $MARKER"
        
        # Start background timer
        (
            sleep "$TIMEOUT"
            if [ -f "$MARKER" ]; then
                echo "[TIMEOUT] Auto-rollback triggered for $SERVICE"
                bash "$0" _rollback "$SERVICE" "$CONFIG"
            fi
        ) &
        TIMER_PID=$!

        {        
            echo "Timer PID: $TIMER_PID"
            echo "----------------------------------------------------"
            echo "Service will ROLLBACK in $TIMEOUT seconds if not confirmed."
            echo "To confirm: service.sh confirm $SERVICE"
            echo "----------------------------------------------------"
        } | tee "$MARKER"

    ;;

    confirm)

        MARKER="${CONFIG}-PENDING-ROLLBACK"
            
        if [ -n "$MARKER" ] && [ -f "$MARKER" ]; then
            rm -f "$MARKER"
            echo "Rollback cancelled. Changes confirmed."
        else
            echo "No pending rollback to confirm."
        fi
    
    ;;

    help)
        echo "Usage: $0 <command> [args...]"
        echo ""
        echo "Commands:"
        echo "  restart [service]           Restart service (default: \$PICOCLAW_SERVICE_NAME or picoclaw)"
        echo "  auto-rollback [service] [config] [timeout]"
        echo "                              Arm rollback timer, restart service"
        echo "  confirm [service] [config]  Cancel pending rollback"
        echo "  _rollback [service] [config]"
        echo "                              Internal: check marker, rollback if present"
        echo ""
        echo "Environment:"
        echo "  PICOCLAW_SERVICE_NAME       Default service name (default: picoclaw)"
        echo "  PICOCLAW_CONFIG             Default config file  (default: picoclaw)"
        exit 0
        ;;

    *)
        echo "Unknown command: $COMMAND"
        echo "Run '$0 help' for usage."
        exit 1
        ;;
esac

#!/bin/sh

assert_is_identifier ()
{
  # Use grep for robust, portable POSIX regex matching.
  if ! echo "$1" | grep -qE '^[a-zA-Z0-9_.-]+$';
  then
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

assert_is_identifier "$1" "Action selector must be a simple identifier" || exit 1

case "$1/$OSTYPE" in
    *)
        PICOCLAW_SERVICE_NAME="${2:-$PICOCLAW_SERVICE_NAME}"
        assert_is_identifier "$PICOCLAW_SERVICE_NAME" "Service name must be a valid service identifier" || exit 1
    ;;&

    logs*)
        LOG_N="${3:-50}"
        assert_is_number "$LOG_N" "Log lines parameter must be numeric" || exit 1
    ;;&

    logs/linux*)
        journalctl --user-unit ${PICOCLAW_SERVICE_NAME} --no-pager -n ${LOG_N}

    ;;

    logs-errors/linux*)
        journalctl --user-unit ${PICOCLAW_SERVICE_NAME} --no-pager -p 3 -n ${LOG_N}
    ;;

    logs/darwin*)
        tail -n ${LOG_N} ~/Library/Logs/${PICOCLAW_SERVICE_NAME}.log
    ;;

    logs-errors/darwin*)
        tail -n ${LOG_N} ~/Library/Logs/${PICOCLAW_SERVICE_NAME}.err.og
    ;;

    service-status/linux*)
        systemctl --user status ${PICOCLAW_SERVICE_NAME}
    ;;

    config-status/*)
        picoclaw status
    ;;

    config-safe/*) 
       jq 'walk(if type == "object" then with_entries(if .key | ascii_downcase |
           (contains("key") or contains("token") or contains("secret")) 
           then .value = "REDACTED" else . end) else . end)' "${PICOCLAW_CONFIG}"
    ;;
    
    *)
        echo "Usage: $0 logs|logs-errors|service-status|config-status|config-safe [service_name] [n_lines]"
        exit 1
    ;;
esac

#!/bin/bash

# config: A tool for presenting config in insecure channels.

# Input validation functions
assert_is_identifier() {
    if ! echo "$1" | grep -qE '^[a-zA-Z_][a-zA-Z0-9_-]*$'; then
        echo "Error: $2" >&2
        return 1
    fi
}

assert_is_file_path() {
    if [ -z "$1" ]; then
        echo "Error: $2" >&2
        return 1
    fi
    local dir
    dir=$(dirname "$1")
    if [ ! -d "$dir" ] && [ "$dir" != "." ]; then
        echo "Error: Directory does not exist: $dir" >&2
        return 1
    fi
}

assert_is_safe() {
    # Only check the command argument, not the entire command line
    local cmd="$1"
    if echo "$cmd" | grep -qE '[\$\`\\]'; then
        echo "Error: $2 - Command contains dangerous characters" >&2
        return 1
    fi
    return 0
}

COMMAND="${1:-help}"

PICOCLAW_CONFIG="${2:-}"
PICOCLAW_CONFIG="${PICOCLAW_CONFIG:-$HOME/.picoclaw/config.json}"

assert_is_safe "$COMMAND" "Command must be a valid identifier" || exit 1
assert_is_file_path "$PICOCLAW_CONFIG" "Config file path must be valid" || exit 1

# Setup paths
BASE=$(basename "$PICOCLAW_CONFIG" .json)
DIR=$(dirname "$PICOCLAW_CONFIG")
STAGING_FILE="$DIR/$BASE.new.json"
SECRETS_FILE="$DIR/.$BASE.secrets.json"
BACKUP_DIR="$DIR/.config_backups"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

# Temp file for staging (cleaned up on script exit)
CONFIG_TMP=""
SECRETS_TMP=""
FINAL_FILE=""
cleanup() { rm -f "$CONFIG_TMP" "$FINAL_FILE" "$SECRETS_TMP"; }
trap cleanup EXIT

SECRET_PATTERN="key|pass|secret|token|auth|credential|sid|sk-"
EXCLUDE_PATTERN="max_tokens|auth_method|enable_auth|auth_type"

# Shared function: extract secrets if not already done
redact_secrets() {
    
    # Check if original has any secrets
    paths=$(jq -c "paths(scalars) | select(.[-1] | tostring | ascii_downcase | (test(\"$SECRET_PATTERN\"; \"i\") and (test(\"$EXCLUDE_PATTERN\"; \"i\") | not)))" "$PICOCLAW_CONFIG" 2>/dev/null) || return 0
    
    # Create staging file in temp (global var for trap cleanup)
    CONFIG_TMP=$(mktemp)
    cp "$PICOCLAW_CONFIG" "$CONFIG_TMP"
    
    # Extract secrets directly to final location
    SECRETS_TMP=$(mktemp)
    echo "{}" > "$SECRETS_TMP"
    chmod 600 "$SECRETS_TMP"
    
    while read -r p; do
        [ -z "$p" ] && continue
        ID="SECRET:$(echo "$p" | jq -r '. | map(tostring) | map(if . | test("^[0-9]+$") then "[" + . + "]" else "." + . end) | join("") | ltrimstr(".")')"
        VAL=$(jq -r "getpath($p)" "$PICOCLAW_CONFIG")
        
        # Add to secrets file
        { rm -f "$SECRETS_TMP" ; jq --arg id "$ID" --arg val "$VAL" '. + {($id): $val}' > "$SECRETS_TMP"; } < "$SECRETS_TMP" 
        
        # Replace with placeholder in staging
        { rm -f "$CONFIG_TMP" ; jq --argjson p "$p" --arg id "$ID" 'setpath($p; $id)' > "${CONFIG_TMP}"; } < "$CONFIG_TMP" 
        
    done <<< "$paths"
}

        
# Output a summary with only configured models, agents, enabled tools, devices, and heartbeat

summarize()
{
    jq '
    def referenced_models:
        [.. | objects | if has("model") then .model elif has("model_name") then .model_name else null end // empty] | unique;

    def used_model_ids:
        referenced_models;

    def is_configured:
        . != null and . != {} and . != [];

    def filter_tools:
        (walk(
            if type == "object" then
                if .enabled == false then
                    null
                else
                    . as $obj | reduce keys_unsorted[] as $k ({}; . + {($k): ($obj[$k] | filter_tools)})
                end
            else
                .
            end
        ) | if type == "object" then with_entries(select(.value != null)) else . end);

    . * {
        model_list: (.model_list // []) | map(select(.model_name as $id | used_model_ids | contains([$id]))),
        agents: (.agents // {}) | map_values(select(is_configured))
    } | .channels = ((.channels // {}) | to_entries | map(select(.value.enabled == true)) | from_entries)
    | .tools = ((.tools // {}) | filter_tools)
    | .providers = ((.providers // {} | to_entries | map(select(.value.api_key | gsub("^\\s+"; "") | gsub("\\s+$"; "") != "")) | from_entries))
    | .devices = (if (.devices.enabled // false) then .devices else null end)
    | .heartbeat = (if (.heartbeat.enabled // false) then .heartbeat else null end)
    ' "${1}" | jq 'with_entries(select(.value != null))'
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    case "$COMMAND" in

        redacted)  
        
            redact_secrets
            cat "$CONFIG_TMP"
        ;;

        summary)
            # Output a summary with only configured models, agents, enabled tools, devices, and heartbeat
            
            redact_secrets
            summarize "$CONFIG_TMP" 
            
            exit 0     
        ;;

        help)
            echo "Usage: $0 <command> [config_file]"
            echo "Commands:"
            echo "  redacted   - Show config (default: PICOCLAW_CONFIG)"
            echo "  summary    - Show config with unused models/chatconfigs filtered"
            exit 0
        ;;

    esac
fi

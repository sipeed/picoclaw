#!/bin/bash

# config-patch: A tool for iterative JSON patching with high-security secret handling.

# Determine script directory for relocatable operation
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SKILL_ROOT="$(dirname "$SCRIPT_DIR")"

# Source shared config and functions
source "$SCRIPT_DIR/config.sh"

COMMAND="${1:-help}"

case "$COMMAND" in

    start|sort)
        
        [ -f "$SECRETS_FILE" ] && { echo "Error: Session active. Commit or reset first."; exit 1; }
        [ ! -f "$PICOCLAW_CONFIG" ] && { echo "Error: $PICOCLAW_CONFIG not found."; exit 1; }
        
        redact_secrets || exit 1

        # On success - Move results final locations (trap will clean if script crashes before this)
        # Clear <file>_TMP so trap doesn't try to delete final files
        
        mv "$CONFIG_TMP" "$STAGING_FILE"
        CONFIG_TMP=""
        mv "$SECRETS_TMP" "$SECRETS_FILE"
        SECRETS_TMP=""
        
        echo "Start: Redacted staging file created. Secrets mapped to $SECRETS_FILE"
        
    ;;&

    start)     
        exit 0
    ;;

    sort)
      
        { rm "$STAGING_FILE" ; jq -S . > "$STAGING_FILE"; } < "$STAGING_FILE"
           
        echo "Sorted: keys in $STAGING_FILE"
        exit 0
    ;;

    commit)
        [ ! -f "$STAGING_FILE" ]     && { echo "Error: No staged changes."; exit 1; }
        [ ! -f "$SECRETS_FILE" ] && { echo "Error: No secrets file found."; exit 1; }
              
        # 3. Restoration & Validation
        FINAL_FILE=$(mktemp)
        
        # Restore secrets: replace path placeholders with stored values
        # Keys in secrets file are the paths (e.g., "SECRET:agents.defaults.model")
        jq --slurpfile secrets "$SECRETS_FILE" 'walk(if type == "string" and $secrets[0][.] != null then $secrets[0][.] else . end)' "$STAGING_FILE" > "$FINAL_FILE"
        
        # Ensure no placeholders survived (means they weren't in the map)
        # Check if any string key in secrets map still exists in output
        while IFS= read -r key; do
            if grep -q "\"$key\"" "$FINAL_FILE"; then
                echo "Error: Placeholder '$key' was not restored. Check for typos in secrets map."
                exit 1
            fi
        done < <(jq -r 'keys[]' "$SECRETS_FILE")
    
        # 4. JSON Integrity Check
        jq . "$FINAL_FILE" > /dev/null 2>&1 || { echo "Error: Invalid JSON output. Aborting."; exit 1; }

        # 5. Backup & Swap (unredacted config)
        mkdir -p "$BACKUP_DIR"
        cp "$PICOCLAW_CONFIG" "$BACKUP_DIR/${BASE}_${TIMESTAMP}.json"
        
        mv "$FINAL_FILE" "$PICOCLAW_CONFIG"
        rm -f "$STAGING_FILE" "$SECRETS_FILE"
        echo "Committed successfully. Backup: $BACKUP_DIR/${BASE}_${TIMESTAMP}.json"
        exit 0
    ;;

    status)
        echo "Target: $PICOCLAW_CONFIG"
        [ -f "$STAGING_FILE" ] && echo "Staging: Active ($STAGING_FILE)" || echo "Staging: None"
        [ -f "$SECRETS_FILE" ] && echo "Secrets: $(jq 'keys | length' "$SECRETS_FILE") tracked" || echo "Secrets: None"
    ;;

    summary)
        # Output a summary of the staged config with only configured models, agents, enabled tools, devices, and heartbeat
        
        summarize "$STAGING_FILE" 
        
        exit 0     
    ;;

    show|redacted|config)
        # Output the full staged config
   
        cat "$STAGING_FILE"
    ;;

    diff)
        [ ! -f "$STAGING_FILE" ] && { echo "Error: No staged changes to diff."; exit 1; }

        redact_secrets
        diff -u "$CONFIG_TMP" "$STAGING_FILE"
        exit 0
    ;;

    rollback)
        # Restore the most recent backup
        LATEST=$(ls -t "$BACKUP_DIR"/${BASE}_*.json 2>/dev/null | head -n 1)
        if [ -z "$LATEST" ]; then
            echo "Error: No backups found for $PICOCLAW_CONFIG"
            exit 1
        fi
        
        LATEST_BASE=$(basename "$LATEST" .json)
        
        cp "$LATEST" "$PICOCLAW_CONFIG"
        
        echo "Rollback: Restored $PICOCLAW_CONFIG from $LATEST"
        exit 0
        ;;

    reset)
        rm -f "$STAGING_FILE" "$SECRETS_FILE"
        echo "Session cleared."
        exit 0
        ;;

    help)
        echo "Usage: $0 <command> [config_file]"
        echo "Commands:"
        echo "  start      - Create staging file with redacted secrets"
        echo "  sort       - Sort JSON keys alphabetically"
        echo "  diff       - Show staged changes"
        echo "  commit     - Apply staged changes to config"
        echo "  reset      - Clear staging (discard changes)"
        echo "  rollback   - Restore from last backup"
        echo "  status     - Show current state"
        echo "  summary    - Show summary of the staged config with unused models/chatconfigs filtered"
        echo "  config     - Show the staged config"
        echo "  <jq expr>  - Apply inline jq patch (e.g. '.agents.model=\"gpt-4\"')"
        exit 0
        ;;

    *)
        # Generic Patching
        redact_secrets
        if [ -f "$STAGING_FILE" ]; then
            SOURCE="$STAGING_FILE"
        else
            SOURCE="$PICOCLAW_CONFIG"
        fi
        TMP=$(mktemp)
        if jq "$COMMAND" "$SOURCE" > "$TMP"; then
            mv "$TMP" "$STAGING_FILE"
            echo "Applied patch to $STAGING_FILE"
        else
            rm -f "$TMP"; exit 1
        fi
        ;;
esac

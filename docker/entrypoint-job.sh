#!/bin/bash
set -e

WORKSPACE="/home/picoclaw/.picoclaw/workspace"

# ── Environment selection ─────────────────────────────────────────────────────
# ENVIRONMENT controls which dashboard URL is targeted.
# Pass --update-env-vars="ENVIRONMENT=UAT" or --update-env-vars="ENVIRONMENT=PREVIEW-PROD"
# BASE_URL can also be set directly to override this mapping.
if [ -z "$BASE_URL" ]; then
  case "${ENVIRONMENT:-UAT}" in
    UAT)
      BASE_URL="https://dashboard.int3nt.info"
      ;;
    PREVIEW-PROD)
      BASE_URL="https://dashboard-preview.intentai.com"
      ;;
    *)
      echo "ERROR: Unknown ENVIRONMENT=${ENVIRONMENT} (valid: UAT, PREVIEW-PROD)"
      exit 1
      ;;
  esac
fi
export BASE_URL
echo "=== Target environment: ${ENVIRONMENT:-UAT} ($BASE_URL) ==="

# Config secret is mounted at .picoclaw-config (not .picoclaw) so it doesn't
# create a read-only tmpfs that would block the GCS workspace volume mount.
# Copy config.json to the location picoclaw expects before anything else runs.
if [ -f "/home/picoclaw/.picoclaw-config/config.json" ]; then
    cp "/home/picoclaw/.picoclaw-config/config.json" "/home/picoclaw/.picoclaw/config.json"
fi

echo "=== Workspace contents ==="
find "$WORKSPACE" -type d -name node_modules -prune -o -type f -print | sort
echo "=========================="

# Wait for LiteLLM sidecar to be ready (Cloud Run sidecars have no startup ordering)
if [ -n "$LITELLM_BASE_URL" ]; then
  echo "Waiting for LiteLLM at $LITELLM_BASE_URL ..."
  for i in $(seq 1 30); do
    if curl -sf "$LITELLM_BASE_URL/health/liveliness" > /dev/null 2>&1; then
      echo "LiteLLM is ready."
      break
    fi
    if [ "$i" -eq 30 ]; then
      echo "ERROR: LiteLLM did not become ready in time."
      exit 1
    fi
    sleep 2
  done
fi

# Wait for the target server to be ready
if [ -n "$BASE_URL" ]; then
  echo "Waiting for server at $BASE_URL ..."
  for i in $(seq 1 30); do
    if curl -sf "$BASE_URL" > /dev/null 2>&1; then
      echo "Server is ready."
      break
    fi
    if [ "$i" -eq 30 ]; then
      echo "ERROR: Server did not become ready in time."
      exit 1
    fi
    sleep 2
  done
fi

# ── Playwright staging area (run / run-all only) ─────────────────────────────
# Copy tests + config to /tmp so Node.js module resolution never walks into the
# GCS-mounted workspace and finds a stale node_modules there. Running from /tmp
# means @playwright/test always resolves from /tmp/pw/node_modules only.
# Skipped for generate/autofix — those only need the workspace (GCS mount).

PW="/tmp/pw"

setup_playwright() {
  mkdir -p "$PW"
  echo "Staging tests from workspace to /tmp/pw..."
  cp -r "$WORKSPACE/tests" "$PW/"
  cp "$WORKSPACE/playwright.config.ts" "$PW/" 2>/dev/null || true
  cp "$WORKSPACE/package.json" "$PW/"
  cp "$WORKSPACE/package-lock.json" "$PW/" 2>/dev/null || true
  cd "$PW"
  echo "Installing dependencies..."
  npm install --prefer-offline 2>&1
}

# ── Helpers ───────────────────────────────────────────────────────────────────

run_group() {
  local group_name="$1"
  shift
  echo ""
  echo "=================================================="
  echo "GROUP: $group_name"
  echo "=================================================="
  "$PW/node_modules/.bin/playwright" test "$@" --reporter=line
}

# ── Dispatch ──────────────────────────────────────────────────────────────────

JOB_TYPE="${JOB_TYPE:-run-all}"

case "$JOB_TYPE" in
  run-all)
    setup_playwright
    # Tests run in dependency order:
    #   auth → knowledge-base (create → operate → delete)
    #       → flow-designer → flow-tester
    #       → profile → organization → settings → logs

    run_group "Auth" \
      tests/auth/login.spec.ts \
      tests/auth/logout.spec.ts \
      tests/auth/forgot-password.spec.ts

    run_group "Knowledge Base - Create" \
      tests/knowledge-base/create-kb-bucket-gcs.spec.ts \
      tests/knowledge-base/create-kb-bucket-website-crawler.spec.ts

    run_group "Knowledge Base - Schedule & Edit" \
      tests/knowledge-base/edit-kb-schedule.spec.ts \
      tests/knowledge-base/schedule-kb-incremental-sync-simple.spec.ts \
      tests/knowledge-base/schedule-kb-incremental-sync-advanced.spec.ts \
      tests/knowledge-base/schedule-kb-full-sync-advanced.spec.ts \
      tests/knowledge-base/schedule-kb-full-sync-simple.spec.ts

    run_group "Knowledge Base - Delete" \
      tests/knowledge-base/delete-kb.spec.ts

    # flow-designer must run before flow-tester (creates the flows)
    run_group "Flow Designer" \
      tests/flow-designer/create-new-flow-user-utterance-node.spec.ts \
      tests/flow-designer/create-new-flow-custom-node.spec.ts \
      tests/flow-designer/create-new-flow-model-node-parser.spec.ts \
      tests/flow-designer/create-new-flow-model-node-without-parser.spec.ts \
      tests/flow-designer/create-new-flow-knowledge-base-node.spec.ts \
      tests/flow-designer/create-new-flow-knowledge-base-web-crawler.spec.ts

    # flow-tester depends on flows created by flow-designer above
    run_group "Flow Tester" \
      tests/flow-tester/test-user-utterance-flow.spec.ts \
      tests/flow-tester/test-custom-node-flow.spec.ts \
      tests/flow-tester/test-model-node-parser-flow.spec.ts \
      tests/flow-tester/test-model-node-without-parser-flow.spec.ts \
      tests/flow-tester/test-kb-flow.spec.ts \
      tests/flow-tester/test-kb-web-crawler-flow.spec.ts

    run_group "Profile" \
      tests/profile/update-profile-name.spec.ts \
      tests/profile/change-password.spec.ts \
      tests/profile/change-email.spec.ts

    run_group "Organization" \
      tests/organization/invite-member-access.spec.ts \
      tests/organization/invite-existing-user.spec.ts \
      tests/organization/change-role-admin-to-developer.spec.ts \
      tests/organization/change-role-developer-to-agent.spec.ts \
      tests/organization/change-role-agent-to-admin.spec.ts \
      tests/organization/deactivate-member-access-control.spec.ts \
      tests/organization/activate-member-access-restored.spec.ts \
      tests/organization/switch-organization.spec.ts \
      tests/organization/upload-organization-logo.spec.ts \
      tests/organization/upload-bot-icon.spec.ts \
      tests/organization/admin-role-sidebar-permissions.spec.ts \
      tests/organization/developer-role-sidebar-permissions.spec.ts \
      tests/organization/agent-role-sidebar-permissions.spec.ts

    run_group "Settings" \
      tests/settings/view-api-keys-settings.spec.ts \
      tests/settings/create-api-key-internal.spec.ts \
      tests/settings/create-api-key-external.spec.ts \
      tests/settings/edit-api-key-description.spec.ts \
      tests/settings/reactivate-api-key.spec.ts \
      tests/settings/revoke-api-key.spec.ts

    run_group "Logs" \
      tests/logs/download-conversation-logs.spec.ts
    ;;

  run)
    if [ -z "$JOB_SPEC" ]; then
      echo "ERROR: JOB_SPEC is required for JOB_TYPE=run"
      exit 1
    fi
    setup_playwright
    echo "Running test: $JOB_SPEC"
    "$PW/node_modules/.bin/playwright" test "$JOB_SPEC" --reporter=line
    ;;

  autofix)
    if [ -z "$JOB_SPEC" ]; then
      echo "ERROR: JOB_SPEC is required for JOB_TYPE=autofix"
      exit 1
    fi
    AREA=$(echo "$JOB_SPEC" | cut -d'/' -f2)
    TEMPLATE="$WORKSPACE/templates/autofix/$AREA.txt"
    if [ ! -f "$TEMPLATE" ]; then
      echo "ERROR: No autofix template for area: $AREA"
      exit 1
    fi
    echo "Autofixing test: $JOB_SPEC ..."
    PROMPT=$(node -e "
const fs = require('fs');
let t = fs.readFileSync('$TEMPLATE', 'utf8');
t = t.replace(/\{\{SPEC_FILE\}\}/g, process.env.JOB_SPEC || '');
t = t.replace(/\{\{BASE_URL\}\}/g, process.env.BASE_URL || 'https://dashboard.int3nt.info');
process.stdout.write(t);
")
    picoclaw agent -m "$PROMPT"
    ;;

  generate)
    if [ -z "$JOB_AREA" ] || [ -z "$JOB_TEST_FILE" ] || [ -z "$JOB_STEPS" ] || [ -z "$JOB_EXPECTED_RESULT" ]; then
      echo "ERROR: JOB_AREA, JOB_TEST_FILE, JOB_STEPS, and JOB_EXPECTED_RESULT are required for JOB_TYPE=generate"
      exit 1
    fi
    TEMPLATE="$WORKSPACE/templates/$JOB_AREA.txt"
    if [ ! -f "$TEMPLATE" ]; then
      echo "ERROR: No template found for area: $JOB_AREA"
      exit 1
    fi
    echo "Generating test from template..."
    PROMPT=$(node -e "
const fs = require('fs');
let t = fs.readFileSync('$TEMPLATE', 'utf8');
t = t.replace(/\{\{TEST_FILE\}\}/g, process.env.JOB_TEST_FILE || '');
t = t.replace(/\{\{STEPS\}\}/g, process.env.JOB_STEPS || '');
t = t.replace(/\{\{EXPECTED_RESULT\}\}/g, process.env.JOB_EXPECTED_RESULT || '');
t = t.replace(/\{\{BASE_URL\}\}/g, process.env.BASE_URL || 'https://dashboard.int3nt.info');
process.stdout.write(t);
")
    picoclaw agent -m "$PROMPT"
    ;;

  prompt)
    if [ -z "$JOB_PROMPT" ]; then
      echo "ERROR: JOB_PROMPT is required for JOB_TYPE=prompt"
      exit 1
    fi
    echo "Running picoclaw with prompt..."
    picoclaw agent -m "$JOB_PROMPT"
    ;;

  *)
    echo "ERROR: Unknown JOB_TYPE=$JOB_TYPE (valid: run-all, run, autofix, generate, prompt)"
    exit 1
    ;;
esac

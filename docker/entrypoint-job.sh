#!/bin/bash
set -e

WORKSPACE="/home/picoclaw/.picoclaw/workspace"

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

cd "$WORKSPACE"

# Always purge node_modules from the GCS bucket via gsutil (bypasses FUSE, which
# does not reliably expose bucket prefixes as directories for [ -d ] checks).
# node_modules in the bucket causes two @playwright/test instances → "test() called here".
echo "Purging node_modules from GCS bucket..."
gsutil -m rm -r "gs://${RESULTS_BUCKET}/node_modules/" 2>/dev/null || true

# Install node_modules to /tmp — GCS FUSE does not support chmod,
# which npm requires when linking bin scripts in node_modules.
if [ ! -d "/tmp/node_modules" ]; then
  echo "Installing dependencies to /tmp..."
  cp "$WORKSPACE/package.json" /tmp/
  cp "$WORKSPACE/package-lock.json" /tmp/ 2>/dev/null || true
  cd /tmp && npm install --prefer-offline 2>&1
  cd "$WORKSPACE"
fi

export PATH="/tmp/node_modules/.bin:$PATH"
export NODE_PATH="/tmp/node_modules"

# Run a group of tests. Exits the script if any test fails.
# Usage: run_group "Group Name" <spec1> <spec2> ...
run_group() {
  local group_name="$1"
  shift
  echo ""
  echo "=================================================="
  echo "GROUP: $group_name"
  echo "=================================================="
  /tmp/node_modules/.bin/playwright test "$@" --reporter=line
}

JOB_TYPE="${JOB_TYPE:-run-all}"

case "$JOB_TYPE" in
  run-all)
    # Tests are run in dependency order:
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
    echo "Running test: $JOB_SPEC"
    /tmp/node_modules/.bin/playwright test "$JOB_SPEC" --reporter=line
    ;;

  generate)
    if [ -z "$JOB_PROMPT" ]; then
      echo "ERROR: JOB_PROMPT is required for JOB_TYPE=generate"
      exit 1
    fi
    echo "Generating test from prompt..."
    picoclaw agent -m "$JOB_PROMPT"
    ;;

  *)
    echo "ERROR: Unknown JOB_TYPE=$JOB_TYPE (valid: run-all, run, generate)"
    exit 1
    ;;
esac

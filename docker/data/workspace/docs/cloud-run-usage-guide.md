# Cloud Run E2E Testing - Usage Guide

## Getting Started

### Access Cloud Shell

Go to: **https://console.cloud.google.com/run/jobs/details/europe-west4/picoclaw-e2e/executions?project=digital-equator-311106&cloudshell=true**

This opens the Cloud Run job page with Cloud Shell terminal at the bottom.

All commands in this guide should be run in that Cloud Shell terminal.

### Basic Command Structure

All commands follow this pattern:

```bash
gcloud run jobs execute picoclaw-e2e \
  --region=europe-west4 \
  --container=picoclaw \
  --update-env-vars="JOB_TYPE=<what-to-do>" \
  --update-env-vars="<additional-options>"
```

**Always include --container=picoclaw** - without it, the command will fail.

---

## How to Run Tests

### Option 1: Run All Tests (Full Regression)

Use this for:
- Pre-deployment validation
- Weekly regression testing
- After major changes

**Command:**

```bash
gcloud run jobs execute picoclaw-e2e \
  --region=europe-west4 \
  --container=picoclaw \
  --update-env-vars="JOB_TYPE=run-all"
```

Runs 46 tests in this order:
Auth → Knowledge Base → Flow Designer → Flow Tester → Profile → Organization → Settings → Logs

---

### Option 2: Run Tests for One Feature

Use this for:
- Testing after feature-specific changes
- Faster feedback when you only changed one area
- Debugging issues in a specific feature

**Command:**

```bash
gcloud run jobs execute picoclaw-e2e \
  --region=europe-west4 \
  --container=picoclaw \
  --update-env-vars="JOB_TYPE=run-feature" \
  --update-env-vars="JOB_FEATURE=<feature-name>"
```

**Available Features:**
- **auth** - Login, logout, forgot password (3 tests)
- **knowledge-base** - KB creation, scheduling, editing, deletion (6 tests)
- **flow-designer** - Creating flows with different node types (6 tests)
- **flow-tester** - Testing created flows (6 tests)
- **profile** - Profile updates, password/email changes (3 tests)
- **organization** - Org switching, member management, roles (13 tests)
- **settings** - API key management (6 tests)
- **logs** - Log downloads (1 test)

**Example - Run all Knowledge Base tests:**

```bash
gcloud run jobs execute picoclaw-e2e \
  --region=europe-west4 \
  --container=picoclaw \
  --update-env-vars="JOB_TYPE=run-feature" \
  --update-env-vars="JOB_FEATURE=knowledge-base"
```

---

### Option 3: Run a Single Test

Use this for:
- Quick verification after fixing a specific bug
- Testing one specific scenario
- Debugging a single failing test

**Command:**

```bash
gcloud run jobs execute picoclaw-e2e \
  --region=europe-west4 \
  --container=picoclaw \
  --update-env-vars="JOB_TYPE=run" \
  --update-env-vars="JOB_SPEC=<test-file-path>"
```

**Example - Run just the login test:**

```bash
gcloud run jobs execute picoclaw-e2e \
  --region=europe-west4 \
  --container=picoclaw \
  --update-env-vars="JOB_TYPE=run" \
  --update-env-vars="JOB_SPEC=tests/auth/login.spec.ts"
```

See Test Coverage section for the full list of test file paths.

---

## How to Generate New Test

Use this when you need to create a new test from scratch.

**Command:**

```bash
gcloud run jobs execute picoclaw-e2e \
  --region=europe-west4 \
  --container=picoclaw \
  --update-env-vars="JOB_TYPE=generate" \
  --update-env-vars="JOB_AREA=<feature-area>" \
  --update-env-vars="JOB_TEST_FILE=<test-name>" \
  --update-env-vars="JOB_STEPS=1. Step one
2. Step two
3. Step three" \
  --update-env-vars="JOB_EXPECTED_RESULT=What should happen"
```

**Available Areas:**
- **auth** - Authentication tests
- **knowledge-base** - Knowledge Base tests
- **flow-designer** - Flow Designer tests
- **flow-tester** - Flow Tester tests
- **profile** - Profile tests
- **organization** - Organization tests
- **settings** - Settings tests
- **logs** - Logs tests

**Example - Generate a new flow test:**

```bash
gcloud run jobs execute picoclaw-e2e \
  --region=europe-west4 \
  --container=picoclaw \
  --flags-file=<(cat <<'EOF'
--update-env-vars:
  JOB_TYPE: generate
  JOB_AREA: flow-designer
  JOB_TEST_FILE: create-new-flow-custom-node
  JOB_STEPS: |
    1. Perform case "Login"
    2. On Select Organization page, select organization "Testing2026!"
    3. User redirected to: https://dashboard.int3nt.info
    4. Click Flow Designer on the left sidebar
    5. On All Flows page, click Add New
    6. Flow canvas page opens displaying default nodes: START, END
    7. Click Add Nodes Button
    8. From Add Nodes menu, select Custom Node
    9. Verify the Custom Node is added to the canvas
  JOB_EXPECTED_RESULT: |
    1. Custom Node is successfully added to the flow canvas
    2. Default nodes (START, END) remain visible
EOF
)
```

**What happens:**
1. System reads the template for the specified area
2. Creates a new test file with your steps
3. Runs the test
4. If it fails, automatically fixes it
5. Repeats until the test passes

The new test will be created at: **tests/<area>/<test-name>.spec.ts**

---

## Testing Different Environments

### UAT (Default)

This is the testing environment. All commands run against UAT by default.

**URL:** https://dashboard.int3nt.info

No extra options needed - just run the command as shown above.

---

### Preview PROD

This is the pre-production environment for final testing before going live.

**URL:** https://dashboard-preview.intentai.com

**Add this to any command:**

```bash
--update-env-vars="ENVIRONMENT=PREVIEW-PROD"
```

**Example - Run all tests on Preview PROD:**

```bash
gcloud run jobs execute picoclaw-e2e \
  --region=europe-west4 \
  --container=picoclaw \
  --update-env-vars="JOB_TYPE=run-all" \
  --update-env-vars="ENVIRONMENT=PREVIEW-PROD"
```

---

## Test Coverage

All available tests organized by feature area. Use these paths with JOB_TYPE=run to run individual tests.

### Auth Tests (3 tests)

| Test | What It Tests |
|------|---------------|
| tests/auth/login.spec.ts | User can login, select organization, reach dashboard |
| tests/auth/logout.spec.ts | User can logout and return to login page |
| tests/auth/forgot-password.spec.ts | User can reset password via email |

### Knowledge Base Tests (6 tests)

| Test | What It Tests |
|------|---------------|
| tests/knowledge-base/create-kb-bucket-gcs.spec.ts | Create KB bucket with Google Cloud Storage source |
| tests/knowledge-base/create-kb-bucket-website-crawler.spec.ts | Create KB bucket with Website Crawler source |
| tests/knowledge-base/schedule-kb-incremental-sync-advanced.spec.ts | Create incremental sync schedule (advanced mode with cron) |
| tests/knowledge-base/schedule-kb-full-sync-simple.spec.ts | Create full sync schedule (simple mode with frequency picker) |
| tests/knowledge-base/edit-kb-schedule.spec.ts | Edit existing KB schedule |
| tests/knowledge-base/delete-kb.spec.ts | Delete a KB bucket |

### Flow Designer Tests (6 tests)

| Test | What It Tests |
|------|---------------|
| tests/flow-designer/create-new-flow-user-utterance-node.spec.ts | Create flow with User Utterance node (captures user input) |
| tests/flow-designer/create-new-flow-custom-node.spec.ts | Create flow with Custom Node (runs Python code) |
| tests/flow-designer/create-new-flow-model-node-parser.spec.ts | Create flow with Model Node that has output parser |
| tests/flow-designer/create-new-flow-model-node-without-parser.spec.ts | Create flow with Model Node (no parser) |
| tests/flow-designer/create-new-flow-knowledge-base-node.spec.ts | Create flow with Knowledge Base node |
| tests/flow-designer/create-new-flow-knowledge-base-web-crawler.spec.ts | Create flow with KB node using web crawler bucket |

### Flow Tester Tests (6 tests)

**⚠️ Important:** These tests depend on flows created by Flow Designer tests. Run Flow Designer first.

| Test | What It Tests |
|------|---------------|
| tests/flow-tester/test-user-utterance-flow.spec.ts | Test User Utterance flow works correctly |
| tests/flow-tester/test-custom-node-flow.spec.ts | Test Custom Node flow executes Python code |
| tests/flow-tester/test-model-node-parser-flow.spec.ts | Test Model Node with parser returns structured data |
| tests/flow-tester/test-model-node-without-parser-flow.spec.ts | Test Model Node without parser returns raw response |
| tests/flow-tester/test-kb-flow.spec.ts | Test Knowledge Base flow retrieves correct information |
| tests/flow-tester/test-kb-web-crawler-flow.spec.ts | Test KB flow with web crawler bucket |

### Profile Tests (3 tests)

| Test | What It Tests |
|------|---------------|
| tests/profile/update-profile-name.spec.ts | User can update their display name |
| tests/profile/change-password.spec.ts | User can change password |
| tests/profile/change-email.spec.ts | User can change email address |

### Organization Tests (13 tests)

| Test | What It Tests |
|------|---------------|
| tests/organization/switch-organization.spec.ts | User can switch between organizations |
| tests/organization/agent-role-sidebar-permissions.spec.ts | Agent role has correct sidebar permissions |
| tests/organization/change-role-agent-to-admin.spec.ts | Admin can promote Agent to Admin |
| tests/organization/admin-role-sidebar-permissions.spec.ts | Admin role has full sidebar access |
| tests/organization/change-role-admin-to-developer.spec.ts | Admin can change Admin to Developer |
| tests/organization/developer-role-sidebar-permissions.spec.ts | Developer role has correct permissions |
| tests/organization/change-role-developer-to-agent.spec.ts | Admin can downgrade Developer to Agent |
| tests/organization/deactivate-member-access-control.spec.ts | Admin can deactivate member |
| tests/organization/activate-member-access-restored.spec.ts | Admin can reactivate member |
| tests/organization/upload-bot-icon.spec.ts | User can upload custom bot icon |
| tests/organization/upload-organization-logo.spec.ts | User can upload organization logo |
| tests/organization/invite-member-access.spec.ts | Admin can invite new member |
| tests/organization/invite-existing-user.spec.ts | Admin can invite existing user |

### Settings Tests (6 tests)

**⚠️ Important:** Run all settings tests together - they depend on each other.

| Test | What It Tests |
|------|---------------|
| tests/settings/view-api-keys-settings.spec.ts | User can view API keys page |
| tests/settings/create-api-key-internal.spec.ts | User can create Internal API key |
| tests/settings/create-api-key-external.spec.ts | User can create External API key |
| tests/settings/edit-api-key-description.spec.ts | User can edit API key description |
| tests/settings/revoke-api-key.spec.ts | User can revoke API key |
| tests/settings/reactivate-api-key.spec.ts | User can reactivate revoked key |

### Logs Tests (1 test)

| Test | What It Tests |
|------|---------------|
| tests/logs/download-conversation-logs.spec.ts | User can download conversation logs as CSV |

---

**Total: 46 tests**

---

## Quick Reference

### Most Common Commands

**1. Run all tests (full regression):**

```bash
gcloud run jobs execute picoclaw-e2e \
  --region=europe-west4 \
  --container=picoclaw \
  --update-env-vars="JOB_TYPE=run-all"
```

**2. Run tests for one feature:**

```bash
gcloud run jobs execute picoclaw-e2e \
  --region=europe-west4 \
  --container=picoclaw \
  --update-env-vars="JOB_TYPE=run-feature" \
  --update-env-vars="JOB_FEATURE=knowledge-base"
```

**3. Run one test:**

```bash
gcloud run jobs execute picoclaw-e2e \
  --region=europe-west4 \
  --container=picoclaw \
  --update-env-vars="JOB_TYPE=run" \
  --update-env-vars="JOB_SPEC=tests/auth/login.spec.ts"
```

**4. Fix a failing test automatically:**

```bash
gcloud run jobs execute picoclaw-e2e \
  --region=europe-west4 \
  --container=picoclaw \
  --update-env-vars="JOB_TYPE=autofix" \
  --update-env-vars="JOB_SPEC=tests/auth/login.spec.ts"
```

**5. Generate a new test:**

```bash
gcloud run jobs execute picoclaw-e2e \
  --region=europe-west4 \
  --container=picoclaw \
  --update-env-vars="JOB_TYPE=generate" \
  --update-env-vars="JOB_AREA=auth" \
  --update-env-vars="JOB_TEST_FILE=new-test" \
  --update-env-vars="JOB_STEPS=1. Step one" \
  --update-env-vars="JOB_EXPECTED_RESULT=Expected result"
```

**6. Run tests on Preview PROD:**

```bash
gcloud run jobs execute picoclaw-e2e \
  --region=europe-west4 \
  --container=picoclaw \
  --update-env-vars="JOB_TYPE=run-all" \
  --update-env-vars="ENVIRONMENT=PREVIEW-PROD"
```

**7. View recent test runs:**

Go to the History tab in the Cloud Run console, or list executions:

```bash
gcloud run jobs executions list \
  --job picoclaw-e2e \
  --region europe-west4 \
  --limit 10
```

---

### Available Features

Use these with JOB_TYPE=run-feature:

- **auth** (3 tests)
- **knowledge-base** (6 tests)
- **flow-designer** (6 tests)
- **flow-tester** (6 tests) - **Must run flow-designer first**
- **profile** (3 tests)
- **organization** (13 tests)
- **settings** (6 tests)
- **logs** (1 test)

---

### Test Environment URLs

- **UAT (default):** https://dashboard.int3nt.info
- **Preview PROD:** https://dashboard-preview.intentai.com

---

### When to Run Tests

| Scenario | Recommended Command |
|----------|---------------------|
| Before deploying to production | JOB_TYPE=run-all on Preview PROD |
| After UI changes | JOB_TYPE=run-feature for affected area |
| After bug fix | JOB_TYPE=run for specific test |
| Test broke after deployment | JOB_TYPE=autofix for failing test |
| Weekly regression | JOB_TYPE=run-all on UAT |
| Feature demo prep | JOB_TYPE=run-feature for that feature |
| Need a new test | JOB_TYPE=generate to create it |

---

## What to Do When Tests Fail

### Step 1: Try Autofix

**When to use autofix:**
- Test failed after a UI change (button text changed, class names changed, etc.)
- Error says "selector not found" or "element not visible"
- Test times out waiting for something
- You're not sure what's wrong

**Command:**

```bash
gcloud run jobs execute picoclaw-e2e \
  --region=europe-west4 \
  --container=picoclaw \
  --update-env-vars="JOB_TYPE=autofix" \
  --update-env-vars="JOB_SPEC=<failing-test-path>"
```

**Example - Fix failing login test:**

```bash
gcloud run jobs execute picoclaw-e2e \
  --region=europe-west4 \
  --container=picoclaw \
  --update-env-vars="JOB_TYPE=autofix" \
  --update-env-vars="JOB_SPEC=tests/auth/login.spec.ts"
```

**What autofix does:**
1. Runs the test
2. Reads the error message
3. Figures out what's wrong (selector changed, timing issue, etc.)
4. Fixes the test code
5. Runs the test again
6. Repeats until it passes

**Success rate:** ~80% of issues can be auto-fixed

---

### Step 2: Check Logs

You can view logs directly in the Cloud Run console:

1. Look at the "History" tab on the job page
2. Click on the failed execution
3. Click "View logs" button
4. Look for error messages that say what went wrong

---

### Step 3: Re-run the Test

Sometimes tests fail due to temporary issues (network, server slowness, etc.). Simply re-run the test:

```bash
gcloud run jobs execute picoclaw-e2e \
  --region=europe-west4 \
  --container=picoclaw \
  --update-env-vars="JOB_TYPE=run" \
  --update-env-vars="JOB_SPEC=tests/path/to/test.spec.ts"
```

---

## Common Issues & Solutions

### Issue 1: Test Times Out

**What you'll see:**
- Error: "Test timeout exceeded"
- Test runs for several minutes then fails

**Solutions:**
1. **Re-run the test** - Sometimes it's just temporary slowness
2. **Try autofix** - It may increase the timeout for that specific step
3. **Check if the server is up** - Go to the URL in your browser

---

### Issue 2: "Selector not found" or "Element not visible"

**What you'll see:**
- Error: "Selector not found"
- Error: "Element is not visible"

**Solutions:**
1. **Run autofix** - This fixes 80% of these issues automatically
2. **Check if the feature exists** - Go to the page manually and verify the button/element exists

---

### Issue 3: Wrong Data or Unexpected Behavior

**What you'll see:**
- Error: "Expected X but got Y"
- Test passes but behavior is wrong

**Solutions:**
1. **Verify manually** - Go through the test steps manually
2. **Check recent changes** - Was this behavior intentionally changed?
3. **Check if it's environment-specific** - Does it happen in UAT but not Preview PROD?

---

### Issue 4: Flow Tester Tests Fail

**What you'll see:**
- Error: "Bot message not appearing"
- Bot doesn't respond

**Solutions:**

Run Flow Designer tests first:

```bash
gcloud run jobs execute picoclaw-e2e \
  --region=europe-west4 \
  --container=picoclaw \
  --update-env-vars="JOB_TYPE=run-feature" \
  --update-env-vars="JOB_FEATURE=flow-designer"
```

Then run Flow Tester tests:

```bash
gcloud run jobs execute picoclaw-e2e \
  --region=europe-west4 \
  --container=picoclaw \
  --update-env-vars="JOB_TYPE=run-feature" \
  --update-env-vars="JOB_FEATURE=flow-tester"
```

---

### Issue 5: API Key Tests Fail

**What you'll see:**
- Error: "API key not found in table"
- Error: "Cannot revoke key"

**Solution:**

Run the full settings test suite (ensures correct order):

```bash
gcloud run jobs execute picoclaw-e2e \
  --region=europe-west4 \
  --container=picoclaw \
  --update-env-vars="JOB_TYPE=run-feature" \
  --update-env-vars="JOB_FEATURE=settings"
```

**Important:** Don't run individual settings tests - they depend on each other.

---

### Issue 6: Organization/Member Tests Fail

**What you'll see:**
- Error: "Member not found"
- Error: "Wrong organization selected"

**Solution:**

Run the full organization test suite:

```bash
gcloud run jobs execute picoclaw-e2e \
  --region=europe-west4 \
  --container=picoclaw \
  --update-env-vars="JOB_TYPE=run-feature" \
  --update-env-vars="JOB_FEATURE=organization"
```

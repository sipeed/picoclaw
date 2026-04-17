# Cloud Run Jobs

The `picoclaw-e2e` Cloud Run Job runs via `entrypoint-job.sh`. Behaviour is controlled by `JOB_TYPE` and `ENVIRONMENT` environment variables passed at execution time with `--update-env-vars`.

**Base command:**
```bash
gcloud run jobs execute picoclaw-e2e \
  --region=europe-west4 \
  --container=picoclaw \
  --update-env-vars="JOB_TYPE=<type>" \
  --update-env-vars="ENVIRONMENT=<env>"
```

> `--container=picoclaw` is required for multi-container jobs. Without it gcloud crashes with `'NoneType' object has no attribute 'template'`.

---

## ENVIRONMENT

Controls which dashboard is targeted. Defaults to `UAT` if not specified.

| Value | URL |
|---|---|
| `UAT` *(default)* | `https://dashboard.int3nt.info` |
| `PREVIEW-PROD` | `https://dashboard-preview.intentai.com` |

```bash
# Run against Preview PROD
gcloud run jobs execute picoclaw-e2e \
  --region=europe-west4 \
  --container=picoclaw \
  --update-env-vars="JOB_TYPE=run" \
  --update-env-vars="JOB_SPEC=tests/auth/login.spec.ts" \
  --update-env-vars="ENVIRONMENT=PREVIEW-PROD"
```

> You can also override the URL directly with `--update-env-vars="BASE_URL=https://..."` which takes precedence over `ENVIRONMENT`.

---

## JOB_TYPE=run-all *(default)*

Runs all Playwright tests in dependency order.

```bash
gcloud run jobs execute picoclaw-e2e \
  --region=europe-west4 \
  --container=picoclaw \
  --update-env-vars="JOB_TYPE=run-all"
```

**Order:** auth → knowledge-base → flow-designer → flow-tester → profile → organization → settings → logs

---

## JOB_TYPE=run

Runs a single Playwright spec file **or an entire folder** of tests.

**Required env vars:**
| Var | Description |
|---|---|
| `JOB_SPEC` | Spec file path or folder path (e.g. `tests/auth/login.spec.ts` or `tests/knowledge-base`) |

```bash
# Single spec file
gcloud run jobs execute picoclaw-e2e \
  --region=europe-west4 \
  --container=picoclaw \
  --update-env-vars="JOB_TYPE=run" \
  --update-env-vars="JOB_SPEC=tests/auth/login.spec.ts"

# All tests in a folder
gcloud run jobs execute picoclaw-e2e \
  --region=europe-west4 \
  --container=picoclaw \
  --update-env-vars="JOB_TYPE=run" \
  --update-env-vars="JOB_SPEC=tests/knowledge-base"
```

**Available folders:** `tests/auth`, `tests/knowledge-base`, `tests/flow-designer`, `tests/flow-tester`, `tests/profile`, `tests/organization`, `tests/settings`, `tests/logs`

---

## JOB_TYPE=autofix

Sends a failing spec file to the picoclaw agent. The agent runs the test, reads the error, edits the spec, and repeats until it passes.

**Required env vars:**
| Var | Description |
|---|---|
| `JOB_SPEC` | Spec file path (e.g. `tests/flow-designer/create-new-flow-custom-node.spec.ts`) |

```bash
gcloud run jobs execute picoclaw-e2e \
  --region=europe-west4 \
  --container=picoclaw \
  --update-env-vars="JOB_TYPE=autofix" \
  --update-env-vars="JOB_SPEC=tests/flow-designer/create-new-flow-custom-node.spec.ts"
```

> The area is derived from the path (e.g. `tests/flow-designer/...` → uses `templates/autofix/flow-designer.txt`).

---

## JOB_TYPE=generate

Generates a new Playwright test file for a given area using a template, then runs and fixes it until it passes.

**Required env vars:**
| Var | Description |
|---|---|
| `JOB_AREA` | Area name (e.g. `flow-designer`) |
| `JOB_TEST_FILE` | Output filename without path or extension (e.g. `create-new-flow-custom-node`) |
| `JOB_STEPS` | Numbered test steps as a plain string |
| `JOB_EXPECTED_RESULT` | Expected results as a plain string |

**Available areas:** `auth`, `flow-designer`, `flow-tester`, `knowledge-base`, `logs`, `organization`, `profile`, `settings`

```bash
gcloud run jobs execute picoclaw-e2e \
  --region=europe-west4 \
  --container=picoclaw \
  --update-env-vars="JOB_TYPE=generate" \
  --update-env-vars="JOB_AREA=flow-designer" \
  --update-env-vars="JOB_TEST_FILE=create-new-flow-custom-node" \
  --update-env-vars="JOB_STEPS=1. Navigate to Flow Designer
2. Click New Flow
3. Add a Custom Node" \
  --update-env-vars="JOB_EXPECTED_RESULT=1. Flow created with custom node visible"
```

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

---

## JOB_TYPE=prompt

Runs the picoclaw agent with a raw prompt. Use this for ad-hoc tasks that don't fit the other job types.

**Required env vars:**
| Var | Description |
|---|---|
| `JOB_PROMPT` | The prompt to send to the picoclaw agent |

```bash
gcloud run jobs execute picoclaw-e2e \
  --region=europe-west4 \
  --container=picoclaw \
  --update-env-vars="JOB_TYPE=prompt" \
  --update-env-vars="JOB_PROMPT=Read tests/auth/login.spec.ts and summarise what it tests."
```

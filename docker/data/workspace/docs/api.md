# Test Server API

A local Express.js server (`docker/data/workspace/server.js`) that exposes endpoints for running Playwright tests and triggering the picoclaw AI agent to generate, run, and auto-fix tests.

**Start the server:**
```bash
cd docker/data/workspace
node server.js
# Listening on http://localhost:3100
```

---

## Endpoints

### `GET /test/files`
Returns all available Playwright spec files.

**Response**
```json
{
  "files": [
    "tests/flow-designer/create-new-flow-custom-node.spec.ts",
    "tests/knowledge-base/schedule-kb-full-sync-advanced.spec.ts"
  ]
}
```

---

### `POST /test/run`
Runs a Playwright spec file and streams output in real-time.

**Body**
| Field | Required | Default | Description |
|---|---|---|---|
| `file` | yes | — | Spec file path from `GET /test/files` |
| `reporter` | no | `"line"` | Playwright reporter (`line`, `list`, `dot`) |

**Example**
```json
{
  "file": "tests/flow-designer/create-new-flow-custom-node.spec.ts",
  "reporter": "line"
}
```

**Response** — `text/plain` chunked stream of Playwright output, ending with `[exit code: N]`

---

### `POST /test/autofix`
Sends a failing spec file to the picoclaw agent, which runs the test, reads the error, edits the spec to fix it, and repeats until it passes. Streams the full agent session output.

**Body**
| Field | Required | Description |
|---|---|---|
| `file` | yes | Spec file path from `GET /test/files` |

**Example**
```json
{
  "file": "tests/flow-designer/create-new-flow-custom-node.spec.ts"
}
```

**Response** — `text/plain` chunked stream. The agent reports iterations, fixes applied, and final PASS/FAIL result.

> The area is derived automatically from the file path (e.g. `tests/flow-designer/...` → uses `templates/autofix/flow-designer.txt`).

---

### `GET /agent/areas`
Returns the list of supported areas for agent endpoints.

**Response**
```json
{
  "areas": ["auth", "flow-designer", "flow-tester", "knowledge-base", "logs", "organization", "profile", "settings"]
}
```

---

### `POST /agent/reference`
Triggers the picoclaw agent to read the dashboard source code for the given area and generate a reference document (e.g. `tests/auth/AUTH_REFERENCE.md`). Run this once per area before generating tests.

**Body**
| Field | Required | Description |
|---|---|---|
| `area` | yes | One of the values from `GET /agent/areas` |

**Example**
```json
{ "area": "auth" }
```

**Response** — `text/plain` chunked stream of the agent session.

---

### `POST /agent/run`
Generates a new Playwright test file for the given area by composing the full agent prompt from a template, filling in only the test-specific parts.

**Body**
| Field | Required | Description |
|---|---|---|
| `area` | yes | One of the values from `GET /agent/areas` |
| `testFile` | yes | Output filename without path or extension (e.g. `login`) |
| `steps` | yes | Numbered test steps as a plain string |
| `expectedResult` | yes | Expected results as a plain string |

**Example**
```json
{
  "area": "auth",
  "testFile": "login",
  "steps": "1. Navigate to /login\n2. Enter email and password\n3. Click Login\n4. Select org Testing2026!",
  "expectedResult": "1. User is redirected to the dashboard\n2. Org name is visible in the sidebar"
}
```

**Response** — `text/plain` chunked stream. The agent reads the reference doc, generates the spec file at `tests/<area>/<testFile>.spec.ts`, runs it, and fixes it until it passes.

---

## Typical Workflow

```
1. POST /agent/reference  { "area": "auth" }
        ↓ generates tests/auth/AUTH_REFERENCE.md

2. POST /agent/run        { "area": "auth", "testFile": "login", "steps": "...", "expectedResult": "..." }
        ↓ generates + runs tests/auth/login.spec.ts

3. POST /test/run         { "file": "tests/auth/login.spec.ts" }
        ↓ re-run anytime

4. POST /test/autofix     { "file": "tests/auth/login.spec.ts" }
        ↓ agent fixes the spec until it passes
```

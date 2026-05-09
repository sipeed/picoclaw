# PicoClaw Integration Fix Design

Date: 2026-05-07

## Scope

Fix 4 integration gaps identified in the CLI/BE/FE audit:

1. **Agent delete response mismatch** — frontend expects `{message}` but backend sends `{status}`
2. **Refactor agent manager** — eliminate code duplication between `pkg/agent/manager` and `web/backend/api/agents.go`
3. **Add Cron API** — expose CRUD for cron jobs via backend REST API
4. **Add MCP API** — expose CRUD for MCP servers via backend REST API
5. **Wire tool registry to API** — make `GET /api/tools` query `pkg/tools/registry.go` instead of static catalog
6. **Add MCP + Cron UI pages** — create frontend pages for MCP and Cron management

## Non-Goals

- Health UI (deferred)
- Web search config UI (already exists per subagent findings — web search tab exists)
- Agent CRUD CLI (CLI only provides direct chat, not CRUD — acceptable)
- Skills system redesign (already well-integrated)

---

## 1. Agent Delete Response Mismatch

### Problem
`web/frontend/src/api/agents.ts:70` expects `Promise<{ message: string }>` but `web/backend/api/agents.go:460` sends `{ status: "ok" }`.

### Fix
Update the frontend to match the backend response:

```typescript
// web/frontend/src/api/agents.ts:70
export async function deleteAgent(slug: string): Promise<{ status: string }> {
  // ... no other changes needed
}
```

The frontend `agents-page.tsx` uses `deleteAgent()` but only logs success via toast — it doesn't read the response body. So this is a low-risk fix.

---

## 2. Refactor Agent Manager

### Problem
`pkg/agent/manager/manager.go` and `web/backend/api/agents.go` both implement identical agent CRUD logic with duplicated types, regex, and file I/O. They diverge over time and cause maintenance burden.

### Approach: Use pkg/agent/manager from API layer

The backend API handler (`web/backend/api/agents.go`) should delegate to `manager.NewManager()`. Both use the same workspace path `~/.picoclaw/workspace/agents`.

### Changes

1. In `web/backend/api/agents.go`:
   - Remove the duplicated `agentManager` struct, `expandAgentPath`, `slugRegex`, and all CRUD methods (List, Get, Create, Update, Delete, Import)
   - Instead, import `github.com/sipeed/picoclaw/pkg/agent/manager` and use `manager.NewManager("")`
   - Keep the HTTP handler wrappers and request/response types
   - `agentManager` becomes a thin wrapper: `&manager.Manager{workspacePath: mgr.workspacePath}` — or just use `manager.NewManager` directly

2. The `agent` struct type in `agents.go` and `pkg/agent/manager/types.go` need to be compared — one may have more fields. Check if they can be unified.

3. Remove `agentCreateRequest`, `agentUpdateRequest`, `agentImportRequest` if `manager.AgentCreateRequest`/`AgentUpdateRequest` exist and are compatible. If `pkg/agent/manager/types.go` uses different struct names, either alias or adapt.

**Decision**: Check `pkg/agent/manager/types.go` to see the actual request/response types before deciding on aliasing vs adaptation.

### Fallback if pkg/agent/manager lacks types
If `pkg/agent/manager/types.go` doesn't export request/response types, keep the HTTP-layer structs in `agents.go` but have the handler methods delegate to `manager.Manager` for actual file I/O. This avoids breaking the API contract while eliminating duplication.

---

## 3. Add Cron API

### CLI Reference
`cmd/picoclaw/internal/cron/` has: list, add, remove, enable, disable. Jobs stored at `workspace/cron/jobs.json`.

### New Backend Endpoints (web/backend/api/cron.go)

| Method | Path | Handler | Purpose |
|--------|------|---------|---------|
| GET | `/api/cron/jobs` | handleListCronJobs | List all jobs |
| POST | `/api/cron/jobs` | handleAddCronJob | Add a new job |
| DELETE | `/api/cron/jobs/{id}` | handleDeleteCronJob | Delete a job |
| POST | `/api/cron/jobs/{id}/enable` | handleEnableCronJob | Enable a job |
| POST | `/api/cron/jobs/{id}/disable` | handleDisableCronJob | Disable a job |

### Implementation
- `pkg/cron/service.go` has `NewCronService(path, nil)` and methods: `ListJobs()`, `AddJob()`, `DeleteJob()`, `EnableJob()`, `DisableJob()`
- Cron service reads/writes `workspace/cron/jobs.json`
- Workspace path comes from config: `cfg.WorkspacePath()` → append `/cron/jobs.json`
- Register routes in `router.go` → `registerCronRoutes(mux)`

### Request/Response Shapes

```json
// GET /api/cron/jobs
{ "jobs": [{ "id": "uuid", "name": "string", "schedule": { "kind": "every"|"cron", "every_ms": 3600000, "expr": "0 9 * * *" }, "message": "string", "channel": "", "to": "", "enabled": true, "last_run": 1234567890, "next_run": 1234567890 }] }

// POST /api/cron/jobs
{ "name": "string", "every": 0, "cron": "0 9 * * *", "message": "string", "channel": "", "to": "" }
// Response: { "job": {...} }

// DELETE /api/cron/jobs/{id} → { "status": "ok" }
// POST /api/cron/jobs/{id}/enable → { "status": "ok" }
// POST /api/cron/jobs/{id}/disable → { "status": "ok" }
```

---

## 4. Add MCP API

### CLI Reference
`cmd/picoclaw/internal/mcp/` has: add, remove, list, edit, test, show. MCP servers stored in `config.json` under `tools.mcp.servers`.

### New Backend Endpoints (web/backend/api/mcp.go)

| Method | Path | Handler | Purpose |
|--------|------|---------|---------|
| GET | `/api/mcp/servers` | handleListMCPServers | List all MCP servers |
| GET | `/api/mcp/servers/{name}` | handleGetMCPServer | Get server config + test |
| POST | `/api/mcp/servers` | handleAddMCPServer | Add new MCP server |
| PUT | `/api/mcp/servers/{name}` | handleUpdateMCPServer | Update server |
| DELETE | `/api/mcp/servers/{name}` | handleDeleteMCPServer | Remove server |
| POST | `/api/mcp/servers/{name}/test` | handleTestMCPServer | Probe server health |

### Implementation
- Read/write `config.json` via `pkg/config`
- MCP server config structure: `name`, `command`, `args[]`, `env{}`, `enabled`
- Workspace path from config for MCP server working dir
- For `test`: use same probe logic from `cmd/picoclaw/internal/mcp/probe.go` (import if available, or replicate inline — check if probe.go exists)

### Request/Response Shapes

```json
// GET /api/mcp/servers
{ "servers": [{ "name": "string", "command": "string", "args": [], "env": {}, "enabled": true, "status": "ok|error|disabled" }] }

// POST /api/mcp/servers
{ "name": "string", "command": "string", "args": [], "env": {}, "enabled": true }

// PUT /api/mcp/servers/{name}
// DELETE /api/mcp/servers/{name} → { "status": "ok" }
// POST /api/mcp/servers/{name}/test → { "status": "ok", "tool_count": 12 }
```

---

## 5. Wire Tool Registry to API

### Problem
`web/backend/api/tools.go` has a hardcoded static `toolCatalog` (lines 75-202) instead of querying `pkg/tools/registry.go` at runtime.

### Fix
Replace the static catalog with a dynamic query:

```go
// In handleListTools, replace static toolCatalog with registry.GetAll()
registry := h.getToolRegistry() // inject or use global singleton
tools := registry.GetAll()
// Map to API response format
```

The challenge: the API response format may differ from `registry.GetAll()` output. Need to map:
- `registry.Tool` has `Name()`, `Description()`, `Parameters()`, `Execute()`
- API response needs `name`, `description`, `category`, `parameters` (JSON schema)

### Approach
Create an adapter function that maps `Tool` interface to `ToolSupportItem` API response struct. Run it at request time so the API always reflects live registry.

### Fallback
If the registry doesn't expose enough metadata (e.g., category), enrich the registry with a `Category()` method on the Tool interface. Or use a static map for category lookup if adding method to interface is too intrusive.

---

## 6. Add MCP + Cron UI Pages

### Frontend Routes
- `/agent/mcp` — MCP server management page
- `/agent/cron` — Cron job management page

### Route Registration
Add to `web/frontend/src/routes/agent.tsx` (or wherever nested agent routes are defined):

```tsx
// agent/mcp route
// agent/cron route
```

### Components to Create
- `web/frontend/src/components/agent/mcp/mcp-page.tsx` — MCP server list with add/edit/delete
- `web/frontend/src/components/agent/mcp/mcp-form-sheet.tsx` — Add/edit MCP server form
- `web/frontend/src/components/agent/cron/cron-page.tsx` — Cron job list with add/delete/enable/disable
- `web/frontend/src/components/agent/cron/cron-form-dialog.tsx` — Add cron job form

### API Clients to Add
- `web/frontend/src/api/cron.ts` — CRUD hooks for cron API
- `web/frontend/src/api/mcp.ts` — CRUD hooks for MCP API

### Design
Follow existing patterns (skills-page.tsx, models-page.tsx). Use sheets/dialogs for forms, list view for main display.

---

## Implementation Order

1. Fix agent delete response mismatch (frontend) — trivial, no risk
2. Add Cron backend API — self-contained, no dependencies
3. Add MCP backend API — self-contained, no dependencies
4. Wire tool registry to API — moderate, touches core pkg
5. Refactor agent manager (use pkg/agent/manager from API) — highest risk, do last
6. Add MCP UI — depends on MCP backend API
7. Add Cron UI — depends on Cron backend API

Steps 2, 3, 6, 7 are independent and can run in parallel via subagents.

---

## Failure Modes

1. **Agent manager refactor breaks API**: If `pkg/agent/manager` types don't match API contract, fallback to keeping handler methods but calling into manager for file I/O only.
2. **Tool registry mapping loses fields**: If `Tool` interface lacks category, add a static category map in `api/tools.go` keyed by tool name, maintained manually until registry is enriched.
3. **Cron/MCP UI diverges from design system**: Follow existing page patterns (skills-page.tsx, models-page.tsx) exactly — same component library, same layout patterns.
4. **MCP server probe fails in API**: The CLI probe uses context with timeout. API handler should also use context with timeout. Fail gracefully with status="error" rather than crashing.
# Research Backend Integration Design
> Status: Draft (pending user approval)
> Date: 2026-05-08
> Author: AI Assistant

## Scope and Non-Goals

### In-Scope
1. **Go Backend Core**: Extend existing packages:
   - `pkg/agent/` to add research agent type and management
   - `pkg/seahorse/` to add research knowledge graph node/edge types
   - `pkg/memory/` to add research report storage
2. **Go Backend API**: Create `web/backend/api/research.go` with endpoints:
   - `GET /api/research/agents` - List research agents
   - `PUT /api/research/agents/{id}/toggle` - Toggle agent active state
   - `GET /api/research/graph` - List knowledge graph nodes
   - `PUT /api/research/graph/nodes` - Update graph nodes
   - `GET /api/research/reports` - List research reports
   - `PUT /api/research/reports` - Update report status/progress
3. **Frontend API Service**: Create `web/frontend/src/api/research.ts` with TanStack Query-ready functions using existing `launcherFetch` pattern
4. **Frontend Integration**: Update all research components to:
   - Remove all hardcoded `defaultAgents`, `defaultReports`, `defaultNodes`
   - Use `useQuery` from TanStack Query to fetch data from new API service
   - Pass API data as props to sub-components

### Non-Goals
- Real-time agent status updates (defer to future)
- Advanced research parameters (scope to initial config only)
- Report export functionality
- Offline mode/fallback to hardcoded data (per user requirement: "no hardcoded details")

## Architecture and Data Flow

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│  Frontend       │     │  Go Backend     │     │  Storage        │
│  research-page  │     │  api/research.go │     │                 │
└───────┬─────────┘     └───────┬─────────┘     └───────┬─────────┘
        │                     │                     │
        │ 1. HTTP Request    │                     │
        │ (launcherFetch)    │                     │
        ├────────────────────►│                     │
        │                     │ 2. Load Config     │
        │                     │ (config.LoadConfig)  │
        │                     ├────────────────────►│
        │                     │                     │
        │                     │ 3. Delegate to     │
        │                     │    pkg/ packages    │
        │                     ├────────────────────►│
        │                     │                     │ 4. SQLite Read/Write
        │                     │                     ├────────────────────►│
        │                     │                     │
        │                     │ 5. Return JSON     │
        │                     │◄────────────────────┤
        │ 6. JSON Response   │                     │
        │◄────────────────────┤                     │
        │                     │                     │
```

## Interfaces/Contracts

### Frontend API (`web/frontend/src/api/research.ts`)
```typescript
import { launcherFetch } from "@/api/http"

// Types
export interface ResearchAgent {
  id: string
  name: string
  active: boolean
  progress: number
  ram: string
  type: "research"
}

export interface ResearchNode {
  name: string
  abbr: string
  x: number
  y: number
}

export interface ResearchReport {
  id: string
  title: string
  pages: number
  words: number
  status: "in-progress" | "complete"
  progress?: number
}

// API Functions (TanStack Query compatible)
export async function listResearchAgents(): Promise<ResearchAgent[]> {
  return launcherFetch<ResearchAgent[]>("/api/research/agents")
}

export async function toggleResearchAgent(id: string): Promise<void> {
  await launcherFetch(`/api/research/agents/${id}/toggle`, { method: "PUT" })
}

export async function listResearchGraph(): Promise<ResearchNode[]> {
  return launcherFetch<ResearchNode[]>("/api/research/graph")
}

export async function listResearchReports(): Promise<ResearchReport[]> {
  return launcherFetch<ResearchReport[]>("/api/research/reports")
}

export async function updateResearchConfig(config: { type: string; depth: string; restrictToGraph: boolean }): Promise<void> {
  await launcherFetch("/api/research/config", { method: "PUT", body: JSON.stringify(config) })
}
```

### Backend Types (additions to existing packages)
#### `pkg/agent/types.go` (extend existing)
```go
// Add to existing Agent struct
type Agent struct {
  // ... existing fields
  Type string `json:"type"` // "general" or "research"
}

// New research agent constants
const (
  ResearchAgentLiterature = "literature-analyzer"
  ResearchAgentExtractor = "data-extractor"
  ResearchAgentValidator = "fact-validator"
  ResearchAgentSynthesizer = "synthesizer"
)
```

#### `pkg/seahorse/types.go` (extend existing)
```go
// Add new research graph node type
type ResearchGraphNode struct {
  Name string `json:"name"`
  Abbr string `json:"abbr"`
  X    float64 `json:"x"`
  Y    float64 `json:"y"`
}
```

#### `pkg/memory/types.go` (extend existing)
```go
// Add new research report type
type ResearchReport struct {
  ID       string `json:"id"`
  Title    string `json:"title"`
  Pages    int    `json:"pages"`
  Words    int    `json:"words"`
  Status   string `json:"status"` // "in-progress" or "complete"
  Progress  int    `json:"progress,omitempty"`
}
```

## Error Handling

### Frontend
- TanStack Query error handling with `error` state in components
- Reuse existing error toast pattern from other pages (skills/agents)
- No offline fallback (per user requirement)

### Backend
- Standard HTTP error codes matching existing pattern (tools.go, skills.go):
  - 400: Bad request (invalid agent ID, invalid config)
  - 404: Resource not found
  - 500: Internal server error
- JSON error response format: `{"error": "message"}`

## Testing Strategy
1. **Go Unit Tests**:
   - `pkg/agent/` research agent type tests
   - `pkg/seahorse/` research graph node tests
   - `pkg/memory/` research report tests
2. **Frontend**:
   - No new unit tests required (existing research components tested via `make test`)
3. **Integration**:
   - Test API endpoints with `go test ./web/backend/...`

## Rollout Notes
1. Run `make generate` before `make build` (per AGENTS.md)
2. New SQLite tables added to existing `pkg/seahorse/store.db` and `pkg/memory/store.db`
3. No data migration required (new tables only)
4. Frontend requires `pnpm run generate` to update route tree (already done)

## Failure-Mode Check
1. **Critical**: Extending existing packages introduces breaking changes to core functionality (e.g., modifying `SummaryNode` in `pkg/seahorse` affects context compaction)
   - **Fix**: Add new research-specific types without modifying existing core types. Use composition instead of mutation.
2. **Minor**: Research agents conflict with existing general agent types in `pkg/agent/`
   - **Fix**: Add `Type` field to existing `Agent` struct with default value "general" to avoid breaking changes.
3. **Minor**: Frontend API calls fail if backend is not running
   - **Fix**: Handled by TanStack Query error states, no offline fallback (per user requirement)

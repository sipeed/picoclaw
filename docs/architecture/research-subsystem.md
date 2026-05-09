# Research Subsystem

> Added: 2026-05-09

PicoClaw is an autonomous agent system. The research subsystem extends the core agent with specialized research capabilities, allowing the agent to perform literature analysis, data extraction, fact validation, and synthesis of research findings.

## System Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│                        PicoClaw Autonomous Agent                    │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────────────┐    │
│  │   Core      │    │   Agent     │    │   Research          │    │
│  │   Agent     │───►│   Manager   │───►│   Subsystem         │    │
│  │   Loop      │    │   (pkg/agent │    │   (pkg/agent/       │    │
│  │             │    │    /manager)│    │    research)        │    │
│  └─────────────┘    └─────────────┘    └─────────────────────┘    │
│                                              │                      │
│                                              ▼                      │
│                           ┌────────────────────────────────┐        │
│                           │   Research Data Layer          │        │
│                           │   (pkg/memory/JSONLStore)     │        │
│                           └────────────────────────────────┘        │
│                                                                      │
│  ┌─────────────────────────────────────────────────────────────────┐ │
│  │  Gateway API Layer (pkg/gateway/agent_api.go)                  │ │
│  │  - GET /api/research/agents  - List research agents           │ │
│  │  - GET /api/research/graph   - List knowledge graph nodes     │ │
│  │  - GET /api/research/reports - List research reports          │ │
│  └─────────────────────────────────────────────────────────────────┘ │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

## Architecture Layers

### 1. Domain Layer (`pkg/agent/`)

The agent package follows clean architecture with two sub-packages:

#### `pkg/agent/manager/`
Core agent lifecycle management:
- Create, read, update, delete agents
- Agent configuration (name, description, system prompt, model)
- Tool permissions and status management

#### `pkg/agent/research/`
Research-specific capabilities:
- `types.go` - Domain models for research agents, nodes, and reports
- `manager.go` - Business logic for research data operations

### 2. Data Layer (`pkg/memory/`)

Research data persistence using JSONLStore:

| Type | Storage Key | Description |
|------|-------------|-------------|
| `ResearchAgent` | `research_agents.jsonl` | Research agent instances |
| `ResearchNode` | `research_nodes.jsonl` | Knowledge graph nodes |
| `ResearchReport` | `research_reports.jsonl` | Generated research reports |

### 3. Gateway Layer (`pkg/gateway/`)

HTTP handlers that delegate to domain layer:

```go
// Gateway imports domain package
import "github.com/sipeed/picoclaw/pkg/agent/research"

// Handler uses manager instead of direct store access
func handleResearchAgentsList(w http.ResponseWriter, r *http.Request) {
    agents, err := researchManager.ListAgents()
    // ...
}
```

## How Research Works

### Agent Types

The research subsystem supports multiple specialized research agent types:

| Agent Type | Purpose |
|------------|---------|
| `literature-analyzer` | Analyze and summarize academic papers |
| `data-extractor` | Extract structured data from documents |
| `fact-validator` | Verify claims against known information |
| `synthesizer` | Combine findings into coherent reports |

### Data Flow

1. **Initialization**: Gateway initializes `research.Manager` with `memory.JSONLStore`
2. **HTTP Request**: Client calls `/api/research/*` endpoints
3. **Domain Processing**: Manager applies business logic (type conversion, validation)
4. **Data Persistence**: Memory layer reads/writes JSONL files
5. **Response**: JSON data returned to client

### API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/research/agents` | GET | List all research agents |
| `/api/research/graph` | GET | List knowledge graph nodes |
| `/api/research/reports` | GET | List research reports |
| `/api/research/config` | GET/PUT | Get or update research configuration |
| `/api/research/export` | GET | Export report as Markdown or PDF |
| `/ws/research` | WS | WebSocket for real-time updates |

### Configuration

Research can be customized via the config API:
- **type**: Research type (literature, comprehensive, systematic, exploratory)
- **depth**: Analysis depth (shallow, deep, ultra)
- **restrict_to_graph**: Limit to knowledge graph sources only

### Export

Reports can be exported in two formats:
- **Markdown**: Plain text with markdown formatting
- **PDF**: Generated via backend (returns text for now)

### Real-time Updates

WebSocket connection at `/ws/research` broadcasts:
- `agent_update`: When agent status changes
- `report_update`: When report progress changes
- `config_change`: When configuration is updated

The frontend automatically reconnects on disconnect with 3-second backoff.

## Code Structure

```
pkg/agent/
├── manager/           # Core agent management
│   ├── manager.go     # Agent CRUD operations
│   └── types.go       # Agent domain types
│
└── research/          # Research subsystem
    ├── manager.go     # Research data operations + ConfigStore
    └── types.go       # Research domain types + Config

pkg/memory/
├── jsonl.go           # JSONL storage implementation
├── types.go           # Memory types (includes ResearchAgent, etc.)
└── store.go          # Store interface

pkg/gateway/
├── agent_api.go       # HTTP handlers (now uses research.Manager)
└── websocket/
    └── hub.go         # WebSocket hub for real-time updates
```

### Frontend Structure

```
web/frontend/
├── src/
│   ├── api/
│   │   └── research.ts     # API functions with offline fallback
│   ├── hooks/
│   │   └── use-research-websocket.ts  # WebSocket hook
│   ├── components/agent/research/
│   │   ├── research-page.tsx     # Main research page
│   │   ├── research-config.tsx   # Configuration panel
│   │   ├── research-agents.tsx   # Agent list
│   │   ├── research-graph.tsx    # Knowledge graph
│   │   └── research-reports.tsx  # Reports with export
│   └── routes/agent/
│       └── research.tsx          # Route definition
```

## Clean Architecture Principles

The research subsystem follows these principles:

1. **Domain/Business Logic in `pkg/`**: Research logic lives in the domain layer, not in the gateway
2. **Gateway as HTTP Handler Only**: Gateway only handles HTTP concerns (request parsing, response formatting)
3. **Dependency Inversion**: `research.Manager` depends on a `Store` interface, not concrete implementation
4. **Type Separation**: Domain types in `research` are separate from persistence types in `memory`

## Integration with Core Agent

The research subsystem integrates with the core autonomous agent through:

1. **Agent Manager**: Uses same workspace path pattern (`~/.picoclaw/workspace/`)
2. **Configuration**: Research agents can use same model selection as core agents
3. **Extensibility**: Future research capabilities can be added to `pkg/agent/research/` without modifying core agent

## Implemented Features

All previously planned features are now implemented:

1. **Real-time agent status updates via WebSocket** - `pkg/gateway/websocket/hub.go`
2. **Advanced research parameters configuration UI** - Config panel with type/depth/restrict options
3. **Report export (PDF, Markdown)** - `/api/research/export` endpoint with download functionality
4. **Offline mode with graceful degradation** - API functions return default data when backend unavailable

## Future Enhancements

Potential future enhancements:
- Real-time agent execution (actually running research tasks)
- Multiple simultaneous research projects
- Research history and versioning
- Collaboration features (share research with team)
- Integration with external research databases (arXiv, PubMed, etc.)
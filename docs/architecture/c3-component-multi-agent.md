# C3 - Component Diagram: Multi-Agent Framework

Detailed view of the multi-agent collaboration components.
Includes both current (PR #423) and planned (Phases 1-4) components.

## Core Multi-Agent Components (Current)

```mermaid
C4Component
    title Component Diagram - Multi-Agent Collaboration (pkg/multiagent + pkg/agent)

    Container_Boundary(agent_pkg, "pkg/agent") {
        Component(loop, "AgentLoop", "loop.go", "Core orchestrator: tool loop, LLM calls, session management")
        Component(registry, "AgentRegistry", "registry.go", "Stores AgentInstance map, resolves by ID, lists all agents")
        Component(instance, "AgentInstance", "instance.go", "Per-agent config: ID, Name, Role, SystemPrompt, tools, workspace")
        Component(resolver_adapter, "registryResolver", "loop.go", "Adapter: bridges AgentRegistry to multiagent.AgentResolver interface")
    }

    Container_Boundary(multiagent_pkg, "pkg/multiagent") {
        Component(blackboard, "Blackboard", "blackboard.go", "Thread-safe shared key-value store with author/scope/timestamp metadata")
        Component(bb_tool, "BlackboardTool", "blackboard_tool.go", "LLM tool: read/write/list/delete on shared context")
        Component(handoff, "ExecuteHandoff", "handoff.go", "Resolves target agent, writes context to blackboard, delegates via RunToolLoop")
        Component(handoff_tool, "HandoffTool", "handoff_tool.go", "LLM tool: delegates sub-task to another agent with optional context")
        Component(list_tool, "ListAgentsTool", "list_agents_tool.go", "LLM tool: returns all registered agents with ID/Name/Role")
        Component(agent_resolver, "AgentResolver", "handoff.go", "Interface: GetAgentInfo(id), ListAgents() - decouples from pkg/agent")
    }

    Container_Boundary(routing_pkg, "pkg/routing") {
        Component(route_resolver, "RouteResolver", "route.go", "Matches message to agent based on channel/chat/peer bindings")
        Component(session_key, "SessionKeyBuilder", "session_key.go", "Builds per-agent session keys from channel+chat+agent")
        Component(agent_id, "AgentID", "agent_id.go", "Normalizes agent identifiers")
    }

    Container_Boundary(providers_pkg, "pkg/providers") {
        Component(fallback, "FallbackChain", "fallback.go", "Tries candidates in order, skips cooled-down, classifies errors")
        Component(cooldown, "CooldownTracker", "cooldown.go", "Per-model failure tracking with exponential backoff")
        Component(error_cls, "ErrorClassifier", "error_classifier.go", "Maps HTTP errors to FailoverReason: rate_limit, billing, auth, etc.")
        Component(factory, "CreateProvider", "factory.go", "Resolves config to provider: HTTP, CLI, OAuth, Fallback")
    }

    Rel(loop, registry, "GetInstance(agentID)")
    Rel(loop, resolver_adapter, "Creates on init")
    Rel(resolver_adapter, registry, "Delegates to")
    Rel(resolver_adapter, agent_resolver, "Implements")

    Rel(loop, blackboard, "getOrCreateBlackboard(sessionKey)")
    Rel(loop, bb_tool, "Registers when >1 agent")
    Rel(loop, handoff_tool, "Registers when >1 agent")
    Rel(loop, list_tool, "Registers when >1 agent")

    Rel(handoff_tool, handoff, "Calls ExecuteHandoff()")
    Rel(handoff, agent_resolver, "GetAgentInfo(targetID)")
    Rel(handoff, blackboard, "Writes handoff context")
    Rel(handoff, loop, "Calls RunToolLoop() for target agent")

    Rel(bb_tool, blackboard, "CRUD operations")
    Rel(list_tool, agent_resolver, "ListAgents()")

    Rel(loop, route_resolver, "ResolveAgent(msg)")
    Rel(loop, session_key, "BuildKey(channel, chat, agent)")
    Rel(loop, fallback, "Chat() with fallback")
    Rel(fallback, cooldown, "Check/update cooldown")
    Rel(fallback, error_cls, "ClassifyError()")
```

## Planned Components (Phases 1-4)

```mermaid
C4Component
    title Planned Components - Hardening Phases

    Container_Boundary(tools_pkg, "pkg/tools (Phase 1-3)") {
        Component(hooks, "ToolHook", "hooks.go", "BeforeExecute/AfterExecute interface for tool call interception")
        Component(groups, "ToolGroups", "groups.go", "Named tool groups: fs, web, exec, sessions, memory")
        Component(policy, "PolicyPipeline", "policy.go", "Layered allow/deny: global -> per-agent -> per-depth")
        Component(loop_det, "LoopDetector", "loop_detector.go", "Generic repeat + ping-pong detection with configurable thresholds")
    }

    Container_Boundary(multiagent_new, "pkg/multiagent (Phase 3-4)") {
        Component(cascade, "CascadeStop", "cascade.go", "RunRegistry + recursive context cancellation")
        Component(spawn, "AsyncSpawn", "spawn.go", "Non-blocking agent invocation via goroutines")
        Component(announce, "AnnounceProtocol", "announce.go", "Result delivery: steer/queue/direct modes")
    }

    Container_Boundary(providers_new, "pkg/providers (Phase 3)") {
        Component(auth_rot, "AuthRotator", "auth_rotation.go", "Round-robin profiles + 2-track cooldown (transient + billing)")
    }

    Container_Boundary(gateway_new, "pkg/gateway (Phase 4)") {
        Component(dedup, "DedupCache", "dedup.go", "Idempotency layer with TTL-based deduplication")
    }

    Rel(hooks, loop_det, "AfterExecute feeds detection")
    Rel(hooks, policy, "BeforeExecute applies policy")
    Rel(policy, groups, "Resolves group references")
    Rel(cascade, spawn, "Tracks child runs")
    Rel(spawn, announce, "Delivers results")
    Rel(auth_rot, fallback, "Enhances with profile rotation")
```

## Known Issues (Pre-Phase 1)

```mermaid
graph TD
    BUG1[Blackboard Split-Brain]:::critical
    BUG2[No Recursion Guard]:::critical
    BUG3[Handoff Ignores Allowlist]:::high
    BUG4[SubagentsConfig.Model Unused]:::low

    BUG1 --> FIX1[Phase 1a: Unify board per session]
    BUG2 --> FIX2[Phase 1b: Depth + cycle detection]
    BUG3 --> FIX3[Phase 1c: Check CanSpawnSubagent]
    BUG4 --> FIX4[Defer to Phase 4]

    classDef critical fill:#ef4444,color:#fff
    classDef high fill:#f59e0b,color:#000
    classDef low fill:#6b7280,color:#fff
```

### Blackboard Split-Brain Detail

```mermaid
sequenceDiagram
    participant RS as registerSharedTools
    participant BT as BlackboardTool
    participant RL as runAgentLoop
    participant SP as System Prompt

    Note over RS: At startup
    RS->>RS: sharedBoard := NewBlackboard()
    RS->>BT: NewBlackboardTool(sharedBoard, agentID)

    Note over RL: At runtime (per message)
    RL->>RL: sessionBoard := getOrCreateBlackboard(sessionKey)
    RL->>SP: sessionBoard.Snapshot() → inject into system prompt

    Note over BT,SP: BUG: sharedBoard ≠ sessionBoard
    BT->>RS: Write to sharedBoard ← WRONG BOARD
    SP->>RL: Read from sessionBoard ← DIFFERENT OBJECT

    Note over BT,SP: FIX: SetBoard(sessionBoard) before execution
```

## Blackboard Data Model

```mermaid
classDiagram
    class Blackboard {
        -entries map[string]BlackboardEntry
        -mu sync.RWMutex
        +Set(key, value, author, scope)
        +Get(key) string
        +GetEntry(key) BlackboardEntry
        +Delete(key)
        +List() []BlackboardEntry
        +Snapshot() string
        +Size() int
        +MarshalJSON() []byte
        +UnmarshalJSON([]byte)
    }

    class BlackboardEntry {
        +Key string
        +Value string
        +Author string
        +Scope string
        +Timestamp time.Time
    }

    class BlackboardTool {
        -board *Blackboard
        -agentID string
        +Name() string
        +Execute(args) string
        +SetBoard(board) void
    }

    class HandoffRequest {
        +FromAgentID string
        +ToAgentID string
        +Task string
        +Context map[string]string
        +Depth int
        +Visited []string
    }

    class HandoffResult {
        +AgentID string
        +Response string
        +Success bool
        +Error string
        +Iterations int
    }

    class AgentResolver {
        <<interface>>
        +GetAgentInfo(agentID) *AgentInfo
        +ListAgents() []AgentInfo
    }

    class AllowlistChecker {
        <<interface>>
        +CanHandoff(from, to) bool
    }

    class AgentInfo {
        +ID string
        +Name string
        +Role string
        +SystemPrompt string
    }

    Blackboard "1" --> "*" BlackboardEntry : stores
    BlackboardTool --> Blackboard : operates on
    HandoffRequest ..> AgentResolver : resolved via
    HandoffRequest ..> AllowlistChecker : verified by
    AgentResolver --> AgentInfo : returns
```

## Tool Policy Pipeline (Phase 2)

```mermaid
graph LR
    subgraph "Input"
        ALL[All Registered Tools]
    end

    subgraph "Pipeline Steps"
        S1[Global Allow/Deny]
        S2[Per-Agent Allow/Deny]
        S3[Per-Depth Deny]
        S4[Sandbox Override]
    end

    subgraph "Output"
        FINAL[Filtered Tools for Agent]
    end

    ALL --> S1 --> S2 --> S3 --> S4 --> FINAL

    style S1 fill:#3b82f6,color:#fff
    style S2 fill:#f59e0b,color:#000
    style S3 fill:#ef4444,color:#fff
    style S4 fill:#8b5cf6,color:#fff
```

```mermaid
graph TD
    subgraph "Tool Groups"
        GFS["group:fs<br/>read_file, write_file, edit_file, append_file, list_dir"]
        GWEB["group:web<br/>web_search, web_fetch"]
        GEXEC["group:exec<br/>exec"]
        GSESS["group:sessions<br/>blackboard, handoff, list_agents, spawn"]
        GMEM["group:memory<br/>memory_search, memory_get"]
    end

    subgraph "Depth Deny Rules"
        D0["Depth 0 (main)<br/>Full access"]
        D1["Depth 1+ (subagent)<br/>Deny: gateway"]
        DL["Depth = max (leaf)<br/>Deny: spawn, handoff, list_agents"]
    end
```

## Async Multi-Agent Flow (Phase 4)

```mermaid
sequenceDiagram
    participant U as User
    participant MA as Main Agent
    participant SP as AsyncSpawn
    participant SA1 as Subagent: Researcher
    participant SA2 as Subagent: Analyst
    participant AN as AnnounceProtocol
    participant BB as Blackboard

    U->>MA: "Research and analyze market trends"
    MA->>SP: AsyncSpawn(researcher, "find market data")
    SP-->>MA: RunID: abc-123
    MA->>SP: AsyncSpawn(analyst, "prepare analysis framework")
    SP-->>MA: RunID: def-456
    MA-->>U: "Working on it — spawned 2 agents..."

    par Parallel execution
        SA1->>BB: write("market_data", findings)
        SA1-->>AN: Complete: "Found 5 key trends"
    and
        SA2->>BB: write("framework", analysis_template)
        SA2-->>AN: Complete: "Framework ready"
    end

    AN->>MA: Announce(researcher result) [steer]
    AN->>MA: Announce(analyst result) [queue]

    MA->>BB: read("market_data")
    MA->>BB: read("framework")
    MA->>MA: Synthesize results
    MA-->>U: "Here's the market analysis..."
```

## Provider Protocol Architecture (PR #213 + #283)

```mermaid
graph TB
    subgraph "Config Layer"
        CFG[config.json]
        ML[model_list - future]
    end

    subgraph "Factory (factory.go)"
        RS[resolveProviderSelection]
        CP[CreateProvider]
    end

    subgraph "Protocol Families"
        subgraph "OpenAI-Compatible"
            OC[openai_compat/provider.go]
            HTTP[HTTPProvider - thin delegate]
        end
        subgraph "Anthropic"
            ANT[anthropic/provider.go]
            CP2[ClaudeProvider]
        end
        subgraph "CLI-Based"
            CC[ClaudeCliProvider]
            CX[CodexCliProvider]
        end
        subgraph "Resilience"
            FB[FallbackChain]
            CD[CooldownTracker]
            EC[ErrorClassifier]
            AR[AuthRotator - Phase 3]
        end
    end

    CFG --> RS
    ML -.-> RS
    RS --> CP
    CP --> HTTP
    CP --> ANT
    CP --> CC
    CP --> CX
    HTTP --> OC
    FB --> CD
    FB --> EC
    AR -.-> FB
```

# Sequence Diagrams

Runtime interaction flows for multi-agent collaboration.

## 1. Agent Handoff Flow (Current)

A main agent delegates a sub-task to a specialized agent.

```mermaid
sequenceDiagram
    participant U as User (Discord/Telegram)
    participant CH as Channel Manager
    participant AL as AgentLoop
    participant RR as RouteResolver
    participant MA as Main Agent
    participant LLM as LLM Provider
    participant HT as HandoffTool
    participant BB as Blackboard
    participant AR as AgentResolver
    participant SA as Specialized Agent

    U->>CH: "Translate this code to Python"
    CH->>AL: Message{channel, chat_id, content}
    AL->>RR: ResolveAgent(channel, chat_id)
    RR-->>AL: "main"
    AL->>MA: Load AgentInstance + session
    AL->>BB: getOrCreateBlackboard(sessionKey)
    AL->>AL: Inject BB snapshot into system prompt
    AL->>LLM: Chat(messages + tools)
    LLM-->>AL: ToolCall{name: "handoff", args: {target: "coder", task: "translate to python"}}

    AL->>HT: Execute(args)
    HT->>AR: GetAgentInfo("coder")
    AR-->>HT: AgentInfo{ID: "coder", Role: "Code Expert"}
    HT->>BB: Set("handoff_context_coder", task + context)
    HT->>AL: RunToolLoop(coderAgent, task)

    AL->>SA: Load AgentInstance("coder")
    AL->>LLM: Chat(coder_system_prompt + task)
    LLM-->>AL: "Here's the Python translation..."
    AL-->>HT: HandoffResult{Response: "...", Success: true}

    HT-->>AL: Tool result string
    AL->>LLM: Chat(messages + tool_result)
    LLM-->>AL: "The coder agent translated your code: ..."
    AL->>CH: Send response
    CH->>U: "The coder agent translated your code: ..."
```

## 2. Blackboard Shared Context Flow (Current)

Multiple agents share data through the blackboard within a session.

```mermaid
sequenceDiagram
    participant A1 as Agent: Researcher
    participant BB as Blackboard
    participant AL as AgentLoop
    participant A2 as Agent: Writer

    Note over A1, A2: Same session, shared blackboard

    A1->>BB: BlackboardTool.write("findings", "3 key points...")
    BB-->>A1: OK

    A1->>BB: BlackboardTool.write("sources", "arxiv:2024...")
    BB-->>A1: OK

    Note over AL: Handoff from Researcher to Writer

    AL->>BB: Snapshot()
    BB-->>AL: "findings: 3 key points... (by researcher)\nsources: arxiv:2024... (by researcher)"

    AL->>A2: System prompt + snapshot + task

    A2->>BB: BlackboardTool.read("findings")
    BB-->>A2: "3 key points..."

    A2->>BB: BlackboardTool.read("sources")
    BB-->>A2: "arxiv:2024..."

    A2->>BB: BlackboardTool.write("draft", "Article based on findings...")
    BB-->>A2: OK

    Note over BB: Blackboard state:<br/>findings (by researcher)<br/>sources (by researcher)<br/>draft (by writer)
```

## 3. Model Fallback Chain Flow (Current)

Provider resilience with automatic failover.

```mermaid
sequenceDiagram
    participant AL as AgentLoop
    participant FB as FallbackChain
    participant CD as CooldownTracker
    participant EC as ErrorClassifier
    participant P1 as Primary: GPT-4o
    participant P2 as Fallback: Claude-3.5
    participant P3 as Fallback: DeepSeek

    AL->>FB: Chat(messages)

    FB->>CD: IsAvailable("gpt-4o")?
    CD-->>FB: Yes

    FB->>P1: Chat(messages)
    P1-->>FB: Error 429 (rate limit)

    FB->>EC: ClassifyError(err)
    EC-->>FB: FailoverReason: RATE_LIMITED (retriable)

    FB->>CD: RecordFailure("gpt-4o", RATE_LIMITED)
    Note over CD: gpt-4o cooldown: 30s

    FB->>CD: IsAvailable("claude-3.5")?
    CD-->>FB: Yes

    FB->>P2: Chat(messages)
    P2-->>FB: Error 503 (overloaded)

    FB->>EC: ClassifyError(err)
    EC-->>FB: FailoverReason: OVERLOADED (retriable)

    FB->>CD: RecordFailure("claude-3.5", OVERLOADED)

    FB->>CD: IsAvailable("deepseek")?
    CD-->>FB: Yes

    FB->>P3: Chat(messages)
    P3-->>FB: LLMResponse{Content: "..."}

    FB->>CD: RecordSuccess("deepseek")
    FB-->>AL: LLMResponse
```

## 4. Route Resolution Flow (Current)

How incoming messages are routed to the correct agent.

```mermaid
sequenceDiagram
    participant MSG as Incoming Message
    participant RR as RouteResolver
    participant REG as AgentRegistry
    participant SK as SessionKeyBuilder

    MSG->>RR: ResolveAgent(channel:"discord", chat:"123", peer_kind:"guild", peer_id:"456")

    RR->>RR: Check bindings
    Note over RR: Binding: {channel: "discord", chat: "123"} -> "support-agent"

    alt Match found
        RR-->>MSG: "support-agent"
    else No match
        RR->>REG: GetDefault()
        REG-->>RR: "main"
        RR-->>MSG: "main"
    end

    MSG->>SK: BuildKey(channel, chat, agentID)
    SK-->>MSG: "discord:123:support-agent"
    Note over MSG: Session key used for:<br/>- Session history<br/>- Blackboard lookup<br/>- State persistence
```

## 5. Multi-Agent Configuration Lifecycle (Current)

From config.json to running agents.

```mermaid
sequenceDiagram
    participant CFG as config.json
    participant REG as AgentRegistry
    participant INST as AgentInstance
    participant LOOP as AgentLoop
    participant TOOLS as Tool Registry

    CFG->>REG: Parse agents.list[]

    loop For each AgentConfig
        REG->>INST: NewAgentInstance(agentCfg, cfg)
        INST->>INST: Set ID, Name, Role, SystemPrompt
        INST->>INST: Create per-agent tools (shell, file, exec)
        INST-->>REG: Register(instance)
    end

    alt No agents.list configured
        REG->>INST: Create implicit "main" agent
        INST-->>REG: Register as default
    end

    REG-->>LOOP: Registry ready

    LOOP->>LOOP: Check registry.ListAgentIDs()

    alt len(agents) > 1
        LOOP->>TOOLS: Register BlackboardTool
        LOOP->>TOOLS: Register HandoffTool
        LOOP->>TOOLS: Register ListAgentsTool
        Note over TOOLS: Multi-agent tools active
    else Single agent
        Note over TOOLS: No multi-agent tools (zero overhead)
    end
```

## 6. Handoff with Guardrails (Phase 1 — Planned)

Handoff with depth limit, cycle detection, and allowlist enforcement.

```mermaid
sequenceDiagram
    participant MA as Main Agent (depth=0)
    participant HT as HandoffTool
    participant GD as Guards
    participant AL as AllowlistChecker
    participant SA as Agent B (depth=1)
    participant SC as Agent C (depth=2)

    MA->>HT: handoff(target="B", task="research")
    HT->>GD: Check depth (0 < maxDepth=3)
    GD-->>HT: OK
    HT->>GD: Check cycle (visited=["main"])
    GD-->>HT: OK (B not in visited)
    HT->>AL: CanHandoff("main", "B")
    AL-->>HT: Allowed

    HT->>SA: ExecuteHandoff(depth=1, visited=["main","B"])
    SA->>HT: handoff(target="C", task="analyze")
    HT->>GD: Check depth (1 < maxDepth=3)
    GD-->>HT: OK
    HT->>GD: Check cycle (visited=["main","B"])
    GD-->>HT: OK (C not in visited)

    HT->>SC: ExecuteHandoff(depth=2, visited=["main","B","C"])
    SC-->>SA: Result
    SA-->>MA: Combined result

    Note over MA: Cycle detection example
    MA->>HT: handoff(target="B", task="...")
    SA->>HT: handoff(target="main", task="...")
    HT->>GD: Check cycle (visited=["main","B"])
    GD-->>HT: BLOCKED: "main" already in visited
    HT-->>SA: Error: handoff cycle detected
```

## 7. Tool Policy Pipeline (Phase 2 — Planned)

How tools are filtered before agent execution.

```mermaid
sequenceDiagram
    participant CFG as Config
    participant PP as PolicyPipeline
    participant GR as ToolGroups
    participant AG as Agent Tools
    participant DEPTH as DepthPolicy

    Note over PP: Agent "researcher" at depth=1

    CFG->>PP: Global deny: ["gateway"]
    PP->>GR: Resolve "gateway" → ["gateway"]
    PP->>PP: Remove "gateway" from tool set

    CFG->>PP: Agent allow: ["group:web", "group:fs", "blackboard"]
    PP->>GR: Resolve groups → [web_search, web_fetch, read_file, ...]
    PP->>PP: Keep only allowed tools

    DEPTH->>PP: Depth=1 deny: ["spawn"]
    PP->>PP: Remove "spawn" from tool set

    PP->>AG: Final tools: [web_search, web_fetch, read_file, write_file, blackboard, handoff]
    Note over AG: Each layer narrows, never widens
```

## 8. Loop Detection (Phase 3 — Planned)

How tool call loops are detected and blocked.

```mermaid
sequenceDiagram
    participant LLM as LLM Provider
    participant HK as ToolHook
    participant LD as LoopDetector
    participant T as Tool

    loop Normal execution (calls 1-9)
        LLM->>HK: BeforeExecute("web_search", {query: "same query"})
        HK->>LD: Check(hash("web_search:same query"))
        LD-->>HK: OK (repeat count < 10)
        HK->>T: Execute
        T-->>HK: Result
        HK->>LD: Record(hash, outcome)
    end

    LLM->>HK: BeforeExecute("web_search", {query: "same query"})
    HK->>LD: Check (repeat count = 10)
    LD-->>HK: WARNING: possible loop
    Note over HK: Warning injected into tool result

    loop Calls 11-19 (with warning)
        LLM->>HK: BeforeExecute("web_search", {query: "same query"})
        HK->>LD: Check
        LD-->>HK: WARNING
    end

    LLM->>HK: BeforeExecute("web_search", {query: "same query"})
    HK->>LD: Check (repeat count = 20)
    LD-->>HK: BLOCKED: tool call repeated too many times
    HK-->>LLM: Error: loop detected, try a different approach
```

## 9. Async Spawn + Announce (Phase 4 — Planned)

Parallel agent execution with result delivery.

```mermaid
sequenceDiagram
    participant U as User
    participant MA as Main Agent
    participant SP as AsyncSpawn
    participant RR as RunRegistry
    participant SA1 as Researcher
    participant SA2 as Analyst
    participant AN as AnnounceProtocol
    participant BB as Blackboard

    U->>MA: "Research market trends and analyze competitors"

    MA->>SP: AsyncSpawn(researcher, "find market data")
    SP->>RR: Register(runID=abc, parent=main)
    SP-->>MA: RunID: abc (non-blocking)

    MA->>SP: AsyncSpawn(analyst, "competitor analysis")
    SP->>RR: Register(runID=def, parent=main)
    SP-->>MA: RunID: def (non-blocking)

    MA-->>U: "Working on it — 2 agents spawned..."

    par Parallel Execution
        SA1->>BB: write("market_data", findings)
        SA1->>RR: Complete(abc)
        SA1->>AN: Announce(to=main, content=findings)
    and
        SA2->>BB: write("competitors", analysis)
        SA2->>RR: Complete(def)
        SA2->>AN: Announce(to=main, content=analysis)
    end

    AN->>MA: Steer: "Researcher completed: found 5 trends"
    AN->>MA: Queue: "Analyst completed: 3 competitors analyzed"

    MA->>BB: read("market_data") + read("competitors")
    MA->>MA: Synthesize
    MA-->>U: "Here's the complete market analysis..."
```

## 10. Cascade Stop (Phase 3 — Planned)

Stopping a parent agent cascades to all children.

```mermaid
sequenceDiagram
    participant U as User
    participant RR as RunRegistry
    participant MA as Main Agent
    participant SA1 as Subagent 1
    participant SA2 as Subagent 2
    participant SA3 as Sub-subagent (child of SA1)

    Note over MA: Active run tree:<br/>main → SA1 → SA3<br/>main → SA2

    U->>RR: CascadeStop("main")

    RR->>MA: Cancel context
    MA->>MA: Stop processing

    RR->>RR: Find children of "main"
    RR->>SA1: Cancel context
    SA1->>SA1: Stop processing

    RR->>RR: Find children of SA1
    RR->>SA3: Cancel context
    SA3->>SA3: Stop processing

    RR->>SA2: Cancel context
    SA2->>SA2: Stop processing

    RR-->>U: Killed 4 runs (main + SA1 + SA2 + SA3)
```

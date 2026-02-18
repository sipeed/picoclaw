# Sequence Diagrams

Runtime interaction flows for multi-agent collaboration.

## 1. Agent Handoff Flow

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

## 2. Blackboard Shared Context Flow

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

## 3. Model Fallback Chain Flow

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

## 4. Route Resolution Flow

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

## 5. Multi-Agent Configuration Lifecycle

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

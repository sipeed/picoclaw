# C3 - Component Diagram: Multi-Agent Framework

Detailed view of the multi-agent collaboration components.

## Core Multi-Agent Components

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
    }

    class HandoffRequest {
        +TargetAgentID string
        +Task string
        +Context map[string]string
        +SessionKey string
    }

    class HandoffResult {
        +AgentID string
        +Response string
        +Success bool
        +Error string
    }

    class AgentResolver {
        <<interface>>
        +GetAgentInfo(agentID) *AgentInfo
        +ListAgents() []AgentInfo
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
    AgentResolver --> AgentInfo : returns
```

## Agent Registry & Instance Model

```mermaid
classDiagram
    class AgentRegistry {
        -agents map[string]*AgentInstance
        -defaultID string
        +Register(instance)
        +GetInstance(id) *AgentInstance
        +GetDefault() *AgentInstance
        +ListAgentIDs() []string
    }

    class AgentInstance {
        +ID string
        +Name string
        +Role string
        +SystemPrompt string
        +Workspace string
        +Model string
        +Skills []string
        +Tools []Tool
        +AllowedSubagents []string
    }

    class AgentConfig {
        +ID string
        +Default bool
        +Name string
        +Role string
        +SystemPrompt string
        +Workspace string
        +Model *AgentModelConfig
        +Skills []string
        +Subagents *SubagentsConfig
    }

    class RouteResolver {
        -bindings []AgentBinding
        +ResolveAgent(channel, chatID, peerKind, peerID) string
    }

    class FallbackChain {
        -primary ModelRef
        -fallbacks []ModelRef
        -cooldown *CooldownTracker
        +Chat(ctx, messages, tools, model, opts) *LLMResponse
    }

    AgentRegistry "1" --> "*" AgentInstance : manages
    AgentConfig ..> AgentInstance : creates
    AgentInstance --> FallbackChain : uses for LLM calls
    RouteResolver --> AgentRegistry : resolves agent from
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
        subgraph "OAuth/Token"
            CA[CodexProvider - OAuth]
            CL[ClaudeProvider - OAuth]
        end
        subgraph "Resilience"
            FB[FallbackChain]
            CD[CooldownTracker]
            EC[ErrorClassifier]
        end
    end

    subgraph "External LLMs"
        OPENAI[OpenAI]
        GROQ[Groq]
        DEEP[DeepSeek]
        OR[OpenRouter]
        ANTH[Anthropic]
        GEM[Gemini]
        OLL[Ollama]
        CLICLI[claude CLI]
        CODCLI[codex CLI]
    end

    CFG --> RS
    ML -.-> RS
    RS --> CP

    CP --> HTTP
    CP --> ANT
    CP --> CC
    CP --> CX
    CP --> CA
    CP --> CL

    HTTP --> OC
    OC --> OPENAI
    OC --> GROQ
    OC --> DEEP
    OC --> OR
    OC --> OLL
    ANT --> ANTH
    CP2 --> ANTH
    OC --> GEM
    CC --> CLICLI
    CX --> CODCLI
    CA --> OPENAI

    FB --> CD
    FB --> EC
    FB -.-> HTTP
    FB -.-> ANT
```

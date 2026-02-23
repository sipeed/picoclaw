# PicoClaw Swarm Mode Architecture

## Overview

PicoClaw Swarm Mode enables multiple PicoClaw instances to work together as a distributed system, providing:
- **Node Discovery**: Automatic peer discovery via UDP gossip protocol
- **Health Monitoring**: Periodic heartbeat and failure detection
- **Load Balancing**: Intelligent task distribution based on node load
- **Handoff Mechanism**: Dynamic task delegation between nodes

## Architecture

The swarm architecture is divided into two distinct planes:

```
┌──────────────────────────────────────────────────────────────���──┐
│                        PicoClaw Swarm                          │
├─────────────────────────────────────────────────────────────────┤
│  Control Plane                   │  Data Plane                  │
│  ├─ Node Discovery              │  ├─ Task Execution           │
│  ├─ Membership Management       │  ├─ Session Transfer         │
│  ├─ Health Monitoring           │  └─ Message Routing          │
│  └─ Load Monitoring             │                              │
└─────────────────────────────────────────────────────────────────┘
```

## Control Plane

The control plane manages cluster state, node membership, and coordination.

### 1. Node Discovery

Nodes discover each other using a lightweight UDP gossip protocol:

```mermaid
sequenceDiagram
    participant Node1
    participant Node2
    participant Node3

    Note over Node1: New node starts
    Node1->>Node1: Bind UDP port (7946)
    Node1->>Node2: Ping + NodeInfo
    Node2->>Node1: Pong + NodeInfo
    Node1->>Node2: Gossip: Known Nodes
    Node2->>Node3: Forward Node1 info
    Node3->>Node2: Ack
    Note over Node1,Node3: Cluster formed
```

**Gossip Protocol Flow:**

```mermaid
graph LR
    A[Node A] -->|Ping| B[Node B]
    B -->|Pong| A
    A -->|Sync| B
    B -->|Forward| C[Node C]
    C -->|Ack| B
    B -->|Update| A
    A -.->|Eventually| C
```

**Key Parameters:**

| Parameter | Default | Description |
|-----------|---------|-------------|
| `gossip_interval` | 1s | Frequency of gossip messages |
| `push_pull_interval` | 30s | Full state sync interval |
| `node_timeout` | 5s | Time before marking node suspect |
| `dead_node_timeout` | 30s | Time before removing dead node |

### 2. Membership Management

Each node maintains a view of the cluster:

```go
type ClusterView struct {
    sync.RWMutex
    localNode  *NodeInfo
    members    map[string]*NodeInfo  // node_id -> NodeInfo
    stateMap   map[string]NodeState   // node_id -> State
}
```

**Node State Machine:**

```mermaid
stateDiagram-v2
    [*] --> Alive: Node joins
    Alive --> Suspect: Missed heartbeat
    Suspect --> Alive: Heartbeat recovered
    Suspect --> Dead: Timeout exceeded
    Dead --> [*]
```

**Node Information:**

```go
type NodeInfo struct {
    ID        string                 // Unique node identifier
    Addr      string                 // IP address
    Port      int                    // Discovery port
    AgentCaps map[string]string      // Capabilities (models, tools)
    LoadScore float64                // Current load (0.0-1.0)
    Labels    map[string]string      // Custom labels
    Timestamp int64                  // Last update time
    Version   string                 // Protocol version
}
```

### 3. Health Monitoring

**Heartbeat Flow:**

```mermaid
sequenceDiagram
    participant N1 as Node 1
    participant N2 as Node 2
    participant HM as Health Monitor

    loop Every gossip_interval
        N1->>N2: Heartbeat (timestamp, load_score)
        N2->>HM: Update state
        HM->>HM: Check timeout
        alt Timeout exceeded
            HM->>HM: Mark as Suspect
            HM->>N1: Probe (are you alive?)
            alt No response
                HM->>HM: Mark as Dead
                HM->>All: Broadcast NodeLeft event
            end
        end
    end
```

### 4. Load Monitoring

Each node continuously monitors its resource usage:

```mermaid
graph TB
    subgraph Load Monitor
        A[CPU Sample] --> D[Score Calculator]
        B[Memory Sample] --> D
        C[Session Count] --> D
        D --> E[Load Score]
    end

    subgraph Weights
        A -.->|0.3| D
        B -.->|0.3| D
        C -.->|0.4| D
    end

    E --> F{Threshold Check}
    F -->|< 0.8| G[Normal Mode]
    F -->|>= 0.8| H[Overloaded - Trigger Handoff]
```

**Load Score Formula:**

```
LoadScore = (CPUUsage × cpu_weight) +
            (MemoryUsage × memory_weight) +
            (SessionRatio × session_weight)

Where:
- CPUUsage = current CPU usage (0.0-1.0)
- MemoryUsage = current memory usage (0.0-1.0)
- SessionRatio = current_sessions / max_sessions
- Default weights: cpu=0.3, memory=0.3, session=0.4
```

## Data Plane

The data plane handles actual task execution and session state transfer.

### 1. Request Flow

```mermaid
sequenceDiagram
    participant User
    participant LB as Entry Point
    participant N1 as Node 1
    participant N2 as Node 2

    User->>LB: Message
    LB->>LB: Check node availability

    alt Node 1 available
        LB->>N1: Forward message
        N1->>N1: Process with LLM
        N1->>User: Response
    else Node 1 overloaded
        LB->>N2: Handoff request
        N2->>N1: Session transfer
        N1->>N2: Session state
        N2->>User: Response (from N2)
    end
```

### 2. Handoff Mechanism

**Handoff Decision Flow:**

```mermaid
flowchart TD
    A[Receive Request] --> B{Should Handoff?}
    B -->|Local load >= threshold| C[Select Target Node]
    B -->|Local load < threshold| D[Process Locally]

    C --> E{Target Available?}
    E -->|Yes| F[Initiate Handoff]
    E -->|No| G[Retry or Fail]

    F --> H[Serialize Session]
    H --> I[Send to Target]
    I --> J{Success?}
    J -->|Yes| K[Update Routing]
    J -->|No| L[Rollback]

    K --> M[Target Processes]
    M --> N[Return Response]
```

**Handoff Protocol:**

```mermaid
sequenceDiagram
    participant Source as Overloaded Node
    participant Target as Selected Node
    participant Client

    Source->>Source: Check load threshold
    Source->>Target: HandoffRequest{session_id, context}

    Target->>Target: Validate request
    alt Accepted
        Target->>Source: HandoffAccept
        Source->>Target: SessionTransfer{messages, tools, state}
        Target->>Target: Restore session
        Target->>Source: TransferComplete
        Source->>Client: Redirect to Target
        Client->>Target: Continue conversation
    else Rejected
        Target->>Source: HandoffReject{reason}
        Source->>Source: Try next node or process locally
    end
```

### 3. Session Transfer

**Session State Structure:**

```go
type SessionState struct {
    SessionID   string
    Messages    []Message      // Conversation history
    Context     map[string]any // Shared context
    Tools       []ToolCall     // Pending tool calls
    Metadata    SessionMeta    // Timestamp, user info, etc.
}
```

**Transfer Flow:**

```mermaid
stateDiagram-v2
    [*] --> Active: Session created
    Active --> Transferring: Handoff initiated
    Transferring --> Active: Transfer failed
    Transferring --> Migrated: Transfer complete
    Migrated --> [*]: Session closed
    Active --> [*]: Session closed
```

## System Architecture

### Component Overview

```mermaid
graph TB
    subgraph "Node 1"
        D1[Discovery Service] --> M1[Membership Manager]
        H1[Handoff Coordinator] --> M1
        L1[Load Monitor] --> H1
        A1[Agent Loop] --> L1
    end

    subgraph "Node 2"
        D2[Discovery Service] --> M2[Membership Manager]
        H2[Handoff Coordinator] --> M2
        L2[Load Monitor] --> H2
        A2[Agent Loop] --> L2
    end

    D1 <-- UDP Gossip --> D2
    D2 <-- UDP Gossip --> D1
    H1 <-- RPC Handoff --> H2
    H2 <-- RPC Handoff --> H1

    TG[Telegram Gateway] --> A1
    TG --> A2
```

### Communication Channels

| Channel | Protocol | Purpose |
|---------|----------|---------|
| Discovery | UDP | Node gossip, heartbeat |
| Handoff RPC | UDP | Session transfer coordination |
| Session Data | UDP | Serialized session state |

## Configuration

### Example Configuration

```json
{
  "swarm": {
    "enabled": true,
    "node_id": "picoclaw-node-1",
    "bind_addr": "127.0.0.1",
    "bind_port": 7946,

    "discovery": {
      "join_addrs": ["127.0.0.1:7946"],
      "gossip_interval": 1,
      "push_pull_interval": 30,
      "node_timeout": 5,
      "dead_node_timeout": 30
    },

    "handoff": {
      "enabled": true,
      "load_threshold": 0.8,
      "timeout": 30,
      "max_retries": 3,
      "retry_delay": 5
    },

    "rpc": {
      "port": 7947,
      "timeout": 10
    },

    "load_monitor": {
      "enabled": true,
      "interval": 5,
      "sample_size": 60,
      "cpu_weight": 0.3,
      "memory_weight": 0.3,
      "session_weight": 0.4
    }
  }
}
```

### Deployment Modes

**Single Entry Point:**

```mermaid
graph LR
    TG[Telegram Gateway] --> N1[Node 1: Coordinator]
    N1 <-- Swarm --> N2[Node 2: Worker]
    N2 <-- Swarm --> N1
    N1 <-- Swarm --> N3[Node 3: Worker]
    N3 <-- Swarm --> N1
```

**Multi-Entry Point (with load balancer):**

```mermaid
graph LR
    LB[Load Balancer] --> N1[Node 1]
    LB --> N2[Node 2]
    N1 <-- Swarm Mesh --> N2
    N2 <-- Swarm Mesh --> N1
    N1 <-- Swarm Mesh --> N3[Node 3]
    N3 <-- Swarm Mesh --> N1
    N2 <-- Swarm Mesh --> N3
    N3 <-- Swarm Mesh --> N2
```

## Event System

The swarm publishes events for monitoring and integration:

```mermaid
graph LR
    A[Node Joined] --> ED[Event Dispatcher]
    B[Node Left] --> ED
    C[Node Suspect] --> ED
    D[Handoff Started] --> ED
    E[Handoff Completed] --> ED

    ED --> H[Handlers]
    H --> L[Logging]
    H --> M[Metrics]
    H --> C[Custom Actions]
```

**Event Types:**

| Event | Description | Payload |
|-------|-------------|---------|
| `NodeJoined` | New node discovered | NodeInfo |
| `NodeLeft` | Node removed | NodeID |
| `NodeSuspect` | Node marked suspect | NodeID |
| `NodeAlive` | Node recovered | NodeInfo |
| `HandoffStarted` | Handoff initiated | HandoffOperation |
| `HandoffCompleted` | Handoff finished | HandoffResult |
| `HandoffFailed` | Handoff error | Error |

## Error Handling

```mermaid
flowchart TD
    A[Operation Failed] --> B{Retryable?}
    B -->|Yes| C[Increment retry count]
    B -->|No| D[Return error]

    C --> E{Max retries reached?}
    E -->|No| F[Wait retry_delay]
    F --> G[Retry operation]

    E -->|Yes| H[Mark node suspect]
    H --> I[Select alternative node]

    G --> J{Success?}
    J -->|Yes| K[Continue]
    J -->|No| C
```

## Security Considerations

1. **Discovery**: UDP gossip is unencrypted - use in trusted networks only
2. **Handoff**: Session data transferred without encryption
3. **Authentication**: No node authentication implemented
4. **Recommendation**: Use VPN or private network for production

## Future Enhancements

1. **Secure Discovery**: Add mTLS for node communication
2. **Consistent Hashing**: Replace random selection with consistent hashing
3. **Session Affinity**: Sticky sessions for better performance
4. **Leader Election**: Automatic coordinator election
5. **Multi-Region**: Geo-distributed cluster support

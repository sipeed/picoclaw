# PicoClaw Swarm API Reference

> PicoClaw Swarm Mode | API Reference
> Last Updated: 2026-02-20

---

## Go API

### Manager

The `Manager` is the main entry point for swarm functionality.

```go
package swarm

// NewManager creates a new swarm manager
func NewManager(
    cfg *config.Config,
    agentLoop *agent.AgentLoop,
    provider providers.LLMProvider,
    localBus *bus.MessageBus,
) *Manager

// Start initializes and starts all swarm components
func (m *Manager) Start(ctx context.Context) error

// Stop gracefully stops all swarm components
func (m *Manager) Stop()

// GetNodeInfo returns this node's information
func (m *Manager) GetNodeInfo() *NodeInfo

// GetDiscoveredNodes returns all discovered nodes
func (m *Manager) GetDiscoveredNodes() []*NodeInfo

// IsNATSConnected returns true if connected to NATS
func (m *Manager) IsNATSConnected() bool

// IsTemporalConnected returns true if connected to Temporal
func (m *Manager) IsTemporalConnected() bool
```

### Dashboard

```go
// NewDashboard creates a new dashboard
func NewDashboard(manager *Manager) *Dashboard

// Start begins the dashboard update loop
func (d *Dashboard) Start(ctx context.Context) error

// Stop stops the dashboard
func (d *Dashboard) Stop()

// Render returns a formatted string representation
func (d *Dashboard) Render() string

// RenderCompact returns a one-line status
func (d *Dashboard) RenderCompact() string

// GetState returns the current dashboard state
func (d *Dashboard) GetState() *DashboardState
```

### Discovery

```go
// NewDiscovery creates a new discovery service
func NewDiscovery(
    bridge *NATSBridge,
    nodeInfo *NodeInfo,
    cfg *config.SwarmConfig,
) *Discovery

// Start begins the discovery service
func (d *Discovery) Start(ctx context.Context) error

// Stop stops the discovery service
func (d *Discovery) Stop()

// GetNodes returns all known nodes, optionally filtered
func (d *Discovery) GetNodes(role, capability string) []*NodeInfo

// SelectWorker selects the best worker for a task
func (d *Discovery) SelectWorker(task *SwarmTask) (*NodeInfo, error)

// SelectWorkerWithPriority selects worker considering priority
func (d *Discovery) SelectWorkerWithPriority(
    task *SwarmTask,
    priority int,
) (*NodeInfo, error)
```

### Worker

```go
// NewWorker creates a new worker
func NewWorker(
    cfg *config.SwarmConfig,
    bridge *NATSBridge,
    temporal *TemporalClient,
    agentLoop *agent.AgentLoop,
    provider providers.LLMProvider,
    nodeInfo *NodeInfo,
) *Worker

// Start begins accepting tasks
func (w *Worker) Start(ctx context.Context) error

// Stop stops accepting new tasks
func (w *Worker) Stop()

// ExecuteTask runs a single task
func (w *Worker) ExecuteTask(ctx context.Context, task *SwarmTask) error

// GetLoad returns current load (0-1)
func (w *Worker) GetLoad() float64
```

### Coordinator

```go
// NewCoordinator creates a new coordinator
func NewCoordinator(
    cfg *config.SwarmConfig,
    bridge *NATSBridge,
    temporal *TemporalClient,
    discovery *Discovery,
    agentLoop *agent.AgentLoop,
    provider providers.LLMProvider,
    localBus *bus.MessageBus,
) *Coordinator

// Start begins accepting requests
func (c *Coordinator) Start(ctx context.Context) error

// Stop stops the coordinator
func (c *Coordinator) Stop()

// SubmitTask submits a task for distributed execution
func (c *Coordinator) SubmitTask(
    ctx context.Context,
    task *SwarmTask,
) (*TaskResult, error)
```

### Types

```go
// NodeInfo represents a node in the swarm
type NodeInfo struct {
    ID           string
    Role         NodeRole  // "coordinator", "worker", "specialist"
    Status       NodeStatus // "online", "busy", "offline", "suspicious"
    Capabilities []string
    Model        string
    Load         float64
    TasksRunning int
    MaxTasks     int
    LastSeen     int64
    StartedAt    int64
    Metadata     map[string]string
}

// SwarmTask represents a task to be executed
type SwarmTask struct {
    ID           string
    Type         string
    Prompt       string
    Priority     int     // 0-3
    Capabilities []string
    ParentID     string
    Dependencies []string
    Context      map[string]interface{}
}

// TaskResult represents the result of a task
type TaskResult struct {
    TaskID    string
    NodeID    string
    Success   bool
    Content   string
    Error     string
    Duration  time.Duration
    Metadata  map[string]interface{}
}
```

---

## NATS Subjects

### Heartbeat

```
picoclaw.swarm.heartbeat.{node_id}
```

### Discovery

```
picoclaw.swarm.discovery.announce
picoclaw.swarm.discovery.query
```

### Task Assignment

```
picoclaw.swarm.task.assign.{node_id}
picoclaw.swarm.task.broadcast.{capability}
```

### Task Results

```
picoclaw.swarm.task.result.{task_id}
picoclaw.swarm.task.progress.{task_id}
```

### System

```
picoclaw.swarm.system.shutdown.{node_id}
```

### Cross-H-id Communication

```
picoclaw.x.{from_hid}.{to_hid}
```

---

## Configuration API

### Config Structure

```go
type SwarmConfig struct {
    Enabled       bool
    Role          string
    Capabilities  []string
    MaxConcurrent int
    HID           string
    SID           string
    NATS          NATSConfig
    Temporal      TemporalConfig
}

type NATSConfig struct {
    URLs              []string
    Embedded          bool
    EmbeddedPort      int
    Credentials       string
    HeartbeatInterval string
    NodeTimeout       string
}

type TemporalConfig struct {
    Address   string
    Namespace string
    TaskQueue string
}
```

### Environment Variables

```go
// LoadFromEnv loads configuration from environment variables
func (c *SwarmConfig) LoadFromEnv() error

// Validate validates the configuration
func (c *SwarmConfig) Validate() error
```

---

## CLI Flags

```bash
# Swarm enablement
--swarm.enabled                    bool    Enable swarm mode
--swarm.role                      string  Node role (coordinator/worker/specialist)

# Identity
--swarm.hid                       string  Human identity
--swarm.sid                       string  Session identity

# NATS
--swarm.nats.urls                 strings NATS server URLs
--swarm.nats.embedded             bool    Use embedded NATS
--swarm.nats.embedded-port        int     Embedded NATS port
--swarm.nats.credentials          string  NATS credentials file

# Capabilities
--swarm.capabilities              strings Node capabilities
--swarm.max-concurrent            int     Max concurrent tasks

# Temporal
--swarm.temporal.address          string  Temporal server address
--swarm.temporal.task-queue       string  Temporal task queue

# Monitoring
--swarm.dashboard                 bool    Enable dashboard output
--swarm.heartbeat-interval       string  Heartbeat interval
--swarm.node-timeout              string  Node timeout
```

---

## Example: Programmatic Usage

### Basic Swarm Setup

```go
package main

import (
    "context"
    "github.com/sipeed/picoclaw/pkg/swarm"
    "github.com/sipeed/picoclaw/pkg/config"
)

func main() {
    ctx := context.Background()

    // Load config
    cfg := config.Load()

    // Create manager
    manager := swarm.NewManager(cfg, agentLoop, provider, bus)

    // Start swarm
    if err := manager.Start(ctx); err != nil {
        log.Fatal(err)
    }
    defer manager.Stop()

    // Create dashboard
    dashboard := swarm.NewDashboard(manager)
    dashboard.Start(ctx)
    defer dashboard.Stop()

    // Run...
    select {}
}
```

### Submit a Task

```go
// Assuming coordinator is set up
task := &swarm.SwarmTask{
    ID:       "task-001",
    Type:     "code_review",
    Prompt:   "Review this code for security issues...",
    Priority: 2, // High priority
    Capabilities: []string{"security", "audit"},
}

result, err := coordinator.SubmitTask(ctx, task)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Result: %s\n", result.Content)
```

### Monitor Swarm Status

```go
// Get all nodes
nodes := manager.GetDiscoveredNodes()
for _, node := range nodes {
    fmt.Printf("%s: %s (%.0f%% load)\n",
        node.ID, node.Status, node.Load*100)
}

// Or use dashboard
fmt.Println(dashboard.Render())
```

---

## Events and Callbacks

### Node Events

```go
// Register callbacks for node join/leave
discovery.OnNodeJoin(func(node *NodeInfo) {
    log.Printf("Node joined: %s", node.ID)
})

discovery.OnNodeLeave(func(nodeID string) {
    log.Printf("Node left: %s", nodeID)
})
```

### Task Events

```go
// Register callbacks for task events
worker.OnTaskAssigned(func(task *SwarmTask) {
    log.Printf("Task assigned: %s", task.ID)
})

worker.OnTaskComplete(func(result *TaskResult) {
    log.Printf("Task complete: %s -> %v",
        result.TaskID, result.Success)
})
```

### Election Events

```go
// Register callbacks for leader changes
electionMgr.OnBecameLeader(func() {
    log.Printf("Became leader!")
    // Promote to coordinator role
})

electionMgr.OnLostLeadership(func() {
    log.Printf("Lost leadership!")
    // Demote to worker role
})
```

---

## Error Handling

### Common Errors

```go
// NATS connection failed
if err := manager.Start(ctx); err != nil {
    if strings.Contains(err.Error(), "NATS") {
        // Check NATS server availability
        log.Fatal("Cannot connect to NATS")
    }
}

// Task timeout
if result, err := coordinator.SubmitTask(ctx, task); err != nil {
    if errors.Is(err, context.DeadlineExceeded) {
        // Handle timeout
    }
}

// No workers available
if _, err := discovery.SelectWorker(task); err != nil {
    if err == ErrNoWorkersAvailable {
        // No matching workers for this task
    }
}
```

---

## Testing API

### Test Helpers

```go
// Create test NATS environment
tn := swarm.StartTestNATS(t)
defer tn.Stop()

// Create test bridge
bridge := swarm.ConnectTestBridge(t, tn.URL, nodeInfo)
defer bridge.Stop()

// Create test node info
nodeInfo := swarm.CreateTestNodeInfo(
    "test-node",
    "worker",
    []string{"test"},
)
```

---

## Performance Considerations

### Connection Pooling

NATS connections are automatically pooled and reused. The `NATSBridge` handles connection management.

### Batch Operations

For multiple task submissions, consider using the workflow API:

```go
// Submit multiple tasks as a workflow
workflow := &swarm.ParallelWorkflow{
    Tasks: []*SwarmTask{task1, task2, task3},
}

result, err := coordinator.ExecuteWorkflow(ctx, workflow)
```

### Memory Management

- Nodes maintain a cache of discovered peers
- Old entries are pruned based on heartbeat timeout
- Adjust `--swarm.node-timeout` for your environment

---

## See Also

- [DEPLOYMENT.md](./DEPLOYMENT.md) - Deployment guide
- [CONFIG.md](./CONFIG.md) - Configuration reference
- [EXAMPLES.md](./EXAMPLES.md) - Code examples

# PicoClaw Swarm Deployment Guide

> PicoClaw Swarm Mode | Deployment Guide
> Last Updated: 2026-02-20

---

## Overview

PicoClaw Swarm enables multiple AI agent instances to work together collaboratively. This guide covers deployment scenarios from local development to production distributed clusters.

---

## Quick Start

### Single Node (Local Development)

```bash
# Start embedded NATS, run as coordinator
picoclaw --swarm.enabled --swarm.role coordinator --swarm.nats.embedded

# In another terminal, start a worker
picoclaw --swarm.enabled --swarm.role worker --swarm.nats.embedded
```

### Multi-Node (Docker Compose)

```yaml
# docker-compose.yml
version: '3.8'
services:
  nats:
    image: nats:latest
    command: "-js"
    ports:
      - "4222:4222"

  coordinator:
    image: picoclaw:latest
    environment:
      - PICOCLAW_SWARM_ENABLED=true
      - PICOCLAW_SWARM_ROLE=coordinator
      - PICOCLAW_SWARM_NATS_URLS=nats://nats:4222

  worker:
    image: picoclaw:latest
    environment:
      - PICOCLAW_SWARM_ENABLED=true
      - PICOCLAW_SWARM_ROLE=worker
      - PICOCLAW_SWARM_CAPABILITIES=code,research
      - PICOCLAW_SWARM_NATS_URLS=nats://nats:4222
    deploy:
      replicas: 3
```

```bash
docker-compose up -d
```

---

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                     PicoClaw Swarm                      │
├─────────────────────────────────────────────────────────┤
│                                                           │
│  ┌──────────────┐         ┌──────────────┐             │
│  │ Coordinator  │◄───────►│   Worker     │             │
│  │              │         │              │             │
│  │ • Task Decomp│         │ • Execute    │             │
│  │ • Scheduling │         │ • Report     │             │
│  │ • Synthesis  │         │              │             │
│  └───────┬──────┘         └──────▲───────┘             │
│          │                        │                      │
│          └────────────────────────┘                      │
│                       │                                 │
│                   ▼─────▼                                │
│              ┌──────────┐                               │
│              │   NATS   │                               │
│              │ JetStream│                               │
│              └──────────┘                               │
│                                                           │
└─────────────────────────────────────────────────────────┘
```

---

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PICOCLAW_SWARM_ENABLED` | Enable swarm mode | `false` |
| `PICOCLAW_SWARM_ROLE` | Node role | `worker` |
| `PICOCLAW_SWARM_NATS_URLS` | NATS servers | `nats://localhost:4222` |
| `PICOCLAW_SWARM_CAPABILITIES` | Node capabilities | `general` |
| `PICOCLAW_SWARM_MAX_CONCURRENT` | Max parallel tasks | `5` |
| `PICOCLAW_SWARM_HID` | Human identity | auto-generated |
| `PICOCLAW_SWARM_SID` | Session identity | auto-generated |

### Config File

```yaml
# ~/.picoclaw/config.yaml
swarm:
  enabled: true
  role: coordinator
  capabilities: ["coordination", "scheduling"]
  max_concurrent: 10

  nats:
    urls:
      - nats://localhost:4222
    embedded: false
    heartbeat_interval: 10s
    node_timeout: 60s

  temporal:
    address: localhost:7233
    task_queue: picoclaw-swarm
```

---

## Deployment Scenarios

### 1. Local Development

**Embedded NATS** (no external dependencies):

```bash
# Terminal 1: Coordinator
picoclaw --swarm.enabled \
         --swarm.role coordinator \
         --swarm.nats.embedded

# Terminal 2-3: Workers
picoclaw --swarm.enabled \
         --swarm.role worker \
         --swarm.capabilities code \
         --swarm.nats.embedded
```

### 2. Single Machine Swarm

**Shared NATS** on the same machine:

```bash
# Start NATS server
nats-server -js -p 4222

# Start nodes
picoclaw --swarm.enabled --swarm.role coordinator \
         --swarm.nats.urls nats://localhost:4222

picoclaw --swarm.enabled --swarm.role worker \
         --swarm.capabilities code,research \
         --swarm.nats.urls nats://localhost:4222
```

### 3. Multi-Machine Swarm

**Distributed NATS cluster**:

```bash
# Machine 1 (NATS + Coordinator)
nats-server -cluster -js -p 4222 \
  -routes nats://machine2:6222,nats://machine3:6222

picoclaw --swarm.enabled --swarm.role coordinator \
         --swarm.nats.urls nats://localhost:4222

# Machine 2-3 (Workers)
picoclaw --swarm.enabled --swarm.role worker \
         --swarm.nats.urls nats://machine1:4222
```

### 4. Kubernetes Deployment

```yaml
# k8s/coordinator.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: picoclaw-coordinator
spec:
  replicas: 1
  selector:
    matchLabels:
      app: picoclaw-coordinator
  template:
    metadata:
      labels:
        app: picoclaw-coordinator
    spec:
      containers:
      - name: picoclaw
        image: picoclaw:latest
        env:
        - name: PICOCLAW_SWARM_ENABLED
          value: "true"
        - name: PICOCLAW_SWARM_ROLE
          value: "coordinator"
        - name: PICOCLAW_SWARM_NATS_URLS
          value: "nats://nats:4222"
---
# k8s/worker.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: picoclaw-worker
spec:
  replicas: 5
  selector:
    matchLabels:
      app: picoclaw-worker
  template:
    metadata:
      labels:
        app: picoclaw-worker
    spec:
      containers:
      - name: picoclaw
        image: picoclaw:latest
        env:
        - name: PICOCLAW_SWARM_ENABLED
          value: "true"
        - name: PICOCLAW_SWARM_ROLE
          value: "worker"
        - name: PICOCLAW_SWARM_NATS_URLS
          value: "nats://nats:4222"
        - name: PICOCLAW_SWARM_CAPABILITIES
          value: "code,research,writing"
```

---

## Node Roles

### Coordinator

**Responsibilities:**
- Receive user requests
- Decompose tasks into sub-tasks
- Route tasks to appropriate workers
- Synthesize results from multiple workers
- Manage workflow state

**Configuration:**
```bash
picoclaw --swarm.role coordinator \
         --swarm.capabilities coordination,scheduling
```

### Worker

**Responsibilities:**
- Execute assigned tasks
- Report progress and results
- Advertise capabilities
- Handle task queues

**Configuration:**
```bash
picoclaw --swarm.role worker \
         --swarm.capabilities code,research \
         --swarm.max_concurrent 5
```

### Specialist

**Responsibilities:**
- Handle domain-specific tasks
- Deep expertise in specific areas
- Targeted capability routing

**Configuration:**
```bash
picoclaw --swarm.role specialist \
         --swarm.capabilities rust,embedded
```

---

## Monitoring

### Dashboard View

```bash
picoclaw --swarm.dashboard
```

Output:
```
╔════════════════════════════════════════════════════════════╗
║           PicoClaw Swarm Status Dashboard                  ║
╚════════════════════════════════════════════════════════════╝

【This Node】
  ID:    claw-a1b2c3d4
  Role:  ⚙️ worker
  Status: ● online
  Load:  [████░░░░░] 40% (2/5)
  Uptime: 2h15m

【Connections】
  NATS:     ✓ Yes nats://localhost:4222
  Temporal: ✓ Yes

【Swarm Statistics】
  Nodes:      5 total, 5 online, 0 offline
  Roles:      1 coordinator(s), 4 worker(s), 0 specialist(s)
  Capacity:   8/25 tasks used

【Discovered Nodes】
  ● claw-coord-01          C online    [██░░░░░░░] 20% (1/5)
  ● claw-a1b2c3d4          W online    [████░░░░░] 40% (2/5)
  ● claw-e5f6g7h8          W online    [████░░░░░] 40% (2/5)
  ● claw-i9j0k1l2          W busy      [█████░░░░] 60% (3/5)
  ● claw-m3n4o5p6          W online    [██░░░░░░░] 20% (1/5)
```

### Logs

```bash
# View swarm logs
picoclaw --swarm.enabled --log.level debug

# Key log patterns:
# - "Node joined swarm" - New node discovered
# - "Task assigned" - Task routed to worker
# - "Task completed" - Worker finished task
# - "Node marked offline" - Node failure detected
```

---

## Troubleshooting

### Nodes Not Discovering Each Other

**Problem:** Workers don't appear in coordinator's node list.

**Solutions:**
1. Check NATS connectivity: `picoclaw --swarm.check-nats`
2. Verify H-id matches (all nodes in same swarm need same H-id)
3. Check firewall rules (NATS port 4222 must be open)

### Tasks Not Being Assigned

**Problem:** Coordinator creates tasks but workers don't receive them.

**Solutions:**
1. Verify worker capabilities match task requirements
2. Check worker load (`--swarm.max_concurrent` limit)
3. Ensure NATS JetStream is enabled (`-js` flag)

### High Memory Usage

**Problem:** Nodes consuming excessive memory.

**Solutions:**
1. Reduce `--swarm.max_concurrent`
2. Decrease NATS subscription buffer size
3. Enable Temporal for long-running workflows (offloads state)

### Temporal Connection Failed

**Problem:** "Temporal connection failed (workflows disabled)" warning.

**Impact:** Tasks still work, but without workflow persistence.

**Solutions:**
1. Start Temporal server: `temporal server start-dev`
2. Or disable Temporal requirement: `--swarm.temporal.address=""`

---

## Security Considerations

### For Development

- Embedded NATS is fine for local testing
- No authentication needed
- Use `--swarm.nats.embedded`

### For Production

1. **Enable NATS Authentication**
   ```bash
   nats-server -js --auth picoclaw_secret
   picoclaw --swarm.nats.credentials /path/to/creds
   ```

2. **Enable TLS**
   ```bash
   nats-server -js -tls
   picoclaw --swarm.nats.urls tls://localhost:4222
   ```

3. **Isolate Swarms by H-id**
   - Different production environments = different H-ids
   - Prevents cross-environment communication

4. **Network Segmentation**
   - Keep NATS ports internal
   - Use VPN for multi-cloud deployments

---

## Performance Tuning

### Small Swarm (2-5 nodes)

```yaml
swarm:
  max_concurrent: 5
  nats:
    heartbeat_interval: 10s
    node_timeout: 30s
```

### Medium Swarm (5-20 nodes)

```yaml
swarm:
  max_concurrent: 10
  nats:
    heartbeat_interval: 5s
    node_timeout: 20s
```

### Large Swarm (20+ nodes)

```yaml
swarm:
  max_concurrent: 20
  nats:
    heartbeat_interval: 3s
    node_timeout: 15s
  temporal:
    enabled: true  # Required for workflow persistence
```

---

## Example: Distributed Code Review Swarm

```bash
# Terminal 1: Coordinator
picoclaw --swarm.role coordinator \
         --swarm.capabilities coordination

# Terminal 2: Code specialist
picoclaw --swarm.role specialist \
         --swarm.capabilities rust,go,python

# Terminal 3: Security specialist
picoclaw --swarm.role specialist \
         --swarm.capabilities security,audit

# Terminal 4: Documentation specialist
picoclaw --swarm.role specialist \
         --swarm.capabilities docs,writing

# Terminal 5: Test specialist
picoclaw --swarm.role specialist \
         --swarm.capabilities testing,qa
```

When you send a code review request:
1. Coordinator decomposes into: code, security, docs, testing reviews
2. Each specialist handles their area in parallel
3. Coordinator synthesizes a comprehensive review

---

## Next Steps

- See [API.md](./API.md) for programmatic usage
- See [CONFIG.md](./CONFIG.md) for all configuration options
- See [EXAMPLES.md](./EXAMPLES.md) for more deployment examples

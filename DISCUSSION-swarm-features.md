# PicoClaw Swarm: What Should a Fleet of Shrimp Actually Do?

> `pkg/swarm/` | Open Discussion | 2026-02-13

---

## The Pitch

So we can now spin up multiple PicoClaws and they find each other over NATS, exchange heartbeats, hand off tasks, and report results. Cool. But right now they're basically just vibing in a group chat — lots of pinging, not much doing.

This discussion is about **what we actually want a swarm of shrimp to pull off**, and in what order we should build it.

---

## P0 — Without These, the Swarm Is Just a Chat Room

### 1. Task Decomposition

`DecomposeTaskActivity` currently returns nil. Every request goes to one shrimp, no matter how big it is. That's like asking one guy to research Rust, Go, *and* Zig embedded dev and write the comparison report — while three other shrimp sit there doing nothing.

The coordinator should be able to look at a request and go: "okay, this breaks into three parallel research jobs and one synthesis job." Three shrimp research simultaneously, a fourth writes the report. Done in a third of the time.

**Open questions:**
- LLM-driven decomposition, rule-based heuristics, or a mix?
- How do we template the decomposition prompt without it getting brittle?
- Max decomposition depth? We don't want recursive splitting all the way to heat death.

### 2. Result Synthesis

`SynthesizeResultsActivity` literally concatenates sub-results with `=== Result 1 ===` headers. It's `fmt.Sprintf` cosplaying as intelligence.

When multiple shrimp contribute partial answers, the coordinator should use the LLM to merge them into one coherent response — not a mechanical paste job.

**Open questions:**
- Keep sub-task metadata in the final output (who ran it, how long it took)?
- If one sub-task failed, do we skip it, flag it, or retry before synthesis?
- Context budget — what happens when sub-results are collectively too long?

### 3. Smarter Capability Routing

Capabilities are currently static strings in config (`capabilities: ["code", "research"]`). That's fine for now, but it's pretty rigid.

Where this should go:
- **Dynamic registration**: shrimp installs a new Skill, its capability list updates automatically.
- **Fuzzy matching**: task needs "code" but shrimp only advertises "golang" and "python" — close enough?
- **Weighted capabilities**: two shrimp both claim "code", but one is clearly better at it. Let the routing reflect that.

---

## P1 — Real Collaboration, Not Just Delegation

### 4. Inter-Shrimp Messaging

Right now shrimp can only talk through task assignments and result callbacks. They can't have a side conversation. If the Rust-research shrimp stumbles on a great Go article, there's no way to toss it to the Go-research shrimp mid-task.

Something like `picoclaw.swarm.chat.{from}.{to}` for point-to-point, maybe a shared topic channel for task-scoped broadcasts.

**Open questions:**
- Trust model — do we blindly trust all shrimp in the same swarm, or add permissions?
- Persist chat history or treat it as ephemeral?

### 5. Shared Context

Each shrimp has its own `MEMORY.md`. Memories are fully isolated. When five shrimp collaborate on one job, they're all working with different context — that's a recipe for contradictory outputs.

We need a task-scoped shared context pool. The coordinator seeds it with background info, workers push intermediate findings to it, everyone reads from it.

Possible backends:
- **NATS KV Store** — lightweight, already in our dependency tree
- **Shared filesystem** — dead simple, doesn't scale
- **Redis** — proven, but adds a dependency (violates the "10MB shrimp" spirit)

### 6. Dynamic Role Switching

Roles are hardcoded at startup (`--role coordinator`). If the coordinator dies, the whole swarm goes headless.

Ideas:
- **Leader election**: if the coordinator disappears, a worker gets promoted. NATS JetStream could handle this, or we go simple with "first to claim wins."
- **Boss does manual labor too**: if all workers are slammed, the coordinator should be able to pick up a task itself.
- **Role fluidity**: a worker could become a specialist on demand, or vice versa, based on what the swarm needs right now.

**Open questions:**
- Election mechanism — JetStream advisory, Raft, or just a NATS-based lock?
- What triggers a role switch? Pure load? Manual? LLM decides?

---

## P2 — Nice to Have

### 7. Priority Queues and Preemption

The `Priority` field on tasks (0=low to 3=critical) is currently decorative. Everything is FIFO. A `priority=3` alert should jump the queue — and maybe interrupt a shrimp that's leisurely composing a blog post.

### 8. DAG Execution

Sub-tasks are all fired in parallel right now with no dependency awareness. We should support:

```
Research Rust ──┐
Research Go   ──┼──> Write Comparison ──> Format & Polish
Research Zig  ──┘
```

The report waits until all three research tasks finish. Format waits for the report. Parallel where possible, sequential where necessary.

### 9. Swarm Dashboard

A live view of the fleet:
- Which shrimp are online, their roles, capabilities
- Per-node load and active tasks
- Task flow in real time
- Historical success rates and latencies

Could be a TUI with `bubbletea` (no browser needed, stays true to the terminal-native ethos) or a minimal web page bolted onto the gateway. Or both.

### 10. Swarm Security

- **NATS TLS + auth**: keep rogue shrimp out
- **Task signing**: make sure tasks actually come from a trusted coordinator
- **Audit trail**: who ran what, when, and what happened

Not urgent for dev, but unavoidable before anyone else runs this.

---

## Design Principles to Keep in Mind

1. **Stay light.** 10MB RAM budget is real. New feature? Sure. New dependency? Think twice.
2. **Degrade gracefully.** No Temporal? Skip workflows. No remote workers? Run locally. Nothing should be a hard requirement except NATS.
3. **Single-shrimp must stay simple.** Swarm code should be invisible when you're running one instance. Zero overhead, zero config needed.
4. **Ship it, then polish it.** Working beats perfect. Get the ugly version running first.

---

## How to Weigh In

Reply with:

```
## Re: [Feature Name]

**Vote**: must-have / nice-to-have / skip / rethink
**Why**: ...
**Alternative idea**: ...
```

Or propose something not on this list. The shrimp are listening.

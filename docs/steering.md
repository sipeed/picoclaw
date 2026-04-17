# Steering

Steering allows injecting messages into an already-running agent loop, interrupting it between tool calls without waiting for the entire cycle to complete.

## How it works

When the agent is executing a sequence of tool calls (e.g. the model requested 3 tools in a single turn), steering checks the queue **after each tool** completes. If it finds queued messages:

1. The remaining tools are **skipped** and receive `"Skipped due to queued user message."` as their result
2. The steering messages are **injected into the conversation context**
3. The model is called again with the updated context, including the user's steering message

```
User ──► Steer("change approach")
                │
Agent Loop      ▼
  ├─ tool[0] ✔  (executed)
  ├─ [polling] → steering found!
  ├─ tool[1] ✘  (skipped)
  ├─ tool[2] ✘  (skipped)
  └─ new LLM turn with steering message
```

## Scoped queues

Steering is now isolated per resolved session scope, not stored in a single
global queue.

- The active turn writes and reads from its own scope key (usually the routed session key such as `agent:<agent_id>:...`)
- `Steer()` still works outside an active turn through a legacy fallback queue
- `Continue()` first dequeues messages for the requested session scope, then falls back to the legacy queue for backwards compatibility

This prevents a message arriving from another chat, DM peer, or routed agent
session from being injected into the wrong conversation.

## Configuration

In `config.json`, under `agents.defaults`:

```json
{
  "agents": {
    "defaults": {
      "steering_mode": "one-at-a-time"
    }
  }
}
```

### Modes

| Value | Behavior |
|-------|----------|
| `"one-at-a-time"` | **(default)** Dequeues only one message per polling cycle. If there are 3 messages in the queue, they are processed one at a time across 3 successive iterations. |
| `"all"` | Drains the entire queue in a single poll. All pending messages are injected into the context together. |

The environment variable `PICOCLAW_AGENTS_DEFAULTS_STEERING_MODE` can be used as an alternative.

## Go API

### Steer — Send a steering message

```go
err := agentLoop.Steer(providers.Message{
    Role:    "user",
    Content: "change direction, focus on X instead",
})
if err != nil {
    // Queue is full (MaxQueueSize=10) or not initialized
}
```

The message is enqueued in a thread-safe manner. Returns an error if the queue is full or not initialized. It will be picked up at the next polling point (after the current tool finishes).

### SteeringMode / SetSteeringMode

```go
// Read the current mode
mode := agentLoop.SteeringMode() // SteeringOneAtATime | SteeringAll

// Change it at runtime
agentLoop.SetSteeringMode(agent.SteeringAll)
```

### Continue — Resume an idle agent

When the agent is idle (it has finished processing and its last message was from the assistant), `Continue` checks if there are steering messages in the queue and uses them to start a new cycle:

```go
response, err := agentLoop.Continue(ctx, sessionKey, channel, chatID)
if err != nil {
    // Error (e.g. "no default agent available")
}
if response == "" {
    // No steering messages in queue, the agent stays idle
}
```

`Continue` internally uses `SkipInitialSteeringPoll: true` to avoid double-dequeuing the same messages (since it already extracted them and passes them directly as input).

`Continue` also resolves the target agent from the provided session key, so
agent-scoped sessions continue on the correct agent instead of always using
the default one.

## Polling points in the loop

Steering is checked at the following points in the agent cycle:

1. **At loop start** — before the first LLM call, to catch messages enqueued during setup
2. **After every tool completes** — including the first and the last. If steering is found and there are remaining tools, they are all skipped immediately
3. **After a direct LLM response** — if a new steering message arrived while the model was generating a non-tool response, the loop continues instead of returning a stale answer
4. **Right before the turn is finalized** — if steering arrived at the very end of the turn, the agent immediately starts a continuation turn instead of leaving the message orphaned in the queue

## Why remaining tools are skipped

When a steering message is detected, all remaining tools in the batch are skipped rather than executed. The alternative — let all tools finish and inject the steering message afterwards — was considered and rejected. Here is why.

### Preventing unwanted side effects

Tools can have **irreversible side effects**. If the user says "no, wait" while the agent is mid-batch, executing the remaining tools means those side effects happen anyway:

| Tool batch | Steering message | With skip | Without skip |
|---|---|---|---|
| `[web_search, send_email]` | "don't send it" | Email **not** sent | Email sent, damage done |
| `[query_db, write_file, spawn_agent]` | "use another database" | Only the query runs | File written + subagent spawned, all wasted |
| `[search₁, search₂, search₃, write_file]` | user changes topic entirely | 1 search | 3 searches + file write, all irrelevant |

### Avoiding wasted time

Tools that take seconds (web fetches, API calls, database queries) would all run to completion before the agent sees the user's correction. In a batch of 3 tools each taking 3-4 seconds, that's 10+ seconds of work that will be discarded.

With skipping, the agent reacts as soon as the current tool finishes — typically within a few seconds instead of waiting for the entire batch.

### The LLM gets full context

Skipped tools receive an explicit error result (`"Skipped due to queued user message."`), so the model knows exactly which actions were not performed. It can then decide whether to re-execute them with the new context, or take a different path entirely.

### Trade-off: sequential execution

Skipping requires tools to run **sequentially** (the previous implementation ran them in parallel). This introduces latency when the LLM requests multiple independent tools in a single turn. In practice, most batches contain 1-2 tools, so the impact is minimal compared to the benefit of being able to stop unwanted actions.

## Skipped tool result format

When steering interrupts a batch, each tool that was not executed receives a `tool` result with:

```
Content: "Skipped due to queued user message."
```

This is saved to the session via `AddFullMessage` and sent to the model, so it is aware that some requested actions were not performed.

## Full flow example

```
1. User: "search for info on X, write a file, and send me a message"

2. LLM responds with 3 tool calls: [web_search, write_file, message]

3. web_search is executed → result saved

4. [polling] → User called Steer("no, search for Y instead")

5. write_file is skipped → "Skipped due to queued user message."
   message is skipped    → "Skipped due to queued user message."

6. Message "search for Y instead" injected into context

7. LLM receives the full updated context and responds accordingly
```

## Automatic bus drain

When the agent loop (`Run()`) starts, it reads inbound messages from a shared message bus. The routing logic determines how each message is handled:

1. **No active turn for the message's session** — the message is dispatched to a **worker goroutine** that processes the full turn (LLM calls, tool execution, steering drain)
2. **An active turn already exists for the same session** — the message is enqueued directly into that session's **steering queue** via `enqueueSteeringMessage`. No background drain goroutine is needed
3. **Non-routable message** (e.g. `system`) — processed synchronously in the main loop

This design enables **parallel processing of messages from different sessions** while keeping same-session messages strictly sequential. Key implications:

- Messages from different users/channels are processed **concurrently** (up to `max_parallel_turns`)
- Messages from the same session are **serialized** — subsequent messages go to the steering queue
- Users don't need to do anything special — their messages are automatically captured as steering when the agent is busy for their session
- Audio messages are transcribed within the worker that processes the turn, so the agent receives text
- `system` inbound messages are processed immediately and do not trigger steering

## Steering with media

Steering messages can include `Media` refs, just like normal inbound user
messages.

- The original `media://` refs are preserved in session history via `AddFullMessage`
- Before the next provider call, steering messages go through the normal media resolution pipeline
- Image refs are converted to data URLs for multimodal providers; non-image refs are resolved the same way as standard inbound media

This applies both to in-turn steering and to idle-session continuation through
`Continue()`.

## Notes

- Steering **does not interrupt** a tool that is currently executing. It waits for the current tool to finish, then checks the queue.
- With `one-at-a-time` mode, if multiple messages are enqueued rapidly, they will be processed one per iteration. This gives the model the opportunity to react to each message individually.
- With `all` mode, all pending messages are combined into a single injection. Useful when you want the agent to receive all the context at once.
- The steering queue has a maximum capacity of 10 messages (`MaxQueueSize`). `Steer()` returns an error when the queue is full. In the bus drain path, the error is logged as a warning and the message is effectively dropped.
- Manual `Steer()` calls made outside an active turn still go to the legacy fallback queue, so older integrations keep working.

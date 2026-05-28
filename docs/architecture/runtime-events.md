# Runtime Events And Event Logging

PicoClaw runtime events are the read-only observation surface for agent, channel, gateway, message bus, and MCP activity. Publishing events and printing logs are separate responsibilities:

- Event publishing: components publish `pkg/events.Event` values to the runtime event bus for hooks, tests, diagnostics, and future UI consumers.
- Event logging: the built-in runtime event logger subscribes to the same bus and prints only the events selected by configuration.

This keeps runtime code focused on publishing events while log policy stays centralized.

## Default Behavior

By default, only `agent.*` events are printed:

```json
{
  "events": {
    "logging": {
      "enabled": true,
      "include": ["agent.*"],
      "min_severity": "info",
      "include_payload": false
    }
  }
}
```

This preserves the previous behavior: agent turn, LLM, tool, steering, subturn, and error events appear in logs. Channel, gateway, bus, and MCP events are still published to the runtime event bus, but they are not printed unless configured.

## Configuration

The configuration lives under `events.logging` in `config.json`:

| Field | Type | Default | Description |
| ----- | ---- | ------- | ----------- |
| `enabled` | bool | `true` | Enables the built-in event logger subscription |
| `include` | string[] | `["agent.*"]` | Event kinds to print; supports exact matches, `*`, and patterns such as `agent.*` |
| `exclude` | string[] | `[]` | Event kinds to suppress after include matching |
| `min_severity` | string | `info` | Minimum severity: `debug`, `info`, `warn`, or `error` |
| `include_payload` | bool | `false` | Adds raw event payloads to log fields |

`include_payload` is disabled by default. Agent events print safe summary fields such as `user_len`, `args_count`, and `content_len` instead of full user messages or tool arguments. Enable raw payload logging only for short-lived diagnostics in a trusted log environment.

## Matching Rules

`include` and `exclude` match the `Event.Kind` string:

```json
{
  "events": {
    "logging": {
      "include": ["gateway.*", "channel.lifecycle.*", "agent.error"],
      "exclude": ["gateway.ready"],
      "min_severity": "info"
    }
  }
}
```

Common patterns:

- `["agent.*"]`: print agent events only.
- `["*"]`: print all runtime events.
- `["gateway.*", "channel.*"]`: print gateway and channel events only.
- `exclude: ["agent.llm.delta"]`: suppress high-volume streaming delta events.
- `min_severity: "warn"`: print warn and error events only.

## Environment Variables

The same settings can be overridden with environment variables:

```bash
PICOCLAW_EVENTS_LOGGING_ENABLED=true
PICOCLAW_EVENTS_LOGGING_INCLUDE="gateway.*,channel.lifecycle.*"
PICOCLAW_EVENTS_LOGGING_EXCLUDE="gateway.ready"
PICOCLAW_EVENTS_LOGGING_MIN_SEVERITY=info
PICOCLAW_EVENTS_LOGGING_INCLUDE_PAYLOAD=false
```

`include` and `exclude` use comma-separated values.

## Event Names And Triggers

The table below lists the current runtime event kinds, when they are emitted, and the most useful event details. `Source`, `Scope`, and `Correlation` are shared envelope fields that may appear on every event. The "Details" column refers to useful payload fields or log summary fields.

### Agent

| Event | Trigger | Details |
| ----- | ------- | ------- |
| `agent.turn.start` | An agent starts processing one user or system input after the turn scope has been created. | `user_len`, `media_count`; scope usually includes `agent_id`, `session_key`, `turn_id`, `channel`, `chat_id`, `message_id` |
| `agent.turn.end` | A turn exits, whether it completed, errored, or was hard-aborted. | `status` (`completed`/`error`/`aborted`), `iterations_total`, `duration_ms`, `final_len` |
| `agent.llm.request` | Before each LLM provider request. | `model`, `messages`, `tools`, `max_tokens` |
| `agent.llm.delta` | Reserved for streaming LLM deltas; the kind is defined, but the current implementation has no natural emit site. | `content_delta_len`, `reasoning_delta_len` |
| `agent.llm.response` | After the LLM provider returns a complete response. | `content_len`, `tool_calls`, `has_reasoning` |
| `agent.llm.retry` | Before retrying an LLM request after context, rate-limit, transient provider, or fallback handling. | `attempt`, `max_retries`, `reason`, `error`, `backoff_ms` |
| `agent.context.compress` | Agent context history is compressed, for example during proactive budget checks or LLM retry handling. | `reason`, `dropped_messages`, `remaining_messages` |
| `agent.session.summarize` | Async session history summarization completes. | `summarized_messages`, `kept_messages`, `summary_len`, `omitted_oversized` |
| `agent.tool.exec_start` | Before the agent executes a tool call. | `tool`, `args_count`; full arguments are not logged by default |
| `agent.tool.exec_end` | After a tool call completes, including successful results, tool errors, and async results. | `tool`, `duration_ms`, `for_llm_len`, `for_user_len`, `is_error`, `async` |
| `agent.tool.exec_skipped` | A tool call is skipped because the tool is unavailable, arguments are invalid, or turn control logic requires skipping it. | `tool`, `reason` |
| `agent.steering.injected` | Queued steering messages are injected into the next LLM context. | `count`, `total_content_len` |
| `agent.follow_up.queued` | An async tool result is queued back into the inbound/follow-up flow. | `source_tool`, `content_len` |
| `agent.interrupt.received` | A turn accepts steering, graceful interrupt, or hard-abort input. | `interrupt_kind`, `role`, `content_len`, `queue_depth`, `hint_len` |
| `agent.subturn.spawn` | A parent turn creates a child turn/subagent. | `child_agent_id`, `label`, `parent_turn_id` |
| `agent.subturn.end` | A child turn ends. | `child_agent_id`, `status` |
| `agent.subturn.result_delivered` | A child turn result is delivered to the target channel/chat. | `target_channel`, `target_chat_id`, `content_len` |
| `agent.subturn.orphan` | A child turn result cannot be delivered or cannot be associated back to its parent turn. | `parent_turn_id`, `child_turn_id`, `reason` |
| `agent.error` | Agent execution reports an error. | `stage`, `error` |

### Channel

| Event | Trigger | Details |
| ----- | ------- | ------- |
| `channel.lifecycle.initialized` | The channel manager creates and registers a channel instance from config. | `type`; scope includes `channel` |
| `channel.lifecycle.started` | Channel `Start()` succeeds and worker goroutines have been started; added channels during hot reload also emit it. | `type` |
| `channel.lifecycle.start_failed` | Channel `Start()` fails. | `type`, `error`; severity is `error` |
| `channel.lifecycle.stopped` | Channel `Stop()` succeeds. | `type` |
| `channel.webhook.registered` | A channel webhook handler is registered on the shared HTTP mux. | `type`; scope includes `channel` |
| `channel.webhook.unregistered` | A channel webhook handler is removed from the shared HTTP mux. | `type`; scope includes `channel` |
| `channel.message.outbound_queued` | An outbound text or media message is queued into its channel worker. | `media`, `content_len`, `reply_to_message_id`; scope comes from the original inbound context |
| `channel.message.outbound_sent` | An outbound text or media message is sent successfully, or a placeholder edit handled the response. | `media`, `content_len`, `message_ids`, `reply_to_message_id` |
| `channel.message.outbound_failed` | An outbound text or media message exhausts retries or hits a permanent failure. | `media`, `content_len`, `retries`, `error`, `reply_to_message_id`; severity is `error` |
| `channel.rate_limited` | A channel worker is waiting for a rate-limit token and the context is canceled, interrupting this delivery. | `media`, `content_len`, `error`, `reply_to_message_id`; severity is `warn` |

### Message Bus

| Event | Trigger | Details |
| ----- | ------- | ------- |
| `bus.publish.failed` | Publishing inbound, outbound, media, audio, or voice-control data fails, or required context is missing. | `stream`, `error`; scope is derived from message context when possible |
| `bus.close.started` | Message bus shutdown begins. | `drained` is usually `0` |
| `bus.close.drained` | Shutdown waits for buffered messages to drain and at least one buffered message was drained. | `drained` |
| `bus.close.completed` | Message bus shutdown completes. | `drained` |

### Gateway

| Event | Trigger | Details |
| ----- | ------- | ------- |
| `gateway.start` | Gateway startup reaches the agent/runtime event bus/bootstrap binding point. | `duration_ms` |
| `gateway.ready` | Gateway services, channel manager, HTTP server, and other core services are ready. | `duration_ms` |
| `gateway.shutdown` | Gateway shutdown begins. | No fixed payload; envelope fields may be the only fields |
| `gateway.reload.started` | Hot reload execution starts. | `duration_ms` |
| `gateway.reload.completed` | Hot reload completes successfully. | `duration_ms` |
| `gateway.reload.failed` | Hot reload fails. | `duration_ms`, `error`; severity is `error` |

### MCP

| Event | Trigger | Details |
| ----- | ------- | ------- |
| `mcp.server.connecting` | The MCP manager is about to connect to a server. | `server`, `type`, `url`, `command` |
| `mcp.server.connected` | An MCP server connects and its tool list has been initialized. | `server`, `type`, `url`, `command`, `tool_count` |
| `mcp.server.failed` | An MCP server connection fails, or the manager is closed before connecting. | `server`, `type`, `url`, `command`, `error`; severity is `error` |
| `mcp.tool.discovered` | A tool from an MCP server is discovered and registered. | `server`, `type`, `url`, `command`, `tool` |
| `mcp.tool.call.start` | The MCP tool wrapper starts a remote tool call. | `server`, `tool`; when emitted inside an agent turn, scope includes turn/chat information |
| `mcp.tool.call.end` | The MCP tool wrapper finishes a remote tool call, including failures. | `server`, `tool`, `duration_ms`, `is_error`, `error` |

## Log Fields

Runtime event logs include stable envelope fields when available:

- `event_id`
- `event_kind`
- `severity`
- `event_time`
- `source_component`
- `source_name`
- `agent_id`
- `session_key`
- `turn_id`
- `channel`
- `account`
- `chat_id`
- `topic_id`
- `space_id`
- `space_type`
- `chat_type`
- `sender_id`
- `message_id`
- `trace_id`
- `parent_turn_id`
- `request_id`
- `reply_to_id`

Agent events add safe payload summaries:

| Event | Summary fields |
| ----- | -------------- |
| `agent.turn.start` | `user_len`, `media_count` |
| `agent.turn.end` | `status`, `iterations_total`, `duration_ms`, `final_len` |
| `agent.llm.request` | `model`, `messages`, `tools`, `max_tokens` |
| `agent.llm.delta` | `content_delta_len`, `reasoning_delta_len` |
| `agent.llm.response` | `content_len`, `tool_calls`, `has_reasoning` |
| `agent.llm.retry` | `attempt`, `max_retries`, `reason`, `error`, `backoff_ms` |
| `agent.context.compress` | `reason`, `dropped_messages`, `remaining_messages` |
| `agent.session.summarize` | `summarized_messages`, `kept_messages`, `summary_len`, `omitted_oversized` |
| `agent.tool.exec_start` | `tool`, `args_count` |
| `agent.tool.exec_end` | `tool`, `duration_ms`, `for_llm_len`, `for_user_len`, `is_error`, `async` |
| `agent.tool.exec_skipped` | `tool`, `reason` |
| `agent.steering.injected` | `count`, `total_content_len` |
| `agent.follow_up.queued` | `source_tool`, `content_len` |
| `agent.interrupt.received` | `interrupt_kind`, `role`, `content_len`, `queue_depth`, `hint_len` |
| `agent.subturn.spawn` | `child_agent_id`, `label` |
| `agent.subturn.end` | `child_agent_id`, `status` |
| `agent.subturn.result_delivered` | `target_channel`, `target_chat_id`, `content_len` |
| `agent.subturn.orphan` | `parent_turn_id`, `child_turn_id`, `reason` |
| `agent.error` | `stage`, `error` |

## Event Domains

Runtime event kinds are defined in `pkg/events/kind.go`. Event logging can select these domains:

- `agent.*`: agent turn, LLM, tool, context, steering, interrupt, subturn, and error events.
- `channel.*`: channel lifecycle, webhook registration, outbound queued/sent/failed, and rate limiting.
- `bus.*`: publish failures and close lifecycle.
- `gateway.*`: start, ready, shutdown, and reload lifecycle.
- `mcp.*`: MCP server connection, tool discovery, and tool call events.

See [`../../config/config.example.json`](../../config/config.example.json) for the default event logging example.

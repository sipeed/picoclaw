# Routing System

> Back to [README](../README.md)

In PicoClaw, the runtime "routing system" is not just one decision.
It is the combined pipeline that decides:

1. which agent handles an inbound message
2. which session dimensions should isolate that conversation
3. whether the turn should use the agent's primary model or a configured light model

This document covers the runtime path in `pkg/routing` and its integration in `pkg/agent`.
It does not describe the launcher's HTTP `ServeMux` routes or the frontend's TanStack Router files under `web/`.

## Routing Layers

| Layer | Files | Responsibility |
| --- | --- | --- |
| Agent dispatch | `pkg/routing/route.go`, `pkg/routing/agent_id.go` | Choose the target agent for the inbound message. |
| Session policy selection | `pkg/routing/route.go` | Decide which dimensions should define session isolation for that routed turn. |
| Model routing | `pkg/routing/router.go`, `pkg/routing/features.go`, `pkg/routing/classifier.go` | Choose between the primary model and a configured light model based on message complexity. |
| Runtime integration | `pkg/agent/registry.go`, `pkg/agent/loop_message.go`, `pkg/agent/loop_turn.go` | Apply the route result, allocate session scope, and select model candidates before provider execution. |

## End-To-End Flow

The normal path for a user message is:

```text
InboundMessage
  -> NormalizeInboundContext
  -> RouteResolver.ResolveRoute(...)
  -> session.AllocateRouteSession(...)
  -> ensureSessionMetadata(...)
  -> Router.SelectModel(...)
  -> provider execution
```

The first half answers "who should handle this message and what session does it belong to".
The second half answers "which model tier should that agent use for this turn".

## Agent Dispatch

`routing.RouteResolver` turns a normalized `bus.InboundContext` into a `ResolvedRoute`:

```go
type ResolvedRoute struct {
    AgentID       string
    Channel       string
    AccountID     string
    SessionPolicy SessionPolicy
    MatchedBy     string
}
```

`MatchedBy` is a debugging aid.
Typical values are:

- `default`
- `dispatch.rule`
- `dispatch.rule:<rule-name>`

## Dispatch Input View

Before matching rules, the resolver builds a normalized `dispatchView`.
Each field is normalized to the exact shape expected by rule matching.

| Selector field | Runtime shape |
| --- | --- |
| `channel` | lowercased channel name |
| `account` | normalized account ID |
| `space` | `<space_type>:<space_id>` |
| `chat` | `<chat_type>:<chat_id>` |
| `topic` | `topic:<topic_id>` |
| `sender` | lowercased canonical sender ID |
| `mentioned` | boolean copied from inbound context |

This means dispatch rules must match the normalized shape, for example:

```json
{
  "agents": {
    "dispatch": {
      "rules": [
        {
          "name": "support-group",
          "agent": "support",
          "when": {
            "channel": "telegram",
            "chat": "group:-100123"
          }
        },
        {
          "name": "slack-mentions",
          "agent": "support",
          "when": {
            "channel": "slack",
            "space": "workspace:t001",
            "mentioned": true
          }
        }
      ]
    }
  }
}
```

## Dispatch Algorithm

`ResolveRoute(...)` follows this sequence:

1. Normalize `channel` and `account`.
2. Clone `session.identity_links` from config.
3. Build the normalized dispatch view.
4. Scan `agents.dispatch.rules` in order.
5. Skip rules with no constraints at all.
6. Return the first rule whose selector fields all match exactly.
7. If no rule matches, fall back to the default agent.

Important consequences:

- first match wins
- there is no score or priority field beyond list order
- invalid target agent IDs fall back to the default agent
- sender matching can see canonical identities produced by `identity_links`

## Default Agent Resolution

If no dispatch rule wins, or if a rule points at an unknown agent, the resolver picks a default agent using this order:

1. the agent marked `default: true`
2. otherwise the first entry in `agents.list`
3. otherwise implicit `main`

Both agent IDs and account IDs are normalized through the helpers in `pkg/routing/agent_id.go`.

## Session Policy Handoff

Agent dispatch does not directly build a session key.
Instead it emits a `SessionPolicy`:

```go
type SessionPolicy struct {
    Dimensions    []string
    IdentityLinks map[string][]string
}
```

The dimensions come from:

- global `session.dimensions`
- or `dispatch_rule.session_dimensions` when the matching rule overrides them

Only these dimension names survive normalization:

- `space`
- `chat`
- `topic`
- `sender`

Invalid or duplicated entries are silently dropped.

`pkg/session/AllocateRouteSession(...)` then turns that policy into:

- a structured `SessionScope`
- a canonical routed session key
- legacy compatibility aliases

So the routing package owns "what should isolate this conversation", while the session package owns "how that isolation becomes keys and durable storage".

## Identity Links

`session.identity_links` is shared between dispatch and session allocation.
That is intentional: a sender canonicalized for routing should also map to the same session identity.

Without that symmetry, the system could route two messages to the same agent but still fragment their history into different sessions.

## Model Routing

The second routing stage decides whether a turn can use a cheaper or faster light model.

Config shape:

```json
{
  "routing": {
    "enabled": true,
    "light_model": "gemini-2.0-flash",
    "threshold": 0.35
  }
}
```

`pkg/routing.Router` compares the current turn against structural features and returns:

- chosen model name
- whether the light model was used
- computed complexity score

If the score is below the threshold, the light model wins.
Otherwise the agent's primary model is used.
At runtime this only matters when the agent actually has light-model candidates configured; otherwise execution stays on the primary candidate set.

## Complexity Features

`ExtractFeatures(...)` computes a language-agnostic feature vector:

| Feature | Meaning |
| --- | --- |
| `TokenEstimate` | Approximate token count; CJK runes count more accurately than a flat rune split. |
| `CodeBlockCount` | Number of fenced code blocks in the current message. |
| `RecentToolCalls` | Tool-call count across the last six history entries. |
| `ConversationDepth` | Total history length. |
| `HasAttachments` | Detects embedded media or common media URL/file extensions. |

This is intentionally structural rather than keyword-based, so the router behaves the same across languages.

## RuleClassifier Scoring

The current classifier is `RuleClassifier`.
It uses a weighted sum capped to `[0, 1]`.

| Signal | Score |
| --- | --- |
| attachments present | `1.00` |
| token estimate `> 200` | `0.35` |
| token estimate `> 50` | `0.15` |
| code block present | `0.40` |
| recent tool calls `> 3` | `0.25` |
| recent tool calls `1..3` | `0.10` |
| conversation depth `> 10` | `0.10` |

The default threshold is `0.35`.
That makes the following behavior intentional:

- trivial chat stays on the light model
- code tasks usually jump to the heavy model immediately
- attachments always force the heavy model
- long, plain-text prompts cross the heavy-model boundary at the default threshold

## Runtime Integration

Agent dispatch and model routing happen in different places:

- `pkg/agent/registry.go` owns `RouteResolver`
- `pkg/agent/loop_message.go` resolves the route and allocates session scope
- `pkg/agent/loop_turn.go:selectCandidates` calls `agent.Router.SelectModel(...)`

When the light model is selected, the agent loop swaps to `agent.LightCandidates`.
When it is not selected, execution stays on the agent's primary provider candidate set.

## Explicit Session Keys

One nuance sits just outside `pkg/routing` but matters for the full routing story.

After a route is allocated, `pkg/agent/loop_utils.go:resolveScopeKey` preserves an explicit incoming session key when the caller already supplied:

- an opaque canonical key
- a legacy `agent:...` key

That makes manual system flows, tests, and compatibility paths deterministic even when the normal routed scope would have produced a different key.

## What This Document Does Not Cover

The repository also contains two unrelated route systems:

- backend HTTP routes registered in `web/backend/api/router.go`
- frontend file routes under `web/frontend/src/routes/`

Those are launcher implementation details.
They are separate from the runtime routing system described here.

## Related Files

- `pkg/routing/route.go`
- `pkg/routing/router.go`
- `pkg/routing/classifier.go`
- `pkg/routing/features.go`
- `pkg/routing/agent_id.go`
- `pkg/session/allocator.go`
- `pkg/agent/registry.go`
- `pkg/agent/loop_message.go`
- `pkg/agent/loop_turn.go`

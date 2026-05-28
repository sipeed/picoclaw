# Dynamic Rate Limiting

PicoClaw prevents 429 errors from LLM provider APIs by enforcing configurable per-model request-rate limits **before** sending each request. Unlike the reactive cooldown/fallback system (which activates *after* a 429 is received), rate limiting is **proactive**: it keeps outbound QPS within the provider's free-tier or plan limits.

## How it works

### Token-bucket algorithm

Each rate-limited model gets a token bucket:

- **Capacity** = `rpm` (burst size equals the per-minute limit)
- **Refill rate** = `rpm / 60` tokens per second
- Tokens are consumed one per LLM call; if the bucket is empty, the call blocks until a token refills or the request context is cancelled

### Call chain integration

```
AgentLoop.callLLM()
  └─ FallbackChain.Execute()         ← iterate candidates
       ├─ CooldownTracker.IsAvailable()   ← skip if post-429 cooldown active
       ├─ RateLimiterRegistry.Wait()      ← NEW: block until token available
       └─ provider.Chat()                 ← actual LLM HTTP call
```

The rate limiter runs **after** the cooldown check and **before** the provider call, so:
- Candidates already in cooldown are skipped entirely (no token consumed)
- Candidates that are available get throttled to the configured RPM

The same check applies in `ExecuteImage`.

### Thread safety

`RateLimiterRegistry` is safe for concurrent use. The per-limiter token bucket uses a fine-grained mutex so concurrent goroutines each acquire their own token independently.

## Configuration

Set `rpm` on any model in `model_list`:

```yaml
model_list:
  - model_name: gpt-4o-free
    provider: openai
    model: gpt-4o
    api_base: https://api.openai.com/v1
    rpm: 3          # max 3 requests per minute
    api_keys:
      - sk-...

  - model_name: claude-haiku
    provider: anthropic
    model: claude-haiku-4-5
    rpm: 60         # 60 rpm (Anthropic free tier)
    api_keys:
      - sk-ant-...

  - model_name: local-llm
    provider: ollama
    model: llama3
    api_base: http://localhost:11434/v1
    # no rpm → unrestricted
```

| Field | Type | Default | Description |
|---|---|---|---|
| `rpm` | `int` | `0` | Requests per minute. `0` means no limit. |

### Interaction with fallbacks

When a model has fallbacks configured, each candidate is rate-limited **independently**:

```yaml
model_list:
  - model_name: gpt4-with-fallback
    provider: openai
    model: gpt-4o
    rpm: 5
    fallbacks:
      - gpt-4o-mini   # must also be in model_list; its own rpm applies
```

If the current candidate's bucket is empty and there are more candidates available, PicoClaw skips the locally saturated candidate and tries the next fallback immediately. Only the last remaining candidate waits for a token to refill. If the context deadline is hit while waiting on that last candidate, the wait error propagates.

For `model_list` aliases that resolve to the same underlying provider/model, rate limiting is keyed by the stable config identity (for example `model_name`) rather than the resolved runtime model string. This preserves distinct RPM settings for multi-key and alias-based configurations.

### Burst behaviour

The bucket starts **full** (burst = RPM). For `rpm: 3`, the first 3 requests fire instantly; subsequent requests are spaced ~20 s apart.

To reduce burstiness for strict APIs, set a lower `rpm` and rely on the steady-state refill.

## Files changed

| File | What |
|---|---|
| `pkg/providers/ratelimiter.go` | `RateLimiter` (token bucket) + `RateLimiterRegistry` |
| `pkg/providers/ratelimiter_test.go` | Unit tests for limiter and registry |
| `pkg/providers/fallback.go` | `FallbackCandidate.RPM` field; `FallbackChain.rl`; `Wait()` call in `Execute`/`ExecuteImage` |
| `pkg/agent/model_resolution.go` | Resolves candidates from `model_list`, preserving stable config identity and propagating `RPM` into `FallbackCandidate` |
| `pkg/agent/loop.go` | Build `RateLimiterRegistry`, register all agents' candidates, pass to `NewFallbackChain` |

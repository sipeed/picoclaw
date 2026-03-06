# DDR: Session Summarization Extraction & Configuration

> Date: 2026-03-06
> Status: In Review
> Packages: `pkg/session`, `pkg/config`, `pkg/agent`

## Problem Frame

Session context grows without bound during long conversations. The original implementation had several issues:

- **Entangled logic**: Summarization was inlined in `pkg/agent/loop.go` — 4 functions (`maybeSummarize`, `forceCompression`, `summarizeSession`, `summarizeBatch`) mixed session I/O, LLM calls, and threshold logic in the agent loop.
- **Race condition**: Async summarization could silently discard messages appended by the user while the LLM was generating a summary.
- **Magic constants**: Token budget (`1024`), temperature (`0.3`), threshold (`20`), percent (`75`) were hardcoded with no user-facing knobs.
- **No injection point**: Tight coupling to `providers.LLMProvider` made unit testing impossible without a live LLM.
- **Flat config**: Two config fields (`summarize_message_threshold`, `summarize_token_percent`) lived as top-level scalars in `AgentDefaults`, not grouped or extensible.

## Decision

### Rule Changes

1. **MUST** extract all summarization orchestration into `pkg/session.SessionManager`.
   - `MaybeSummarize(key)` — threshold check + async dispatch.
   - `ForceCompression(key)` — emergency drop of older half.
   - `ApplySummarization(key, summary, snapshotLen, keepLast)` — atomic merge that compares `snapshotLen` to current message count, preserving messages appended during async work.
   - `summarizeSession(sessionKey string)` — unexported, handles single/multi-part batching.

2. **MUST** define a `Summarizer` interface in `pkg/session`:

   ```go
   type Summarizer interface {
       Summarize(ctx context.Context, messages []providers.Message, existingSummary string) (string, error)
   }
   ```

   A `SummarizeFunc` adapter MUST be provided for testing convenience.

3. **MUST** inject the `Summarizer` at construction via functional options:
   - `WithSummarizer(s Summarizer, cfg config.SummarizationConfig) Option` — generic.
   - `WithLLMSummarizer(provider, model, agentID string, cfg config.SummarizationConfig) Option` — convenience, constructs `LLMSummarizer` internally.
   - When `summarizer` is nil, `MaybeSummarize` is a no-op.

4. **MUST** replace all magic constants with named defaults in `pkg/config`:
   | Constant | Value | Field |
   |---|---|---|
   | `DefaultSummarizeMessageThreshold` | `20` | `MessageThreshold` |
   | `DefaultSummarizeTokenPercent` | `75` | `TokenPercent` |
   | `DefaultKeepLastMessages` | `4` | `KeepLastMessages` |
   | `DefaultContextWindow` | `8192` | `ContextWindow` |
   | `DefaultSummaryMaxTokens` | `1024` | `SummaryMaxTokens` |
   | `DefaultSummaryTemperature` | `0.3` | `SummaryTemperature` |
   | `DefaultMultiPartBatchThreshold` | `10` | `MultiPartBatchThreshold` |
   | `DefaultTimeout` | `120s` | `Timeout` |
   | `DefaultMaxSingleMsgTokenRatio` | `0.5` | `MaxSingleMsgTokenRatio` |
   | `DefaultForceCompressionMinMsgs` | `4` | `ForceCompressionMinMessages` |
   | `DefaultCharsPerToken` | `2.5` | `CharsPerToken` |

5. **MUST** use a single `SummarizationConfig` struct in `pkg/config` as the sole config type for both user-facing JSON fields and internal tuning parameters:

   ```go
   type SummarizationConfig struct {
       // User-facing (documented in config.example.json)
       MessageThreshold   int     `json:"message_threshold,omitempty"`
       TokenPercent       int     `json:"token_percent,omitempty"`
       KeepLastMessages   int     `json:"keep_last_messages,omitempty"`
       ContextWindow      int     `json:"context_window,omitempty"`
       SummaryMaxTokens   int     `json:"summary_max_tokens,omitempty"`
       SummaryTemperature float64 `json:"summary_temperature,omitempty"`

       // Internal (undocumented, tunable for advanced use)
       MultiPartBatchThreshold     int           `json:"multi_part_batch_threshold,omitempty"`
       Timeout                     time.Duration `json:"timeout,omitempty"`
       MaxSingleMsgTokenRatio      float64       `json:"max_single_msg_token_ratio,omitempty"`
       ForceCompressionMinMessages int           `json:"force_compression_min_messages,omitempty"`
       CharsPerToken               float64       `json:"chars_per_token,omitempty"`
   }
   ```

   Nested under `agents.defaults.summarization` in JSON config. The session package imports and uses this directly — no separate config struct.

6. **MUST** preserve backward compatibility: `AgentDefaults.GetSummarization()` merges the new `Summarization *SummarizationConfig` with legacy flat fields `SummarizeMessageThreshold` / `SummarizeTokenPercent`. New struct takes priority; legacy fields apply only when the corresponding new field is zero.

7. **SHOULD** deprecate `summarize_message_threshold` and `summarize_token_percent` in `AgentDefaults`. These fields MUST remain parseable with `omitempty` but SHOULD NOT appear in new documentation or `config.example.json`.

8. **MAY** omit all `SummarizationConfig` fields — zero values are replaced by sensible defaults via `SummarizationConfig.WithDefaults()`.

### Definitions

- **Snapshot length**: The message count captured before dispatching async summarization. Used by `ApplySummarization` to detect concurrent writes.
- **Multi-part batching**: When the message batch exceeds `MultiPartBatchThreshold`, it is split in half; each half is summarized in parallel via goroutines; results are merged into a single summary.
- **Force compression**: Emergency fallback that drops the older half of messages and prepends a `[system]` note. No LLM call required.

### Migration

Old config (deprecated):

```json
{
  "agents": {
    "defaults": {
      "summarize_message_threshold": 30,
      "summarize_token_percent": 80
    }
  }
}
```

New config (preferred):

```json
{
  "agents": {
    "defaults": {
      "summarization": {
        "message_threshold": 30,
        "token_percent": 80
      }
    }
  }
}
```

Both forms are supported. If both are present, the nested struct wins.

## Rationale

### Core Reasons

- **Separation of concerns**: Agent loop should orchestrate turn logic, not session housekeeping. Summarization is a session-level concern.
- **Testability**: Interface injection enables deterministic unit tests with `SummarizeFunc` stubs — no LLM, no network, no flakes.
- **Race safety**: `ApplySummarization` compares snapshot length to current state, so messages arriving during async summarization are never lost.
- **Configurability**: Users on constrained hardware (RISC-V, 10MB RAM) need different thresholds than users on GPT-4 with 128k context.
- **Extensibility**: New summarization strategies (e.g., embeddings-based, local model) plug in via the `Summarizer` interface.

5. **Unified config** — Single `config.SummarizationConfig` used everywhere. No duplicate struct in session package, no field-by-field mapping.

- **Config hygiene**: Grouping related fields under `summarization` prevents `AgentDefaults` from growing into a flat bag of 30+ fields.

### Alternatives Considered

1. **Do nothing** — Keep inlined in `loop.go`.
   - Rejected: Untestable without LLM. Race condition on concurrent writes. Magic constants not user-configurable.

2. **Separate `SessionSummarizer` struct** — Standalone struct with its own state.
   - Rejected: Duplicates session access logic. Requires either passing sessions in/out or giving the struct direct storage access — both leak responsibilities.

3. **Mutable closure via `ConfigureSummarizer(fn)`** — Post-construction injection.
   - Rejected: Mutable after init violates immutability principle. Anonymous closures are harder to test and document.

4. **Config as flat fields** — Add `keep_last_messages`, `summary_max_tokens`, etc. directly to `AgentDefaults`.
   - Rejected: Would add 6 more top-level fields. Not cohesive. Harder to feature-flag or version.

5. **Separate config struct per package** — `session.SummarizerConfig` with its own defaults, mapped field-by-field from `config.SummarizationConfig` in `instance.go`.
   - Rejected: Duplicate types for the same data. Tedious mapping code. Easy to drift.

### Trade-offs

| Dimension   | Cost                                  | Benefit                                          |
| ----------- | ------------------------------------- | ------------------------------------------------ |
| Complexity  | New interface + options pattern       | Clean test seam, swappable strategies            |
| Migration   | Must support two config shapes        | Zero breaking changes for existing users         |
| Performance | One extra goroutine per summarization | Non-blocking — user doesn't wait for LLM summary |
| Binary size | ~0 (no new deps)                      | N/A                                              |

### Risks & Mitigations

| Risk                                                   | Mitigation                                                                                            |
| ------------------------------------------------------ | ----------------------------------------------------------------------------------------------------- |
| Snapshot stale (messages deleted during summarization) | `ApplySummarization` returns early if `snapshotLen > currentLen`                                      |
| Duplicate in-flight summarizations                     | `sync.Map` inflight guard — second call is a no-op                                                    |
| LLM summarization fails                                | Logged and skipped; session continues unsummarized. `ForceCompression` as fallback at token overflow. |
| Legacy config silently ignored                         | `GetSummarization()` merges both; tested in `config_test.go`                                          |

## Consequences

### Immediate Impacts

| What                         | Change                                                                                                             |
| ---------------------------- | ------------------------------------------------------------------------------------------------------------------ |
| `pkg/agent/loop.go`          | Calls `agent.Sessions.MaybeSummarize(key)` / `ForceCompression(key)` — no direct summarization logic               |
| `pkg/agent/instance.go`      | Wires `WithLLMSummarizer(provider, model, agentID, cfg)` at construction                                           |
| `pkg/session/manager.go`     | Gains `summarizer`, `summarizerCfg` (`config.SummarizationConfig`), `inflight` fields + 4 new methods              |
| `pkg/session/summarizer.go`  | New file: interface, `SummarizeFunc` adapter, `LLMSummarizer`, prompt builder (no config types or constants)       |
| `pkg/config/config.go`       | `SummarizationConfig` struct (all fields + `WithDefaults()`), `GetSummarization()` method, named default constants |
| `config/config.example.json` | Uses nested `summarization` block                                                                                  |
| README config examples       | Show `summarization` in quickstart snippets                                                                        |

### Follow-up Tasks

| Task                                                                               | Owner (role) | Priority                 |
| ---------------------------------------------------------------------------------- | ------------ | ------------------------ |
| Remove deprecated `summarize_message_threshold` / `summarize_token_percent` fields | Maintainer   | Low — next major version |
| Add env var override for `summarization.*` fields                                  | Maintainer   | Medium                   |
| Evaluate embeddings-based summarization via `Summarizer` interface                 | Contributor  | Low                      |

### Test / Verification Plan

| Acceptance Criterion                         | Test                                                 |
| -------------------------------------------- | ---------------------------------------------------- |
| Summarization triggers above threshold       | `TestMaybeSummarize_AboveMessageThreshold`           |
| No-op below threshold                        | `TestMaybeSummarize_BelowThreshold`                  |
| Dedup prevents concurrent summarizations     | `TestMaybeSummarize_DeduplicatesConcurrent`          |
| Messages appended during async are preserved | `TestApplySummarization_PreservesNewMessages`        |
| Stale snapshot rejected                      | `TestApplySummarization_StaleSnapshot`               |
| Force compression drops older half           | `TestForceCompression_DropsOldestHalf`               |
| Multi-part batching for large sessions       | `TestSummarizeSession_MultiPart`                     |
| Concurrent read/write safety                 | `TestConcurrent_SummarizeAndWrite`                   |
| Token estimation (Latin + CJK)               | `TestEstimateTokens_Latin`, `TestEstimateTokens_CJK` |
| Legacy config fallback works                 | `TestGetSummarization_LegacyFallback`                |
| New config overrides legacy                  | `TestGetSummarization_NewOverridesLegacy`            |
| Default config has correct thresholds        | `TestDefaultConfig_SummarizationThresholds`          |

All tests pass as of 2026-03-06 (1455 passed, 0 failed).

### Rollback Plan

Revert the commits that introduce `pkg/session/summarizer.go`, the `Summarizer` interface, and the config changes. The legacy flat fields remain functional — no data migration needed.

### Review Date

2026-06-06 (3 months) — evaluate whether legacy flat config fields can be removed.

# PicoClaw Context Window Improvements — Design Document

## Date
2025-01-09

## Scope

Implement 7 context-window improvements to prevent "Context window exceeded" errors, based on research comparing PicoClaw's current approach with OpenCode's proven conservative strategy.

## Non-Goals
- No changes to provider adapters (Anthropic, OpenAI, etc.)
- No changes to the session store (JSONL) format
- No changes to the seahorse database schema
- No changes to tool execution behavior

## Architecture

### Improvement 1: Fix Token Estimation
- **File**: `pkg/tokenizer/estimator.go`
- **Change**: Remove `CharsPerToken` config (already at 4.0 chars/token). Add `CharsPerToken` config struct to `pkg/tokenizer` for future tuning.
- **Status**: Already partially implemented (4.0 chars/token). Need to add Config struct.
- **Risk**: Very low. The 4.0 heuristic is already the default.

### Improvement 2: Add ContextOverflowError Type
- **File**: Create `pkg/agent/errors.go`
- **Change**: Add structured `ContextOverflowError` with Model, ContextWindow, RequestedTokens, Reason fields. Update `pipeline_llm.go` to wrap/recognize this error.
- **Risk**: Low. New file; minimal existing dependency changes.

### Improvement 3: Token-Aware Fresh Tail Protection
- **File**: `pkg/seahorse/short_constants.go`, `pkg/seahorse/short_assembler.go`
- **Change**: Replace fixed `FreshTailCount` (32 messages) with token-budget-aware `CalculateFreshTailBudget(contextWindow)` (OpenCode's proven approach: 25% of usable context, bounded by 2000–8000 tokens, with minimum 2 turns).
- **Risk**: Medium. Changes core assembler logic. All seahorse tests will need updates for the new budget calculation.

### Improvement 4: Structured Summary Template
- **File**: `pkg/seahorse/short_compaction.go`
- **Change**: Add `SummaryTemplate` constant and `UseStructuredSummaries` toggle. Apply template in `generateLeafSummary` and `generateCondensedSummary`.
- **Risk**: Low. Template is additive; old behavior preserved as fallback.

### Improvement 5: Tool Output Pruning
- **File**: Create `pkg/utils/tool_pruner.go`
- **Change**: Add `PruneToolOutputs(messages, protectedTurns, pruneThreshold)` that walks backward and replaces old tool outputs with stubs.
- **Risk**: Medium. New utility; needs careful edge-case handling (orphaned tool results).

### Improvement 6: Proactive vs Reactive Compaction Separation
- **File**: `pkg/agent/context_manager.go`, `pkg/agent/pipeline_setup.go`, `pkg/agent/pipeline_llm.go`
- **Change**: Add `CompactReason` type with `proactive`, `retry`, `overflow` constants. Rename `ContextCompressReasonProactive` and `ContextCompressReasonRetry` to use the new type. Update all references.
- **Risk**: Medium. Refactors enum type used across pipeline.

### Improvement 7: Tool Output Truncation with Disk Saving
- **File**: Create `pkg/utils/truncate.go`
- **Change**: Add `TruncatedResult` struct and `TruncateToolOutput` function. Saves full output to disk, returns preview + hint.
- **Risk**: Low. New utility; no existing callers.

## Testing Strategy
- Unit tests for new utilities (`pkg/utils/tool_pruner_test.go`, `pkg/utils/truncate_test.go`)
- Update `pkg/seahorse/short_assembler_test.go` for token-budget fresh tail
- Update `pkg/agent/pipeline_llm_test.go` (or create one) for `ContextOverflowError` handling
- Ensure `make test` passes

## Rollout Notes
- The `FreshTailCount` constant in `short_constants.go` will be deprecated but kept for backward compatibility (exported but unused).
- `ContextCompressReasonProactive` and `ContextCompressReasonRetry` constants will be kept as deprecated aliases.

## Failure Mode Analysis
1. **Fresh tail budget too large**: Could reduce usable context. Mitigated by MinPreserveRecentTokens bound and 25% ratio.
2. **Structured summary too verbose**: Could increase token count instead of saving. Mitigated by keeping old format as fallback.
3. **Tool pruning breaks tool-call chains**: Could orphan tool results. Mitigated by protectedTurns parameter and turn-boundary safety.

## Approved
Proceed to implementation.

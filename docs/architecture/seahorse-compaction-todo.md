# Seahorse Compaction Architecture TODO

Goal: keep interactive turns latency-bounded while preserving Seahorse as the
long-term memory quality layer.

## Target Architecture

1. Fast response path
   - Assemble context.
   - If it fits, call the LLM immediately.
   - If it does not fit, schedule background compaction and deterministically
     trim assembled history until it fits.
   - Never run a long series of LLM summarization calls before the first user
     response.

2. Background compaction path
   - After final delivery and whenever proactive pressure is detected, enqueue
     one compaction job per session.
   - Jobs run with a deadline and per-session dedupe.
   - This path improves future turns; it is not required for the current turn
     to answer.

3. Emergency retry path
   - If the provider still returns a context overflow, run bounded aggressive
     compaction and retry.
   - This is the only synchronous `CompactUntilUnder` path.
   - If retry still cannot fit, fail closed instead of looping or hanging.

4. Observability
   - Log reason, budget, duration, and dedupe skips.
   - Keep enough signal to diagnose long compactions without surfacing raw
     tool feedback to user chats.

## Implementation Checklist

- [x] Add effective history budget that reserves system/tool/response tokens.
- [x] Cap protected Seahorse fresh-tail tokens.
- [x] Keep proactive compaction off the aggressive `CompactUntilUnder` path.
- [x] Make proactive setup fully non-blocking by scheduling background compact
      and using deterministic trim for the active turn.
- [x] Reuse one background compaction scheduler for post-delivery and
      proactive pressure.
- [x] Add timeout around emergency aggressive retry compaction.
- [x] Add tests proving proactive setup does not wait for `Compact`.
- [ ] Add tests proving retry still uses bounded aggressive compaction.
- [ ] Add runtime metrics/events for compaction duration and skips.

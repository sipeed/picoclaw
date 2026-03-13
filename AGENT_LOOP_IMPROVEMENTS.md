# Agent Loop Improvements Plan

This document outlines a series of tasks to improve the main loop of the agentic stack (`AgentLoop` in `pkg/agent/loop.go`), focusing on Performance, Reliability, Architecture, and Features.

## Phase 1: Architecture & Maintainability (Refactoring)
- [x] **Extract LLM Call & Retry Logic:** Refactor `runLLMIteration` by moving the LLM calling, fallback chain, and context window retry logic into a dedicated method (e.g., `executeLLMWithRetry`).
- [x] **Extract Tool Execution Logic:** Refactor `runLLMIteration` by moving the parallel tool execution (`sync.WaitGroup`), channel routing, and async callbacks into a dedicated method (e.g., `executeToolBatch`).
- [x] **State Machine / Explicit Flow:** Clean up the main loop logic to reduce nested `if/for` blocks and make the transition between generating, executing tools, and compressing context more explicit.

## Phase 2: Reliability & Error Handling
- [x] **Graceful Recovery on Tool Panic:** Add `defer recover()` inside the parallel tool execution goroutines to prevent a panicked tool from crashing the entire `AgentLoop`. Return the panic as an error string to the LLM.
- [x] **Exponential Backoff:** Replace the linear backoff in LLM retries (`time.Duration(retry+1) * 5 * time.Second`) with exponential backoff and jitter to better handle rate limits.
- [ ] **Granular Error Classification:** Update `LLMProvider` interfaces to return structured, typed errors (e.g., `providers.ErrContextLengthExceeded`) instead of relying on fragile string matching.

## Phase 3: Performance & Latency
- [x] **Concurrent Message Processing:** Evaluate introducing a worker pool or goroutines to process independent user requests concurrently without blocking the entire agent instance.
- [ ] **Background Summarization:** Offload `maybeSummarize` and context compression to a background worker to unblock the main thread and respond to the user faster.
- [ ] **Streaming Responses:** Implement streaming LLM token generation directly to the `bus.PublishOutbound` instead of waiting for full generation.

## Phase 4: Features
- [ ] **Human-in-the-Loop:** Introduce an approval prompt state for high-risk tools (e.g., SQL execution) that pauses the loop until a user explicitly replies "Yes/No".
- [ ] **Background / Long-Running Tasks:** Implement tools that allow the LLM to run slow operations in the background, releasing the main loop and notifying the user asynchronously upon completion.
- [ ] **Multi-Agent Orchestration:** Create a Supervisor Loop where an agent can delegate tasks to other `AgentInstance`s and synthesize their results.

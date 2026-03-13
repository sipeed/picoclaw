# Issue #783 Investigation and Fix Implementation Document

## 1. Problem Clarification (Confirmed)

- Phenomenon: When `agents.*.model.primary/fallbacks` use `model_name` aliases (e.g., `step-3.5-flash`), the fallback chain parses the alias as a real `provider/model`, resulting in a potentially empty `provider` and an incorrect `model`.
- Root Cause: `ResolveCandidates` only performs `ParseModelRef` on strings without first mapping aliases to real `model` fields via `model_list`.
- Impact:
  - Fallback execution might send the alias directly to an OpenAI-compatible provider, triggering an `Unknown Model` error.
  - When `defaults.provider` is empty, the logs show an empty `provider=` value.

## 2. Objectives

- Fix fallback candidate resolution: Prioritize resolving aliases through `model_list`.
- Maintain backward compatibility: If no match is found in `model_list`, fall back to the existing `ParseModelRef` logic.
- Add supplementary tests: Cover aliases, nested path models (e.g., `openrouter/stepfun/...`), and empty default providers.
- Verify code style: Ensure consistency with the current repository's style (naming, error handling, test structure).

## 3. Online Best Practices Research Conclusions (Completed)

- [x] Researched recommended handling of the `model` field by OpenAI-compatible gateways (e.g., OpenRouter).
- [x] Researched best practices for multi-provider/fallback design (candidate resolution, log observability).
- [x] Mapped external recommendations to actionable constraints for this repository.

External Reference Key Points (from official documentation of OpenRouter/LiteLLM/Cloudflare AI Gateway, etc.):

- Prefer explicit configuration; do not rely on string splitting to infer the provider.
- Model identifiers for gateways should retain full path semantics to avoid truncation causing "Unknown Model" errors.
- Fallback and primary paths should share the same resolution strategy to avoid cases where the main path works but the fallback path fails.

Reference Links:

- OpenRouter Provider Routing: https://openrouter.ai/docs/guides/routing/provider-selection
- OpenRouter Model Fallbacks: https://openrouter.ai/docs/guides/routing/model-fallbacks
- OpenRouter Chat Completion API: https://openrouter.ai/docs/api-reference/chat-completion
- LiteLLM Router Architecture: https://docs.litellm.ai/docs/router_architecture
- Cloudflare AI Gateway Chat Completion: https://developers.cloudflare.com/ai-gateway/usage/chat-completion/

Actionable Constraints for This Repository:

- Map `model_name -> model_list.model` during the fallback candidate construction phase.
- Retain legacy parsing behavior when no mapping is found to ensure compatibility.
- Use new tests to lock down scenarios involving "aliases + nested model paths + empty default providers."

## 4. Implementation Steps (Executed Sequentially)

- [x] Step 1: Align with existing code patterns and locate minimal change points (`pkg/agent` + `pkg/providers`).
- [x] Step 2: Implement "model_list-based fallback candidate resolution."
- [x] Step 3: Add/update unit tests to cover the issue scenarios.
- [x] Step 4: Perform code style consistency review (against existing file styles).
- [x] Step 5: Run quality gates (LSP + `make check`).

## 5. Execution Record

- Status: Completed
- Changes implemented:
  - `pkg/providers/fallback.go`: Added `ResolveCandidatesWithLookup` while keeping `ResolveCandidates` for backward compatibility.
  - `pkg/agent/instance.go`: Prioritized resolving aliases via `model_list` before building fallback candidates; complemented models without protocols with the default `openai/` prefix before parsing.
  - `pkg/providers/fallback_test.go`: Added tests for alias resolution and deduplication.
  - `pkg/agent/instance_test.go`: Added tests for agent-side alias resolution to nested model paths and models without protocols.
- Style alignment check (Completed): Consistent with existing patterns in `pkg/providers/fallback_test.go` and `pkg/providers/model_ref_test.go`.
- Quality verification (Completed): All checks passed after running `make generate` followed by `make check`.

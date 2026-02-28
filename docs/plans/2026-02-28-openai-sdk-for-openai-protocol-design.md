# OpenAI SDK for OpenAI Protocol Design

## Background

The project currently uses:

- `codex_provider` with `github.com/openai/openai-go/v3` for Codex-specific backend.
- `openai_compat` with manual HTTP JSON for OpenAI-compatible multi-provider support.

`openai_compat` is intentionally broad and handles provider dialect differences. For protocol `openai`, we can use the official SDK with lower integration risk if we isolate it to OpenAI-only path.

## Decision

Adopt hybrid routing:

1. `openai` protocol (API key path) uses a new SDK-backed provider.
2. Other OpenAI-compatible protocols continue using existing `openai_compat` HTTP provider.
3. `openai` oauth/token paths remain on existing Codex provider path.

## Goals

- Improve OpenAI protocol correctness/maintainability via official SDK.
- Avoid destabilizing non-OpenAI compatible providers.
- Preserve existing external provider interface and agent loop behavior.

## Non-Goals

- Full migration of all OpenAI-compatible providers to SDK.
- Removing `openai_compat`.
- Mapping non-existent SDK fields into `ReasoningContent`, `ReasoningDetails`, `ThoughtSignature` for OpenAI protocol.

## Architecture

### New Provider

Create `pkg/providers/openai_sdk/provider.go` implementing `providers.LLMProvider`.

Core construction inputs:

- `apiKey`
- `apiBase`
- `proxy`
- `requestTimeout`
- `maxTokensField`

Implementation uses:

- `openai.NewClient(...)`
- `option.WithBaseURL(...)`
- `option.WithAPIKey(...)`
- `option.WithHTTPClient(...)`

### Factory Routing

Update `CreateProviderFromConfig`:

- `protocol == openai` + API key path => `OpenAISDKProvider`
- `protocol == openai` + oauth/token => existing codex auth provider (unchanged)
- all other openai-compatible protocols => existing `HTTPProvider` (`openai_compat`)

## Data Mapping

### Request Mapping

`providers.Message` -> `openai.ChatCompletionMessageParamUnion`:

- `system`, `user`, `assistant`, `tool` roles
- assistant tool calls mapped where needed

`providers.ToolDefinition` -> SDK function tools list.

Options mapping:

- `max_tokens` -> `MaxTokens` or `MaxCompletionTokens` (respect `maxTokensField`)
- `temperature` -> `Temperature`
- `prompt_cache_key` -> `PromptCacheKey` (OpenAI path only)

### Response Mapping

From first choice:

- `Content`
- `ToolCalls`
- `FinishReason`
- `Usage`

OpenAI SDK path intentionally does not map:

- `ReasoningContent`
- `ReasoningDetails`
- `ThoughtSignature`

These fields remain available for dialect providers on `openai_compat` path.

## Error Handling

- Surface SDK errors with status/type/code where available.
- Preserve current provider error semantics as much as possible (human-readable failure context).
- Keep timeout/proxy failures actionable.

## Testing Strategy

### Unit Tests (`openai_sdk/provider_test.go`)

- Basic content response parsing.
- Tool call response parsing.
- Max token field routing (`max_tokens` vs `max_completion_tokens`).
- Prompt cache key inclusion.
- Timeout behavior.
- Proxy behavior.

### Factory Tests

- OpenAI API-key config returns `*OpenAISDKProvider`.
- OpenAI oauth/token continues existing path.
- Non-openai protocols still return `*HTTPProvider`.

### Regression

- Existing `openai_compat` tests remain green.
- Full test suite passes (`go test ./...`).

## Risks and Mitigations

1. Behavior drift between SDK and HTTP paths.
   - Mitigation: focused parity tests for fields/options used by agent loop.

2. Incomplete message/tool mapping edge cases.
   - Mitigation: explicit role/tool test matrix and conservative fallback behavior.

3. Future duplication between SDK and HTTP logic.
   - Mitigation: keep SDK path narrow (OpenAI-only) and avoid over-abstracting in this iteration.

## Rollout

1. Introduce new provider with tests.
2. Route `openai` protocol API-key path in factory.
3. Run provider package and full suite tests.
4. Keep capability profile/override logic in `openai_compat` for non-openai protocols.

## Acceptance Criteria

- OpenAI protocol (API key path) no longer uses `openai_compat`.
- Non-openai protocols continue working on `openai_compat` unchanged.
- No regression in tool call flow and usage accounting.
- Test suite passes.

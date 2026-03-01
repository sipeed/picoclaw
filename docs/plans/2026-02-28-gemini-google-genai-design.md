# Gemini/Google via GenAI SDK Design

## Background

Current Gemini/Google traffic goes through `openai_compat` HTTP calls. This creates two issues:

1. Gemini-specific behavior is coupled to OpenAI-compatible wire format branches.
2. Gemini thought-signature handling depends on dialect-specific JSON mapping in HTTP code.

We already accepted a hybrid provider strategy in this branch:

- OpenAI protocol uses official OpenAI SDK.
- Other protocols stay on their existing adapter.

This document applies the same strategy to Gemini-family protocols:

- Route `gemini/*` and `google/*` to `google.golang.org/genai`.
- Keep `antigravity/*` unchanged.

## Goals

- Move Gemini/Google transport to official `genai` SDK.
- Keep `providers.LLMProvider.Chat` output contract unchanged.
- Preserve thought-signature set/use compatibility across runtime and session history.
- Minimize regression risk by isolating routing changes.

## Non-Goals

- Migrating `antigravity/*` to `genai`.
- Refactoring all provider-selection legacy paths in one pass.
- Changing agent/session schema.

## Why `antigravity/*` Stays Separate

`antigravity` uses Cloud Code Assist private endpoints (`cloudcode-pa.googleapis.com/v1internal:*`) with custom envelope fields (`project`, `requestType`, `requestId`, etc.). This is not the normal Gemini API surface used by `genai`.

Conclusion: `antigravity/*` remains on its dedicated provider.

## Candidate Approaches

### Option 1 (Recommended): New `gemini_sdk` provider, protocol routing split

- Add `pkg/providers/gemini_sdk`.
- Route `gemini` and `google` protocols to this provider.
- Keep `openai_compat` for other OpenAI-compatible protocols.
- Remove only Gemini-specific request-branch logic from `openai_compat`.

Pros:
- Clear boundaries and low coupling.
- Easy to test and roll back.
- Matches existing OpenAI SDK migration pattern.

Cons:
- One extra provider package.

### Option 2: Keep `openai_compat`, internally branch into `genai`

Pros:
- Fewer top-level provider packages.

Cons:
- `openai_compat` grows in complexity and mixed responsibilities.
- Harder long-term maintenance.

### Option 3: One-shot migrate `gemini/google/antigravity`

Pros:
- Superficial unification.

Cons:
- Highest risk.
- `antigravity` protocol mismatch makes this brittle.

## Decision

Adopt Option 1.

## Target Architecture

### New Provider

Create `pkg/providers/gemini_sdk/provider.go` implementing `providers.LLMProvider` with `genai.Client`.

Provider constructor inputs:

- `apiKey`
- `apiBase` (optional override)
- `proxy`
- `requestTimeout`

### Factory Routing (`model_list` path)

In `CreateProviderFromConfig`:

- `case "gemini", "google"` => `gemini_sdk.NewProvider(...)`
- `case "antigravity"` => unchanged.
- Other protocol routing unchanged.

## Request Mapping

`providers.Message` -> `[]*genai.Content` + `GenerateContentConfig`:

- `system` -> `config.SystemInstruction`
- `user` text -> user text part
- `assistant` text -> model text part
- `assistant` tool calls -> model `FunctionCall` parts
- `tool`/tool result -> user `FunctionResponse` parts

Tool definitions:

- map to `Tool.FunctionDeclarations`.

Options:

- `max_tokens` -> `MaxOutputTokens`
- `temperature` -> `Temperature`
- `prompt_cache_key` ignored for Gemini (consistent with current behavior)

## Response Mapping

From `GenerateContentResponse` first candidate:

- `LLMResponse.Content` <- `resp.Text()`
- `LLMResponse.ToolCalls` <- `part.FunctionCall` entries
- `LLMResponse.Usage` <- `UsageMetadata` counts
- `LLMResponse.FinishReason` mapping:
  - tool calls present -> `tool_calls`
  - `MAX_TOKENS` -> `length`
  - else -> `stop`

## ThoughtSignature Compatibility (Set + Use)

### Source field in SDK

- `genai.Part.ThoughtSignature` (`[]byte`) alongside `Part.FunctionCall`.

### Write path (set)

When parsing response function calls:

- Store signature into `ToolCall.ExtraContent.Google.ThoughtSignature`.
- Mirror same value into `ToolCall.Function.ThoughtSignature` for backward compatibility.

### Read path (use)

When rebuilding assistant tool-call history for the next SDK request:

1. Read `ToolCall.ExtraContent.Google.ThoughtSignature` (preferred).
2. Fallback to `ToolCall.Function.ThoughtSignature`.
3. If missing/invalid, continue without signature.

This guarantees compatibility across mixed old/new session data.

## Session Serialization/Deserialization Compatibility

Session persistence serializes `providers.Message` as JSON.

Key compatibility facts:

- `ToolCall.ThoughtSignature` is non-serialized (`json:"-"`).
- Serialized fields are `Function.ThoughtSignature` and `ExtraContent.Google.ThoughtSignature`.

Compatibility strategy:

- New code writes both serialized fields.
- New code reads both (priority: `extra_content` then `function`).
- Old session files (function-only) remain usable.
- New session files remain usable by old readers through mirrored function field.

## `openai_compat` Cleanup After Migration

### Safe to remove in Phase A

- Gemini host-based `prompt_cache_key` request suppression.
- Gemini/Google request-side model-prefix special casing tied to Gemini routing.

### Keep in Phase A

- Generic response parsing support for `extra_content.google.thought_signature`.

Rationale: this may still appear from non-gemini OpenAI-compatible gateways.

## Testing Plan (Design-Level)

1. Provider routing tests:
   - `gemini/*` -> `*gemini_sdk.Provider`
   - `google/*` -> `*gemini_sdk.Provider`
   - `antigravity/*` unchanged
2. Mapping tests:
   - message roles, tool declarations, max tokens, temperature
3. Thought-signature tests:
   - response signature -> extra_content + function mirror
   - history rebuild prefers extra_content, falls back to function
4. Session compatibility tests:
   - old session payload replays correctly
   - new payload round-trips via JSON
5. Regression:
   - providers package tests
   - full `go test ./...`

## Acceptance Criteria

- Gemini/Google protocols use `genai` SDK provider.
- `Provider.Chat` external behavior remains stable.
- ThoughtSignature set/use is compatible across old/new sessions.
- `antigravity` behavior unchanged.
- Full test suite passes.

# OpenAI SDK for OpenAI Protocol Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Route `openai` protocol (API-key path) to a new SDK-backed provider using `openai-go/v3`, while keeping all other OpenAI-compatible protocols on existing `openai_compat` HTTP provider.

**Architecture:** Add a dedicated `openai_sdk` provider implementing `LLMProvider`, map shared message/tool/options into Chat Completions params, and update factory routing to select SDK only for OpenAI API-key path. Preserve codex oauth/token behavior and leave non-openai protocols untouched.

**Tech Stack:** Go 1.25+, `github.com/openai/openai-go/v3`, `httptest`, table-driven tests.

---

Use @superpowers/test-driven-development for each task and @superpowers/verification-before-completion before final handoff.

### Task 1: Scaffold `openai_sdk` Provider With Minimal Failing Test

**Files:**
- Create: `pkg/providers/openai_sdk/provider.go`
- Create: `pkg/providers/openai_sdk/provider_test.go`

**Step 1: Write failing test for basic text response parsing**

```go
func TestOpenAISDKProvider_Chat_BasicContent(t *testing.T) {
    // httptest server returns minimal chat.completions JSON
    // assert response.Content and FinishReason
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/providers/openai_sdk -run BasicContent -v`
Expected: FAIL (provider not implemented / compile errors).

**Step 3: Implement minimal provider constructor + Chat skeleton**

- Create client with `option.WithBaseURL`, `option.WithAPIKey`, `option.WithHTTPClient`.
- Call `client.Chat.Completions.New(...)` with minimal params.
- Parse first choice content + finish reason.

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/providers/openai_sdk -run BasicContent -v`
Expected: PASS.

**Step 5: Commit**

```bash
git add pkg/providers/openai_sdk/provider.go pkg/providers/openai_sdk/provider_test.go
git commit -m "feat(providers): add openai sdk provider skeleton"
```

### Task 2: Add Message/Tool Mapping Coverage

**Files:**
- Modify: `pkg/providers/openai_sdk/provider.go`
- Modify: `pkg/providers/openai_sdk/provider_test.go`

**Step 1: Write failing tests for roles and tool calls**

- user/system/assistant/tool message mapping.
- response tool_calls -> internal `providers.ToolCall` mapping.

**Step 2: Run tests to verify they fail**

Run: `go test ./pkg/providers/openai_sdk -run 'Message|ToolCall' -v`
Expected: FAIL on mapping assertions.

**Step 3: Implement mapping helpers**

- `buildChatMessages(...)`
- `buildChatTools(...)`
- `parseChoiceToolCalls(...)`

**Step 4: Run tests to verify pass**

Run: `go test ./pkg/providers/openai_sdk -run 'Message|ToolCall' -v`
Expected: PASS.

**Step 5: Commit**

```bash
git add pkg/providers/openai_sdk/provider.go pkg/providers/openai_sdk/provider_test.go
git commit -m "feat(openai-sdk): map messages and tool calls for chat completions"
```

### Task 3: Add Options Mapping (`max_tokens`, `temperature`, `prompt_cache_key`)

**Files:**
- Modify: `pkg/providers/openai_sdk/provider.go`
- Modify: `pkg/providers/openai_sdk/provider_test.go`

**Step 1: Write failing tests for options mapping**

- `max_tokens` with default field -> `max_tokens`.
- `max_tokens` with config `max_completion_tokens`.
- `temperature` mapped.
- `prompt_cache_key` mapped for OpenAI path.

**Step 2: Run tests to verify failure**

Run: `go test ./pkg/providers/openai_sdk -run 'MaxTokens|Temperature|PromptCacheKey' -v`
Expected: FAIL.

**Step 3: Implement options mapping**

- Add `maxTokensField` handling in provider config.
- Apply SDK params fields accordingly.

**Step 4: Run tests to verify pass**

Run: `go test ./pkg/providers/openai_sdk -run 'MaxTokens|Temperature|PromptCacheKey' -v`
Expected: PASS.

**Step 5: Commit**

```bash
git add pkg/providers/openai_sdk/provider.go pkg/providers/openai_sdk/provider_test.go
git commit -m "feat(openai-sdk): map max tokens, temperature, and prompt cache key"
```

### Task 4: Add Timeout/Proxy and Error-Path Coverage

**Files:**
- Modify: `pkg/providers/openai_sdk/provider.go`
- Modify: `pkg/providers/openai_sdk/provider_test.go`

**Step 1: Write failing tests**

- Request timeout using slow test server.
- Proxy transport setup verification.
- Non-200 response error formatting.

**Step 2: Run tests to verify failure**

Run: `go test ./pkg/providers/openai_sdk -run 'Timeout|Proxy|HTTPError' -v`
Expected: FAIL.

**Step 3: Implement HTTP client construction and robust errors**

- Build `http.Client` with timeout + optional proxy transport.
- Surface SDK/API errors with actionable messages.

**Step 4: Run tests to verify pass**

Run: `go test ./pkg/providers/openai_sdk -run 'Timeout|Proxy|HTTPError' -v`
Expected: PASS.

**Step 5: Commit**

```bash
git add pkg/providers/openai_sdk/provider.go pkg/providers/openai_sdk/provider_test.go
git commit -m "feat(openai-sdk): support proxy, timeout, and error handling"
```

### Task 5: Wire Factory Routing for `openai` API-Key Path

**Files:**
- Modify: `pkg/providers/factory_provider.go`
- Modify: `pkg/providers/factory_provider_test.go`
- Optionally modify: `pkg/providers/http_provider.go` (if helper reuse is needed)

**Step 1: Write failing factory tests**

- `openai` + API key => returns `*OpenAISDKProvider`.
- `openai` + oauth/token => existing codex provider path unchanged.
- non-openai protocols => existing `*HTTPProvider` path unchanged.

**Step 2: Run tests to verify failure**

Run: `go test ./pkg/providers -run 'CreateProviderFromConfig_.*OpenAI.*' -v`
Expected: FAIL until routing updated.

**Step 3: Implement routing logic**

- In `case "openai":` branch, keep auth-method split.
- API-key HTTP path creates SDK provider instead of HTTPProvider.

**Step 4: Run tests to verify pass**

Run: `go test ./pkg/providers -run 'CreateProviderFromConfig_.*OpenAI.*' -v`
Expected: PASS.

**Step 5: Commit**

```bash
git add pkg/providers/factory_provider.go pkg/providers/factory_provider_test.go
git commit -m "refactor(factory): route openai api-key protocol to sdk provider"
```

### Task 6: Full Verification and Regression Scan

**Files:**
- Modify (if needed): tests/docs only

**Step 1: Run focused package tests**

Run: `go test ./pkg/providers/openai_sdk ./pkg/providers/...`
Expected: PASS.

**Step 2: Run full test suite**

Run: `go test ./...`
Expected: PASS.

**Step 3: Verify routing boundaries**

Run: `rg -n "case \"openai\"|openai_sdk|openai_compat" pkg/providers/factory_provider.go pkg/providers`
Expected: openai API-key path points to sdk provider; other protocols remain on openai_compat.

**Step 4: Inspect changes and ensure clean state**

Run: `git status --short`
Expected: clean.

**Step 5: Commit any final test/docs adjustments**

```bash
git add -A
git commit -m "test/providers: finalize openai sdk routing regression coverage"
```

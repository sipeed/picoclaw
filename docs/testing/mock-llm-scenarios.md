# Mock LLM Scenarios

Picoclaw's mock LLM scenario harness provides deterministic end-to-end tests for
agent runtime behavior without calling a real model.

Use it when a regression spans more than one package, for example:

- model tool-call planning
- tool execution and tool-result history
- channel/chat context propagation
- async delivery and duplicate suppression
- task registry or task board state
- restart/idempotency behavior
- oversized tool output truncation

The harness lives in `pkg/testharness/llmscenario`.

## Shape

A scenario scripts model responses in order:

1. The runtime sends messages and tool definitions to `ScriptedProvider`.
2. The current `ProviderStep` optionally asserts the request shape.
3. The step returns either a final text response or tool calls.
4. Picoclaw executes real or stub tools.
5. The next provider step can assert that the tool result was fed back into
   model context.

Example:

```go
provider := llmscenario.NewScriptedProvider(
    "scenario-model",
    llmscenario.ProviderStep{
        Name: "request tool",
        Assert: llmscenario.RequireToolDefinition("scenario_extract_recipe"),
        Response: llmscenario.ToolCallResponse(
            "I will extract it.",
            llmscenario.ToolCall("call_1", "scenario_extract_recipe", map[string]any{
                "url": "https://example.test/reel",
            }),
        ),
    },
    llmscenario.ProviderStep{
        Name: "final answer",
        Assert: llmscenario.RequireLastMessage("tool", "recipe extracted"),
        Response: llmscenario.TextResponse("Final recipe..."),
    },
)
```

Then run the real `AgentLoop` or `tools.RunToolLoop` with that provider.

## Test Boundary

These are not unit tests. They are deterministic runtime integration tests:

- no real LLM
- no network provider dependency
- real Picoclaw loop/tool/registry/delivery code where the scenario needs it

Prefer this harness over live Telegram/manual testing for regressions that need
to stay fixed.

# Model Fallbacks

PicoClaw supports automatic fallback chains for LLM models, ensuring high availability even when primary models are unavailable. This guide explains how to configure and understand fallback behavior.

## Overview

The fallback system provides:

- **High availability** - Automatically switch to backup models on failure
- **Rate limit handling** - Cooldown tracking for rate-limited providers
- **Intelligent error classification** - Only retry on retriable errors
- **Image model support** - Separate fallback chains for vision tasks

## Configuration

### Simple Fallbacks (Defaults)

Configure fallback models at the agent defaults level:

```json
{
  "agents": {
    "defaults": {
      "model": "anthropic/claude-opus-4-5",
      "model_fallbacks": ["anthropic/claude-sonnet-4", "gpt-4o", "glm-4.7"]
    }
  }
}
```

### Structured Model Configuration

For multi-agent setups, use the object format:

```json
{
  "agents": {
    "list": [
      {
        "id": "assistant",
        "model": {
          "primary": "anthropic/claude-opus-4-5",
          "fallbacks": ["anthropic/claude-sonnet-4", "gpt-4o"]
        }
      }
    ]
  }
}
```

### Image Model Fallbacks

Configure separate fallbacks for image/vision tasks:

```json
{
  "agents": {
    "defaults": {
      "image_model": "anthropic/claude-opus-4-5",
      "image_model_fallbacks": ["gpt-4o", "anthropic/claude-sonnet-4"]
    }
  }
}
```

## Model Format

Models are specified in the format `provider/model`:

```
anthropic/claude-opus-4-5
openrouter/anthropic/claude-opus-4-5
groq/llama-3.1-70b-versatile
openai/gpt-4o
zhipu/glm-4.7
```

If no provider is specified, the default provider is used.

## Fallback Behavior

### Execution Order

1. Try primary model
2. If failure, try first fallback
3. Continue through fallbacks until success
4. If all fail, return aggregate error

### Error Classification

The system classifies errors to decide whether to fallback:

| Error Type | Retriable | Action |
|------------|-----------|--------|
| Rate limit (429) | Yes | Fallback with cooldown |
| Server error (5xx) | Yes | Fallback immediately |
| Timeout | Yes | Fallback immediately |
| Invalid request (400) | No | Return error immediately |
| Auth error (401/403) | No | Return error immediately |
| Context canceled | No | Return error immediately |

### Cooldown Tracking

When a provider hits rate limits, it enters cooldown:

- **Duration**: Based on rate limit response
- **Effect**: Provider is skipped during cooldown
- **Recovery**: Provider used again after cooldown expires

### Success Handling

When a request succeeds:

1. Response is returned immediately
2. Provider cooldown is reset
3. No further fallbacks are attempted

## Fallback Chain Resolution

The system deduplicates models in the chain:

```json
{
  "model": {
    "primary": "gpt-4o",
    "fallbacks": ["gpt-4o", "anthropic/claude-sonnet-4", "gpt-4o"]
  }
}
```

Resolves to: `gpt-4o`, `anthropic/claude-sonnet-4`

## Example Scenarios

### Scenario 1: Primary Rate Limited

```
Request → claude-opus-4-5 (429 rate limit)
       → claude-sonnet-4 (success)
       → Return response
```

### Scenario 2: All Models Fail

```
Request → claude-opus-4-5 (500 server error)
       → gpt-4o (429 rate limit)
       → glm-4.7 (timeout)
       → Return aggregate error with all attempts
```

### Scenario 3: Non-Retriable Error

```
Request → claude-opus-4-5 (400 invalid request)
       → Return error immediately (no fallback)
```

### Scenario 4: Provider in Cooldown

```
Request → claude-opus-4-5 (skipped - in cooldown)
       → gpt-4o (success)
       → Return response
```

## Image Model Fallbacks

Image/vision requests use a simplified fallback chain:

- **No cooldown checks** - Image endpoints have different rate limits
- **Dimension errors abort** - Image size errors are non-retriable
- **Same fallback order** - Uses configured fallback list

### Configuration

```json
{
  "agents": {
    "defaults": {
      "image_model": "anthropic/claude-opus-4-5",
      "image_model_fallbacks": ["gpt-4o"]
    }
  }
}
```

## Subagent Model Overrides

Control which models spawned subagents use:

```json
{
  "agents": {
    "list": [
      {
        "id": "assistant",
        "subagents": {
          "allow_agents": ["researcher"],
          "model": {
            "primary": "gpt-4o-mini",
            "fallbacks": ["glm-4-flash"]
          }
        }
      }
    ]
  }
}
```

## Debugging Fallbacks

Enable debug mode to see fallback decisions:

```bash
picoclaw agent --debug -m "Hello"
```

Debug output includes:

- Candidate model list
- Each attempt and its result
- Cooldown status
- Final successful model or error

### Example Debug Output

```
[providers] Resolved candidates: [claude-opus-4-5, claude-sonnet-4, gpt-4o]
[providers] Attempt 1: claude-opus-4-5 - rate limited (cooldown: 60s)
[providers] Attempt 2: claude-sonnet-4 - success (1.2s)
[providers] Fallback complete: claude-sonnet-4
```

## Environment Variables

Override model configuration with environment variables:

```bash
# Primary model
export PICOCLAW_AGENTS_DEFAULTS_MODEL="anthropic/claude-opus-4-5"

# Fallback models (JSON array)
export PICOCLAW_AGENTS_DEFAULTS_MODEL_FALLBACKS='["gpt-4o", "glm-4.7"]'

# Image model
export PICOCLAW_AGENTS_DEFAULTS_IMAGE_MODEL="gpt-4o"
```

## Best Practices

### 1. Order by Preference

List models in order of quality/preference:

```json
{
  "model_fallbacks": ["claude-sonnet-4", "gpt-4o", "glm-4.7"]
}
```

### 2. Mix Providers

Use different providers to avoid single points of failure:

```json
{
  "model_fallbacks": [
    "anthropic/claude-sonnet-4",
    "openai/gpt-4o",
    "zhipu/glm-4.7"
  ]
}
```

### 3. Consider Costs

Order by cost when appropriate:

```json
{
  "model_fallbacks": [
    "groq/llama-3.1-70b-versatile",
    "openai/gpt-4o-mini",
    "anthropic/claude-sonnet-4"
  ]
}
```

### 4. Test Fallback Chain

Verify your fallback chain works:

```bash
# Simulate primary failure by using invalid API key
picoclaw agent --debug -m "Test"
```

## Related Topics

- [Multi-Agent System](multi-agent.md) - Agent-specific model configuration
- [Providers](../providers/README.md) - Configure LLM providers
- [Environment Variables](environment-variables.md) - Override settings

# Multi-Agent System

PicoClaw supports running multiple specialized agents, each with its own configuration, workspace, and model settings. This enables scenarios like having separate agents for research, coding, general assistance, or different teams.

## Overview

The multi-agent system allows you to:

- Create specialized agents for different tasks
- Isolate workspaces between agents
- Use different models for different agents
- Control which subagents can be spawned
- Route messages to specific agents based on source

## Configuration

Agents are defined in the `agents.list` array in your configuration file:

```json
{
  "agents": {
    "defaults": {
      "workspace": "~/.picoclaw/workspace",
      "restrict_to_workspace": true,
      "model": "glm-4.7",
      "max_tokens": 8192,
      "temperature": 0.7
    },
    "list": [
      {
        "id": "assistant",
        "default": true,
        "name": "General Assistant",
        "workspace": "~/.picoclaw/workspace/assistant",
        "model": {
          "primary": "anthropic/claude-opus-4-5",
          "fallbacks": ["gpt-4o"]
        },
        "subagents": {
          "allow_agents": ["researcher", "coder"]
        }
      },
      {
        "id": "researcher",
        "name": "Research Agent",
        "model": "perplexity/llama-3.1-sonar-large-128k-online",
        "skills": ["web-search"]
      },
      {
        "id": "coder",
        "name": "Coding Agent",
        "workspace": "~/.picoclaw/workspace/coder",
        "model": "anthropic/claude-sonnet-4"
      }
    ]
  }
}
```

## Agent Properties

| Property | Type | Required | Description |
|----------|------|----------|-------------|
| `id` | string | Yes | Unique agent identifier |
| `default` | bool | No | Whether this is the default agent |
| `name` | string | No | Human-readable display name |
| `workspace` | string | No | Agent-specific workspace directory |
| `model` | string/object | No | Model configuration (see below) |
| `skills` | []string | No | Filter available skills for this agent |
| `subagents` | object | No | Subagent spawning configuration |

## Model Configuration

The `model` property can be specified in two formats:

### String Format (Simple)

```json
{
  "model": "anthropic/claude-opus-4-5"
}
```

### Object Format (With Fallbacks)

```json
{
  "model": {
    "primary": "anthropic/claude-opus-4-5",
    "fallbacks": ["gpt-4o", "glm-4.7"]
  }
}
```

See [Model Fallbacks](model-fallbacks.md) for detailed fallback configuration.

## Subagent Configuration

Control which agents can be spawned by an agent using the `subagents` property:

```json
{
  "id": "assistant",
  "subagents": {
    "allow_agents": ["researcher", "coder"]
  }
}
```

### Wildcard Permission

Allow spawning any agent:

```json
{
  "subagents": {
    "allow_agents": ["*"]
  }
}
```

### Subagent Model Override

Override the model for spawned subagents:

```json
{
  "subagents": {
    "allow_agents": ["researcher"],
    "model": {
      "primary": "gpt-4o-mini",
      "fallbacks": ["glm-4-flash"]
    }
  }
}
```

## Workspace Isolation

Each agent can have its own isolated workspace:

```
~/.picoclaw/workspace/
├── assistant/          # Agent: assistant
│   ├── sessions/
│   ├── memory/
│   ├── AGENT.md
│   └── IDENTITY.md
├── researcher/         # Agent: researcher
│   ├── sessions/
│   └── ...
└── coder/              # Agent: coder
    ├── sessions/
    └── ...
```

### Benefits of Workspace Isolation

- **Separate conversation history** - Each agent maintains its own sessions
- **Independent memory** - Long-term memory (MEMORY.md) is agent-specific
- **Isolated skills** - Each agent can have custom skills
- **Different configurations** - AGENT.md can define agent-specific behavior

## Default Agent Selection

The default agent is selected when no binding matches. PicoClaw uses this priority:

1. Agent with `"default": true`
2. First agent in the list
3. Implicit "main" agent (if no agents defined)

## Implicit Single Agent

If no agents are defined in `agents.list`, PicoClaw creates an implicit "main" agent using the defaults:

```json
{
  "agents": {
    "defaults": {
      "model": "glm-4.7"
    }
  }
}
```

This is equivalent to:

```json
{
  "agents": {
    "list": [
      {
        "id": "main",
        "default": true
      }
    ]
  }
}
```

## Use Cases

### Personal vs Work Agents

```json
{
  "agents": {
    "list": [
      {
        "id": "personal",
        "default": true,
        "name": "Personal Assistant",
        "workspace": "~/.picoclaw/workspace/personal"
      },
      {
        "id": "work",
        "name": "Work Assistant",
        "workspace": "~/.picoclaw/workspace/work",
        "model": "anthropic/claude-opus-4-5"
      }
    ]
  },
  "bindings": [
    {
      "agent_id": "personal",
      "match": { "channel": "telegram", "peer": { "kind": "user", "id": "123456789" } }
    },
    {
      "agent_id": "work",
      "match": { "channel": "slack", "team_id": "T12345" }
    }
  ]
}
```

### Specialized Task Agents

```json
{
  "agents": {
    "list": [
      {
        "id": "assistant",
        "default": true,
        "name": "General Assistant",
        "subagents": {
          "allow_agents": ["researcher", "coder", "writer"]
        }
      },
      {
        "id": "researcher",
        "name": "Research Specialist",
        "model": "perplexity/llama-3.1-sonar-large-128k-online",
        "skills": ["web-search"]
      },
      {
        "id": "coder",
        "name": "Code Specialist",
        "model": "anthropic/claude-sonnet-4"
      },
      {
        "id": "writer",
        "name": "Writing Specialist",
        "model": "anthropic/claude-opus-4-5"
      }
    ]
  }
}
```

### Model-Specific Agents

Use different models for different capabilities:

```json
{
  "agents": {
    "list": [
      {
        "id": "fast",
        "name": "Fast Responses",
        "model": "groq/llama-3.1-70b-versatile"
      },
      {
        "id": "smart",
        "name": "Complex Reasoning",
        "model": "anthropic/claude-opus-4-5"
      },
      {
        "id": "cheap",
        "name": "Simple Tasks",
        "model": "openai/gpt-4o-mini"
      }
    ]
  }
}
```

## Environment Variables

Override agent settings with environment variables:

```bash
# Default model for all agents
export PICOCLAW_AGENTS_DEFAULTS_MODEL="anthropic/claude-opus-4-5"

# Fallback models
export PICOCLAW_AGENTS_DEFAULTS_MODEL_FALLBACKS='["gpt-4o", "glm-4.7"]'
```

## Related Topics

- [Message Routing](routing.md) - Route messages to specific agents
- [Model Fallbacks](model-fallbacks.md) - Configure model fallback chains
- [Session Management](session-management.md) - Understand session scoping per agent
- [Spawn Tool](../tools/README.md) - Spawn subagents from conversations

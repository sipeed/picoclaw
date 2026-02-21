# Multi-Agent Setup Tutorial

This tutorial guides you through setting up multiple specialized agents with PicoClaw.

## Prerequisites

- 20 minutes
- PicoClaw installed and configured
- A working LLM provider
- Basic understanding of PicoClaw configuration

## Overview

Multi-agent setup allows you to run specialized agents for different tasks:

```
┌──────────────────────────────────────────────────────────┐
│                        PicoClaw                          │
│                                                          │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐  │
│  │ Default  │  │ Research │  │ Coding   │  │ Hardware │  │
│  │ Agent    │  │ Agent    │  │ Agent    │  │ Agent    │  │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘  │
│       ▲              ▲             ▲             ▲       │
│       │              │             │             │       │
│  ┌────┴────┐    ┌────┴────┐   ┌────┴────┐   ┌────┴────┐  │
│  │Telegram │    │Discord  │   │ CLI     │   │  CLI    │  │
│  └─────────┘    └─────────┘   └─────────┘   └─────────┘  │
└──────────────────────────────────────────────────────────┘
```

## Use Cases

- **Specialized behavior**: Different agents for different tasks
- **Model optimization**: Use cheaper models for simple tasks
- **Access control**: Different permissions per channel
- **Workspace isolation**: Separate files and sessions

## Step 1: Understand Agent Configuration

### Default Agent

The default agent handles all unbound requests:

```json
{
  "agents": {
    "defaults": {
      "workspace": "~/.picoclaw/workspace",
      "model": "anthropic/claude-opus-4-5"
    }
  }
}
```

### Named Agents

Add agents to the `list` array:

```json
{
  "agents": {
    "defaults": { ... },
    "list": [
      {
        "id": "research",
        "workspace": "~/.picoclaw/workspaces/research",
        "model": "anthropic/claude-opus-4-5"
      }
    ]
  }
}
```

## Step 2: Create Agent Workspaces

Each agent needs its own workspace:

```bash
# Create workspaces
mkdir -p ~/.picoclaw/workspaces/research
mkdir -p ~/.picoclaw/workspaces/coding
mkdir -p ~/.picoclaw/workspaces/hardware
```

## Step 3: Configure Multiple Agents

Edit `~/.picoclaw/config.json`:

```json
{
  "agents": {
    "defaults": {
      "workspace": "~/.picoclaw/workspace",
      "model": "anthropic/claude-opus-4-5",
      "max_tokens": 8192,
      "temperature": 0.7
    },
    "list": [
      {
        "id": "research",
        "workspace": "~/.picoclaw/workspaces/research",
        "model": "anthropic/claude-opus-4-5",
        "temperature": 0.3,
        "system_prompt": "You are a research assistant specialized in gathering and analyzing information."
      },
      {
        "id": "coding",
        "workspace": "~/.picoclaw/workspaces/coding",
        "model": "anthropic/claude-sonnet-4",
        "temperature": 0.2,
        "system_prompt": "You are a coding assistant focused on writing clean, efficient code."
      },
      {
        "id": "hardware",
        "workspace": "~/.picoclaw/workspaces/hardware",
        "model": "anthropic/claude-sonnet-4",
        "restrict_to_workspace": false
      }
    ]
  }
}
```

## Step 4: Customize Agent Behavior

Create AGENT.md for each workspace:

### Research Agent

```bash
nano ~/.picoclaw/workspaces/research/AGENT.md
```

```markdown
# Research Agent

## Role
You are a research assistant specialized in gathering and analyzing information.

## Capabilities
- Web search and summarization
- Fact-checking and source verification
- Creating research summaries
- Organizing information

## Behavior
- Always cite sources
- Be thorough but concise
- Verify claims with multiple sources
- Store findings in memory
```

### Coding Agent

```bash
nano ~/.picoclaw/workspaces/coding/AGENT.md
```

```markdown
# Coding Agent

## Role
You are a coding assistant focused on writing clean, efficient code.

## Capabilities
- Writing code in various languages
- Code review and optimization
- Debugging and testing
- Documentation

## Behavior
- Follow best practices
- Include comments for complex logic
- Write testable code
- Explain your solutions
```

### Hardware Agent

```bash
nano ~/.picoclaw/workspaces/hardware/AGENT.md
```

```markdown
# Hardware Agent

## Role
You are a hardware controller for IoT and embedded systems.

## Capabilities
- I2C device communication
- SPI device control
- Sensor reading and logging
- Actuator control

## Behavior
- Always check device availability before operations
- Log all hardware interactions
- Handle errors gracefully
- Report sensor readings with units
```

## Step 5: Set Up Routing

Routing determines which agent handles which requests.

### Channel Binding

Bind specific channels to specific agents:

```json
{
  "bindings": [
    {
      "channel": "telegram",
      "chat_id": "123456789",
      "agent_id": "default"
    },
    {
      "channel": "discord",
      "chat_id": "987654321",
      "agent_id": "research"
    }
  ]
}
```

### CLI Routing

Use the agent ID with CLI:

```bash
# Use default agent
picoclaw agent -m "Hello"

# Use specific agent
picoclaw agent --agent research -m "Research quantum computing"
picoclaw agent --agent coding -m "Write a Python script"
picoclaw agent --agent hardware -m "Read temperature sensor"
```

### Short form:

```bash
picoclaw agent -a research -m "Research topic"
```

## Step 6: Configure Subagents

Allow agents to spawn subagents for complex tasks:

```json
{
  "agents": {
    "defaults": {
      "subagents": ["research", "coding", "hardware"]
    },
    "list": [
      {
        "id": "research",
        "subagents": []
      },
      {
        "id": "coding",
        "subagents": ["research"]
      },
      {
        "id": "hardware",
        "subagents": []
      }
    ]
  }
}
```

### Using Subagents

The default agent can delegate to specialists:

```
User: "Research REST APIs and then create a Python client"

Agent: I'll research REST APIs first, then write the code.

[Spawns research agent]
Research complete: REST APIs are...

[Spawns coding agent]
Here's the Python client code...
```

## Step 7: Test Your Setup

### Test Each Agent

```bash
# Test default agent
picoclaw agent -m "Hello from default"

# Test research agent
picoclaw agent -a research -m "Research machine learning"

# Test coding agent
picoclaw agent -a coding -m "Write a hello world in Python"

# Test hardware agent
picoclaw agent -a hardware -m "List I2C devices"
```

### Verify Workspace Isolation

```bash
# Each agent has its own session
ls ~/.picoclaw/workspace/sessions/
ls ~/.picoclaw/workspaces/research/sessions/
ls ~/.picoclaw/workspaces/coding/sessions/
```

## Step 8: Model Optimization

Use different models for different agents:

```json
{
  "agents": {
    "list": [
      {
        "id": "simple",
        "model": "openai/gpt-4o-mini",
        "description": "Fast, cheap tasks"
      },
      {
        "id": "complex",
        "model": "anthropic/claude-opus-4-5",
        "description": "Complex reasoning"
      }
    ]
  }
}
```

### Fallbacks Per Agent

```json
{
  "agents": {
    "list": [
      {
        "id": "research",
        "model": "anthropic/claude-opus-4-5",
        "model_fallbacks": [
          "anthropic/claude-sonnet-4",
          "openai/gpt-4o"
        ]
      }
    ]
  }
}
```

## Step 9: Advanced Routing Patterns

### Regex Routing

Route based on message patterns:

```json
{
  "routing": {
    "rules": [
      {
        "pattern": "^code:|^implement:|^debug:",
        "agent_id": "coding"
      },
      {
        "pattern": "^research:|^search:|^find:",
        "agent_id": "research"
      },
      {
        "pattern": "^sensor:|^i2c:|^hardware:",
        "agent_id": "hardware"
      }
    ]
  }
}
```

### Time-Based Routing

Use different agents at different times:

```json
{
  "routing": {
    "rules": [
      {
        "time_range": {"start": "09:00", "end": "17:00"},
        "agent_id": "work"
      },
      {
        "time_range": {"start": "17:00", "end": "09:00"},
        "agent_id": "personal"
      }
    ]
  }
}
```

## Step 10: Monitoring Multiple Agents

### Status Command

```bash
picoclaw status
```

Output:

```
PicoClaw Status
===============

Agents:
  - default (anthropic/claude-opus-4-5)
    Sessions: 5
  - research (anthropic/claude-opus-4-5)
    Sessions: 3
  - coding (anthropic/claude-sonnet-4)
    Sessions: 8
  - hardware (anthropic/claude-sonnet-4)
    Sessions: 2

Channels:
  - telegram: connected
  - discord: connected
```

## Practical Example: Dev Team Setup

Here's a complete configuration for a development team:

```json
{
  "agents": {
    "defaults": {
      "workspace": "~/.picoclaw/workspace",
      "model": "anthropic/claude-sonnet-4",
      "max_tokens": 4096
    },
    "list": [
      {
        "id": "code-review",
        "workspace": "~/.picoclaw/workspaces/code-review",
        "model": "anthropic/claude-opus-4-5",
        "system_prompt": "You are a code reviewer. Analyze code for bugs, security issues, and improvements."
      },
      {
        "id": "docs",
        "workspace": "~/.picoclaw/workspaces/docs",
        "model": "anthropic/claude-sonnet-4",
        "system_prompt": "You are a technical writer. Create clear, comprehensive documentation."
      },
      {
        "id": "devops",
        "workspace": "~/.picoclaw/workspaces/devops",
        "model": "anthropic/claude-sonnet-4",
        "system_prompt": "You are a DevOps engineer. Help with deployment, CI/CD, and infrastructure."
      }
    ]
  },
  "bindings": [
    {
      "channel": "discord",
      "chat_id": "code-review-channel-id",
      "agent_id": "code-review"
    },
    {
      "channel": "discord",
      "chat_id": "docs-channel-id",
      "agent_id": "docs"
    }
  ]
}
```

## Troubleshooting

### Agent Not Found

```
Error: agent 'xyz' not found
```

Check the agent ID in your config matches what you're using.

### Wrong Agent Responding

1. Check bindings configuration
2. Verify routing rules
3. Check channel/chat IDs

### Workspace Issues

```
Error: workspace not accessible
```

1. Check workspace paths exist
2. Verify permissions
3. Check disk space

## Best Practices

1. **Use descriptive agent IDs** - Easy to remember and type
2. **Match models to tasks** - Use appropriate models for each agent
3. **Isolate workspaces** - Keep agent data separate
4. **Configure subagents wisely** - Only allow necessary delegation
5. **Monitor usage** - Track which agents are used most
6. **Document agents** - Clear AGENT.md for each

## Next Steps

- [Routing Documentation](../user-guide/advanced/routing.md) - Advanced routing
- [Spawn Tool](../user-guide/tools/spawn.md) - Subagent spawning
- [Session Management](../user-guide/advanced/session-management.md) - Session handling

## Summary

You learned:
- How to configure multiple agents
- How to create specialized agent workspaces
- How to set up channel bindings
- How to use subagents
- How to optimize models per agent

You can now run a multi-agent PicoClaw system!

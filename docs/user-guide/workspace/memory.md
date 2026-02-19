# Long-term Memory

MEMORY.md provides persistent memory that the agent remembers across sessions.

## Location

```
~/.picoclaw/workspace/memory/MEMORY.md
```

## Purpose

Unlike session history (which resets per session), long-term memory persists across all conversations. Use it to store:

- User preferences
- Important facts
- Recurring context
- Personal information

## Example Content

```markdown
# Long-term Memory

## User Information

- Name: Alice Chen
- Location: Shanghai, China
- Timezone: UTC+8
- Occupation: Software Engineer

## Preferences

- Prefers concise responses
- Likes bullet points over paragraphs
- Interested in AI and machine learning
- Uses Python for most projects

## Ongoing Projects

### Project Alpha
- Status: In development
- Tech stack: Python, FastAPI, PostgreSQL
- Next steps: Implement authentication

### Learning Goals
- Learn Rust programming
- Study Kubernetes
- Practice system design

## Important Dates

- 2025-03-01: Project deadline
- 2025-03-15: Team offsite

## Notes

- Works remotely
- Available 9 AM - 6 PM UTC+8
- Prefers async communication
```

## How It Works

1. Agent reads MEMORY.md at the start of each conversation
2. Content provides context for responses
3. Agent can reference stored information
4. You can ask the agent to update memory

## Updating Memory

Ask the agent to remember something:

```
"Remember that I prefer using tabs over spaces for indentation"
```

Or manually edit the file:

```bash
vim ~/.picoclaw/workspace/memory/MEMORY.md
```

## Best Practices

1. **Keep it organized** - Use clear sections
2. **Update regularly** - Keep information current
3. **Be specific** - Include relevant details
4. **Don't overload** - Focus on important information

## Memory vs Session

| Feature | MEMORY.md | Session History |
|---------|-----------|-----------------|
| Persistence | Forever | Per session |
| Scope | All conversations | Single conversation |
| Content | Important facts | Full conversation |
| Size | Recommended < 10KB | Auto-summarized |

## Example Usage

```bash
# User: What projects am I working on?

# Agent (reading MEMORY.md):
Based on your memory, you're working on:
- Project Alpha (Python/FastAPI) - in development
- Learning Rust and Kubernetes
```

## See Also

- [Workspace Structure](structure.md)
- [AGENT.md Guide](agent-md.md)

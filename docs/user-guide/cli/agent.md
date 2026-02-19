# picoclaw agent

Interact with the AI agent directly from the command line.

## Usage

```bash
# One-shot message
picoclaw agent -m "Your message here"

# Interactive mode
picoclaw agent

# With debug logging
picoclaw agent --debug -m "Hello"

# With specific session
picoclaw agent -s "my-session" -m "Continue our chat"
```

## Options

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--message` | `-m` | - | Send a single message and exit |
| `--session` | `-s` | `cli:default` | Session key for conversation history |
| `--debug` | `-d` | `false` | Enable debug logging |

## Modes

### One-Shot Mode

Send a single message and receive a response:

```bash
picoclaw agent -m "What is 2+2?"
```

Output:
```
🦞 2 + 2 equals 4.
```

### Interactive Mode

Start a continuous chat session:

```bash
picoclaw agent
```

```
🦞 Interactive mode (Ctrl+C to exit)

🦞 You: Hello!

🦞 Hello! How can I help you today?

🦞 You: Tell me a joke

🦞 Why don't scientists trust atoms? Because they make up everything!

🦞 You: exit
Goodbye!
```

### Debug Mode

See detailed processing information:

```bash
picoclaw agent --debug -m "Hello"
```

Debug output includes:
- LLM API requests
- Tool invocations
- Response parsing
- Timing information

## Sessions

Sessions maintain conversation history:

```bash
# Start a new session
picoclaw agent -s "work-project" -m "I'm starting a new project"

# Continue the same session
picoclaw agent -s "work-project" -m "What should we do next?"
```

Sessions are stored in `~/.picoclaw/workspace/sessions/`.

## Examples

```bash
# Quick question
picoclaw agent -m "What's the capital of France?"

# Code assistance
picoclaw agent -m "Write a Python function to reverse a string"

# With web search
picoclaw agent -m "Search for the latest news about AI and summarize"

# Continue previous conversation
picoclaw agent -s "morning-standup" -m "What did we discuss yesterday?"
```

## See Also

- [First Chat](../../getting-started/first-chat.md)
- [CLI Reference](../cli-reference.md)

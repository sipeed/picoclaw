# Your First Chat

After [installing](installation.md) and [configuring](quick-start.md) PicoClaw, you're ready to chat!

## One-Shot Messages

Send a single message and get a response:

```bash
picoclaw agent -m "What is 2+2?"
```

The agent will process your message and print the response.

## Interactive Mode

Start an interactive chat session:

```bash
picoclaw agent
```

```
🦞 Interactive mode (Ctrl+C to exit)

🦞 You: Hello, who are you?

🦞 I'm PicoClaw, your AI assistant! I'm an ultra-lightweight AI agent written in Go. How can I help you today?

🦞 You: What can you do?

🦞 I can help you with many tasks:
- Answer questions and provide information
- Search the web for current information
- Read and write files in the workspace
- Execute shell commands
- Schedule reminders and tasks
- And much more!
```

### Interactive Commands

| Command | Action |
|---------|--------|
| Type a message | Send to the agent |
| `exit` or `quit` | Exit interactive mode |
| `Ctrl+C` | Exit interactive mode |

## Debug Mode

See what's happening behind the scenes:

```bash
picoclaw agent --debug -m "Hello"
```

Debug mode shows:
- LLM requests
- Tool calls
- Response processing

## Using Tools

The agent can use tools automatically. Just ask:

```bash
picoclaw agent -m "Search the web for the latest AI news and summarize it"
```

The agent will:
1. Use the `web_search` tool
2. Use the `web_fetch` tool to get content
3. Summarize the findings

## Sessions

Your conversation history is automatically saved. Continue a conversation:

```bash
# Start a session
picoclaw agent -s "project-planning" -m "I'm planning a new project"

# Continue later
picoclaw agent -s "project-planning" -m "What should we do next?"
```

Session files are stored in `~/.picoclaw/workspace/sessions/`.

## With Gateway

Connect to chat platforms:

```bash
# Start the gateway
picoclaw gateway
```

Then chat via:
- Telegram
- Discord
- Slack
- LINE
- And more

See [Channels](../user-guide/channels/README.md) for setup guides.

## Tips for Better Chats

1. **Be specific** - Clear questions get better answers
2. **Provide context** - Background info helps understanding
3. **Use sessions** - Maintain conversation context
4. **Check debug mode** - Understand what's happening

## Next Steps

- [CLI Reference](../user-guide/cli-reference.md) - All commands
- [Tools Overview](../user-guide/tools/README.md) - Available tools
- [Channels Guide](../user-guide/channels/README.md) - Connect to chat apps

# Data Flow and Message Bus

This document explains how messages flow through PicoClaw's message bus architecture.

## Message Bus Overview

The message bus is the central communication hub in PicoClaw. It decouples channels (which receive external messages) from the agent loop (which processes them).

### Message Types

```go
// InboundMessage represents a message from an external platform
type InboundMessage struct {
    Channel    string            // Platform identifier (e.g., "telegram", "discord")
    SenderID   string            // User identifier on the platform
    ChatID     string            // Chat/conversation identifier
    Content    string            // Message text content
    Media      []string          // Optional media URLs or file paths
    SessionKey string            // Optional pre-computed session key
    Metadata   map[string]string // Additional platform-specific metadata
}

// OutboundMessage represents a response to be sent
type OutboundMessage struct {
    Channel string // Target channel
    ChatID  string // Target chat/conversation
    Content string // Response content
}
```

## Message Flow Diagram

```
┌─────────────────────────────────────────────────────────────────────────┐
│                            External Platform                             │
│                              (e.g., Telegram)                           │
└─────────────────────────────────────┬───────────────────────────────────┘
                                      │ User sends message
                                      ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                            Telegram Channel                              │
│                                                                          │
│  1. Receive update from Telegram API                                    │
│  2. Extract sender, chat, content                                       │
│  3. Call HandleMessage()                                                │
└─────────────────────────────────────┬───────────────────────────────────┘
                                      │ bus.PublishInbound(msg)
                                      ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                            Message Bus                                   │
│                                                                          │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                     Inbound Channel (buffer: 100)                │   │
│  │                                                                   │   │
│  │   InboundMessage{Channel:"telegram", ChatID:"123", ...}         │   │
│  └─────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────┬───────────────────────────────────┘
                                      │ bus.ConsumeInbound(ctx)
                                      ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                            Agent Loop                                    │
│                                                                          │
│  ┌────────────────────────────────────────────────────────────────┐    │
│  │  1. Consume from inbound bus                                    │    │
│  │  2. Route to appropriate agent                                  │    │
│  │  3. Build context (history, system prompt, skills)             │    │
│  │  4. Call LLM provider                                           │    │
│  │  5. Handle tool calls (if any)                                  │    │
│  │  6. Save to session                                             │    │
│  │  7. Publish response to outbound bus                            │    │
│  └────────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────┬───────────────────────────────────┘
                                      │ bus.PublishOutbound(msg)
                                      ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                            Message Bus                                   │
│                                                                          │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                     Outbound Channel (buffer: 100)               │   │
│  │                                                                   │   │
│  │   OutboundMessage{Channel:"telegram", ChatID:"123", ...}        │   │
│  └─────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────┬───────────────────────────────────┘
                                      │ bus.SubscribeOutbound(ctx)
                                      ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                            Telegram Channel                              │
│                                                                          │
│  1. Receive outbound message                                            │
│  2. Send via Telegram API                                               │
└─────────────────────────────────────┬───────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                            External Platform                             │
│                              (e.g., Telegram)                           │
│                                                                          │
│                            User receives response                        │
└─────────────────────────────────────────────────────────────────────────┘
```

## Detailed Flow

### 1. Inbound Message Processing

When a message arrives from an external platform:

```go
// In the channel implementation (e.g., telegram.go)
func (c *TelegramChannel) handleUpdate(update tgbotapi.Update) {
    // Extract message details
    senderID := strconv.FormatInt(update.Message.From.ID, 10)
    chatID := strconv.FormatInt(update.Message.Chat.ID, 10)
    content := update.Message.Text

    // Publish to message bus
    c.HandleMessage(senderID, chatID, content, nil, nil)
}

// In BaseChannel
func (c *BaseChannel) HandleMessage(senderID, chatID, content string,
                                    media []string, metadata map[string]string) {
    if !c.IsAllowed(senderID) {
        return // Skip unauthorized users
    }

    msg := bus.InboundMessage{
        Channel:  c.name,
        SenderID: senderID,
        ChatID:   chatID,
        Content:  content,
        Media:    media,
        Metadata: metadata,
    }

    c.bus.PublishInbound(msg)
}
```

### 2. Agent Loop Processing

The agent loop continuously processes messages:

```go
func (al *AgentLoop) Run(ctx context.Context) error {
    al.running.Store(true)

    for al.running.Load() {
        select {
        case <-ctx.Done():
            return nil
        default:
            // Consume message from bus
            msg, ok := al.bus.ConsumeInbound(ctx)
            if !ok {
                continue
            }

            // Process the message
            response, err := al.processMessage(ctx, msg)
            if err != nil {
                response = fmt.Sprintf("Error: %v", err)
            }

            // Publish response
            if response != "" {
                al.bus.PublishOutbound(bus.OutboundMessage{
                    Channel: msg.Channel,
                    ChatID:  msg.ChatID,
                    Content: response,
                })
            }
        }
    }
    return nil
}
```

### 3. Message Routing

Messages are routed to appropriate agents:

```go
func (al *AgentLoop) processMessage(ctx context.Context, msg bus.InboundMessage) (string, error) {
    // Route to determine agent and session key
    route := al.registry.ResolveRoute(routing.RouteInput{
        Channel:    msg.Channel,
        AccountID:  msg.Metadata["account_id"],
        Peer:       extractPeer(msg),
        ParentPeer: extractParentPeer(msg),
        GuildID:    msg.Metadata["guild_id"],
        TeamID:     msg.Metadata["team_id"],
    })

    agent, ok := al.registry.GetAgent(route.AgentID)
    if !ok {
        agent = al.registry.GetDefaultAgent()
    }

    return al.runAgentLoop(ctx, agent, processOptions{
        SessionKey:  route.SessionKey,
        Channel:     msg.Channel,
        ChatID:      msg.ChatID,
        UserMessage: msg.Content,
    })
}
```

### 4. Context Building

The context builder assembles the LLM prompt:

```go
messages := agent.ContextBuilder.BuildMessages(
    history,        // Previous messages from session
    summary,        // Summary of older messages (if any)
    userMessage,    // Current user message
    nil,            // Additional context
    channel,        // Current channel
    chatID,         // Current chat ID
)
```

The built messages include:
1. System prompt (from AGENT.md, IDENTITY.md, etc.)
2. Available tools list
3. Available skills
4. Session summary (if exists)
5. Conversation history
6. Current user message

### 5. LLM Call with Fallback

```go
// With fallback chain
if len(agent.Candidates) > 1 && al.fallback != nil {
    fbResult, err := al.fallback.Execute(ctx, agent.Candidates,
        func(ctx context.Context, provider, model string) (*LLMResponse, error) {
            return agent.Provider.Chat(ctx, messages, tools, model, options)
        },
    )
    // Handle result...
}
```

### 6. Tool Execution

When the LLM returns tool calls:

```go
for _, tc := range response.ToolCalls {
    // Execute the tool with context
    toolResult := agent.Tools.ExecuteWithContext(
        ctx,
        tc.Name,
        tc.Arguments,
        channel,
        chatID,
        asyncCallback, // For async tools
    )

    // Send ForUser content immediately if not silent
    if !toolResult.Silent && toolResult.ForUser != "" {
        al.bus.PublishOutbound(bus.OutboundMessage{
            Channel: channel,
            ChatID:  chatID,
            Content: toolResult.ForUser,
        })
    }

    // Add tool result to messages for next LLM call
    messages = append(messages, providers.Message{
        Role:       "tool",
        Content:    toolResult.ForLLM,
        ToolCallID: tc.ID,
    })
}
```

### 7. Outbound Message Handling

Channels subscribe to outbound messages:

```go
// In gateway main loop
for {
    select {
    case <-ctx.Done():
        return
    case msg, ok := msgBus.SubscribeOutbound(ctx):
        if !ok {
            continue
        }

        // Get the appropriate channel
        channel, exists := channelManager.GetChannel(msg.Channel)
        if !exists {
            continue
        }

        // Send via channel
        channel.Send(ctx, msg)
    }
}
```

## Session Key Management

Session keys uniquely identify conversations:

```
Format: "channel:chatID" or "agent:agentID:channel:chatID"

Examples:
- "telegram:123456789"           # Telegram private chat
- "telegram:-1001234567890"      # Telegram group
- "discord:123456789012345678"   # Discord channel
- "agent:main:telegram:123456"   # Agent-specific session
```

## Async Tool Flow

Some tools (like spawn) operate asynchronously:

```
1. Tool receives Execute() call
2. Tool returns AsyncResult immediately
3. Tool starts work in goroutine
4. When complete, tool calls AsyncCallback
5. Callback publishes result via PublishInbound
6. AgentLoop processes result in next iteration
```

## Error Handling

Errors are handled at multiple levels:

1. **Channel level**: Connection errors, API errors
2. **Bus level**: Full buffer, closed channel
3. **Agent level**: LLM errors, tool errors
4. **Session level**: Save/load errors

Error responses are sent to users when appropriate:

```go
if err != nil {
    response = fmt.Sprintf("Error processing message: %v", err)
}
```

## Graceful Shutdown

The message bus supports graceful shutdown:

```go
// Stop accepting new messages
mb.Close()

// AgentLoop checks running flag
for al.running.Load() {
    // ...
}

// Context cancellation for in-flight operations
ctx, cancel := context.WithCancel(context.Background())
// ... on shutdown ...
cancel()
```

## Performance Characteristics

- **Buffer capacity**: 100 messages per channel
- **Concurrency**: Multiple channels can publish simultaneously
- **Thread safety**: All operations are thread-safe via mutex
- **Memory**: Messages are held in memory until processed

## Best Practices

1. **Always check context cancellation** in long-running operations
2. **Use timeouts** for external API calls
3. **Handle full buffers** gracefully
4. **Log message processing** for debugging
5. **Don't block** the message bus with slow operations

# Message Bus API Reference

This document provides detailed reference for the Message Bus API.

## Overview

The Message Bus is the central communication hub in PicoClaw. It enables asynchronous communication between channels and the agent loop.

## Types

### MessageBus

The main message bus structure.

```go
type MessageBus struct {
    inbound  chan InboundMessage
    outbound chan OutboundMessage
    handlers map[string]MessageHandler
    closed   bool
    mu       sync.RWMutex
}
```

### InboundMessage

Message from an external platform.

```go
type InboundMessage struct {
    Channel    string            `json:"channel"`
    SenderID   string            `json:"sender_id"`
    ChatID     string            `json:"chat_id"`
    Content    string            `json:"content"`
    Media      []string          `json:"media,omitempty"`
    SessionKey string            `json:"session_key"`
    Metadata   map[string]string `json:"metadata,omitempty"`
}
```

**Fields:**

| Field | Type | Description |
|-------|------|-------------|
| Channel | string | Platform identifier (e.g., "telegram", "discord") |
| SenderID | string | User identifier on the platform |
| ChatID | string | Chat/conversation identifier |
| Content | string | Message text content |
| Media | []string | Optional media URLs or paths |
| SessionKey | string | Optional pre-computed session key |
| Metadata | map[string]string | Platform-specific metadata |

**Common Metadata Keys:**

| Key | Description |
|-----|-------------|
| message_id | Platform message ID |
| reply_to_id | ID of message being replied to |
| peer_kind | "direct", "group", "channel" |
| peer_id | Peer identifier |
| account_id | Bot account identifier |
| guild_id | Discord guild ID |
| team_id | Slack team ID |

### OutboundMessage

Message to be sent to a platform.

```go
type OutboundMessage struct {
    Channel string `json:"channel"`
    ChatID  string `json:"chat_id"`
    Content string `json:"content"`
}
```

**Fields:**

| Field | Type | Description |
|-------|------|-------------|
| Channel | string | Target platform |
| ChatID | string | Target chat/conversation |
| Content | string | Message content |

### MessageHandler

Function type for handling inbound messages.

```go
type MessageHandler func(InboundMessage) error
```

## Functions

### NewMessageBus

Creates a new message bus.

```go
func NewMessageBus() *MessageBus
```

**Returns:** New MessageBus with buffered channels (capacity: 100)

```go
bus := bus.NewMessageBus()
```

## Methods

### PublishInbound

Publishes an inbound message to the bus.

```go
func (mb *MessageBus) PublishInbound(msg InboundMessage)
```

**Parameters:**
- `msg`: Message to publish

**Behavior:**
- Non-blocking if channel has capacity
- Silently drops message if bus is closed

```go
bus.PublishInbound(bus.InboundMessage{
    Channel:  "telegram",
    SenderID: "123456",
    ChatID:   "123456",
    Content:  "Hello!",
})
```

### ConsumeInbound

Consumes an inbound message from the bus.

```go
func (mb *MessageBus) ConsumeInbound(ctx context.Context) (InboundMessage, bool)
```

**Parameters:**
- `ctx`: Context for cancellation

**Returns:**
- `InboundMessage`: The consumed message
- `bool`: True if message was received, false if context cancelled

```go
msg, ok := bus.ConsumeInbound(ctx)
if !ok {
    // Context cancelled
    return
}
// Process msg
```

### PublishOutbound

Publishes an outbound message to the bus.

```go
func (mb *MessageBus) PublishOutbound(msg OutboundMessage)
```

**Parameters:**
- `msg`: Message to publish

```go
bus.PublishOutbound(bus.OutboundMessage{
    Channel: "telegram",
    ChatID:  "123456",
    Content: "Hi there!",
})
```

### SubscribeOutbound

Subscribes to outbound messages.

```go
func (mb *MessageBus) SubscribeOutbound(ctx context.Context) (OutboundMessage, bool)
```

**Parameters:**
- `ctx`: Context for cancellation

**Returns:**
- `OutboundMessage`: The outbound message
- `bool`: True if message was received, false if context cancelled

```go
for {
    msg, ok := bus.SubscribeOutbound(ctx)
    if !ok {
        break // Context cancelled
    }

    // Send msg to platform
    channel.Send(ctx, msg)
}
```

### RegisterHandler

Registers a handler for a channel.

```go
func (mb *MessageBus) RegisterHandler(channel string, handler MessageHandler)
```

**Parameters:**
- `channel`: Channel name
- `handler`: Handler function

```go
bus.RegisterHandler("telegram", func(msg bus.InboundMessage) error {
    // Handle message
    return nil
})
```

### GetHandler

Retrieves a handler for a channel.

```go
func (mb *MessageBus) GetHandler(channel string) (MessageHandler, bool)
```

**Parameters:**
- `channel`: Channel name

**Returns:**
- `MessageHandler`: The registered handler
- `bool`: True if handler exists

```go
handler, ok := bus.GetHandler("telegram")
if ok {
    handler(msg)
}
```

### Close

Closes the message bus.

```go
func (mb *MessageBus) Close()
```

**Behavior:**
- Closes inbound and outbound channels
- Prevents new messages from being published
- Safe to call multiple times

```go
bus.Close()
```

## Usage Patterns

### Producer (Channel)

Channels publish inbound messages:

```go
type TelegramChannel struct {
    bus *bus.MessageBus
}

func (c *TelegramChannel) handleUpdate(update Update) {
    c.bus.PublishInbound(bus.InboundMessage{
        Channel:  "telegram",
        SenderID: strconv.FormatInt(update.Message.From.ID, 10),
        ChatID:   strconv.FormatInt(update.Message.Chat.ID, 10),
        Content:  update.Message.Text,
        Metadata: map[string]string{
            "message_id": strconv.FormatInt(update.Message.MessageID, 10),
        },
    })
}
```

### Consumer (Agent Loop)

The agent loop consumes inbound messages:

```go
func (al *AgentLoop) Run(ctx context.Context) error {
    for al.running.Load() {
        select {
        case <-ctx.Done():
            return nil
        default:
            msg, ok := al.bus.ConsumeInbound(ctx)
            if !ok {
                continue
            }

            response, err := al.processMessage(ctx, msg)
            if err != nil {
                response = fmt.Sprintf("Error: %v", err)
            }

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

### Subscriber (Gateway)

Gateways subscribe to outbound messages:

```go
func (g *Gateway) runOutboundLoop(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            return
        default:
            msg, ok := g.bus.SubscribeOutbound(ctx)
            if !ok {
                continue
            }

            channel, ok := g.channels.GetChannel(msg.Channel)
            if !ok {
                continue
            }

            if err := channel.Send(ctx, msg); err != nil {
                log.Printf("Failed to send: %v", err)
            }
        }
    }
}
```

## Thread Safety

The MessageBus is thread-safe:

- All operations are protected by mutex
- Multiple goroutines can publish simultaneously
- Multiple goroutines can consume simultaneously

```go
// Safe to call from multiple goroutines
go func() {
    bus.PublishInbound(msg1)
}()

go func() {
    bus.PublishInbound(msg2)
}()
```

## Buffer Behavior

Channels are buffered with capacity 100:

- Publish is non-blocking until buffer is full
- When buffer is full, publish blocks until space available
- This provides backpressure to prevent memory issues

## Error Handling

### Closed Bus

Operations on a closed bus are safe:

```go
bus.Close()

// Safe - message is dropped
bus.PublishInbound(msg) // No-op

// Safe - returns empty message and false
msg, ok := bus.ConsumeInbound(ctx) // ok == false
```

### Context Cancellation

Consume/Subscribe respect context:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

msg, ok := bus.ConsumeInbound(ctx)
if !ok {
    // Timed out or cancelled
}
```

## Best Practices

### Publishing

1. **Check Context**: Respect context in publishers
2. **Handle Full Buffer**: Be prepared for blocking
3. **Don't Block**: Don't hold locks while publishing

```go
func (c *Channel) publishAsync(msg bus.InboundMessage) {
    go func() {
        select {
        case <-c.ctx.Done():
            return
        default:
            c.bus.PublishInbound(msg)
        }
    }()
}
```

### Consuming

1. **Use Context**: Always use context for cancellation
2. **Check OK**: Always check the ok return value
3. **Handle Errors**: Process errors gracefully

```go
for {
    msg, ok := bus.ConsumeInbound(ctx)
    if !ok {
        if ctx.Err() != nil {
            // Context cancelled
            return ctx.Err()
        }
        // Bus closed
        return nil
    }

    if err := process(msg); err != nil {
        // Handle error
    }
}
```

### Graceful Shutdown

1. **Stop Publishing**: Stop accepting new messages
2. **Drain Queues**: Process remaining messages
3. **Close Bus**: Close when done

```go
// Stop accepting new messages
stopChannels()

// Drain with timeout
drainCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
for {
    msg, ok := bus.ConsumeInbound(drainCtx)
    if !ok {
        break
    }
    process(msg)
}
cancel()

// Close bus
bus.Close()
```

## Performance Characteristics

| Metric | Value |
|--------|-------|
| Buffer Capacity | 100 messages per channel |
| Memory per Message | ~200 bytes (approximate) |
| Thread Safety | Yes (mutex protected) |
| Blocking | Yes (when buffer full) |

## See Also

- [Data Flow](../data-flow.md)
- [Creating Channels](../extending/creating-channels.md)
- [Bus Implementation](https://github.com/sipeed/picoclaw/tree/main/pkg/bus)

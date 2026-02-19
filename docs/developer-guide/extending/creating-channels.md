# Creating Channel Integrations

This guide explains how to create channel integrations for PicoClaw.

## Overview

Channels connect PicoClaw to external chat platforms like Telegram, Discord, Slack, etc. Each channel implements the `Channel` interface and is responsible for:

1. Receiving messages from the platform
2. Publishing them to the message bus
3. Subscribing to outbound messages
4. Sending responses to the platform

## Channel Interface

All channels must implement this interface:

```go
type Channel interface {
    Name() string
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    Send(ctx context.Context, msg bus.OutboundMessage) error
    IsRunning() bool
    IsAllowed(senderID string) bool
}
```

## Base Channel

PicoClaw provides a `BaseChannel` struct that handles common functionality:

```go
type BaseChannel struct {
    config    interface{}
    bus       *bus.MessageBus
    running   bool
    name      string
    allowList []string
}

// HandleMessage publishes a message to the bus
func (c *BaseChannel) HandleMessage(senderID, chatID, content string,
                                    media []string, metadata map[string]string)

// IsAllowed checks if sender is in the allowlist
func (c *BaseChannel) IsAllowed(senderID string) bool

// Name returns the channel name
func (c *BaseChannel) Name() string

// IsRunning returns the running state
func (c *BaseChannel) IsRunning() bool
```

## Creating a Basic Channel

### Step 1: Define the Channel Struct

```go
package channels

import (
    "context"
    "fmt"
    "sync"

    "github.com/sipeed/picoclaw/pkg/bus"
)

// MyChannelConfig holds channel configuration
type MyChannelConfig struct {
    APIToken  string   `json:"api_token"`
    AllowList []string `json:"allow_list"`
}

// MyChannel implements the Channel interface
type MyChannel struct {
    *BaseChannel          // Embed base functionality
    config       *MyChannelConfig
    client       *MyAPIClient
    mu           sync.Mutex
}
```

### Step 2: Create Constructor

```go
func NewMyChannel(cfg *MyChannelConfig, msgBus *bus.MessageBus) *MyChannel {
    return &MyChannel{
        BaseChannel: NewBaseChannel("mychannel", cfg, msgBus, cfg.AllowList),
        config:      cfg,
        client:      NewMyAPIClient(cfg.APIToken),
    }
}
```

### Step 3: Implement Start

```go
func (c *MyChannel) Start(ctx context.Context) error {
    c.mu.Lock()
    defer c.mu.Unlock()

    if c.running {
        return nil
    }

    // Start receiving messages from the platform
    go c.receiveLoop(ctx)

    c.setRunning(true)
    return nil
}

func (c *MyChannel) receiveLoop(ctx context.Context) {
    // Subscribe to platform updates
    updates, err := c.client.Subscribe(ctx)
    if err != nil {
        // Handle error
        return
    }

    for {
        select {
        case <-ctx.Done():
            return
        case update, ok := <-updates:
            if !ok {
                return
            }
            c.handleUpdate(update)
        }
    }
}
```

### Step 4: Implement Stop

```go
func (c *MyChannel) Stop(ctx context.Context) error {
    c.mu.Lock()
    defer c.mu.Unlock()

    if !c.running {
        return nil
    }

    // Close client connections
    if err := c.client.Close(); err != nil {
        return fmt.Errorf("failed to close client: %w", err)
    }

    c.setRunning(false)
    return nil
}
```

### Step 5: Implement Send

```go
func (c *MyChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
    if !c.running {
        return fmt.Errorf("channel not running")
    }

    // Send message through platform API
    return c.client.SendMessage(ctx, msg.ChatID, msg.Content)
}
```

### Step 6: Handle Incoming Messages

```go
func (c *MyChannel) handleUpdate(update MyUpdate) {
    // Extract message details
    senderID := update.Sender.ID
    chatID := update.Chat.ID
    content := update.Message.Text

    // Collect media if present
    var media []string
    if update.Message.Attachment != "" {
        media = append(media, update.Message.Attachment)
    }

    // Build metadata
    metadata := map[string]string{
        "message_id": update.ID,
        "timestamp":  update.Timestamp,
    }

    // Publish to message bus
    c.HandleMessage(senderID, chatID, content, media, metadata)
}
```

## Complete Example: WebSocket Channel

```go
package channels

import (
    "context"
    "encoding/json"
    "fmt"
    "sync"

    "github.com/gorilla/websocket"
    "github.com/sipeed/picoclaw/pkg/bus"
)

type WebSocketConfig struct {
    URL       string   `json:"url"`
    Token     string   `json:"token"`
    AllowList []string `json:"allow_list"`
}

type WebSocketChannel struct {
    *BaseChannel
    config  *WebSocketConfig
    conn    *websocket.Conn
    mu      sync.Mutex
    sendCh  chan bus.OutboundMessage
}

func NewWebSocketChannel(cfg *WebSocketConfig, msgBus *bus.MessageBus) *WebSocketChannel {
    return &WebSocketChannel{
        BaseChannel: NewBaseChannel("websocket", cfg, msgBus, cfg.AllowList),
        config:      cfg,
        sendCh:      make(chan bus.OutboundMessage, 100),
    }
}

func (c *WebSocketChannel) Start(ctx context.Context) error {
    c.mu.Lock()
    defer c.mu.Unlock()

    if c.running {
        return nil
    }

    // Connect to WebSocket server
    headers := make(map[string][]string)
    if c.config.Token != "" {
        headers["Authorization"] = []string{"Bearer " + c.config.Token}
    }

    conn, _, err := websocket.DefaultDialer.Dial(c.config.URL, headers)
    if err != nil {
        return fmt.Errorf("failed to connect: %w", err)
    }

    c.conn = conn

    // Start receive goroutine
    go c.receiveLoop(ctx)

    // Start send goroutine
    go c.sendLoop(ctx)

    c.setRunning(true)
    return nil
}

func (c *WebSocketChannel) Stop(ctx context.Context) error {
    c.mu.Lock()
    defer c.mu.Unlock()

    if !c.running {
        return nil
    }

    // Close send channel
    close(c.sendCh)

    // Close connection
    if c.conn != nil {
        c.conn.Close()
    }

    c.setRunning(false)
    return nil
}

func (c *WebSocketChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
    if !c.running {
        return fmt.Errorf("channel not running")
    }

    select {
    case c.sendCh <- msg:
        return nil
    default:
        return fmt.Errorf("send buffer full")
    }
}

func (c *WebSocketChannel) receiveLoop(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            return
        default:
            _, message, err := c.conn.ReadMessage()
            if err != nil {
                // Handle error - reconnect?
                return
            }

            c.handleMessage(message)
        }
    }
}

func (c *WebSocketChannel) sendLoop(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            return
        case msg, ok := <-c.sendCh:
            if !ok {
                return
            }

            // Create WebSocket message
            wsMsg := map[string]interface{}{
                "chat_id": msg.ChatID,
                "content": msg.Content,
            }

            data, _ := json.Marshal(wsMsg)
            if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
                // Handle error
            }
        }
    }
}

func (c *WebSocketChannel) handleMessage(data []byte) {
    var msg struct {
        SenderID string `json:"sender_id"`
        ChatID   string `json:"chat_id"`
        Content  string `json:"content"`
    }

    if err := json.Unmarshal(data, &msg); err != nil {
        return
    }

    // Publish to bus
    c.HandleMessage(msg.SenderID, msg.ChatID, msg.Content, nil, nil)
}
```

## Registering Channels

Channels are registered in the channel manager:

```go
// In gateway setup
channelManager := channels.NewManager()

// Add your channel
myChannel := channels.NewMyChannel(&config.Channels.MyChannel, msgBus)
channelManager.Register(myChannel)

// Start all channels
channelManager.StartAll(ctx)
```

## Handling Rich Media

### Receiving Media

```go
func (c *MyChannel) handleUpdate(update MyUpdate) {
    var media []string

    // Collect images
    for _, img := range update.Message.Images {
        media = append(media, img.URL)
    }

    // Collect files
    for _, file := range update.Message.Files {
        media = append(media, file.URL)
    }

    c.HandleMessage(senderID, chatID, content, media, metadata)
}
```

### Sending Media

Some platforms support rich message formatting:

```go
func (c *MyChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
    // Check for markdown support
    if supportsMarkdown(msg.Content) {
        return c.client.SendMarkdown(ctx, msg.ChatID, msg.Content)
    }

    // Plain text fallback
    return c.client.SendMessage(ctx, msg.ChatID, msg.Content)
}
```

## Metadata

Use metadata to pass platform-specific information:

```go
metadata := map[string]string{
    "message_id":     update.ID,
    "reply_to_id":    update.ReplyToID,
    "peer_kind":      "direct", // or "group", "channel"
    "peer_id":        update.Chat.ID,
    "account_id":     c.accountID,
    "guild_id":       update.GuildID,   // For Discord-like platforms
    "team_id":        update.TeamID,    // For Slack-like platforms
}

c.HandleMessage(senderID, chatID, content, media, metadata)
```

This metadata is used for:
- Message routing
- Reply threading
- Multi-account support

## Configuration

Add your channel configuration to the config structure:

```go
// In pkg/config/config.go
type ChannelsConfig struct {
    Telegram  TelegramConfig  `json:"telegram"`
    Discord   DiscordConfig   `json:"discord"`
    MyChannel MyChannelConfig `json:"mychannel"`
    // ...
}
```

Example configuration:

```json
{
  "channels": {
    "mychannel": {
      "api_token": "your-token-here",
      "allow_list": ["user1", "user2"]
    }
  }
}
```

## Error Handling

Handle errors gracefully:

```go
func (c *MyChannel) receiveLoop(ctx context.Context) {
    defer func() {
        if r := recover(); r != nil {
            // Log panic and attempt recovery
        }
    }()

    for {
        select {
        case <-ctx.Done():
            return
        default:
            if err := c.receiveMessage(ctx); err != nil {
                // Log error
                if isConnectionError(err) {
                    // Attempt reconnection
                    c.reconnect(ctx)
                }
            }
        }
    }
}
```

## Testing Channels

```go
package channels

import (
    "context"
    "testing"

    "github.com/sipeed/picoclaw/pkg/bus"
)

func TestMyChannel(t *testing.T) {
    // Create message bus
    msgBus := bus.NewMessageBus()

    // Create channel
    cfg := &MyChannelConfig{
        APIToken: "test-token",
    }
    channel := NewMyChannel(cfg, msgBus)

    // Test name
    if channel.Name() != "mychannel" {
        t.Errorf("expected name 'mychannel', got %s", channel.Name())
    }

    // Test start/stop
    ctx := context.Background()
    if err := channel.Start(ctx); err != nil {
        t.Fatalf("failed to start: %v", err)
    }

    if !channel.IsRunning() {
        t.Error("channel should be running")
    }

    if err := channel.Stop(ctx); err != nil {
        t.Fatalf("failed to stop: %v", err)
    }

    if channel.IsRunning() {
        t.Error("channel should not be running")
    }
}

func TestAllowList(t *testing.T) {
    msgBus := bus.NewMessageBus()
    cfg := &MyChannelConfig{
        APIToken:  "test-token",
        AllowList: []string{"user1", "user2"},
    }
    channel := NewMyChannel(cfg, msgBus)

    // Test allowed users
    if !channel.IsAllowed("user1") {
        t.Error("user1 should be allowed")
    }

    // Test disallowed users
    if channel.IsAllowed("user3") {
        t.Error("user3 should not be allowed")
    }
}
```

## Best Practices

1. **Graceful Shutdown**: Handle context cancellation properly
2. **Reconnection**: Implement automatic reconnection for network issues
3. **Rate Limiting**: Respect platform rate limits
4. **Error Logging**: Log errors for debugging
5. **Buffer Sizes**: Use appropriate buffer sizes for channels
6. **Thread Safety**: Use mutexes for shared state
7. **Backpressure**: Handle slow consumers gracefully

## See Also

- [Message Bus API](../api/message-bus.md)
- [Existing Channels](https://github.com/sipeed/picoclaw/tree/main/pkg/channels)
- [Telegram Channel](https://github.com/sipeed/picoclaw/tree/main/pkg/channels/telegram.go)

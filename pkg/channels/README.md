# PicoClaw Channel System Refactor: Complete Development Guide

> **Branch**: `refactor/channel-system`
> **Status**: Active development (~40 commits)
> **Scope**: `pkg/channels/`, `pkg/bus/`, `pkg/media/`, `pkg/identity/`, `cmd/picoclaw/internal/gateway/`

---

## Table of Contents

- [Part 1: Architecture Overview](#part-1-architecture-overview)
- [Part 2: Migration Guide — From main Branch to Refactored Branch](#part-2-migration-guide--from-main-branch-to-refactored-branch)
- [Part 3: New Channel Development Guide — Implementing a Channel from Scratch](#part-3-new-channel-development-guide--implementing-a-channel-from-scratch)
- [Part 4: Core Subsystem Details](#part-4-core-subsystem-details)
- [Part 5: Key Design Decisions and Conventions](#part-5-key-design-decisions-and-conventions)
- [Appendix: Complete File Listing and Interface Quick Reference](#appendix-complete-file-listing-and-interface-quick-reference)

---

## Part 1: Architecture Overview

### 1.1 Before and After Comparison

**Before Refactor (main branch)**:

```
pkg/channels/
├── telegram.go          # Each channel directly in the channels package
├── discord.go
├── slack.go
├── manager.go           # Manager directly references each channel type
├── ...
```

- All channel implementations lived at the top level of `pkg/channels/`
- Manager constructed each channel via `switch` or `if-else` chains
- Routing info like Peer and MessageID was buried in `Metadata map[string]string`
- No rate limiting or retry on message sending
- No unified media file lifecycle management
- Each channel ran its own HTTP server
- Group chat trigger filtering logic was scattered across channels

**After Refactor (refactor/channel-system branch)**:

```
pkg/channels/
├── base.go              # BaseChannel shared abstraction layer
├── interfaces.go        # Optional capability interfaces (TypingCapable, MessageEditor, ReactionCapable, PlaceholderCapable, PlaceholderRecorder)
├── media.go             # MediaSender optional interface
├── webhook.go           # WebhookHandler, HealthChecker optional interfaces
├── errors.go            # Sentinel errors (ErrNotRunning, ErrRateLimit, ErrTemporary, ErrSendFailed)
├── errutil.go           # Error classification helpers
├── registry.go          # Factory registry (RegisterFactory / getFactory)
├── manager.go           # Unified orchestration: Worker queues, rate limiting, retries, Typing/Placeholder, shared HTTP
├── split.go             # Smart long-message splitting (preserves code block integrity)
├── telegram/            # Each channel in its own sub-package
│   ├── init.go          # Factory registration
│   ├── telegram.go      # Implementation
│   └── telegram_commands.go
├── discord/
│   ├── init.go
│   └── discord.go
├── slack/ line/ onebot/ dingtalk/ feishu/ wecom/ qq/ whatsapp/ maixcam/ pico/
│   └── ...

pkg/bus/
├── bus.go               # MessageBus (buffer 64, safe close + drain)
├── types.go             # Structured message types (Peer, SenderInfo, MediaPart, InboundMessage, OutboundMessage, OutboundMediaMessage)

pkg/media/
├── store.go             # MediaStore interface + FileMediaStore implementation (two-phase release, TTL cleanup)

pkg/identity/
├── identity.go          # Unified user identity: canonical "platform:id" format + backward-compatible matching
```

### 1.2 Message Flow Overview

```
┌────────────┐      InboundMessage       ┌───────────┐      LLM + Tools      ┌────────────┐
│  Telegram   │──┐                        │           │                        │            │
│  Discord    │──┤   PublishInbound()     │           │   PublishOutbound()   │            │
│  Slack      │──┼──────────────────────▶ │ MessageBus │ ◀─────────────────── │ AgentLoop  │
│  LINE       │──┤   (buffered chan, 64)  │           │   (buffered chan, 64) │            │
│  ...        │──┘                        │           │                        │            │
└────────────┘                            └─────┬─────┘                        └────────────┘
                                                │
                            SubscribeOutbound() │  SubscribeOutboundMedia()
                                                ▼
                                    ┌───────────────────┐
                                    │   Manager          │
                                    │   ├── dispatchOutbound()    Route to Worker queues
                                    │   ├── dispatchOutboundMedia()
                                    │   ├── runWorker()           Message split + sendWithRetry()
                                    │   ├── runMediaWorker()      sendMediaWithRetry()
                                    │   ├── preSend()             Stop Typing + Undo Reaction + Edit Placeholder
                                    │   └── runTTLJanitor()       Clean up expired Typing/Placeholder
                                    └────────┬──────────┘
                                             │
                                   channel.Send() / SendMedia()
                                             │
                                             ▼
                                    ┌────────────────┐
                                    │ Platform APIs   │
                                    └────────────────┘
```

### 1.3 Key Design Principles

| Principle | Description |
|-----------|-------------|
| **Sub-package Isolation** | Each channel is a standalone Go sub-package, depending on `BaseChannel` and interfaces from the `channels` parent package |
| **Factory Registration** | Sub-packages self-register via `init()`, Manager looks up factories by name, eliminating import coupling |
| **Capability Discovery** | Optional capabilities are declared via interfaces (`MediaSender`, `TypingCapable`, `ReactionCapable`, `PlaceholderCapable`, `MessageEditor`, `WebhookHandler`), discovered by Manager via runtime type assertions |
| **Structured Messages** | Peer, MessageID, and SenderInfo promoted from Metadata to first-class fields on InboundMessage |
| **Error Classification** | Channels return sentinel errors (`ErrRateLimit`, `ErrTemporary`, etc.), Manager uses these to determine retry strategy |
| **Centralized Orchestration** | Rate limiting, message splitting, retries, and Typing/Reaction/Placeholder management are all handled by Manager and BaseChannel; channels only need to implement Send |

---

## Part 2: Migration Guide — From main Branch to Refactored Branch

### 2.1 If You Have Unmerged Channel Changes

#### Step 1: Identify which files you modified

On the main branch, channel files were directly in `pkg/channels/` top level, e.g.:
- `pkg/channels/telegram.go`
- `pkg/channels/discord.go`

After refactoring, these files have been removed and code moved to corresponding sub-packages:
- `pkg/channels/telegram/telegram.go`
- `pkg/channels/discord/discord.go`

#### Step 2: Understand the structural change mapping

| main branch file | Refactored branch location | Changes |
|---|---|---|
| `pkg/channels/telegram.go` | `pkg/channels/telegram/telegram.go` + `init.go` | Package name changed from `channels` to `telegram` |
| `pkg/channels/discord.go` | `pkg/channels/discord/discord.go` + `init.go` | Same as above |
| `pkg/channels/manager.go` | `pkg/channels/manager.go` | Extensively rewritten |
| _(did not exist)_ | `pkg/channels/base.go` | New shared abstraction layer |
| _(did not exist)_ | `pkg/channels/registry.go` | New factory registry |
| _(did not exist)_ | `pkg/channels/errors.go` + `errutil.go` | New error classification system |
| _(did not exist)_ | `pkg/channels/interfaces.go` | New optional capability interfaces |
| _(did not exist)_ | `pkg/channels/media.go` | New MediaSender interface |
| _(did not exist)_ | `pkg/channels/webhook.go` | New WebhookHandler/HealthChecker |
| _(did not exist)_ | `pkg/channels/split.go` | New message splitting (migrated from utils) |
| _(did not exist)_ | `pkg/bus/types.go` | New structured message types |
| _(did not exist)_ | `pkg/media/store.go` | New media file lifecycle management |
| _(did not exist)_ | `pkg/identity/identity.go` | New unified user identity |

#### Step 3: Migrate your channel code

Using Telegram as an example, the main changes are:

**3a. Package declaration and imports**

```go
// Old code (main branch)
package channels

import (
    "github.com/sipeed/picoclaw/pkg/bus"
    "github.com/sipeed/picoclaw/pkg/config"
)

// New code (refactored branch)
package telegram

import (
    "github.com/sipeed/picoclaw/pkg/bus"
    "github.com/sipeed/picoclaw/pkg/channels"     // Reference parent package
    "github.com/sipeed/picoclaw/pkg/config"
    "github.com/sipeed/picoclaw/pkg/identity"      // New
    "github.com/sipeed/picoclaw/pkg/media"          // New (if media support needed)
)
```

**3b. Struct embeds BaseChannel**

```go
// Old code: directly held bus, config, etc. fields
type TelegramChannel struct {
    bus       *bus.MessageBus
    config    *config.Config
    running   bool
    allowList []string
    // ...
}

// New code: embed BaseChannel, which provides bus, running, allowList, etc.
type TelegramChannel struct {
    *channels.BaseChannel          // Embed shared abstraction
    bot    *telego.Bot
    config *config.Config
    // ... only channel-specific fields
}
```

**3c. Constructor**

```go
// Old code: direct assignment
func NewTelegramChannel(cfg *config.Config, bus *bus.MessageBus) (*TelegramChannel, error) {
    return &TelegramChannel{
        bus:       bus,
        config:    cfg,
        allowList: cfg.Channels.Telegram.AllowFrom,
        // ...
    }, nil
}

// New code: use NewBaseChannel + functional options
func NewTelegramChannel(cfg *config.Config, bus *bus.MessageBus) (*TelegramChannel, error) {
    base := channels.NewBaseChannel(
        "telegram",                    // Name
        cfg.Channels.Telegram,         // Raw config (any type)
        bus,                           // Message bus
        cfg.Channels.Telegram.AllowFrom, // Allow list
        channels.WithMaxMessageLength(4096),                     // Platform message length limit
        channels.WithGroupTrigger(cfg.Channels.Telegram.GroupTrigger), // Group trigger config
    )
    return &TelegramChannel{
        BaseChannel: base,
        bot:         bot,
        config:      cfg,
    }, nil
}
```

**3d. Start/Stop lifecycle**

```go
// New code: use SetRunning atomic operation
func (c *TelegramChannel) Start(ctx context.Context) error {
    // ... initialize bot, webhook, etc.
    c.SetRunning(true)    // Must be called after ready
    go bh.Start()
    return nil
}

func (c *TelegramChannel) Stop(ctx context.Context) error {
    c.SetRunning(false)   // Must be called before cleanup
    // ... stop bot handler, cancel context
    return nil
}
```

**3e. Send method error returns**

```go
// Old code: returns plain error
func (c *TelegramChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
    if !c.running { return fmt.Errorf("not running") }
    // ...
    if err != nil { return err }
}

// New code: must return sentinel errors for Manager to determine retry strategy
func (c *TelegramChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
    if !c.IsRunning() {
        return channels.ErrNotRunning    // ← Manager will not retry
    }
    // ...
    if err != nil {
        // Use ClassifySendError to wrap error based on HTTP status code
        return channels.ClassifySendError(statusCode, err)
        // Or manually wrap:
        // return fmt.Errorf("%w: %v", channels.ErrTemporary, err)
        // return fmt.Errorf("%w: %v", channels.ErrRateLimit, err)
        // return fmt.Errorf("%w: %v", channels.ErrSendFailed, err)
    }
    return nil
}
```

**3f. Message reception (Inbound)**

```go
// Old code: directly construct InboundMessage and publish
msg := bus.InboundMessage{
    Channel:  "telegram",
    SenderID: senderID,
    ChatID:   chatID,
    Content:  content,
    Metadata: map[string]string{
        "peer_kind": "group",     // Routing info buried in metadata
        "peer_id":   chatID,
        "message_id": msgID,
    },
}
c.bus.PublishInbound(ctx, msg)

// New code: use BaseChannel.HandleMessage with structured fields
sender := bus.SenderInfo{
    Platform:    "telegram",
    PlatformID:  strconv.FormatInt(from.ID, 10),
    CanonicalID: identity.BuildCanonicalID("telegram", strconv.FormatInt(from.ID, 10)),
    Username:    from.Username,
    DisplayName: from.FirstName,
}

peer := bus.Peer{
    Kind: "group",    // or "direct"
    ID:   chatID,
}

// HandleMessage internally calls IsAllowedSender for permission checks, builds MediaScope, and publishes to bus
c.HandleMessage(ctx, peer, messageID, senderID, chatID, content, mediaRefs, metadata, sender)
```

**3g. Add factory registration (required)**

Create `init.go` for your channel:

```go
// pkg/channels/telegram/init.go
package telegram

import (
    "github.com/sipeed/picoclaw/pkg/bus"
    "github.com/sipeed/picoclaw/pkg/channels"
    "github.com/sipeed/picoclaw/pkg/config"
)

func init() {
    channels.RegisterFactory("telegram", func(cfg *config.Config, b *bus.MessageBus) (channels.Channel, error) {
        return NewTelegramChannel(cfg, b)
    })
}
```

**3h. Import sub-package in Gateway**

```go
// cmd/picoclaw/internal/gateway/helpers.go
import (
    _ "github.com/sipeed/picoclaw/pkg/channels/telegram"   // Triggers init() registration
    _ "github.com/sipeed/picoclaw/pkg/channels/discord"
    _ "github.com/sipeed/picoclaw/pkg/channels/your_new_channel"  // New addition
)
```

#### Step 4: Migrate bus message usage

If your code directly reads routing fields from `InboundMessage.Metadata`:

```go
// Old code
peerKind := msg.Metadata["peer_kind"]
peerID   := msg.Metadata["peer_id"]
msgID    := msg.Metadata["message_id"]

// New code
peerKind := msg.Peer.Kind      // First-class field
peerID   := msg.Peer.ID        // First-class field
msgID    := msg.MessageID       // First-class field
sender   := msg.Sender          // bus.SenderInfo struct
scope    := msg.MediaScope       // Media lifecycle scope
```

#### Step 5: Migrate allow-list checks

```go
// Old code
if !c.isAllowed(senderID) { return }

// New code: prefer structured check
if !c.IsAllowedSender(sender) { return }
// Or fall back to string check:
if !c.IsAllowed(senderID) { return }
```

`BaseChannel.HandleMessage` already handles this logic internally — no need to duplicate the check in your channel.

### 2.2 If You Have Manager Modifications

The Manager has been completely rewritten. Your modifications will need to account for the new architecture:

| Old Manager Responsibility | New Manager Responsibility |
|---|---|
| Directly construct channels (switch/if-else) | Look up and construct via factory registry |
| Directly call channel.Send | Per-channel Worker queues + rate limiting + retries |
| No message splitting | Automatic splitting based on MaxMessageLength |
| Each channel runs its own HTTP server | Unified shared HTTP server |
| No Typing/Placeholder management | Unified preSend handles Typing stop + Reaction undo + Placeholder edit; inbound-side BaseChannel.HandleMessage auto-orchestrates Typing/Reaction/Placeholder |
| No TTL cleanup | runTTLJanitor periodically cleans up expired Typing/Reaction/Placeholder entries |

### 2.3 If You Have Agent Loop Modifications

Main changes to the Agent Loop:

1. **MediaStore injection**: `agentLoop.SetMediaStore(mediaStore)` — Agent resolves media references produced by tools via MediaStore
2. **ChannelManager injection**: `agentLoop.SetChannelManager(channelManager)` — Agent can query channel state
3. **OutboundMediaMessage**: Agent now sends media messages via `bus.PublishOutboundMedia()` instead of embedding them in text replies
4. **extractPeer**: Routing uses `msg.Peer` structured fields instead of Metadata lookups

---

## Part 3: New Channel Development Guide — Implementing a Channel from Scratch

### 3.1 Minimum Implementation Checklist

To add a new chat platform (e.g., `matrix`), you need to:

1. ✅ Create sub-package directory `pkg/channels/matrix/`
2. ✅ Create `init.go` — factory registration
3. ✅ Create `matrix.go` — channel implementation
4. ✅ Add blank import in Gateway helpers
5. ✅ Add config check in Manager.initChannels()
6. ✅ Add config struct in `pkg/config/`

### 3.2 Complete Template

#### `pkg/channels/matrix/init.go`

```go
package matrix

import (
    "github.com/sipeed/picoclaw/pkg/bus"
    "github.com/sipeed/picoclaw/pkg/channels"
    "github.com/sipeed/picoclaw/pkg/config"
)

func init() {
    channels.RegisterFactory("matrix", func(cfg *config.Config, b *bus.MessageBus) (channels.Channel, error) {
        return NewMatrixChannel(cfg, b)
    })
}
```

#### `pkg/channels/matrix/matrix.go`

```go
package matrix

import (
    "context"
    "fmt"

    "github.com/sipeed/picoclaw/pkg/bus"
    "github.com/sipeed/picoclaw/pkg/channels"
    "github.com/sipeed/picoclaw/pkg/config"
    "github.com/sipeed/picoclaw/pkg/identity"
    "github.com/sipeed/picoclaw/pkg/logger"
)

// MatrixChannel implements channels.Channel for the Matrix protocol.
type MatrixChannel struct {
    *channels.BaseChannel            // Must embed
    config *config.Config
    ctx    context.Context
    cancel context.CancelFunc
    // ... Matrix SDK client, etc.
}

func NewMatrixChannel(cfg *config.Config, msgBus *bus.MessageBus) (*MatrixChannel, error) {
    matrixCfg := cfg.Channels.Matrix // Assumes this field exists in config

    base := channels.NewBaseChannel(
        "matrix",                           // Channel name (globally unique)
        matrixCfg,                          // Raw config
        msgBus,                             // Message bus
        matrixCfg.AllowFrom,                // Allow list
        channels.WithMaxMessageLength(65536), // Matrix message length limit
        channels.WithGroupTrigger(matrixCfg.GroupTrigger),
    )

    return &MatrixChannel{
        BaseChannel: base,
        config:      cfg,
    }, nil
}

// ========== Required Channel Interface Methods ==========

func (c *MatrixChannel) Start(ctx context.Context) error {
    c.ctx, c.cancel = context.WithCancel(ctx)

    // 1. Initialize Matrix client
    // 2. Start listening for messages
    // 3. Mark as running
    c.SetRunning(true)

    logger.InfoC("matrix", "Matrix channel started")
    return nil
}

func (c *MatrixChannel) Stop(ctx context.Context) error {
    c.SetRunning(false)

    if c.cancel != nil {
        c.cancel()
    }

    logger.InfoC("matrix", "Matrix channel stopped")
    return nil
}

func (c *MatrixChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
    // 1. Check running state
    if !c.IsRunning() {
        return channels.ErrNotRunning
    }

    // 2. Send message to Matrix
    err := c.sendToMatrix(ctx, msg.ChatID, msg.Content)
    if err != nil {
        // 3. Must use error classification wrapping
        //    If you have an HTTP status code:
        //    return channels.ClassifySendError(statusCode, err)
        //    If it's a network error:
        //    return channels.ClassifyNetError(err)
        //    If manual classification is needed:
        return fmt.Errorf("%w: %v", channels.ErrTemporary, err)
    }

    return nil
}

// ========== Incoming Message Handling ==========

func (c *MatrixChannel) handleIncoming(roomID, senderID, displayName, content string, msgID string) {
    // 1. Construct structured sender identity
    sender := bus.SenderInfo{
        Platform:    "matrix",
        PlatformID:  senderID,
        CanonicalID: identity.BuildCanonicalID("matrix", senderID),
        Username:    senderID,
        DisplayName: displayName,
    }

    // 2. Determine Peer type (direct vs group)
    peer := bus.Peer{
        Kind: "group",    // or "direct"
        ID:   roomID,
    }

    // 3. Group chat filtering (if applicable)
    isGroup := peer.Kind == "group"
    if isGroup {
        isMentioned := false // Detect @mentions based on platform specifics
        shouldRespond, cleanContent := c.ShouldRespondInGroup(isMentioned, content)
        if !shouldRespond {
            return
        }
        content = cleanContent
    }

    // 4. Handle media attachments (if any)
    var mediaRefs []string
    store := c.GetMediaStore()
    if store != nil {
        // Download attachment locally → store.Store() → get ref
        // mediaRefs = append(mediaRefs, ref)
    }

    // 5. Call HandleMessage to publish to bus
    //    HandleMessage internally will:
    //    - Check IsAllowedSender/IsAllowed
    //    - Build MediaScope
    //    - Publish InboundMessage
    c.HandleMessage(
        c.ctx,
        peer,
        msgID,                   // Platform message ID
        senderID,                // Raw sender ID
        roomID,                  // Chat/room ID
        content,                 // Message content
        mediaRefs,               // Media reference list
        nil,                     // Extra metadata (usually nil)
        sender,                  // SenderInfo (variadic parameter)
    )
}

// ========== Internal Methods ==========

func (c *MatrixChannel) sendToMatrix(ctx context.Context, roomID, content string) error {
    // Actual Matrix SDK call
    return nil
}
```

### 3.3 Optional Capability Interfaces

Depending on platform capabilities, your channel can optionally implement the following interfaces:

#### MediaSender — Send Media Attachments

```go
// If the platform supports sending images/files/audio/video
func (c *MatrixChannel) SendMedia(ctx context.Context, msg bus.OutboundMediaMessage) error {
    if !c.IsRunning() {
        return channels.ErrNotRunning
    }

    store := c.GetMediaStore()
    if store == nil {
        return fmt.Errorf("no media store: %w", channels.ErrSendFailed)
    }

    for _, part := range msg.Parts {
        localPath, err := store.Resolve(part.Ref)
        if err != nil {
            logger.ErrorCF("matrix", "Failed to resolve media", map[string]any{
                "ref": part.Ref, "error": err.Error(),
            })
            continue
        }

        // Call the appropriate API based on part.Type ("image"|"audio"|"video"|"file")
        switch part.Type {
        case "image":
            // Upload image to Matrix
        default:
            // Upload file to Matrix
        }
    }
    return nil
}
```

#### TypingCapable — Typing Indicator

```go
// If the platform supports "typing..." indicators
func (c *MatrixChannel) StartTyping(ctx context.Context, chatID string) (stop func(), err error) {
    // Call Matrix API to send typing indicator
    // The returned stop function must be idempotent
    stopped := false
    return func() {
        if !stopped {
            stopped = true
            // Call Matrix API to stop typing
        }
    }, nil
}
```

#### ReactionCapable — Message Reaction Indicator

```go
// If the platform supports adding emoji reactions to inbound messages (e.g., Slack's 👀, OneBot's emoji 289)
func (c *MatrixChannel) ReactToMessage(ctx context.Context, chatID, messageID string) (undo func(), err error) {
    // Call Matrix API to add reaction to message
    // The returned undo function removes the reaction, must be idempotent
    err = c.addReaction(chatID, messageID, "eyes")
    if err != nil {
        return func() {}, err
    }
    return func() {
        c.removeReaction(chatID, messageID, "eyes")
    }, nil
}
```

#### MessageEditor — Message Editing

```go
// If the platform supports editing sent messages (used for Placeholder replacement)
func (c *MatrixChannel) EditMessage(ctx context.Context, chatID, messageID, content string) error {
    // Call Matrix API to edit message
    return nil
}
```

#### WebhookHandler — HTTP Webhook Reception

```go
// If the channel receives messages via webhook (rather than long-polling/WebSocket)
func (c *MatrixChannel) WebhookPath() string {
    return "/webhook/matrix"   // Path will be registered on the shared HTTP server
}

func (c *MatrixChannel) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // Handle webhook request
}
```

#### HealthChecker — Health Check Endpoint

```go
func (c *MatrixChannel) HealthPath() string {
    return "/health/matrix"
}

func (c *MatrixChannel) HealthHandler(w http.ResponseWriter, r *http.Request) {
    if c.IsRunning() {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("OK"))
    } else {
        w.WriteHeader(http.StatusServiceUnavailable)
    }
}
```

### 3.4 Inbound-side Typing/Reaction/Placeholder Auto-orchestration

`BaseChannel.HandleMessage` automatically detects whether the channel implements `TypingCapable`, `ReactionCapable`, and/or `PlaceholderCapable` **before** publishing the inbound message, and triggers the corresponding indicators. The three pipelines are completely independent and do not interfere with each other:

```go
// Automatically executed inside BaseChannel.HandleMessage (no manual calls needed):
if c.owner != nil && c.placeholderRecorder != nil {
    // Typing — independent pipeline
    if tc, ok := c.owner.(TypingCapable); ok {
        if stop, err := tc.StartTyping(ctx, chatID); err == nil {
            c.placeholderRecorder.RecordTypingStop(c.name, chatID, stop)
        }
    }
    // Reaction — independent pipeline
    if rc, ok := c.owner.(ReactionCapable); ok && messageID != "" {
        if undo, err := rc.ReactToMessage(ctx, chatID, messageID); err == nil {
            c.placeholderRecorder.RecordReactionUndo(c.name, chatID, undo)
        }
    }
    // Placeholder — independent pipeline
    if pc, ok := c.owner.(PlaceholderCapable); ok {
        if phID, err := pc.SendPlaceholder(ctx, chatID); err == nil && phID != "" {
            c.placeholderRecorder.RecordPlaceholder(c.name, chatID, phID)
        }
    }
}
```

**This means**:
- Channels implementing `TypingCapable` (Telegram, Discord, LINE, Pico) do not need to manually call `StartTyping` + `RecordTypingStop` in `handleMessage`
- Channels implementing `ReactionCapable` (Slack, OneBot) do not need to manually call `AddReaction` + `RecordTypingStop` in `handleMessage`
- Channels implementing `PlaceholderCapable` (Telegram, Discord, Pico) do not need to manually send placeholder messages and call `RecordPlaceholder` in `handleMessage`
- Channels only need to implement the corresponding interface; `HandleMessage` handles orchestration automatically
- Channels that don't implement these interfaces are unaffected (type assertions will fail and be skipped)
- `PlaceholderCapable`'s `SendPlaceholder` method internally decides whether to send based on the configured `PlaceholderConfig.Enabled`; returning `("", nil)` skips registration

**Owner Injection**: Manager automatically calls `SetOwner(ch)` in `initChannel` to inject the concrete channel into BaseChannel — no manual setup required from developers.

When the Agent finishes processing a message, Manager's `preSend` automatically:
1. Calls the recorded `stop()` to stop Typing
2. Calls the recorded `undo()` to undo Reaction
3. If there is a Placeholder and the channel implements `MessageEditor`, attempts to edit the Placeholder with the final reply (skipping Send)

### 3.5 Register Configuration and Gateway Integration

#### Add configuration in `pkg/config/config.go`

```go
type ChannelsConfig struct {
    // ... existing channels
    Matrix  MatrixChannelConfig  `yaml:"matrix" json:"matrix"`
}

type MatrixChannelConfig struct {
    Enabled    bool     `yaml:"enabled" json:"enabled"`
    HomeServer string   `yaml:"home_server" json:"home_server"`
    Token      string   `yaml:"token" json:"token"`
    AllowFrom  []string `yaml:"allow_from" json:"allow_from"`
    GroupTrigger GroupTriggerConfig `yaml:"group_trigger" json:"group_trigger"`
}
```

#### Add entry in Manager.initChannels()

```go
// In the initChannels() method of pkg/channels/manager.go
if m.config.Channels.Matrix.Enabled && m.config.Channels.Matrix.Token != "" {
    m.initChannel("matrix", "Matrix")
}
```

#### Add blank import in Gateway

```go
// cmd/picoclaw/internal/gateway/helpers.go
import (
    _ "github.com/sipeed/picoclaw/pkg/channels/matrix"
)
```

---

## Part 4: Core Subsystem Details

### 4.1 MessageBus

**Files**: `pkg/bus/bus.go`, `pkg/bus/types.go`

```go
type MessageBus struct {
    inbound       chan InboundMessage       // buffer = 64
    outbound      chan OutboundMessage      // buffer = 64
    outboundMedia chan OutboundMediaMessage  // buffer = 64
    done          chan struct{}             // Close signal
    closed        atomic.Bool              // Prevents double-close
}
```

**Key Behaviors**:

| Method | Behavior |
|--------|----------|
| `PublishInbound(ctx, msg)` | Check closed → send to inbound channel → block/timeout/close |
| `ConsumeInbound(ctx)` | Read from inbound → block/close/cancel |
| `PublishOutbound(ctx, msg)` | Send to outbound channel |
| `SubscribeOutbound(ctx)` | Read from outbound (called by Manager dispatcher) |
| `PublishOutboundMedia(ctx, msg)` | Send to outboundMedia channel |
| `SubscribeOutboundMedia(ctx)` | Read from outboundMedia (called by Manager media dispatcher) |
| `Close()` | CAS close → close(done) → drain all channels (**does not close the channels themselves** to avoid concurrent send-on-closed panic) |

**Design Notes**:
- Buffer size increased from 16 to 64 to reduce blocking under burst load
- `Close()` does not close the underlying channels (only closes the `done` signal channel), because there may be concurrent `Publish` goroutines
- Drain loop ensures buffered messages are not silently dropped

### 4.2 Structured Message Types

**File**: `pkg/bus/types.go`

```go
// Routing peer
type Peer struct {
    Kind string `json:"kind"`  // "direct" | "group" | "channel" | ""
    ID   string `json:"id"`
}

// Sender identity information
type SenderInfo struct {
    Platform    string `json:"platform,omitempty"`     // "telegram", "discord", ...
    PlatformID  string `json:"platform_id,omitempty"`  // Platform-native ID
    CanonicalID string `json:"canonical_id,omitempty"` // "platform:id" canonical format
    Username    string `json:"username,omitempty"`
    DisplayName string `json:"display_name,omitempty"`
}

// Inbound message
type InboundMessage struct {
    Channel    string            // Source channel name
    SenderID   string            // Sender ID (prefer CanonicalID)
    Sender     SenderInfo        // Structured sender info
    ChatID     string            // Chat/room ID
    Content    string            // Message text
    Media      []string          // Media reference list (media://...)
    Peer       Peer              // Routing peer (first-class field)
    MessageID  string            // Platform message ID (first-class field)
    MediaScope string            // Media lifecycle scope
    SessionKey string            // Session key
    Metadata   map[string]string // Only for channel-specific extensions
}

// Outbound text message
type OutboundMessage struct {
    Channel string
    ChatID  string
    Content string
}

// Outbound media message
type OutboundMediaMessage struct {
    Channel string
    ChatID  string
    Parts   []MediaPart
}

// Media part
type MediaPart struct {
    Type        string // "image" | "audio" | "video" | "file"
    Ref         string // "media://uuid"
    Caption     string
    Filename    string
    ContentType string
}
```

### 4.3 BaseChannel

**File**: `pkg/channels/base.go`

BaseChannel is the shared abstraction layer for all channels, providing the following capabilities:

| Method/Feature | Description |
|---|---|
| `Name() string` | Channel name |
| `IsRunning() bool` | Atomically read running state |
| `SetRunning(bool)` | Atomically set running state |
| `MaxMessageLength() int` | Message length limit (rune count), 0 = unlimited |
| `IsAllowed(senderID string) bool` | Legacy allow-list check (supports `"id\|username"` and `"@username"` formats) |
| `IsAllowedSender(sender SenderInfo) bool` | New allow-list check (delegates to `identity.MatchAllowed`) |
| `ShouldRespondInGroup(isMentioned, content) (bool, string)` | Unified group chat trigger filtering logic |
| `HandleMessage(...)` | Unified inbound message handling: permission check → build MediaScope → auto-trigger Typing/Reaction → publish to Bus |
| `SetMediaStore(s) / GetMediaStore()` | MediaStore injected by Manager |
| `SetPlaceholderRecorder(r) / GetPlaceholderRecorder()` | PlaceholderRecorder injected by Manager |
| `SetOwner(ch)` | Concrete channel reference injected by Manager (used for Typing/Reaction type assertions in HandleMessage) |

**Functional Options**:

```go
channels.WithMaxMessageLength(4096)        // Set platform message length limit
channels.WithGroupTrigger(groupTriggerCfg) // Set group trigger configuration
```

### 4.4 Factory Registry

**File**: `pkg/channels/registry.go`

```go
type ChannelFactory func(cfg *config.Config, bus *bus.MessageBus) (Channel, error)

func RegisterFactory(name string, f ChannelFactory)   // Called in sub-package init()
func getFactory(name string) (ChannelFactory, bool)    // Called internally by Manager
```

The factory registry is protected by `sync.RWMutex` and registrations occur during `init()` phase (completed at process startup). Manager looks up factories by name in `initChannel()` and calls them.

### 4.5 Error Classification and Retries

**Files**: `pkg/channels/errors.go`, `pkg/channels/errutil.go`

#### Sentinel Errors

```go
var (
    ErrNotRunning = errors.New("channel not running")   // Permanent: do not retry
    ErrRateLimit  = errors.New("rate limited")           // Fixed delay: retry after 1s
    ErrTemporary  = errors.New("temporary failure")      // Exponential backoff: 500ms * 2^attempt, max 8s
    ErrSendFailed = errors.New("send failed")            // Permanent: do not retry
)
```

#### Error Classification Helpers

```go
// Automatically classify based on HTTP status code
func ClassifySendError(statusCode int, rawErr error) error {
    // 429 → ErrRateLimit
    // 5xx → ErrTemporary
    // 4xx → ErrSendFailed
}

// Wrap network errors as temporary
func ClassifyNetError(err error) error {
    // → ErrTemporary
}
```

#### Manager Retry Strategy (`sendWithRetry`)

```
Max retries:      3
Rate limit delay:  1 second
Base backoff:      500 milliseconds
Max backoff:       8 seconds

Retry logic:
  ErrNotRunning → Fail immediately, no retry
  ErrSendFailed → Fail immediately, no retry
  ErrRateLimit  → Wait 1s → retry
  ErrTemporary  → Wait 500ms * 2^attempt (max 8s) → retry
  Other unknown → Wait 500ms * 2^attempt (max 8s) → retry
```

### 4.6 Manager Orchestration

**File**: `pkg/channels/manager.go`

#### Per-channel Worker Architecture

```go
type channelWorker struct {
    ch         Channel                      // Channel instance
    queue      chan bus.OutboundMessage      // Outbound text queue (buffered 16)
    mediaQueue chan bus.OutboundMediaMessage // Outbound media queue (buffered 16)
    done       chan struct{}                // Text worker completion signal
    mediaDone  chan struct{}                // Media worker completion signal
    limiter    *rate.Limiter                // Per-channel rate limiter
}
```

#### Per-channel Rate Limit Configuration

```go
var channelRateConfig = map[string]float64{
    "telegram": 20,   // 20 msg/s
    "discord":  1,    // 1 msg/s
    "slack":    1,    // 1 msg/s
    "line":     10,   // 10 msg/s
}
// Default: 10 msg/s
// burst = max(1, ceil(rate/2))
```

#### Lifecycle Management

```
StartAll:
  1. Iterate registered channels → channel.Start(ctx)
  2. Create channelWorker for each successfully started channel
  3. Start goroutines:
     - runWorker (per-channel outbound text)
     - runMediaWorker (per-channel outbound media)
     - dispatchOutbound (route from bus to worker queues)
     - dispatchOutboundMedia (route from bus to media worker queues)
     - runTTLJanitor (every 10s clean up expired typing/placeholder)
  4. Start shared HTTP server (if configured)

StopAll:
  1. Shut down shared HTTP server (5s timeout)
  2. Cancel dispatcher context
  3. Close text worker queues → wait for drain to complete
  4. Close media worker queues → wait for drain to complete
  5. Stop each channel (channel.Stop)
```

#### Typing/Reaction/Placeholder Management

```go
// Manager implements PlaceholderRecorder interface
func (m *Manager) RecordPlaceholder(channel, chatID, placeholderID string)
func (m *Manager) RecordTypingStop(channel, chatID string, stop func())
func (m *Manager) RecordReactionUndo(channel, chatID string, undo func())

// Inbound side: BaseChannel.HandleMessage auto-orchestrates
// BaseChannel.HandleMessage, before PublishInbound, auto-triggers via owner type assertions:
//   - TypingCapable.StartTyping       → RecordTypingStop
//   - ReactionCapable.ReactToMessage  → RecordReactionUndo
//   - PlaceholderCapable.SendPlaceholder → RecordPlaceholder
// All three are independent and do not interfere with each other. Channels don't need to call these manually.

// Outbound side: pre-send processing
func (m *Manager) preSend(ctx, name, msg, ch) bool {
    key := name + ":" + msg.ChatID
    // 1. Stop Typing (call stored stop function)
    // 2. Undo Reaction (call stored undo function)
    // 3. Attempt to edit Placeholder (if channel implements MessageEditor)
    //    Success → return true (skip Send)
    //    Failure → return false (proceed with Send)
}
```

Manager storage is fully separated; three pipelines do not interfere:

```go
Manager {
    typingStops   sync.Map  // "channel:chatID" → typingEntry    ← manages TypingCapable
    reactionUndos sync.Map  // "channel:chatID" → reactionEntry  ← manages ReactionCapable
    placeholders  sync.Map  // "channel:chatID" → placeholderEntry
}
```

TTL Cleanup:
- Typing stop functions: 5-minute TTL (auto-calls stop and deletes on expiry)
- Reaction undo functions: 5-minute TTL (auto-calls undo and deletes on expiry)
- Placeholder IDs: 10-minute TTL (deletes on expiry)
- Cleanup interval: 10 seconds

### 4.7 Message Splitting

**File**: `pkg/channels/split.go`

`SplitMessage(content string, maxLen int) []string`

Smart splitting strategy:
1. Calculate effective split point = maxLen - 10% buffer (to reserve space for code block closure)
2. Prefer splitting at newlines
3. Otherwise split at spaces/tabs
4. Detect unclosed code blocks (` ``` `)
5. If a code block is unclosed:
   - Attempt to extend to maxLen to include the closing fence
   - If the code block is too long, inject close/reopen fences (`\n```\n` + header)
   - Last resort: split before the code block starts

### 4.8 MediaStore

**File**: `pkg/media/store.go`

```go
type MediaStore interface {
    Store(localPath string, meta MediaMeta, scope string) (ref string, err error)
    Resolve(ref string) (localPath string, err error)
    ResolveWithMeta(ref string) (localPath string, meta MediaMeta, err error)
    ReleaseAll(scope string) error
}
```

**FileMediaStore Implementation**:
- Pure in-memory mapping, no file copy/move
- Reference format: `media://<uuid>`
- Scope format: `channel:chatID:messageID` (generated by `BuildMediaScope`)
- **Two-phase operation**:
  - Phase 1 (holding lock): collect and delete entries from map
  - Phase 2 (no lock): delete files from disk
  - Purpose: minimize lock contention
- **TTL Cleanup**: `NewFileMediaStoreWithCleanup` → `Start()` launches background cleanup goroutine
- Cleanup interval and max TTL are controlled by configuration

### 4.9 Identity

**File**: `pkg/identity/identity.go`

```go
// Build canonical ID
func BuildCanonicalID(platform, platformID string) string
// → "telegram:123456"

// Parse canonical ID
func ParseCanonicalID(canonical string) (platform, id string, ok bool)

// Match against allow list (backward-compatible)
func MatchAllowed(sender bus.SenderInfo, allowed string) bool
```

`MatchAllowed` supported allow-list formats:
| Format | Matching |
|--------|----------|
| `"123456"` | Matches `sender.PlatformID` |
| `"@alice"` | Matches `sender.Username` |
| `"123456\|alice"` | Matches PlatformID or Username (legacy format compatibility) |
| `"telegram:123456"` | Exact match on `sender.CanonicalID` (new format) |

### 4.10 Shared HTTP Server

**File**: `pkg/channels/manager.go`'s `SetupHTTPServer`

Manager creates a single `http.Server` and auto-discovers and registers:
- Channels implementing `WebhookHandler` → mounted at `wh.WebhookPath()`
- Channels implementing `HealthChecker` → mounted at `hc.HealthPath()`
- Global health endpoint registered by `health.Server.RegisterOnMux`

Timeout configuration: ReadTimeout = 30s, WriteTimeout = 30s

---

## Part 5: Key Design Decisions and Conventions

### 5.1 Mandatory Conventions

1. **Error classification is a contract**: A channel's `Send` method **must** return sentinel errors (or wrap them). Manager's retry strategy relies entirely on `errors.Is` checks. Returning unclassified errors will cause Manager to treat them as "unknown errors" (exponential backoff retry).

2. **SetRunning is a lifecycle signal**: **Must** call `c.SetRunning(true)` after successful `Start`, and **must** call `c.SetRunning(false)` at the beginning of `Stop`. **Must** check `c.IsRunning()` in `Send` and return `ErrNotRunning`.

3. **HandleMessage includes permission checks**: Do not perform your own permission checks before calling `HandleMessage` (unless you need platform-specific preprocessing before the check). `HandleMessage` already calls `IsAllowedSender`/`IsAllowed` internally.

4. **Message splitting is handled by Manager**: A channel's `Send` method does not need to handle long message splitting. Manager automatically splits based on `MaxMessageLength()` before calling `Send`. Channels only need to declare the limit via `WithMaxMessageLength`.

5. **Typing/Reaction/Placeholder is handled by BaseChannel + Manager automatically**: A channel's `Send` method does not need to manage Typing stop, Reaction undo, or Placeholder editing. `BaseChannel.HandleMessage` auto-triggers `TypingCapable`, `ReactionCapable`, and `PlaceholderCapable` on the inbound side (via `owner` type assertions); Manager's `preSend` auto-stops Typing, undoes Reaction, and edits Placeholder on the outbound side. Channels only need to implement the corresponding interfaces.

6. **Factory registration belongs in init()**: Each sub-package must have an `init.go` file calling `channels.RegisterFactory`. Gateway must trigger registration via blank imports (`_ "pkg/channels/xxx"`).

### 5.2 Metadata Field Usage Conventions

**Do NOT put the following information in Metadata anymore**:
- `peer_kind` / `peer_id` → Use `InboundMessage.Peer`
- `message_id` → Use `InboundMessage.MessageID`
- `sender_platform` / `sender_username` → Use `InboundMessage.Sender`

**Metadata should only be used for**:
- Channel-specific extension information (e.g., Telegram's `reply_to_message_id`)
- Temporary information that doesn't fit into structured fields

### 5.3 Concurrency Safety Conventions

- `BaseChannel.running`: Uses `atomic.Bool`, thread-safe
- `Manager.channels` / `Manager.workers`: Protected by `sync.RWMutex`
- `Manager.placeholders` / `Manager.typingStops` / `Manager.reactionUndos`: Uses `sync.Map`
- `MessageBus.closed`: Uses `atomic.Bool`
- `FileMediaStore`: Uses `sync.RWMutex`, two-phase operation to minimize lock-hold time
- Channel Worker queue: Go channel, inherently concurrent-safe

### 5.4 Testing Conventions

Existing test files:
- `pkg/channels/base_test.go` — BaseChannel unit tests
- `pkg/channels/manager_test.go` — Manager unit tests
- `pkg/channels/split_test.go` — Message splitting tests
- `pkg/channels/errors_test.go` — Error type tests
- `pkg/channels/errutil_test.go` — Error classification tests

To add tests for a new channel:
```bash
go test ./pkg/channels/matrix/ -v              # Sub-package tests
go test ./pkg/channels/ -run TestSpecific -v    # Framework tests
make test                                       # Full test suite
```

---

## Appendix: Complete File Listing and Interface Quick Reference

### A.1 Framework Layer Files

| File | Responsibility |
|------|---------------|
| `pkg/channels/base.go` | BaseChannel struct, Channel interface, MessageLengthProvider, BaseChannelOption, HandleMessage |
| `pkg/channels/interfaces.go` | TypingCapable, MessageEditor, ReactionCapable, PlaceholderCapable, PlaceholderRecorder interfaces |
| `pkg/channels/media.go` | MediaSender interface |
| `pkg/channels/webhook.go` | WebhookHandler, HealthChecker interfaces |
| `pkg/channels/errors.go` | ErrNotRunning, ErrRateLimit, ErrTemporary, ErrSendFailed sentinels |
| `pkg/channels/errutil.go` | ClassifySendError, ClassifyNetError helpers |
| `pkg/channels/registry.go` | RegisterFactory, getFactory factory registry |
| `pkg/channels/manager.go` | Manager: Worker queues, rate limiting, retries, preSend, shared HTTP, TTL janitor |
| `pkg/channels/split.go` | SplitMessage long-message splitting |
| `pkg/bus/bus.go` | MessageBus implementation |
| `pkg/bus/types.go` | Peer, SenderInfo, InboundMessage, OutboundMessage, OutboundMediaMessage, MediaPart |
| `pkg/media/store.go` | MediaStore interface, FileMediaStore implementation |
| `pkg/identity/identity.go` | BuildCanonicalID, ParseCanonicalID, MatchAllowed |

### A.2 Channel Sub-packages

| Sub-package | Registered Name | Optional Interfaces |
|-------------|----------------|-------------------|
| `pkg/channels/telegram/` | `"telegram"` | MessageEditor, MediaSender, TypingCapable, PlaceholderCapable |
| `pkg/channels/discord/` | `"discord"` | MessageEditor, TypingCapable, PlaceholderCapable |
| `pkg/channels/slack/` | `"slack"` | ReactionCapable |
| `pkg/channels/line/` | `"line"` | WebhookHandler, HealthChecker, TypingCapable |
| `pkg/channels/onebot/` | `"onebot"` | ReactionCapable |
| `pkg/channels/dingtalk/` | `"dingtalk"` | WebhookHandler |
| `pkg/channels/feishu/` | `"feishu"` | WebhookHandler (architecture-specific build tags) |
| `pkg/channels/wecom/` | `"wecom"` + `"wecom_app"` | WebhookHandler |
| `pkg/channels/qq/` | `"qq"` | — |
| `pkg/channels/whatsapp/` | `"whatsapp"` | — |
| `pkg/channels/maixcam/` | `"maixcam"` | — |
| `pkg/channels/pico/` | `"pico"` | WebhookHandler (Pico Protocol), TypingCapable, PlaceholderCapable |

### A.3 Interface Quick Reference

```go
// ===== Required =====
type Channel interface {
    Name() string
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    Send(ctx context.Context, msg bus.OutboundMessage) error
    IsRunning() bool
    IsAllowed(senderID string) bool
    IsAllowedSender(sender bus.SenderInfo) bool
}

// ===== Optional =====
type MediaSender interface {
    SendMedia(ctx context.Context, msg bus.OutboundMediaMessage) error
}

type TypingCapable interface {
    StartTyping(ctx context.Context, chatID string) (stop func(), err error)
}

type ReactionCapable interface {
    ReactToMessage(ctx context.Context, chatID, messageID string) (undo func(), err error)
}

type PlaceholderCapable interface {
    SendPlaceholder(ctx context.Context, chatID string) (messageID string, err error)
}

type MessageEditor interface {
    EditMessage(ctx context.Context, chatID, messageID, content string) error
}

type WebhookHandler interface {
    WebhookPath() string
    http.Handler
}

type HealthChecker interface {
    HealthPath() string
    HealthHandler(w http.ResponseWriter, r *http.Request)
}

type MessageLengthProvider interface {
    MaxMessageLength() int
}

// ===== Injected by Manager =====
type PlaceholderRecorder interface {
    RecordPlaceholder(channel, chatID, placeholderID string)
    RecordTypingStop(channel, chatID string, stop func())
    RecordReactionUndo(channel, chatID string, undo func())
}
```

### A.4 Gateway Startup Sequence (Complete Bootstrap Flow)

```go
// 1. Create core components
msgBus     := bus.NewMessageBus()
provider   := providers.CreateProvider(cfg)
agentLoop  := agent.NewAgentLoop(cfg, msgBus, provider)

// 2. Create media store (with TTL cleanup)
mediaStore := media.NewFileMediaStoreWithCleanup(cleanerConfig)
mediaStore.Start()

// 3. Create Channel Manager (triggers initChannels → factory lookup → construct → inject MediaStore/PlaceholderRecorder/Owner)
channelManager := channels.NewManager(cfg, msgBus, mediaStore)

// 4. Inject references
agentLoop.SetChannelManager(channelManager)
agentLoop.SetMediaStore(mediaStore)

// 5. Configure shared HTTP server
channelManager.SetupHTTPServer(addr, healthServer)

// 6. Start
channelManager.StartAll(ctx)  // Start channels + workers + dispatchers + HTTP server
go agentLoop.Run(ctx)          // Start Agent message loop

// 7. Shutdown (signal-triggered)
cancel()                       // Cancel context
msgBus.Close()                 // Signal close + drain
channelManager.StopAll(shutdownCtx)  // Stop HTTP + workers + channels
mediaStore.Stop()              // Stop TTL cleanup
agentLoop.Stop()               // Stop Agent
```

### A.5 Per-channel Rate Limit Reference

| Channel | Rate (msg/s) | Burst |
|---------|-------------|-------|
| telegram | 20 | 10 |
| discord | 1 | 1 |
| slack | 1 | 1 |
| line | 10 | 5 |
| _others_ | 10 (default) | 5 |

### A.6 Known Limitations and Caveats

1. **Media cleanup temporarily disabled**: The `ReleaseAll` call in the Agent loop is commented out (`refactor(loop): disable media cleanup to prevent premature file deletion`) because session boundaries are not yet clearly defined. TTL cleanup remains active.

2. **Feishu architecture-specific compilation**: The Feishu channel uses build tags to distinguish 32-bit and 64-bit architectures (`feishu_32.go` / `feishu_64.go`).

3. **WeCom has two factories**: `"wecom"` (Bot mode) and `"wecom_app"` (App mode) are registered separately.

4. **Pico Protocol**: `pkg/channels/pico/` implements a custom PicoClaw native protocol channel that receives messages via webhook.
# QQ Channel Stability Improvements Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add WebSocket reconnection, API retry with backoff, and rate limiting to the QQ channel for improved stability.

**Architecture:** Three independent improvements layered on top of the existing QQ channel:
1. WebSocket reconnection with exponential backoff (5s → 5min max)
2. API retry with exponential backoff for transient errors
3. Per-chat token bucket rate limiting (group: 500ms, direct: 200ms intervals)

**Tech Stack:** Go, botgo SDK, sync/atomic, time.Timer

---

## File Structure

```
pkg/channels/qq/
├── qq.go              # Modify: Add reconnection, retry, rate limiting
├── qq_test.go         # Modify: Add tests for new functionality
├── qq_stability.go    # Create: Rate limiter and retry helper (new file)
└── qq_stability_test.go  # Create: Tests for stability utilities
```

**Config Changes:**
```
pkg/config/config.go   # Modify: Add retry/rate limit config fields to QQConfig
```

---

## Task 1: Add Reconnection Logic

**Files:**
- Modify: `pkg/channels/qq/qq.go:76-162` (Start/Stop methods)
- Modify: `pkg/channels/qq/qq.go` (add reconnect constants and state)
- Test: `pkg/channels/qq/qq_test.go`

- [ ] **Step 1: Add reconnect constants to qq.go**

Add after line 32 (after `typingSeconds`):
```go
// Reconnection constants
const (
    reconnectInitial    = 5 * time.Second
    reconnectMax        = 5 * time.Minute
    reconnectMultiplier = 2.0
)
```

- [ ] **Step 2: Add reconnect state fields to QQChannel struct**

Modify struct at line 34-59, add:
```go
type QQChannel struct {
    // ... existing fields ...

    // Reconnection state
    reconnectMu   sync.Mutex
    reconnecting  bool
    stopReconnect chan struct{}
}
```

- [ ] **Step 3: Modify Start() to handle reconnection**

Replace the goroutine at line 126-133 (the sessionManager.Start call) with a reconnect-aware wrapper:

```go
// startSession starts the WebSocket session with reconnection support.
func (c *QQChannel) startSession(ctx context.Context) {
    go func() {
        for {
            select {
            case <-ctx.Done():
                return
            case <-c.stopReconnect:
                return
            default:
                c.runSession(ctx)
                // Session ended unexpectedly, attempt reconnect
                if !c.isStopping() {
                    c.reconnect(ctx)
                }
            }
        }
    }()
}

func (c *QQChannel) isStopping() bool {
    return !c.IsRunning()
}

func (c *QQChannel) runSession(ctx context.Context) {
    wsInfo, err := c.api.WS(ctx, nil, "")
    if err != nil {
        logger.ErrorCF("qq", "Failed to get websocket info", map[string]any{
            "error": err.Error(),
        })
        return
    }

    logger.InfoCF("qq", "Got WebSocket info", map[string]any{
        "shards": wsInfo.Shards,
    })

    c.sessionManager = botgo.NewSessionManager()
    if err := c.sessionManager.Start(wsInfo, c.tokenSource, c.createIntent()); err != nil {
        logger.ErrorCF("qq", "WebSocket session error", map[string]any{
            "error": err.Error(),
        })
        c.SetRunning(false)
    }
}

func (c *QQChannel) createIntent() event.Handler {
    return event.RegisterHandlers(
        c.handleC2CMessage(),
        c.handleGroupATMessage(),
    )
}

func (c *QQChannel) reconnect(ctx context.Context) {
    c.reconnectMu.Lock()
    if c.reconnecting {
        c.reconnectMu.Unlock()
        return
    }
    c.reconnecting = true
    c.reconnectMu.Unlock()

    defer func() {
        c.reconnectMu.Lock()
        c.reconnecting = false
        c.reconnectMu.Unlock()
    }()

    backoff := reconnectInitial
    for {
        select {
        case <-ctx.Done():
            return
        case <-c.stopReconnect:
            return
        default:
        }

        logger.InfoCF("qq", "Attempting to reconnect", map[string]any{
            "backoff": backoff.String(),
        })

        // Reset internal state for new session
        c.resetSessionState()

        c.runSession(ctx)

        if c.IsRunning() {
            logger.InfoC("qq", "Reconnected successfully")
            return
        }

        logger.WarnCF("qq", "Reconnect failed, retrying", map[string]any{
            "backoff": backoff.String(),
        })

        select {
        case <-ctx.Done():
            return
        case <-c.stopReconnect:
            return
        case <-time.After(backoff):
            if backoff < reconnectMax {
                backoff = time.Duration(float64(backoff) * reconnectMultiplier)
                if backoff > reconnectMax {
                    backoff = reconnectMax
                }
            }
        }
    }
}

func (c *QQChannel) resetSessionState() {
    // Clear transient state that may be invalid after reconnection
    c.chatType.Range(func(key, value interface{}) bool {
        c.chatType.Delete(key)
        return true
    })
    c.lastMsgID.Range(func(key, value interface{}) bool {
        c.lastMsgID.Delete(key)
        return true
    })
    c.msgSeqCounters.Range(func(key, value interface{}) bool {
        c.msgSeqCounters.Delete(key)
        return true
    })

    // Re-register reasoning channel if configured
    if c.config.ReasoningChannelID != "" {
        c.chatType.Store(c.config.ReasoningChannelID, "group")
    }
}
```

- [ ] **Step 4: Modify Stop() to handle reconnect goroutine cleanup**

Replace Stop method (lines 150-162):
```go
func (c *QQChannel) Stop(ctx context.Context) error {
    logger.InfoC("qq", "Stopping QQ bot")
    c.SetRunning(false)

    // Signal reconnection loop to stop
    c.reconnectMu.Lock()
    c.reconnecting = false
    c.reconnectMu.Unlock()

    // Close stop channel to terminate reconnect goroutine
    if c.stopReconnect != nil {
        close(c.stopReconnect)
    }

    // Signal dedup janitor to stop
    c.stopOnce.Do(func() { close(c.done) })

    if c.cancel != nil {
        c.cancel()
    }

    return nil
}
```

- [ ] **Step 5: Update NewQQChannel to initialize stopReconnect**

Modify NewQQChannel (line 68-74):
```go
func NewQQChannel(cfg config.QQConfig, messageBus *bus.MessageBus) (*QQChannel, error) {
    base := channels.NewBaseChannel("qq", cfg, messageBus, cfg.AllowFrom,
        channels.WithMaxMessageLength(cfg.MaxMessageLength),
        channels.WithGroupTrigger(cfg.GroupTrigger),
        channels.WithReasoningChannelID(cfg.ReasoningChannelID),
    )

    return &QQChannel{
        BaseChannel:      base,
        config:          cfg,
        dedup:           make(map[string]time.Time),
        done:            make(chan struct{}),
        stopReconnect:   make(chan struct{}),
    }, nil
}
```

- [ ] **Step 6: Update Start() to use new flow**

Modify Start method (lines 76-148):
```go
func (c *QQChannel) Start(ctx context.Context) error {
    if c.config.AppID == "" || c.config.AppSecret == "" {
        return fmt.Errorf("QQ app_id and app_secret not configured")
    }

    botgo.SetLogger(logger.NewLogger("botgo"))
    logger.InfoC("qq", "Starting QQ bot (WebSocket mode)")

    // Reinitialize shutdown signals for clean restart
    c.done = make(chan struct{})
    c.stopOnce = sync.Once{}
    c.stopReconnect = make(chan struct{})
    c.reconnectMu.Lock()
    c.reconnecting = false
    c.reconnectMu.Unlock()

    // create token source
    credentials := &token.QQBotCredentials{
        AppID:     c.config.AppID,
        AppSecret: c.config.AppSecret,
    }
    c.tokenSource = token.NewQQBotTokenSource(credentials)

    // create child context
    c.ctx, c.cancel = context.WithCancel(ctx)

    // start auto-refresh token goroutine
    if err := token.StartRefreshAccessToken(c.ctx, c.tokenSource); err != nil {
        return fmt.Errorf("failed to start token refresh: %w", err)
    }

    // initialize OpenAPI client
    c.api = botgo.NewOpenAPI(c.config.AppID, c.tokenSource).WithTimeout(5 * time.Second)

    // start session with reconnection support
    go c.startSession(c.ctx)

    // start dedup janitor goroutine
    go c.dedupJanitor()

    // Pre-register reasoning_channel_id as group chat if configured
    if c.config.ReasoningChannelID != "" {
        c.chatType.Store(c.config.ReasoningChannelID, "group")
    }

    c.SetRunning(true)
    logger.InfoC("qq", "QQ bot started successfully")

    return nil
}
```

- [ ] **Step 7: Run tests to verify changes**

Run: `go test ./pkg/channels/qq/... -v -count=1`
Expected: Existing tests pass

- [ ] **Step 8: Commit**

```bash
git add pkg/channels/qq/qq.go
git commit -m "feat(qq): add WebSocket reconnection with exponential backoff"
```

---

## Task 2: Add API Retry with Exponential Backoff

**Files:**
- Modify: `pkg/channels/qq/qq.go` (add retry logic to Send/SendMedia)
- Modify: `pkg/channels/qq/qq.go` (add retry constants and helpers)
- Test: `pkg/channels/qq/qq_test.go`

- [ ] **Step 1: Add retry constants**

Add after reconnect constants:
```go
// Retry constants
const (
    maxRetries         = 3
    retryInitialDelay  = 500 * time.Millisecond
    retryMaxDelay     = 10 * time.Second
    retryMultiplier    = 2.0
)
```

- [ ] **Step 2: Add error classification helper**

Add at end of qq.go (after sanitizeURLs):
```go
// isRetryableError returns true if the error should be retried.
func isRetryableError(err error) bool {
    if err == nil {
        return false
    }
    errStr := err.Error()
    // Common transient error patterns
    return strings.Contains(errStr, "timeout") ||
           strings.Contains(errStr, "context deadline") ||
           strings.Contains(errStr, "connection reset") ||
           strings.Contains(errStr, "connection refused") ||
           strings.Contains(errStr, "temporary failure") ||
           strings.Contains(errStr, "429") ||
           strings.Contains(errStr, "500") ||
           strings.Contains(errStr, "502") ||
           strings.Contains(errStr, "503")
}

// calculateBackoff returns the next backoff duration.
func calculateBackoff(attempt int) time.Duration {
    backoff := retryInitialDelay * time.Duration(1<<uint(attempt))
    if backoff > retryMaxDelay {
        backoff = retryMaxDelay
    }
    return backoff
}
```

- [ ] **Step 3: Modify Send() to add retry logic**

Replace Send method (lines 179-232):
```go
func (c *QQChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
    if !c.IsRunning() {
        return channels.ErrNotRunning
    }

    chatKind := c.getChatKind(msg.ChatID)

    // Build message with content.
    msgToCreate := &dto.MessageToCreate{
        Content: msg.Content,
        MsgType: dto.TextMsg,
    }

    // Use Markdown message type if enabled in config.
    if c.config.SendMarkdown {
        msgToCreate.MsgType = dto.MarkdownMsg
        msgToCreate.Markdown = &dto.Markdown{
            Content: msg.Content,
        }
        // Clear plain content to avoid sending duplicate text.
        msgToCreate.Content = ""
    }

    c.applyReplyContext(msg.ChatID, msgToCreate)

    // Sanitize URLs in group messages to avoid QQ's URL blacklist rejection.
    if chatKind == "group" {
        if msgToCreate.Content != "" {
            msgToCreate.Content = sanitizeURLs(msgToCreate.Content)
        }
        if msgToCreate.Markdown != nil && msgToCreate.Markdown.Content != "" {
            msgToCreate.Markdown.Content = sanitizeURLs(msgToCreate.Markdown.Content)
        }
    }

    // Retry loop for transient failures
    var lastErr error
    for attempt := 0; attempt < maxRetries; attempt++ {
        if attempt > 0 {
            backoff := calculateBackoff(attempt - 1)
            logger.InfoCF("qq", "Retrying send", map[string]any{
                "attempt": attempt,
                "chat_id": msg.ChatID,
                "backoff": backoff.String(),
            })

            select {
            case <-ctx.Done():
                return ctx.Err()
            case <-time.After(backoff):
            }
        }

        var err error
        if chatKind == "group" {
            _, err = c.api.PostGroupMessage(ctx, msg.ChatID, msgToCreate)
        } else {
            _, err = c.api.PostC2CMessage(ctx, msg.ChatID, msgToCreate)
        }

        if err == nil {
            return nil
        }

        lastErr = err
        logger.WarnCF("qq", "Send attempt failed", map[string]any{
            "attempt":  attempt + 1,
            "chat_id":  msg.ChatID,
            "chat_kind": chatKind,
            "error":    err.Error(),
        })

        if !isRetryableError(err) {
            break
        }
    }

    logger.ErrorCF("qq", "Failed to send message after retries", map[string]any{
        "chat_id":   msg.ChatID,
        "chat_kind": chatKind,
        "error":     lastErr.Error(),
    })
    return fmt.Errorf("qq send: %w", channels.ErrTemporary)
}
```

- [ ] **Step 4: Modify SendMedia() to add retry logic**

Replace SendMedia method (lines 314-380):

The SendMedia method should have retry logic added to both the upload step and the send step. Add a retry wrapper:

```go
func (c *QQChannel) SendMedia(ctx context.Context, msg bus.OutboundMediaMessage) error {
    if !c.IsRunning() {
        return channels.ErrNotRunning
    }

    chatKind := c.getChatKind(msg.ChatID)

    for _, part := range msg.Parts {
        if isHTTPURL(part.Ref) {
            richMedia := &dto.RichMediaMessage{
                FileType:   qqFileType(part.Type),
                URL:        part.Ref,
                SrvSendMsg: true,
            }

            var lastErr error
            for attempt := 0; attempt < maxRetries; attempt++ {
                if attempt > 0 {
                    backoff := calculateBackoff(attempt - 1)
                    select {
                    case <-ctx.Done():
                        return ctx.Err()
                    case <-time.After(backoff):
                    }
                }

                var sendErr error
                if chatKind == "group" {
                    _, sendErr = c.api.PostGroupMessage(ctx, msg.ChatID, richMedia)
                } else {
                    _, sendErr = c.api.PostC2CMessage(ctx, msg.ChatID, richMedia)
                }

                if sendErr == nil {
                    continue
                }

                lastErr = sendErr
                if !isRetryableError(sendErr) {
                    break
                }
            }

            if lastErr != nil {
                logger.ErrorCF("qq", "Failed to send remote media after retries", map[string]any{
                    "type":    part.Type,
                    "chat_id": msg.ChatID,
                    "error":   lastErr.Error(),
                })
                return fmt.Errorf("qq send media: %w", channels.ErrTemporary)
            }
            continue
        }

        // Local file upload with retry
        store := c.GetMediaStore()
        if store == nil {
            return fmt.Errorf("qq send media: media store not configured for local media ref %q", part.Ref)
        }

        resolved, err := store.Resolve(part.Ref)
        if err != nil {
            return fmt.Errorf("qq send media: resolve local media ref %q: %w", part.Ref, err)
        }

        var fileInfo []byte
        var uploadErr error
        for attempt := 0; attempt < maxRetries; attempt++ {
            if attempt > 0 {
                backoff := calculateBackoff(attempt - 1)
                select {
                case <-ctx.Done():
                    return ctx.Err()
                case <-time.After(backoff):
                }
            }

            fileInfo, uploadErr = c.uploadLocalMedia(ctx, chatKind, msg.ChatID, part.Type, part.Filename, resolved)
            if uploadErr == nil {
                break
            }

            if !isRetryableError(uploadErr) {
                break
            }
        }

        if uploadErr != nil {
            logger.ErrorCF("qq", "Failed to upload local media after retries", map[string]any{
                "type":     part.Type,
                "chat_id":  msg.ChatID,
                "ref":      part.Ref,
                "resolved": resolved,
                "error":    uploadErr.Error(),
            })
            return fmt.Errorf("qq send media: %w", uploadErr)
        }

        if err := c.sendUploadedMedia(ctx, chatKind, msg.ChatID, part, fileInfo); err != nil {
            logger.ErrorCF("qq", "Failed to send uploaded media", map[string]any{
                "type":    part.Type,
                "chat_id": msg.ChatID,
                "error":   err.Error(),
            })
            return err
        }
    }

    return nil
}
```

Note: The inner `sendUploadedMedia` call is not retried as it depends on the already-uploaded file which should still be valid.

- [ ] **Step 5: Run tests to verify changes**

Run: `go test ./pkg/channels/qq/... -v -count=1`
Expected: Existing tests pass

- [ ] **Step 6: Commit**

```bash
git add pkg/channels/qq/qq.go
git commit -m "feat(qq): add API retry with exponential backoff for transient errors"
```

---

## Task 3: Add Token Bucket Rate Limiting

**Files:**
- Create: `pkg/channels/qq/qq_ratelimit.go` (rate limiter implementation)
- Modify: `pkg/channels/qq/qq.go` (integrate rate limiter into Send/SendMedia)
- Test: `pkg/channels/qq/qq_ratelimit_test.go`

- [ ] **Step 1: Create qq_ratelimit.go with token bucket implementation**

Create file `pkg/channels/qq/qq_ratelimit.go`:
```go
package qq

import (
    "sync"
    "time"
)

// rateLimiter implements a per-chat token bucket rate limiter.
type rateLimiter struct {
    mu       sync.Mutex
    lastSend map[string]time.Time
    interval time.Duration
}

// newRateLimiter creates a rate limiter with the specified minimum interval between sends.
func newRateLimiter(interval time.Duration) *rateLimiter {
    return &rateLimiter{
        lastSend: make(map[string]time.Time),
        interval: interval,
    }
}

// wait waits if necessary until enough time has passed since the last send to this chat.
// Returns the amount of time waited (0 if no wait was needed).
func (r *rateLimiter) wait(chatID string) time.Duration {
    r.mu.Lock()
    defer r.mu.Unlock()

    now := time.Now()
    if lastSend, ok := r.lastSend[chatID]; ok {
        elapsed := now.Sub(lastSend)
        if elapsed < r.interval {
            waitTime := r.interval - elapsed
            time.Sleep(waitTime)
            now = time.Now()
        }
    }

    r.lastSend[chatID] = now
    return 0
}

// waitWithContext waits with context cancellation support.
func (r *rateLimiter) waitWithContext(chatID string, ctx context.Context) error {
    r.mu.Lock()
    now := time.Now()
    var waitDuration time.Duration

    if lastSend, ok := r.lastSend[chatID]; ok {
        elapsed := now.Sub(lastSend)
        if elapsed < r.interval {
            waitDuration = r.interval - elapsed
        }
    }
    r.mu.Unlock()

    if waitDuration > 0 {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-time.After(waitDuration):
        }

        r.mu.Lock()
        r.lastSend[chatID] = time.Now()
        r.mu.Unlock()
    } else {
        r.mu.Lock()
        r.lastSend[chatID] = now
        r.mu.Unlock()
    }

    return nil
}

// clear removes the rate limit entry for a chat (e.g., after reconnection).
func (r *rateLimiter) clear(chatID string) {
    r.mu.Lock()
    defer r.mu.Unlock()
    delete(r.lastSend, chatID)
}

// clearAll removes all rate limit entries.
func (r *rateLimiter) clearAll() {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.lastSend = make(map[string]time.Time)
}
```

- [ ] **Step 2: Create test file qq_ratelimit_test.go**

Create file `pkg/channels/qq/qq_ratelimit_test.go`:
```go
package qq

import (
    "context"
    "sync"
    "testing"
    "time"
)

func TestRateLimiter_Wait(t *testing.T) {
    limiter := newRateLimiter(50 * time.Millisecond)

    // First call should not wait
    start := time.Now()
    limiter.wait("chat-1")
    elapsed := time.Since(start)
    if elapsed > 10*time.Millisecond {
        t.Errorf("first wait should be immediate, got %v", elapsed)
    }

    // Second call to same chat should wait
    start = time.Now()
    limiter.wait("chat-1")
    elapsed = time.Since(start)
    if elapsed < 40*time.Millisecond {
        t.Errorf("second wait should be at least 40ms, got %v", elapsed)
    }

    // Different chat should not wait
    start = time.Now()
    limiter.wait("chat-2")
    elapsed = time.Since(start)
    if elapsed > 10*time.Millisecond {
        t.Errorf("different chat should not wait, got %v", elapsed)
    }
}

func TestRateLimiter_Clear(t *testing.T) {
    limiter := newRateLimiter(100 * time.Millisecond)

    limiter.wait("chat-1")

    limiter.clear("chat-1")

    start := time.Now()
    limiter.wait("chat-1")
    elapsed := time.Since(start)
    if elapsed > 10*time.Millisecond {
        t.Errorf("after clear, wait should be immediate, got %v", elapsed)
    }
}

func TestRateLimiter_ClearAll(t *testing.T) {
    limiter := newRateLimiter(100 * time.Millisecond)

    limiter.wait("chat-1")
    limiter.wait("chat-2")

    limiter.clearAll()

    start := time.Now()
    limiter.wait("chat-1")
    elapsed := time.Since(start)
    if elapsed > 10*time.Millisecond {
        t.Errorf("after clearAll, wait should be immediate, got %v", elapsed)
    }
}

func TestRateLimiter_WaitWithContext_Cancelled(t *testing.T) {
    limiter := newRateLimiter(1 * time.Second)

    ctx, cancel := context.WithCancel(context.Background())

    // Start a goroutine that will cancel immediately
    go func() {
        time.Sleep(10 * time.Millisecond)
        cancel()
    }()

    start := time.Now()
    err := limiter.waitWithContext("chat-1", ctx)
    elapsed := time.Since(start)

    if err == nil {
        t.Error("expected context cancellation error")
    }
    if elapsed >= 100*time.Millisecond {
        t.Errorf("should have returned early due to cancellation, got %v", elapsed)
    }
}

func TestRateLimiter_WaitWithContext_Success(t *testing.T) {
    limiter := newRateLimiter(50 * time.Millisecond)

    ctx := context.Background()

    start := time.Now()
    err := limiter.waitWithContext("chat-1", ctx)
    elapsed := time.Since(start)

    if err != nil {
        t.Errorf("unexpected error: %v", err)
    }
    if elapsed > 10*time.Millisecond {
        t.Errorf("first wait should be immediate, got %v", elapsed)
    }
}

func TestRateLimiter_Concurrent(t *testing.T) {
    limiter := newRateLimiter(10 * time.Millisecond)

    var wg sync.WaitGroup
    chatIDs := []string{"chat-1", "chat-2", "chat-3"}

    // Run 10 iterations for each chat concurrently
    for i := 0; i < 10; i++ {
        for _, chatID := range chatIDs {
            wg.Add(1)
            go func(id string) {
                defer wg.Done()
                limiter.wait(id)
            }(chatID)
        }
    }

    // All goroutines should complete within reasonable time
    done := make(chan struct{})
    go func() {
        wg.Wait()
        close(done)
    }()

    select {
    case <-done:
        // Success
    case <-time.After(5 * time.Second):
        t.Error("concurrent rate limiter test timed out")
    }
}
```

- [ ] **Step 3: Run rate limiter tests**

Run: `go test ./pkg/channels/qq/... -v -run RateLimiter -count=1`
Expected: All tests pass

- [ ] **Step 4: Integrate rate limiter into QQChannel**

Modify QQChannel struct to add rate limiter fields (add after line 44):

```go
type QQChannel struct {
    *channels.BaseChannel
    config         config.QQConfig
    api            openapi.OpenAPI
    tokenSource    oauth2.TokenSource
    ctx            context.Context
    cancel         context.CancelFunc
    sessionManager botgo.SessionManager

    // Chat routing: track whether a chatID is group or direct.
    chatType sync.Map // chatID → "group" | "direct"

    // Passive reply: store last inbound message ID per chat.
    lastMsgID sync.Map // chatID → string

    // msg_seq: per-chat atomic counter for multi-part replies.
    msgSeqCounters sync.Map // chatID → *atomic.Uint64

    // Time-based dedup replacing the unbounded map.
    dedup   map[string]time.Time
    muDedup sync.Mutex

    // done is closed on Stop to shut down the dedup janitor.
    done     chan struct{}
    stopOnce sync.Once

    // Reconnection state
    reconnectMu   sync.Mutex
    reconnecting bool
    stopReconnect chan struct{}

    // Rate limiting
    groupRateLimiter  *rateLimiter
    directRateLimiter *rateLimiter
}
```

- [ ] **Step 5: Initialize rate limiters in NewQQChannel**

Modify NewQQChannel:
```go
func NewQQChannel(cfg config.QQConfig, messageBus *bus.MessageBus) (*QQChannel, error) {
    base := channels.NewBaseChannel("qq", cfg, messageBus, cfg.AllowFrom,
        channels.WithMaxMessageLength(cfg.MaxMessageLength),
        channels.WithGroupTrigger(cfg.GroupTrigger),
        channels.WithReasoningChannelID(cfg.ReasoningChannelID),
    )

    return &QQChannel{
        BaseChannel:      base,
        config:           cfg,
        dedup:            make(map[string]time.Time),
        done:             make(chan struct{}),
        stopReconnect:    make(chan struct{}),
        groupRateLimiter:  newRateLimiter(500 * time.Millisecond),  // 20 msg/min with headroom
        directRateLimiter: newRateLimiter(200 * time.Millisecond), // 5 msg/sec with headroom
    }, nil
}
```

- [ ] **Step 6: Apply rate limiting in Send method**

Add at beginning of Send method (after IsRunning check):
```go
// Apply rate limiting before sending
if chatKind == "group" {
    if err := c.groupRateLimiter.waitWithContext(msg.ChatID, ctx); err != nil {
        return fmt.Errorf("qq send: %w", err)
    }
} else {
    if err := c.directRateLimiter.waitWithContext(msg.ChatID, ctx); err != nil {
        return fmt.Errorf("qq send: %w", err)
    }
}
```

- [ ] **Step 7: Clear rate limiters on reconnection**

Add to `resetSessionState()`:
```go
func (c *QQChannel) resetSessionState() {
    // Clear transient state that may be invalid after reconnection
    c.chatType.Range(func(key, value interface{}) bool {
        c.chatType.Delete(key)
        return true
    })
    c.lastMsgID.Range(func(key, value interface{}) bool {
        c.lastMsgID.Delete(key)
        return true
    })
    c.msgSeqCounters.Range(func(key, value interface{}) bool {
        c.msgSeqCounters.Delete(key)
        return true
    })

    // Clear rate limiters to allow immediate sending after reconnection
    if c.groupRateLimiter != nil {
        c.groupRateLimiter.clearAll()
    }
    if c.directRateLimiter != nil {
        c.directRateLimiter.clearAll()
    }

    // Re-register reasoning channel if configured
    if c.config.ReasoningChannelID != "" {
        c.chatType.Store(c.config.ReasoningChannelID, "group")
    }
}
```

- [ ] **Step 8: Run tests to verify changes**

Run: `go test ./pkg/channels/qq/... -v -count=1`
Expected: All tests pass

- [ ] **Step 9: Commit**

```bash
git add pkg/channels/qq/qq_ratelimit.go pkg/channels/qq/qq_ratelimit_test.go pkg/channels/qq/qq.go
git commit -m "feat(qq): add per-chat token bucket rate limiting"
```

---

## Task 4: Add Configurable Retry/Rate Limit Parameters (Optional Enhancement)

**Files:**
- Modify: `pkg/config/config.go` (add new config fields)

- [ ] **Step 1: Add optional config fields to QQConfig**

Modify QQConfig struct (line 349-358):
```go
type QQConfig struct {
    Enabled            bool                `json:"enabled"                 env:"PICOCLAW_CHANNELS_QQ_ENABLED"`
    AppID              string              `json:"app_id"                  env:"PICOCLAW_CHANNELS_QQ_APP_ID"`
    AppSecret          string              `json:"app_secret"              env:"PICOCLAW_CHANNELS_QQ_APP_SECRET"`
    AllowFrom          FlexibleStringSlice `json:"allow_from"              env:"PICOCLAW_CHANNELS_QQ_ALLOW_FROM"`
    GroupTrigger       GroupTriggerConfig  `json:"group_trigger,omitempty"`
    MaxMessageLength   int                 `json:"max_message_length"      env:"PICOCLAW_CHANNELS_QQ_MAX_MESSAGE_LENGTH"`
    SendMarkdown       bool                `json:"send_markdown"           env:"PICOCLAW_CHANNELS_QQ_SEND_MARKDOWN"`
    ReasoningChannelID string              `json:"reasoning_channel_id"    env:"PICOCLAW_CHANNELS_QQ_REASONING_CHANNEL_ID"`

    // Stability settings (optional, defaults applied if not set)
    MaxRetries         int `json:"max_retries,omitempty"         env:"PICOCLAW_CHANNELS_QQ_MAX_RETRIES"`
    RateLimitGroupMS   int `json:"rate_limit_group_ms,omitempty" env:"PICOCLAW_CHANNELS_QQ_RATE_LIMIT_GROUP_MS"`
    RateLimitDirectMS  int `json:"rate_limit_direct_ms,omitempty" env:"PICOCLAW_CHANNELS_QQ_RATE_LIMIT_DIRECT_MS"`
}
```

- [ ] **Step 2: Apply config values in NewQQChannel**

Modify NewQQChannel to use config values with defaults:
```go
func NewQQChannel(cfg config.QQConfig, messageBus *bus.MessageBus) (*QQChannel, error) {
    base := channels.NewBaseChannel("qq", cfg, messageBus, cfg.AllowFrom,
        channels.WithMaxMessageLength(cfg.MaxMessageLength),
        channels.WithGroupTrigger(cfg.GroupTrigger),
        channels.WithReasoningChannelID(cfg.ReasoningChannelID),
    )

    // Apply stability settings with defaults
    maxRetries := cfg.MaxRetries
    if maxRetries <= 0 {
        maxRetries = 3
    }
    rateLimitGroup := time.Duration(cfg.RateLimitGroupMS) * time.Millisecond
    if rateLimitGroup <= 0 {
        rateLimitGroup = 500 * time.Millisecond
    }
    rateLimitDirect := time.Duration(cfg.RateLimitDirectMS) * time.Millisecond
    if rateLimitDirect <= 0 {
        rateLimitDirect = 200 * time.Millisecond
    }

    return &QQChannel{
        BaseChannel:      base,
        config:           cfg,
        dedup:            make(map[string]time.Time),
        done:             make(chan struct{}),
        stopReconnect:    make(chan struct{}),
        groupRateLimiter:  newRateLimiter(rateLimitGroup),
        directRateLimiter: newRateLimiter(rateLimitDirect),
        maxRetries:       maxRetries,
    }, nil
}
```

- [ ] **Step 3: Add maxRetries field to struct and constants**

Update struct and constant handling:
```go
const (
    defaultMaxRetries = 3
)

// QQChannel struct needs:
// maxRetries int
```

- [ ] **Step 4: Commit**

```bash
git add pkg/config/config.go pkg/channels/qq/qq.go
git commit -m "feat(qq): add configurable retry and rate limit parameters"
```

---

## Task 5: Final Integration Test

**Files:**
- Test: `pkg/channels/qq/qq_test.go`

- [ ] **Step 1: Run full test suite**

Run: `go test ./pkg/channels/qq/... -v -count=1`
Expected: All tests pass

- [ ] **Step 2: Run linter**

Run: `golangci-lint run ./pkg/channels/qq/...`
Expected: No errors (warnings acceptable)

- [ ] **Step 3: Build to verify compilation**

Run: `go build ./...`
Expected: Successful build

- [ ] **Step 4: Commit any remaining changes**

```bash
git add -A
git commit -m "test(qq): add integration tests for stability improvements"
```

---

## Summary

| Task | Description | Files Changed |
|------|-------------|---------------|
| 1 | WebSocket reconnection with exponential backoff | qq.go |
| 2 | API retry with exponential backoff | qq.go |
| 3 | Per-chat token bucket rate limiting | qq_ratelimit.go, qq_ratelimit_test.go, qq.go |
| 4 | Configurable parameters (optional) | config.go, qq.go |
| 5 | Final integration test | - |

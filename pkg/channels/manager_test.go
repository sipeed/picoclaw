package channels

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/time/rate"

	"github.com/sipeed/picoclaw/pkg/bus"
)

// mockChannel is a test double that delegates Send to a configurable function.
type mockChannel struct {
	BaseChannel
	sendFn func(ctx context.Context, msg bus.OutboundMessage) error
}

func (m *mockChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	return m.sendFn(ctx, msg)
}

func (m *mockChannel) Start(ctx context.Context) error { return nil }
func (m *mockChannel) Stop(ctx context.Context) error  { return nil }

// newTestManager creates a minimal Manager suitable for unit tests.
func newTestManager() *Manager {
	return &Manager{
		channels: make(map[string]Channel),
		workers:  make(map[string]*channelWorker),
	}
}

func TestSendWithRetry_Success(t *testing.T) {
	m := newTestManager()
	var callCount int
	ch := &mockChannel{
		sendFn: func(_ context.Context, _ bus.OutboundMessage) error {
			callCount++
			return nil
		},
	}
	w := &channelWorker{
		ch:      ch,
		limiter: rate.NewLimiter(rate.Inf, 1),
	}

	ctx := context.Background()
	msg := bus.OutboundMessage{Channel: "test", ChatID: "1", Content: "hello"}

	m.sendWithRetry(ctx, "test", w, msg)

	if callCount != 1 {
		t.Fatalf("expected 1 Send call, got %d", callCount)
	}
}

func TestSendWithRetry_TemporaryThenSuccess(t *testing.T) {
	m := newTestManager()
	var callCount int
	ch := &mockChannel{
		sendFn: func(_ context.Context, _ bus.OutboundMessage) error {
			callCount++
			if callCount <= 2 {
				return fmt.Errorf("network error: %w", ErrTemporary)
			}
			return nil
		},
	}
	w := &channelWorker{
		ch:      ch,
		limiter: rate.NewLimiter(rate.Inf, 1),
	}

	ctx := context.Background()
	msg := bus.OutboundMessage{Channel: "test", ChatID: "1", Content: "hello"}

	m.sendWithRetry(ctx, "test", w, msg)

	if callCount != 3 {
		t.Fatalf("expected 3 Send calls (2 failures + 1 success), got %d", callCount)
	}
}

func TestSendWithRetry_PermanentFailure(t *testing.T) {
	m := newTestManager()
	var callCount int
	ch := &mockChannel{
		sendFn: func(_ context.Context, _ bus.OutboundMessage) error {
			callCount++
			return fmt.Errorf("bad chat ID: %w", ErrSendFailed)
		},
	}
	w := &channelWorker{
		ch:      ch,
		limiter: rate.NewLimiter(rate.Inf, 1),
	}

	ctx := context.Background()
	msg := bus.OutboundMessage{Channel: "test", ChatID: "1", Content: "hello"}

	m.sendWithRetry(ctx, "test", w, msg)

	if callCount != 1 {
		t.Fatalf("expected 1 Send call (no retry for permanent failure), got %d", callCount)
	}
}

func TestSendWithRetry_NotRunning(t *testing.T) {
	m := newTestManager()
	var callCount int
	ch := &mockChannel{
		sendFn: func(_ context.Context, _ bus.OutboundMessage) error {
			callCount++
			return ErrNotRunning
		},
	}
	w := &channelWorker{
		ch:      ch,
		limiter: rate.NewLimiter(rate.Inf, 1),
	}

	ctx := context.Background()
	msg := bus.OutboundMessage{Channel: "test", ChatID: "1", Content: "hello"}

	m.sendWithRetry(ctx, "test", w, msg)

	if callCount != 1 {
		t.Fatalf("expected 1 Send call (no retry for ErrNotRunning), got %d", callCount)
	}
}

func TestSendWithRetry_RateLimitRetry(t *testing.T) {
	m := newTestManager()
	var callCount int
	ch := &mockChannel{
		sendFn: func(_ context.Context, _ bus.OutboundMessage) error {
			callCount++
			if callCount == 1 {
				return fmt.Errorf("429: %w", ErrRateLimit)
			}
			return nil
		},
	}
	w := &channelWorker{
		ch:      ch,
		limiter: rate.NewLimiter(rate.Inf, 1),
	}

	ctx := context.Background()
	msg := bus.OutboundMessage{Channel: "test", ChatID: "1", Content: "hello"}

	start := time.Now()
	m.sendWithRetry(ctx, "test", w, msg)
	elapsed := time.Since(start)

	if callCount != 2 {
		t.Fatalf("expected 2 Send calls (1 rate limit + 1 success), got %d", callCount)
	}
	// Should have waited at least rateLimitDelay (1s) but allow some slack
	if elapsed < 900*time.Millisecond {
		t.Fatalf("expected at least ~1s delay for rate limit retry, got %v", elapsed)
	}
}

func TestSendWithRetry_MaxRetriesExhausted(t *testing.T) {
	m := newTestManager()
	var callCount int
	ch := &mockChannel{
		sendFn: func(_ context.Context, _ bus.OutboundMessage) error {
			callCount++
			return fmt.Errorf("timeout: %w", ErrTemporary)
		},
	}
	w := &channelWorker{
		ch:      ch,
		limiter: rate.NewLimiter(rate.Inf, 1),
	}

	ctx := context.Background()
	msg := bus.OutboundMessage{Channel: "test", ChatID: "1", Content: "hello"}

	m.sendWithRetry(ctx, "test", w, msg)

	expected := maxRetries + 1 // initial attempt + maxRetries retries
	if callCount != expected {
		t.Fatalf("expected %d Send calls, got %d", expected, callCount)
	}
}

func TestSendWithRetry_UnknownError(t *testing.T) {
	m := newTestManager()
	var callCount int
	ch := &mockChannel{
		sendFn: func(_ context.Context, _ bus.OutboundMessage) error {
			callCount++
			if callCount == 1 {
				return errors.New("random unexpected error")
			}
			return nil
		},
	}
	w := &channelWorker{
		ch:      ch,
		limiter: rate.NewLimiter(rate.Inf, 1),
	}

	ctx := context.Background()
	msg := bus.OutboundMessage{Channel: "test", ChatID: "1", Content: "hello"}

	m.sendWithRetry(ctx, "test", w, msg)

	if callCount != 2 {
		t.Fatalf("expected 2 Send calls (unknown error treated as temporary), got %d", callCount)
	}
}

func TestSendWithRetry_ContextCancelled(t *testing.T) {
	m := newTestManager()
	var callCount int
	ch := &mockChannel{
		sendFn: func(_ context.Context, _ bus.OutboundMessage) error {
			callCount++
			return fmt.Errorf("timeout: %w", ErrTemporary)
		},
	}
	w := &channelWorker{
		ch:      ch,
		limiter: rate.NewLimiter(rate.Inf, 1),
	}

	ctx, cancel := context.WithCancel(context.Background())
	msg := bus.OutboundMessage{Channel: "test", ChatID: "1", Content: "hello"}

	// Cancel context after first Send attempt returns
	ch.sendFn = func(_ context.Context, _ bus.OutboundMessage) error {
		callCount++
		cancel()
		return fmt.Errorf("timeout: %w", ErrTemporary)
	}

	m.sendWithRetry(ctx, "test", w, msg)

	// Should have called Send once, then noticed ctx canceled during backoff
	if callCount != 1 {
		t.Fatalf("expected 1 Send call before context cancellation, got %d", callCount)
	}
}

func TestWorkerRateLimiter(t *testing.T) {
	m := newTestManager()

	var mu sync.Mutex
	var sendTimes []time.Time

	ch := &mockChannel{
		sendFn: func(_ context.Context, _ bus.OutboundMessage) error {
			mu.Lock()
			sendTimes = append(sendTimes, time.Now())
			mu.Unlock()
			return nil
		},
	}

	// Create a worker with a low rate: 2 msg/s, burst 1
	w := &channelWorker{
		ch:      ch,
		queue:   make(chan bus.OutboundMessage, 10),
		done:    make(chan struct{}),
		limiter: rate.NewLimiter(2, 1),
	}

	ctx := t.Context()

	go m.runWorker(ctx, "test", w)

	// Enqueue 4 messages
	for i := range 4 {
		w.queue <- bus.OutboundMessage{Channel: "test", ChatID: "1", Content: fmt.Sprintf("msg%d", i)}
	}

	// Wait enough time for all messages to be sent (4 msgs at 2/s = ~2s, give extra margin)
	time.Sleep(3 * time.Second)

	mu.Lock()
	times := make([]time.Time, len(sendTimes))
	copy(times, sendTimes)
	mu.Unlock()

	if len(times) != 4 {
		t.Fatalf("expected 4 sends, got %d", len(times))
	}

	// Verify rate limiting: total duration should be at least 1s
	// (first message immediate, then ~500ms between each subsequent one at 2/s)
	totalDuration := times[len(times)-1].Sub(times[0])
	if totalDuration < 1*time.Second {
		t.Fatalf("expected total duration >= 1s for 4 msgs at 2/s rate, got %v", totalDuration)
	}
}

func TestNewChannelWorker_DefaultRate(t *testing.T) {
	ch := &mockChannel{}
	w := newChannelWorker("unknown_channel", ch)

	if w.limiter == nil {
		t.Fatal("expected limiter to be non-nil")
	}
	if w.limiter.Limit() != rate.Limit(defaultRateLimit) {
		t.Fatalf("expected rate limit %v, got %v", rate.Limit(defaultRateLimit), w.limiter.Limit())
	}
}

func TestNewChannelWorker_ConfiguredRate(t *testing.T) {
	ch := &mockChannel{}

	for name, expectedRate := range channelRateConfig {
		w := newChannelWorker(name, ch)
		if w.limiter.Limit() != rate.Limit(expectedRate) {
			t.Fatalf("channel %s: expected rate %v, got %v", name, expectedRate, w.limiter.Limit())
		}
	}
}

func TestRunWorker_MessageSplitting(t *testing.T) {
	m := newTestManager()

	var mu sync.Mutex
	var received []string

	ch := &mockChannelWithLength{
		mockChannel: mockChannel{
			sendFn: func(_ context.Context, msg bus.OutboundMessage) error {
				mu.Lock()
				received = append(received, msg.Content)
				mu.Unlock()
				return nil
			},
		},
		maxLen: 5,
	}

	w := &channelWorker{
		ch:      ch,
		queue:   make(chan bus.OutboundMessage, 10),
		done:    make(chan struct{}),
		limiter: rate.NewLimiter(rate.Inf, 1),
	}

	ctx := t.Context()

	go m.runWorker(ctx, "test", w)

	// Send a message that should be split
	w.queue <- bus.OutboundMessage{Channel: "test", ChatID: "1", Content: "hello world"}

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	count := len(received)
	mu.Unlock()

	if count < 2 {
		t.Fatalf("expected message to be split into at least 2 chunks, got %d", count)
	}
}

// mockChannelWithLength implements MessageLengthProvider.
type mockChannelWithLength struct {
	mockChannel
	maxLen int
}

func (m *mockChannelWithLength) MaxMessageLength() int {
	return m.maxLen
}

func TestSendWithRetry_ExponentialBackoff(t *testing.T) {
	m := newTestManager()

	var callTimes []time.Time
	var callCount atomic.Int32
	ch := &mockChannel{
		sendFn: func(_ context.Context, _ bus.OutboundMessage) error {
			callTimes = append(callTimes, time.Now())
			callCount.Add(1)
			return fmt.Errorf("timeout: %w", ErrTemporary)
		},
	}
	w := &channelWorker{
		ch:      ch,
		limiter: rate.NewLimiter(rate.Inf, 1),
	}

	ctx := context.Background()
	msg := bus.OutboundMessage{Channel: "test", ChatID: "1", Content: "hello"}

	start := time.Now()
	m.sendWithRetry(ctx, "test", w, msg)
	totalElapsed := time.Since(start)

	// With maxRetries=3: attempts at 0, ~500ms, ~1.5s, ~3.5s
	// Total backoff: 500ms + 1s + 2s = 3.5s
	// Allow some margin
	if totalElapsed < 3*time.Second {
		t.Fatalf("expected total elapsed >= 3s for exponential backoff, got %v", totalElapsed)
	}

	if int(callCount.Load()) != maxRetries+1 {
		t.Fatalf("expected %d calls, got %d", maxRetries+1, callCount.Load())
	}
}

// --- Phase 10: preSend orchestration tests ---

// mockMessageEditor is a channel that supports MessageEditor.
type mockMessageEditor struct {
	mockChannel
	editFn func(ctx context.Context, chatID, messageID, content string) error
}

func (m *mockMessageEditor) EditMessage(ctx context.Context, chatID, messageID, content string) error {
	return m.editFn(ctx, chatID, messageID, content)
}

func TestPreSend_PlaceholderEditSuccess(t *testing.T) {
	m := newTestManager()
	var sendCalled bool
	var editCalled bool

	ch := &mockMessageEditor{
		mockChannel: mockChannel{
			sendFn: func(_ context.Context, _ bus.OutboundMessage) error {
				sendCalled = true
				return nil
			},
		},
		editFn: func(_ context.Context, chatID, messageID, content string) error {
			editCalled = true
			if chatID != "123" {
				t.Fatalf("expected chatID 123, got %s", chatID)
			}
			if messageID != "456" {
				t.Fatalf("expected messageID 456, got %s", messageID)
			}
			if content != "hello" {
				t.Fatalf("expected content 'hello', got %s", content)
			}
			return nil
		},
	}

	// Register placeholder
	m.RecordPlaceholder("test", "123", "456")

	msg := bus.OutboundMessage{Channel: "test", ChatID: "123", Content: "hello"}
	edited := m.preSend(context.Background(), "test", msg, ch)

	if !edited {
		t.Fatal("expected preSend to return true (placeholder edited)")
	}
	if !editCalled {
		t.Fatal("expected EditMessage to be called")
	}
	if sendCalled {
		t.Fatal("expected Send to NOT be called when placeholder edited")
	}
}

func TestPreSend_PlaceholderEditFails_FallsThrough(t *testing.T) {
	m := newTestManager()

	ch := &mockMessageEditor{
		mockChannel: mockChannel{
			sendFn: func(_ context.Context, _ bus.OutboundMessage) error {
				return nil
			},
		},
		editFn: func(_ context.Context, _, _, _ string) error {
			return fmt.Errorf("edit failed")
		},
	}

	m.RecordPlaceholder("test", "123", "456")

	msg := bus.OutboundMessage{Channel: "test", ChatID: "123", Content: "hello"}
	edited := m.preSend(context.Background(), "test", msg, ch)

	if edited {
		t.Fatal("expected preSend to return false when edit fails")
	}
}

func TestPreSend_TypingStopCalled(t *testing.T) {
	m := newTestManager()
	var stopCalled bool

	ch := &mockChannel{
		sendFn: func(_ context.Context, _ bus.OutboundMessage) error {
			return nil
		},
	}

	m.RecordTypingStop("test", "123", func() {
		stopCalled = true
	})

	msg := bus.OutboundMessage{Channel: "test", ChatID: "123", Content: "hello"}
	m.preSend(context.Background(), "test", msg, ch)

	if !stopCalled {
		t.Fatal("expected typing stop func to be called")
	}
}

func TestPreSend_NoRegisteredState(t *testing.T) {
	m := newTestManager()

	ch := &mockChannel{
		sendFn: func(_ context.Context, _ bus.OutboundMessage) error {
			return nil
		},
	}

	msg := bus.OutboundMessage{Channel: "test", ChatID: "123", Content: "hello"}
	edited := m.preSend(context.Background(), "test", msg, ch)

	if edited {
		t.Fatal("expected preSend to return false with no registered state")
	}
}

func TestPreSend_TypingAndPlaceholder(t *testing.T) {
	m := newTestManager()
	var stopCalled bool
	var editCalled bool

	ch := &mockMessageEditor{
		mockChannel: mockChannel{
			sendFn: func(_ context.Context, _ bus.OutboundMessage) error {
				return nil
			},
		},
		editFn: func(_ context.Context, _, _, _ string) error {
			editCalled = true
			return nil
		},
	}

	m.RecordTypingStop("test", "123", func() {
		stopCalled = true
	})
	m.RecordPlaceholder("test", "123", "456")

	msg := bus.OutboundMessage{Channel: "test", ChatID: "123", Content: "hello"}
	edited := m.preSend(context.Background(), "test", msg, ch)

	if !stopCalled {
		t.Fatal("expected typing stop to be called")
	}
	if !editCalled {
		t.Fatal("expected EditMessage to be called")
	}
	if !edited {
		t.Fatal("expected preSend to return true")
	}
}

func TestRecordPlaceholder_ConcurrentSafe(t *testing.T) {
	m := newTestManager()

	var wg sync.WaitGroup
	for i := range 100 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			chatID := fmt.Sprintf("chat_%d", i%10)
			m.RecordPlaceholder("test", chatID, fmt.Sprintf("msg_%d", i))
		}(i)
	}
	wg.Wait()
}

func TestRecordTypingStop_ConcurrentSafe(t *testing.T) {
	m := newTestManager()

	var wg sync.WaitGroup
	for i := range 100 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			chatID := fmt.Sprintf("chat_%d", i%10)
			m.RecordTypingStop("test", chatID, func() {})
		}(i)
	}
	wg.Wait()
}

func TestSendWithRetry_PreSendEditsPlaceholder(t *testing.T) {
	m := newTestManager()
	var sendCalled bool

	ch := &mockMessageEditor{
		mockChannel: mockChannel{
			sendFn: func(_ context.Context, _ bus.OutboundMessage) error {
				sendCalled = true
				return nil
			},
		},
		editFn: func(_ context.Context, _, _, _ string) error {
			return nil // edit succeeds
		},
	}

	m.RecordPlaceholder("test", "123", "456")

	w := &channelWorker{
		ch:      ch,
		limiter: rate.NewLimiter(rate.Inf, 1),
	}

	msg := bus.OutboundMessage{Channel: "test", ChatID: "123", Content: "hello"}
	m.sendWithRetry(context.Background(), "test", w, msg)

	if sendCalled {
		t.Fatal("expected Send to NOT be called when placeholder was edited")
	}
}

// --- Dispatcher exit tests (Step 1) ---

func TestDispatcherExitsOnCancel(t *testing.T) {
	mb := bus.NewMessageBus()
	defer mb.Close()

	m := &Manager{
		channels: make(map[string]Channel),
		workers:  make(map[string]*channelWorker),
		bus:      mb,
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	go func() {
		m.dispatchOutbound(ctx)
		close(done)
	}()

	// Cancel context and verify the dispatcher exits quickly
	cancel()

	select {
	case <-done:
		// success
	case <-time.After(2 * time.Second):
		t.Fatal("dispatchOutbound did not exit within 2s after context cancel")
	}
}

func TestDispatcherMediaExitsOnCancel(t *testing.T) {
	mb := bus.NewMessageBus()
	defer mb.Close()

	m := &Manager{
		channels: make(map[string]Channel),
		workers:  make(map[string]*channelWorker),
		bus:      mb,
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	go func() {
		m.dispatchOutboundMedia(ctx)
		close(done)
	}()

	cancel()

	select {
	case <-done:
		// success
	case <-time.After(2 * time.Second):
		t.Fatal("dispatchOutboundMedia did not exit within 2s after context cancel")
	}
}

// --- TTL Janitor tests (Step 2) ---

func TestTypingStopJanitorEviction(t *testing.T) {
	m := newTestManager()

	var stopCalled atomic.Bool
	// Store a typing entry with a creation time far in the past
	m.typingStops.Store("test:123", typingEntry{
		stop:      func() { stopCalled.Store(true) },
		createdAt: time.Now().Add(-10 * time.Minute), // well past typingStopTTL
	})

	// Run janitor with a short-lived context
	ctx, cancel := context.WithCancel(context.Background())

	// Manually trigger the janitor logic once by simulating a tick
	go func() {
		// Override janitor to run immediately
		now := time.Now()
		m.typingStops.Range(func(key, value any) bool {
			if entry, ok := value.(typingEntry); ok {
				if now.Sub(entry.createdAt) > typingStopTTL {
					if _, loaded := m.typingStops.LoadAndDelete(key); loaded {
						entry.stop()
					}
				}
			}
			return true
		})
		cancel()
	}()

	<-ctx.Done()

	if !stopCalled.Load() {
		t.Fatal("expected typing stop function to be called by janitor eviction")
	}

	// Verify entry was deleted
	if _, loaded := m.typingStops.Load("test:123"); loaded {
		t.Fatal("expected typing entry to be deleted after eviction")
	}
}

func TestPlaceholderJanitorEviction(t *testing.T) {
	m := newTestManager()

	// Store a placeholder entry with a creation time far in the past
	m.placeholders.Store("test:456", placeholderEntry{
		id:        "msg_old",
		createdAt: time.Now().Add(-20 * time.Minute), // well past placeholderTTL
	})

	// Simulate janitor logic
	now := time.Now()
	m.placeholders.Range(func(key, value any) bool {
		if entry, ok := value.(placeholderEntry); ok {
			if now.Sub(entry.createdAt) > placeholderTTL {
				m.placeholders.Delete(key)
			}
		}
		return true
	})

	// Verify entry was deleted
	if _, loaded := m.placeholders.Load("test:456"); loaded {
		t.Fatal("expected placeholder entry to be deleted after eviction")
	}
}

func TestPreSendStillWorksWithWrappedTypes(t *testing.T) {
	m := newTestManager()
	var stopCalled bool
	var editCalled bool

	ch := &mockMessageEditor{
		mockChannel: mockChannel{
			sendFn: func(_ context.Context, _ bus.OutboundMessage) error {
				return nil
			},
		},
		editFn: func(_ context.Context, chatID, messageID, content string) error {
			editCalled = true
			if messageID != "ph_id" {
				t.Fatalf("expected messageID ph_id, got %s", messageID)
			}
			return nil
		},
	}

	// Use the new wrapped types via the public API
	m.RecordTypingStop("test", "chat1", func() {
		stopCalled = true
	})
	m.RecordPlaceholder("test", "chat1", "ph_id")

	msg := bus.OutboundMessage{Channel: "test", ChatID: "chat1", Content: "response"}
	edited := m.preSend(context.Background(), "test", msg, ch)

	if !stopCalled {
		t.Fatal("expected typing stop to be called via wrapped type")
	}
	if !editCalled {
		t.Fatal("expected EditMessage to be called via wrapped type")
	}
	if !edited {
		t.Fatal("expected preSend to return true")
	}
}

// --- Lazy worker creation tests (Step 6) ---

func TestLazyWorkerCreation(t *testing.T) {
	m := newTestManager()

	ch := &mockChannel{
		sendFn: func(_ context.Context, _ bus.OutboundMessage) error {
			return nil
		},
	}

	// RegisterChannel should NOT create a worker
	m.RegisterChannel("lazy", ch)

	m.mu.RLock()
	_, chExists := m.channels["lazy"]
	_, wExists := m.workers["lazy"]
	m.mu.RUnlock()

	if !chExists {
		t.Fatal("expected channel to be registered")
	}
	if wExists {
		t.Fatal("expected worker to NOT be created by RegisterChannel (lazy creation)")
	}
}

// --- FastID uniqueness test (Step 5) ---

func TestBuildMediaScope_FastIDUniqueness(t *testing.T) {
	seen := make(map[string]bool)

	for range 1000 {
		scope := BuildMediaScope("test", "chat1", "")
		if seen[scope] {
			t.Fatalf("duplicate scope generated: %s", scope)
		}
		seen[scope] = true
	}

	// Verify format: "channel:chatID:id"
	scope := BuildMediaScope("telegram", "42", "")
	parts := 0
	for _, c := range scope {
		if c == ':' {
			parts++
		}
	}
	if parts != 2 {
		t.Fatalf("expected scope to have 2 colons (channel:chatID:id), got: %s", scope)
	}
}

func TestBuildMediaScope_WithMessageID(t *testing.T) {
	scope := BuildMediaScope("discord", "chat99", "msg123")
	expected := "discord:chat99:msg123"
	if scope != expected {
		t.Fatalf("expected %s, got %s", expected, scope)
	}
}

// --- Status / TaskStatus message handling tests ---

// mockEditorWithSendID implements MessageEditor and MessageSenderWithID.
type mockEditorWithSendID struct {
	mockChannel
	editFn     func(ctx context.Context, chatID, messageID, content string) error
	sendWithID func(ctx context.Context, chatID, content string) (string, error)
}

func (m *mockEditorWithSendID) EditMessage(ctx context.Context, chatID, messageID, content string) error {
	return m.editFn(ctx, chatID, messageID, content)
}

func (m *mockEditorWithSendID) SendWithID(ctx context.Context, chatID, content string) (string, error) {
	return m.sendWithID(ctx, chatID, content)
}

func TestHandleStatusSend_EditsPlaceholder(t *testing.T) {
	m := newTestManager()
	var editCalled bool
	var editedContent string

	ch := &mockEditorWithSendID{
		mockChannel: mockChannel{
			sendFn: func(_ context.Context, _ bus.OutboundMessage) error { return nil },
		},
		editFn: func(_ context.Context, _, messageID, content string) error {
			editCalled = true
			editedContent = content
			if messageID != "ph-42" {
				t.Fatalf("expected messageID ph-42, got %s", messageID)
			}
			return nil
		},
		sendWithID: func(_ context.Context, _, _ string) (string, error) {
			t.Fatal("SendWithID should not be called when placeholder exists")
			return "", nil
		},
	}

	w := &channelWorker{ch: ch, limiter: rate.NewLimiter(rate.Inf, 1)}

	// Register a placeholder
	m.RecordPlaceholder("test", "123", "ph-42")

	msg := bus.OutboundMessage{Channel: "test", ChatID: "123", Content: "status update 1", IsStatus: true}
	m.handleStatusSend(context.Background(), "test", w, msg)

	if !editCalled {
		t.Fatal("expected EditMessage to be called on placeholder")
	}
	if editedContent != "status update 1" {
		t.Fatalf("expected content 'status update 1', got %s", editedContent)
	}
}

func TestHandleStatusSend_EditsTrackedStatus(t *testing.T) {
	m := newTestManager()
	var editCalled bool

	ch := &mockEditorWithSendID{
		mockChannel: mockChannel{
			sendFn: func(_ context.Context, _ bus.OutboundMessage) error { return nil },
		},
		editFn: func(_ context.Context, _, messageID, _ string) error {
			editCalled = true
			if messageID != "status-99" {
				t.Fatalf("expected messageID status-99, got %s", messageID)
			}
			return nil
		},
		sendWithID: func(_ context.Context, _, _ string) (string, error) {
			t.Fatal("SendWithID should not be called when statusMsgID exists")
			return "", nil
		},
	}

	w := &channelWorker{ch: ch, limiter: rate.NewLimiter(rate.Inf, 1)}

	// Pre-store a tracked status message
	m.statusMsgIDs.Store("test:123", statusMsgEntry{messageID: "status-99", createdAt: time.Now()})

	msg := bus.OutboundMessage{Channel: "test", ChatID: "123", Content: "update 2", IsStatus: true}
	m.handleStatusSend(context.Background(), "test", w, msg)

	if !editCalled {
		t.Fatal("expected EditMessage to be called on tracked status message")
	}
}

func TestHandleStatusSend_SendsNewAndTracks(t *testing.T) {
	m := newTestManager()
	var sendWithIDCalled bool

	ch := &mockEditorWithSendID{
		mockChannel: mockChannel{
			sendFn: func(_ context.Context, _ bus.OutboundMessage) error { return nil },
		},
		editFn: func(_ context.Context, _, _, _ string) error {
			return nil
		},
		sendWithID: func(_ context.Context, chatID, content string) (string, error) {
			sendWithIDCalled = true
			if chatID != "123" {
				t.Fatalf("expected chatID 123, got %s", chatID)
			}
			return "new-msg-1", nil
		},
	}

	w := &channelWorker{ch: ch, limiter: rate.NewLimiter(rate.Inf, 1)}

	// No placeholder, no tracked status -> should use SendWithID
	msg := bus.OutboundMessage{Channel: "test", ChatID: "123", Content: "first status", IsStatus: true}
	m.handleStatusSend(context.Background(), "test", w, msg)

	if !sendWithIDCalled {
		t.Fatal("expected SendWithID to be called")
	}

	// Verify tracked
	v, ok := m.statusMsgIDs.Load("test:123")
	if !ok {
		t.Fatal("expected statusMsgIDs to contain tracked entry")
	}
	entry := v.(statusMsgEntry)
	if entry.messageID != "new-msg-1" {
		t.Fatalf("expected messageID new-msg-1, got %s", entry.messageID)
	}
}

func TestHandleTaskStatusSend_EditsExisting(t *testing.T) {
	m := newTestManager()
	var editCalled bool

	ch := &mockEditorWithSendID{
		mockChannel: mockChannel{
			sendFn: func(_ context.Context, _ bus.OutboundMessage) error { return nil },
		},
		editFn: func(_ context.Context, _, messageID, content string) error {
			editCalled = true
			if messageID != "task-msg-1" {
				t.Fatalf("expected messageID task-msg-1, got %s", messageID)
			}
			if content != "task progress 50%" {
				t.Fatalf("expected content 'task progress 50%%', got %s", content)
			}
			return nil
		},
		sendWithID: func(_ context.Context, _, _ string) (string, error) {
			t.Fatal("SendWithID should not be called when task message exists")
			return "", nil
		},
	}

	w := &channelWorker{ch: ch, limiter: rate.NewLimiter(rate.Inf, 1)}

	// Pre-store task message
	m.taskMsgIDs.Store("task-abc", statusMsgEntry{messageID: "task-msg-1", createdAt: time.Now()})

	msg := bus.OutboundMessage{
		Channel:      "test",
		ChatID:       "123",
		Content:      "task progress 50%",
		IsTaskStatus: true,
		TaskID:       "task-abc",
	}
	m.handleTaskStatusSend(context.Background(), "test", w, msg)

	if !editCalled {
		t.Fatal("expected EditMessage to be called")
	}
}

func TestHandleTaskStatusSend_SendsNewAndTracks(t *testing.T) {
	m := newTestManager()
	var sendWithIDCalled bool

	ch := &mockEditorWithSendID{
		mockChannel: mockChannel{
			sendFn: func(_ context.Context, _ bus.OutboundMessage) error { return nil },
		},
		editFn: func(_ context.Context, _, _, _ string) error { return nil },
		sendWithID: func(_ context.Context, _, _ string) (string, error) {
			sendWithIDCalled = true
			return "new-task-msg", nil
		},
	}

	w := &channelWorker{ch: ch, limiter: rate.NewLimiter(rate.Inf, 1)}

	msg := bus.OutboundMessage{
		Channel:      "test",
		ChatID:       "123",
		Content:      "task started",
		IsTaskStatus: true,
		TaskID:       "task-xyz",
	}
	m.handleTaskStatusSend(context.Background(), "test", w, msg)

	if !sendWithIDCalled {
		t.Fatal("expected SendWithID to be called")
	}

	v, ok := m.taskMsgIDs.Load("task-xyz")
	if !ok {
		t.Fatal("expected taskMsgIDs to contain tracked entry")
	}
	entry := v.(statusMsgEntry)
	if entry.messageID != "new-task-msg" {
		t.Fatalf("expected messageID new-task-msg, got %s", entry.messageID)
	}
}

func TestHandleTaskStatusSend_FallbackToSend(t *testing.T) {
	m := newTestManager()
	var sendCalled bool

	// Channel without SendWithID -- only has Send
	ch := &mockChannel{
		sendFn: func(_ context.Context, msg bus.OutboundMessage) error {
			sendCalled = true
			if msg.Content != "task status" {
				t.Fatalf("expected content 'task status', got %s", msg.Content)
			}
			return nil
		},
	}

	w := &channelWorker{ch: ch, limiter: rate.NewLimiter(rate.Inf, 1)}

	msg := bus.OutboundMessage{
		Channel:      "test",
		ChatID:       "123",
		Content:      "task status",
		IsTaskStatus: true,
		TaskID:       "task-fallback",
	}
	m.handleTaskStatusSend(context.Background(), "test", w, msg)

	if !sendCalled {
		t.Fatal("expected fallback Send to be called")
	}
}

func TestPreSend_EditsStatusMessage(t *testing.T) {
	m := newTestManager()
	var editCalled bool

	ch := &mockMessageEditor{
		mockChannel: mockChannel{
			sendFn: func(_ context.Context, _ bus.OutboundMessage) error { return nil },
		},
		editFn: func(_ context.Context, _, messageID, _ string) error {
			editCalled = true
			if messageID != "status-msg-77" {
				t.Fatalf("expected messageID status-msg-77, got %s", messageID)
			}
			return nil
		},
	}

	// Store a tracked status message
	m.statusMsgIDs.Store("test:123", statusMsgEntry{messageID: "status-msg-77", createdAt: time.Now()})

	msg := bus.OutboundMessage{Channel: "test", ChatID: "123", Content: "final response"}
	edited := m.preSend(context.Background(), "test", msg, ch)

	if !edited {
		t.Fatal("expected preSend to return true (status message edited)")
	}
	if !editCalled {
		t.Fatal("expected EditMessage to be called")
	}

	// Verify status message was consumed (LoadAndDelete)
	if _, loaded := m.statusMsgIDs.Load("test:123"); loaded {
		t.Fatal("expected statusMsgIDs entry to be deleted after preSend")
	}
}

func TestRunWorker_RoutesStatusMessages(t *testing.T) {
	m := newTestManager()

	var regularSendCount atomic.Int32
	var sendWithIDCount atomic.Int32

	ch := &mockEditorWithSendID{
		mockChannel: mockChannel{
			sendFn: func(_ context.Context, _ bus.OutboundMessage) error {
				regularSendCount.Add(1)
				return nil
			},
		},
		editFn: func(_ context.Context, _, _, _ string) error {
			return nil
		},
		sendWithID: func(_ context.Context, _, _ string) (string, error) {
			sendWithIDCount.Add(1)
			return "tracked-1", nil
		},
	}

	w := &channelWorker{
		ch:      ch,
		queue:   make(chan bus.OutboundMessage, 10),
		done:    make(chan struct{}),
		limiter: rate.NewLimiter(rate.Inf, 1),
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go m.runWorker(ctx, "test", w)

	// Send a status message for chatID "1" (routed to handleStatusSend -> SendWithID)
	w.queue <- bus.OutboundMessage{Channel: "test", ChatID: "1", Content: "status", IsStatus: true}
	// Send a task status message for chatID "2" (routed to handleTaskStatusSend -> SendWithID)
	w.queue <- bus.OutboundMessage{Channel: "test", ChatID: "2", Content: "task", IsTaskStatus: true, TaskID: "t1"}
	// Send a regular message for chatID "3" (no tracked status -> regular Send)
	w.queue <- bus.OutboundMessage{Channel: "test", ChatID: "3", Content: "hello"}

	time.Sleep(200 * time.Millisecond)

	if regularSendCount.Load() != 1 {
		t.Fatalf("expected 1 regular Send call, got %d", regularSendCount.Load())
	}
	if sendWithIDCount.Load() != 2 {
		t.Fatalf("expected 2 SendWithID calls (status + task), got %d", sendWithIDCount.Load())
	}
}

func TestStatusMsgTTLJanitor(t *testing.T) {
	m := newTestManager()

	// Store entries with timestamps in the past
	m.statusMsgIDs.Store("test:old", statusMsgEntry{
		messageID: "old-status",
		createdAt: time.Now().Add(-10 * time.Minute),
	})
	m.taskMsgIDs.Store("task-old", statusMsgEntry{
		messageID: "old-task",
		createdAt: time.Now().Add(-60 * time.Minute),
	})
	// Store a fresh entry that should survive
	m.statusMsgIDs.Store("test:fresh", statusMsgEntry{
		messageID: "fresh-status",
		createdAt: time.Now(),
	})

	// Simulate janitor logic
	now := time.Now()
	m.statusMsgIDs.Range(func(key, value any) bool {
		if entry, ok := value.(statusMsgEntry); ok {
			if now.Sub(entry.createdAt) > statusMsgTTL {
				m.statusMsgIDs.Delete(key)
			}
		}
		return true
	})
	m.taskMsgIDs.Range(func(key, value any) bool {
		if entry, ok := value.(statusMsgEntry); ok {
			if now.Sub(entry.createdAt) > taskMsgTTL {
				m.taskMsgIDs.Delete(key)
			}
		}
		return true
	})

	if _, loaded := m.statusMsgIDs.Load("test:old"); loaded {
		t.Fatal("expected old status entry to be evicted")
	}
	if _, loaded := m.taskMsgIDs.Load("task-old"); loaded {
		t.Fatal("expected old task entry to be evicted")
	}
	if _, loaded := m.statusMsgIDs.Load("test:fresh"); !loaded {
		t.Fatal("expected fresh status entry to survive")
	}
}

// --- DraftSender tests ---

// mockDraftSender implements DraftSender + MessageSenderWithID + MessageEditor.
type mockDraftSender struct {
	mockChannel
	draftFn    func(ctx context.Context, chatID string, draftID int, content string) error
	editFn     func(ctx context.Context, chatID, messageID, content string) error
	sendWithID func(ctx context.Context, chatID, content string) (string, error)
}

func (m *mockDraftSender) SendDraft(ctx context.Context, chatID string, draftID int, content string) error {
	return m.draftFn(ctx, chatID, draftID, content)
}

func (m *mockDraftSender) EditMessage(ctx context.Context, chatID, messageID, content string) error {
	return m.editFn(ctx, chatID, messageID, content)
}

func (m *mockDraftSender) SendWithID(ctx context.Context, chatID, content string) (string, error) {
	return m.sendWithID(ctx, chatID, content)
}

func TestHandleStatusSend_UsesDraftSender(t *testing.T) {
	m := newTestManager()
	var draftCalled bool
	var draftContent string
	var draftDID int

	ch := &mockDraftSender{
		mockChannel: mockChannel{
			sendFn: func(_ context.Context, _ bus.OutboundMessage) error { return nil },
		},
		draftFn: func(_ context.Context, chatID string, draftID int, content string) error {
			draftCalled = true
			draftContent = content
			draftDID = draftID
			return nil
		},
		editFn: func(_ context.Context, _, _, _ string) error {
			t.Fatal("EditMessage should not be called when draft succeeds")
			return nil
		},
		sendWithID: func(_ context.Context, _, _ string) (string, error) {
			t.Fatal("SendWithID should not be called when draft succeeds")
			return "", nil
		},
	}

	w := &channelWorker{ch: ch, limiter: rate.NewLimiter(rate.Inf, 1)}

	msg := bus.OutboundMessage{Channel: "test", ChatID: "123", Content: "streaming preview", IsStatus: true}
	m.handleStatusSend(context.Background(), "test", w, msg)

	if !draftCalled {
		t.Fatal("expected SendDraft to be called")
	}
	if draftContent != "streaming preview" {
		t.Fatalf("expected draft content 'streaming preview', got %s", draftContent)
	}
	if draftDID == 0 {
		t.Fatal("expected non-zero draftID")
	}

	// Second call should reuse the same draftID
	draftCalled = false
	var secondDID int
	ch.draftFn = func(_ context.Context, _ string, draftID int, _ string) error {
		draftCalled = true
		secondDID = draftID
		return nil
	}
	msg.Content = "streaming preview updated"
	m.handleStatusSend(context.Background(), "test", w, msg)

	if !draftCalled {
		t.Fatal("expected SendDraft to be called again")
	}
	if secondDID != draftDID {
		t.Fatalf("expected same draftID %d, got %d", draftDID, secondDID)
	}
}

func TestHandleStatusSend_DraftFails_FallsToEdit(t *testing.T) {
	m := newTestManager()
	var editCalled bool

	ch := &mockDraftSender{
		mockChannel: mockChannel{
			sendFn: func(_ context.Context, _ bus.OutboundMessage) error { return nil },
		},
		draftFn: func(_ context.Context, _ string, _ int, _ string) error {
			return fmt.Errorf("draft not supported in group")
		},
		editFn: func(_ context.Context, _, _, _ string) error {
			editCalled = true
			return nil
		},
		sendWithID: func(_ context.Context, _, _ string) (string, error) {
			return "msg-1", nil
		},
	}

	w := &channelWorker{ch: ch, limiter: rate.NewLimiter(rate.Inf, 1)}

	// No existing placeholder/status — draft fails, then SendWithID
	msg := bus.OutboundMessage{Channel: "test", ChatID: "123", Content: "preview", IsStatus: true}
	m.handleStatusSend(context.Background(), "test", w, msg)

	// Draft failed, so it should fall through; no placeholder → no edit → SendWithID
	if editCalled {
		t.Fatal("expected EditMessage NOT to be called (no placeholder)")
	}
}

func TestHandleTaskStatusSend_UsesDraftSender(t *testing.T) {
	m := newTestManager()
	var draftCalled bool

	ch := &mockDraftSender{
		mockChannel: mockChannel{
			sendFn: func(_ context.Context, _ bus.OutboundMessage) error { return nil },
		},
		draftFn: func(_ context.Context, _ string, _ int, _ string) error {
			draftCalled = true
			return nil
		},
		editFn: func(_ context.Context, _, _, _ string) error {
			t.Fatal("EditMessage should not be called when draft succeeds")
			return nil
		},
		sendWithID: func(_ context.Context, _, _ string) (string, error) {
			t.Fatal("SendWithID should not be called when draft succeeds")
			return "", nil
		},
	}

	w := &channelWorker{ch: ch, limiter: rate.NewLimiter(rate.Inf, 1)}

	msg := bus.OutboundMessage{
		Channel:      "test",
		ChatID:       "123",
		Content:      "task progress 50%",
		IsTaskStatus: true,
		TaskID:       "task-draft",
	}
	m.handleTaskStatusSend(context.Background(), "test", w, msg)

	if !draftCalled {
		t.Fatal("expected SendDraft to be called for task status")
	}
}

func TestPreSend_ClearsDraftState(t *testing.T) {
	m := newTestManager()

	ch := &mockChannel{
		sendFn: func(_ context.Context, _ bus.OutboundMessage) error { return nil },
	}

	// Store a draft-based status entry (draftID != 0, messageID empty)
	m.statusMsgIDs.Store("test:123", statusMsgEntry{draftID: 42, createdAt: time.Now()})

	msg := bus.OutboundMessage{Channel: "test", ChatID: "123", Content: "final response"}
	edited := m.preSend(context.Background(), "test", msg, ch)

	// Draft-based entries don't trigger edit; the final sendMessage replaces the draft
	if edited {
		t.Fatal("expected preSend to return false for draft-based status (sendMessage replaces draft)")
	}

	// Verify draft state was consumed
	if _, loaded := m.statusMsgIDs.Load("test:123"); loaded {
		t.Fatal("expected draft status entry to be deleted after preSend")
	}
}

func TestGenerateDraftID_Stable(t *testing.T) {
	id1 := generateDraftID("telegram:123")
	id2 := generateDraftID("telegram:123")
	if id1 != id2 {
		t.Fatalf("expected stable draft ID, got %d vs %d", id1, id2)
	}
	if id1 == 0 {
		t.Fatal("expected non-zero draft ID")
	}

	// Different key should produce different ID
	id3 := generateDraftID("telegram:456")
	if id1 == id3 {
		t.Fatalf("expected different draft IDs for different keys, both got %d", id1)
	}
}

// TestPreSend_DismissesDraftBeforeSend verifies that preSend explicitly
// dismisses a draft-based status bubble (via SendDraft with empty text)
// before proceeding to send the permanent message.  This prevents ghost
// draft bubbles when a user message arrives between the last draft update
// and the final sendMessage.
func TestPreSend_DismissesDraftBeforeSend(t *testing.T) {
	m := newTestManager()

	var dismissCalled bool
	var dismissContent string

	ch := &mockDraftSender{
		mockChannel: mockChannel{
			sendFn: func(_ context.Context, _ bus.OutboundMessage) error { return nil },
		},
		draftFn: func(_ context.Context, _ string, _ int, content string) error {
			dismissCalled = true
			dismissContent = content
			return nil
		},
		editFn:     func(_ context.Context, _, _, _ string) error { return nil },
		sendWithID: func(_ context.Context, _, _ string) (string, error) { return "", nil },
	}

	// Store a draft-based status entry (simulates active streaming)
	m.statusMsgIDs.Store("test:123", statusMsgEntry{draftID: 42, createdAt: time.Now()})

	msg := bus.OutboundMessage{Channel: "test", ChatID: "123", Content: "final response"}
	edited := m.preSend(context.Background(), "test", msg, ch)

	if edited {
		t.Fatal("expected preSend to return false for draft-based status")
	}
	if !dismissCalled {
		t.Fatal("expected preSend to call SendDraft to dismiss the draft")
	}
	if dismissContent != "" {
		t.Fatalf("expected empty dismiss content, got %q", dismissContent)
	}
}

// TestRecordTypingStop_CleansUpOldEntry verifies that recording a new
// typing stop function calls the previous stop first.
func TestRecordTypingStop_CleansUpOldEntry(t *testing.T) {
	m := newTestManager()

	var oldStopped atomic.Bool

	m.RecordTypingStop("tg", "42", func() { oldStopped.Store(true) })

	// Record a new one — old stop should fire
	m.RecordTypingStop("tg", "42", func() {})

	if !oldStopped.Load() {
		t.Fatal("expected old typing stop to be called when new entry is recorded")
	}
}

// TestRecordReactionUndo_CleansUpOldEntry verifies that recording a new
// reaction undo function calls the previous undo first.
func TestRecordReactionUndo_CleansUpOldEntry(t *testing.T) {
	m := newTestManager()

	var oldUndone atomic.Bool

	m.RecordReactionUndo("tg", "42", func() { oldUndone.Store(true) })

	// Record a new one — old undo should fire
	m.RecordReactionUndo("tg", "42", func() {})

	if !oldUndone.Load() {
		t.Fatal("expected old reaction undo to be called when new entry is recorded")
	}
}

// TestPreSend_DraftDismiss_ClearsEditTimes verifies that dismissing a draft
// in preSend also clears the statusEditTimes entry for that key, preventing
// stale throttle state from affecting the next processing cycle.
func TestPreSend_DraftDismiss_ClearsEditTimes(t *testing.T) {
	m := newTestManager()

	ch := &mockDraftSender{
		mockChannel: mockChannel{
			sendFn: func(_ context.Context, _ bus.OutboundMessage) error { return nil },
		},
		draftFn:    func(_ context.Context, _ string, _ int, _ string) error { return nil },
		editFn:     func(_ context.Context, _, _, _ string) error { return nil },
		sendWithID: func(_ context.Context, _, _ string) (string, error) { return "", nil },
	}

	key := "test:123"
	m.statusMsgIDs.Store(key, statusMsgEntry{draftID: 42, createdAt: time.Now()})
	m.statusEditTimes.Store(key, time.Now())

	msg := bus.OutboundMessage{Channel: "test", ChatID: "123", Content: "final"}
	m.preSend(context.Background(), "test", msg, ch)

	if _, loaded := m.statusEditTimes.Load(key); loaded {
		t.Fatal("expected statusEditTimes to be cleared after draft dismiss")
	}
}

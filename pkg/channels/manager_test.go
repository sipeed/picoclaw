package channels

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/time/rate"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/utils"
)

// mockChannel is a test double that delegates Send to a configurable function.
type mockChannel struct {
	BaseChannel
	sendFn            func(ctx context.Context, msg bus.OutboundMessage) error
	startFn           func(ctx context.Context) error
	stopFn            func(ctx context.Context) error
	sentMessages      []bus.OutboundMessage
	placeholdersSent  int
	editedMessages    int
	lastPlaceholderID string
}

func (m *mockChannel) Send(ctx context.Context, msg bus.OutboundMessage) ([]string, error) {
	m.sentMessages = append(m.sentMessages, msg)
	if m.sendFn == nil {
		return nil, nil
	}
	return nil, m.sendFn(ctx, msg)
}

func (m *mockChannel) Start(ctx context.Context) error {
	if m.startFn != nil {
		return m.startFn(ctx)
	}
	return nil
}

func (m *mockChannel) Stop(ctx context.Context) error {
	if m.stopFn != nil {
		return m.stopFn(ctx)
	}
	return nil
}

func (m *mockChannel) SendPlaceholder(ctx context.Context, chatID string) (string, error) {
	m.placeholdersSent++
	m.lastPlaceholderID = "mock-ph-123"
	return m.lastPlaceholderID, nil
}

func (m *mockChannel) EditMessage(ctx context.Context, chatID, messageID, content string) error {
	m.editedMessages++
	return nil
}

type mockMediaChannel struct {
	mockChannel
	sendMediaFn       func(ctx context.Context, msg bus.OutboundMediaMessage) ([]string, error)
	sentMediaMessages []bus.OutboundMediaMessage
}

func (m *mockMediaChannel) SendMedia(ctx context.Context, msg bus.OutboundMediaMessage) ([]string, error) {
	m.sentMediaMessages = append(m.sentMediaMessages, msg)
	if m.sendMediaFn != nil {
		return m.sendMediaFn(ctx, msg)
	}
	return nil, nil
}

type mockDeletingMediaChannel struct {
	mockMediaChannel
	deleteCalls     int
	dismissedChatID string
	lastDeleted     struct {
		chatID    string
		messageID string
	}
}

func (m *mockDeletingMediaChannel) DeleteMessage(
	_ context.Context,
	chatID string,
	messageID string,
) error {
	m.deleteCalls++
	m.lastDeleted.chatID = chatID
	m.lastDeleted.messageID = messageID
	return nil
}

func (m *mockDeletingMediaChannel) DismissToolFeedbackMessage(_ context.Context, chatID string) {
	m.dismissedChatID = chatID
}

type mockStreamer struct {
	finalizeFn func(context.Context, string) error
}

func (m *mockStreamer) Update(context.Context, string) error { return nil }

func (m *mockStreamer) Finalize(ctx context.Context, content string) error {
	if m.finalizeFn != nil {
		return m.finalizeFn(ctx, content)
	}
	return nil
}

func (m *mockStreamer) Cancel(context.Context) {}

type mockStreamingChannel struct {
	mockMessageEditor
	streamer        Streamer
	resolveChatIDFn func(chatID string, outboundCtx *bus.InboundContext) string
}

func (m *mockStreamingChannel) BeginStream(context.Context, string) (Streamer, error) {
	if m.streamer == nil {
		return nil, errors.New("missing streamer")
	}
	return m.streamer, nil
}

func (m *mockStreamingChannel) ToolFeedbackMessageChatID(
	chatID string,
	outboundCtx *bus.InboundContext,
) string {
	if m.resolveChatIDFn != nil {
		return m.resolveChatIDFn(chatID, outboundCtx)
	}
	return chatID
}

// newTestManager creates a minimal Manager suitable for unit tests.
func newTestManager() *Manager {
	return &Manager{
		channels: make(map[string]Channel),
		workers:  make(map[string]*channelWorker),
		bus:      bus.NewMessageBus(),
	}
}

func TestStartAll_AllChannelsFail_ReturnsJoinedError(t *testing.T) {
	m := newTestManager()
	errA := errors.New("channel-a start failed")
	errB := errors.New("channel-b start failed")

	m.channels["a"] = &mockChannel{
		startFn: func(_ context.Context) error { return errA },
	}
	m.channels["b"] = &mockChannel{
		startFn: func(_ context.Context) error { return errB },
	}

	err := m.StartAll(t.Context())
	if err == nil {
		t.Fatal("expected StartAll to fail when all channels fail")
	}
	if !strings.Contains(err.Error(), "failed to start any enabled channels") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !errors.Is(err, errA) {
		t.Fatalf("expected error to wrap errA, got: %v", err)
	}
	if !errors.Is(err, errB) {
		t.Fatalf("expected error to wrap errB, got: %v", err)
	}
	if len(m.workers) != 0 {
		t.Fatalf("expected no workers on full startup failure, got %d", len(m.workers))
	}
	if m.dispatchTask != nil {
		t.Fatal("expected dispatch task to be cleared on full startup failure")
	}
}

func TestStartAll_PartialFailure_StartsSuccessfulWorkers(t *testing.T) {
	m := newTestManager()
	errBad := errors.New("bad channel start failed")
	processed := make(chan struct{}, 1)

	m.channels["good"] = &mockChannel{
		sendFn: func(_ context.Context, msg bus.OutboundMessage) error {
			if msg.Channel == "good" {
				select {
				case processed <- struct{}{}:
				default:
				}
			}
			return nil
		},
	}
	m.channels["bad"] = &mockChannel{
		startFn: func(_ context.Context) error { return errBad },
	}

	err := m.StartAll(t.Context())
	if err != nil {
		t.Fatalf("expected StartAll to succeed with partial channel failures, got: %v", err)
	}
	if len(m.workers) != 1 {
		t.Fatalf("expected exactly 1 active worker, got %d", len(m.workers))
	}
	if _, ok := m.workers["good"]; !ok {
		t.Fatal("expected worker for successful channel 'good'")
	}
	if _, ok := m.workers["bad"]; ok {
		t.Fatal("did not expect worker for failed channel 'bad'")
	}
	if m.dispatchTask == nil {
		t.Fatal("expected dispatch task to run when at least one channel starts")
	}

	pubCtx, pubCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer pubCancel()
	if err := m.bus.PublishOutbound(pubCtx, testOutboundMessage(bus.OutboundMessage{
		Channel: "good",
		ChatID:  "chat-1",
		Content: "hello",
	})); err != nil {
		t.Fatalf("PublishOutbound() error = %v", err)
	}

	select {
	case <-processed:
		// worker processed outbound message as expected
	case <-time.After(2 * time.Second):
		t.Fatal("expected successful channel worker to process outbound message")
	}

	stopCtx, stopCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer stopCancel()
	if err := m.StopAll(stopCtx); err != nil {
		t.Fatalf("StopAll() error = %v", err)
	}
}

func testOutboundMessage(msg bus.OutboundMessage) bus.OutboundMessage {
	if msg.Context.Channel == "" && msg.Context.ChatID == "" {
		msg.Context = bus.NewOutboundContext(msg.Channel, msg.ChatID, msg.ReplyToMessageID)
	}
	return bus.NormalizeOutboundMessage(msg)
}

func testOutboundMediaMessage(msg bus.OutboundMediaMessage) bus.OutboundMediaMessage {
	if msg.Context.Channel == "" && msg.Context.ChatID == "" {
		msg.Context = bus.NewOutboundContext(msg.Channel, msg.ChatID, "")
	}
	return bus.NormalizeOutboundMediaMessage(msg)
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
	msg := testOutboundMessage(bus.OutboundMessage{Channel: "test", ChatID: "1", Content: "hello"})

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
	msg := testOutboundMessage(bus.OutboundMessage{Channel: "test", ChatID: "1", Content: "hello"})

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
	msg := testOutboundMessage(bus.OutboundMessage{Channel: "test", ChatID: "1", Content: "hello"})

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
	msg := testOutboundMessage(bus.OutboundMessage{Channel: "test", ChatID: "1", Content: "hello"})

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
	msg := testOutboundMessage(bus.OutboundMessage{Channel: "test", ChatID: "1", Content: "hello"})

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
	msg := testOutboundMessage(bus.OutboundMessage{Channel: "test", ChatID: "1", Content: "hello"})

	m.sendWithRetry(ctx, "test", w, msg)

	expected := maxRetries + 1 // initial attempt + maxRetries retries
	if callCount != expected {
		t.Fatalf("expected %d Send calls, got %d", expected, callCount)
	}
}

func TestSendMedia_Success(t *testing.T) {
	m := newTestManager()
	var callCount int
	ch := &mockMediaChannel{
		sendMediaFn: func(_ context.Context, _ bus.OutboundMediaMessage) ([]string, error) {
			callCount++
			return nil, nil
		},
	}
	w := &channelWorker{
		ch:      ch,
		limiter: rate.NewLimiter(rate.Inf, 1),
	}
	m.channels["test"] = ch
	m.workers["test"] = w

	err := m.SendMedia(context.Background(), testOutboundMediaMessage(bus.OutboundMediaMessage{
		Channel: "test",
		ChatID:  "chat1",
		Parts:   []bus.MediaPart{{Ref: "media://abc"}},
	}))
	if err != nil {
		t.Fatalf("SendMedia() error = %v", err)
	}
	if callCount != 1 {
		t.Fatalf("expected 1 SendMedia call, got %d", callCount)
	}
}

func TestSendMedia_PropagatesFailure(t *testing.T) {
	m := newTestManager()
	ch := &mockMediaChannel{
		sendMediaFn: func(_ context.Context, _ bus.OutboundMediaMessage) ([]string, error) {
			return nil, fmt.Errorf("bad upload: %w", ErrSendFailed)
		},
	}
	w := &channelWorker{
		ch:      ch,
		limiter: rate.NewLimiter(rate.Inf, 1),
	}
	m.channels["test"] = ch
	m.workers["test"] = w

	err := m.SendMedia(context.Background(), testOutboundMediaMessage(bus.OutboundMediaMessage{
		Channel: "test",
		ChatID:  "chat1",
		Parts:   []bus.MediaPart{{Ref: "media://abc"}},
	}))
	if err == nil {
		t.Fatal("expected SendMedia to return error")
	}
	if !errors.Is(err, ErrSendFailed) {
		t.Fatalf("expected ErrSendFailed, got %v", err)
	}
}

func TestSendMedia_UnsupportedChannelReturnsError(t *testing.T) {
	m := newTestManager()
	ch := &mockChannel{
		sendFn: func(_ context.Context, _ bus.OutboundMessage) error {
			return nil
		},
	}
	w := &channelWorker{
		ch:      ch,
		limiter: rate.NewLimiter(rate.Inf, 1),
	}
	m.channels["test"] = ch
	m.workers["test"] = w

	err := m.SendMedia(context.Background(), testOutboundMediaMessage(bus.OutboundMediaMessage{
		Channel: "test",
		ChatID:  "chat1",
		Parts:   []bus.MediaPart{{Ref: "media://abc"}},
	}))
	if err == nil {
		t.Fatal("expected SendMedia to return error for unsupported channel")
	}
	if !strings.Contains(err.Error(), "does not support media sending") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSendMedia_DeletesPlaceholderBeforeSending(t *testing.T) {
	m := newTestManager()
	ch := &mockDeletingMediaChannel{
		mockMediaChannel: mockMediaChannel{
			sendMediaFn: func(_ context.Context, _ bus.OutboundMediaMessage) ([]string, error) {
				return nil, nil
			},
		},
	}
	w := &channelWorker{
		ch:      ch,
		limiter: rate.NewLimiter(rate.Inf, 1),
	}
	m.channels["test"] = ch
	m.workers["test"] = w
	m.RecordPlaceholder("test", "chat1", "placeholder-1")

	err := m.SendMedia(context.Background(), testOutboundMediaMessage(bus.OutboundMediaMessage{
		Channel: "test",
		ChatID:  "chat1",
		Parts:   []bus.MediaPart{{Ref: "media://abc"}},
	}))
	if err != nil {
		t.Fatalf("SendMedia() error = %v", err)
	}
	if ch.deleteCalls != 1 {
		t.Fatalf("expected placeholder delete to be called once, got %d", ch.deleteCalls)
	}
	if ch.lastDeleted.chatID != "chat1" || ch.lastDeleted.messageID != "placeholder-1" {
		t.Fatalf("unexpected placeholder deletion target: %+v", ch.lastDeleted)
	}
	if len(ch.sentMediaMessages) != 1 {
		t.Fatalf("expected media to be sent once, got %d", len(ch.sentMediaMessages))
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
	msg := testOutboundMessage(bus.OutboundMessage{Channel: "test", ChatID: "1", Content: "hello"})

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
	msg := testOutboundMessage(bus.OutboundMessage{Channel: "test", ChatID: "1", Content: "hello"})

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
		w.queue <- testOutboundMessage(bus.OutboundMessage{Channel: "test", ChatID: "1", Content: fmt.Sprintf("msg%d", i)})
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
	w := newChannelWorker("unknown_channel", ch, "unknown_channel")

	if w.limiter == nil {
		t.Fatal("expected limiter to be non-nil")
	}
	if w.limiter.Limit() != rate.Limit(defaultRateLimit) {
		t.Fatalf("expected rate limit %v, got %v", rate.Limit(defaultRateLimit), w.limiter.Limit())
	}
}

func TestNewChannelWorker_ConfiguredRate(t *testing.T) {
	ch := &mockChannel{}

	for channelType, expectedRate := range channelRateConfig {
		w := newChannelWorker(channelType, ch, channelType)
		if w.limiter.Limit() != rate.Limit(expectedRate) {
			t.Fatalf("channel %s: expected rate %v, got %v", channelType, expectedRate, w.limiter.Limit())
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
	w.queue <- testOutboundMessage(bus.OutboundMessage{Channel: "test", ChatID: "1", Content: "hello world"})

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
	msg := testOutboundMessage(bus.OutboundMessage{Channel: "test", ChatID: "1", Content: "hello"})

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
	editFn            func(ctx context.Context, chatID, messageID, content string) error
	finalizeFn        func(ctx context.Context, msg bus.OutboundMessage) ([]string, bool)
	finalizeCalled    bool
	recordedChatID    string
	recordedMessageID string
	recordedContent   string
	clearedChatID     string
	dismissedChatID   string
}

func (m *mockMessageEditor) EditMessage(ctx context.Context, chatID, messageID, content string) error {
	return m.editFn(ctx, chatID, messageID, content)
}

func (m *mockMessageEditor) RecordToolFeedbackMessage(chatID, messageID, content string) {
	m.recordedChatID = chatID
	m.recordedMessageID = messageID
	m.recordedContent = content
}

func (m *mockMessageEditor) ClearToolFeedbackMessage(chatID string) {
	m.clearedChatID = chatID
}

func (m *mockMessageEditor) DismissToolFeedbackMessage(_ context.Context, chatID string) {
	m.dismissedChatID = chatID
}

func (m *mockMessageEditor) FinalizeToolFeedbackMessage(
	ctx context.Context,
	msg bus.OutboundMessage,
) ([]string, bool) {
	m.finalizeCalled = true
	if m.finalizeFn == nil {
		return nil, false
	}
	return m.finalizeFn(ctx, msg)
}

type mockResolvedToolFeedbackEditor struct {
	mockMessageEditor
	resolveChatIDFn func(chatID string, outboundCtx *bus.InboundContext) string
}

type mockDeletingMessageEditor struct {
	mockMessageEditor
	deleteCalls      int
	deletedChatID    string
	deletedMessageID string
}

func (m *mockDeletingMessageEditor) DeleteMessage(_ context.Context, chatID, messageID string) error {
	m.deleteCalls++
	m.deletedChatID = chatID
	m.deletedMessageID = messageID
	return nil
}

func (m *mockResolvedToolFeedbackEditor) ToolFeedbackMessageChatID(
	chatID string,
	outboundCtx *bus.InboundContext,
) string {
	if m.resolveChatIDFn != nil {
		return m.resolveChatIDFn(chatID, outboundCtx)
	}
	return chatID
}

type mockPreparedToolFeedbackEditor struct {
	mockMessageEditor
	prepareFn func(content string) string
}

func (m *mockPreparedToolFeedbackEditor) PrepareToolFeedbackMessageContent(content string) string {
	if m.prepareFn != nil {
		return m.prepareFn(content)
	}
	return content
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

	msg := testOutboundMessage(bus.OutboundMessage{Channel: "test", ChatID: "123", Content: "hello"})
	_, edited := m.preSend(context.Background(), "test", msg, ch)

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

func TestPreSend_ToolFeedbackPlaceholderEditRecordsTrackedMessage(t *testing.T) {
	m := newTestManager()

	ch := &mockMessageEditor{
		editFn: func(_ context.Context, chatID, messageID, content string) error {
			if chatID != "123" || messageID != "456" || content != "hello" {
				t.Fatalf("unexpected edit args: %s %s %s", chatID, messageID, content)
			}
			return nil
		},
	}

	m.RecordPlaceholder("test", "123", "456")

	msg := testOutboundMessage(bus.OutboundMessage{
		Channel: "test",
		ChatID:  "123",
		Content: "hello",
		Context: bus.InboundContext{
			Channel: "test",
			ChatID:  "123",
			Raw: map[string]string{
				"message_kind": "tool_feedback",
			},
		},
	})
	_, edited := m.preSend(context.Background(), "test", msg, ch)
	if !edited {
		t.Fatal("expected preSend to edit placeholder")
	}
	if ch.recordedChatID != "123" || ch.recordedMessageID != "456" {
		t.Fatalf("expected tracked message 123/456, got %q/%q", ch.recordedChatID, ch.recordedMessageID)
	}
}

func TestPreSend_ToolFeedbackPlaceholderEditUsesResolvedTrackedChatID(t *testing.T) {
	m := newTestManager()

	ch := &mockResolvedToolFeedbackEditor{
		mockMessageEditor: mockMessageEditor{
			editFn: func(_ context.Context, chatID, messageID, content string) error {
				if chatID != "-100123" || messageID != "456" || content != "hello" {
					t.Fatalf("unexpected edit args: %s %s %s", chatID, messageID, content)
				}
				return nil
			},
		},
		resolveChatIDFn: func(chatID string, outboundCtx *bus.InboundContext) string {
			if chatID != "-100123" {
				t.Fatalf("expected raw chat ID, got %q", chatID)
			}
			if outboundCtx == nil || outboundCtx.TopicID != "42" {
				t.Fatalf("expected topic-aware outbound context, got %+v", outboundCtx)
			}
			return chatID + "/" + outboundCtx.TopicID
		},
	}

	m.RecordPlaceholder("test", "-100123", "456")

	msg := testOutboundMessage(bus.OutboundMessage{
		Channel: "test",
		ChatID:  "-100123",
		Content: "hello",
		Context: bus.InboundContext{
			Channel: "test",
			ChatID:  "-100123",
			TopicID: "42",
			Raw: map[string]string{
				"message_kind": "tool_feedback",
			},
		},
	})
	_, edited := m.preSend(context.Background(), "test", msg, ch)
	if !edited {
		t.Fatal("expected preSend to edit placeholder")
	}
	if ch.recordedChatID != "-100123/42" || ch.recordedMessageID != "456" {
		t.Fatalf("expected resolved tracked message -100123/42/456, got %q/%q",
			ch.recordedChatID, ch.recordedMessageID)
	}
}

func TestPreSend_ToolFeedbackPlaceholderEditUsesPreparedContent(t *testing.T) {
	m := newTestManager()

	const rawContent = "🔧 `read_file`\n" + "<raw>"
	const preparedContent = "🔧 `read_file`\n&lt;raw&gt;"

	ch := &mockPreparedToolFeedbackEditor{
		mockMessageEditor: mockMessageEditor{
			editFn: func(_ context.Context, chatID, messageID, content string) error {
				if chatID != "123" || messageID != "456" {
					t.Fatalf("unexpected edit target: %s/%s", chatID, messageID)
				}
				if content != InitialAnimatedToolFeedbackContent(preparedContent) {
					t.Fatalf("unexpected prepared content: %q", content)
				}
				return nil
			},
		},
		prepareFn: func(content string) string {
			if content != rawContent {
				t.Fatalf("unexpected raw tool feedback: %q", content)
			}
			return preparedContent
		},
	}

	m.RecordPlaceholder("test", "123", "456")

	msg := testOutboundMessage(bus.OutboundMessage{
		Channel: "test",
		ChatID:  "123",
		Content: rawContent,
		Context: bus.InboundContext{
			Channel: "test",
			ChatID:  "123",
			Raw: map[string]string{
				"message_kind": "tool_feedback",
			},
		},
	})

	_, edited := m.preSend(context.Background(), "test", msg, ch)
	if !edited {
		t.Fatal("expected preSend to edit placeholder")
	}
	if ch.recordedContent != preparedContent {
		t.Fatalf("expected tracked content %q, got %q", preparedContent, ch.recordedContent)
	}
}

func TestPreSend_NonToolFeedbackLeavesTrackedMessageForChannelSend(t *testing.T) {
	m := newTestManager()
	ch := &mockMessageEditor{}

	msg := testOutboundMessage(bus.OutboundMessage{
		Channel: "test",
		ChatID:  "123",
		Content: "final reply",
		Context: bus.InboundContext{
			Channel: "test",
			ChatID:  "123",
		},
	})

	_, edited := m.preSend(context.Background(), "test", msg, ch)
	if edited {
		t.Fatal("expected preSend to fall through when no placeholder exists")
	}
	if ch.dismissedChatID != "" {
		t.Fatalf("expected tracked tool feedback cleanup to be deferred to channel send, got %q", ch.dismissedChatID)
	}
}

func TestPreSend_NonToolFeedbackDefersTrackedMessageFinalizationToChannelSend(t *testing.T) {
	m := newTestManager()
	ch := &mockMessageEditor{
		finalizeFn: func(_ context.Context, msg bus.OutboundMessage) ([]string, bool) {
			if msg.ChatID != "123" || msg.Content != "final reply" {
				t.Fatalf("unexpected finalize msg: %+v", msg)
			}
			return []string{"tool-msg-1"}, true
		},
	}

	msg := testOutboundMessage(bus.OutboundMessage{
		Channel: "test",
		ChatID:  "123",
		Content: "final reply",
		Context: bus.InboundContext{
			Channel: "test",
			ChatID:  "123",
		},
	})

	msgIDs, handled := m.preSend(context.Background(), "test", msg, ch)
	if handled {
		t.Fatalf("expected preSend to defer to channel Send, got msgIDs=%v", msgIDs)
	}
	if len(msgIDs) != 0 {
		t.Fatalf("expected no msgIDs from preSend, got %v", msgIDs)
	}
	if ch.dismissedChatID != "" {
		t.Fatalf("expected tracked cleanup to remain in channel Send, got %q", ch.dismissedChatID)
	}
	if ch.finalizeCalled {
		t.Fatal("expected preSend to skip channel tool feedback finalization")
	}
}

func TestPreSend_ToolFeedbackSeparateMessagesDeletesPlaceholderAndSkipsEdit(t *testing.T) {
	m := newTestManager()
	m.config = &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				ToolFeedback: config.ToolFeedbackConfig{
					Enabled:          true,
					SeparateMessages: true,
				},
			},
		},
	}

	ch := &mockDeletingMessageEditor{
		mockMessageEditor: mockMessageEditor{
			editFn: func(_ context.Context, _, _, _ string) error {
				t.Fatal("expected placeholder edit to be skipped in separate message mode")
				return nil
			},
		},
	}

	m.RecordPlaceholder("test", "123", "456")

	msg := testOutboundMessage(bus.OutboundMessage{
		Channel: "test",
		ChatID:  "123",
		Content: "hello",
		Context: bus.InboundContext{
			Channel: "test",
			ChatID:  "123",
			Raw: map[string]string{
				"message_kind": "tool_feedback",
			},
		},
	})

	msgIDs, handled := m.preSend(context.Background(), "test", msg, ch)
	if handled {
		t.Fatalf("expected preSend to fall through so the channel can send a new message, got %v", msgIDs)
	}
	if ch.deleteCalls != 1 {
		t.Fatalf("expected placeholder deletion, got %d delete calls", ch.deleteCalls)
	}
	if ch.deletedChatID != "123" || ch.deletedMessageID != "456" {
		t.Fatalf("unexpected placeholder deletion target: %s/%s", ch.deletedChatID, ch.deletedMessageID)
	}
	if ch.recordedMessageID != "" {
		t.Fatalf("expected no tracked placeholder record, got %q", ch.recordedMessageID)
	}
	if ch.clearedChatID != "123" {
		t.Fatalf("expected tracked tool feedback state to be cleared before sending, got %q", ch.clearedChatID)
	}
}

func TestPreSend_ThoughtPlaceholderDeleteAndSkipsEdit(t *testing.T) {
	m := newTestManager()

	ch := &mockDeletingMessageEditor{
		mockMessageEditor: mockMessageEditor{
			editFn: func(_ context.Context, _, _, _ string) error {
				t.Fatal("expected thought message to bypass placeholder edit")
				return nil
			},
		},
	}

	m.RecordPlaceholder("test", "123", "456")

	msg := testOutboundMessage(bus.OutboundMessage{
		Channel: "test",
		ChatID:  "123",
		Content: "thinking trace",
		Context: bus.InboundContext{
			Channel: "test",
			ChatID:  "123",
			Raw: map[string]string{
				"message_kind": "thought",
			},
		},
	})

	msgIDs, handled := m.preSend(context.Background(), "test", msg, ch)
	if handled {
		t.Fatalf(
			"expected thought message to fall through so the channel can send a structured message, got %v",
			msgIDs,
		)
	}
	if ch.deleteCalls != 1 {
		t.Fatalf("expected placeholder deletion, got %d delete calls", ch.deleteCalls)
	}
	if ch.deletedChatID != "123" || ch.deletedMessageID != "456" {
		t.Fatalf("unexpected placeholder deletion target: %s/%s", ch.deletedChatID, ch.deletedMessageID)
	}
	if _, ok := m.placeholders.Load("test:123"); ok {
		t.Fatal("expected placeholder to be consumed before structured thought send")
	}
}

func TestSendWithRetry_ToolCallsPlaceholderDeleteAndFallsThroughToSend(t *testing.T) {
	m := newTestManager()

	ch := &mockDeletingMessageEditor{
		mockMessageEditor: mockMessageEditor{
			mockChannel: mockChannel{
				sendFn: func(_ context.Context, msg bus.OutboundMessage) error {
					if got := msg.Context.Raw["message_kind"]; got != "tool_calls" {
						t.Fatalf("expected tool_calls message kind, got %q", got)
					}
					if msg.Content != "" {
						t.Fatalf("expected empty tool_calls content, got %q", msg.Content)
					}
					return nil
				},
			},
			editFn: func(_ context.Context, _, _, _ string) error {
				t.Fatal("expected tool_calls message to bypass placeholder edit")
				return nil
			},
		},
	}

	m.RecordPlaceholder("test", "123", "456")

	w := &channelWorker{
		ch:      ch,
		limiter: rate.NewLimiter(rate.Inf, 1),
	}

	msg := testOutboundMessage(bus.OutboundMessage{
		Channel: "test",
		ChatID:  "123",
		Context: bus.InboundContext{
			Channel: "test",
			ChatID:  "123",
			Raw: map[string]string{
				"message_kind": "tool_calls",
				"tool_calls":   `[{"id":"call_1","type":"function","function":{"name":"read_file","arguments":"{}"},"extra_content":{"tool_feedback_explanation":"Looking up config"}}]`,
			},
		},
	})

	m.sendWithRetry(context.Background(), "test", w, msg)

	if ch.deleteCalls != 1 {
		t.Fatalf("expected placeholder deletion, got %d delete calls", ch.deleteCalls)
	}
	if ch.deletedChatID != "123" || ch.deletedMessageID != "456" {
		t.Fatalf("unexpected placeholder deletion target: %s/%s", ch.deletedChatID, ch.deletedMessageID)
	}
	if len(ch.sentMessages) != 1 {
		t.Fatalf("expected structured tool_calls message to be sent once, got %d", len(ch.sentMessages))
	}
}

func TestPreSend_NonToolFeedbackSeparateMessagesClearsTrackedMessageWithoutDismiss(t *testing.T) {
	m := newTestManager()
	m.config = &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				ToolFeedback: config.ToolFeedbackConfig{
					Enabled:          true,
					SeparateMessages: true,
				},
			},
		},
	}

	ch := &mockMessageEditor{}

	msg := testOutboundMessage(bus.OutboundMessage{
		Channel: "test",
		ChatID:  "123",
		Content: "final reply",
		Context: bus.InboundContext{
			Channel: "test",
			ChatID:  "123",
		},
	})

	_, handled := m.preSend(context.Background(), "test", msg, ch)
	if handled {
		t.Fatal("expected preSend to leave final delivery to the channel")
	}
	if ch.clearedChatID != "123" {
		t.Fatalf("expected tracked tool feedback state to be cleared, got %q", ch.clearedChatID)
	}
	if ch.dismissedChatID != "" {
		t.Fatalf("expected tracked tool feedback message to be preserved, got dismissal for %q", ch.dismissedChatID)
	}
	if ch.finalizeCalled {
		t.Fatal("expected separate message mode to skip in-place finalization")
	}
}

func TestPreSend_StaleToolFeedbackDoesNotConsumeStreamActiveMarker(t *testing.T) {
	m := newTestManager()
	m.streamActive.Store("test:123", true)
	m.RecordPlaceholder("test", "123", "placeholder-1")

	var editedContent string
	ch := &mockMessageEditor{
		editFn: func(_ context.Context, chatID, messageID, content string) error {
			if chatID != "123" || messageID != "placeholder-1" {
				t.Fatalf("unexpected edit target: %s/%s", chatID, messageID)
			}
			editedContent = content
			return nil
		},
	}

	toolFeedback := testOutboundMessage(bus.OutboundMessage{
		Channel: "test",
		ChatID:  "123",
		Content: "🔧 `read_file`\nReading config",
		Context: bus.InboundContext{
			Channel: "test",
			ChatID:  "123",
			Raw: map[string]string{
				"message_kind": "tool_feedback",
			},
		},
	})

	msgIDs, handled := m.preSend(context.Background(), "test", toolFeedback, ch)
	if !handled {
		t.Fatal("expected stale tool feedback to be dropped after stream finalize")
	}
	if len(msgIDs) != 0 {
		t.Fatalf("expected no delivered message IDs for stale feedback, got %v", msgIDs)
	}
	if _, ok := m.streamActive.Load("test:123"); !ok {
		t.Fatal("expected streamActive marker to remain for the final outbound message")
	}
	if _, ok := m.placeholders.Load("test:123"); !ok {
		t.Fatal("expected placeholder cleanup to remain deferred to the final outbound message")
	}
	if ch.editedMessages != 0 {
		t.Fatalf("expected no placeholder edit for stale feedback, got %d edits", ch.editedMessages)
	}

	finalMsg := testOutboundMessage(bus.OutboundMessage{
		Channel: "test",
		ChatID:  "123",
		Content: "final streamed reply",
		Context: bus.InboundContext{
			Channel: "test",
			ChatID:  "123",
		},
	})

	_, handled = m.preSend(context.Background(), "test", finalMsg, ch)
	if !handled {
		t.Fatal("expected final outbound message to consume streamActive marker")
	}
	if _, ok := m.streamActive.Load("test:123"); ok {
		t.Fatal("expected streamActive marker to be cleared by final outbound message")
	}
	if _, ok := m.placeholders.Load("test:123"); ok {
		t.Fatal("expected placeholder to be cleaned up by final outbound message")
	}
	if editedContent != "final streamed reply" {
		t.Fatalf("editedContent = %q, want final streamed reply", editedContent)
	}
}

func TestPreSendMedia_LeavesTrackedMessageForChannelSend(t *testing.T) {
	m := newTestManager()
	ch := &mockDeletingMediaChannel{}

	m.preSendMedia(context.Background(), "test", bus.OutboundMediaMessage{
		ChatID: "123",
		Context: bus.InboundContext{
			Channel: "test",
			ChatID:  "123",
		},
	}, ch)

	if ch.dismissedChatID != "" {
		t.Fatalf(
			"expected tracked tool feedback cleanup to be deferred to channel media send, got %q",
			ch.dismissedChatID,
		)
	}
}

func TestPreSendMedia_SeparateMessagesClearsTrackedMessageWithoutDismiss(t *testing.T) {
	m := newTestManager()
	m.config = &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				ToolFeedback: config.ToolFeedbackConfig{
					Enabled:          true,
					SeparateMessages: true,
				},
			},
		},
	}

	ch := &mockMessageEditor{}

	m.preSendMedia(context.Background(), "test", bus.OutboundMediaMessage{
		ChatID: "123",
		Context: bus.InboundContext{
			Channel: "test",
			ChatID:  "123",
		},
	}, ch)

	if ch.clearedChatID != "123" {
		t.Fatalf("expected tracked tool feedback state to be cleared before media delivery, got %q", ch.clearedChatID)
	}
	if ch.dismissedChatID != "" {
		t.Fatalf("expected tracked tool feedback message to be preserved"+
			" for media delivery, got %q", ch.dismissedChatID)
	}
}

func TestSplitOutboundMessageContent_ToolFeedbackTruncatesInsteadOfSplitting(t *testing.T) {
	msg := testOutboundMessage(bus.OutboundMessage{
		Channel: "test",
		ChatID:  "123",
		Content: "\U0001f527 `read_file`\nRead README.md first to confirm the current project structure before editing the config example.",
		Context: bus.InboundContext{
			Channel: "test",
			ChatID:  "123",
			Raw: map[string]string{
				"message_kind": "tool_feedback",
			},
		},
	})

	chunks := splitOutboundMessageContent(msg, 40)
	if len(chunks) != 1 {
		t.Fatalf("len(chunks) = %d, want 1", len(chunks))
	}
	want := utils.FitToolFeedbackMessage(msg.Content, 40-MaxToolFeedbackAnimationFrameLength())
	if chunks[0] != want {
		t.Fatalf("chunk = %q, want %q", chunks[0], want)
	}
}

func TestSplitOutboundMessageContent_ToolFeedbackReservesAnimationFrame(t *testing.T) {
	msg := testOutboundMessage(bus.OutboundMessage{
		Channel: "test",
		ChatID:  "123",
		Content: "🔧 `read_file`\n1234567890",
		Context: bus.InboundContext{
			Channel: "test",
			ChatID:  "123",
			Raw: map[string]string{
				"message_kind": "tool_feedback",
			},
		},
	})

	chunks := splitOutboundMessageContent(msg, len([]rune(msg.Content)))
	if len(chunks) != 1 {
		t.Fatalf("len(chunks) = %d, want 1", len(chunks))
	}

	animated := formatAnimatedToolFeedbackContent(chunks[0], strings.Repeat(".", MaxToolFeedbackAnimationFrameLength()))
	if got, maxLen := len([]rune(animated)), len([]rune(msg.Content)); got > maxLen {
		t.Fatalf("animated len = %d, want <= %d; content=%q", got, maxLen, animated)
	}
}

func TestGetStreamer_FinalizeDismissesTrackedToolFeedback(t *testing.T) {
	m := newTestManager()
	ch := &mockStreamingChannel{
		mockMessageEditor: mockMessageEditor{},
		streamer: &mockStreamer{
			finalizeFn: func(_ context.Context, content string) error {
				if content != "final reply" {
					t.Fatalf("unexpected finalize content: %q", content)
				}
				return nil
			},
		},
	}
	m.channels["test"] = ch

	streamer, ok := m.GetStreamer(context.Background(), "test", "123")
	if !ok {
		t.Fatal("expected streamer to be available")
	}
	if err := streamer.Finalize(context.Background(), "final reply"); err != nil {
		t.Fatalf("Finalize() error = %v", err)
	}
	if ch.dismissedChatID != "123" {
		t.Fatalf("expected tracked tool feedback to be dismissed for chat 123, got %q", ch.dismissedChatID)
	}
	if _, ok := m.streamActive.Load("test:123"); !ok {
		t.Fatal("expected streamActive marker to be recorded after finalize")
	}
}

func TestGetStreamer_FinalizeSeparateMessagesClearsTrackedToolFeedback(t *testing.T) {
	m := newTestManager()
	m.config = &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				ToolFeedback: config.ToolFeedbackConfig{
					Enabled:          true,
					SeparateMessages: true,
				},
			},
		},
	}
	ch := &mockStreamingChannel{
		mockMessageEditor: mockMessageEditor{},
		streamer: &mockStreamer{
			finalizeFn: func(_ context.Context, content string) error {
				if content != "final reply" {
					t.Fatalf("unexpected finalize content: %q", content)
				}
				return nil
			},
		},
	}
	m.channels["test"] = ch

	streamer, ok := m.GetStreamer(context.Background(), "test", "123")
	if !ok {
		t.Fatal("expected streamer to be available")
	}
	if err := streamer.Finalize(context.Background(), "final reply"); err != nil {
		t.Fatalf("Finalize() error = %v", err)
	}
	if ch.clearedChatID != "123" {
		t.Fatalf("expected tracked tool feedback to be cleared for chat 123, got %q", ch.clearedChatID)
	}
	if ch.dismissedChatID != "" {
		t.Fatalf("expected tracked tool feedback message to be preserved, got dismissal for %q", ch.dismissedChatID)
	}
	if _, ok := m.streamActive.Load("test:123"); !ok {
		t.Fatal("expected streamActive marker to be recorded after finalize")
	}
}

func TestGetStreamer_FinalizeDismissesResolvedTrackedToolFeedback(t *testing.T) {
	m := newTestManager()
	ch := &mockStreamingChannel{
		mockMessageEditor: mockMessageEditor{},
		streamer: &mockStreamer{
			finalizeFn: func(_ context.Context, content string) error {
				if content != "final reply" {
					t.Fatalf("unexpected finalize content: %q", content)
				}
				return nil
			},
		},
		resolveChatIDFn: func(chatID string, outboundCtx *bus.InboundContext) string {
			if outboundCtx == nil {
				t.Fatal("expected outbound context during stream finalize")
			}
			if outboundCtx.ChatID != "-100123/42" {
				t.Fatalf("unexpected outbound context: %+v", outboundCtx)
			}
			return outboundCtx.ChatID
		},
	}
	m.channels["test"] = ch

	streamer, ok := m.GetStreamer(context.Background(), "test", "-100123/42")
	if !ok {
		t.Fatal("expected streamer to be available")
	}
	if err := streamer.Finalize(context.Background(), "final reply"); err != nil {
		t.Fatalf("Finalize() error = %v", err)
	}
	if ch.dismissedChatID != "-100123/42" {
		t.Fatalf("expected resolved tracked tool feedback dismissal, got %q", ch.dismissedChatID)
	}
	if _, ok := m.streamActive.Load("test:-100123/42"); !ok {
		t.Fatal("expected streamActive marker to be recorded after finalize")
	}
}

func TestPreSend_PlaceholderEditSuccessDismissesResolvedTrackedToolFeedback(t *testing.T) {
	m := newTestManager()

	ch := &mockResolvedToolFeedbackEditor{
		mockMessageEditor: mockMessageEditor{
			editFn: func(_ context.Context, chatID, messageID, content string) error {
				if chatID != "-100123" || messageID != "456" || content != "done" {
					t.Fatalf("unexpected edit args: %s %s %s", chatID, messageID, content)
				}
				return nil
			},
		},
		resolveChatIDFn: func(chatID string, outboundCtx *bus.InboundContext) string {
			if outboundCtx == nil || outboundCtx.TopicID != "42" {
				t.Fatalf("expected topic-aware outbound context, got %+v", outboundCtx)
			}
			return chatID + "/" + outboundCtx.TopicID
		},
	}

	m.RecordPlaceholder("test", "-100123", "456")

	msg := testOutboundMessage(bus.OutboundMessage{
		Channel: "test",
		ChatID:  "-100123",
		Content: "done",
		Context: bus.InboundContext{
			Channel: "test",
			ChatID:  "-100123",
			TopicID: "42",
		},
	})

	_, edited := m.preSend(context.Background(), "test", msg, ch)
	if !edited {
		t.Fatal("expected preSend to edit placeholder")
	}
	if ch.dismissedChatID != "-100123/42" {
		t.Fatalf("expected resolved tracked dismissal, got %q", ch.dismissedChatID)
	}
}

func TestGetStreamer_FinalizeFailureDoesNotDismissTrackedToolFeedback(t *testing.T) {
	m := newTestManager()
	ch := &mockStreamingChannel{
		mockMessageEditor: mockMessageEditor{},
		streamer: &mockStreamer{
			finalizeFn: func(context.Context, string) error {
				return errors.New("finalize failed")
			},
		},
	}
	m.channels["test"] = ch

	streamer, ok := m.GetStreamer(context.Background(), "test", "123")
	if !ok {
		t.Fatal("expected streamer to be available")
	}
	if err := streamer.Finalize(context.Background(), "final reply"); err == nil {
		t.Fatal("expected Finalize() to fail")
	}
	if ch.dismissedChatID != "" {
		t.Fatalf("expected no tool feedback dismissal on finalize failure, got %q", ch.dismissedChatID)
	}
	if _, ok := m.streamActive.Load("test:123"); ok {
		t.Fatal("expected no streamActive marker after finalize failure")
	}
}

func TestRunWorker_ToolFeedbackSkipsMarkerSplitting(t *testing.T) {
	m := newTestManager()
	m.config = &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				SplitOnMarker: true,
			},
		},
	}

	var (
		mu       sync.Mutex
		received []string
	)
	ch := &mockChannelWithLength{
		mockChannel: mockChannel{
			sendFn: func(_ context.Context, msg bus.OutboundMessage) error {
				mu.Lock()
				received = append(received, msg.Content)
				mu.Unlock()
				return nil
			},
		},
		maxLen: 200,
	}

	w := &channelWorker{
		ch:      ch,
		queue:   make(chan bus.OutboundMessage, 1),
		done:    make(chan struct{}),
		limiter: rate.NewLimiter(rate.Inf, 1),
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go m.runWorker(ctx, "test", w)

	content := "🔧 `read_file`\nRead current config first.<|[SPLIT]|>Then update the example."
	w.queue <- testOutboundMessage(bus.OutboundMessage{
		Channel: "test",
		ChatID:  "123",
		Content: content,
		Context: bus.InboundContext{
			Channel: "test",
			ChatID:  "123",
			Raw: map[string]string{
				"message_kind": "tool_feedback",
			},
		},
	})

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if len(received) != 1 {
		t.Fatalf("len(received) = %d, want 1", len(received))
	}
	if received[0] != content {
		t.Fatalf("received[0] = %q, want %q", received[0], content)
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

	msg := testOutboundMessage(bus.OutboundMessage{Channel: "test", ChatID: "123", Content: "hello"})
	_, edited := m.preSend(context.Background(), "test", msg, ch)

	if edited {
		t.Fatal("expected preSend to return false when edit fails")
	}
}

func TestInvokeTypingStop_CallsRegisteredStop(t *testing.T) {
	m := newTestManager()
	var stopCalled bool

	m.RecordTypingStop("telegram", "chat123", func() {
		stopCalled = true
	})

	m.InvokeTypingStop("telegram", "chat123")

	if !stopCalled {
		t.Fatal("expected typing stop func to be called")
	}
}

func TestInvokeTypingStop_NoOpWhenNoEntry(t *testing.T) {
	m := newTestManager()
	// Should not panic
	m.InvokeTypingStop("telegram", "nonexistent")
}

func TestInvokeTypingStop_Idempotent(t *testing.T) {
	m := newTestManager()
	var callCount int

	m.RecordTypingStop("telegram", "chat123", func() {
		callCount++
	})

	m.InvokeTypingStop("telegram", "chat123")
	m.InvokeTypingStop("telegram", "chat123") // Second call: entry already removed, no-op

	if callCount != 1 {
		t.Fatalf("expected stop to be called once, got %d", callCount)
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

	msg := testOutboundMessage(bus.OutboundMessage{Channel: "test", ChatID: "123", Content: "hello"})
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

	msg := testOutboundMessage(bus.OutboundMessage{Channel: "test", ChatID: "123", Content: "hello"})
	_, edited := m.preSend(context.Background(), "test", msg, ch)

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

	msg := testOutboundMessage(bus.OutboundMessage{Channel: "test", ChatID: "123", Content: "hello"})
	_, edited := m.preSend(context.Background(), "test", msg, ch)

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

func TestRecordTypingStop_ReplacesExistingStop(t *testing.T) {
	m := newTestManager()
	var oldStopCalls int
	var newStopCalls int

	m.RecordTypingStop("test", "123", func() {
		oldStopCalls++
	})

	m.RecordTypingStop("test", "123", func() {
		newStopCalls++
	})

	if oldStopCalls != 1 {
		t.Fatalf("expected previous typing stop to be called once when replaced, got %d", oldStopCalls)
	}
	if newStopCalls != 0 {
		t.Fatalf("expected replacement typing stop to stay active until preSend, got %d calls", newStopCalls)
	}

	msg := testOutboundMessage(bus.OutboundMessage{Channel: "test", ChatID: "123", Content: "hello"})
	m.preSend(context.Background(), "test", msg, &mockChannel{})

	if newStopCalls != 1 {
		t.Fatalf("expected replacement typing stop to be called by preSend, got %d", newStopCalls)
	}
	if oldStopCalls != 1 {
		t.Fatalf("expected previous typing stop to not be called again, got %d", oldStopCalls)
	}
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

	msg := testOutboundMessage(bus.OutboundMessage{Channel: "test", ChatID: "123", Content: "hello"})
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

	msg := testOutboundMessage(bus.OutboundMessage{Channel: "test", ChatID: "chat1", Content: "response"})
	_, edited := m.preSend(context.Background(), "test", msg, ch)

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

func TestManager_PlaceholderConsumedByResponse(t *testing.T) {
	mgr := &Manager{
		channels:     make(map[string]Channel),
		workers:      make(map[string]*channelWorker),
		placeholders: sync.Map{},
	}

	mockCh := &mockChannel{
		sendFn: func(ctx context.Context, msg bus.OutboundMessage) error {
			return nil
		},
	}
	worker := newChannelWorker("mock", mockCh, "mock")
	mgr.channels["mock"] = mockCh
	mgr.workers["mock"] = worker

	ctx := context.Background()
	key := "mock:chat-1"

	// Simulate a placeholder recorded by base.go HandleMessage
	mgr.RecordPlaceholder("mock", "chat-1", "ph-123")

	if _, ok := mgr.placeholders.Load(key); !ok {
		t.Fatal("expected placeholder to be recorded")
	}

	// Transcription feedback arrives first — it should consume the placeholder
	// and be delivered via EditMessage, not Send.
	msgTranscript := testOutboundMessage(bus.OutboundMessage{
		Channel: "mock",
		ChatID:  "chat-1",
		Content: "Transcript: hello",
	})
	mgr.sendWithRetry(ctx, "mock", worker, msgTranscript)

	if mockCh.editedMessages != 1 {
		t.Errorf("expected 1 edited message (placeholder consumed by transcript), got %d", mockCh.editedMessages)
	}
	if len(mockCh.sentMessages) != 0 {
		t.Errorf("expected 0 normal messages (transcript used edit), got %d", len(mockCh.sentMessages))
	}

	// Placeholder should be gone now
	if _, ok := mgr.placeholders.Load(key); ok {
		t.Error("expected placeholder to be removed after being consumed")
	}

	// Final LLM response arrives — no placeholder left, so it goes through Send
	msgFinal := testOutboundMessage(bus.OutboundMessage{
		Channel: "mock",
		ChatID:  "chat-1",
		Content: "Final Answer",
	})
	mgr.sendWithRetry(ctx, "mock", worker, msgFinal)

	if len(mockCh.sentMessages) != 1 {
		t.Errorf("expected 1 normal message sent, got %d", len(mockCh.sentMessages))
	}
}

func TestSendMessage_Synchronous(t *testing.T) {
	m := newTestManager()

	var received []bus.OutboundMessage
	ch := &mockChannel{
		sendFn: func(_ context.Context, msg bus.OutboundMessage) error {
			received = append(received, msg)
			return nil
		},
	}

	w := &channelWorker{
		ch:      ch,
		limiter: rate.NewLimiter(rate.Inf, 1),
	}
	m.channels["test"] = ch
	m.workers["test"] = w

	msg := testOutboundMessage(bus.OutboundMessage{
		Channel:          "test",
		ChatID:           "123",
		Content:          "hello world",
		ReplyToMessageID: "msg-456",
	})

	err := m.SendMessage(context.Background(), msg)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// SendMessage is synchronous — message should already be delivered
	if len(received) != 1 {
		t.Fatalf("expected 1 message sent, got %d", len(received))
	}
	if received[0].ReplyToMessageID != "msg-456" {
		t.Fatalf("expected ReplyToMessageID msg-456, got %s", received[0].ReplyToMessageID)
	}
	if received[0].Content != "hello world" {
		t.Fatalf("expected content 'hello world', got %s", received[0].Content)
	}
}

func TestSendMessage_UnknownChannel(t *testing.T) {
	m := newTestManager()

	msg := testOutboundMessage(bus.OutboundMessage{
		Channel: "nonexistent",
		ChatID:  "123",
		Content: "hello",
	})

	err := m.SendMessage(context.Background(), msg)
	if err == nil {
		t.Fatal("expected error for unknown channel")
	}
}

func TestSendMessage_NoWorker(t *testing.T) {
	m := newTestManager()

	ch := &mockChannel{
		sendFn: func(_ context.Context, _ bus.OutboundMessage) error { return nil },
	}
	m.channels["test"] = ch
	// No worker registered

	msg := testOutboundMessage(bus.OutboundMessage{
		Channel: "test",
		ChatID:  "123",
		Content: "hello",
	})

	err := m.SendMessage(context.Background(), msg)
	if err == nil {
		t.Fatal("expected error when no worker exists")
	}
}

func TestSendMessage_WithRetry(t *testing.T) {
	m := newTestManager()

	var callCount int
	ch := &mockChannel{
		sendFn: func(_ context.Context, _ bus.OutboundMessage) error {
			callCount++
			if callCount == 1 {
				return fmt.Errorf("transient: %w", ErrTemporary)
			}
			return nil
		},
	}

	w := &channelWorker{
		ch:      ch,
		limiter: rate.NewLimiter(rate.Inf, 1),
	}
	m.channels["test"] = ch
	m.workers["test"] = w

	msg := testOutboundMessage(bus.OutboundMessage{
		Channel: "test",
		ChatID:  "123",
		Content: "retry me",
	})

	err := m.SendMessage(context.Background(), msg)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if callCount != 2 {
		t.Fatalf("expected 2 Send calls (1 failure + 1 success), got %d", callCount)
	}
}

func TestSendMessage_ContextOnlyUsesContextAddressing(t *testing.T) {
	m := newTestManager()

	var received []bus.OutboundMessage
	ch := &mockChannel{
		sendFn: func(_ context.Context, msg bus.OutboundMessage) error {
			received = append(received, msg)
			return nil
		},
	}

	w := &channelWorker{
		ch:      ch,
		limiter: rate.NewLimiter(rate.Inf, 1),
	}
	m.channels["test"] = ch
	m.workers["test"] = w

	msg := testOutboundMessage(bus.OutboundMessage{
		Context: bus.NewOutboundContext("test", "123", "msg-9"),
		Content: "hello",
	})

	if err := m.SendMessage(context.Background(), msg); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(received) != 1 {
		t.Fatalf("expected 1 message sent, got %d", len(received))
	}
	if received[0].Channel != "test" || received[0].ChatID != "123" {
		t.Fatalf("expected mirrored legacy address, got %+v", received[0])
	}
	if received[0].Context.Channel != "test" || received[0].Context.ChatID != "123" {
		t.Fatalf("expected context address to be preserved, got %+v", received[0].Context)
	}
	if received[0].ReplyToMessageID != "msg-9" {
		t.Fatalf("expected reply_to_message_id msg-9, got %q", received[0].ReplyToMessageID)
	}
}

func TestSendMessage_WithSplitting(t *testing.T) {
	m := newTestManager()

	var received []string
	ch := &mockChannelWithLength{
		mockChannel: mockChannel{
			sendFn: func(_ context.Context, msg bus.OutboundMessage) error {
				received = append(received, msg.Content)
				return nil
			},
		},
		maxLen: 5,
	}

	w := &channelWorker{
		ch:      ch,
		limiter: rate.NewLimiter(rate.Inf, 1),
	}
	m.channels["test"] = ch
	m.workers["test"] = w

	msg := testOutboundMessage(bus.OutboundMessage{
		Channel: "test",
		ChatID:  "123",
		Content: "hello world",
	})

	err := m.SendMessage(context.Background(), msg)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(received) < 2 {
		t.Fatalf("expected message to be split into at least 2 chunks, got %d", len(received))
	}
}

func TestSendMedia_ContextOnlyUsesContextAddressing(t *testing.T) {
	m := newTestManager()

	var received []bus.OutboundMediaMessage
	ch := &mockMediaChannel{
		sendMediaFn: func(_ context.Context, msg bus.OutboundMediaMessage) ([]string, error) {
			received = append(received, msg)
			return nil, nil
		},
	}

	w := &channelWorker{
		ch:      ch,
		limiter: rate.NewLimiter(rate.Inf, 1),
	}
	m.channels["test"] = ch
	m.workers["test"] = w

	msg := testOutboundMediaMessage(bus.OutboundMediaMessage{
		Context: bus.NewOutboundContext("test", "media-chat", ""),
		Parts:   []bus.MediaPart{{Type: "image", Ref: "media://1"}},
	})

	if err := m.SendMedia(context.Background(), msg); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(received) != 1 {
		t.Fatalf("expected 1 media message sent, got %d", len(received))
	}
	if received[0].Channel != "test" || received[0].ChatID != "media-chat" {
		t.Fatalf("expected mirrored legacy media address, got %+v", received[0])
	}
	if received[0].Context.Channel != "test" || received[0].Context.ChatID != "media-chat" {
		t.Fatalf("expected media context address to be preserved, got %+v", received[0].Context)
	}
}

func TestSendMessage_PreservesOrdering(t *testing.T) {
	m := newTestManager()

	var order []string
	ch := &mockChannel{
		sendFn: func(_ context.Context, msg bus.OutboundMessage) error {
			order = append(order, msg.Content)
			return nil
		},
	}

	w := &channelWorker{
		ch:      ch,
		limiter: rate.NewLimiter(rate.Inf, 1),
	}
	m.channels["test"] = ch
	m.workers["test"] = w

	// Send two messages sequentially — they must arrive in order
	_ = m.SendMessage(context.Background(), testOutboundMessage(bus.OutboundMessage{
		Channel: "test", ChatID: "1", Content: "first",
	}))
	_ = m.SendMessage(context.Background(), testOutboundMessage(bus.OutboundMessage{
		Channel: "test", ChatID: "1", Content: "second",
	}))

	if len(order) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(order))
	}
	if order[0] != "first" || order[1] != "second" {
		t.Fatalf("expected [first, second], got %v", order)
	}
}

func TestManager_SendPlaceholder(t *testing.T) {
	mgr := &Manager{
		channels:     make(map[string]Channel),
		workers:      make(map[string]*channelWorker),
		placeholders: sync.Map{},
	}

	mockCh := &mockChannel{
		sendFn: func(ctx context.Context, msg bus.OutboundMessage) error {
			return nil
		},
	}
	mgr.channels["mock"] = mockCh

	ctx := context.Background()

	// SendPlaceholder should send a placeholder and record it
	ok := mgr.SendPlaceholder(ctx, "mock", "chat-1")
	if !ok {
		t.Fatal("expected SendPlaceholder to succeed")
	}
	if mockCh.placeholdersSent != 1 {
		t.Errorf("expected 1 placeholder sent, got %d", mockCh.placeholdersSent)
	}

	key := "mock:chat-1"
	if _, loaded := mgr.placeholders.Load(key); !loaded {
		t.Error("expected placeholder to be recorded in manager")
	}

	// SendPlaceholder on unknown channel should return false
	ok = mgr.SendPlaceholder(ctx, "unknown", "chat-1")
	if ok {
		t.Error("expected SendPlaceholder to fail for unknown channel")
	}
}

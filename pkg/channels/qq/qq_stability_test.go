package qq

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/openapi/options"
	"golang.org/x/oauth2"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
)

// mockQQAPI implements qqAPI for testing
type mockQQAPI struct {
	callCount     atomic.Int32
	wsCalled      atomic.Bool
	postCalled    atomic.Bool
	lastChatID    string
	lastChatKind  string
	shouldFail    bool
	failErr       error
	failRemaining atomic.Int32
}

func (m *mockQQAPI) WS(ctx context.Context, params map[string]string, body string) (*dto.WebsocketAP, error) {
	m.wsCalled.Store(true)
	return &dto.WebsocketAP{
		Shards: 1,
	}, nil
}

func (m *mockQQAPI) PostGroupMessage(
	ctx context.Context, groupID string, msg dto.APIMessage, opt ...options.Option,
) (*dto.Message, error) {
	m.postCalled.Store(true)
	m.lastChatID = groupID
	m.lastChatKind = "group"
	m.callCount.Add(1)

	if m.shouldFail {
		remaining := m.failRemaining.Add(-1)
		if remaining >= 0 {
			return nil, m.failErr
		}
	}

	return &dto.Message{ID: "msg-123"}, nil
}

func (m *mockQQAPI) PostC2CMessage(
	ctx context.Context, userID string, msg dto.APIMessage, opt ...options.Option,
) (*dto.Message, error) {
	m.postCalled.Store(true)
	m.lastChatID = userID
	m.lastChatKind = "direct"
	m.callCount.Add(1)

	if m.shouldFail {
		remaining := m.failRemaining.Add(-1)
		if remaining >= 0 {
			return nil, m.failErr
		}
	}

	return &dto.Message{ID: "msg-123"}, nil
}

func (m *mockQQAPI) Transport(ctx context.Context, method, url string, body any) ([]byte, error) {
	return []byte(`{"file_info": "test-file-info"}`), nil
}

func newTestQQChannel(t *testing.T, api *mockQQAPI) *QQChannel {
	cfg := config.QQConfig{
		AppID:     "test-app-id",
		AppSecret: "test-app-secret",
	}

	ch := &QQChannel{
		BaseChannel:       channels.NewBaseChannel("qq", cfg, nil, config.FlexibleStringSlice{"*"}),
		config:            cfg,
		api:               api,
		dedup:             make(map[string]time.Time),
		done:              make(chan struct{}),
		groupRateLimiter:  newRateLimiter(500 * time.Millisecond),
		directRateLimiter: newRateLimiter(200 * time.Millisecond),
	}
	ch.SetRunning(true)

	return ch
}

// TestSend_Success tests successful send without retry
func TestSend_Success(t *testing.T) {
	api := &mockQQAPI{}
	ch := newTestQQChannel(t, api)

	ch.chatType.Store("group-1", "group")

	err := ch.Send(context.Background(), bus.OutboundMessage{
		ChatID:  "group-1",
		Content: "Hello",
	})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if !api.postCalled.Load() {
		t.Error("expected PostGroupMessage to be called")
	}
}

// TestSend_WithRetry tests that retries happen on transient errors
func TestSend_WithRetry(t *testing.T) {
	api := &mockQQAPI{
		shouldFail:    true,
		failErr:       errors.New("connection reset"),
		failRemaining: atomic.Int32{},
	}
	api.failRemaining.Store(2) // Fail first 2 times, succeed on 3rd

	ch := newTestQQChannel(t, api)
	ch.chatType.Store("group-1", "group")

	err := ch.Send(context.Background(), bus.OutboundMessage{
		ChatID:  "group-1",
		Content: "Hello",
	})
	// Should succeed after retries
	if err != nil {
		t.Errorf("expected success after retry, got %v", err)
	}
	if api.callCount.Load() != 3 {
		t.Errorf("expected 3 calls, got %d", api.callCount.Load())
	}
}

// TestSend_NonRetryableError tests that non-retryable errors don't retry
func TestSend_NonRetryableError(t *testing.T) {
	api := &mockQQAPI{
		shouldFail:    true,
		failErr:       errors.New("unauthorized"),
		failRemaining: atomic.Int32{},
	}
	api.failRemaining.Store(10) // Will keep failing

	ch := newTestQQChannel(t, api)
	ch.chatType.Store("group-1", "group")

	err := ch.Send(context.Background(), bus.OutboundMessage{
		ChatID:  "group-1",
		Content: "Hello",
	})

	// Should fail after first attempt (unauthorized is not retryable)
	if err == nil {
		t.Error("expected error for unauthorized")
	}
	if api.callCount.Load() > 1 {
		t.Errorf("expected only 1 call for non-retryable error, got %d", api.callCount.Load())
	}
}

// TestSend_ContextCancellation tests that retries stop on context cancellation
func TestSend_ContextCancellation(t *testing.T) {
	api := &mockQQAPI{
		shouldFail:    true,
		failErr:       errors.New("timeout"),
		failRemaining: atomic.Int32{},
	}
	api.failRemaining.Store(100) // Keep failing

	ch := newTestQQChannel(t, api)
	ch.chatType.Store("group-1", "group")

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := ch.Send(ctx, bus.OutboundMessage{
		ChatID:  "group-1",
		Content: "Hello",
	})

	// Should fail due to context cancellation
	if err == nil {
		t.Error("expected error due to context cancellation")
	}
}

// TestSend_RateLimiting tests that rate limiting delays messages
func TestSend_RateLimiting(t *testing.T) {
	api := &mockQQAPI{}
	ch := newTestQQChannel(t, api)
	ch.chatType.Store("group-1", "group")

	start := time.Now()

	// Send first message
	err := ch.Send(context.Background(), bus.OutboundMessage{
		ChatID:  "group-1",
		Content: "Message 1",
	})
	if err != nil {
		t.Errorf("first send failed: %v", err)
	}

	// Send second message immediately
	err = ch.Send(context.Background(), bus.OutboundMessage{
		ChatID:  "group-1",
		Content: "Message 2",
	})
	if err != nil {
		t.Errorf("second send failed: %v", err)
	}

	elapsed := time.Since(start)

	// Should have waited at least 400ms (group rate limit is 500ms)
	if elapsed < 400*time.Millisecond {
		t.Errorf("expected at least 400ms delay due to rate limiting, got %v", elapsed)
	}
}

// TestSend_DifferentChatsNoRateLimit tests that different chats don't rate limit each other
func TestSend_DifferentChatsNoRateLimit(t *testing.T) {
	api := &mockQQAPI{}
	ch := newTestQQChannel(t, api)
	ch.chatType.Store("group-1", "group")
	ch.chatType.Store("group-2", "group")

	start := time.Now()

	// Send to different chats
	err := ch.Send(context.Background(), bus.OutboundMessage{
		ChatID:  "group-1",
		Content: "Message 1",
	})
	if err != nil {
		t.Errorf("first send failed: %v", err)
	}

	err = ch.Send(context.Background(), bus.OutboundMessage{
		ChatID:  "group-2",
		Content: "Message 2",
	})
	if err != nil {
		t.Errorf("second send failed: %v", err)
	}

	elapsed := time.Since(start)

	// Should not have waited much (different chats)
	if elapsed > 100*time.Millisecond {
		t.Errorf("expected no delay for different chats, got %v", elapsed)
	}
}

// TestSend_C2CRateLimit tests C2C (private) rate limiting
func TestSend_C2CRateLimit(t *testing.T) {
	api := &mockQQAPI{}
	ch := newTestQQChannel(t, api)
	ch.chatType.Store("user-1", "direct")

	start := time.Now()

	// Send first message
	err := ch.Send(context.Background(), bus.OutboundMessage{
		ChatID:  "user-1",
		Content: "Message 1",
	})
	if err != nil {
		t.Errorf("first send failed: %v", err)
	}

	// Send second message immediately
	err = ch.Send(context.Background(), bus.OutboundMessage{
		ChatID:  "user-1",
		Content: "Message 2",
	})
	if err != nil {
		t.Errorf("second send failed: %v", err)
	}

	elapsed := time.Since(start)

	// C2C rate limit is 200ms, should have waited at least 150ms
	if elapsed < 150*time.Millisecond {
		t.Errorf("expected at least 150ms delay for C2C rate limiting, got %v", elapsed)
	}
}

// TestSend_GroupVsDirectRateLimit tests that group and direct rate limiters are independent
func TestSend_GroupVsDirectRateLimit(t *testing.T) {
	api := &mockQQAPI{}
	ch := newTestQQChannel(t, api)
	ch.chatType.Store("group-1", "group")
	ch.chatType.Store("user-1", "direct")

	start := time.Now()

	// Send to group
	err := ch.Send(context.Background(), bus.OutboundMessage{
		ChatID:  "group-1",
		Content: "Group message",
	})
	if err != nil {
		t.Errorf("group send failed: %v", err)
	}

	// Send to C2C immediately - should not wait (different rate limiters)
	err = ch.Send(context.Background(), bus.OutboundMessage{
		ChatID:  "user-1",
		Content: "Direct message",
	})
	if err != nil {
		t.Errorf("direct send failed: %v", err)
	}

	elapsed := time.Since(start)

	// Should not have waited much (different rate limiters: group=500ms, direct=200ms)
	// The C2C message should only wait 200ms, not 500ms
	if elapsed > 300*time.Millisecond {
		t.Errorf("expected less than 300ms delay (direct rate limit), got %v", elapsed)
	}
}

// TestIsRetryableError tests error classification
func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		err       string
		retryable bool
	}{
		{"timeout", true},
		{"context deadline exceeded", true},
		{"connection reset by peer", true},
		{"connection refused", true},
		{"temporary failure", true},
		{"rate limit: 429", true},
		{"server error: 500", true},
		{"bad gateway: 502", true},
		{"service unavailable: 503", true},
		{"unauthorized", false},
		{"invalid request", false},
		{"not found", false},
	}

	for _, tt := range tests {
		t.Run(tt.err, func(t *testing.T) {
			result := isRetryableError(errors.New(tt.err))
			if result != tt.retryable {
				t.Errorf("isRetryableError(%q) = %v, want %v", tt.err, result, tt.retryable)
			}
		})
	}
}

// TestCalculateBackoff tests exponential backoff calculation
func TestCalculateBackoff(t *testing.T) {
	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{0, 500 * time.Millisecond},
		{1, 1 * time.Second},
		{2, 2 * time.Second},
		{3, 4 * time.Second},
		{4, 8 * time.Second},
		{5, 10 * time.Second}, // Should cap at 10s
		{6, 10 * time.Second}, // Should stay at 10s
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			ch := newTestQQChannel(t, &mockQQAPI{})
			result := ch.calculateBackoff(tt.attempt)
			if result != tt.expected {
				t.Errorf("calculateBackoff(%d) = %v, want %v", tt.attempt, result, tt.expected)
			}
		})
	}
}

// TestQQChannel_ResetSessionState tests that session state is properly reset
func TestQQChannel_ResetSessionState(t *testing.T) {
	api := &mockQQAPI{}
	ch := newTestQQChannel(t, api)

	// Set some initial state
	ch.chatType.Store("group-1", "group")
	ch.lastMsgID.Store("group-1", "msg-old")
	ch.msgSeqCounters.Store("group-1", new(atomic.Uint64))

	// Pre-load rate limiter
	ch.groupRateLimiter.wait("group-1")
	ch.directRateLimiter.wait("user-1")

	// Reset session state
	ch.resetSessionState()

	// Verify chatType was cleared
	if _, ok := ch.chatType.Load("group-1"); ok {
		t.Error("expected chatType to be cleared")
	}

	// Verify lastMsgID was cleared
	if _, ok := ch.lastMsgID.Load("group-1"); ok {
		t.Error("expected lastMsgID to be cleared")
	}
}

// TestQQChannel_ResetSessionState_ReasoningChannel tests that reasoning channel is re-registered
func TestQQChannel_ResetSessionState_ReasoningChannel(t *testing.T) {
	ch := &QQChannel{
		BaseChannel:       channels.NewBaseChannel("qq", config.QQConfig{}, nil, config.FlexibleStringSlice{"*"}),
		config:            config.QQConfig{ReasoningChannelID: "reasoning-group"},
		dedup:             make(map[string]time.Time),
		groupRateLimiter:  newRateLimiter(500 * time.Millisecond),
		directRateLimiter: newRateLimiter(200 * time.Millisecond),
	}

	ch.resetSessionState()

	if kind, _ := ch.chatType.Load("reasoning-group"); kind != "group" {
		t.Error("expected reasoning channel to be re-registered as group")
	}
}

// TestQQChannel_ConfigDrivenParams tests that config values override defaults.
func TestQQChannel_ConfigDrivenParams(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*config.QQConfig)
		check func(*testing.T, *QQChannel)
	}{
		{
			name:  "reconnect initial uses config when set",
			setup: func(cfg *config.QQConfig) { cfg.ReconnectInitialMs = 10000 },
			check: func(t *testing.T, ch *QQChannel) {
				if got := ch.reconnectInitial(); got != 10*time.Second {
					t.Errorf("reconnectInitial() = %v, want 10s", got)
				}
			},
		},
		{
			name:  "reconnect initial falls back to default when zero",
			setup: func(cfg *config.QQConfig) { cfg.ReconnectInitialMs = 0 },
			check: func(t *testing.T, ch *QQChannel) {
				if got := ch.reconnectInitial(); got != 5*time.Second {
					t.Errorf("reconnectInitial() = %v, want 5s", got)
				}
			},
		},
		{
			name:  "reconnect max uses config when set",
			setup: func(cfg *config.QQConfig) { cfg.ReconnectMaxMs = 60000 },
			check: func(t *testing.T, ch *QQChannel) {
				if got := ch.reconnectMax(); got != 60*time.Second {
					t.Errorf("reconnectMax() = %v, want 60s", got)
				}
			},
		},
		{
			name:  "max retries uses config when set",
			setup: func(cfg *config.QQConfig) { cfg.MaxRetries = 5 },
			check: func(t *testing.T, ch *QQChannel) {
				if got := ch.maxRetries(); got != 5 {
					t.Errorf("maxRetries() = %v, want 5", got)
				}
			},
		},
		{
			name:  "max retries falls back to default when zero",
			setup: func(cfg *config.QQConfig) { cfg.MaxRetries = 0 },
			check: func(t *testing.T, ch *QQChannel) {
				if got := ch.maxRetries(); got != 3 {
					t.Errorf("maxRetries() = %v, want 3", got)
				}
			},
		},
		{
			name:  "retry max delay uses config when set",
			setup: func(cfg *config.QQConfig) { cfg.RetryMaxDelayMs = 30000 },
			check: func(t *testing.T, ch *QQChannel) {
				if got := ch.retryMaxDelay(); got != 30*time.Second {
					t.Errorf("retryMaxDelay() = %v, want 30s", got)
				}
			},
		},
		{
			name: "rate limiters use config values",
			setup: func(cfg *config.QQConfig) {
				cfg.RateLimitGroupMs = 1000
				cfg.RateLimitDirectMs = 500
			},
			check: func(t *testing.T, ch *QQChannel) {
				// Verify rate limiters are created with config values
				start := time.Now()
				ch.groupRateLimiter.wait("test-chat")
				ch.groupRateLimiter.wait("test-chat")
				elapsed := time.Since(start)
				if elapsed < 900*time.Millisecond {
					t.Errorf("group rate limiter did not delay ~1s, got %v", elapsed)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.QQConfig{AppID: "test", AppSecret: "test"}
			tt.setup(&cfg)
			// Rate limiter test needs NewQQChannel to initialize rate limiters
			if tt.name == "rate limiters use config values" {
				ch, err := NewQQChannel(cfg, nil)
				if err != nil {
					t.Fatalf("NewQQChannel failed: %v", err)
				}
				tt.check(t, ch)
				return
			}
			ch := &QQChannel{config: cfg}
			tt.check(t, ch)
		})
	}
}

// Verify mockQQAPI implements qqAPI interface at compile time
var _ qqAPI = (*mockQQAPI)(nil)

// Suppress unused import warning
var _ = oauth2.Token{}

package agent

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	runtimeevents "github.com/sipeed/picoclaw/pkg/events"
	"github.com/sipeed/picoclaw/pkg/providers"
)

// ---------------------------------------------------------------------------
// Factory registry tests
// ---------------------------------------------------------------------------

func TestRegisterContextManager_Success(t *testing.T) {
	cleanup := resetCMRegistry()
	defer cleanup()

	factory := func(cfg json.RawMessage, al *AgentLoop) (ContextManager, error) {
		return &noopContextManager{}, nil
	}
	if err := RegisterContextManager("test_cm", factory); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	f, ok := lookupContextManager("test_cm")
	if !ok {
		t.Fatal("expected factory to be registered")
	}
	if f == nil {
		t.Fatal("expected non-nil factory")
	}
}

func TestRegisterContextManager_EmptyName(t *testing.T) {
	cleanup := resetCMRegistry()
	defer cleanup()

	err := RegisterContextManager("", func(cfg json.RawMessage, al *AgentLoop) (ContextManager, error) {
		return &noopContextManager{}, nil
	})
	if err == nil {
		t.Fatal("expected error for empty name")
	}
	if !strings.Contains(err.Error(), "name is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRegisterContextManager_NilFactory(t *testing.T) {
	cleanup := resetCMRegistry()
	defer cleanup()

	err := RegisterContextManager("nil_factory", nil)
	if err == nil {
		t.Fatal("expected error for nil factory")
	}
	if !strings.Contains(err.Error(), "factory is nil") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRegisterContextManager_Duplicate(t *testing.T) {
	cleanup := resetCMRegistry()
	defer cleanup()

	factory := func(cfg json.RawMessage, al *AgentLoop) (ContextManager, error) {
		return &noopContextManager{}, nil
	}
	if err := RegisterContextManager("dup_cm", factory); err != nil {
		t.Fatalf("first registration failed: %v", err)
	}
	err := RegisterContextManager("dup_cm", factory)
	if err == nil {
		t.Fatal("expected error for duplicate registration")
	}
	if !strings.Contains(err.Error(), "already registered") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLookupContextManager_Unknown(t *testing.T) {
	cleanup := resetCMRegistry()
	defer cleanup()

	_, ok := lookupContextManager("nonexistent")
	if ok {
		t.Fatal("expected lookup to fail for unknown name")
	}
}

// ---------------------------------------------------------------------------
// resolveContextManager tests
// ---------------------------------------------------------------------------

func TestResolveContextManager_Default(t *testing.T) {
	cleanup := resetCMRegistry()
	defer cleanup()

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         t.TempDir(),
				ModelName:         "test-model",
				MaxTokens:         4096,
				MaxToolIterations: 10,
				ContextManager:    "", // default → legacy
			},
		},
	}
	al := newCMTestAgentLoop(cfg)

	cm := al.contextManager
	if cm == nil {
		t.Fatal("expected non-nil context manager")
	}
	if _, ok := cm.(*legacyContextManager); !ok {
		t.Fatalf("expected *legacyContextManager, got %T", cm)
	}
}

func TestResolveContextManager_ExplicitLegacy(t *testing.T) {
	cleanup := resetCMRegistry()
	defer cleanup()

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         t.TempDir(),
				ModelName:         "test-model",
				MaxTokens:         4096,
				MaxToolIterations: 10,
				ContextManager:    "legacy",
			},
		},
	}
	al := newCMTestAgentLoop(cfg)

	if _, ok := al.contextManager.(*legacyContextManager); !ok {
		t.Fatalf("expected *legacyContextManager, got %T", al.contextManager)
	}
}

func TestResolveContextManager_UnknownFallsBackToLegacy(t *testing.T) {
	cleanup := resetCMRegistry()
	defer cleanup()

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         t.TempDir(),
				ModelName:         "test-model",
				MaxTokens:         4096,
				MaxToolIterations: 10,
				ContextManager:    "unknown_cm",
			},
		},
	}
	al := newCMTestAgentLoop(cfg)

	if _, ok := al.contextManager.(*legacyContextManager); !ok {
		t.Fatalf("expected fallback to *legacyContextManager, got %T", al.contextManager)
	}
}

func TestResolveContextManager_RegisteredFactory(t *testing.T) {
	cleanup := resetCMRegistry()
	defer cleanup()

	factory := func(cfg json.RawMessage, al *AgentLoop) (ContextManager, error) {
		return &noopContextManager{}, nil
	}
	if err := RegisterContextManager("custom_cm", factory); err != nil {
		t.Fatalf("register failed: %v", err)
	}

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         t.TempDir(),
				ModelName:         "test-model",
				MaxTokens:         4096,
				MaxToolIterations: 10,
				ContextManager:    "custom_cm",
			},
		},
	}
	al := newCMTestAgentLoop(cfg)

	if _, ok := al.contextManager.(*noopContextManager); !ok {
		t.Fatalf("expected *noopContextManager, got %T", al.contextManager)
	}
}

func TestResolveContextManager_FactoryError(t *testing.T) {
	cleanup := resetCMRegistry()
	defer cleanup()

	factory := func(cfg json.RawMessage, al *AgentLoop) (ContextManager, error) {
		return nil, os.ErrPermission
	}
	if err := RegisterContextManager("broken_cm", factory); err != nil {
		t.Fatalf("register failed: %v", err)
	}

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         t.TempDir(),
				ModelName:         "test-model",
				MaxTokens:         4096,
				MaxToolIterations: 10,
				ContextManager:    "broken_cm",
			},
		},
	}
	al := newCMTestAgentLoop(cfg)

	// Should fall back to legacy when factory returns error
	if _, ok := al.contextManager.(*legacyContextManager); !ok {
		t.Fatalf("expected fallback to *legacyContextManager on factory error, got %T", al.contextManager)
	}
}

// ---------------------------------------------------------------------------
// Legacy Assemble tests
// ---------------------------------------------------------------------------

func TestLegacyAssemble_Passthrough(t *testing.T) {
	cfg := testConfig(t)
	al := newCMTestAgentLoop(cfg)

	agent := al.registry.GetDefaultAgent()
	if agent == nil {
		t.Fatal("expected default agent")
	}

	history := []providers.Message{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "hi there"},
	}
	agent.Sessions.SetHistory("test-session", history)

	resp, err := al.contextManager.Assemble(context.Background(), &AssembleRequest{
		SessionKey: "test-session",
		Budget:     8000,
		MaxTokens:  4096,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.History) != len(history) {
		t.Fatalf("expected %d messages, got %d", len(history), len(resp.History))
	}
	for i, msg := range resp.History {
		if msg.Content != history[i].Content || msg.Role != history[i].Role {
			t.Fatalf("message %d mismatch: want %+v, got %+v", i, history[i], msg)
		}
	}
}

func TestLegacyAssemble_EmptyHistory(t *testing.T) {
	cfg := testConfig(t)
	al := newCMTestAgentLoop(cfg)

	resp, err := al.contextManager.Assemble(context.Background(), &AssembleRequest{
		SessionKey: "test-session",
		Budget:     8000,
		MaxTokens:  4096,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.History) != 0 {
		t.Fatalf("expected empty messages, got %d", len(resp.History))
	}
}

// ---------------------------------------------------------------------------
// Legacy Compact overflow tests
// ---------------------------------------------------------------------------

func TestLegacyCompact_Overflow(t *testing.T) {
	cfg := testConfig(t)
	al := newCMTestAgentLoop(cfg)

	defaultAgent := al.registry.GetDefaultAgent()
	if defaultAgent == nil {
		t.Fatal("expected default agent")
	}

	history := []providers.Message{
		{Role: "user", Content: "msg 1"},
		{Role: "assistant", Content: "resp 1"},
		{Role: "user", Content: "msg 2"},
		{Role: "assistant", Content: "resp 2"},
		{Role: "user", Content: "msg 3"},
	}
	defaultAgent.Sessions.SetHistory("session-overflow", history)

	runtimeCh, closeRuntimeEvents := subscribeRuntimeEventsForTest(
		t,
		al,
		16,
		runtimeevents.KindAgentContextCompress,
	)
	defer closeRuntimeEvents()

	err := al.contextManager.Compact(context.Background(), &CompactRequest{
		SessionKey: "session-overflow",
		Reason:     ContextCompressReasonRetry,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// After overflow compression, history should be shorter
	newHistory := defaultAgent.Sessions.GetHistory("session-overflow")
	if len(newHistory) >= len(history) {
		t.Fatalf("expected compressed history, got %d messages (was %d)", len(newHistory), len(history))
	}

	// Summary should contain compression note
	summary := defaultAgent.Sessions.GetSummary("session-overflow")
	if !strings.Contains(summary, "Emergency compression") {
		t.Fatalf("expected compression note in summary, got %q", summary)
	}

	// Event should carry the proactive reason
	events := collectRuntimeEventStream(runtimeCh)
	compressEvt, ok := findRuntimeEvent(events, runtimeevents.KindAgentContextCompress)
	if !ok {
		t.Fatal("expected context compress event")
	}
	payload, ok := compressEvt.Payload.(ContextCompressPayload)
	if !ok {
		t.Fatalf("expected ContextCompressPayload, got %T", compressEvt.Payload)
	}
	if payload.Reason != ContextCompressReasonRetry {
		t.Fatalf("expected retry reason, got %q", payload.Reason)
	}
}

func TestLegacyCompact_Overflow_ProactiveReason(t *testing.T) {
	cfg := testConfig(t)
	al := newCMTestAgentLoop(cfg)

	defaultAgent := al.registry.GetDefaultAgent()
	if defaultAgent == nil {
		t.Fatal("expected default agent")
	}

	history := []providers.Message{
		{Role: "user", Content: "msg 1"},
		{Role: "assistant", Content: "resp 1"},
		{Role: "user", Content: "msg 2"},
		{Role: "assistant", Content: "resp 2"},
		{Role: "user", Content: "msg 3"},
	}
	defaultAgent.Sessions.SetHistory("session-proactive", history)

	runtimeCh, closeRuntimeEvents := subscribeRuntimeEventsForTest(
		t,
		al,
		16,
		runtimeevents.KindAgentContextCompress,
	)
	defer closeRuntimeEvents()

	err := al.contextManager.Compact(context.Background(), &CompactRequest{
		SessionKey: "session-proactive",
		Reason:     ContextCompressReasonProactive,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	events := collectRuntimeEventStream(runtimeCh)
	compressEvt, ok := findRuntimeEvent(events, runtimeevents.KindAgentContextCompress)
	if !ok {
		t.Fatal("expected context compress event")
	}
	payload, ok := compressEvt.Payload.(ContextCompressPayload)
	if !ok {
		t.Fatalf("expected ContextCompressPayload, got %T", compressEvt.Payload)
	}
	if payload.Reason != ContextCompressReasonProactive {
		t.Fatalf("expected proactive reason, got %q", payload.Reason)
	}
}

func TestLegacyCompact_Overflow_TooShortToCompress(t *testing.T) {
	cfg := testConfig(t)
	al := newCMTestAgentLoop(cfg)

	defaultAgent := al.registry.GetDefaultAgent()
	if defaultAgent == nil {
		t.Fatal("expected default agent")
	}

	history := []providers.Message{
		{Role: "user", Content: "only one"},
	}
	defaultAgent.Sessions.SetHistory("session-tiny", history)

	err := al.contextManager.Compact(context.Background(), &CompactRequest{
		SessionKey: "session-tiny",
		Reason:     ContextCompressReasonRetry,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// History should be unchanged (too short to compress)
	newHistory := defaultAgent.Sessions.GetHistory("session-tiny")
	if len(newHistory) != len(history) {
		t.Fatalf("expected history unchanged, got %d messages (was %d)", len(newHistory), len(history))
	}
}

// ---------------------------------------------------------------------------
// Legacy Compact post-turn tests
// ---------------------------------------------------------------------------

func TestLegacyCompact_PostTurn_BelowThreshold(t *testing.T) {
	cfg := testConfig(t)
	al := newCMTestAgentLoop(cfg)

	defaultAgent := al.registry.GetDefaultAgent()
	if defaultAgent == nil {
		t.Fatal("expected default agent")
	}

	// Small history, below summarization thresholds
	history := []providers.Message{
		{Role: "user", Content: "hi"},
		{Role: "assistant", Content: "hello"},
	}
	defaultAgent.Sessions.SetHistory("session-small", history)

	err := al.contextManager.Compact(context.Background(), &CompactRequest{
		SessionKey: "session-small",
		Reason:     ContextCompressReasonSummarize,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// History should remain unchanged
	newHistory := defaultAgent.Sessions.GetHistory("session-small")
	if len(newHistory) != len(history) {
		t.Fatalf("expected unchanged history, got %d messages (was %d)", len(newHistory), len(history))
	}
}

func TestLegacyCompact_PostTurn_ExceedsMessageThreshold(t *testing.T) {
	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:                 t.TempDir(),
				ModelName:                 "test-model",
				MaxTokens:                 4096,
				MaxToolIterations:         10,
				ContextWindow:             8000,
				SummarizeMessageThreshold: 2,
				SummarizeTokenPercent:     75,
			},
		},
	}
	msgBus := bus.NewMessageBus()
	al := NewAgentLoop(cfg, msgBus, &simpleMockProvider{response: "summary"})

	defaultAgent := al.registry.GetDefaultAgent()
	if defaultAgent == nil {
		t.Fatal("expected default agent")
	}

	// 6 messages > threshold of 2
	history := []providers.Message{
		{Role: "user", Content: "q1"},
		{Role: "assistant", Content: "a1"},
		{Role: "user", Content: "q2"},
		{Role: "assistant", Content: "a2"},
		{Role: "user", Content: "q3"},
		{Role: "assistant", Content: "a3"},
	}
	defaultAgent.Sessions.SetHistory("session-threshold", history)

	runtimeCh, closeRuntimeEvents := subscribeRuntimeEventsForTest(
		t,
		al,
		16,
		runtimeevents.KindAgentSessionSummarize,
	)
	defer closeRuntimeEvents()

	err := al.contextManager.Compact(context.Background(), &CompactRequest{
		SessionKey: "session-threshold",
		Reason:     ContextCompressReasonSummarize,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	waitForRuntimeEvent(t, runtimeCh, 5*time.Second, func(evt runtimeevents.Event) bool {
		return evt.Kind == runtimeevents.KindAgentSessionSummarize
	})

	newHistory := defaultAgent.Sessions.GetHistory("session-threshold")
	if len(newHistory) >= len(history) {
		t.Fatalf("expected summarization to reduce history from %d messages, got %d", len(history), len(newHistory))
	}
}

// ---------------------------------------------------------------------------
// Legacy Ingest tests
// ---------------------------------------------------------------------------

func TestLegacyIngest_NoOp(t *testing.T) {
	cfg := testConfig(t)
	al := newCMTestAgentLoop(cfg)

	err := al.contextManager.Ingest(context.Background(), &IngestRequest{
		SessionKey: "session-ingest",
		Message:    providers.Message{Role: "user", Content: "test"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Mock ContextManager — verifies dispatch through AgentLoop
// ---------------------------------------------------------------------------

func TestAgentLoop_UsesCustomContextManager(t *testing.T) {
	cleanup := resetCMRegistry()
	defer cleanup()

	mock := &trackingContextManager{}
	factory := func(cfg json.RawMessage, al *AgentLoop) (ContextManager, error) {
		return mock, nil
	}
	if err := RegisterContextManager("tracking_cm", factory); err != nil {
		t.Fatalf("register failed: %v", err)
	}

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         t.TempDir(),
				ModelName:         "test-model",
				MaxTokens:         4096,
				MaxToolIterations: 10,
				ContextManager:    "tracking_cm",
			},
		},
	}
	al := newCMTestAgentLoop(cfg)

	// Verify the mock was installed
	if al.contextManager != mock {
		t.Fatalf("expected mock context manager, got %T", al.contextManager)
	}

	// Direct method calls
	_, err := mock.Assemble(context.Background(), &AssembleRequest{
		SessionKey: "s1",
		Budget:     8000,
		MaxTokens:  4096,
	})
	if err != nil {
		t.Fatalf("Assemble error: %v", err)
	}
	if mock.assembleCalls.Load() != 1 {
		t.Fatalf("expected 1 assemble call, got %d", mock.assembleCalls.Load())
	}

	err = mock.Compact(context.Background(), &CompactRequest{
		SessionKey: "s1",
		Reason:     ContextCompressReasonRetry,
	})
	if err != nil {
		t.Fatalf("Compact error: %v", err)
	}
	if mock.compactCalls.Load() != 1 {
		t.Fatalf("expected 1 compact call, got %d", mock.compactCalls.Load())
	}

	err = mock.Ingest(context.Background(), &IngestRequest{
		SessionKey: "s1",
		Message:    providers.Message{Role: "user", Content: "test"},
	})
	if err != nil {
		t.Fatalf("Ingest error: %v", err)
	}
	if mock.ingestCalls.Load() != 1 {
		t.Fatalf("expected 1 ingest call, got %d", mock.ingestCalls.Load())
	}
}

func TestIngestCalledDuringTurn(t *testing.T) {
	cleanup := resetCMRegistry()
	defer cleanup()

	mock := &trackingContextManager{}
	factory := func(cfg json.RawMessage, al *AgentLoop) (ContextManager, error) {
		return mock, nil
	}
	if err := RegisterContextManager("ingest_track_cm", factory); err != nil {
		t.Fatalf("register failed: %v", err)
	}

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         t.TempDir(),
				ModelName:         "test-model",
				MaxTokens:         4096,
				MaxToolIterations: 10,
				ContextManager:    "ingest_track_cm",
			},
		},
	}

	msgBus := bus.NewMessageBus()
	al := NewAgentLoop(cfg, msgBus, &simpleMockProvider{response: "done"})
	defaultAgent := al.registry.GetDefaultAgent()
	if defaultAgent == nil {
		t.Fatal("expected default agent")
	}

	// Run a turn — ingestMessage is called for user message and final assistant message
	_, err := al.runAgentLoop(context.Background(), defaultAgent, processOptions{
		SessionKey:      "session-ingest-turn",
		Channel:         "cli",
		ChatID:          "direct",
		UserMessage:     "test ingest",
		DefaultResponse: defaultResponse,
		EnableSummary:   false,
		SendResponse:    false,
	})
	if err != nil {
		t.Fatalf("runAgentLoop failed: %v", err)
	}

	// Should have at least 2 ingest calls: user message + final assistant message
	if mock.ingestCalls.Load() < 2 {
		t.Fatalf("expected >= 2 ingest calls during turn, got %d", mock.ingestCalls.Load())
	}
}

// ---------------------------------------------------------------------------
// forceCompression edge cases (via legacy Compact)
// ---------------------------------------------------------------------------

func TestLegacyCompact_Overflow_SingleTurnKeepsLastUserMessage(t *testing.T) {
	cfg := testConfig(t)
	al := newCMTestAgentLoop(cfg)

	defaultAgent := al.registry.GetDefaultAgent()
	if defaultAgent == nil {
		t.Fatal("expected default agent")
	}

	// History with only 2 messages — forceCompression should still handle it
	history := []providers.Message{
		{Role: "user", Content: "first question"},
		{Role: "assistant", Content: "first answer"},
	}
	defaultAgent.Sessions.SetHistory("session-2msg", history)

	err := al.contextManager.Compact(context.Background(), &CompactRequest{
		SessionKey: "session-2msg",
		Reason:     ContextCompressReasonRetry,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	newHistory := defaultAgent.Sessions.GetHistory("session-2msg")
	// With 2 messages, forceCompression returns false (len <= 2), so no compression
	if len(newHistory) != len(history) {
		t.Fatalf("expected no compression for 2-message history, got %d", len(newHistory))
	}
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// noopContextManager is a minimal ContextManager that does nothing.
type noopContextManager struct{}

func (m *noopContextManager) Assemble(_ context.Context, req *AssembleRequest) (*AssembleResponse, error) {
	return &AssembleResponse{}, nil
}
func (m *noopContextManager) Compact(_ context.Context, _ *CompactRequest) error { return nil }
func (m *noopContextManager) Ingest(_ context.Context, _ *IngestRequest) error   { return nil }
func (m *noopContextManager) Clear(_ context.Context, _ string) error            { return nil }

// trackingContextManager tracks call counts for each method.
type trackingContextManager struct {
	assembleCalls atomic.Int64
	compactCalls  atomic.Int64
	ingestCalls   atomic.Int64
	mu            sync.Mutex
	lastAssemble  *AssembleRequest
	lastCompact   *CompactRequest
	lastIngest    *IngestRequest
}

func (m *trackingContextManager) Assemble(_ context.Context, req *AssembleRequest) (*AssembleResponse, error) {
	m.assembleCalls.Add(1)
	m.mu.Lock()
	m.lastAssemble = req
	m.mu.Unlock()
	return &AssembleResponse{}, nil
}

func (m *trackingContextManager) Compact(_ context.Context, req *CompactRequest) error {
	m.compactCalls.Add(1)
	m.mu.Lock()
	m.lastCompact = req
	m.mu.Unlock()
	return nil
}

func (m *trackingContextManager) Ingest(_ context.Context, req *IngestRequest) error {
	m.ingestCalls.Add(1)
	m.mu.Lock()
	m.lastIngest = req
	m.mu.Unlock()
	return nil
}

func (m *trackingContextManager) Clear(_ context.Context, _ string) error { return nil }

// resetCMRegistry clears the global factory registry and returns a cleanup
// function that restores the original state after the test.
func resetCMRegistry() func() {
	cmRegistryMu.Lock()
	original := make(map[string]ContextManagerFactory, len(cmRegistry))
	for k, v := range cmRegistry {
		original[k] = v
	}
	cmRegistry = make(map[string]ContextManagerFactory)
	cmRegistryMu.Unlock()

	return func() {
		cmRegistryMu.Lock()
		cmRegistry = original
		cmRegistryMu.Unlock()
	}
}

func testConfig(t *testing.T) *config.Config {
	t.Helper()
	return &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         t.TempDir(),
				ModelName:         "test-model",
				MaxTokens:         4096,
				MaxToolIterations: 10,
			},
		},
	}
}

func newCMTestAgentLoop(cfg *config.Config) *AgentLoop {
	msgBus := bus.NewMessageBus()
	return NewAgentLoop(cfg, msgBus, &simpleMockProvider{response: "test"})
}

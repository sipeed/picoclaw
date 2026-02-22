package miniapp

import (
	"bufio"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"runtime"
	"sort"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/skills"
	"github.com/sipeed/picoclaw/pkg/stats"
)

// buildInitData constructs a valid initData string from params and a bot token.
func buildInitData(params map[string]string, botToken string) string {
	// Build data-check-string
	var pairs []string
	for k, v := range params {
		pairs = append(pairs, fmt.Sprintf("%s=%s", k, v))
	}
	sort.Strings(pairs)
	dataCheckString := strings.Join(pairs, "\n")

	// Compute secret key
	secretKeyMac := hmac.New(sha256.New, []byte("WebAppData"))
	secretKeyMac.Write([]byte(botToken))
	secretKey := secretKeyMac.Sum(nil)

	// Compute hash
	hashMac := hmac.New(sha256.New, secretKey)
	hashMac.Write([]byte(dataCheckString))
	hash := hex.EncodeToString(hashMac.Sum(nil))

	// Build query string
	values := url.Values{}
	for k, v := range params {
		values.Set(k, v)
	}
	values.Set("hash", hash)
	return values.Encode()
}

func TestValidateInitData(t *testing.T) {
	botToken := "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11"

	t.Run("valid initData", func(t *testing.T) {
		params := map[string]string{
			"query_id": "AAHdF6IQAAAAAN0XohDhrOrc",
			"user":     `{"id":279058397,"first_name":"Vlad"}`,
			"auth_date": "1234567890",
		}
		initData := buildInitData(params, botToken)
		if !ValidateInitData(initData, botToken) {
			t.Error("ValidateInitData() returned false for valid data")
		}
	})

	t.Run("tampered data", func(t *testing.T) {
		params := map[string]string{
			"query_id": "AAHdF6IQAAAAAN0XohDhrOrc",
			"user":     `{"id":279058397,"first_name":"Vlad"}`,
			"auth_date": "1234567890",
		}
		initData := buildInitData(params, botToken)
		// Tamper with the data
		initData = strings.Replace(initData, "Vlad", "Evil", 1)
		if ValidateInitData(initData, botToken) {
			t.Error("ValidateInitData() returned true for tampered data")
		}
	})

	t.Run("wrong bot token", func(t *testing.T) {
		params := map[string]string{
			"auth_date": "1234567890",
		}
		initData := buildInitData(params, botToken)
		if ValidateInitData(initData, "wrong-token") {
			t.Error("ValidateInitData() returned true for wrong bot token")
		}
	})

	t.Run("missing hash", func(t *testing.T) {
		if ValidateInitData("auth_date=1234567890", botToken) {
			t.Error("ValidateInitData() returned true for missing hash")
		}
	})

	t.Run("empty initData", func(t *testing.T) {
		if ValidateInitData("", botToken) {
			t.Error("ValidateInitData() returned true for empty initData")
		}
	})

	t.Run("invalid query string", func(t *testing.T) {
		if ValidateInitData("%%%invalid", botToken) {
			t.Error("ValidateInitData() returned true for invalid query string")
		}
	})
}

// ── StateNotifier tests ──

func TestStateNotifier_FanOut(t *testing.T) {
	n := NewStateNotifier()
	ch1 := n.Subscribe()
	ch2 := n.Subscribe()
	defer n.Unsubscribe(ch1)
	defer n.Unsubscribe(ch2)

	n.Notify()

	select {
	case <-ch1:
	case <-time.After(100 * time.Millisecond):
		t.Error("ch1 did not receive notification")
	}
	select {
	case <-ch2:
	case <-time.After(100 * time.Millisecond):
		t.Error("ch2 did not receive notification")
	}
}

func TestStateNotifier_Coalesce(t *testing.T) {
	n := NewStateNotifier()
	ch := n.Subscribe()
	defer n.Unsubscribe(ch)

	// Multiple rapid notifications should coalesce into one
	n.Notify()
	n.Notify()
	n.Notify()

	select {
	case <-ch:
	case <-time.After(100 * time.Millisecond):
		t.Error("ch did not receive notification")
	}

	// Channel should be empty now (coalesced)
	select {
	case <-ch:
		t.Error("expected no second notification (should coalesce)")
	case <-time.After(50 * time.Millisecond):
	}
}

// ── SSE endpoint tests ──

type mockDataProvider struct{}

func (m *mockDataProvider) ListSkills() []skills.SkillInfo {
	return []skills.SkillInfo{{Name: "test-skill", Description: "A test", Source: "local"}}
}
func (m *mockDataProvider) GetPlanInfo() PlanInfo {
	return PlanInfo{HasPlan: false, Status: "none"}
}
func (m *mockDataProvider) GetSessionStats() *stats.Stats {
	return nil
}
func (m *mockDataProvider) GetActiveSessions() []SessionInfo {
	return []SessionInfo{}
}
func (m *mockDataProvider) GetGitInfo() GitInfo {
	return GitInfo{}
}

type mockSender struct{}

func (m *mockSender) SendCommand(senderID, chatID, command string) {}

const testBotToken = "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11"

func testInitData() string {
	return buildInitData(map[string]string{
		"user":      `{"id":279058397,"first_name":"Test"}`,
		"auth_date": "1234567890",
	}, testBotToken)
}

func TestSSE_AuthRequired(t *testing.T) {
	notifier := NewStateNotifier()
	h := NewHandler(&mockDataProvider{}, &mockSender{}, testBotToken, notifier)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest("GET", "/miniapp/api/events", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestSSE_Headers(t *testing.T) {
	notifier := NewStateNotifier()
	h := NewHandler(&mockDataProvider{}, &mockSender{}, testBotToken, notifier)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	ts := httptest.NewServer(mux)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/miniapp/api/events?initData=" + url.QueryEscape(testInitData()))
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer resp.Body.Close()

	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("expected Content-Type text/event-stream, got %q", ct)
	}
	if cc := resp.Header.Get("Cache-Control"); cc != "no-cache" {
		t.Errorf("expected Cache-Control no-cache, got %q", cc)
	}
}

func TestSSE_InitialEvents(t *testing.T) {
	notifier := NewStateNotifier()
	h := NewHandler(&mockDataProvider{}, &mockSender{}, testBotToken, notifier)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	ts := httptest.NewServer(mux)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/miniapp/api/events?initData=" + url.QueryEscape(testInitData()))
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	events := make(map[string]bool)
	deadline := time.After(2 * time.Second)

	for len(events) < 3 {
		done := make(chan bool, 1)
		go func() {
			done <- scanner.Scan()
		}()
		select {
		case ok := <-done:
			if !ok {
				t.Fatalf("scanner ended early: %v", scanner.Err())
			}
		case <-deadline:
			t.Fatalf("timed out waiting for events, got: %v", events)
		}
		line := scanner.Text()
		if strings.HasPrefix(line, "event: ") {
			events[strings.TrimPrefix(line, "event: ")] = true
		}
	}

	for _, name := range []string{"plan", "session", "skills"} {
		if !events[name] {
			t.Errorf("missing initial event %q", name)
		}
	}
}

func TestStateNotifier_UnsubscribeStopsDelivery(t *testing.T) {
	n := NewStateNotifier()
	ch := n.Subscribe()

	n.Unsubscribe(ch)
	n.Notify()

	select {
	case <-ch:
		t.Error("received notification after Unsubscribe")
	case <-time.After(50 * time.Millisecond):
	}
}

func TestStateNotifier_SubscribeCycleNoLeak(t *testing.T) {
	n := NewStateNotifier()

	for i := 0; i < 100; i++ {
		ch := n.Subscribe()
		n.Unsubscribe(ch)
	}

	n.mu.Lock()
	count := len(n.subs)
	n.mu.Unlock()

	if count != 0 {
		t.Errorf("expected 0 subscribers after cycle, got %d", count)
	}
}

func TestSSE_ClientDisconnectCleansUp(t *testing.T) {
	notifier := NewStateNotifier()
	h := NewHandler(&mockDataProvider{}, &mockSender{}, testBotToken, notifier)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	ts := httptest.NewServer(mux)
	defer ts.Close()

	// Baseline goroutine count
	runtime.GC()
	time.Sleep(50 * time.Millisecond)
	baseGoroutines := runtime.NumGoroutine()

	// Open and close several SSE connections
	for i := 0; i < 5; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		req, _ := http.NewRequestWithContext(ctx, "GET",
			ts.URL+"/miniapp/api/events?initData="+url.QueryEscape(testInitData()), nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("request %d failed: %v", i, err)
		}
		// Read at least one event line to confirm the handler is running
		scanner := bufio.NewScanner(resp.Body)
		scanner.Scan()
		// Disconnect
		cancel()
		resp.Body.Close()
	}

	// Wait for goroutines to wind down
	time.Sleep(200 * time.Millisecond)
	runtime.GC()
	time.Sleep(50 * time.Millisecond)

	finalGoroutines := runtime.NumGoroutine()
	leaked := finalGoroutines - baseGoroutines
	if leaked > 2 { // small tolerance for runtime jitter
		t.Errorf("possible goroutine leak: baseline=%d, final=%d, leaked=%d",
			baseGoroutines, finalGoroutines, leaked)
	}

	// Verify all subscribers were cleaned up
	notifier.mu.Lock()
	subCount := len(notifier.subs)
	notifier.mu.Unlock()
	if subCount != 0 {
		t.Errorf("expected 0 subscribers after disconnect, got %d", subCount)
	}
}

func TestSSE_NotifyDrivesSubsequentEvents(t *testing.T) {
	notifier := NewStateNotifier()
	provider := &mutatingDataProvider{}
	h := NewHandler(provider, &mockSender{}, testBotToken, notifier)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	ts := httptest.NewServer(mux)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/miniapp/api/events?initData=" + url.QueryEscape(testInitData()))
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	// Drain initial 3 events
	drainEvents(t, scanner, 3, 2*time.Second)

	// Mutate state and notify — diff dedup should detect the change and send a new event
	provider.mutated.Store(true)
	notifier.Notify()

	events := drainEvents(t, scanner, 1, 2*time.Second)
	if !events["plan"] {
		t.Errorf("expected plan event after mutation, got: %v", events)
	}
}

func TestSSE_DiffDedupSuppressesDuplicate(t *testing.T) {
	notifier := NewStateNotifier()
	h := NewHandler(&mockDataProvider{}, &mockSender{}, testBotToken, notifier)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	ts := httptest.NewServer(mux)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/miniapp/api/events?initData=" + url.QueryEscape(testInitData()))
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	// Drain initial events
	drainEvents(t, scanner, 3, 2*time.Second)

	// Notify with unchanged data — should produce zero new event lines
	notifier.Notify()

	// Give the handler time to process
	time.Sleep(100 * time.Millisecond)

	// Try to read — nothing should arrive since data hasn't changed
	gotExtra := make(chan string, 1)
	go func() {
		if scanner.Scan() {
			gotExtra <- scanner.Text()
		}
	}()

	select {
	case line := <-gotExtra:
		// Only fail if it's an actual event (not an empty keepalive)
		if strings.HasPrefix(line, "event:") {
			t.Errorf("received duplicate event despite unchanged data: %s", line)
		}
	case <-time.After(200 * time.Millisecond):
		// Expected: no duplicate sent
	}
}

// ── helpers ──

// mutatingDataProvider returns different PlanInfo after mutate is set.
type mutatingDataProvider struct {
	mutated atomic.Bool
}

func (m *mutatingDataProvider) ListSkills() []skills.SkillInfo {
	return []skills.SkillInfo{{Name: "test-skill", Description: "A test", Source: "local"}}
}
func (m *mutatingDataProvider) GetPlanInfo() PlanInfo {
	if m.mutated.Load() {
		return PlanInfo{HasPlan: true, Status: "executing", CurrentPhase: 1, TotalPhases: 2}
	}
	return PlanInfo{HasPlan: false, Status: "none"}
}
func (m *mutatingDataProvider) GetSessionStats() *stats.Stats { return nil }
func (m *mutatingDataProvider) GetActiveSessions() []SessionInfo {
	return []SessionInfo{}
}
func (m *mutatingDataProvider) GetGitInfo() GitInfo {
	return GitInfo{}
}

// drainEvents reads SSE event lines until it collects `want` distinct event names or times out.
func drainEvents(t *testing.T, scanner *bufio.Scanner, want int, timeout time.Duration) map[string]bool {
	t.Helper()
	events := make(map[string]bool)
	deadline := time.After(timeout)
	for len(events) < want {
		done := make(chan bool, 1)
		go func() { done <- scanner.Scan() }()
		select {
		case ok := <-done:
			if !ok {
				return events
			}
		case <-deadline:
			return events
		}
		line := scanner.Text()
		if strings.HasPrefix(line, "event: ") {
			events[strings.TrimPrefix(line, "event: ")] = true
		}
	}
	return events
}

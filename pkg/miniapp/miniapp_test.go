package miniapp

import (
	"bufio"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	gitpkg "github.com/sipeed/picoclaw/pkg/git"
	"github.com/sipeed/picoclaw/pkg/skills"
	"github.com/sipeed/picoclaw/pkg/stats"
)

// buildInitData constructs a valid initData string from params and a bot token.
func buildInitData(params map[string]string, botToken string) string {
	// Build data-check-string
	pairs := make([]string, 0, len(params))
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
			"query_id":  "AAHdF6IQAAAAAN0XohDhrOrc",
			"user":      `{"id":279058397,"first_name":"Vlad"}`,
			"auth_date": freshAuthDate(),
		}
		initData := buildInitData(params, botToken)
		if !ValidateInitData(initData, botToken) {
			t.Error("ValidateInitData() returned false for valid data")
		}
	})

	t.Run("tampered data", func(t *testing.T) {
		params := map[string]string{
			"query_id":  "AAHdF6IQAAAAAN0XohDhrOrc",
			"user":      `{"id":279058397,"first_name":"Vlad"}`,
			"auth_date": freshAuthDate(),
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
			"auth_date": freshAuthDate(),
		}
		initData := buildInitData(params, botToken)
		if ValidateInitData(initData, "wrong-token") {
			t.Error("ValidateInitData() returned true for wrong bot token")
		}
	})

	t.Run("expired auth_date", func(t *testing.T) {
		params := map[string]string{
			"auth_date": "1234567890",
		}
		initData := buildInitData(params, botToken)
		if ValidateInitData(initData, botToken) {
			t.Error("ValidateInitData() returned true for expired auth_date")
		}
	})

	t.Run("missing hash", func(t *testing.T) {
		now := freshAuthDate()
		if ValidateInitData("auth_date="+now, botToken) {
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

func (m *mockDataProvider) GetSessionGraph() *SessionGraphData { return nil }

func (m *mockDataProvider) GetGitRepos() []GitRepoSummary {
	return nil
}

func (m *mockDataProvider) GetGitRepoDetail(name string) GitInfo {
	return GitInfo{Name: name}
}

func (m *mockDataProvider) GetContextInfo() ContextInfo {
	return ContextInfo{Workspace: "/mock/workspace"}
}

func (m *mockDataProvider) GetSystemPrompt() string {
	return "mock system prompt"
}

type mockSender struct{}

func (m *mockSender) SendCommand(senderID, chatID, command string) {}

const testBotToken = "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11"

func freshAuthDate() string {
	return strconv.FormatInt(time.Now().Unix(), 10)
}

func testInitData() string {
	return buildInitData(map[string]string{
		"user":      `{"id":279058397,"first_name":"Test"}`,
		"auth_date": freshAuthDate(),
	}, testBotToken)
}

func TestSSE_AuthRequired(t *testing.T) {
	notifier := NewStateNotifier()
	h := NewHandler(&mockDataProvider{}, &mockSender{}, testBotToken, notifier, nil, "")
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
	h := NewHandler(&mockDataProvider{}, &mockSender{}, testBotToken, notifier, nil, "")
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
	h := NewHandler(&mockDataProvider{}, &mockSender{}, testBotToken, notifier, nil, "")
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

	for len(events) < 4 {
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

	for _, name := range []string{"plan", "session", "skills", "dev"} {
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
	h := NewHandler(&mockDataProvider{}, &mockSender{}, testBotToken, notifier, nil, "")
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
	h := NewHandler(provider, &mockSender{}, testBotToken, notifier, nil, "")
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
	// Drain initial 6 events (plan, session, skills, dev, context, prompt)
	drainEvents(t, scanner, 6, 2*time.Second)

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
	h := NewHandler(&mockDataProvider{}, &mockSender{}, testBotToken, notifier, nil, "")
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
	// Drain initial events (plan, session, skills, dev, context, prompt)
	drainEvents(t, scanner, 6, 2*time.Second)

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

func (m *mutatingDataProvider) GetSessionGraph() *SessionGraphData { return nil }

func (m *mutatingDataProvider) GetGitRepos() []GitRepoSummary {
	return nil
}

func (m *mutatingDataProvider) GetGitRepoDetail(name string) GitInfo {
	return GitInfo{Name: name}
}

func (m *mutatingDataProvider) GetContextInfo() ContextInfo {
	return ContextInfo{Workspace: "/mock/workspace"}
}

func (m *mutatingDataProvider) GetSystemPrompt() string {
	return "mock system prompt"
}

// ── Dev proxy tests ──

func TestDevProxy_RegisterAndActivate(t *testing.T) {
	h := NewHandler(&mockDataProvider{}, &mockSender{}, testBotToken, NewStateNotifier(), nil, "")

	// Initially empty
	if got := h.GetDevTarget(); got != "" {
		t.Errorf("expected empty target, got %q", got)
	}

	// Register a target
	id, err := h.RegisterDevTarget("frontend", "http://localhost:3000")
	if err != nil {
		t.Fatalf("RegisterDevTarget failed: %v", err)
	}
	if id == "" {
		t.Fatal("expected non-empty id")
	}

	// Still inactive until activated
	if got := h.GetDevTarget(); got != "" {
		t.Errorf("expected empty target before activation, got %q", got)
	}

	// Activate
	if err := h.ActivateDevTarget(id); err != nil {
		t.Fatalf("ActivateDevTarget failed: %v", err)
	}
	if got := h.GetDevTarget(); got == "" {
		t.Error("expected non-empty target after activation")
	}

	// Deactivate
	if err := h.DeactivateDevTarget(); err != nil {
		t.Fatalf("DeactivateDevTarget failed: %v", err)
	}
	if got := h.GetDevTarget(); got != "" {
		t.Errorf("expected empty target after deactivation, got %q", got)
	}
}

func TestDevProxy_UnregisterActive(t *testing.T) {
	h := NewHandler(&mockDataProvider{}, &mockSender{}, testBotToken, NewStateNotifier(), nil, "")

	id, _ := h.RegisterDevTarget("api", "http://localhost:8080")
	h.ActivateDevTarget(id)

	if err := h.UnregisterDevTarget(id); err != nil {
		t.Fatalf("UnregisterDevTarget failed: %v", err)
	}
	if got := h.GetDevTarget(); got != "" {
		t.Errorf("expected empty target after unregister of active, got %q", got)
	}
	if targets := h.ListDevTargets(); len(targets) != 0 {
		t.Errorf("expected 0 targets, got %d", len(targets))
	}
}

func TestDevProxy_UnregisterNotFound(t *testing.T) {
	h := NewHandler(&mockDataProvider{}, &mockSender{}, testBotToken, NewStateNotifier(), nil, "")
	if err := h.UnregisterDevTarget("999"); err == nil {
		t.Error("expected error for non-existent target")
	}
}

func TestDevProxy_ActivateNotFound(t *testing.T) {
	h := NewHandler(&mockDataProvider{}, &mockSender{}, testBotToken, NewStateNotifier(), nil, "")
	if err := h.ActivateDevTarget("999"); err == nil {
		t.Error("expected error for non-existent target")
	}
}

func TestDevProxy_LocalhostOnly(t *testing.T) {
	h := NewHandler(&mockDataProvider{}, &mockSender{}, testBotToken, NewStateNotifier(), nil, "")

	// External host should be rejected at registration
	if _, err := h.RegisterDevTarget("ext", "http://example.com:3000"); err == nil {
		t.Error("expected error for external host, got nil")
	}

	// 127.0.0.1 should be allowed
	if _, err := h.RegisterDevTarget("local4", "http://127.0.0.1:8080"); err != nil {
		t.Errorf("expected 127.0.0.1 to be allowed, got %v", err)
	}

	// ::1 should be allowed
	if _, err := h.RegisterDevTarget("local6", "http://[::1]:9000"); err != nil {
		t.Errorf("expected [::1] to be allowed, got %v", err)
	}
}

func TestDevProxy_IPv4Rewrite(t *testing.T) {
	h := NewHandler(&mockDataProvider{}, &mockSender{}, testBotToken, NewStateNotifier(), nil, "")

	id, _ := h.RegisterDevTarget("front", "http://localhost:3000")
	h.ActivateDevTarget(id)

	// After activation, the internal target should use 127.0.0.1 instead of localhost
	got := h.GetDevTarget()
	if strings.Contains(got, "localhost") {
		t.Errorf("expected localhost to be rewritten to 127.0.0.1, got %q", got)
	}
	if !strings.Contains(got, "127.0.0.1") {
		t.Errorf("expected 127.0.0.1 in target, got %q", got)
	}
}

func TestDevProxy_ListDevTargets(t *testing.T) {
	h := NewHandler(&mockDataProvider{}, &mockSender{}, testBotToken, NewStateNotifier(), nil, "")

	h.RegisterDevTarget("api", "http://localhost:8080")
	h.RegisterDevTarget("frontend", "http://localhost:3000")

	targets := h.ListDevTargets()
	if len(targets) != 2 {
		t.Fatalf("expected 2 targets, got %d", len(targets))
	}
	// Should be sorted by ID
	if targets[0].Name != "api" {
		t.Errorf("expected first target name=api, got %q", targets[0].Name)
	}
	if targets[1].Name != "frontend" {
		t.Errorf("expected second target name=frontend, got %q", targets[1].Name)
	}
}

func TestDevProxy_ReverseProxy(t *testing.T) {
	// Create a backend server
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, "path=%s", r.URL.Path)
	}))
	defer backend.Close()

	h := NewHandler(&mockDataProvider{}, &mockSender{}, testBotToken, NewStateNotifier(), nil, "")
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	// Register and activate the test backend
	id, err := h.RegisterDevTarget("backend", backend.URL)
	if err != nil {
		t.Fatalf("RegisterDevTarget failed: %v", err)
	}
	if err := h.ActivateDevTarget(id); err != nil {
		t.Fatalf("ActivateDevTarget failed: %v", err)
	}

	// Request through the proxy
	req := httptest.NewRequest("GET", "/miniapp/dev/hello", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if got := w.Body.String(); got != "path=/hello" {
		t.Errorf("expected path=/hello, got %q", got)
	}
}

func TestDevProxy_ErrorHandler(t *testing.T) {
	h := NewHandler(&mockDataProvider{}, &mockSender{}, testBotToken, NewStateNotifier(), nil, "")
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	// Register a target that points to a non-existent server
	id, _ := h.RegisterDevTarget("dead", "http://127.0.0.1:19999")
	h.ActivateDevTarget(id)

	req := httptest.NewRequest("GET", "/miniapp/dev/", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadGateway {
		t.Errorf("expected 502, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "Cannot connect") {
		t.Errorf("expected error page with 'Cannot connect', got %q", body)
	}
}

func TestDevProxy_503WhenNotConfigured(t *testing.T) {
	h := NewHandler(&mockDataProvider{}, &mockSender{}, testBotToken, NewStateNotifier(), nil, "")
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest("GET", "/miniapp/dev/", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

func TestDevProxy_APIEndpoint(t *testing.T) {
	notifier := NewStateNotifier()
	h := NewHandler(&mockDataProvider{}, &mockSender{}, testBotToken, notifier, nil, "")
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	initData := testInitData()

	// GET — initially inactive with empty targets
	resp, err := http.Get(ts.URL + "/miniapp/api/dev?initData=" + url.QueryEscape(initData))
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer resp.Body.Close()
	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)
	if result["active"] != false {
		t.Errorf("expected active=false, got %v", result["active"])
	}
	targets, ok := result["targets"].([]any)
	if !ok {
		t.Fatalf("expected targets array, got %T", result["targets"])
	}
	if len(targets) != 0 {
		t.Errorf("expected 0 targets, got %d", len(targets))
	}

	// Register a target via the manager
	id, _ := h.RegisterDevTarget("frontend", "http://localhost:4000")

	// POST — activate target
	body := strings.NewReader(`{"action":"activate","id":"` + id + `"}`)
	resp2, err := http.Post(ts.URL+"/miniapp/api/dev?initData="+url.QueryEscape(initData), "application/json", body)
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp2.Body.Close()
	var result2 map[string]any
	json.NewDecoder(resp2.Body).Decode(&result2)
	if result2["active"] != true {
		t.Errorf("expected active=true after activate, got %v", result2["active"])
	}
	if result2["target"] != "http://localhost:4000" {
		t.Errorf("expected target=http://localhost:4000, got %v", result2["target"])
	}
	if result2["active_id"] != id {
		t.Errorf("expected active_id=%s, got %v", id, result2["active_id"])
	}

	// POST — deactivate
	body2 := strings.NewReader(`{"action":"deactivate"}`)
	resp3, err := http.Post(ts.URL+"/miniapp/api/dev?initData="+url.QueryEscape(initData), "application/json", body2)
	if err != nil {
		t.Fatalf("POST deactivate failed: %v", err)
	}
	defer resp3.Body.Close()
	var result3 map[string]any
	json.NewDecoder(resp3.Body).Decode(&result3)
	if result3["active"] != false {
		t.Errorf("expected active=false after deactivate, got %v", result3["active"])
	}
}

// ── Registration edge cases ──

func TestDevProxy_RegisterUniqueIDs(t *testing.T) {
	h := NewHandler(&mockDataProvider{}, &mockSender{}, testBotToken, NewStateNotifier(), nil, "")

	id1, _ := h.RegisterDevTarget("a", "http://localhost:3000")
	id2, _ := h.RegisterDevTarget("b", "http://localhost:3001")
	id3, _ := h.RegisterDevTarget("c", "http://localhost:3002")

	if id1 == id2 || id2 == id3 || id1 == id3 {
		t.Errorf("IDs must be unique: got %q, %q, %q", id1, id2, id3)
	}
}

func TestDevProxy_RegisterInvalidURL(t *testing.T) {
	h := NewHandler(&mockDataProvider{}, &mockSender{}, testBotToken, NewStateNotifier(), nil, "")

	if _, err := h.RegisterDevTarget("bad", "://not-a-url"); err == nil {
		t.Error("expected error for malformed URL")
	}
}

func TestDevProxy_RegisterVariousLocalhost(t *testing.T) {
	h := NewHandler(&mockDataProvider{}, &mockSender{}, testBotToken, NewStateNotifier(), nil, "")

	cases := []struct {
		name   string
		target string
		ok     bool
	}{
		{"localhost with port", "http://localhost:3000", true},
		{"localhost no port", "http://localhost", true},
		{"127.0.0.1 with port", "http://127.0.0.1:8080", true},
		{"127.0.0.1 no port", "http://127.0.0.1", true},
		{"::1 with port", "http://[::1]:9000", true},
		{"::1 no port", "http://[::1]", true},
		{"external host", "http://evil.com:3000", false},
		{"ip addr", "http://192.168.1.1:3000", false},
		{"0.0.0.0", "http://0.0.0.0:3000", false},
		{"https localhost", "https://localhost:3000", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := h.RegisterDevTarget(tc.name, tc.target)
			if tc.ok && err != nil {
				t.Errorf("expected success, got %v", err)
			}
			if !tc.ok && err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

// ── Activation switching ──

func TestDevProxy_SwitchActiveTarget(t *testing.T) {
	h := NewHandler(&mockDataProvider{}, &mockSender{}, testBotToken, NewStateNotifier(), nil, "")

	id1, _ := h.RegisterDevTarget("api", "http://localhost:8080")
	id2, _ := h.RegisterDevTarget("frontend", "http://localhost:3000")

	// Activate first
	h.ActivateDevTarget(id1)
	got := h.GetDevTarget()
	if !strings.Contains(got, "8080") {
		t.Errorf("expected 8080 in target, got %q", got)
	}

	// Switch to second — should replace without error
	if err := h.ActivateDevTarget(id2); err != nil {
		t.Fatalf("switch activate failed: %v", err)
	}
	got = h.GetDevTarget()
	if !strings.Contains(got, "3000") {
		t.Errorf("expected 3000 in target after switch, got %q", got)
	}
}

func TestDevProxy_ReactivateSameTarget(t *testing.T) {
	h := NewHandler(&mockDataProvider{}, &mockSender{}, testBotToken, NewStateNotifier(), nil, "")

	id, _ := h.RegisterDevTarget("api", "http://localhost:8080")
	h.ActivateDevTarget(id)

	// Re-activating the same target should succeed
	if err := h.ActivateDevTarget(id); err != nil {
		t.Fatalf("re-activate failed: %v", err)
	}
	if got := h.GetDevTarget(); got == "" {
		t.Error("target should still be active after re-activate")
	}
}

func TestDevProxy_ActivateAfterDeactivate(t *testing.T) {
	h := NewHandler(&mockDataProvider{}, &mockSender{}, testBotToken, NewStateNotifier(), nil, "")

	id, _ := h.RegisterDevTarget("api", "http://localhost:8080")
	h.ActivateDevTarget(id)
	h.DeactivateDevTarget()

	// Should be able to re-activate
	if err := h.ActivateDevTarget(id); err != nil {
		t.Fatalf("activate after deactivate failed: %v", err)
	}
	if got := h.GetDevTarget(); got == "" {
		t.Error("expected non-empty target after re-activate")
	}
}

// ── Unregister edge cases ──

func TestDevProxy_UnregisterInactiveTarget(t *testing.T) {
	h := NewHandler(&mockDataProvider{}, &mockSender{}, testBotToken, NewStateNotifier(), nil, "")

	id1, _ := h.RegisterDevTarget("api", "http://localhost:8080")
	id2, _ := h.RegisterDevTarget("frontend", "http://localhost:3000")
	h.ActivateDevTarget(id1) // api is active

	// Unregister the INACTIVE one — proxy should remain pointing at api
	if err := h.UnregisterDevTarget(id2); err != nil {
		t.Fatalf("unregister inactive failed: %v", err)
	}
	if got := h.GetDevTarget(); got == "" {
		t.Error("proxy should still be active after unregistering inactive target")
	}
	if targets := h.ListDevTargets(); len(targets) != 1 {
		t.Errorf("expected 1 target remaining, got %d", len(targets))
	}
}

func TestDevProxy_UnregisterTwice(t *testing.T) {
	h := NewHandler(&mockDataProvider{}, &mockSender{}, testBotToken, NewStateNotifier(), nil, "")

	id, _ := h.RegisterDevTarget("api", "http://localhost:8080")
	h.UnregisterDevTarget(id)

	if err := h.UnregisterDevTarget(id); err == nil {
		t.Error("expected error for double unregister")
	}
}

// ── IPv4 rewrite edge cases ──

func TestDevProxy_IPv4NoRewriteFor127(t *testing.T) {
	h := NewHandler(&mockDataProvider{}, &mockSender{}, testBotToken, NewStateNotifier(), nil, "")

	id, _ := h.RegisterDevTarget("api", "http://127.0.0.1:8080")
	h.ActivateDevTarget(id)

	got := h.GetDevTarget()
	if !strings.Contains(got, "127.0.0.1:8080") {
		t.Errorf("expected 127.0.0.1:8080 unchanged, got %q", got)
	}
}

func TestDevProxy_IPv4NoRewriteForIPv6(t *testing.T) {
	h := NewHandler(&mockDataProvider{}, &mockSender{}, testBotToken, NewStateNotifier(), nil, "")

	id, _ := h.RegisterDevTarget("api", "http://[::1]:9000")
	h.ActivateDevTarget(id)

	got := h.GetDevTarget()
	// [::1] should not be rewritten to 127.0.0.1
	if strings.Contains(got, "127.0.0.1") {
		t.Errorf("expected [::1] NOT to be rewritten, got %q", got)
	}
}

func TestDevProxy_IPv4RewriteLocalhostNoPort(t *testing.T) {
	h := NewHandler(&mockDataProvider{}, &mockSender{}, testBotToken, NewStateNotifier(), nil, "")

	id, _ := h.RegisterDevTarget("api", "http://localhost")
	h.ActivateDevTarget(id)

	got := h.GetDevTarget()
	if strings.Contains(got, "localhost") {
		t.Errorf("expected localhost to be rewritten, got %q", got)
	}
	if !strings.Contains(got, "127.0.0.1") {
		t.Errorf("expected 127.0.0.1 in rewritten target, got %q", got)
	}
}

// ── devStatus ──

func TestDevProxy_DevStatusEmpty(t *testing.T) {
	h := NewHandler(&mockDataProvider{}, &mockSender{}, testBotToken, NewStateNotifier(), nil, "")

	status := h.devStatus()
	if status["active"] != false {
		t.Errorf("expected active=false, got %v", status["active"])
	}
	if status["active_id"] != "" {
		t.Errorf("expected empty active_id, got %v", status["active_id"])
	}
	if status["target"] != "" {
		t.Errorf("expected empty target, got %v", status["target"])
	}
	targets := status["targets"].([]DevTarget)
	if len(targets) != 0 {
		t.Errorf("expected 0 targets, got %d", len(targets))
	}
}

func TestDevProxy_DevStatusTargetsButNoActive(t *testing.T) {
	h := NewHandler(&mockDataProvider{}, &mockSender{}, testBotToken, NewStateNotifier(), nil, "")

	h.RegisterDevTarget("api", "http://localhost:8080")
	h.RegisterDevTarget("frontend", "http://localhost:3000")

	status := h.devStatus()
	if status["active"] != false {
		t.Errorf("expected active=false, got %v", status["active"])
	}
	targets := status["targets"].([]DevTarget)
	if len(targets) != 2 {
		t.Errorf("expected 2 targets, got %d", len(targets))
	}
}

func TestDevProxy_DevStatusReturnsOriginalURL(t *testing.T) {
	h := NewHandler(&mockDataProvider{}, &mockSender{}, testBotToken, NewStateNotifier(), nil, "")

	id, _ := h.RegisterDevTarget("front", "http://localhost:3000")
	h.ActivateDevTarget(id)

	status := h.devStatus()
	// devStatus should return the ORIGINAL URL (localhost), not the rewritten 127.0.0.1
	if status["target"] != "http://localhost:3000" {
		t.Errorf("expected original URL http://localhost:3000, got %v", status["target"])
	}
}

func TestDevProxy_DevStatusActiveID(t *testing.T) {
	h := NewHandler(&mockDataProvider{}, &mockSender{}, testBotToken, NewStateNotifier(), nil, "")

	id1, _ := h.RegisterDevTarget("api", "http://localhost:8080")
	id2, _ := h.RegisterDevTarget("frontend", "http://localhost:3000")

	h.ActivateDevTarget(id2)
	status := h.devStatus()

	if status["active_id"] != id2 {
		t.Errorf("expected active_id=%s, got %v", id2, status["active_id"])
	}

	// Switch
	h.ActivateDevTarget(id1)
	status = h.devStatus()
	if status["active_id"] != id1 {
		t.Errorf("expected active_id=%s after switch, got %v", id1, status["active_id"])
	}

	// Deactivate
	h.DeactivateDevTarget()
	status = h.devStatus()
	if status["active_id"] != "" {
		t.Errorf("expected empty active_id after deactivate, got %v", status["active_id"])
	}
}

// ── ListDevTargets ──

func TestDevProxy_ListDevTargetsEmpty(t *testing.T) {
	h := NewHandler(&mockDataProvider{}, &mockSender{}, testBotToken, NewStateNotifier(), nil, "")
	targets := h.ListDevTargets()
	if len(targets) != 0 {
		t.Errorf("expected 0, got %d", len(targets))
	}
}

func TestDevProxy_ListDevTargetsStableOrder(t *testing.T) {
	h := NewHandler(&mockDataProvider{}, &mockSender{}, testBotToken, NewStateNotifier(), nil, "")

	// Register in reverse order of expected sort
	h.RegisterDevTarget("c", "http://localhost:3003")
	h.RegisterDevTarget("b", "http://localhost:3002")
	h.RegisterDevTarget("a", "http://localhost:3001")

	targets := h.ListDevTargets()
	for i := 1; i < len(targets); i++ {
		if targets[i].ID < targets[i-1].ID {
			t.Errorf("targets not sorted by ID: %s < %s", targets[i].ID, targets[i-1].ID)
		}
	}
}

func TestDevProxy_ListDevTargetsAfterUnregister(t *testing.T) {
	h := NewHandler(&mockDataProvider{}, &mockSender{}, testBotToken, NewStateNotifier(), nil, "")

	id1, _ := h.RegisterDevTarget("a", "http://localhost:3001")
	h.RegisterDevTarget("b", "http://localhost:3002")

	h.UnregisterDevTarget(id1)
	targets := h.ListDevTargets()
	if len(targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(targets))
	}
	if targets[0].Name != "b" {
		t.Errorf("expected remaining target name=b, got %q", targets[0].Name)
	}
}

// ── Proxy path stripping ──

func TestDevProxy_PathStripping(t *testing.T) {
	var capturedPaths []string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPaths = append(capturedPaths, r.URL.Path+"?"+r.URL.RawQuery)
		w.WriteHeader(200)
	}))
	defer backend.Close()

	h := NewHandler(&mockDataProvider{}, &mockSender{}, testBotToken, NewStateNotifier(), nil, "")
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	id, _ := h.RegisterDevTarget("back", backend.URL)
	h.ActivateDevTarget(id)

	cases := []struct {
		path     string
		expected string
	}{
		{"/miniapp/dev/", "/?"},
		{"/miniapp/dev/hello", "/hello?"},
		{"/miniapp/dev/path/deep", "/path/deep?"},
		{"/miniapp/dev/search?q=test", "/search?q=test"},
	}
	for _, tc := range cases {
		capturedPaths = nil
		req := httptest.NewRequest("GET", tc.path, nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if len(capturedPaths) != 1 || capturedPaths[0] != tc.expected {
			t.Errorf("path %q: expected %q, got %v", tc.path, tc.expected, capturedPaths)
		}
	}
}

func TestDevProxy_RootPathStripRedirect(t *testing.T) {
	h := NewHandler(&mockDataProvider{}, &mockSender{}, testBotToken, NewStateNotifier(), nil, "")
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	id, _ := h.RegisterDevTarget("back", "http://127.0.0.1:19999")
	h.ActivateDevTarget(id)

	// /miniapp/dev (no trailing slash) triggers Go's ServeMux redirect to /miniapp/dev/
	req := httptest.NewRequest("GET", "/miniapp/dev", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusMovedPermanently && w.Code != http.StatusTemporaryRedirect {
		t.Errorf("expected redirect for /miniapp/dev, got %d", w.Code)
	}
}

// ── ErrorHandler details ──

func TestDevProxy_ErrorHandlerHTMLContent(t *testing.T) {
	h := NewHandler(&mockDataProvider{}, &mockSender{}, testBotToken, NewStateNotifier(), nil, "")
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	id, _ := h.RegisterDevTarget("dead", "http://127.0.0.1:19999")
	h.ActivateDevTarget(id)

	req := httptest.NewRequest("GET", "/miniapp/dev/", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	// Check Content-Type
	ct := w.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Errorf("expected text/html content-type, got %q", ct)
	}

	body := w.Body.String()
	// Target URL should be displayed
	if !strings.Contains(body, "127.0.0.1:19999") {
		t.Errorf("expected target URL in error page, got %q", body)
	}
	// Should be valid HTML
	if !strings.Contains(body, "<!DOCTYPE html>") {
		t.Errorf("expected HTML doctype in error page")
	}
}

// ── Notifier integration ──

func TestDevProxy_NotifierTriggeredOnRegister(t *testing.T) {
	notifier := NewStateNotifier()
	ch := notifier.Subscribe()
	defer notifier.Unsubscribe(ch)
	h := NewHandler(&mockDataProvider{}, &mockSender{}, testBotToken, notifier, nil, "")

	h.RegisterDevTarget("api", "http://localhost:8080")

	select {
	case <-ch:
	case <-time.After(100 * time.Millisecond):
		t.Error("expected notification on register")
	}
}

func TestDevProxy_NotifierTriggeredOnUnregister(t *testing.T) {
	notifier := NewStateNotifier()
	h := NewHandler(&mockDataProvider{}, &mockSender{}, testBotToken, notifier, nil, "")
	id, _ := h.RegisterDevTarget("api", "http://localhost:8080")

	ch := notifier.Subscribe()
	defer notifier.Unsubscribe(ch)

	h.UnregisterDevTarget(id)

	select {
	case <-ch:
	case <-time.After(100 * time.Millisecond):
		t.Error("expected notification on unregister")
	}
}

func TestDevProxy_NotifierTriggeredOnActivate(t *testing.T) {
	notifier := NewStateNotifier()
	h := NewHandler(&mockDataProvider{}, &mockSender{}, testBotToken, notifier, nil, "")
	id, _ := h.RegisterDevTarget("api", "http://localhost:8080")

	ch := notifier.Subscribe()
	defer notifier.Unsubscribe(ch)

	h.ActivateDevTarget(id)

	select {
	case <-ch:
	case <-time.After(100 * time.Millisecond):
		t.Error("expected notification on activate")
	}
}

func TestDevProxy_NotifierTriggeredOnDeactivate(t *testing.T) {
	notifier := NewStateNotifier()
	h := NewHandler(&mockDataProvider{}, &mockSender{}, testBotToken, notifier, nil, "")
	id, _ := h.RegisterDevTarget("api", "http://localhost:8080")
	h.ActivateDevTarget(id)

	ch := notifier.Subscribe()
	defer notifier.Unsubscribe(ch)

	h.DeactivateDevTarget()

	select {
	case <-ch:
	case <-time.After(100 * time.Millisecond):
		t.Error("expected notification on deactivate")
	}
}

// ── API endpoint edge cases ──

func TestDevAPI_InvalidJSON(t *testing.T) {
	h := NewHandler(&mockDataProvider{}, &mockSender{}, testBotToken, NewStateNotifier(), nil, "")
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	body := strings.NewReader(`{invalid json`)
	resp, err := http.Post(ts.URL+"/miniapp/api/dev?initData="+url.QueryEscape(testInitData()),
		"application/json", body)
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid JSON, got %d", resp.StatusCode)
	}
}

func TestDevAPI_UnknownAction(t *testing.T) {
	h := NewHandler(&mockDataProvider{}, &mockSender{}, testBotToken, NewStateNotifier(), nil, "")
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	body := strings.NewReader(`{"action":"magic"}`)
	resp, err := http.Post(ts.URL+"/miniapp/api/dev?initData="+url.QueryEscape(testInitData()),
		"application/json", body)
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()
	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)
	if result["error"] == nil {
		t.Error("expected error field for unknown action")
	}
}

func TestDevAPI_ActivateMissingID(t *testing.T) {
	h := NewHandler(&mockDataProvider{}, &mockSender{}, testBotToken, NewStateNotifier(), nil, "")
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	body := strings.NewReader(`{"action":"activate"}`)
	resp, err := http.Post(ts.URL+"/miniapp/api/dev?initData="+url.QueryEscape(testInitData()),
		"application/json", body)
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()
	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)
	if result["error"] == nil {
		t.Error("expected error for activate without id")
	}
}

func TestDevAPI_ActivateNonExistentID(t *testing.T) {
	h := NewHandler(&mockDataProvider{}, &mockSender{}, testBotToken, NewStateNotifier(), nil, "")
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	body := strings.NewReader(`{"action":"activate","id":"999"}`)
	resp, err := http.Post(ts.URL+"/miniapp/api/dev?initData="+url.QueryEscape(testInitData()),
		"application/json", body)
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()
	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)
	if result["error"] == nil {
		t.Error("expected error for non-existent ID")
	}
}

func TestDevAPI_MethodNotAllowed(t *testing.T) {
	h := NewHandler(&mockDataProvider{}, &mockSender{}, testBotToken, NewStateNotifier(), nil, "")
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	req, _ := http.NewRequest("DELETE", ts.URL+"/miniapp/api/dev?initData="+url.QueryEscape(testInitData()), nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("DELETE failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", resp.StatusCode)
	}
}

func TestDevAPI_GetReturnsTargetsArray(t *testing.T) {
	notifier := NewStateNotifier()
	h := NewHandler(&mockDataProvider{}, &mockSender{}, testBotToken, notifier, nil, "")
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	// Register two targets
	id1, _ := h.RegisterDevTarget("api", "http://localhost:8080")
	h.RegisterDevTarget("frontend", "http://localhost:3000")
	h.ActivateDevTarget(id1)

	resp, err := http.Get(ts.URL + "/miniapp/api/dev?initData=" + url.QueryEscape(testInitData()))
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer resp.Body.Close()

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)

	targets, ok := result["targets"].([]any)
	if !ok {
		t.Fatalf("expected targets array, got %T", result["targets"])
	}
	if len(targets) != 2 {
		t.Errorf("expected 2 targets, got %d", len(targets))
	}
	if result["active_id"] != id1 {
		t.Errorf("expected active_id=%s, got %v", id1, result["active_id"])
	}
}

func TestDevAPI_DeactivateWhenAlreadyInactive(t *testing.T) {
	h := NewHandler(&mockDataProvider{}, &mockSender{}, testBotToken, NewStateNotifier(), nil, "")
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	body := strings.NewReader(`{"action":"deactivate"}`)
	resp, err := http.Post(ts.URL+"/miniapp/api/dev?initData="+url.QueryEscape(testInitData()),
		"application/json", body)
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)
	// Should succeed even if already inactive
	if result["error"] != nil {
		t.Errorf("expected no error for deactivate when inactive, got %v", result["error"])
	}
	if result["active"] != false {
		t.Errorf("expected active=false, got %v", result["active"])
	}
}

// ── Concurrency ──

func TestDevProxy_ConcurrentRegisterActivate(t *testing.T) {
	h := NewHandler(&mockDataProvider{}, &mockSender{}, testBotToken, NewStateNotifier(), nil, "")

	const n = 50
	done := make(chan struct{}, n)

	for i := 0; i < n; i++ {
		go func(i int) {
			defer func() { done <- struct{}{} }()
			name := fmt.Sprintf("target-%d", i)
			target := fmt.Sprintf("http://localhost:%d", 3000+i)
			id, err := h.RegisterDevTarget(name, target)
			if err != nil {
				return
			}
			h.ActivateDevTarget(id)
			h.GetDevTarget()
			h.ListDevTargets()
		}(i)
	}

	for i := 0; i < n; i++ {
		<-done
	}

	targets := h.ListDevTargets()
	if len(targets) != n {
		t.Errorf("expected %d targets after concurrent registration, got %d", n, len(targets))
	}
}

func TestDevProxy_ConcurrentActivateDeactivate(t *testing.T) {
	h := NewHandler(&mockDataProvider{}, &mockSender{}, testBotToken, NewStateNotifier(), nil, "")

	id1, _ := h.RegisterDevTarget("api", "http://localhost:8080")
	id2, _ := h.RegisterDevTarget("frontend", "http://localhost:3000")

	const n = 100
	done := make(chan struct{}, n)

	for i := 0; i < n; i++ {
		go func(i int) {
			defer func() { done <- struct{}{} }()
			if i%3 == 0 {
				h.ActivateDevTarget(id1)
			} else if i%3 == 1 {
				h.ActivateDevTarget(id2)
			} else {
				h.DeactivateDevTarget()
			}
		}(i)
	}

	for i := 0; i < n; i++ {
		<-done
	}

	// Should not panic — result is indeterminate but state should be consistent
	h.GetDevTarget()
	h.ListDevTargets()
}

// ── SSE dev event with targets ──

func TestSSE_DevEventContainsTargets(t *testing.T) {
	notifier := NewStateNotifier()
	h := NewHandler(&mockDataProvider{}, &mockSender{}, testBotToken, notifier, nil, "")
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	h.RegisterDevTarget("frontend", "http://localhost:3000")

	ts := httptest.NewServer(mux)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/miniapp/api/events?initData=" + url.QueryEscape(testInitData()))
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	deadline := time.After(2 * time.Second)

	var devData string
	for {
		scanDone := make(chan bool, 1)
		go func() { scanDone <- scanner.Scan() }()
		select {
		case ok := <-scanDone:
			if !ok {
				t.Fatal("scanner ended early")
			}
		case <-deadline:
			t.Fatal("timed out waiting for dev event")
		}
		line := scanner.Text()
		if strings.HasPrefix(line, "event: dev") {
			// Read the next data line
			scanDone2 := make(chan bool, 1)
			go func() { scanDone2 <- scanner.Scan() }()
			select {
			case <-scanDone2:
			case <-deadline:
				t.Fatal("timed out waiting for data line")
			}
			devData = strings.TrimPrefix(scanner.Text(), "data: ")
			break
		}
		if devData != "" {
			break
		}
	}

	var parsed map[string]any
	if err := json.Unmarshal([]byte(devData), &parsed); err != nil {
		t.Fatalf("failed to parse dev event data: %v", err)
	}
	targets, ok := parsed["targets"].([]any)
	if !ok {
		t.Fatalf("expected targets array in SSE dev event, got %T", parsed["targets"])
	}
	if len(targets) != 1 {
		t.Errorf("expected 1 target in SSE dev event, got %d", len(targets))
	}
}

// ── validateLocalhostURL ──

func TestValidateLocalhostURL(t *testing.T) {
	cases := []struct {
		target string
		ok     bool
	}{
		{"http://localhost:3000", true},
		{"http://127.0.0.1:8080", true},
		{"http://[::1]:9000", true},
		{"http://example.com", false},
		{"http://10.0.0.1:3000", false},
		{"://bad", false},
	}
	for _, tc := range cases {
		t.Run(tc.target, func(t *testing.T) {
			_, err := validateLocalhostURL(tc.target)
			if tc.ok && err != nil {
				t.Errorf("expected ok, got %v", err)
			}
			if !tc.ok && err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

// ── escapeHTMLString ──

func TestEscapeHTMLString(t *testing.T) {
	cases := []struct {
		in, out string
	}{
		{"hello", "hello"},
		{"<script>", "&lt;script&gt;"},
		{`"quoted"`, "&quot;quoted&quot;"},
		{"a & b", "a &amp; b"},
		{"<>&\"", "&lt;&gt;&amp;&quot;"},
	}
	for _, tc := range cases {
		got := escapeHTMLString(tc.in)
		if got != tc.out {
			t.Errorf("escapeHTMLString(%q) = %q, want %q", tc.in, got, tc.out)
		}
	}
}

// ── Handler implements DevTargetManager ──

func TestHandler_ImplementsDevTargetManager(t *testing.T) {
	h := NewHandler(&mockDataProvider{}, &mockSender{}, testBotToken, NewStateNotifier(), nil, "")
	var _ DevTargetManager = h // compile-time check
}

// ── Nil notifier safety ──

func TestDevProxy_NilNotifier(t *testing.T) {
	// Handler should not panic when notifier is nil
	h := &Handler{
		provider:   &mockDataProvider{},
		sender:     &mockSender{},
		botToken:   testBotToken,
		devTargets: make(map[string]*DevTarget),
	}

	id, err := h.RegisterDevTarget("api", "http://localhost:8080")
	if err != nil {
		t.Fatalf("RegisterDevTarget with nil notifier: %v", err)
	}
	if err := h.ActivateDevTarget(id); err != nil {
		t.Fatalf("ActivateDevTarget with nil notifier: %v", err)
	}
	h.DeactivateDevTarget()
	h.UnregisterDevTarget(id)
}

// ── E2E full lifecycle ──

func TestDevProxy_FullLifecycle(t *testing.T) {
	// Complete user journey: Register → Activate → Proxy works → Switch target →
	// Deactivate → Re-activate → Unregister active → Proxy gone
	backend1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "backend1")
	}))
	defer backend1.Close()
	backend2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "backend2")
	}))
	defer backend2.Close()

	h := NewHandler(&mockDataProvider{}, &mockSender{}, testBotToken, NewStateNotifier(), nil, "")
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	// Step 1: Register two targets
	id1, err := h.RegisterDevTarget("svc1", backend1.URL)
	if err != nil {
		t.Fatalf("register svc1: %v", err)
	}
	id2, err := h.RegisterDevTarget("svc2", backend2.URL)
	if err != nil {
		t.Fatalf("register svc2: %v", err)
	}
	if targets := h.ListDevTargets(); len(targets) != 2 {
		t.Fatalf("expected 2 targets, got %d", len(targets))
	}

	// Step 2: Proxy should return 503 (nothing active yet)
	req := httptest.NewRequest("GET", "/miniapp/dev/", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 before activation, got %d", w.Code)
	}

	// Step 3: Activate first target, proxy should reach backend1
	if err := h.ActivateDevTarget(id1); err != nil {
		t.Fatalf("activate id1: %v", err)
	}
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/miniapp/dev/", nil))
	if w.Code != 200 || w.Body.String() != "backend1" {
		t.Fatalf("expected backend1, got %d %q", w.Code, w.Body.String())
	}

	// Step 4: Switch to second target
	if err := h.ActivateDevTarget(id2); err != nil {
		t.Fatalf("activate id2: %v", err)
	}
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/miniapp/dev/", nil))
	if w.Code != 200 || w.Body.String() != "backend2" {
		t.Fatalf("expected backend2, got %d %q", w.Code, w.Body.String())
	}

	// Step 5: Deactivate
	h.DeactivateDevTarget()
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/miniapp/dev/", nil))
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 after deactivate, got %d", w.Code)
	}

	// Step 6: Re-activate second target (should work)
	if err := h.ActivateDevTarget(id2); err != nil {
		t.Fatalf("re-activate id2: %v", err)
	}
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/miniapp/dev/", nil))
	if w.Body.String() != "backend2" {
		t.Fatalf("expected backend2 after re-activate, got %q", w.Body.String())
	}

	// Step 7: Unregister the ACTIVE target — proxy should go down
	if err := h.UnregisterDevTarget(id2); err != nil {
		t.Fatalf("unregister id2: %v", err)
	}
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/miniapp/dev/", nil))
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 after unregister active, got %d", w.Code)
	}

	// Step 8: Only svc1 remains
	targets := h.ListDevTargets()
	if len(targets) != 1 || targets[0].ID != id1 {
		t.Fatalf("expected only svc1 remaining, got %+v", targets)
	}

	// Step 9: Unregister last
	h.UnregisterDevTarget(id1)
	if targets := h.ListDevTargets(); len(targets) != 0 {
		t.Fatalf("expected 0 targets, got %d", len(targets))
	}
}

// ── SSE dev event on activate/deactivate ──

func TestSSE_DevEventUpdatesOnActivateDeactivate(t *testing.T) {
	notifier := NewStateNotifier()
	h := NewHandler(&mockDataProvider{}, &mockSender{}, testBotToken, notifier, nil, "")
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	id, _ := h.RegisterDevTarget("frontend", "http://localhost:3000")

	resp, err := http.Get(ts.URL + "/miniapp/api/events?initData=" + url.QueryEscape(testInitData()))
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	// Drain initial 6 events (plan, session, skills, dev, context, prompt)
	drainEvents(t, scanner, 6, 2*time.Second)

	// Activate — should trigger a dev event with active=true
	h.ActivateDevTarget(id)
	events := drainEvents(t, scanner, 1, 2*time.Second)
	if !events["dev"] {
		t.Errorf("expected dev event after activate, got: %v", events)
	}

	// Deactivate — should trigger another dev event
	h.DeactivateDevTarget()
	events = drainEvents(t, scanner, 1, 2*time.Second)
	if !events["dev"] {
		t.Errorf("expected dev event after deactivate, got: %v", events)
	}
}

// ── API auth on /miniapp/api/dev ──

func TestDevAPI_AuthRequired(t *testing.T) {
	h := NewHandler(&mockDataProvider{}, &mockSender{}, testBotToken, NewStateNotifier(), nil, "")
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	// GET without initData
	req := httptest.NewRequest("GET", "/miniapp/api/dev", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("GET without auth: expected 401, got %d", w.Code)
	}

	// POST without initData
	req = httptest.NewRequest("POST", "/miniapp/api/dev", strings.NewReader(`{"action":"deactivate"}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("POST without auth: expected 401, got %d", w.Code)
	}
}

// ── Proxy endpoint has no auth (by design, for iframe) ──

func TestDevProxy_NoAuthRequired(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "ok")
	}))
	defer backend.Close()

	h := NewHandler(&mockDataProvider{}, &mockSender{}, testBotToken, NewStateNotifier(), nil, "")
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	id, _ := h.RegisterDevTarget("back", backend.URL)
	h.ActivateDevTarget(id)

	// No initData — should still work (no auth on proxy endpoint)
	req := httptest.NewRequest("GET", "/miniapp/dev/", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Errorf("expected 200 without auth on proxy, got %d", w.Code)
	}
	if w.Body.String() != "ok" {
		t.Errorf("expected 'ok', got %q", w.Body.String())
	}
}

// ── API response JSON structure ──

func TestDevAPI_ResponseTargetFields(t *testing.T) {
	h := NewHandler(&mockDataProvider{}, &mockSender{}, testBotToken, NewStateNotifier(), nil, "")
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	id, _ := h.RegisterDevTarget("frontend", "http://localhost:3000")
	h.ActivateDevTarget(id)

	resp, err := http.Get(ts.URL + "/miniapp/api/dev?initData=" + url.QueryEscape(testInitData()))
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer resp.Body.Close()

	var result struct {
		Active   bool   `json:"active"`
		ActiveID string `json:"active_id"`
		Target   string `json:"target"`
		Targets  []struct {
			ID     string `json:"id"`
			Name   string `json:"name"`
			Target string `json:"target"`
		} `json:"targets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if !result.Active {
		t.Error("expected active=true")
	}
	if result.ActiveID != id {
		t.Errorf("expected active_id=%s, got %s", id, result.ActiveID)
	}
	if result.Target != "http://localhost:3000" {
		t.Errorf("expected target=http://localhost:3000, got %s", result.Target)
	}
	if len(result.Targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(result.Targets))
	}
	tgt := result.Targets[0]
	if tgt.ID != id {
		t.Errorf("target.id: expected %s, got %s", id, tgt.ID)
	}
	if tgt.Name != "frontend" {
		t.Errorf("target.name: expected frontend, got %s", tgt.Name)
	}
	if tgt.Target != "http://localhost:3000" {
		t.Errorf("target.target: expected http://localhost:3000, got %s", tgt.Target)
	}
}

// ── Proxy forwards Host header correctly ──

func TestDevProxy_HostHeaderForwarded(t *testing.T) {
	var capturedHost string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHost = r.Host
		w.WriteHeader(200)
	}))
	defer backend.Close()

	h := NewHandler(&mockDataProvider{}, &mockSender{}, testBotToken, NewStateNotifier(), nil, "")
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	id, _ := h.RegisterDevTarget("back", backend.URL)
	h.ActivateDevTarget(id)

	req := httptest.NewRequest("GET", "/miniapp/dev/", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	// ReverseProxy sets Host to backend's host by default
	if capturedHost == "" {
		t.Error("expected Host header to be forwarded to backend")
	}
}

// ── injectDevProxyScript tests ──

func TestInjectDevProxyScript_HeadTag(t *testing.T) {
	html := []byte(`<!DOCTYPE html><html><head><title>Test</title></head><body><h1>Hi</h1></body></html>`)
	result := injectDevProxyScript(html)
	s := string(result)
	if !strings.Contains(s, `<script data-dev-proxy>`) {
		t.Error("expected script to be injected")
	}
	// Script should appear before </head>
	scriptIdx := strings.Index(s, `<script data-dev-proxy>`)
	headIdx := strings.Index(s, `</head>`)
	if scriptIdx >= headIdx {
		t.Errorf("script should be before </head>: scriptIdx=%d, headIdx=%d", scriptIdx, headIdx)
	}
	// Original content should be preserved
	if !strings.Contains(s, `<title>Test</title>`) {
		t.Error("original <title> content should be preserved")
	}
	if !strings.Contains(s, `<h1>Hi</h1>`) {
		t.Error("original <body> content should be preserved")
	}
}

func TestInjectDevProxyScript_NoHead(t *testing.T) {
	html := []byte(`<!DOCTYPE html><html><body><p>Hello</p></body></html>`)
	result := injectDevProxyScript(html)
	s := string(result)
	if !strings.Contains(s, `<script data-dev-proxy>`) {
		t.Error("expected script to be injected")
	}
	// Script should appear right after <body>
	bodyIdx := strings.Index(s, `<body>`)
	scriptIdx := strings.Index(s, `<script data-dev-proxy>`)
	if scriptIdx != bodyIdx+len(`<body>`) {
		t.Errorf("script should be immediately after <body>: bodyIdx=%d, scriptIdx=%d", bodyIdx, scriptIdx)
	}
}

func TestInjectDevProxyScript_Minimal(t *testing.T) {
	html := []byte(`<div>just a div</div>`)
	result := injectDevProxyScript(html)
	s := string(result)
	if !strings.Contains(s, `<script data-dev-proxy>`) {
		t.Error("expected script to be injected")
	}
	// Script should be at the beginning
	if !strings.HasPrefix(s, `<script data-dev-proxy>`) {
		t.Error("script should be prepended when no head/body tags exist")
	}
	if !strings.Contains(s, `<div>just a div</div>`) {
		t.Error("original content should be preserved")
	}
}

func TestInjectDevProxyScript_BodyWithAttributes(t *testing.T) {
	html := []byte(`<html><body class="dark" id="main"><p>Content</p></body></html>`)
	result := injectDevProxyScript(html)
	s := string(result)
	// Script should be after the full <body ...> opening tag
	bodyCloseIdx := strings.Index(s, `id="main">`) + len(`id="main">`)
	scriptIdx := strings.Index(s, `<script data-dev-proxy>`)
	if scriptIdx != bodyCloseIdx {
		t.Errorf("script should follow <body> closing '>': bodyClose=%d, scriptIdx=%d", bodyCloseIdx, scriptIdx)
	}
}

func TestInjectDevProxyScript_CaseInsensitive(t *testing.T) {
	html := []byte(`<HTML><HEAD><TITLE>Upper</TITLE></HEAD><BODY>test</BODY></HTML>`)
	result := injectDevProxyScript(html)
	s := string(result)
	if !strings.Contains(s, `<script data-dev-proxy>`) {
		t.Error("expected script injection with uppercase tags")
	}
	// Should still inject before </HEAD>
	scriptIdx := strings.Index(s, `<script data-dev-proxy>`)
	headIdx := strings.Index(s, `</HEAD>`)
	if scriptIdx >= headIdx {
		t.Errorf("script should be before </HEAD>")
	}
}

func TestInjectDevProxyScript_ScriptContent(t *testing.T) {
	html := []byte(`<html><head></head><body></body></html>`)
	result := injectDevProxyScript(html)
	s := string(result)
	// Verify the script rewrites fetch and XHR
	if !strings.Contains(s, `/miniapp/dev`) {
		t.Error("script should contain /miniapp/dev prefix")
	}
	if !strings.Contains(s, `window.fetch`) {
		t.Error("script should patch window.fetch")
	}
	if !strings.Contains(s, `XMLHttpRequest.prototype.open`) {
		t.Error("script should patch XMLHttpRequest.prototype.open")
	}
}

func TestDevProxy_ResponseRewriting(t *testing.T) {
	// Backend returns HTML with text/html content type
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<!DOCTYPE html><html><head><title>App</title></head><body><h1>Hello</h1></body></html>`)
	}))
	defer backend.Close()

	h := NewHandler(&mockDataProvider{}, &mockSender{}, testBotToken, NewStateNotifier(), nil, "")
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	id, _ := h.RegisterDevTarget("app", backend.URL)
	h.ActivateDevTarget(id)

	req := httptest.NewRequest("GET", "/miniapp/dev/", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, `<script data-dev-proxy>`) {
		t.Error("expected dev proxy script to be injected into HTML response")
	}
	if !strings.Contains(body, `<title>App</title>`) {
		t.Error("original HTML content should be preserved")
	}
	// Script should be before </head>
	scriptIdx := strings.Index(body, `<script data-dev-proxy>`)
	headIdx := strings.Index(body, `</head>`)
	if scriptIdx >= headIdx {
		t.Errorf("script should be injected before </head>")
	}
}

func TestDevProxy_ResponseRewriting_NonHTML(t *testing.T) {
	// Backend returns JSON — should NOT be modified
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status":"ok"}`)
	}))
	defer backend.Close()

	h := NewHandler(&mockDataProvider{}, &mockSender{}, testBotToken, NewStateNotifier(), nil, "")
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	id, _ := h.RegisterDevTarget("api", backend.URL)
	h.ActivateDevTarget(id)

	req := httptest.NewRequest("GET", "/miniapp/dev/api/status", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if strings.Contains(body, `<script`) {
		t.Error("script should NOT be injected into non-HTML response")
	}
	if body != `{"status":"ok"}` {
		t.Errorf("JSON response should be unchanged, got %q", body)
	}
}

func TestDevProxy_ResponseRewriting_ContentLength(t *testing.T) {
	originalHTML := `<!DOCTYPE html><html><head></head><body>Test</body></html>`
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, originalHTML)
	}))
	defer backend.Close()

	h := NewHandler(&mockDataProvider{}, &mockSender{}, testBotToken, NewStateNotifier(), nil, "")
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	id, _ := h.RegisterDevTarget("app", backend.URL)
	h.ActivateDevTarget(id)

	req := httptest.NewRequest("GET", "/miniapp/dev/", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	body := w.Body.String()
	clHeader := w.Header().Get("Content-Length")
	if clHeader != "" {
		cl, _ := strconv.Atoi(clHeader)
		if cl != len(body) {
			t.Errorf("Content-Length mismatch: header=%d, actual=%d", cl, len(body))
		}
	}
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

func initMiniAppGitRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()

	runGit := func(dir string, args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %s: %s: %v", strings.Join(args, " "), strings.TrimSpace(string(out)), err)
		}
	}

	runGit(repo, "init")
	runGit(repo, "config", "user.email", "test@test.com")
	runGit(repo, "config", "user.name", "Test")

	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("# Test\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	runGit(repo, "add", "-A")
	runGit(repo, "commit", "-m", "initial")

	return repo
}

func TestAPIWorktrees_List(t *testing.T) {
	repo := initMiniAppGitRepo(t)
	wtPath := filepath.Join(repo, ".worktrees", "api-list")
	if _, err := gitpkg.CreateWorktree(repo, wtPath, "plan/api-list"); err != nil {
		t.Fatalf("CreateWorktree: %v", err)
	}

	h := NewHandler(&mockDataProvider{}, &mockSender{}, testBotToken, NewStateNotifier(), nil, repo)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest("GET", "/miniapp/api/worktrees?initData="+url.QueryEscape(testInitData()), nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var items []struct {
		Name   string `json:"name"`
		Branch string `json:"branch"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &items); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 worktree, got %d", len(items))
	}
	if items[0].Name != "api-list" {
		t.Errorf("Name = %q, want %q", items[0].Name, "api-list")
	}
	if items[0].Branch != "plan/api-list" {
		t.Errorf("Branch = %q, want %q", items[0].Branch, "plan/api-list")
	}
}

func TestAPIWorktrees_MergeAndDispose(t *testing.T) {
	repo := initMiniAppGitRepo(t)

	mergePath := filepath.Join(repo, ".worktrees", "api-merge")
	if _, err := gitpkg.CreateWorktree(repo, mergePath, "plan/api-merge"); err != nil {
		t.Fatalf("CreateWorktree merge: %v", err)
	}
	if err := os.WriteFile(filepath.Join(mergePath, "merged.txt"), []byte("from worktree"), 0o644); err != nil {
		t.Fatalf("WriteFile merge: %v", err)
	}
	if err := gitpkg.AutoCommit(mergePath, "add merged.txt"); err != nil {
		t.Fatalf("AutoCommit merge: %v", err)
	}

	disposePath := filepath.Join(repo, ".worktrees", "api-dispose")
	if _, err := gitpkg.CreateWorktree(repo, disposePath, "plan/api-dispose"); err != nil {
		t.Fatalf("CreateWorktree dispose: %v", err)
	}
	if err := os.WriteFile(filepath.Join(disposePath, "dirty.txt"), []byte("dirty"), 0o644); err != nil {
		t.Fatalf("WriteFile dispose: %v", err)
	}

	h := NewHandler(&mockDataProvider{}, &mockSender{}, testBotToken, NewStateNotifier(), nil, repo)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	mergeReq := httptest.NewRequest(
		http.MethodPost,
		"/miniapp/api/worktrees?initData="+url.QueryEscape(testInitData()),
		strings.NewReader(`{"action":"merge","name":"api-merge"}`),
	)
	mergeReq.Header.Set("Content-Type", "application/json")
	mergeW := httptest.NewRecorder()
	mux.ServeHTTP(mergeW, mergeReq)
	if mergeW.Code != http.StatusOK {
		t.Fatalf("merge expected 200, got %d: %s", mergeW.Code, mergeW.Body.String())
	}
	if _, err := os.Stat(filepath.Join(repo, "merged.txt")); os.IsNotExist(err) {
		t.Fatal("merged.txt should exist after merge")
	}

	disposeReq := httptest.NewRequest(
		http.MethodPost,
		"/miniapp/api/worktrees?initData="+url.QueryEscape(testInitData()),
		strings.NewReader(`{"action":"dispose","name":"api-dispose"}`),
	)
	disposeReq.Header.Set("Content-Type", "application/json")
	disposeW := httptest.NewRecorder()
	mux.ServeHTTP(disposeW, disposeReq)
	if disposeW.Code != http.StatusConflict {
		t.Fatalf("dispose without force expected 409, got %d: %s", disposeW.Code, disposeW.Body.String())
	}

	disposeForceReq := httptest.NewRequest(
		http.MethodPost,
		"/miniapp/api/worktrees?initData="+url.QueryEscape(testInitData()),
		strings.NewReader(`{"action":"dispose","name":"api-dispose","force":true}`),
	)
	disposeForceReq.Header.Set("Content-Type", "application/json")
	disposeForceW := httptest.NewRecorder()
	mux.ServeHTTP(disposeForceW, disposeForceReq)
	if disposeForceW.Code != http.StatusOK {
		t.Fatalf("dispose with force expected 200, got %d: %s", disposeForceW.Code, disposeForceW.Body.String())
	}
	if _, err := os.Stat(disposePath); !os.IsNotExist(err) {
		t.Fatalf("worktree dir should be removed, stat err: %v", err)
	}
}

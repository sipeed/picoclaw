package dashboard

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/agent"
	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/health"
	"github.com/sipeed/picoclaw/pkg/providers"
)

func testSetup(t *testing.T) (*health.Server, *config.Config, *agent.AgentLoop, *channels.Manager) {
	t.Helper()
	cfg := config.DefaultConfig()
	cfg.Gateway.Host = "127.0.0.1"
	cfg.Gateway.Port = 0

	msgBus := bus.NewMessageBus()

	provider, _, err := providers.CreateProvider(cfg)
	if err != nil {
		// Use a nil-safe approach: create loop without provider for testing
		t.Logf("Provider creation failed (expected in test): %v", err)
	}

	al := agent.NewAgentLoop(cfg, msgBus, provider)

	cm, err := channels.NewManager(cfg, msgBus)
	if err != nil {
		t.Fatalf("Failed to create channel manager: %v", err)
	}

	srv := health.NewServer("127.0.0.1", 0)
	return srv, cfg, al, cm
}

func TestMount(t *testing.T) {
	srv, cfg, al, cm := testSetup(t)
	Mount(srv, cfg, al, cm)
	// Mount should not panic — that's the main test
}

func TestStatusAPI(t *testing.T) {
	_, cfg, al, cm := testSetup(t)
	startTime := time.Now()

	handler := statusHandler(cfg, al, cm, startTime)
	req := httptest.NewRequest(http.MethodGet, "/dashboard/api/status", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if _, ok := resp["uptime"]; !ok {
		t.Error("missing uptime field")
	}
	if _, ok := resp["running"]; !ok {
		t.Error("missing running field")
	}
	if _, ok := resp["channels"]; !ok {
		t.Error("missing channels field")
	}
}

func TestConfigAPI(t *testing.T) {
	cfg := config.DefaultConfig()
	// Set a fake key to test masking
	if len(cfg.ModelList) > 0 {
		cfg.ModelList[0].APIKey = "sk-1234567890abcdef"
	}

	handler := configGetHandler(cfg)
	req := httptest.NewRequest(http.MethodGet, "/dashboard/api/config", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	body := w.Body.String()
	if strings.Contains(body, "1234567890abcdef") {
		t.Error("API key should be masked in config response")
	}
}

func TestAgentsAPI(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Agents.List = []config.AgentConfig{
		{ID: "test-agent", Name: "Test Agent", Default: true},
	}

	handler := agentsHandler(cfg)
	req := httptest.NewRequest(http.MethodGet, "/dashboard/api/agents", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	list, ok := resp["list"].([]any)
	if !ok {
		t.Fatal("missing list field")
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(list))
	}
}

func TestModelsAPI(t *testing.T) {
	cfg := config.DefaultConfig()
	if len(cfg.ModelList) > 0 {
		cfg.ModelList[0].APIKey = "sk-supersecretkey12345"
	}

	handler := modelsHandler(cfg)
	req := httptest.NewRequest(http.MethodGet, "/dashboard/api/models", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	body := w.Body.String()
	if strings.Contains(body, "supersecretkey") {
		t.Error("API key should be masked in models response")
	}
}

func TestAPIKeyMasking(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"short", "****"},
		{"12345678", "****"},
		{"sk-1234567890abcdef", "sk-...cdef"},
		{"sk-ant-api03-very-long-key-here-xxxx", "sk-...xxxx"},
	}

	for _, tt := range tests {
		got := maskKey(tt.input)
		if got != tt.expected {
			t.Errorf("maskKey(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestSSEBroker(t *testing.T) {
	broker := NewBroker()

	if broker.ClientCount() != 0 {
		t.Fatalf("expected 0 clients, got %d", broker.ClientCount())
	}

	// Test publish with no clients doesn't panic
	broker.Publish("test", `{"hello":"world"}`)
}

func TestFragmentStatus(t *testing.T) {
	_, cfg, al, cm := testSetup(t)
	startTime := time.Now()

	handler := fragmentStatus(cfg, al, cm, startTime)
	req := httptest.NewRequest(http.MethodGet, "/dashboard/fragments/status", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "status-bar") {
		t.Error("status fragment should contain status-bar div")
	}
	if !strings.Contains(body, "Uptime:") {
		t.Error("status fragment should contain uptime")
	}
}

func TestFragmentAgents(t *testing.T) {
	cfg := config.DefaultConfig()
	msgBus := bus.NewMessageBus()
	al := agent.NewAgentLoop(cfg, msgBus, nil)

	handler := fragmentAgents(cfg, al)
	req := httptest.NewRequest(http.MethodGet, "/dashboard/fragments/agents", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "agents-table") {
		t.Error("agents fragment should contain agents-table div")
	}
}

func TestFragmentModels(t *testing.T) {
	cfg := config.DefaultConfig()
	if len(cfg.ModelList) > 0 {
		cfg.ModelList[0].APIKey = "sk-should-be-masked-key"
	}

	handler := fragmentModels(cfg)
	req := httptest.NewRequest(http.MethodGet, "/dashboard/fragments/models", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "models-table") {
		t.Error("models fragment should contain models-table div")
	}
	if strings.Contains(body, "should-be-masked") {
		t.Error("API key should be masked in models fragment")
	}
}

func TestFormatUptime(t *testing.T) {
	tests := []struct {
		duration time.Duration
		expected string
	}{
		{5 * time.Second, "5s"},
		{65 * time.Second, "1m05s"},
		{3661 * time.Second, "1h01m01s"},
	}

	for _, tt := range tests {
		got := formatUptime(tt.duration)
		if got != tt.expected {
			t.Errorf("formatUptime(%v) = %q, want %q", tt.duration, got, tt.expected)
		}
	}
}

func TestExtractProvider(t *testing.T) {
	tests := []struct {
		model    string
		expected string
	}{
		{"openai/gpt-4o", "openai"},
		{"anthropic/claude-3", "anthropic"},
		{"glm-4.7", "glm-4.7"},
		{"", ""},
	}

	for _, tt := range tests {
		got := extractProvider(tt.model)
		if got != tt.expected {
			t.Errorf("extractProvider(%q) = %q, want %q", tt.model, got, tt.expected)
		}
	}
}

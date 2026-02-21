package dashboard

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
)

func TestAgentCreateHandler(t *testing.T) {
	cfg := config.DefaultConfig()
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	handler := agentCreateHandler(cfg, configPath)

	form := url.Values{
		"id":   {"test-agent"},
		"name": {"Test Agent"},
	}
	req := httptest.NewRequest(http.MethodPost, "/dashboard/crud/agents/create", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	if len(cfg.Agents.List) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(cfg.Agents.List))
	}
	if cfg.Agents.List[0].ID != "test-agent" {
		t.Errorf("expected id 'test-agent', got %q", cfg.Agents.List[0].ID)
	}
	if cfg.Agents.List[0].Name != "Test Agent" {
		t.Errorf("expected name 'Test Agent', got %q", cfg.Agents.List[0].Name)
	}

	// Verify config file was written
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("config file should have been created")
	}
}

func TestAgentCreateHandlerDuplicateID(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Agents.List = []config.AgentConfig{
		{ID: "existing", Name: "Existing Agent"},
	}
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	handler := agentCreateHandler(cfg, configPath)

	form := url.Values{
		"id":   {"existing"},
		"name": {"Duplicate"},
	}
	req := httptest.NewRequest(http.MethodPost, "/dashboard/crud/agents/create", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}

	if len(cfg.Agents.List) != 1 {
		t.Fatalf("agent list should still have 1 agent, got %d", len(cfg.Agents.List))
	}
}

func TestAgentCreateHandlerMissingID(t *testing.T) {
	cfg := config.DefaultConfig()
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	handler := agentCreateHandler(cfg, configPath)

	form := url.Values{
		"name": {"No ID Agent"},
	}
	req := httptest.NewRequest(http.MethodPost, "/dashboard/crud/agents/create", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAgentUpdateHandler(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Agents.List = []config.AgentConfig{
		{ID: "agent-1", Name: "Old Name"},
	}
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	handler := agentUpdateHandler(cfg, configPath)

	form := url.Values{
		"id":   {"agent-1"},
		"name": {"New Name"},
	}
	req := httptest.NewRequest(http.MethodPost, "/dashboard/crud/agents/update", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	if cfg.Agents.List[0].Name != "New Name" {
		t.Errorf("expected name 'New Name', got %q", cfg.Agents.List[0].Name)
	}
}

func TestAgentDeleteHandler(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Agents.List = []config.AgentConfig{
		{ID: "agent-1", Name: "Agent 1"},
		{ID: "agent-2", Name: "Agent 2"},
	}
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	handler := agentDeleteHandler(cfg, configPath)

	form := url.Values{
		"id": {"agent-1"},
	}
	req := httptest.NewRequest(http.MethodPost, "/dashboard/crud/agents/delete", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	if len(cfg.Agents.List) != 1 {
		t.Fatalf("expected 1 agent after delete, got %d", len(cfg.Agents.List))
	}
	if cfg.Agents.List[0].ID != "agent-2" {
		t.Errorf("expected remaining agent to be 'agent-2', got %q", cfg.Agents.List[0].ID)
	}
}

func TestDefaultsUpdateHandler(t *testing.T) {
	cfg := config.DefaultConfig()
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	handler := defaultsUpdateHandler(cfg, configPath)

	form := url.Values{
		"model":               {"gpt-4o"},
		"max_tokens":          {"8192"},
		"max_tool_iterations": {"25"},
		"workspace":           {"~/workspace"},
	}
	req := httptest.NewRequest(http.MethodPost, "/dashboard/crud/agents/defaults", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	if cfg.Agents.Defaults.Model != "gpt-4o" {
		t.Errorf("expected model 'gpt-4o', got %q", cfg.Agents.Defaults.Model)
	}
	if cfg.Agents.Defaults.MaxTokens != 8192 {
		t.Errorf("expected max_tokens 8192, got %d", cfg.Agents.Defaults.MaxTokens)
	}
	if cfg.Agents.Defaults.MaxToolIterations != 25 {
		t.Errorf("expected max_tool_iterations 25, got %d", cfg.Agents.Defaults.MaxToolIterations)
	}
	if cfg.Agents.Defaults.Workspace != "~/workspace" {
		t.Errorf("expected workspace '~/workspace', got %q", cfg.Agents.Defaults.Workspace)
	}
}

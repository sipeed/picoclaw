package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/memory"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/session"
	"github.com/sipeed/picoclaw/pkg/utils"
)

func sessionsTestDir(t *testing.T, configPath string) string {
	t.Helper()

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	dir := filepath.Join(cfg.Agents.Defaults.Workspace, "sessions")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	return dir
}

func TestHandleListSessions_JSONLStorage(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	dir := sessionsTestDir(t, configPath)
	store, storeErr := memory.NewJSONLStore(dir)
	if storeErr != nil {
		t.Fatalf("NewJSONLStore() error = %v", storeErr)
	}

	sessionKey := legacyPicoSessionPrefix + "history-jsonl"
	if err := store.AddFullMessage(nil, sessionKey, providers.Message{
		Role:    "user",
		Content: "Explain why the history API is empty after migration.",
	}); err != nil {
		t.Fatalf("AddFullMessage(user) error = %v", err)
	}
	if err := store.AddFullMessage(nil, sessionKey, providers.Message{
		Role:    "assistant",
		Content: "Because the API still reads only legacy JSON session files.",
	}); err != nil {
		t.Fatalf("AddFullMessage(assistant) error = %v", err)
	}
	if err := store.AddFullMessage(nil, sessionKey, providers.Message{
		Role:    "tool",
		Content: "ignored",
	}); err != nil {
		t.Fatalf("AddFullMessage(tool) error = %v", err)
	}
	if err := store.SetSummary(nil, sessionKey, "JSONL-backed session"); err != nil {
		t.Fatalf("SetSummary() error = %v", err)
	}

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/sessions", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var items []sessionListItem
	if err := json.Unmarshal(rec.Body.Bytes(), &items); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].ID != "history-jsonl" {
		t.Fatalf("items[0].ID = %q, want %q", items[0].ID, "history-jsonl")
	}
	if items[0].MessageCount != 2 {
		t.Fatalf("items[0].MessageCount = %d, want 2", items[0].MessageCount)
	}
	if items[0].Title != "Explain why the history API is empty after migration." {
		t.Fatalf(
			"items[0].Title = %q, want %q",
			items[0].Title,
			"Explain why the history API is empty after migration.",
		)
	}
	if items[0].Preview != "Explain why the history API is empty after migration." {
		t.Fatalf("items[0].Preview = %q", items[0].Preview)
	}
}

func TestHandleListSessions_TitleUsesFirstUserMessage(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	dir := sessionsTestDir(t, configPath)
	store, storeErr := memory.NewJSONLStore(dir)
	if storeErr != nil {
		t.Fatalf("NewJSONLStore() error = %v", storeErr)
	}

	sessionKey := legacyPicoSessionPrefix + "summary-title"
	if err := store.AddFullMessage(nil, sessionKey, providers.Message{
		Role:    "user",
		Content: "fallback preview",
	}); err != nil {
		t.Fatalf("AddFullMessage() error = %v", err)
	}
	if err := store.SetSummary(
		nil,
		sessionKey,
		"  This summary is intentionally longer than sixty characters so it must be truncated in the history menu.  ",
	); err != nil {
		t.Fatalf("SetSummary() error = %v", err)
	}

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/sessions", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var items []sessionListItem
	if err := json.Unmarshal(rec.Body.Bytes(), &items); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	expectedTitle := truncateRunes("fallback preview", maxSessionTitleRunes)
	if items[0].Title != expectedTitle {
		t.Fatalf("items[0].Title = %q", items[0].Title)
	}
	if items[0].Preview != "fallback preview" {
		t.Fatalf("items[0].Preview = %q, want %q", items[0].Preview, "fallback preview")
	}
}

func TestHandleGetSession_JSONLStorage(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	dir := sessionsTestDir(t, configPath)
	store, err := memory.NewJSONLStore(dir)
	if err != nil {
		t.Fatalf("NewJSONLStore() error = %v", err)
	}

	sessionKey := legacyPicoSessionPrefix + "detail-jsonl"
	for _, msg := range []providers.Message{
		{Role: "user", Content: "first"},
		{Role: "assistant", Content: "second"},
		{Role: "tool", Content: "ignored"},
	} {
		if err := store.AddFullMessage(nil, sessionKey, msg); err != nil {
			t.Fatalf("AddFullMessage() error = %v", err)
		}
	}
	if err := store.SetSummary(nil, sessionKey, "detail summary"); err != nil {
		t.Fatalf("SetSummary() error = %v", err)
	}

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/sessions/detail-jsonl", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp struct {
		ID       string `json:"id"`
		Summary  string `json:"summary"`
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if resp.ID != "detail-jsonl" {
		t.Fatalf("resp.ID = %q, want %q", resp.ID, "detail-jsonl")
	}
	if resp.Summary != "detail summary" {
		t.Fatalf("resp.Summary = %q, want %q", resp.Summary, "detail summary")
	}
	if len(resp.Messages) != 2 {
		t.Fatalf("len(resp.Messages) = %d, want 2", len(resp.Messages))
	}
	if resp.Messages[0].Role != "user" || resp.Messages[0].Content != "first" {
		t.Fatalf("first message = %#v, want user/first", resp.Messages[0])
	}
	if resp.Messages[1].Role != "assistant" || resp.Messages[1].Content != "second" {
		t.Fatalf("second message = %#v, want assistant/second", resp.Messages[1])
	}
}

func TestHandleSessions_JSONLScopeDiscovery(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	dir := sessionsTestDir(t, configPath)
	store, storeErr := memory.NewJSONLStore(dir)
	if storeErr != nil {
		t.Fatalf("NewJSONLStore() error = %v", storeErr)
	}

	sessionKey := "sk_v1_scope_discovery"
	if err := store.AddFullMessage(nil, sessionKey, providers.Message{
		Role:    "user",
		Content: "scope discovered session",
	}); err != nil {
		t.Fatalf("AddFullMessage() error = %v", err)
	}
	if err := store.SetSummary(nil, sessionKey, "scope summary"); err != nil {
		t.Fatalf("SetSummary() error = %v", err)
	}

	scopeData, err := json.Marshal(session.SessionScope{
		Version:    session.ScopeVersionV1,
		AgentID:    "main",
		Channel:    "pico",
		Account:    "default",
		Dimensions: []string{"sender"},
		Values: map[string]string{
			"sender": "pico:scope-jsonl",
		},
	})
	if err != nil {
		t.Fatalf("Marshal(scope) error = %v", err)
	}
	if err := store.UpsertSessionMeta(nil, sessionKey, scopeData, nil); err != nil {
		t.Fatalf("UpsertSessionMeta() error = %v", err)
	}

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	listRec := httptest.NewRecorder()
	listReq := httptest.NewRequest(http.MethodGet, "/api/sessions", nil)
	mux.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d, body=%s", listRec.Code, http.StatusOK, listRec.Body.String())
	}

	var items []sessionListItem
	if err := json.Unmarshal(listRec.Body.Bytes(), &items); err != nil {
		t.Fatalf("Unmarshal(list) error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].ID != "scope-jsonl" {
		t.Fatalf("items[0].ID = %q, want %q", items[0].ID, "scope-jsonl")
	}

	detailRec := httptest.NewRecorder()
	detailReq := httptest.NewRequest(http.MethodGet, "/api/sessions/scope-jsonl", nil)
	mux.ServeHTTP(detailRec, detailReq)
	if detailRec.Code != http.StatusOK {
		t.Fatalf("detail status = %d, want %d, body=%s", detailRec.Code, http.StatusOK, detailRec.Body.String())
	}

	deleteRec := httptest.NewRecorder()
	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/sessions/scope-jsonl", nil)
	mux.ServeHTTP(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want %d, body=%s", deleteRec.Code, http.StatusNoContent, deleteRec.Body.String())
	}
}

func TestHandleGetSession_OmitsTransientThoughtMessages(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	dir := sessionsTestDir(t, configPath)
	store, err := memory.NewJSONLStore(dir)
	if err != nil {
		t.Fatalf("NewJSONLStore() error = %v", err)
	}

	sessionKey := picoSessionPrefix + "detail-transient-thought"
	for _, msg := range []providers.Message{
		{Role: "user", Content: "hello"},
		{Role: "assistant", ReasoningContent: "internal chain of thought"},
		{Role: "assistant", Content: "final visible answer"},
	} {
		if err := store.AddFullMessage(nil, sessionKey, msg); err != nil {
			t.Fatalf("AddFullMessage() error = %v", err)
		}
	}

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/sessions/detail-transient-thought", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp struct {
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if len(resp.Messages) != 2 {
		t.Fatalf("len(resp.Messages) = %d, want 2", len(resp.Messages))
	}
	if resp.Messages[0].Role != "user" || resp.Messages[0].Content != "hello" {
		t.Fatalf("first message = %#v, want user/hello", resp.Messages[0])
	}
	if resp.Messages[1].Role != "assistant" || resp.Messages[1].Content != "final visible answer" {
		t.Fatalf("second message = %#v, want assistant/final visible answer", resp.Messages[1])
	}
}

func TestHandleGetSession_ReconstructsVisibleMessageToolOutput(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	dir := sessionsTestDir(t, configPath)
	store, err := memory.NewJSONLStore(dir)
	if err != nil {
		t.Fatalf("NewJSONLStore() error = %v", err)
	}

	sessionKey := picoSessionPrefix + "detail-message-tool"
	for _, msg := range []providers.Message{
		{Role: "user", Content: "test"},
		{
			Role:    "assistant",
			Content: "",
			ToolCalls: []providers.ToolCall{
				{
					ID:   "call_1",
					Type: "function",
					Function: &providers.FunctionCall{
						Name:      "message",
						Arguments: `{"content":"visible tool output"}`,
					},
				},
			},
		},
		{Role: "tool", Content: "Message sent to pico:pico:detail-message-tool", ToolCallID: "call_1"},
		{Role: "assistant", Content: handledToolResponseSummaryText},
	} {
		if err := store.AddFullMessage(nil, sessionKey, msg); err != nil {
			t.Fatalf("AddFullMessage() error = %v", err)
		}
	}

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/sessions/detail-message-tool", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp struct {
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if len(resp.Messages) != 3 {
		t.Fatalf("len(resp.Messages) = %d, want 3", len(resp.Messages))
	}
	if !strings.Contains(resp.Messages[1].Content, "`message`") {
		t.Fatalf("tool summary message = %#v, want message tool summary", resp.Messages[1])
	}
	if resp.Messages[2].Role != "assistant" || resp.Messages[2].Content != "visible tool output" {
		t.Fatalf("assistant message = %#v, want visible tool output", resp.Messages[2])
	}
}

func TestHandleGetSession_PreservesFinalAssistantReplyAfterMessageToolOutput(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	dir := sessionsTestDir(t, configPath)
	store, err := memory.NewJSONLStore(dir)
	if err != nil {
		t.Fatalf("NewJSONLStore() error = %v", err)
	}

	sessionKey := picoSessionPrefix + "detail-message-tool-final-reply"
	for _, msg := range []providers.Message{
		{Role: "user", Content: "test"},
		{
			Role: "assistant",
			ToolCalls: []providers.ToolCall{
				{
					ID:   "call_1",
					Type: "function",
					Function: &providers.FunctionCall{
						Name:      "message",
						Arguments: `{"content":"visible tool output"}`,
					},
				},
			},
		},
		{Role: "tool", Content: "Message sent to pico:pico:detail-message-tool-final-reply", ToolCallID: "call_1"},
		{Role: "assistant", Content: "final assistant reply"},
	} {
		if err := store.AddFullMessage(nil, sessionKey, msg); err != nil {
			t.Fatalf("AddFullMessage() error = %v", err)
		}
	}

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/sessions/detail-message-tool-final-reply", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp struct {
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if len(resp.Messages) != 4 {
		t.Fatalf("len(resp.Messages) = %d, want 4", len(resp.Messages))
	}
	if !strings.Contains(resp.Messages[1].Content, "`message`") {
		t.Fatalf("tool summary message = %#v, want message tool summary", resp.Messages[1])
	}
	if resp.Messages[2].Role != "assistant" || resp.Messages[2].Content != "visible tool output" {
		t.Fatalf("interim assistant message = %#v, want visible tool output", resp.Messages[2])
	}
	if resp.Messages[3].Role != "assistant" || resp.Messages[3].Content != "final assistant reply" {
		t.Fatalf("final assistant message = %#v, want final assistant reply", resp.Messages[3])
	}
}

func TestHandleListSessions_MessageCountUsesVisibleTranscript(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	dir := sessionsTestDir(t, configPath)
	store, err := memory.NewJSONLStore(dir)
	if err != nil {
		t.Fatalf("NewJSONLStore() error = %v", err)
	}

	sessionKey := picoSessionPrefix + "list-visible-count"
	for _, msg := range []providers.Message{
		{Role: "user", Content: "test"},
		{
			Role: "assistant",
			ToolCalls: []providers.ToolCall{
				{
					ID:   "call_1",
					Type: "function",
					Function: &providers.FunctionCall{
						Name:      "message",
						Arguments: `{"content":"visible tool output"}`,
					},
				},
			},
		},
		{Role: "tool", Content: "Message sent to pico:pico:list-visible-count", ToolCallID: "call_1"},
		{Role: "assistant", Content: handledToolResponseSummaryText},
	} {
		if err := store.AddFullMessage(nil, sessionKey, msg); err != nil {
			t.Fatalf("AddFullMessage() error = %v", err)
		}
	}

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/sessions", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var items []sessionListItem
	if err := json.Unmarshal(rec.Body.Bytes(), &items); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].MessageCount != 3 {
		t.Fatalf("items[0].MessageCount = %d, want 3", items[0].MessageCount)
	}
}

func TestHandleGetSession_PreservesToolSummaryAndAssistantContent(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	dir := sessionsTestDir(t, configPath)
	store, err := memory.NewJSONLStore(dir)
	if err != nil {
		t.Fatalf("NewJSONLStore() error = %v", err)
	}

	sessionKey := picoSessionPrefix + "detail-tool-summary-and-content"
	for _, msg := range []providers.Message{
		{Role: "user", Content: "check file"},
		{
			Role:    "assistant",
			Content: "model final reply",
			ToolCalls: []providers.ToolCall{
				{
					ID:   "call_1",
					Type: "function",
					Function: &providers.FunctionCall{
						Name:      "read_file",
						Arguments: `{"path":"README.md","start_line":1,"end_line":10}`,
					},
				},
			},
		},
	} {
		if err := store.AddFullMessage(nil, sessionKey, msg); err != nil {
			t.Fatalf("AddFullMessage() error = %v", err)
		}
	}

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/sessions/detail-tool-summary-and-content", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp struct {
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if len(resp.Messages) != 3 {
		t.Fatalf("len(resp.Messages) = %d, want 3", len(resp.Messages))
	}
	if resp.Messages[0].Role != "user" || resp.Messages[0].Content != "check file" {
		t.Fatalf("first message = %#v, want user/check file", resp.Messages[0])
	}
	if !strings.Contains(resp.Messages[1].Content, "`read_file`") {
		t.Fatalf("tool summary message = %#v, want read_file summary", resp.Messages[1])
	}
	if resp.Messages[2].Role != "assistant" || resp.Messages[2].Content != "model final reply" {
		t.Fatalf("assistant message = %#v, want model final reply", resp.Messages[2])
	}
}

func TestHandleGetSession_UsesConfiguredToolFeedbackMaxArgsLength(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	cfg.Agents.Defaults.ToolFeedback.MaxArgsLength = 20
	err = config.SaveConfig(configPath, cfg)
	if err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	dir := sessionsTestDir(t, configPath)
	store, err := memory.NewJSONLStore(dir)
	if err != nil {
		t.Fatalf("NewJSONLStore() error = %v", err)
	}

	argsJSON := `{"path":"README.md","start_line":1,"end_line":10,"extra":"abcdefghijklmnopqrstuvwxyz"}`
	sessionKey := picoSessionPrefix + "detail-tool-summary-max-args"
	err = store.AddFullMessage(nil, sessionKey, providers.Message{Role: "user", Content: "check file"})
	if err != nil {
		t.Fatalf("AddFullMessage(user) error = %v", err)
	}
	err = store.AddFullMessage(nil, sessionKey, providers.Message{
		Role: "assistant",
		ToolCalls: []providers.ToolCall{{
			ID:   "call_1",
			Type: "function",
			Function: &providers.FunctionCall{
				Name:      "read_file",
				Arguments: argsJSON,
			},
		}},
	})
	if err != nil {
		t.Fatalf("AddFullMessage(assistant) error = %v", err)
	}

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/sessions/detail-tool-summary-max-args", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp struct {
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
	}
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if len(resp.Messages) < 2 {
		t.Fatalf("len(resp.Messages) = %d, want at least 2", len(resp.Messages))
	}

	wantPreview := utils.Truncate(argsJSON, 20)
	if !strings.Contains(resp.Messages[1].Content, wantPreview) {
		t.Fatalf("tool summary = %q, want preview %q", resp.Messages[1].Content, wantPreview)
	}
	if strings.Contains(resp.Messages[1].Content, argsJSON) {
		t.Fatalf("tool summary = %q, expected configured truncation", resp.Messages[1].Content)
	}
}

func TestHandleGetSession_IncludesMediaOnlyMessages(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	dir := sessionsTestDir(t, configPath)
	store, err := memory.NewJSONLStore(dir)
	if err != nil {
		t.Fatalf("NewJSONLStore() error = %v", err)
	}

	sessionKey := picoSessionPrefix + "detail-media-only"
	if err := store.AddFullMessage(nil, sessionKey, providers.Message{
		Role:  "user",
		Media: []string{"data:image/png;base64,abc123"},
	}); err != nil {
		t.Fatalf("AddFullMessage(user) error = %v", err)
	}

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/sessions/detail-media-only", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp struct {
		Messages []struct {
			Role    string   `json:"role"`
			Content string   `json:"content"`
			Media   []string `json:"media"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if len(resp.Messages) != 1 {
		t.Fatalf("len(resp.Messages) = %d, want 1", len(resp.Messages))
	}
	if resp.Messages[0].Role != "user" || len(resp.Messages[0].Media) != 1 {
		t.Fatalf("message = %#v, want user message with media", resp.Messages[0])
	}
}

func TestHandleSessions_SupportsJSONLMessagesUpToStoreCap(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	dir := sessionsTestDir(t, configPath)
	store, err := memory.NewJSONLStore(dir)
	if err != nil {
		t.Fatalf("NewJSONLStore() error = %v", err)
	}

	sessionKey := picoSessionPrefix + "detail-large-jsonl"
	largeContent := strings.Repeat("x", 9*1024*1024)
	if err := store.AddFullMessage(nil, sessionKey, providers.Message{
		Role:    "user",
		Content: largeContent,
	}); err != nil {
		t.Fatalf("AddFullMessage() error = %v", err)
	}

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	listRec := httptest.NewRecorder()
	listReq := httptest.NewRequest(http.MethodGet, "/api/sessions", nil)
	mux.ServeHTTP(listRec, listReq)

	if listRec.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d, body=%s", listRec.Code, http.StatusOK, listRec.Body.String())
	}

	var items []sessionListItem
	if err := json.Unmarshal(listRec.Body.Bytes(), &items); err != nil {
		t.Fatalf("list Unmarshal() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}

	detailRec := httptest.NewRecorder()
	detailReq := httptest.NewRequest(http.MethodGet, "/api/sessions/detail-large-jsonl", nil)
	mux.ServeHTTP(detailRec, detailReq)

	if detailRec.Code != http.StatusOK {
		t.Fatalf(
			"detail status = %d, want %d, body=%s",
			detailRec.Code,
			http.StatusOK,
			detailRec.Body.String(),
		)
	}

	var resp struct {
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(detailRec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("detail Unmarshal() error = %v", err)
	}
	if len(resp.Messages) != 1 {
		t.Fatalf("len(resp.Messages) = %d, want 1", len(resp.Messages))
	}
	if resp.Messages[0].Role != "user" {
		t.Fatalf("resp.Messages[0].Role = %q, want %q", resp.Messages[0].Role, "user")
	}
	if got := len(resp.Messages[0].Content); got != len(largeContent) {
		t.Fatalf("len(resp.Messages[0].Content) = %d, want %d", got, len(largeContent))
	}
}

func TestHandleListSessions_UsesImagePreviewForMediaOnlyMessage(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	dir := sessionsTestDir(t, configPath)
	store, err := memory.NewJSONLStore(dir)
	if err != nil {
		t.Fatalf("NewJSONLStore() error = %v", err)
	}

	sessionKey := picoSessionPrefix + "preview-media-only"
	if err := store.AddFullMessage(nil, sessionKey, providers.Message{
		Role:  "user",
		Media: []string{"data:image/png;base64,abc123"},
	}); err != nil {
		t.Fatalf("AddFullMessage() error = %v", err)
	}

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/sessions", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var items []sessionListItem
	if err := json.Unmarshal(rec.Body.Bytes(), &items); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].Preview != "[image]" {
		t.Fatalf("items[0].Preview = %q, want %q", items[0].Preview, "[image]")
	}
	if items[0].MessageCount != 1 {
		t.Fatalf("items[0].MessageCount = %d, want 1", items[0].MessageCount)
	}
}

func TestHandleDeleteSession_JSONLStorage(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	dir := sessionsTestDir(t, configPath)
	store, err := memory.NewJSONLStore(dir)
	if err != nil {
		t.Fatalf("NewJSONLStore() error = %v", err)
	}

	sessionKey := legacyPicoSessionPrefix + "delete-jsonl"
	if err := store.AddFullMessage(nil, sessionKey, providers.Message{
		Role:    "user",
		Content: "delete me",
	}); err != nil {
		t.Fatalf("AddFullMessage() error = %v", err)
	}
	if err := store.SetSummary(nil, sessionKey, "delete summary"); err != nil {
		t.Fatalf("SetSummary() error = %v", err)
	}

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/sessions/delete-jsonl", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusNoContent, rec.Body.String())
	}

	base := filepath.Join(dir, sanitizeSessionKey(sessionKey))
	for _, path := range []string{base + ".jsonl", base + ".meta.json"} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("expected %s to be removed, stat err = %v", path, err)
		}
	}
}

func TestHandleGetSession_LegacyJSONFallback(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	dir := sessionsTestDir(t, configPath)
	manager := session.NewSessionManager(dir)
	sessionKey := legacyPicoSessionPrefix + "legacy-json"
	manager.AddMessage(sessionKey, "user", "legacy user")
	manager.AddMessage(sessionKey, "assistant", "legacy assistant")
	if err := manager.Save(sessionKey); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/sessions/legacy-json", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestHandleSessions_FiltersEmptyJSONLFiles(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	dir := sessionsTestDir(t, configPath)
	base := filepath.Join(dir, sanitizeSessionKey(legacyPicoSessionPrefix+"empty-jsonl"))
	if err := os.WriteFile(base+".jsonl", []byte{}, 0o644); err != nil {
		t.Fatalf("WriteFile(jsonl) error = %v", err)
	}

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	listRec := httptest.NewRecorder()
	listReq := httptest.NewRequest(http.MethodGet, "/api/sessions", nil)
	mux.ServeHTTP(listRec, listReq)

	if listRec.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d, body=%s", listRec.Code, http.StatusOK, listRec.Body.String())
	}

	var items []sessionListItem
	if err := json.Unmarshal(listRec.Body.Bytes(), &items); err != nil {
		t.Fatalf("Unmarshal(list) error = %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("len(items) = %d, want 0", len(items))
	}

	detailRec := httptest.NewRecorder()
	detailReq := httptest.NewRequest(http.MethodGet, "/api/sessions/empty-jsonl", nil)
	mux.ServeHTTP(detailRec, detailReq)

	if detailRec.Code != http.StatusNotFound {
		t.Fatalf("detail status = %d, want %d, body=%s", detailRec.Code, http.StatusNotFound, detailRec.Body.String())
	}
}

func TestHandleSessions_ListsLegacyJSONLWithoutMeta(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	dir := sessionsTestDir(t, configPath)
	sessionKey := legacyPicoSessionPrefix + "missing-meta"
	base := filepath.Join(dir, sanitizeSessionKey(sessionKey))
	line, err := json.Marshal(providers.Message{Role: "user", Content: "recover me"})
	if err != nil {
		t.Fatalf("Marshal(message) error = %v", err)
	}
	if err := os.WriteFile(base+".jsonl", append(line, '\n'), 0o644); err != nil {
		t.Fatalf("WriteFile(jsonl) error = %v", err)
	}

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	listRec := httptest.NewRecorder()
	listReq := httptest.NewRequest(http.MethodGet, "/api/sessions", nil)
	mux.ServeHTTP(listRec, listReq)

	if listRec.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d, body=%s", listRec.Code, http.StatusOK, listRec.Body.String())
	}

	var items []sessionListItem
	if err := json.Unmarshal(listRec.Body.Bytes(), &items); err != nil {
		t.Fatalf("Unmarshal(list) error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].ID != "missing-meta" {
		t.Fatalf("items[0].ID = %q, want %q", items[0].ID, "missing-meta")
	}

	detailRec := httptest.NewRecorder()
	detailReq := httptest.NewRequest(http.MethodGet, "/api/sessions/missing-meta", nil)
	mux.ServeHTTP(detailRec, detailReq)

	if detailRec.Code != http.StatusOK {
		t.Fatalf("detail status = %d, want %d, body=%s", detailRec.Code, http.StatusOK, detailRec.Body.String())
	}
}

func TestHandleSessions_IgnoresMetaJSONInLegacyFallback(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	dir := sessionsTestDir(t, configPath)
	metaOnly := filepath.Join(dir, "agent_main_pico_direct_pico_meta-only.meta.json")
	metaOnlyContent := []byte(`{"key":"agent:main:pico:direct:pico:meta-only","summary":"meta only"}`)
	if err := os.WriteFile(metaOnly, metaOnlyContent, 0o644); err != nil {
		t.Fatalf("WriteFile(meta) error = %v", err)
	}

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	listRec := httptest.NewRecorder()
	listReq := httptest.NewRequest(http.MethodGet, "/api/sessions", nil)
	mux.ServeHTTP(listRec, listReq)

	if listRec.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d, body=%s", listRec.Code, http.StatusOK, listRec.Body.String())
	}

	var items []sessionListItem
	if err := json.Unmarshal(listRec.Body.Bytes(), &items); err != nil {
		t.Fatalf("Unmarshal(list) error = %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("len(items) = %d, want 0", len(items))
	}
}

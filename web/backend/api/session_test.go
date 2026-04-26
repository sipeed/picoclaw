package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

func TestHandleListSessions_TransientThoughtDoesNotInflateMessageCount(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	dir := sessionsTestDir(t, configPath)
	sessionKey := legacyPicoSessionPrefix + "history-jsonl-transient"
	base := filepath.Join(dir, sanitizeSessionKey(sessionKey))
	now := time.Now().UTC()

	rawJSONL := strings.Join([]string{
		`{"role":"user","content":"keep me"}`,
		`{"role":"assistant","content":"","reasoning_content":"dangling thought"}`,
		`{"role":"assistant","content":"and me"}`,
	}, "\n") + "\n"
	if err := os.WriteFile(base+".jsonl", []byte(rawJSONL), 0o644); err != nil {
		t.Fatalf("WriteFile(jsonl) error = %v", err)
	}
	metaData, err := json.Marshal(memory.SessionMeta{
		Key:       sessionKey,
		Count:     3,
		Skip:      0,
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("Marshal(meta) error = %v", err)
	}
	if err := os.WriteFile(base+".meta.json", metaData, 0o644); err != nil {
		t.Fatalf("WriteFile(meta) error = %v", err)
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
	if items[0].ID != "history-jsonl-transient" {
		t.Fatalf("items[0].ID = %q, want %q", items[0].ID, "history-jsonl-transient")
	}
	if items[0].MessageCount != 2 {
		t.Fatalf("items[0].MessageCount = %d, want 2 after dropping transient thought", items[0].MessageCount)
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

func TestHandleGetSession_HidesHandledToolAttachmentsBackedByMediaRefs(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	dir := sessionsTestDir(t, configPath)
	store, err := memory.NewJSONLStore(dir)
	if err != nil {
		t.Fatalf("NewJSONLStore() error = %v", err)
	}

	sessionKey := legacyPicoSessionPrefix + "attachment-history"
	for _, msg := range []providers.Message{
		{Role: "user", Content: "send me the report"},
		{
			Role:    "assistant",
			Content: handledToolResponseSummaryText,
			Attachments: []providers.Attachment{{
				Type:        "file",
				Ref:         "media://attachment-1",
				Filename:    "report.txt",
				ContentType: "text/plain",
			}},
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
	req := httptest.NewRequest(http.MethodGet, "/api/sessions/attachment-history", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp struct {
		Messages []sessionChatMessage `json:"messages"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if len(resp.Messages) != 1 {
		t.Fatalf("len(resp.Messages) = %d, want 1", len(resp.Messages))
	}
	if resp.Messages[0].Role != "user" || resp.Messages[0].Content != "send me the report" {
		t.Fatalf("message = %#v, want only user request", resp.Messages[0])
	}
}

func TestHandleGetSession_ExposesHandledToolAttachmentsWithDurableURL(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	dir := sessionsTestDir(t, configPath)
	store, err := memory.NewJSONLStore(dir)
	if err != nil {
		t.Fatalf("NewJSONLStore() error = %v", err)
	}

	sessionKey := legacyPicoSessionPrefix + "attachment-history-durable"
	for _, msg := range []providers.Message{
		{Role: "user", Content: "send me the report"},
		{
			Role:    "assistant",
			Content: handledToolResponseSummaryText,
			Attachments: []providers.Attachment{{
				Type:        "file",
				URL:         "https://example.com/report.txt",
				Filename:    "report.txt",
				ContentType: "text/plain",
			}},
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
	req := httptest.NewRequest(http.MethodGet, "/api/sessions/attachment-history-durable", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp struct {
		Messages []sessionChatMessage `json:"messages"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if len(resp.Messages) != 2 {
		t.Fatalf("len(resp.Messages) = %d, want 2", len(resp.Messages))
	}

	assistant := resp.Messages[1]
	if assistant.Role != "assistant" {
		t.Fatalf("assistant role = %q, want assistant", assistant.Role)
	}
	if assistant.Content != "" {
		t.Fatalf("assistant content = %q, want empty string", assistant.Content)
	}
	if len(assistant.Attachments) != 1 {
		t.Fatalf("len(assistant.Attachments) = %d, want 1", len(assistant.Attachments))
	}
	if assistant.Attachments[0].URL != "https://example.com/report.txt" {
		t.Fatalf(
			"attachment url = %q, want %q",
			assistant.Attachments[0].URL,
			"https://example.com/report.txt",
		)
	}
	if assistant.Attachments[0].Filename != "report.txt" {
		t.Fatalf("attachment filename = %q, want %q", assistant.Attachments[0].Filename, "report.txt")
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

func TestHandleGetSession_SkipsTransientThoughtMessages(t *testing.T) {
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
			Kind    string `json:"kind"`
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

func TestHandleGetSession_ReconstructsThoughtFromAssistantReasoningContent(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	dir := sessionsTestDir(t, configPath)
	store, err := memory.NewJSONLStore(dir)
	if err != nil {
		t.Fatalf("NewJSONLStore() error = %v", err)
	}

	sessionKey := picoSessionPrefix + "detail-reasoning-content"
	for _, msg := range []providers.Message{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "final visible answer", ReasoningContent: "internal chain of thought"},
	} {
		if err := store.AddFullMessage(nil, sessionKey, msg); err != nil {
			t.Fatalf("AddFullMessage() error = %v", err)
		}
	}

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/sessions/detail-reasoning-content", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp struct {
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
			Kind    string `json:"kind"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if len(resp.Messages) != 3 {
		t.Fatalf("len(resp.Messages) = %d, want 3", len(resp.Messages))
	}
	if resp.Messages[1].Role != "assistant" ||
		resp.Messages[1].Content != "internal chain of thought" ||
		resp.Messages[1].Kind != "thought" {
		t.Fatalf("thought message = %#v, want assistant thought/internal chain of thought", resp.Messages[1])
	}
	if resp.Messages[2].Role != "assistant" || resp.Messages[2].Content != "final visible answer" {
		t.Fatalf("final message = %#v, want assistant/final visible answer", resp.Messages[2])
	}
}

func TestHandleGetSession_ReconstructsRefreshMatrixForThoughtAndToolSummary(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	dir := sessionsTestDir(t, configPath)
	store, err := memory.NewJSONLStore(dir)
	if err != nil {
		t.Fatalf("NewJSONLStore() error = %v", err)
	}

	sessionKey := picoSessionPrefix + "detail-refresh-matrix"
	for _, msg := range []providers.Message{
		{Role: "user", Content: "turn1"},
		{Role: "assistant", Content: "plain visible", ReasoningContent: "plain thought"},
		{Role: "user", Content: "turn2"},
		{
			Role:             "assistant",
			ReasoningContent: "tool thought",
			ToolCalls: []providers.ToolCall{{
				ID:   "call_read_file",
				Type: "function",
				Function: &providers.FunctionCall{
					Name:      "read_file",
					Arguments: `{"path":"README.md"}`,
				},
			}},
		},
		{Role: "tool", ToolCallID: "call_read_file", Content: "file result"},
		{Role: "user", Content: "turn3"},
		{
			Role:    "assistant",
			Content: "tool visible only",
			ToolCalls: []providers.ToolCall{{
				ID:   "call_list_dir",
				Type: "function",
				Function: &providers.FunctionCall{
					Name:      "list_dir",
					Arguments: `{"path":"."}`,
				},
			}},
		},
		{Role: "tool", ToolCallID: "call_list_dir", Content: "dir result"},
		{Role: "user", Content: "turn4"},
		{
			Role:             "assistant",
			Content:          "tool visible and thought",
			ReasoningContent: "tool mixed thought",
			ToolCalls: []providers.ToolCall{{
				ID:   "call_exec",
				Type: "function",
				Function: &providers.FunctionCall{
					Name:      "exec",
					Arguments: `{"command":"pwd"}`,
				},
			}},
		},
		{Role: "tool", ToolCallID: "call_exec", Content: "pwd result"},
	} {
		if err := store.AddFullMessage(nil, sessionKey, msg); err != nil {
			t.Fatalf("AddFullMessage() error = %v", err)
		}
	}

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/sessions/detail-refresh-matrix", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp struct {
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
			Kind    string `json:"kind"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if len(resp.Messages) != 13 {
		t.Fatalf("len(resp.Messages) = %d, want 13", len(resp.Messages))
	}

	assertMessage := func(index int, role, kind, content string) {
		t.Helper()
		msg := resp.Messages[index]
		if msg.Role != role || msg.Kind != kind || msg.Content != content {
			t.Fatalf("messages[%d] = %#v, want role=%q kind=%q content=%q", index, msg, role, kind, content)
		}
	}

	assertMessage(0, "user", "", "turn1")
	assertMessage(1, "assistant", "thought", "plain thought")
	assertMessage(2, "assistant", "", "plain visible")
	assertMessage(3, "user", "", "turn2")
	assertMessage(4, "assistant", "thought", "tool thought")
	if !strings.Contains(resp.Messages[5].Content, "`read_file`") {
		t.Fatalf("messages[5] = %#v, want read_file tool summary", resp.Messages[5])
	}
	assertMessage(6, "user", "", "turn3")
	if !strings.Contains(resp.Messages[7].Content, "`list_dir`") {
		t.Fatalf("messages[7] = %#v, want list_dir tool summary", resp.Messages[7])
	}
	assertMessage(8, "assistant", "", "tool visible only")
	assertMessage(9, "user", "", "turn4")
	assertMessage(10, "assistant", "thought", "tool mixed thought")
	if !strings.Contains(resp.Messages[11].Content, "`exec`") {
		t.Fatalf("messages[11] = %#v, want exec tool summary", resp.Messages[11])
	}
	assertMessage(12, "assistant", "", "tool visible and thought")
}

func TestHandleGetSession_ReconstructsVisibleMessageToolOutputWithoutDuplicateSummary(t *testing.T) {
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
	if len(resp.Messages) != 2 {
		t.Fatalf("len(resp.Messages) = %d, want 2", len(resp.Messages))
	}
	if resp.Messages[0].Role != "user" || resp.Messages[0].Content != "test" {
		t.Fatalf("first message = %#v, want user/test", resp.Messages[0])
	}
	if resp.Messages[1].Role != "assistant" || resp.Messages[1].Content != "visible tool output" {
		t.Fatalf("assistant message = %#v, want visible tool output", resp.Messages[1])
	}
	for _, msg := range resp.Messages {
		if msg.Role == "tool" || strings.Contains(msg.Content, "`message`") {
			t.Fatalf("unexpected raw tool or duplicate message-tool summary: %#v", msg)
		}
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
	if len(resp.Messages) != 3 {
		t.Fatalf("len(resp.Messages) = %d, want 3", len(resp.Messages))
	}
	if resp.Messages[0].Role != "user" || resp.Messages[0].Content != "test" {
		t.Fatalf("first message = %#v, want user/test", resp.Messages[0])
	}
	if resp.Messages[1].Role != "assistant" || resp.Messages[1].Content != "visible tool output" {
		t.Fatalf("interim assistant message = %#v, want visible tool output", resp.Messages[1])
	}
	if resp.Messages[2].Role != "assistant" || resp.Messages[2].Content != "final assistant reply" {
		t.Fatalf("final assistant message = %#v, want final assistant reply", resp.Messages[2])
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
	if items[0].MessageCount != 2 {
		t.Fatalf("items[0].MessageCount = %d, want 2", items[0].MessageCount)
	}
}

func TestHandleGetSession_DoesNotDuplicateAssistantToolCallContent(t *testing.T) {
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
			Content: "Read the file before replying.",
			ToolCalls: []providers.ToolCall{
				{
					ID:   "call_1",
					Type: "function",
					Function: &providers.FunctionCall{
						Name:      "read_file",
						Arguments: `{"path":"README.md","start_line":1,"end_line":10}`,
					},
					ExtraContent: &providers.ExtraContent{
						ToolFeedbackExplanation: "Read the file before replying.",
					},
				},
			},
		},
		{Role: "tool", Content: "raw read_file result", ToolCallID: "call_1"},
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
	if len(resp.Messages) != 2 {
		t.Fatalf("len(resp.Messages) = %d, want 2", len(resp.Messages))
	}
	if resp.Messages[0].Role != "user" || resp.Messages[0].Content != "check file" {
		t.Fatalf("first message = %#v, want user/check file", resp.Messages[0])
	}
	if !strings.Contains(resp.Messages[1].Content, "`read_file`") {
		t.Fatalf("tool summary message = %#v, want read_file summary", resp.Messages[1])
	}
	if !strings.Contains(resp.Messages[1].Content, "Read the file before replying.") {
		t.Fatalf("tool summary message = %#v, want tool explanation", resp.Messages[1])
	}
}

func TestHandleGetSession_PreservesDistinctAssistantToolCallContent(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	dir := sessionsTestDir(t, configPath)
	store, err := memory.NewJSONLStore(dir)
	if err != nil {
		t.Fatalf("NewJSONLStore() error = %v", err)
	}

	sessionKey := picoSessionPrefix + "detail-tool-summary-distinct-content"
	for _, msg := range []providers.Message{
		{Role: "user", Content: "check file"},
		{
			Role:    "assistant",
			Content: "I will summarize the findings after reading the file.",
			ToolCalls: []providers.ToolCall{
				{
					ID:   "call_1",
					Type: "function",
					Function: &providers.FunctionCall{
						Name:      "read_file",
						Arguments: `{"path":"README.md","start_line":1,"end_line":10}`,
					},
					ExtraContent: &providers.ExtraContent{
						ToolFeedbackExplanation: "Read the file before replying.",
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
	req := httptest.NewRequest(http.MethodGet, "/api/sessions/detail-tool-summary-distinct-content", nil)
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
	if !strings.Contains(resp.Messages[1].Content, "`read_file`") {
		t.Fatalf("tool summary message = %#v, want read_file summary", resp.Messages[1])
	}
	if resp.Messages[2].Role != "assistant" ||
		resp.Messages[2].Content != "I will summarize the findings after reading the file." {
		t.Fatalf("assistant content = %#v, want preserved distinct content", resp.Messages[2])
	}
}

func TestHandleGetSession_PreservesMediaWhenAssistantToolCallContentDuplicatesSummary(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	dir := sessionsTestDir(t, configPath)
	store, err := memory.NewJSONLStore(dir)
	if err != nil {
		t.Fatalf("NewJSONLStore() error = %v", err)
	}

	sessionKey := picoSessionPrefix + "detail-tool-summary-duplicate-content-with-media"
	for _, msg := range []providers.Message{
		{Role: "user", Content: "check screenshot"},
		{
			Role:    "assistant",
			Content: "Reviewing the generated screenshot.",
			Media:   []string{"data:image/png;base64,abc123"},
			ToolCalls: []providers.ToolCall{
				{
					ID:   "call_1",
					Type: "function",
					Function: &providers.FunctionCall{
						Name:      "view_image",
						Arguments: `{"path":"artifact.png"}`,
					},
					ExtraContent: &providers.ExtraContent{
						ToolFeedbackExplanation: "Reviewing the generated screenshot.",
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
	req := httptest.NewRequest(http.MethodGet, "/api/sessions/detail-tool-summary-duplicate-content-with-media", nil)
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
	if len(resp.Messages) != 3 {
		t.Fatalf("len(resp.Messages) = %d, want 3", len(resp.Messages))
	}
	if !strings.Contains(resp.Messages[1].Content, "`view_image`") {
		t.Fatalf("tool summary message = %#v, want view_image summary", resp.Messages[1])
	}
	if resp.Messages[2].Role != "assistant" {
		t.Fatalf("assistant message role = %q, want assistant", resp.Messages[2].Role)
	}
	if resp.Messages[2].Content != "Reviewing the generated screenshot." {
		t.Fatalf("assistant content = %q, want preserved duplicated content with media", resp.Messages[2].Content)
	}
	if len(resp.Messages[2].Media) != 1 || resp.Messages[2].Media[0] != "data:image/png;base64,abc123" {
		t.Fatalf("assistant media = %#v, want preserved media", resp.Messages[2].Media)
	}
	for _, msg := range resp.Messages {
		if msg.Role == "tool" || strings.Contains(msg.Content, "raw read_file result") {
			t.Fatalf("unexpected raw tool result in history: %#v", msg)
		}
	}
}

func TestHandleGetSession_PreservesAttachmentsWhenAssistantToolCallContentDuplicatesSummary(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	dir := sessionsTestDir(t, configPath)
	store, err := memory.NewJSONLStore(dir)
	if err != nil {
		t.Fatalf("NewJSONLStore() error = %v", err)
	}

	sessionKey := picoSessionPrefix + "detail-tool-summary-duplicate-content-with-attachments"
	for _, msg := range []providers.Message{
		{Role: "user", Content: "check report"},
		{
			Role:    "assistant",
			Content: "Reviewing the generated report.",
			Attachments: []providers.Attachment{{
				Type:        "file",
				URL:         "https://example.com/report.txt",
				Filename:    "report.txt",
				ContentType: "text/plain",
			}},
			ToolCalls: []providers.ToolCall{
				{
					ID:   "call_1",
					Type: "function",
					Function: &providers.FunctionCall{
						Name:      "read_file",
						Arguments: `{"path":"report.txt"}`,
					},
					ExtraContent: &providers.ExtraContent{
						ToolFeedbackExplanation: "Reviewing the generated report.",
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
	req := httptest.NewRequest(
		http.MethodGet,
		"/api/sessions/detail-tool-summary-duplicate-content-with-attachments",
		nil,
	)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp struct {
		Messages []sessionChatMessage `json:"messages"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if len(resp.Messages) != 3 {
		t.Fatalf("len(resp.Messages) = %d, want 3", len(resp.Messages))
	}
	if !strings.Contains(resp.Messages[1].Content, "`read_file`") {
		t.Fatalf("tool summary message = %#v, want read_file summary", resp.Messages[1])
	}
	if resp.Messages[2].Role != "assistant" {
		t.Fatalf("assistant message role = %q, want assistant", resp.Messages[2].Role)
	}
	if resp.Messages[2].Content != "Reviewing the generated report." {
		t.Fatalf("assistant content = %q, want preserved duplicated content", resp.Messages[2].Content)
	}
	if len(resp.Messages[2].Attachments) != 1 {
		t.Fatalf("len(assistant.Attachments) = %d, want 1", len(resp.Messages[2].Attachments))
	}
	if resp.Messages[2].Attachments[0].URL != "https://example.com/report.txt" {
		t.Fatalf("attachment url = %q, want report URL", resp.Messages[2].Attachments[0].URL)
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
	explanation := "Read README.md first to confirm the current project structure before editing the config example."
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
			ExtraContent: &providers.ExtraContent{
				ToolFeedbackExplanation: explanation,
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

	wantPreview := utils.Truncate(explanation, 20)
	if !strings.Contains(resp.Messages[1].Content, wantPreview) {
		t.Fatalf("tool summary = %q, want preview %q", resp.Messages[1].Content, wantPreview)
	}
	wantArgsPreview := visibleAssistantToolArgsPreview(providers.ToolCall{
		Function: &providers.FunctionCall{Arguments: argsJSON},
	}, 20)
	if !strings.Contains(resp.Messages[1].Content, wantArgsPreview) {
		t.Fatalf("tool summary = %q, want args preview %q", resp.Messages[1].Content, wantArgsPreview)
	}
	if !strings.Contains(resp.Messages[1].Content, "`read_file`") {
		t.Fatalf("tool summary = %q, want read_file summary", resp.Messages[1].Content)
	}
}

func TestHandleGetSession_FallsBackToLegacyToolArgumentsWhenExplanationMissing(t *testing.T) {
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
	sessionKey := picoSessionPrefix + "detail-tool-summary-legacy-args"
	if err := store.AddFullMessage(
		nil,
		sessionKey,
		providers.Message{Role: "user", Content: "check file"},
	); err != nil {
		t.Fatalf("AddFullMessage(user) error = %v", err)
	}
	if err := store.AddFullMessage(nil, sessionKey, providers.Message{
		Role: "assistant",
		ToolCalls: []providers.ToolCall{{
			ID:   "call_1",
			Type: "function",
			Function: &providers.FunctionCall{
				Name:      "read_file",
				Arguments: argsJSON,
			},
		}},
	}); err != nil {
		t.Fatalf("AddFullMessage(assistant) error = %v", err)
	}

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/sessions/detail-tool-summary-legacy-args", nil)
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
	if len(resp.Messages) < 2 {
		t.Fatalf("len(resp.Messages) = %d, want at least 2", len(resp.Messages))
	}

	wantPreview := visibleAssistantToolArgsPreview(providers.ToolCall{
		Function: &providers.FunctionCall{Arguments: argsJSON},
	}, 20)
	if !strings.Contains(resp.Messages[1].Content, "`read_file`") {
		t.Fatalf("tool summary = %q, want read_file summary", resp.Messages[1].Content)
	}
	if !strings.Contains(resp.Messages[1].Content, wantPreview) {
		t.Fatalf("tool summary = %q, want legacy args preview %q", resp.Messages[1].Content, wantPreview)
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

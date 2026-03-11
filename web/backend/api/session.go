package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
)

// registerSessionRoutes binds session list and detail endpoints to the ServeMux.
func (h *Handler) registerSessionRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/sessions", h.handleListSessions)
	mux.HandleFunc("GET /api/sessions/{id}", h.handleGetSession)
	mux.HandleFunc("DELETE /api/sessions/{id}", h.handleDeleteSession)
}

// sessionFile mirrors the on-disk session JSON structure from pkg/session.
type sessionFile struct {
	Key      string              `json:"key"`
	Messages []providers.Message `json:"messages"`
	Summary  string              `json:"summary,omitempty"`
	Created  time.Time           `json:"created"`
	Updated  time.Time           `json:"updated"`
}

// sessionListItem is a lightweight summary returned by GET /api/sessions.
type sessionListItem struct {
	ID           string `json:"id"`
	Preview      string `json:"preview"`
	MessageCount int    `json:"message_count"`
	Created      string `json:"created"`
	Updated      string `json:"updated"`
}

// picoSessionPrefix is the key prefix used by the gateway's routing for Pico
// channel sessions. The full key format is:
//
//	agent:main:pico:direct:pico:<session-uuid>
//
// The sanitized filename replaces ':' with '_', so on disk it becomes:
//
//	agent_main_pico_direct_pico_<session-uuid>.json
const picoSessionPrefix = "agent:main:pico:direct:pico:"

// extractPicoSessionID extracts the session UUID from a full session key.
// Returns the UUID and true if the key matches the Pico session pattern.
func extractPicoSessionID(key string) (string, bool) {
	if strings.HasPrefix(key, picoSessionPrefix) {
		return strings.TrimPrefix(key, picoSessionPrefix), true
	}
	return "", false
}

// sessionsDir resolves the path to the gateway's session storage directory.
// It reads the workspace from config, falling back to ~/.picoclaw/workspace.
func (h *Handler) sessionsDir() (string, error) {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		return "", err
	}

	workspace := cfg.Agents.Defaults.Workspace
	if workspace == "" {
		home, _ := os.UserHomeDir()
		workspace = filepath.Join(home, ".picoclaw", "workspace")
	}

	// Expand ~ prefix
	if len(workspace) > 0 && workspace[0] == '~' {
		home, _ := os.UserHomeDir()
		if len(workspace) > 1 && workspace[1] == '/' {
			workspace = home + workspace[1:]
		} else {
			workspace = home
		}
	}

	return filepath.Join(workspace, "sessions"), nil
}

// handleListSessions returns a list of Pico session summaries.
//
//	GET /api/sessions
func (h *Handler) handleListSessions(w http.ResponseWriter, r *http.Request) {
	dir, err := h.sessionsDir()
	if err != nil {
		http.Error(w, "failed to resolve sessions directory", http.StatusInternalServerError)
		return
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		// Directory doesn't exist yet = no sessions
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]sessionListItem{})
		return
	}

	items := []sessionListItem{}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}

		var sess sessionFile
		if err := json.Unmarshal(data, &sess); err != nil {
			continue
		}

		// Only include Pico channel sessions
		sessionID, ok := extractPicoSessionID(sess.Key)
		if !ok {
			continue
		}

		// Build a preview from the first user message
		preview := ""
		for _, msg := range sess.Messages {
			if msg.Role == "user" && strings.TrimSpace(msg.Content) != "" {
				preview = msg.Content
				break
			}
		}
		if len([]rune(preview)) > 60 {
			preview = string([]rune(preview)[:60]) + "..."
		}
		if preview == "" {
			preview = "(empty)"
		}

		// Only count non-empty user and assistant messages
		validMessageCount := 0
		for _, msg := range sess.Messages {
			if (msg.Role == "user" || msg.Role == "assistant") && strings.TrimSpace(msg.Content) != "" {
				validMessageCount++
			}
		}

		items = append(items, sessionListItem{
			ID:           sessionID,
			Preview:      preview,
			MessageCount: validMessageCount,
			Created:      sess.Created.Format(time.RFC3339),
			Updated:      sess.Updated.Format(time.RFC3339),
		})
	}

	// Sort by updated descending (most recent first)
	sort.Slice(items, func(i, j int) bool {
		return items[i].Updated > items[j].Updated
	})

	// Pagination parameters
	offsetStr := r.URL.Query().Get("offset")
	limitStr := r.URL.Query().Get("limit")

	offset := 0
	limit := 20 // Default limit

	if val, err := strconv.Atoi(offsetStr); err == nil && val >= 0 {
		offset = val
	}
	if val, err := strconv.Atoi(limitStr); err == nil && val > 0 {
		limit = val
	}

	totalItems := len(items)

	end := offset + limit
	if offset >= totalItems {
		items = []sessionListItem{} // Out of bounds, return empty
	} else {
		if end > totalItems {
			end = totalItems
		}
		items = items[offset:end]
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

// handleGetSession returns the full message history for a specific session.
//
//	GET /api/sessions/{id}
func (h *Handler) handleGetSession(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")
	if sessionID == "" {
		http.Error(w, "missing session id", http.StatusBadRequest)
		return
	}

	dir, err := h.sessionsDir()
	if err != nil {
		http.Error(w, "failed to resolve sessions directory", http.StatusInternalServerError)
		return
	}

	// The sanitized filename replaces ':' with '_':
	// agent:main:pico:direct:pico:<uuid> -> agent_main_pico_direct_pico_<uuid>.json
	filename := strings.ReplaceAll(picoSessionPrefix+sessionID, ":", "_") + ".json"

	data, err := os.ReadFile(filepath.Join(dir, filename))
	if err != nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	var sess sessionFile
	if err := json.Unmarshal(data, &sess); err != nil {
		http.Error(w, "failed to parse session", http.StatusInternalServerError)
		return
	}

	// Convert to a simpler format for the frontend
	type chatMessage struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}

	messages := make([]chatMessage, 0, len(sess.Messages))
	for _, msg := range sess.Messages {
		// Only include user and assistant messages that have actual content
		if (msg.Role == "user" || msg.Role == "assistant") && strings.TrimSpace(msg.Content) != "" {
			messages = append(messages, chatMessage{
				Role:    msg.Role,
				Content: msg.Content,
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"id":       sessionID,
		"messages": messages,
		"summary":  sess.Summary,
		"created":  sess.Created.Format(time.RFC3339),
		"updated":  sess.Updated.Format(time.RFC3339),
	})
}

// handleDeleteSession deletes a specific session.
//
//	DELETE /api/sessions/{id}
func (h *Handler) handleDeleteSession(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")
	if sessionID == "" {
		http.Error(w, "missing session id", http.StatusBadRequest)
		return
	}

	dir, err := h.sessionsDir()
	if err != nil {
		http.Error(w, "failed to resolve sessions directory", http.StatusInternalServerError)
		return
	}

	// The sanitized filename replaces ':' with '_':
	// agent:main:pico:direct:pico:<uuid> -> agent_main_pico_direct_pico_<uuid>.json
	filename := strings.ReplaceAll(picoSessionPrefix+sessionID, ":", "_") + ".json"
	filePath := filepath.Join(dir, filename)

	if err := os.Remove(filePath); err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "session not found", http.StatusNotFound)
		} else {
			http.Error(w, "failed to delete session", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

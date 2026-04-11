package api

import (
	"bufio"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/utils"
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
	Title        string `json:"title"`
	Preview      string `json:"preview"`
	MessageCount int    `json:"message_count"`
	Created      string `json:"created"`
	Updated      string `json:"updated"`
}

type sessionChatMessage struct {
	Role    string   `json:"role"`
	Content string   `json:"content"`
	Media   []string `json:"media,omitempty"`
}

type sessionMetaFile struct {
	Key       string    `json:"key"`
	Summary   string    `json:"summary"`
	Skip      int       `json:"skip"`
	Count     int       `json:"count"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// picoSessionPrefix is the key prefix used by the gateway's routing for Pico
// channel sessions. The full key format is:
//
//	agent:main:pico:direct:pico:<session-uuid>
//
// The sanitized filename replaces ':' with '_', so on disk it becomes:
//
//	agent_main_pico_direct_pico_<session-uuid>.json
const (
	picoSessionPrefix          = "agent:main:pico:direct:pico:"
	sanitizedPicoSessionPrefix = "agent_main_pico_direct_pico_"
	// Keep the session API aligned with the shared JSONL store reader limit in
	// pkg/memory/jsonl.go so oversized lines fail consistently everywhere.
	maxSessionJSONLLineSize = 10 * 1024 * 1024
	maxSessionTitleRunes    = 60

	handledToolResponseSummaryText = "Requested output delivered via tool attachment."
)

func defaultToolFeedbackMaxArgsLength() int {
	defaults := config.AgentDefaults{}
	return defaults.GetToolFeedbackMaxArgsLength()
}

// extractPicoSessionID extracts the session UUID from a full session key.
// Returns the UUID and true if the key matches the Pico session pattern.
func extractPicoSessionID(key string) (string, bool) {
	if strings.HasPrefix(key, picoSessionPrefix) {
		return strings.TrimPrefix(key, picoSessionPrefix), true
	}
	return "", false
}

func extractPicoSessionIDFromSanitizedKey(key string) (string, bool) {
	if strings.HasPrefix(key, sanitizedPicoSessionPrefix) {
		return strings.TrimPrefix(key, sanitizedPicoSessionPrefix), true
	}
	return "", false
}

func sanitizeSessionKey(key string) string {
	return strings.ReplaceAll(key, ":", "_")
}

func (h *Handler) readLegacySession(dir, sessionID string) (sessionFile, error) {
	path := filepath.Join(dir, sanitizeSessionKey(picoSessionPrefix+sessionID)+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return sessionFile{}, err
	}

	var sess sessionFile
	if err := json.Unmarshal(data, &sess); err != nil {
		return sessionFile{}, err
	}
	return sess, nil
}

func (h *Handler) readSessionMeta(path, sessionKey string) (sessionMetaFile, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return sessionMetaFile{Key: sessionKey}, nil
	}
	if err != nil {
		return sessionMetaFile{}, err
	}

	var meta sessionMetaFile
	if err := json.Unmarshal(data, &meta); err != nil {
		return sessionMetaFile{}, err
	}
	if meta.Key == "" {
		meta.Key = sessionKey
	}
	return meta, nil
}

func (h *Handler) readSessionMessages(path string, skip int) ([]providers.Message, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	msgs := make([]providers.Message, 0)
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), maxSessionJSONLLineSize)

	seen := 0
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		seen++
		if seen <= skip {
			continue
		}

		var msg providers.Message
		if err := json.Unmarshal(line, &msg); err != nil {
			continue
		}
		msgs = append(msgs, msg)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return msgs, nil
}

func (h *Handler) readJSONLSession(dir, sessionID string) (sessionFile, error) {
	sessionKey := picoSessionPrefix + sessionID
	base := filepath.Join(dir, sanitizeSessionKey(sessionKey))
	jsonlPath := base + ".jsonl"
	metaPath := base + ".meta.json"

	meta, err := h.readSessionMeta(metaPath, sessionKey)
	if err != nil {
		return sessionFile{}, err
	}

	messages, err := h.readSessionMessages(jsonlPath, meta.Skip)
	if err != nil {
		return sessionFile{}, err
	}

	updated := meta.UpdatedAt
	created := meta.CreatedAt
	if created.IsZero() || updated.IsZero() {
		if info, statErr := os.Stat(jsonlPath); statErr == nil {
			if created.IsZero() {
				created = info.ModTime()
			}
			if updated.IsZero() {
				updated = info.ModTime()
			}
		}
	}

	return sessionFile{
		Key:      meta.Key,
		Messages: messages,
		Summary:  meta.Summary,
		Created:  created,
		Updated:  updated,
	}, nil
}

func buildSessionListItem(sessionID string, sess sessionFile, toolFeedbackMaxArgsLength int) sessionListItem {
	preview := ""
	for _, msg := range sess.Messages {
		if msg.Role == "user" {
			preview = sessionMessagePreview(msg)
		}
		if preview != "" {
			break
		}
	}
	preview = truncateRunes(preview, maxSessionTitleRunes)

	if preview == "" {
		preview = "(empty)"
	}
	title := preview

	validMessageCount := len(visibleSessionMessages(sess.Messages, toolFeedbackMaxArgsLength))

	return sessionListItem{
		ID:           sessionID,
		Title:        title,
		Preview:      preview,
		MessageCount: validMessageCount,
		Created:      sess.Created.Format(time.RFC3339),
		Updated:      sess.Updated.Format(time.RFC3339),
	}
}

func isEmptySession(sess sessionFile) bool {
	return len(sess.Messages) == 0 && strings.TrimSpace(sess.Summary) == ""
}

func truncateRunes(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	runes := []rune(strings.TrimSpace(s))
	if len(runes) <= maxLen {
		return string(runes)
	}
	return string(runes[:maxLen]) + "..."
}

func sessionMessageVisible(msg providers.Message) bool {
	return strings.TrimSpace(msg.Content) != "" || len(msg.Media) > 0
}

func sessionMessagePreview(msg providers.Message) string {
	if content := strings.TrimSpace(msg.Content); content != "" {
		return content
	}
	if len(msg.Media) > 0 {
		return "[image]"
	}
	return ""
}

func visibleSessionMessages(messages []providers.Message, toolFeedbackMaxArgsLength int) []sessionChatMessage {
	transcript := make([]sessionChatMessage, 0, len(messages))

	for _, msg := range messages {
		switch msg.Role {
		case "user":
			if sessionMessageVisible(msg) {
				transcript = append(transcript, sessionChatMessage{
					Role:    "user",
					Content: msg.Content,
					Media:   append([]string(nil), msg.Media...),
				})
			}

		case "assistant":
			toolSummaryMessages := visibleAssistantToolSummaryMessages(msg.ToolCalls, toolFeedbackMaxArgsLength)
			if len(toolSummaryMessages) > 0 {
				transcript = append(transcript, toolSummaryMessages...)
			}

			visibleToolMessages := visibleAssistantToolMessages(msg.ToolCalls)
			if len(visibleToolMessages) > 0 {
				transcript = append(transcript, visibleToolMessages...)
			}

			// Pico web chat can persist both visible `message` tool output and a
			// later plain assistant reply in the same turn. Hide only the fixed
			// internal summary that marks handled tool delivery.
			if !sessionMessageVisible(msg) || assistantMessageInternalOnly(msg) {
				continue
			}

			transcript = append(transcript, sessionChatMessage{
				Role:    "assistant",
				Content: msg.Content,
				Media:   append([]string(nil), msg.Media...),
			})
		}
	}

	return transcript
}

func assistantMessageInternalOnly(msg providers.Message) bool {
	return strings.TrimSpace(msg.Content) == handledToolResponseSummaryText
}

func visibleAssistantToolSummaryMessages(
	toolCalls []providers.ToolCall,
	toolFeedbackMaxArgsLength int,
) []sessionChatMessage {
	if len(toolCalls) == 0 {
		return nil
	}
	if toolFeedbackMaxArgsLength <= 0 {
		toolFeedbackMaxArgsLength = defaultToolFeedbackMaxArgsLength()
	}

	messages := make([]sessionChatMessage, 0, len(toolCalls))
	for _, tc := range toolCalls {
		name := tc.Name
		argsJSON := ""
		if tc.Function != nil {
			if name == "" {
				name = tc.Function.Name
			}
			argsJSON = tc.Function.Arguments
		}

		if strings.TrimSpace(name) == "" {
			continue
		}

		if strings.TrimSpace(argsJSON) == "" && len(tc.Arguments) > 0 {
			if encodedArgs, err := json.Marshal(tc.Arguments); err == nil {
				argsJSON = string(encodedArgs)
			}
		}

		argsPreview := strings.TrimSpace(argsJSON)
		if argsPreview == "" {
			argsPreview = "{}"
		}

		messages = append(messages, sessionChatMessage{
			Role:    "assistant",
			Content: utils.FormatToolFeedbackMessage(name, utils.Truncate(argsPreview, toolFeedbackMaxArgsLength)),
		})
	}

	return messages
}

func visibleAssistantToolMessages(toolCalls []providers.ToolCall) []sessionChatMessage {
	if len(toolCalls) == 0 {
		return nil
	}

	messages := make([]sessionChatMessage, 0, len(toolCalls))
	for _, tc := range toolCalls {
		name := tc.Name
		argsJSON := ""
		if tc.Function != nil {
			if name == "" {
				name = tc.Function.Name
			}
			argsJSON = tc.Function.Arguments
		}

		switch name {
		case "message":
			var args struct {
				Content string `json:"content"`
			}
			if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
				continue
			}
			if strings.TrimSpace(args.Content) == "" {
				continue
			}
			messages = append(messages, sessionChatMessage{
				Role:    "assistant",
				Content: args.Content,
			})
		}
	}

	return messages
}

// sessionsDir resolves the path to the gateway's session storage directory.
// It reads the workspace from config, falling back to ~/.picoclaw/workspace.
func (h *Handler) sessionsDir() (string, error) {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		return "", err
	}

	return resolveSessionsDir(cfg.Agents.Defaults.Workspace), nil
}

func (h *Handler) sessionRuntimeSettings() (string, int, error) {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		return "", 0, err
	}

	return resolveSessionsDir(cfg.Agents.Defaults.Workspace), cfg.Agents.Defaults.GetToolFeedbackMaxArgsLength(), nil
}

func resolveSessionsDir(workspace string) string {
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

	return filepath.Join(workspace, "sessions")
}

// handleListSessions returns a list of Pico session summaries.
//
//	GET /api/sessions
func (h *Handler) handleListSessions(w http.ResponseWriter, r *http.Request) {
	dir, toolFeedbackMaxArgsLength, err := h.sessionRuntimeSettings()
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
	seen := make(map[string]struct{})

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		var (
			sessionID string
			sess      sessionFile
			loadErr   error
			ok        bool
		)

		switch {
		case strings.HasSuffix(name, ".jsonl"):
			sessionID, ok = extractPicoSessionIDFromSanitizedKey(strings.TrimSuffix(name, ".jsonl"))
			if !ok {
				continue
			}
			sess, loadErr = h.readJSONLSession(dir, sessionID)
			if loadErr == nil && isEmptySession(sess) {
				continue
			}
		case strings.HasSuffix(name, ".meta.json"):
			continue
		case filepath.Ext(name) == ".json":
			base := strings.TrimSuffix(name, ".json")
			if _, statErr := os.Stat(filepath.Join(dir, base+".jsonl")); statErr == nil {
				if jsonlSessionID, found := extractPicoSessionIDFromSanitizedKey(base); found {
					if jsonlSess, jsonlErr := h.readJSONLSession(
						dir,
						jsonlSessionID,
					); jsonlErr == nil &&
						!isEmptySession(jsonlSess) {
						continue
					}
				}
			}
			data, err := os.ReadFile(filepath.Join(dir, name))
			if err != nil {
				continue
			}
			if err := json.Unmarshal(data, &sess); err != nil {
				continue
			}
			if isEmptySession(sess) {
				continue
			}
			sessionID, ok = extractPicoSessionID(sess.Key)
			if !ok {
				continue
			}
			if _, exists := seen[sessionID]; exists {
				continue
			}
		default:
			continue
		}

		if loadErr != nil {
			continue
		}
		if _, exists := seen[sessionID]; exists {
			continue
		}

		seen[sessionID] = struct{}{}
		items = append(items, buildSessionListItem(sessionID, sess, toolFeedbackMaxArgsLength))
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

	dir, toolFeedbackMaxArgsLength, err := h.sessionRuntimeSettings()
	if err != nil {
		http.Error(w, "failed to resolve sessions directory", http.StatusInternalServerError)
		return
	}

	sess, err := h.readJSONLSession(dir, sessionID)
	if err == nil && isEmptySession(sess) {
		err = os.ErrNotExist
	}
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			sess, err = h.readLegacySession(dir, sessionID)
			if err == nil && isEmptySession(sess) {
				err = os.ErrNotExist
			}
		}
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				http.Error(w, "session not found", http.StatusNotFound)
			} else {
				http.Error(w, "failed to parse session", http.StatusInternalServerError)
			}
			return
		}
	}

	messages := visibleSessionMessages(sess.Messages, toolFeedbackMaxArgsLength)

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

	base := filepath.Join(dir, sanitizeSessionKey(picoSessionPrefix+sessionID))
	jsonlPath := base + ".jsonl"
	metaPath := base + ".meta.json"
	legacyPath := base + ".json"

	removed := false
	for _, path := range []string{jsonlPath, metaPath, legacyPath} {
		if err := os.Remove(path); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			http.Error(w, "failed to delete session", http.StatusInternalServerError)
			return
		}
		removed = true
	}

	if !removed {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

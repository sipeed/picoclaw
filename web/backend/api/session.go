package api

import (
	"bufio"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"os/exec"
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
	mux.HandleFunc("PATCH /api/sessions/{id}", h.handleRenameSession)
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
	Channel      string `json:"channel"`
	PeerName     string `json:"peer_name,omitempty"`
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

// knownSanitizedPrefixes maps channel prefixes to human-readable channel names.
var knownSanitizedPrefixes = []struct {
	prefix  string
	channel string
}{
	{sanitizedPicoSessionPrefix, "pico"},
	{"agent_main_whatsapp_native_direct_", "whatsapp"},
	{"agent_main_telegram_direct_", "telegram"},
	{"agent_main_discord_direct_", "discord"},
	{"agent_main_slack_direct_", "slack"},
	{"agent_main_matrix_direct_", "matrix"},
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

// extractAnySessionID extracts session ID and channel name from any known
// sanitized key prefix. Returns sessionID, channel, ok.
func extractAnySessionID(key string) (string, string, bool) {
	for _, p := range knownSanitizedPrefixes {
		if strings.HasPrefix(key, p.prefix) {
			return strings.TrimPrefix(key, p.prefix), p.channel, true
		}
	}
	return "", "", false
}

// extractAnySessionIDFromKey extracts session ID and channel from unsanitized key.
func extractAnySessionIDFromKey(key string) (string, string, bool) {
	for _, p := range knownSanitizedPrefixes {
		unsanitized := strings.ReplaceAll(p.prefix, "_", ":")
		if strings.HasPrefix(key, unsanitized) {
			return strings.TrimPrefix(key, unsanitized), p.channel, true
		}
	}
	return "", "", false
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
	return h.readJSONLSessionByBase(dir, sanitizeSessionKey(picoSessionPrefix+sessionID))
}

// readJSONLSessionByBase reads a JSONL session by its sanitized base name (without extension).
func (h *Handler) readJSONLSessionByBase(dir, baseName string) (sessionFile, error) {
	base := filepath.Join(dir, baseName)
	jsonlPath := base + ".jsonl"
	metaPath := base + ".meta.json"

	sessionKey := strings.ReplaceAll(baseName, "_", ":")
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

func buildSessionListItem(sessionID, channel string, sess sessionFile) sessionListItem {
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

	validMessageCount := len(visibleSessionMessages(sess.Messages))

	return sessionListItem{
		ID:           sessionID,
		Title:        title,
		Preview:      preview,
		Channel:      channel,
		MessageCount: validMessageCount,
		Created:      sess.Created.Format(time.RFC3339),
		Updated:      sess.Updated.Format(time.RFC3339),
	}
}

func isEmptySession(sess sessionFile) bool {
	return len(sess.Messages) == 0 && strings.TrimSpace(sess.Summary) == ""
}

// whatsappPeerInfo holds resolved WhatsApp peer data.
type whatsappPeerInfo struct {
	Phone    string // "+5521..."
	PushName string // "Willian Santos"
}

// resolveWhatsAppLIDs resolves WhatsApp LID session IDs to phone numbers
// and push names using the whatsmeow SQLite store.
func (h *Handler) resolveWhatsAppLIDs(lids []string) map[string]whatsappPeerInfo {
	if len(lids) == 0 {
		return nil
	}

	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		return nil
	}
	workspace := cfg.Agents.Defaults.Workspace
	if workspace == "" {
		home, _ := os.UserHomeDir()
		workspace = filepath.Join(home, ".picoclaw", "workspace")
	}
	if len(workspace) > 0 && workspace[0] == '~' {
		home, _ := os.UserHomeDir()
		if len(workspace) > 1 && workspace[1] == '/' {
			workspace = home + workspace[1:]
		} else {
			workspace = home
		}
	}

	dbPath := filepath.Join(workspace, "whatsapp", "store.db")
	if _, err := os.Stat(dbPath); err != nil {
		return nil
	}

	result := make(map[string]whatsappPeerInfo)
	for _, lid := range lids {
		cleanLID := strings.TrimSuffix(lid, "@lid")
		info := whatsappPeerInfo{}

		// Resolve LID → phone
		query := "SELECT pn FROM whatsmeow_lid_map WHERE lid='" + cleanLID + "';"
		if out, err := exec.Command("sqlite3", dbPath, query).Output(); err == nil {
			phone := strings.TrimSpace(string(out))
			if phone != "" {
				info.Phone = "+" + phone

				// Resolve phone → name via contacts table (prefer push_name, fallback to full_name)
				contactQuery := "SELECT COALESCE(NULLIF(push_name,''), NULLIF(full_name,'')) FROM whatsmeow_contacts WHERE their_jid LIKE '" + phone + "%' LIMIT 1;"
				if nameOut, err := exec.Command("sqlite3", dbPath, contactQuery).Output(); err == nil {
					name := strings.TrimSpace(string(nameOut))
					if name != "" {
						info.PushName = name
					}
				}
			}
		}

		if info.Phone != "" {
			result[lid] = info
		}
	}
	return result
}

// findAndReadSession tries all known channel prefixes to find and read a session.
func (h *Handler) findAndReadSession(dir, sessionID string) (sessionFile, error) {
	for _, p := range knownSanitizedPrefixes {
		baseName := p.prefix + sessionID
		sess, err := h.readJSONLSessionByBase(dir, baseName)
		if err == nil && !isEmptySession(sess) {
			return sess, nil
		}
		// Try legacy JSON
		legacyPath := filepath.Join(dir, baseName+".json")
		data, err := os.ReadFile(legacyPath)
		if err == nil {
			var legacySess sessionFile
			if json.Unmarshal(data, &legacySess) == nil && !isEmptySession(legacySess) {
				return legacySess, nil
			}
		}
	}
	return sessionFile{}, os.ErrNotExist
}

// findSessionFiles returns all file paths for a session across all channel prefixes.
func findSessionFiles(dir, sessionID string) []string {
	var paths []string
	for _, p := range knownSanitizedPrefixes {
		base := filepath.Join(dir, p.prefix+sessionID)
		for _, ext := range []string{".jsonl", ".meta.json", ".json"} {
			path := base + ext
			if _, err := os.Stat(path); err == nil {
				paths = append(paths, path)
			}
		}
	}
	return paths
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

func visibleSessionMessages(messages []providers.Message) []sessionChatMessage {
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
			visibleToolMessages := visibleAssistantToolMessages(msg.ToolCalls)
			if len(visibleToolMessages) > 0 {
				transcript = append(transcript, visibleToolMessages...)
			}

			// Pico web chat can persist both visible `message` tool output and a
			// later plain assistant reply in the same turn. Hide only the fixed
			// internal summary that marks handled tool delivery.
			if len(visibleToolMessages) > 0 || !sessionMessageVisible(msg) || assistantMessageInternalOnly(msg) {
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

		var channel string
		switch {
		case strings.HasSuffix(name, ".jsonl"):
			baseName := strings.TrimSuffix(name, ".jsonl")
			sessionID, channel, ok = extractAnySessionID(baseName)
			if !ok {
				continue
			}
			sess, loadErr = h.readJSONLSessionByBase(dir, baseName)
			if loadErr == nil && isEmptySession(sess) {
				continue
			}
		case strings.HasSuffix(name, ".meta.json"):
			continue
		case filepath.Ext(name) == ".json":
			base := strings.TrimSuffix(name, ".json")
			if _, statErr := os.Stat(filepath.Join(dir, base+".jsonl")); statErr == nil {
				if _, _, found := extractAnySessionID(base); found {
					if jsonlSess, jsonlErr := h.readJSONLSessionByBase(
						dir,
						base,
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
			sessionID, channel, ok = extractAnySessionIDFromKey(sess.Key)
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
		items = append(items, buildSessionListItem(sessionID, channel, sess))
	}

	// Sort by updated descending (most recent first)
	sort.Slice(items, func(i, j int) bool {
		return items[i].Updated > items[j].Updated
	})

	// Apply custom titles
	if customTitles, err := h.loadCustomTitles(); err == nil {
		for i := range items {
			if t, ok := customTitles[items[i].ID]; ok {
				items[i].Title = t
			}
		}
	}

	// Resolve WhatsApp LIDs to phone numbers
	var whatsappLIDs []string
	for _, item := range items {
		if item.Channel == "whatsapp" {
			whatsappLIDs = append(whatsappLIDs, item.ID)
		}
	}
	if lidMap := h.resolveWhatsAppLIDs(whatsappLIDs); len(lidMap) > 0 {
		for i := range items {
			if info, ok := lidMap[items[i].ID]; ok {
				items[i].PeerName = info.Phone
				// Use push_name as default title if no custom title was set
				if info.PushName != "" && items[i].Title == items[i].Preview {
					items[i].Title = info.PushName
				}
			}
		}
	}

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

	sess, err := h.findAndReadSession(dir, sessionID)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			http.Error(w, "session not found", http.StatusNotFound)
		} else {
			http.Error(w, "failed to parse session", http.StatusInternalServerError)
		}
		return
	}

	messages := visibleSessionMessages(sess.Messages)

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

	paths := findSessionFiles(dir, sessionID)

	removed := false
	for _, path := range paths {
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

// customTitlesPath returns the path to the custom session titles file.
func (h *Handler) customTitlesPath() (string, error) {
	dir, err := h.sessionsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, ".session-titles.json"), nil
}

// loadCustomTitles reads the custom session titles map from disk.
func (h *Handler) loadCustomTitles() (map[string]string, error) {
	path, err := h.customTitlesPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]string), nil
		}
		return nil, err
	}
	var titles map[string]string
	if err := json.Unmarshal(data, &titles); err != nil {
		return make(map[string]string), nil
	}
	return titles, nil
}

// saveCustomTitles writes the custom session titles map to disk.
func (h *Handler) saveCustomTitles(titles map[string]string) error {
	path, err := h.customTitlesPath()
	if err != nil {
		return err
	}
	data, err := json.Marshal(titles)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// handleRenameSession updates the custom title for a session.
//
//	PATCH /api/sessions/{id}
func (h *Handler) handleRenameSession(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")
	if sessionID == "" {
		http.Error(w, "missing session id", http.StatusBadRequest)
		return
	}

	var body struct {
		Title string `json:"title"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	title := strings.TrimSpace(body.Title)
	if title == "" {
		http.Error(w, "title must not be empty", http.StatusBadRequest)
		return
	}
	if len([]rune(title)) > maxSessionTitleRunes {
		runes := []rune(title)
		title = string(runes[:maxSessionTitleRunes])
	}

	titles, err := h.loadCustomTitles()
	if err != nil {
		http.Error(w, "failed to load titles", http.StatusInternalServerError)
		return
	}

	titles[sessionID] = title
	if err := h.saveCustomTitles(titles); err != nil {
		http.Error(w, "failed to save title", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"title": title})
}

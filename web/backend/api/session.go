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
	"github.com/sipeed/picoclaw/pkg/memory"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/providers/messageutil"
	"github.com/sipeed/picoclaw/pkg/session"
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
	Role        string                  `json:"role"`
	Content     string                  `json:"content"`
	Kind        string                  `json:"kind,omitempty"`
	CreatedAt   *time.Time              `json:"created_at,omitempty"`
	Media       []string                `json:"media,omitempty"`
	Attachments []sessionChatAttachment `json:"attachments,omitempty"`
	ToolCalls   []utils.VisibleToolCall `json:"tool_calls,omitempty"`
}

type sessionChatAttachment struct {
	Type        string `json:"type,omitempty"`
	URL         string `json:"url,omitempty"`
	Filename    string `json:"filename,omitempty"`
	ContentType string `json:"content_type,omitempty"`
}

// legacyPicoSessionPrefix is the legacy key prefix used by older Pico JSON/JSONL
// sessions before structured scope metadata existed.
const (
	legacyPicoSessionPrefix = "agent:main:pico:direct:pico:"
	picoSessionPrefix       = legacyPicoSessionPrefix

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

// extractLegacyPicoSessionID extracts the session UUID from an old Pico key.
// Returns the UUID and true if the key matches the Pico session pattern.
func extractLegacyPicoSessionID(key string) (string, bool) {
	if strings.HasPrefix(key, legacyPicoSessionPrefix) {
		return strings.TrimPrefix(key, legacyPicoSessionPrefix), true
	}
	return "", false
}

func sanitizeSessionKey(key string) string {
	key = strings.ReplaceAll(key, ":", "_")
	key = strings.ReplaceAll(key, "/", "_")
	key = strings.ReplaceAll(key, "\\", "_")
	return key
}

func (h *Handler) readLegacySession(path string) (sessionFile, error) {
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

func (h *Handler) readSessionMeta(path, sessionKey string) (memory.SessionMeta, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return memory.SessionMeta{Key: sessionKey}, nil
	}
	if err != nil {
		return memory.SessionMeta{}, err
	}

	var meta memory.SessionMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return memory.SessionMeta{}, err
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
		if messageutil.IsTransientAssistantThoughtMessage(msg) {
			continue
		}
		msgs = append(msgs, msg)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return msgs, nil
}

func (h *Handler) readJSONLSession(dir, sessionKey string) (sessionFile, error) {
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

type picoJSONLSessionRef struct {
	ID  string
	Key string
}

type picoLegacySessionRef struct {
	ID   string
	Path string
}

func extractPicoSessionIDFromScope(scope session.SessionScope) (string, bool) {
	if !strings.EqualFold(strings.TrimSpace(scope.Channel), "pico") {
		return "", false
	}

	candidates := []string{
		strings.TrimSpace(scope.Values["sender"]),
		strings.TrimSpace(scope.Values["chat"]),
	}
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if idx := strings.Index(candidate, "pico:"); idx >= 0 {
			sessionID := strings.TrimSpace(candidate[idx+len("pico:"):])
			if sessionID != "" {
				return sessionID, true
			}
		}
	}
	return "", false
}

func sessionRefFromMeta(meta memory.SessionMeta) (picoJSONLSessionRef, bool) {
	if len(meta.Scope) == 0 {
		if sessionID, ok := extractLegacyPicoSessionID(meta.Key); ok {
			return picoJSONLSessionRef{ID: sessionID, Key: meta.Key}, true
		}
		for _, alias := range meta.Aliases {
			if sessionID, ok := extractLegacyPicoSessionID(alias); ok {
				return picoJSONLSessionRef{ID: sessionID, Key: meta.Key}, true
			}
		}
		return picoJSONLSessionRef{}, false
	}
	var scope session.SessionScope
	if err := json.Unmarshal(meta.Scope, &scope); err != nil {
		return picoJSONLSessionRef{}, false
	}
	sessionID, ok := extractPicoSessionIDFromScope(scope)
	if !ok {
		if legacySessionID, ok := extractLegacyPicoSessionID(meta.Key); ok {
			return picoJSONLSessionRef{ID: legacySessionID, Key: meta.Key}, true
		}
		for _, alias := range meta.Aliases {
			if legacySessionID, ok := extractLegacyPicoSessionID(alias); ok {
				return picoJSONLSessionRef{ID: legacySessionID, Key: meta.Key}, true
			}
		}
		return picoJSONLSessionRef{}, false
	}
	return picoJSONLSessionRef{ID: sessionID, Key: meta.Key}, true
}

func (h *Handler) findPicoJSONLSessions(dir string) ([]picoJSONLSessionRef, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	refs := make([]picoJSONLSessionRef, 0)
	seen := make(map[string]struct{})
	metaBackedBases := make(map[string]struct{})
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".meta.json") {
			continue
		}
		name := entry.Name()
		metaPath := filepath.Join(dir, name)
		meta, err := h.readSessionMeta(metaPath, "")
		if err != nil {
			continue
		}
		ref, ok := sessionRefFromMeta(meta)
		if !ok || ref.Key == "" || ref.ID == "" {
			continue
		}
		metaBackedBases[strings.TrimSuffix(name, ".meta.json")] = struct{}{}
		if _, exists := seen[ref.ID]; exists {
			continue
		}
		seen[ref.ID] = struct{}{}
		refs = append(refs, ref)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jsonl") {
			continue
		}
		name := entry.Name()
		base := strings.TrimSuffix(name, ".jsonl")
		if _, ok := metaBackedBases[base]; ok {
			continue
		}
		ref, ok := jsonlSessionRefFromFilename(name)
		if !ok || ref.Key == "" || ref.ID == "" {
			continue
		}
		if _, exists := seen[ref.ID]; exists {
			continue
		}
		seen[ref.ID] = struct{}{}
		refs = append(refs, ref)
	}
	return refs, nil
}

func (h *Handler) findPicoJSONLSession(dir, sessionID string) (picoJSONLSessionRef, error) {
	refs, err := h.findPicoJSONLSessions(dir)
	if err != nil {
		return picoJSONLSessionRef{}, err
	}
	for _, ref := range refs {
		if ref.ID == sessionID {
			return ref, nil
		}
	}
	return picoJSONLSessionRef{}, os.ErrNotExist
}

func (h *Handler) findLegacyPicoSessions(dir string) ([]picoLegacySessionRef, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	refs := make([]picoLegacySessionRef, 0)
	seen := make(map[string]struct{})
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || filepath.Ext(name) != ".json" || strings.HasSuffix(name, ".meta.json") {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		sess, err := h.readLegacySession(path)
		if err != nil || isEmptySession(sess) {
			continue
		}

		sessionID, ok := extractLegacyPicoSessionID(sess.Key)
		if !ok || sessionID == "" {
			continue
		}
		if _, exists := seen[sessionID]; exists {
			continue
		}
		seen[sessionID] = struct{}{}
		refs = append(refs, picoLegacySessionRef{ID: sessionID, Path: path})
	}
	return refs, nil
}

func jsonlSessionRefFromFilename(name string) (picoJSONLSessionRef, bool) {
	if !strings.HasSuffix(name, ".jsonl") {
		return picoJSONLSessionRef{}, false
	}
	base := strings.TrimSuffix(name, ".jsonl")
	if base == "" {
		return picoJSONLSessionRef{}, false
	}

	legacyPrefix := sanitizeSessionKey(legacyPicoSessionPrefix)
	if strings.HasPrefix(base, legacyPrefix) {
		sessionID := strings.TrimPrefix(base, legacyPrefix)
		if sessionID == "" {
			return picoJSONLSessionRef{}, false
		}
		return picoJSONLSessionRef{
			ID:  sessionID,
			Key: legacyPicoSessionPrefix + sessionID,
		}, true
	}

	if session.IsOpaqueSessionKey(base) {
		return picoJSONLSessionRef{
			ID:  base,
			Key: base,
		}, true
	}

	return picoJSONLSessionRef{}, false
}

func (h *Handler) findLegacyPicoSession(dir, sessionID string) (picoLegacySessionRef, error) {
	refs, err := h.findLegacyPicoSessions(dir)
	if err != nil {
		return picoLegacySessionRef{}, err
	}
	for _, ref := range refs {
		if ref.ID == sessionID {
			return ref, nil
		}
	}
	return picoLegacySessionRef{}, os.ErrNotExist
}

func buildSessionListItem(sessionID string, sess sessionFile, toolFeedbackMaxArgsLength int) sessionListItem {
	transcript := visibleSessionMessages(sess.Messages, toolFeedbackMaxArgsLength)

	preview := ""
	for _, msg := range transcript {
		if msg.Role == "user" {
			preview = sessionChatMessagePreview(msg)
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

	return sessionListItem{
		ID:           sessionID,
		Title:        title,
		Preview:      preview,
		MessageCount: len(transcript),
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

func sessionChatMessageVisible(msg sessionChatMessage) bool {
	return strings.TrimSpace(msg.Content) != "" ||
		len(msg.Media) > 0 ||
		len(msg.Attachments) > 0 ||
		len(msg.ToolCalls) > 0
}

func sessionChatMessagePreview(msg sessionChatMessage) string {
	if content := strings.TrimSpace(msg.Content); content != "" {
		return content
	}
	if len(msg.Attachments) > 0 {
		if strings.EqualFold(strings.TrimSpace(msg.Attachments[0].Type), "image") {
			return "[image]"
		}
		return "[attachment]"
	}
	if len(msg.Media) > 0 {
		if strings.HasPrefix(strings.TrimSpace(msg.Media[0]), "data:image/") {
			return "[image]"
		}
		return "[attachment]"
	}
	if len(msg.ToolCalls) > 0 {
		return "[tool call]"
	}
	return ""
}

func visibleSessionMessages(messages []providers.Message, toolFeedbackMaxArgsLength int) []sessionChatMessage {
	return sessionTranscriptMessages(messages, toolFeedbackMaxArgsLength, false)
}

func detailSessionMessages(messages []providers.Message, toolFeedbackMaxArgsLength int) []sessionChatMessage {
	return sessionTranscriptMessages(messages, toolFeedbackMaxArgsLength, true)
}

func sessionTranscriptMessages(
	messages []providers.Message,
	toolFeedbackMaxArgsLength int,
	includeThoughts bool,
) []sessionChatMessage {
	transcript := make([]sessionChatMessage, 0, len(messages))

	for _, msg := range messages {
		attachments := sessionAttachments(msg)

		switch msg.Role {
		case "tool":
			continue

		case "user":
			chatMsg := sessionChatMessage{
				Role:        "user",
				Content:     msg.Content,
				CreatedAt:   msg.CreatedAt,
				Media:       append([]string(nil), msg.Media...),
				Attachments: attachments,
			}
			if sessionChatMessageVisible(chatMsg) {
				transcript = append(transcript, chatMsg)
			}

		case "assistant":
			if messageutil.IsTransientAssistantThoughtMessage(msg) {
				continue
			}
			if includeThoughts {
				if thoughtMsg, ok := assistantThoughtMessage(msg); ok {
					transcript = append(transcript, thoughtMsg)
				}
			}

			toolCallsMsg, hasToolCallsMsg := assistantToolCallsMessage(
				msg.ToolCalls,
				toolFeedbackMaxArgsLength,
				msg.CreatedAt,
			)
			visibleToolMessages := visibleAssistantToolMessages(msg.ToolCalls, msg.CreatedAt)

			// Pico web chat can persist both visible `message` tool output and a
			// later plain assistant reply in the same turn. Hide only the fixed
			// internal summary that marks handled tool delivery.
			content := msg.Content
			if assistantMessageInternalOnly(msg) {
				if len(attachments) == 0 {
					if hasToolCallsMsg {
						transcript = append(transcript, toolCallsMsg)
					}
					if len(visibleToolMessages) > 0 {
						transcript = append(transcript, visibleToolMessages...)
					}
					continue
				}
				content = ""
			}
			if hasToolCallsMsg && utils.ToolCallExplanationDuplicatesContent(content, msg.ToolCalls) {
				content = ""
			}

			chatMsg := sessionChatMessage{
				Role:        "assistant",
				Content:     content,
				CreatedAt:   msg.CreatedAt,
				Media:       append([]string(nil), msg.Media...),
				Attachments: attachments,
			}
			if !sessionChatMessageVisible(chatMsg) {
				if hasToolCallsMsg {
					transcript = append(transcript, toolCallsMsg)
				}
				if len(visibleToolMessages) > 0 {
					transcript = append(transcript, visibleToolMessages...)
				}
				continue
			}

			transcript = append(transcript, chatMsg)
			if hasToolCallsMsg {
				transcript = append(transcript, toolCallsMsg)
			}
			if len(visibleToolMessages) > 0 {
				transcript = append(transcript, visibleToolMessages...)
			}
		}
	}

	return filterSessionChatMessages(transcript)
}

func filterSessionChatMessages(messages []sessionChatMessage) []sessionChatMessage {
	filtered := messages[:0]
	for _, msg := range messages {
		if msg.Role != "user" && msg.Role != "assistant" {
			continue
		}
		filtered = append(filtered, msg)
	}
	return filtered
}

func sessionAttachments(msg providers.Message) []sessionChatAttachment {
	if len(msg.Attachments) == 0 {
		return nil
	}

	attachments := make([]sessionChatAttachment, 0, len(msg.Attachments))
	for _, attachment := range msg.Attachments {
		urlValue, ok := sessionAttachmentURL(attachment)
		if !ok {
			continue
		}
		attachmentType := strings.TrimSpace(attachment.Type)
		if attachmentType == "" {
			attachmentType = sessionAttachmentType(attachment)
		}
		attachments = append(attachments, sessionChatAttachment{
			Type:        attachmentType,
			URL:         urlValue,
			Filename:    strings.TrimSpace(attachment.Filename),
			ContentType: strings.TrimSpace(attachment.ContentType),
		})
	}

	if len(attachments) == 0 {
		return nil
	}
	return attachments
}

func sessionAttachmentURL(attachment providers.Attachment) (string, bool) {
	if rawURL := strings.TrimSpace(attachment.URL); rawURL != "" {
		return rawURL, true
	}

	ref := strings.TrimSpace(attachment.Ref)
	if ref == "" {
		return "", false
	}
	if strings.HasPrefix(ref, "media://") {
		// Persisted session history must only expose durable attachment locations.
		// media:// refs depend on the live in-memory MediaStore and may stop
		// resolving after a restart or cleanup, so omit them from reopened history.
		return "", false
	}
	return ref, true
}

func sessionAttachmentType(attachment providers.Attachment) string {
	contentType := strings.ToLower(strings.TrimSpace(attachment.ContentType))
	filename := strings.ToLower(strings.TrimSpace(attachment.Filename))
	rawRef := strings.ToLower(strings.TrimSpace(attachment.Ref))
	rawURL := strings.ToLower(strings.TrimSpace(attachment.URL))

	switch {
	case strings.HasPrefix(contentType, "image/"),
		strings.HasPrefix(rawRef, "data:image/"),
		strings.HasPrefix(rawURL, "data:image/"):
		return "image"
	case strings.HasPrefix(contentType, "audio/"):
		return "audio"
	case strings.HasPrefix(contentType, "video/"):
		return "video"
	}

	switch ext := filepath.Ext(filename); ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp", ".svg":
		return "image"
	case ".mp3", ".wav", ".ogg", ".m4a", ".flac", ".aac", ".wma", ".opus":
		return "audio"
	case ".mp4", ".avi", ".mov", ".webm", ".mkv":
		return "video"
	default:
		return "file"
	}
}

func assistantMessageInternalOnly(msg providers.Message) bool {
	return strings.TrimSpace(msg.Content) == handledToolResponseSummaryText
}

func assistantThoughtMessage(msg providers.Message) (sessionChatMessage, bool) {
	reasoning := strings.TrimSpace(msg.ReasoningContent)
	if reasoning == "" {
		return sessionChatMessage{}, false
	}
	if reasoning == strings.TrimSpace(msg.Content) {
		return sessionChatMessage{}, false
	}
	return sessionChatMessage{
		Role:      "assistant",
		Content:   reasoning,
		Kind:      "thought",
		CreatedAt: msg.CreatedAt,
	}, true
}

func assistantToolCallsMessage(
	toolCalls []providers.ToolCall,
	toolFeedbackMaxArgsLength int,
	createdAt *time.Time,
) (sessionChatMessage, bool) {
	if len(toolCalls) == 0 {
		return sessionChatMessage{}, false
	}
	if toolFeedbackMaxArgsLength <= 0 {
		toolFeedbackMaxArgsLength = defaultToolFeedbackMaxArgsLength()
	}

	visibleToolCalls := utils.BuildVisibleToolCalls(toolCalls, toolFeedbackMaxArgsLength)
	if len(visibleToolCalls) == 0 {
		return sessionChatMessage{}, false
	}

	return sessionChatMessage{
		Role:      "assistant",
		Kind:      "tool_calls",
		CreatedAt: createdAt,
		ToolCalls: visibleToolCalls,
	}, true
}

func visibleAssistantToolArgsPreview(
	tc providers.ToolCall,
	toolFeedbackMaxArgsLength int,
) string {
	return utils.VisibleToolCallArgumentsPreview(tc, toolFeedbackMaxArgsLength)
}

func visibleAssistantToolMessages(toolCalls []providers.ToolCall, createdAt *time.Time) []sessionChatMessage {
	if len(toolCalls) == 0 {
		return nil
	}

	messages := make([]sessionChatMessage, 0, len(toolCalls))
	for _, tc := range toolCalls {
		name, argsJSON := utils.VisibleToolCallNameAndArguments(tc)
		if name != "message" {
			continue
		}
		content, ok := parseMessageToolContent(argsJSON)
		if !ok {
			continue
		}
		messages = append(messages, sessionChatMessage{
			Role:      "assistant",
			Content:   content,
			CreatedAt: createdAt,
		})
	}

	return messages
}

func parseMessageToolContent(argsJSON string) (string, bool) {
	var args struct {
		Content string `json:"content"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", false
	}
	if strings.TrimSpace(args.Content) == "" {
		return "", false
	}
	return args.Content, true
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

	if _, err := os.ReadDir(dir); err != nil {
		// Directory doesn't exist yet = no sessions
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]sessionListItem{})
		return
	}

	items := []sessionListItem{}
	seen := make(map[string]struct{})

	if refs, findErr := h.findPicoJSONLSessions(dir); findErr == nil {
		for _, ref := range refs {
			sess, loadErr := h.readJSONLSession(dir, ref.Key)
			if loadErr != nil || isEmptySession(sess) {
				continue
			}
			seen[ref.ID] = struct{}{}
			items = append(items, buildSessionListItem(ref.ID, sess, toolFeedbackMaxArgsLength))
		}
	}

	if legacyRefs, findErr := h.findLegacyPicoSessions(dir); findErr == nil {
		for _, ref := range legacyRefs {
			if _, exists := seen[ref.ID]; exists {
				continue
			}
			sess, loadErr := h.readLegacySession(ref.Path)
			if loadErr != nil || isEmptySession(sess) {
				continue
			}
			seen[ref.ID] = struct{}{}
			items = append(items, buildSessionListItem(ref.ID, sess, toolFeedbackMaxArgsLength))
		}
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

	ref, refErr := h.findPicoJSONLSession(dir, sessionID)
	var sess sessionFile
	err = refErr
	if refErr == nil {
		sess, err = h.readJSONLSession(dir, ref.Key)
	}
	if err == nil && isEmptySession(sess) {
		err = os.ErrNotExist
	}
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if legacyRef, legacyErr := h.findLegacyPicoSession(dir, sessionID); legacyErr == nil {
				sess, err = h.readLegacySession(legacyRef.Path)
			}
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

	for i := range sess.Messages {
		if sess.Messages[i].CreatedAt == nil {
			sess.Messages[i].CreatedAt = &sess.Updated
		}
	}
	messages := detailSessionMessages(sess.Messages, toolFeedbackMaxArgsLength)

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

	removed := false
	if ref, err := h.findPicoJSONLSession(dir, sessionID); err == nil {
		base := filepath.Join(dir, sanitizeSessionKey(ref.Key))
		for _, path := range []string{base + ".jsonl", base + ".meta.json"} {
			if err := os.Remove(path); err != nil {
				if os.IsNotExist(err) {
					continue
				}
				http.Error(w, "failed to delete session", http.StatusInternalServerError)
				return
			}
			removed = true
		}
	}

	if legacyRef, err := h.findLegacyPicoSession(dir, sessionID); err == nil {
		if err := os.Remove(legacyRef.Path); err != nil {
			if !os.IsNotExist(err) {
				http.Error(w, "failed to delete session", http.StatusInternalServerError)
				return
			}
		} else {
			removed = true
		}
	}

	if !removed {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

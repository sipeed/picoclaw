package session

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/providers"
)

type Session struct {
	Key      string              `json:"key"`
	Messages []providers.Message `json:"messages"`
	Summary  string              `json:"summary,omitempty"`
	Created  time.Time           `json:"created"`
	Updated  time.Time           `json:"updated"`
}

type SessionManager struct {
	sessions      map[string]*Session
	mu            sync.RWMutex
	storage       string
	summarizer    Summarizer                 // optional; nil = no summarization
	summarizerCfg config.SummarizationConfig // filled via WithDefaults at construction
	inflight      sync.Map                   // sessionKey → true (dedup guard)
}

// Option configures a SessionManager at construction time.
type Option func(*SessionManager)

// WithSummarizer enables background summarization and emergency compression.
// The Summarizer is called to produce summaries via the LLM. Zero-valued
// config fields are replaced with defaults.
func WithSummarizer(s Summarizer, cfg config.SummarizationConfig) Option {
	return func(sm *SessionManager) {
		sm.summarizer = s
		sm.summarizerCfg = cfg.WithDefaults()
	}
}

func NewSessionManager(storage string, opts ...Option) *SessionManager {
	sm := &SessionManager{
		sessions: make(map[string]*Session),
		storage:  storage,
	}

	for _, o := range opts {
		o(sm)
	}

	if storage != "" {
		os.MkdirAll(storage, 0o755)
		sm.loadSessions()
	}

	return sm
}

func (sm *SessionManager) GetOrCreate(key string) *Session {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, ok := sm.sessions[key]
	if ok {
		return session
	}

	session = &Session{
		Key:      key,
		Messages: []providers.Message{},
		Created:  time.Now(),
		Updated:  time.Now(),
	}
	sm.sessions[key] = session

	return session
}

func (sm *SessionManager) AddMessage(sessionKey, role, content string) {
	sm.AddFullMessage(sessionKey, providers.Message{
		Role:    role,
		Content: content,
	})
}

// AddFullMessage adds a complete message with tool calls and tool call ID to the session.
// This is used to save the full conversation flow including tool calls and tool results.
func (sm *SessionManager) AddFullMessage(sessionKey string, msg providers.Message) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, ok := sm.sessions[sessionKey]
	if !ok {
		session = &Session{
			Key:      sessionKey,
			Messages: []providers.Message{},
			Created:  time.Now(),
		}
		sm.sessions[sessionKey] = session
	}

	session.Messages = append(session.Messages, msg)
	session.Updated = time.Now()
}

func (sm *SessionManager) GetHistory(key string) []providers.Message {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	session, ok := sm.sessions[key]
	if !ok {
		return []providers.Message{}
	}

	history := make([]providers.Message, len(session.Messages))
	copy(history, session.Messages)
	return history
}

func (sm *SessionManager) GetSummary(key string) string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	session, ok := sm.sessions[key]
	if !ok {
		return ""
	}
	return session.Summary
}

func (sm *SessionManager) SetSummary(key string, summary string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, ok := sm.sessions[key]
	if ok {
		session.Summary = summary
		session.Updated = time.Now()
	}
}

func (sm *SessionManager) TruncateHistory(key string, keepLast int) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, ok := sm.sessions[key]
	if !ok {
		return
	}

	if keepLast <= 0 {
		session.Messages = []providers.Message{}
		session.Updated = time.Now()
		return
	}

	if len(session.Messages) <= keepLast {
		return
	}

	session.Messages = session.Messages[len(session.Messages)-keepLast:]
	session.Updated = time.Now()
}

// sanitizeFilename converts a session key into a cross-platform safe filename.
// Session keys use "channel:chatID" (e.g. "telegram:123456") but ':' is the
// volume separator on Windows, so filepath.Base would misinterpret the key.
// We replace it with '_'. The original key is preserved inside the JSON file,
// so loadSessions still maps back to the right in-memory key.
func sanitizeFilename(key string) string {
	return strings.ReplaceAll(key, ":", "_")
}

func (sm *SessionManager) Save(key string) error {
	if sm.storage == "" {
		return nil
	}

	filename := sanitizeFilename(key)

	// filepath.IsLocal rejects empty names, "..", absolute paths, and
	// OS-reserved device names (NUL, COM1 … on Windows).
	// The extra checks reject "." and any directory separators so that
	// the session file is always written directly inside sm.storage.
	if filename == "." || !filepath.IsLocal(filename) || strings.ContainsAny(filename, `/\`) {
		return os.ErrInvalid
	}

	// Snapshot under read lock, then perform slow file I/O after unlock.
	sm.mu.RLock()
	stored, ok := sm.sessions[key]
	if !ok {
		sm.mu.RUnlock()
		return nil
	}

	snapshot := Session{
		Key:     stored.Key,
		Summary: stored.Summary,
		Created: stored.Created,
		Updated: stored.Updated,
	}
	if len(stored.Messages) > 0 {
		snapshot.Messages = make([]providers.Message, len(stored.Messages))
		copy(snapshot.Messages, stored.Messages)
	} else {
		snapshot.Messages = []providers.Message{}
	}
	sm.mu.RUnlock()

	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}

	sessionPath := filepath.Join(sm.storage, filename+".json")
	tmpFile, err := os.CreateTemp(sm.storage, "session-*.tmp")
	if err != nil {
		return err
	}

	tmpPath := tmpFile.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err := tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()
		return err
	}
	if err := tmpFile.Chmod(0o644); err != nil {
		_ = tmpFile.Close()
		return err
	}
	if err := tmpFile.Sync(); err != nil {
		_ = tmpFile.Close()
		return err
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}

	if err := os.Rename(tmpPath, sessionPath); err != nil {
		return err
	}
	cleanup = false
	return nil
}

func (sm *SessionManager) loadSessions() error {
	files, err := os.ReadDir(sm.storage)
	if err != nil {
		return err
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		if filepath.Ext(file.Name()) != ".json" {
			continue
		}

		sessionPath := filepath.Join(sm.storage, file.Name())
		data, err := os.ReadFile(sessionPath)
		if err != nil {
			continue
		}

		var session Session
		if err := json.Unmarshal(data, &session); err != nil {
			continue
		}

		sm.sessions[session.Key] = &session
	}

	return nil
}

// SetHistory updates the messages of a session.
func (sm *SessionManager) SetHistory(key string, history []providers.Message) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, ok := sm.sessions[key]
	if ok {
		// Create a deep copy to strictly isolate internal state
		// from the caller's slice.
		msgs := make([]providers.Message, len(history))
		copy(msgs, history)
		session.Messages = msgs
		session.Updated = time.Now()
	}
}

// ApplySummarization atomically sets a summary and trims old messages while
// preserving any messages that were appended after the caller's snapshot.
//
// snapshotLen is the len(session.Messages) at the time the caller took its
// snapshot. If the session has grown since then, the new tail is preserved.
// If messages were removed (e.g. another compression), the operation is
// skipped and returns false.
//
// keepLast is the number of messages from the *snapshot* to retain (counted
// from the end of the snapshot window, not the current session).
func (sm *SessionManager) ApplySummarization(key, summary string, snapshotLen, keepLast int) bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, ok := sm.sessions[key]
	if !ok {
		return false
	}

	currentLen := len(session.Messages)

	// Stale: session was truncated/replaced since our snapshot.
	if currentLen < snapshotLen {
		return false
	}

	// Messages appended by the main loop after the snapshot was taken.
	newTail := session.Messages[snapshotLen:]

	// Messages to keep from the original snapshot window.
	var keptFromSnapshot []providers.Message
	if keepLast > 0 && keepLast < snapshotLen {
		keptFromSnapshot = session.Messages[snapshotLen-keepLast : snapshotLen]
	} else if keepLast >= snapshotLen {
		keptFromSnapshot = session.Messages[:snapshotLen]
	}

	merged := make([]providers.Message, 0, len(keptFromSnapshot)+len(newTail))
	merged = append(merged, keptFromSnapshot...)
	merged = append(merged, newTail...)

	session.Messages = merged
	session.Summary = summary
	session.Updated = time.Now()
	return true
}

// EstimateTokens estimates token count for a list of messages using the
// configured chars-per-token ratio and unicode rune counting (CJK-aware).
func (sm *SessionManager) EstimateTokens(messages []providers.Message) int {
	totalRunes := 0
	for _, m := range messages {
		totalRunes += utf8.RuneCountInString(m.Content)
	}
	return int(float64(totalRunes) / sm.summarizerCfg.CharsPerToken)
}

// MaybeSummarize checks whether the session's history exceeds the configured
// thresholds and, if so, triggers background summarization. At most one
// summarization runs per session key at a time.
// No-op if no Summarizer was provided via WithSummarizer.
func (sm *SessionManager) MaybeSummarize(sessionKey string) {
	if sm.summarizer == nil {
		return
	}

	history := sm.GetHistory(sessionKey)
	tokenEstimate := sm.EstimateTokens(history)
	cfg := sm.summarizerCfg
	threshold := cfg.ContextWindow * cfg.TokenPercent / 100

	if len(history) <= cfg.MessageThreshold && tokenEstimate <= threshold {
		return
	}

	if _, loaded := sm.inflight.LoadOrStore(sessionKey, true); loaded {
		return // already running for this session
	}

	go func() {
		defer sm.inflight.Delete(sessionKey)
		logger.Debug("Memory threshold reached. Optimizing conversation history...")
		sm.summarizeSession(sessionKey)
	}()
}

// summarizeSession performs the actual summarization: snapshot history,
// generate summary via LLM, and atomically apply the result while
// preserving any messages added by the main loop during summarization.
func (sm *SessionManager) summarizeSession(sessionKey string) {
	cfg := sm.summarizerCfg
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	history := sm.GetHistory(sessionKey)
	existingSummary := sm.GetSummary(sessionKey)
	snapshotLen := len(history)

	keepLast := cfg.KeepLastMessages
	if snapshotLen <= keepLast {
		return
	}

	toSummarize := history[:snapshotLen-keepLast]

	// Oversized message guard: skip individual messages that exceed
	// MaxSingleMsgTokenRatio of the context window.
	maxMessageTokens := int(float64(cfg.ContextWindow) * cfg.MaxSingleMsgTokenRatio)
	var validMessages []providers.Message
	omitted := false

	for _, m := range toSummarize {
		if m.Role != "user" && m.Role != "assistant" {
			continue
		}
		msgTokens := int(float64(utf8.RuneCountInString(m.Content)) / cfg.CharsPerToken)
		if msgTokens > maxMessageTokens {
			omitted = true
			continue
		}
		validMessages = append(validMessages, m)
	}

	if len(validMessages) == 0 {
		return
	}

	// Produce summary, splitting into two batches for large conversations.
	var finalSummary string
	if len(validMessages) > cfg.MultiPartBatchThreshold {
		mid := len(validMessages) / 2
		part1 := validMessages[:mid]
		part2 := validMessages[mid:]

		var s1, s2 string
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			s1, _ = sm.summarizer.Summarize(ctx, part1, "")
		}()
		go func() {
			defer wg.Done()
			s2, _ = sm.summarizer.Summarize(ctx, part2, "")
		}()
		wg.Wait()

		// Merge the two partial summaries.
		mergeMessages := []providers.Message{
			{
				Role: "user",
				Content: fmt.Sprintf(
					"Merge these two conversation summaries into one cohesive summary:\n\n1: %s\n\n2: %s",
					s1,
					s2,
				),
			},
		}
		merged, err := sm.summarizer.Summarize(ctx, mergeMessages, "")
		if err == nil {
			finalSummary = merged
		} else {
			finalSummary = s1 + " " + s2
		}
	} else {
		var err error
		finalSummary, err = sm.summarizer.Summarize(ctx, validMessages, existingSummary)
		if err != nil {
			logger.WarnCF("session", "Summarization failed", map[string]any{"error": err.Error()})
			return
		}
	}

	if omitted && finalSummary != "" {
		finalSummary += "\n[Note: Some oversized messages were omitted from this summary for efficiency.]"
	}

	if finalSummary == "" {
		return
	}

	// Atomically apply: sets summary and trims old messages while preserving
	// anything the main loop appended after our snapshot.
	applied := sm.ApplySummarization(sessionKey, finalSummary, snapshotLen, keepLast)
	if applied {
		sm.Save(sessionKey)
		logger.InfoCF("session", "Summarization applied", map[string]any{
			"session_key":  sessionKey,
			"snapshot_len": snapshotLen,
			"keep_last":    keepLast,
		})
	} else {
		logger.WarnCF("session", "Summarization skipped (stale snapshot)", map[string]any{
			"session_key":  sessionKey,
			"snapshot_len": snapshotLen,
		})
	}
}

// ForceCompression aggressively reduces context when the LLM returns a
// context-window error. It drops the oldest ~50% of conversation messages,
// preserving the system prompt (first message) and the last message.
//
// This is called synchronously from the LLM retry loop — the main loop is
// already blocked, so there is no concurrent-append concern here.
// No-op if no Summarizer was provided via WithSummarizer.
func (sm *SessionManager) ForceCompression(sessionKey string) {
	cfg := sm.summarizerCfg.WithDefaults()
	history := sm.GetHistory(sessionKey)
	if len(history) <= cfg.ForceCompressionMinMessages {
		return
	}

	// history[0] is the system prompt, history[len-1] is the trigger message.
	conversation := history[1 : len(history)-1]
	if len(conversation) == 0 {
		return
	}

	mid := len(conversation) / 2
	droppedCount := mid
	keptConversation := conversation[mid:]

	newHistory := make([]providers.Message, 0, 1+len(keptConversation)+1)

	// Append compression note to the system prompt to avoid consecutive
	// system messages (rejected by some APIs like Zhipu).
	compressionNote := fmt.Sprintf(
		"\n\n[System Note: Emergency compression dropped %d oldest messages due to context limit]",
		droppedCount,
	)
	enhancedSystemPrompt := history[0]
	enhancedSystemPrompt.Content += compressionNote
	newHistory = append(newHistory, enhancedSystemPrompt)
	newHistory = append(newHistory, keptConversation...)
	newHistory = append(newHistory, history[len(history)-1])

	sm.SetHistory(sessionKey, newHistory)
	sm.Save(sessionKey)

	logger.WarnCF("session", "Forced compression executed", map[string]any{
		"session_key":  sessionKey,
		"dropped_msgs": droppedCount,
		"new_count":    len(newHistory),
	})
}

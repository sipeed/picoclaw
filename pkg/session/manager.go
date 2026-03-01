package session

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/providers"
)

type Session struct {
	Key      string              `json:"key"`
	Messages []providers.Message `json:"messages"`
	Summary  string              `json:"summary,omitempty"`
	Created  time.Time           `json:"created"`
	Updated  time.Time           `json:"updated"`
}

const sessionIndexFilename = "index.json"

var (
	removeFile              = os.Remove
	warningWriter io.Writer = os.Stderr
)

type scopeIndex struct {
	ActiveSessionKey string    `json:"active_session_key"`
	OrderedSessions  []string  `json:"ordered_sessions"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type sessionIndex struct {
	Version        int                    `json:"version"`
	Scopes         map[string]*scopeIndex `json:"scopes"`
	PendingDeletes []string               `json:"pending_deletes,omitempty"`
}

type SessionMeta struct {
	Ordinal    int       `json:"ordinal"`
	SessionKey string    `json:"session_key"`
	UpdatedAt  time.Time `json:"updated_at"`
	MessageCnt int       `json:"message_cnt"`
	Active     bool      `json:"active"`
}

type SessionManager struct {
	sessions  map[string]*Session
	mu        sync.RWMutex
	storage   string
	index     sessionIndex
	indexPath string
}

func NewSessionManager(storage string) *SessionManager {
	sm := &SessionManager{
		sessions: make(map[string]*Session),
		storage:  storage,
		index: sessionIndex{
			Version: 1,
			Scopes:  make(map[string]*scopeIndex),
		},
	}

	if storage != "" {
		os.MkdirAll(storage, 0o755)
		sm.indexPath = filepath.Join(storage, sessionIndexFilename)
		sm.loadSessions()
		sm.loadIndex()
	}

	return sm
}

func (sm *SessionManager) ResolveActive(scopeKey string) (string, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := time.Now()
	scope, changed := sm.ensureScopeLocked(scopeKey, now)
	if changed {
		if err := sm.saveIndexLocked(); err != nil {
			return "", err
		}
	}
	return scope.ActiveSessionKey, nil
}

func (sm *SessionManager) StartNew(scopeKey string) (string, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := time.Now()
	prevScopeEntryExists := false
	prevScopeEntryWasNil := false
	var prevScopeSnapshot *scopeIndex
	if sm.index.Scopes != nil {
		if existingScope, ok := sm.index.Scopes[scopeKey]; ok {
			prevScopeEntryExists = true
			if existingScope == nil {
				prevScopeEntryWasNil = true
			} else {
				prevScopeSnapshot = cloneScopeIndex(existingScope)
			}
		}
	}

	orderedForOrdinal := []string{scopeKey}
	if prevScopeSnapshot != nil && len(prevScopeSnapshot.OrderedSessions) > 0 {
		orderedForOrdinal = prevScopeSnapshot.OrderedSessions
	}

	newOrdinal := 2
	for _, existing := range orderedForOrdinal {
		ordinal, ok := sessionOrdinal(scopeKey, existing)
		if !ok {
			continue
		}
		if ordinal >= newOrdinal {
			newOrdinal = ordinal + 1
		}
	}

	newSessionKey := scopeKey + "#" + strconv.Itoa(newOrdinal)

	created := false
	if _, ok := sm.sessions[newSessionKey]; !ok {
		sm.sessions[newSessionKey] = &Session{
			Key:      newSessionKey,
			Messages: []providers.Message{},
			Created:  now,
			Updated:  now,
		}
		created = true
	}
	if err := sm.saveSessionLocked(newSessionKey); err != nil {
		if created {
			delete(sm.sessions, newSessionKey)
		}
		return "", err
	}

	scope, _ := sm.ensureScopeLocked(scopeKey, now)
	scope.ActiveSessionKey = newSessionKey
	scope.OrderedSessions = prependSessionUnique(scope.OrderedSessions, newSessionKey)
	scope.UpdatedAt = now
	if err := sm.saveIndexLocked(); err != nil {
		if prevScopeEntryExists {
			if sm.index.Scopes == nil {
				sm.index.Scopes = make(map[string]*scopeIndex)
			}
			if prevScopeEntryWasNil {
				sm.index.Scopes[scopeKey] = nil
			} else {
				sm.index.Scopes[scopeKey] = cloneScopeIndex(prevScopeSnapshot)
			}
		} else if sm.index.Scopes != nil {
			delete(sm.index.Scopes, scopeKey)
		}
		if created {
			delete(sm.sessions, newSessionKey)
			_ = sm.deleteSessionFile(newSessionKey)
		}
		return "", err
	}
	return newSessionKey, nil
}

func (sm *SessionManager) List(scopeKey string) ([]SessionMeta, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := time.Now()
	scope, changed := sm.ensureScopeLocked(scopeKey, now)
	if changed {
		if err := sm.saveIndexLocked(); err != nil {
			return nil, err
		}
	}

	list := make([]SessionMeta, 0, len(scope.OrderedSessions))
	for i, key := range scope.OrderedSessions {
		meta := SessionMeta{
			Ordinal:    i + 1,
			SessionKey: key,
			Active:     key == scope.ActiveSessionKey,
		}

		if session, ok := sm.sessions[key]; ok {
			meta.UpdatedAt = session.Updated
			meta.MessageCnt = len(session.Messages)
		}
		list = append(list, meta)
	}
	return list, nil
}

func (sm *SessionManager) Resume(scopeKey string, index int) (string, error) {
	if index < 1 {
		return "", fmt.Errorf("session index must be >= 1")
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := time.Now()
	scope, changed := sm.ensureScopeLocked(scopeKey, now)
	if changed {
		if err := sm.saveIndexLocked(); err != nil {
			return "", err
		}
	}

	if index > len(scope.OrderedSessions) {
		return "", fmt.Errorf("session index %d out of range", index)
	}

	scope.ActiveSessionKey = scope.OrderedSessions[index-1]
	scope.UpdatedAt = now
	if err := sm.saveIndexLocked(); err != nil {
		return "", err
	}
	return scope.ActiveSessionKey, nil
}

func (sm *SessionManager) DeleteSession(sessionKey string) error {
	sm.mu.Lock()
	changed := false
	delete(sm.sessions, sessionKey)

	now := time.Now()
	for scopeKey, scope := range sm.index.Scopes {
		if scope == nil {
			delete(sm.index.Scopes, scopeKey)
			changed = true
			continue
		}

		filtered := scope.OrderedSessions[:0]
		removed := false
		for _, key := range scope.OrderedSessions {
			if key == sessionKey {
				removed = true
				changed = true
				continue
			}
			filtered = append(filtered, key)
		}
		scope.OrderedSessions = filtered
		if !removed {
			continue
		}

		if scope.ActiveSessionKey == sessionKey {
			if len(scope.OrderedSessions) > 0 {
				scope.ActiveSessionKey = scope.OrderedSessions[0]
			} else {
				scope.ActiveSessionKey = ""
			}
		}

		if len(scope.OrderedSessions) == 0 {
			delete(sm.index.Scopes, scopeKey)
			continue
		}
		scope.UpdatedAt = now
	}

	if changed {
		if err := sm.saveIndexLocked(); err != nil {
			sm.mu.Unlock()
			return err
		}
	}
	sm.mu.Unlock()

	if err := sm.deleteSessionFile(sessionKey); err != nil {
		sm.warnf("failed to delete session file for %q, deferred retry on startup: %v", sessionKey, err)
		sm.mu.Lock()
		pendingChanged := sm.addPendingDeleteLocked(sessionKey)
		if pendingChanged {
			if err := sm.saveIndexLocked(); err != nil {
				sm.mu.Unlock()
				sm.warnf("failed to persist deferred delete for %q: %v", sessionKey, err)
				return nil
			}
		}
		sm.mu.Unlock()
		return nil
	}

	sm.mu.Lock()
	pendingChanged := sm.removePendingDeleteLocked(sessionKey)
	if pendingChanged {
		if err := sm.saveIndexLocked(); err != nil {
			sm.mu.Unlock()
			sm.warnf("failed to persist cleanup of deferred delete for %q: %v", sessionKey, err)
			return nil
		}
	}
	sm.mu.Unlock()

	return nil
}

func (sm *SessionManager) Prune(scopeKey string, limit int) ([]string, error) {
	if limit < 1 {
		return nil, fmt.Errorf("limit must be >= 1")
	}

	sm.mu.Lock()
	now := time.Now()
	scope, changed := sm.ensureScopeLocked(scopeKey, now)
	if changed {
		if err := sm.saveIndexLocked(); err != nil {
			sm.mu.Unlock()
			return nil, err
		}
	}

	if len(scope.OrderedSessions) <= limit {
		sm.mu.Unlock()
		return []string{}, nil
	}

	candidates := append([]string(nil), scope.OrderedSessions[limit:]...)
	sm.mu.Unlock()

	pruned := make([]string, 0, len(candidates))
	for _, sessionKey := range candidates {
		if err := sm.DeleteSession(sessionKey); err != nil {
			return pruned, err
		}
		pruned = append(pruned, sessionKey)
	}
	return pruned, nil
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

	// Snapshot under read lock, then perform slow file I/O after unlock.
	sm.mu.RLock()
	stored, ok := sm.sessions[key]
	if !ok {
		sm.mu.RUnlock()
		return nil
	}

	snapshot := cloneSession(stored)
	sm.mu.RUnlock()

	return sm.writeSessionSnapshot(snapshot)
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
		if file.Name() == sessionIndexFilename {
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
		if session.Key == "" {
			continue
		}

		sm.sessions[session.Key] = &session
	}

	return nil
}

func (sm *SessionManager) loadIndex() error {
	if sm.storage == "" {
		return nil
	}

	data, err := os.ReadFile(sm.indexPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var loaded sessionIndex
	if err := json.Unmarshal(data, &loaded); err != nil {
		return err
	}
	if loaded.Version == 0 {
		loaded.Version = 1
	}
	if loaded.Scopes == nil {
		loaded.Scopes = make(map[string]*scopeIndex)
	}

	changed := false
	seenPending := make(map[string]struct{}, len(loaded.PendingDeletes))
	retryPending := make([]string, 0, len(loaded.PendingDeletes))
	for _, sessionKey := range loaded.PendingDeletes {
		if sessionKey == "" {
			changed = true
			continue
		}
		if _, dup := seenPending[sessionKey]; dup {
			changed = true
			continue
		}
		seenPending[sessionKey] = struct{}{}

		// Deferred-delete sessions should not be visible even if stale files remain.
		delete(sm.sessions, sessionKey)

		if err := sm.deleteSessionFile(sessionKey); err != nil {
			// Invalid paths are unrecoverable; drop them from retry queue.
			if errors.Is(err, os.ErrInvalid) {
				changed = true
				sm.warnf("dropping invalid deferred delete key %q: %v", sessionKey, err)
				continue
			}
			sm.warnf("retry deferred session delete failed for %q: %v", sessionKey, err)
			retryPending = append(retryPending, sessionKey)
			continue
		}
		changed = true
	}
	if len(retryPending) != len(loaded.PendingDeletes) {
		changed = true
	}
	loaded.PendingDeletes = retryPending

	for scopeKey, scope := range loaded.Scopes {
		if scope == nil {
			delete(loaded.Scopes, scopeKey)
			changed = true
			continue
		}

		filtered := make([]string, 0, len(scope.OrderedSessions))
		seen := make(map[string]struct{}, len(scope.OrderedSessions))
		for _, sessionKey := range scope.OrderedSessions {
			if sessionKey == "" {
				changed = true
				continue
			}
			if _, exists := sm.sessions[sessionKey]; !exists {
				changed = true
				continue
			}
			if _, dup := seen[sessionKey]; dup {
				changed = true
				continue
			}
			seen[sessionKey] = struct{}{}
			filtered = append(filtered, sessionKey)
		}

		if len(filtered) == 0 {
			delete(loaded.Scopes, scopeKey)
			changed = true
			continue
		}

		if len(filtered) != len(scope.OrderedSessions) {
			changed = true
		}
		scope.OrderedSessions = filtered
		if _, ok := seen[scope.ActiveSessionKey]; !ok {
			scope.ActiveSessionKey = scope.OrderedSessions[0]
			changed = true
		}
	}

	sm.index = loaded
	if changed {
		return sm.saveIndexLocked()
	}
	return nil
}

func (sm *SessionManager) saveIndexLocked() error {
	if sm.storage == "" {
		return nil
	}
	if sm.index.Scopes == nil {
		sm.index.Scopes = make(map[string]*scopeIndex)
	}
	sm.index.Version = 1

	data, err := json.MarshalIndent(sm.index, "", "  ")
	if err != nil {
		return err
	}

	tmpFile, err := os.CreateTemp(sm.storage, "index-*.tmp")
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
	if err := os.Rename(tmpPath, sm.indexPath); err != nil {
		return err
	}
	cleanup = false
	return nil
}

func (sm *SessionManager) ensureScopeLocked(scopeKey string, now time.Time) (*scopeIndex, bool) {
	if sm.index.Scopes == nil {
		sm.index.Scopes = make(map[string]*scopeIndex)
	}

	scope, ok := sm.index.Scopes[scopeKey]
	if !ok || scope == nil {
		scope = &scopeIndex{
			ActiveSessionKey: scopeKey,
			OrderedSessions:  []string{scopeKey},
			UpdatedAt:        now,
		}
		sm.index.Scopes[scopeKey] = scope
		return scope, true
	}

	changed := false
	if len(scope.OrderedSessions) == 0 {
		scope.OrderedSessions = []string{scopeKey}
		changed = true
	}
	if scope.ActiveSessionKey == "" {
		scope.ActiveSessionKey = scope.OrderedSessions[0]
		changed = true
	}
	if changed {
		scope.UpdatedAt = now
	}
	return scope, changed
}

func prependSessionUnique(ordered []string, sessionKey string) []string {
	next := make([]string, 0, len(ordered)+1)
	next = append(next, sessionKey)
	for _, existing := range ordered {
		if existing == sessionKey {
			continue
		}
		next = append(next, existing)
	}
	return next
}

func cloneScopeIndex(scope *scopeIndex) *scopeIndex {
	if scope == nil {
		return nil
	}
	cloned := &scopeIndex{
		ActiveSessionKey: scope.ActiveSessionKey,
		UpdatedAt:        scope.UpdatedAt,
	}
	cloned.OrderedSessions = append([]string(nil), scope.OrderedSessions...)
	return cloned
}

func cloneSession(stored *Session) Session {
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
	return snapshot
}

func (sm *SessionManager) warnf(format string, args ...any) {
	if warningWriter == nil {
		return
	}
	_, _ = fmt.Fprintf(warningWriter, "warning: "+format+"\n", args...)
}

func (sm *SessionManager) addPendingDeleteLocked(sessionKey string) bool {
	if sessionKey == "" {
		return false
	}
	for _, existing := range sm.index.PendingDeletes {
		if existing == sessionKey {
			return false
		}
	}
	sm.index.PendingDeletes = append(sm.index.PendingDeletes, sessionKey)
	return true
}

func (sm *SessionManager) removePendingDeleteLocked(sessionKey string) bool {
	if len(sm.index.PendingDeletes) == 0 {
		return false
	}
	filtered := sm.index.PendingDeletes[:0]
	removed := false
	for _, existing := range sm.index.PendingDeletes {
		if existing == sessionKey {
			removed = true
			continue
		}
		filtered = append(filtered, existing)
	}
	sm.index.PendingDeletes = filtered
	return removed
}

func (sm *SessionManager) saveSessionLocked(key string) error {
	if sm.storage == "" {
		return nil
	}

	stored, ok := sm.sessions[key]
	if !ok {
		return fmt.Errorf("session %q not found", key)
	}

	return sm.writeSessionSnapshot(cloneSession(stored))
}

func (sm *SessionManager) writeSessionSnapshot(snapshot Session) error {
	if sm.storage == "" {
		return nil
	}

	filename := sanitizeFilename(snapshot.Key)

	// filepath.IsLocal rejects empty names, "..", absolute paths, and
	// OS-reserved device names (NUL, COM1 … on Windows).
	// The extra checks reject "." and any directory separators so that
	// the session file is always written directly inside sm.storage.
	if filename == "." || !filepath.IsLocal(filename) || strings.ContainsAny(filename, `/\`) {
		return os.ErrInvalid
	}

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

func (sm *SessionManager) deleteSessionFile(sessionKey string) error {
	if sm.storage == "" {
		return nil
	}

	filename := sanitizeFilename(sessionKey)
	if filename == "." || !filepath.IsLocal(filename) || strings.ContainsAny(filename, `/\`) {
		return os.ErrInvalid
	}

	sessionPath := filepath.Join(sm.storage, filename+".json")
	if err := removeFile(sessionPath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return nil
}

func sessionOrdinal(scopeKey, sessionKey string) (int, bool) {
	if sessionKey == scopeKey {
		return 1, true
	}

	prefix := scopeKey + "#"
	if !strings.HasPrefix(sessionKey, prefix) {
		return 0, false
	}

	n, err := strconv.Atoi(strings.TrimPrefix(sessionKey, prefix))
	if err != nil || n < 2 {
		return 0, false
	}

	return n, true
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

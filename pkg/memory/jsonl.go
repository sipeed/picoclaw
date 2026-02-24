package memory

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/providers"
)

// sessionMeta holds per-session metadata stored in a .meta.json file.
type sessionMeta struct {
	Key       string    `json:"key"`
	Summary   string    `json:"summary"`
	Skip      int       `json:"skip"`
	Count     int       `json:"count"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// JSONLStore implements Store using append-only JSONL files.
//
// Each session is stored as two files:
//
//	{sanitized_key}.jsonl      — one JSON-encoded message per line, append-only
//	{sanitized_key}.meta.json  — session metadata (summary, logical truncation offset)
//
// Messages are never physically deleted from the JSONL file. Instead,
// TruncateHistory records a "skip" offset in the metadata file and
// GetHistory ignores lines before that offset. This keeps all writes
// append-only, which is both fast and crash-safe.
type JSONLStore struct {
	dir string

	mu    sync.Mutex
	locks map[string]*sync.Mutex
}

// NewJSONLStore creates a new JSONL-backed store rooted at dir.
func NewJSONLStore(dir string) (*JSONLStore, error) {
	err := os.MkdirAll(dir, 0o755)
	if err != nil {
		return nil, fmt.Errorf("memory: create directory: %w", err)
	}
	return &JSONLStore{
		dir:   dir,
		locks: make(map[string]*sync.Mutex),
	}, nil
}

// sessionLock returns (or creates) a per-session mutex.
func (s *JSONLStore) sessionLock(key string) *sync.Mutex {
	s.mu.Lock()
	defer s.mu.Unlock()

	l, ok := s.locks[key]
	if !ok {
		l = &sync.Mutex{}
		s.locks[key] = l
	}
	return l
}

func (s *JSONLStore) jsonlPath(key string) string {
	return filepath.Join(s.dir, sanitizeKey(key)+".jsonl")
}

func (s *JSONLStore) metaPath(key string) string {
	return filepath.Join(s.dir, sanitizeKey(key)+".meta.json")
}

// sanitizeKey converts a session key to a safe filename component.
// Mirrors pkg/session.sanitizeFilename so that migration paths match.
func sanitizeKey(key string) string {
	return strings.ReplaceAll(key, ":", "_")
}

// readMeta loads the metadata file for a session.
// Returns a zero-value sessionMeta if the file does not exist.
func (s *JSONLStore) readMeta(key string) (sessionMeta, error) {
	data, err := os.ReadFile(s.metaPath(key))
	if os.IsNotExist(err) {
		return sessionMeta{Key: key}, nil
	}
	if err != nil {
		return sessionMeta{}, fmt.Errorf("memory: read meta: %w", err)
	}
	var meta sessionMeta
	err = json.Unmarshal(data, &meta)
	if err != nil {
		return sessionMeta{}, fmt.Errorf("memory: decode meta: %w", err)
	}
	return meta, nil
}

// writeMeta atomically writes the metadata file (temp + rename).
func (s *JSONLStore) writeMeta(key string, meta sessionMeta) error {
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("memory: encode meta: %w", err)
	}

	target := s.metaPath(key)
	tmp := target + ".tmp"

	err = os.WriteFile(tmp, data, 0o644)
	if err != nil {
		return fmt.Errorf("memory: write meta tmp: %w", err)
	}

	err = os.Rename(tmp, target)
	if err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("memory: rename meta: %w", err)
	}
	return nil
}

// readMessages reads all valid JSON lines from a .jsonl file.
// Malformed trailing lines (e.g. from a crash) are silently skipped.
func readMessages(path string) ([]providers.Message, error) {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return []providers.Message{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("memory: open jsonl: %w", err)
	}
	defer f.Close()

	var msgs []providers.Message
	scanner := bufio.NewScanner(f)
	// Allow up to 1 MB per line for messages with large content.
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var msg providers.Message
		if json.Unmarshal(line, &msg) != nil {
			// Corrupt line — likely a partial write from a crash.
			// Skip it; this is the standard JSONL recovery pattern.
			continue
		}
		msgs = append(msgs, msg)
	}
	if scanner.Err() != nil {
		return nil, fmt.Errorf("memory: scan jsonl: %w", scanner.Err())
	}

	if msgs == nil {
		msgs = []providers.Message{}
	}
	return msgs, nil
}

func (s *JSONLStore) AddMessage(
	_ context.Context, sessionKey, role, content string,
) error {
	return s.addMsg(sessionKey, providers.Message{
		Role:    role,
		Content: content,
	})
}

func (s *JSONLStore) AddFullMessage(
	_ context.Context, sessionKey string, msg providers.Message,
) error {
	return s.addMsg(sessionKey, msg)
}

// addMsg is the shared implementation for AddMessage and AddFullMessage.
func (s *JSONLStore) addMsg(sessionKey string, msg providers.Message) error {
	l := s.sessionLock(sessionKey)
	l.Lock()
	defer l.Unlock()

	// Append the message as a single JSON line.
	line, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("memory: marshal message: %w", err)
	}
	line = append(line, '\n')

	f, err := os.OpenFile(
		s.jsonlPath(sessionKey),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND,
		0o644,
	)
	if err != nil {
		return fmt.Errorf("memory: open jsonl for append: %w", err)
	}
	_, writeErr := f.Write(line)
	closeErr := f.Close()
	if writeErr != nil {
		return fmt.Errorf("memory: append message: %w", writeErr)
	}
	if closeErr != nil {
		return fmt.Errorf("memory: close jsonl: %w", closeErr)
	}

	// Update metadata.
	meta, err := s.readMeta(sessionKey)
	if err != nil {
		return err
	}
	now := time.Now()
	if meta.Count == 0 && meta.CreatedAt.IsZero() {
		meta.CreatedAt = now
	}
	meta.Count++
	meta.UpdatedAt = now

	return s.writeMeta(sessionKey, meta)
}

func (s *JSONLStore) GetHistory(
	_ context.Context, sessionKey string,
) ([]providers.Message, error) {
	l := s.sessionLock(sessionKey)
	l.Lock()
	defer l.Unlock()

	meta, err := s.readMeta(sessionKey)
	if err != nil {
		return nil, err
	}

	msgs, err := readMessages(s.jsonlPath(sessionKey))
	if err != nil {
		return nil, err
	}

	// Apply logical truncation: skip the first meta.Skip messages.
	if meta.Skip > 0 && meta.Skip < len(msgs) {
		msgs = msgs[meta.Skip:]
	} else if meta.Skip >= len(msgs) {
		msgs = []providers.Message{}
	}

	return msgs, nil
}

func (s *JSONLStore) GetSummary(
	_ context.Context, sessionKey string,
) (string, error) {
	l := s.sessionLock(sessionKey)
	l.Lock()
	defer l.Unlock()

	meta, err := s.readMeta(sessionKey)
	if err != nil {
		return "", err
	}
	return meta.Summary, nil
}

func (s *JSONLStore) SetSummary(
	_ context.Context, sessionKey, summary string,
) error {
	l := s.sessionLock(sessionKey)
	l.Lock()
	defer l.Unlock()

	meta, err := s.readMeta(sessionKey)
	if err != nil {
		return err
	}
	now := time.Now()
	if meta.CreatedAt.IsZero() {
		meta.CreatedAt = now
	}
	meta.Summary = summary
	meta.UpdatedAt = now

	return s.writeMeta(sessionKey, meta)
}

func (s *JSONLStore) TruncateHistory(
	_ context.Context, sessionKey string, keepLast int,
) error {
	l := s.sessionLock(sessionKey)
	l.Lock()
	defer l.Unlock()

	meta, err := s.readMeta(sessionKey)
	if err != nil {
		return err
	}

	// If the meta count might be stale (e.g. after a crash during
	// addMsg), reconcile with the actual line count on disk.
	if meta.Count == 0 {
		msgs, readErr := readMessages(s.jsonlPath(sessionKey))
		if readErr != nil {
			return readErr
		}
		meta.Count = len(msgs)
	}

	if keepLast <= 0 {
		meta.Skip = meta.Count
	} else {
		effective := meta.Count - meta.Skip
		if keepLast < effective {
			meta.Skip = meta.Count - keepLast
		}
	}
	meta.UpdatedAt = time.Now()

	return s.writeMeta(sessionKey, meta)
}

func (s *JSONLStore) SetHistory(
	_ context.Context,
	sessionKey string,
	history []providers.Message,
) error {
	l := s.sessionLock(sessionKey)
	l.Lock()
	defer l.Unlock()

	// Rewrite the JSONL file atomically (temp + rename).
	target := s.jsonlPath(sessionKey)
	tmp := target + ".tmp"

	f, err := os.Create(tmp)
	if err != nil {
		return fmt.Errorf("memory: create jsonl tmp: %w", err)
	}

	for i, msg := range history {
		line, marshalErr := json.Marshal(msg)
		if marshalErr != nil {
			f.Close()
			_ = os.Remove(tmp)
			return fmt.Errorf("memory: marshal message %d: %w", i, marshalErr)
		}
		line = append(line, '\n')
		_, writeErr := f.Write(line)
		if writeErr != nil {
			f.Close()
			_ = os.Remove(tmp)
			return fmt.Errorf("memory: write message %d: %w", i, writeErr)
		}
	}

	err = f.Close()
	if err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("memory: close jsonl tmp: %w", err)
	}

	err = os.Rename(tmp, target)
	if err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("memory: rename jsonl: %w", err)
	}

	// Reset metadata: skip=0, count=len(history).
	meta, err := s.readMeta(sessionKey)
	if err != nil {
		return err
	}
	now := time.Now()
	if meta.CreatedAt.IsZero() {
		meta.CreatedAt = now
	}
	meta.Skip = 0
	meta.Count = len(history)
	meta.UpdatedAt = now

	return s.writeMeta(sessionKey, meta)
}

func (s *JSONLStore) Close() error {
	return nil
}

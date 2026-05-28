package wecom

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type wecomRoute struct {
	ReqID     string    `json:"req_id"`
	ChatID    string    `json:"chat_id"`
	ChatType  uint32    `json:"chat_type"`
	ExpiresAt time.Time `json:"expires_at"`
}

type reqIDStore struct {
	mu     sync.Mutex
	path   string
	routes map[string]wecomRoute
}

func newReqIDStore(path string) *reqIDStore {
	if path == "" {
		path = defaultReqIDStorePath()
	}
	s := &reqIDStore{
		path:   path,
		routes: make(map[string]wecomRoute),
	}
	_ = s.load()
	return s
}

func defaultReqIDStorePath() string {
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return filepath.Join(home, ".picoclaw", "wecom", "reqid-store.json")
	}
	return filepath.Join(os.TempDir(), "picoclaw-wecom-reqid-store.json")
}

func (s *reqIDStore) Put(chatID, reqID string, chatType uint32, ttl time.Duration) error {
	if reqID == "" || chatID == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.deleteExpiredLocked(time.Now())
	s.routes[chatID] = wecomRoute{
		ReqID:     reqID,
		ChatID:    chatID,
		ChatType:  chatType,
		ExpiresAt: time.Now().Add(ttl),
	}
	return s.saveLocked()
}

func (s *reqIDStore) Get(chatID string) (wecomRoute, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.deleteExpiredLocked(time.Now())
	route, ok := s.routes[chatID]
	return route, ok
}

func (s *reqIDStore) Delete(chatID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.routes, chatID)
	return s.saveLocked()
}

func (s *reqIDStore) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}

	var routes map[string]wecomRoute
	if err := json.Unmarshal(data, &routes); err != nil {
		return err
	}
	s.routes = routes
	s.deleteExpiredLocked(time.Now())
	return nil
}

func (s *reqIDStore) deleteExpiredLocked(now time.Time) {
	for chatID, route := range s.routes {
		if !route.ExpiresAt.IsZero() && now.After(route.ExpiresAt) {
			delete(s.routes, chatID)
		}
	}
}

func (s *reqIDStore) saveLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s.routes, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0o600)
}

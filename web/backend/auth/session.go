package auth

import (
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type SessionStore interface {
	Create(sessionID string, ttl time.Duration) (*Session, error)
	Get(sessionID string) (*Session, bool)
	Delete(sessionID string)
	Validate(sessionID string) bool
	Refresh(sessionID string, ttl time.Duration) bool
	CleanupExpired()
}

type MemorySessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

func NewMemorySessionStore() *MemorySessionStore {
	store := &MemorySessionStore{
		sessions: make(map[string]*Session),
	}
	go store.cleanupRoutine()
	return store
}

func (s *MemorySessionStore) Create(sessionID string, ttl time.Duration) (*Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session := &Session{
		ID:        sessionID,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(ttl),
	}
	s.sessions[sessionID] = session
	return session, nil
}

func (s *MemorySessionStore) Get(sessionID string) (*Session, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, exists := s.sessions[sessionID]
	if !exists || session.IsExpired() {
		return nil, false
	}
	return session, true
}

func (s *MemorySessionStore) Delete(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, sessionID)
}

func (s *MemorySessionStore) Validate(sessionID string) bool {
	_, exists := s.Get(sessionID)
	return exists
}

func (s *MemorySessionStore) Refresh(sessionID string, ttl time.Duration) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, exists := s.sessions[sessionID]
	if !exists || session.IsExpired() {
		return false
	}
	session.Refresh(ttl)
	return true
}

func (s *MemorySessionStore) CleanupExpired() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for id, session := range s.sessions {
		if session.IsExpired() {
			delete(s.sessions, id)
		}
	}
}

func (s *MemorySessionStore) cleanupRoutine() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.CleanupExpired()
	}
}

const bcryptCost = 12

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func VerifyPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

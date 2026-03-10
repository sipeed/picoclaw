package acp

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
)

type Manager struct {
	sessions map[string]*Session
	mu       sync.RWMutex
}

var (
	GlobalManager *Manager
	once          sync.Once
)

func GetManager() *Manager {
	once.Do(func() {
		GlobalManager = &Manager{
			sessions: make(map[string]*Session),
		}
	})
	return GlobalManager
}

func generateUUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func (m *Manager) Spawn(agentID, mode, command, cwd, label string, args []string) (*Session, error) {
	sessionID := generateUUID()
	key := fmt.Sprintf("agent:%s:acp:%s", agentID, sessionID)

	session := NewSession(key, agentID, mode, command, cwd, label, args)
	if err := session.Start(); err != nil {
		return nil, err
	}

	m.mu.Lock()
	m.sessions[key] = session
	m.mu.Unlock()

	return session, nil
}

func (m *Manager) GetSession(key string) (*Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	session, ok := m.sessions[key]
	return session, ok
}

func (m *Manager) CloseSession(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	session, ok := m.sessions[key]
	if !ok {
		return fmt.Errorf("session not found: %s", key)
	}
	err := session.Close()
	delete(m.sessions, key)
	return err
}

func (m *Manager) ListSessions() []*Session {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var list []*Session
	for _, s := range m.sessions {
		list = append(list, s)
	}
	return list
}

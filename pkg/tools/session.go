package tools

import (
	"errors"
	"io"
	"os"
	"sync"

	"github.com/google/uuid"
)

var (
	ErrSessionNotFound = errors.New("session not found")
	ErrSessionDone     = errors.New("session already completed")
	ErrPTYNotSupported = errors.New("PTY is not supported on this platform")
	ErrNoStdin         = errors.New("no stdin available")
)

type ProcessSession struct {
	mu          sync.Mutex
	ID          string
	PID         int
	Command     string
	PTY         bool
	Background  bool
	StartTime   int64
	ExitCode    int
	Status      string
	stdinWriter io.Writer
	stdoutPipe  io.Reader
	ptyMaster   *os.File
}

func (s *ProcessSession) IsDone() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Status == "done" || s.Status == "exited"
}

func (s *ProcessSession) GetStatus() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Status
}

func (s *ProcessSession) SetStatus(status string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Status = status
}

func (s *ProcessSession) GetExitCode() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.ExitCode
}

func (s *ProcessSession) SetExitCode(code int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ExitCode = code
}

func (s *ProcessSession) killProcess() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Status != "running" {
		return ErrSessionDone
	}

	pid := s.PID
	if pid <= 0 {
		return ErrSessionNotFound
	}

	if err := killProcessGroup(pid); err != nil {
		return err
	}

	s.Status = "done"
	s.ExitCode = -1
	return nil
}

func (s *ProcessSession) Kill() error {
	return s.killProcess()
}

func (s *ProcessSession) Write(data string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Status != "running" {
		return ErrSessionDone
	}

	var writer io.Writer
	if s.PTY && s.ptyMaster != nil {
		writer = s.ptyMaster
	} else if s.stdinWriter != nil {
		writer = s.stdinWriter
	} else {
		return ErrNoStdin
	}

	_, err := writer.Write([]byte(data))
	return err
}

func (s *ProcessSession) Read() (string, error) {
	s.mu.Lock()
	if s.Status != "running" {
		s.mu.Unlock()
		return "", ErrSessionDone
	}

	var reader io.Reader
	if s.PTY && s.ptyMaster != nil {
		reader = s.ptyMaster
	} else if s.stdoutPipe != nil {
		reader = s.stdoutPipe
	} else {
		s.mu.Unlock()
		return "", nil
	}

	buf := make([]byte, 4096)
	s.mu.Unlock()

	n, err := reader.Read(buf)
	if err != nil {
		return "", s.handleReadError(err)
	}

	return string(buf[:n]), nil
}

func (s *ProcessSession) ToSessionInfo() SessionInfo {
	s.mu.Lock()
	defer s.mu.Unlock()

	return SessionInfo{
		ID:        s.ID,
		Command:   s.Command,
		Status:    s.Status,
		PID:       s.PID,
		StartedAt: s.StartTime,
	}
}

type SessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*ProcessSession
}

func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*ProcessSession),
	}
}

func (sm *SessionManager) Add(session *ProcessSession) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.sessions[session.ID] = session
}

func (sm *SessionManager) Get(sessionID string) (*ProcessSession, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	session, ok := sm.sessions[sessionID]
	if !ok {
		return nil, ErrSessionNotFound
	}

	return session, nil
}

func (sm *SessionManager) Remove(sessionID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.sessions, sessionID)
}

func (sm *SessionManager) List() []SessionInfo {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	var result []SessionInfo
	for id, session := range sm.sessions {
		if session.IsDone() {
			delete(sm.sessions, id)
			continue
		}

		result = append(result, session.ToSessionInfo())
	}

	return result
}

func (sm *SessionManager) Cleanup() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for id, session := range sm.sessions {
		if session.IsDone() {
			delete(sm.sessions, id)
		}
	}
}

func generateSessionID() string {
	return uuid.New().String()[:8]
}

type SessionInfo struct {
	ID        string `json:"id"`
	Command   string `json:"command"`
	Status    string `json:"status"`
	PID       int    `json:"pid"`
	StartedAt int64  `json:"startedAt"`
}
